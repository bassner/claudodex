package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bassner/claudodex/internal/auth"
	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/convert"
	"github.com/bassner/claudodex/internal/ratelimit"
)

const maxMessagesBody = 64 << 20

var errPreviousResponseNotFound = errors.New("previous response not found")

type upstreamStreamEventError struct {
	typ     string
	message string
}

func (e upstreamStreamEventError) Error() string {
	if strings.TrimSpace(e.message) != "" {
		return e.message
	}
	return "Codex upstream stream failed"
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxMessagesBody+1))
	if err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "failed to read request body")
		return
	}
	if len(body) > maxMessagesBody {
		writeAnthropicError(w, http.StatusRequestEntityTooLarge, "invalid_request_error", "request body is too large")
		return
	}
	var anthropicReq convert.AnthropicRequest
	if err := json.Unmarshal(body, &anthropicReq); err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "request body is not valid JSON")
		return
	}

	sessionID := sessionIDFromRequest(r)
	result, err := convert.AnthropicToCodex(anthropicReq, convert.ConvertOptions{
		SessionID:   sessionID,
		Models:      s.cfg.ModelConfig,
		CodexModels: s.cfg.Models,
	})
	if err != nil {
		var bad convert.BadRequestError
		if errors.As(err, &bad) {
			writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", bad.Message)
			return
		}
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	route := codex.MaterializeRoute(codexRouteForResult(result, sessionID))
	chainKey := route.ThreadID
	fullRequest := result.Request
	upstreamRequest := result.Request
	traceID := s.nextTraceID()
	traceBase := map[string]any{
		"request_id":                 traceID,
		"session_id":                 sessionID,
		"thread_id":                  route.ThreadID,
		"parent_thread_id":           route.ParentThreadID,
		"subagent":                   route.Subagent,
		"chain_key":                  chainKey,
		"model":                      result.Request.Model,
		"original_model":             result.OriginalModel,
		"stream":                     result.Stream,
		"body_bytes":                 len(body),
		"full_input_items":           len(fullRequest.Input),
		"anthropic_messages":         len(anthropicReq.Messages),
		"anthropic_max_tokens":       anthropicReq.MaxTokens,
		"anthropic_system_bytes":     len(anthropicReq.System),
		"claude_auto_compact_window": os.Getenv("CLAUDE_CODE_AUTO_COMPACT_WINDOW"),
		"claude_max_context_tokens":  os.Getenv("CLAUDE_CODE_MAX_CONTEXT_TOKENS"),
	}
	replayedStateless, replayReason, replayTrimItems, replayInputItems := s.applyStatelessReplayDetailed(chainKey, &fullRequest)
	usedImplicitResume := false
	resumeReason := "not_attempted"
	resumePrefixItems := 0
	resumeInputItems := len(upstreamRequest.Input)
	// A previous_response_id is safe for store:false only while continuing the
	// same live WebSocket conversation. Across ordinary HTTP requests, replay
	// the complete response item sequence, including encrypted reasoning.
	if s.hasWebSocket(chainKey) {
		usedImplicitResume, resumeReason, resumePrefixItems, resumeInputItems = s.applyImplicitResumeDetailed(chainKey, &upstreamRequest)
	} else {
		resumeReason = "no_live_websocket"
	}
	if !usedImplicitResume {
		upstreamRequest = fullRequest
		resumeInputItems = len(upstreamRequest.Input)
	}
	traceBase = mergeTraceFields(traceBase, map[string]any{
		"stateless_replay":     replayedStateless,
		"replay_reason":        replayReason,
		"replay_trim_items":    replayTrimItems,
		"replay_input_items":   replayInputItems,
		"implicit_resume":      usedImplicitResume,
		"resume_reason":        resumeReason,
		"resume_prefix_items":  resumePrefixItems,
		"resume_input_items":   resumeInputItems,
		"upstream_input_items": len(upstreamRequest.Input),
		"previous_response_id": upstreamRequest.PreviousResponseID,
	})
	s.trace("messages.request", traceBase)
	remainingGenerationAttempts := 2
	createResponse := func(req codex.Request) (*http.Response, error) {
		s.trace("upstream.waiting_headers", mergeTraceFields(traceBase, map[string]any{
			"generation_attempt":            3 - remainingGenerationAttempts,
			"generation_attempts_remaining": remainingGenerationAttempts,
		}))
		resp, used, err := s.createCodexResponse(r, req, route, remainingGenerationAttempts)
		if used < 0 {
			used = 0
		}
		if used > remainingGenerationAttempts {
			used = remainingGenerationAttempts
		}
		remainingGenerationAttempts -= used
		return resp, err
	}

	createStarted := time.Now()
	upstream, err := createResponse(upstreamRequest)
	if err != nil {
		s.trace("upstream.create_error", mergeTraceFields(traceBase, upstreamCreateErrorTraceFields(err, map[string]any{
			"attempt":                       1,
			"elapsed_ms":                    traceDurationMS(createStarted),
			"generation_attempts_remaining": remainingGenerationAttempts,
		})))
	} else {
		s.trace("upstream.opened", mergeTraceFields(traceBase, map[string]any{
			"attempt":                       1,
			"elapsed_ms":                    traceDurationMS(createStarted),
			"status":                        upstream.StatusCode,
			"transport":                     upstream.Header.Get("x-claudodex-transport"),
			"ws_reused":                     upstream.Header.Get("x-claudodex-ws-reused"),
			"response_header_retries":       upstream.Header.Get("x-claudodex-response-header-retries"),
			"generation_attempts_remaining": remainingGenerationAttempts,
		}))
	}
	if err != nil && usedImplicitResume && remainingGenerationAttempts > 0 {
		s.trace("resume.retry_full", mergeTraceFields(traceBase, map[string]any{
			"reason": "create_error",
			"error":  err.Error(),
		}))
		s.clearImplicitResume(chainKey)
		s.closeWebSocket(chainKey)
		createStarted = time.Now()
		upstream, err = createResponse(fullRequest)
		if err != nil {
			s.trace("upstream.create_error", mergeTraceFields(traceBase, upstreamCreateErrorTraceFields(err, map[string]any{
				"attempt":                       2,
				"elapsed_ms":                    traceDurationMS(createStarted),
				"generation_attempts_remaining": remainingGenerationAttempts,
			})))
		} else {
			s.trace("upstream.opened", mergeTraceFields(traceBase, map[string]any{
				"attempt":                       2,
				"elapsed_ms":                    traceDurationMS(createStarted),
				"status":                        upstream.StatusCode,
				"transport":                     upstream.Header.Get("x-claudodex-transport"),
				"ws_reused":                     upstream.Header.Get("x-claudodex-ws-reused"),
				"response_header_retries":       upstream.Header.Get("x-claudodex-response-header-retries"),
				"generation_attempts_remaining": remainingGenerationAttempts,
				"upstream_input_items":          len(fullRequest.Input),
				"previous_response_id":          "",
			}))
		}
	}
	if err != nil {
		writeMappedUpstreamError(w, err)
		return
	}

	fallbackInputTokens := estimateTokenCountFromBytes(body, false)
	if result.Stream {
		applyRateLimitHeaders(w.Header(), upstream.Header, false)
		err = s.streamAnthropicWithSchemas(w, upstream.Body, result.OriginalModel, result.ToolSchemas, fallbackInputTokens, chainKey, fullRequest, traceBase, usedImplicitResume)
		_ = upstream.Body.Close()
		if shouldRetryStream(r, err, usedImplicitResume) && remainingGenerationAttempts > 0 {
			s.trace("resume.retry_full", mergeTraceFields(traceBase, map[string]any{
				"reason": "stream_error",
				"error":  err.Error(),
			}))
			s.clearImplicitResume(chainKey)
			s.closeWebSocket(chainKey)
			createStarted = time.Now()
			upstream, err = createResponse(fullRequest)
			if err == nil {
				s.trace("upstream.opened", mergeTraceFields(traceBase, map[string]any{
					"attempt":                       2,
					"elapsed_ms":                    traceDurationMS(createStarted),
					"status":                        upstream.StatusCode,
					"transport":                     upstream.Header.Get("x-claudodex-transport"),
					"ws_reused":                     upstream.Header.Get("x-claudodex-ws-reused"),
					"response_header_retries":       upstream.Header.Get("x-claudodex-response-header-retries"),
					"generation_attempts_remaining": remainingGenerationAttempts,
					"upstream_input_items":          len(fullRequest.Input),
					"previous_response_id":          "",
				}))
				applyRateLimitHeaders(w.Header(), upstream.Header, false)
				err = s.streamAnthropicWithSchemas(w, upstream.Body, result.OriginalModel, result.ToolSchemas, fallbackInputTokens, chainKey, fullRequest, mergeTraceFields(traceBase, map[string]any{
					"implicit_resume":      false,
					"upstream_input_items": len(fullRequest.Input),
					"previous_response_id": "",
				}), false)
				_ = upstream.Body.Close()
			} else {
				s.trace("upstream.create_error", mergeTraceFields(traceBase, upstreamCreateErrorTraceFields(err, map[string]any{
					"attempt":                       2,
					"elapsed_ms":                    traceDurationMS(createStarted),
					"generation_attempts_remaining": remainingGenerationAttempts,
				})))
			}
		}
		if err != nil {
			var timeoutErr *codex.ResponseHeaderTimeoutError
			if errors.As(err, &timeoutErr) {
				writeMappedUpstreamError(w, err)
			} else {
				writeAnthropicError(w, http.StatusBadGateway, "api_error", "Codex upstream stream failed")
			}
		}
		return
	}
	applyRateLimitHeaders(w.Header(), upstream.Header, false)
	err = s.writeNonStreamingMessageWithSchemas(w, upstream.Body, result.OriginalModel, result.ToolSchemas, fallbackInputTokens, chainKey, fullRequest, traceBase)
	_ = upstream.Body.Close()
	if shouldRetryStream(r, err, usedImplicitResume) && remainingGenerationAttempts > 0 {
		s.trace("resume.retry_full", mergeTraceFields(traceBase, map[string]any{
			"reason": "stream_error",
			"error":  err.Error(),
		}))
		s.clearImplicitResume(chainKey)
		s.closeWebSocket(chainKey)
		createStarted = time.Now()
		upstream, err = createResponse(fullRequest)
		if err == nil {
			s.trace("upstream.opened", mergeTraceFields(traceBase, map[string]any{
				"attempt":                       2,
				"elapsed_ms":                    traceDurationMS(createStarted),
				"status":                        upstream.StatusCode,
				"transport":                     upstream.Header.Get("x-claudodex-transport"),
				"ws_reused":                     upstream.Header.Get("x-claudodex-ws-reused"),
				"response_header_retries":       upstream.Header.Get("x-claudodex-response-header-retries"),
				"generation_attempts_remaining": remainingGenerationAttempts,
				"upstream_input_items":          len(fullRequest.Input),
				"previous_response_id":          "",
			}))
			applyRateLimitHeaders(w.Header(), upstream.Header, false)
			err = s.writeNonStreamingMessageWithSchemas(w, upstream.Body, result.OriginalModel, result.ToolSchemas, fallbackInputTokens, chainKey, fullRequest, mergeTraceFields(traceBase, map[string]any{
				"implicit_resume":      false,
				"upstream_input_items": len(fullRequest.Input),
				"previous_response_id": "",
			}))
			_ = upstream.Body.Close()
		} else {
			s.trace("upstream.create_error", mergeTraceFields(traceBase, upstreamCreateErrorTraceFields(err, map[string]any{
				"attempt":                       2,
				"elapsed_ms":                    traceDurationMS(createStarted),
				"generation_attempts_remaining": remainingGenerationAttempts,
			})))
		}
	}
	if err != nil {
		var timeoutErr *codex.ResponseHeaderTimeoutError
		if errors.As(err, &timeoutErr) {
			writeMappedUpstreamError(w, err)
		} else {
			writeAnthropicError(w, http.StatusBadGateway, "api_error", err.Error())
		}
	}
}

func (s *Server) createCodexResponse(r *http.Request, req codex.Request, route codex.Route, generationAttemptBudget int) (*http.Response, int, error) {
	attemptsUsed := 0
	finish := func(resp *http.Response, err error) (*http.Response, int, error) {
		if attemptsUsed > 0 {
			codex.SetCreateResponseAttempts(resp, attemptsUsed)
			err = codex.WithCreateResponseAttempts(err, attemptsUsed)
		}
		return resp, attemptsUsed, err
	}
	remainingAttempts := func() int {
		remaining := generationAttemptBudget - attemptsUsed
		if remaining < 0 {
			return 0
		}
		return remaining
	}
	observeHTTPAttempt := func(resp *http.Response, err error) {
		used := codex.CreateResponseAttempts(resp, err)
		if used > remainingAttempts() {
			used = remainingAttempts()
		}
		attemptsUsed += used
	}

	store := auth.NewStore(s.cfg.Home)
	file, err := auth.NewRefresher(store, s.cfg.HTTPClient, s.cfg.TokenEndpoint).EnsureFresh(r.Context(), 5*time.Minute)
	if err != nil {
		return finish(nil, err)
	}
	installationID, err := auth.InstallationID(s.cfg.Home)
	if err != nil {
		return finish(nil, err)
	}
	req.ClientMetadata = map[string]string{"x-codex-installation-id": installationID}

	client := codex.Client{BaseURL: s.cfg.CodexBaseURL, HTTPClient: s.cfg.HTTPClient, Version: s.cfg.Version, ResponseHeaderAttempts: remainingAttempts()}
	credentials := codex.Credentials{
		AccessToken:    file.Tokens.AccessToken,
		AccountID:      file.Tokens.AccountID,
		InstallationID: installationID,
		FedRAMP:        file.Tokens.ChatGPTAccountIsFedRAMP,
	}
	if s.codexWebSocketEnabled() && (strings.TrimSpace(route.Subagent) != "" || strings.TrimSpace(req.PreviousResponseID) != "") {
		reused := s.hasWebSocket(route.ThreadID)
		if conversation := s.websocketConversation(route.ThreadID); conversation != nil {
			waitForWebSocket := strings.TrimSpace(req.PreviousResponseID) != ""
			resp, err := conversation.CreateResponse(r.Context(), client, req, credentials, route, waitForWebSocket)
			if err == nil {
				attemptsUsed++
				resp.Header.Set("x-claudodex-transport", "websocket")
				resp.Header.Set("x-claudodex-ws-reused", strconv.FormatBool(reused))
				return finish(resp, nil)
			}
			if errors.Is(err, codex.ErrWebSocketBusy) && strings.TrimSpace(req.PreviousResponseID) == "" {
				// No request was sent: preserve the full budget for the HTTP fallback.
			} else {
				attemptsUsed++
				s.closeWebSocket(route.ThreadID)
				if strings.TrimSpace(req.PreviousResponseID) != "" || remainingAttempts() == 0 {
					return finish(nil, err)
				}
			}
		}
	}
	if remainingAttempts() == 0 {
		return finish(nil, errors.New("Codex generation attempt budget exhausted"))
	}
	client.ResponseHeaderAttempts = remainingAttempts()
	resp, err := client.CreateResponse(r.Context(), req, credentials, route)
	observeHTTPAttempt(resp, err)
	if err == nil {
		resp.Header.Set("x-claudodex-transport", "http")
		return finish(resp, nil)
	}
	var upstream *codex.UpstreamError
	if !errors.As(err, &upstream) || upstream.Status != http.StatusUnauthorized {
		return finish(nil, err)
	}
	if remainingAttempts() == 0 {
		return finish(nil, err)
	}

	file, refreshErr := auth.NewRefresher(store, s.cfg.HTTPClient, s.cfg.TokenEndpoint).Refresh(r.Context())
	if refreshErr != nil {
		return finish(nil, refreshErr)
	}
	credentials.AccessToken = file.Tokens.AccessToken
	credentials.AccountID = file.Tokens.AccountID
	credentials.FedRAMP = file.Tokens.ChatGPTAccountIsFedRAMP
	client.ResponseHeaderAttempts = remainingAttempts()
	resp, err = client.CreateResponse(r.Context(), req, credentials, route)
	observeHTTPAttempt(resp, err)
	if err == nil {
		resp.Header.Set("x-claudodex-transport", "http")
	}
	return finish(resp, err)
}

func upstreamCreateErrorTraceFields(err error, fields map[string]any) map[string]any {
	fields = mergeTraceFields(fields, map[string]any{"error": err.Error()})
	var timeoutErr *codex.ResponseHeaderTimeoutError
	if errors.As(err, &timeoutErr) {
		fields = mergeTraceFields(fields, map[string]any{
			"response_header_timeout":    true,
			"response_header_timeout_ms": timeoutErr.Timeout.Milliseconds(),
			"response_header_attempts":   timeoutErr.Attempts,
		})
	}
	return fields
}

func (s *Server) codexWebSocketEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("CLAUDODEX_DISABLE_CODEX_WEBSOCKET")))
	return value != "1" && value != "true" && value != "yes" && value != "on"
}

func shouldRetryStream(r *http.Request, err error, usedImplicitResume bool) bool {
	if err == nil {
		return false
	}
	if r != nil && r.Context().Err() != nil {
		return false
	}
	if usedImplicitResume {
		return true
	}
	var upstreamEvent upstreamStreamEventError
	if errors.As(err, &upstreamEvent) || errors.Is(err, errPreviousResponseNotFound) {
		return false
	}
	return isRetryableTransportError(err)
}

func isRetryableTransportError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	for _, fragment := range []string{
		"stream error",
		"internal_error",
		"unexpected eof",
		"connection reset",
		"use of closed network connection",
		"http2",
	} {
		if strings.Contains(message, fragment) {
			return true
		}
	}
	return false
}

func codexRouteForResult(result convert.Result, parentSessionID string) codex.Route {
	sessionID := strings.TrimSpace(result.RouteSessionID)
	if sessionID == "" {
		sessionID = parentSessionID
	}
	route := codex.Route{SessionID: sessionID, ThreadID: sessionID}
	if result.ParentThreadID != "" {
		route.ParentThreadID = result.ParentThreadID
	}
	if result.Subagent != "" {
		route.Subagent = result.Subagent
	}
	return route
}

func (s *Server) streamAnthropic(w http.ResponseWriter, body io.Reader, model string) {
	_ = s.streamAnthropicWithSchemas(w, body, model, nil, 0, "", codex.Request{}, nil, false)
}

func (s *Server) streamAnthropicWithSchemas(w http.ResponseWriter, body io.Reader, model string, toolSchemas map[string]map[string]any, fallbackInputTokens int, chainKey string, fullRequest codex.Request, traceBase map[string]any, retryEarlyUpstreamErrors bool) error {
	w.Header().Set("content-type", "text/event-stream")
	w.Header().Set("cache-control", "no-cache")
	w.Header().Set("connection", "keep-alive")
	flusher, _ := w.(http.Flusher)
	wrote := false
	var pending []convert.AnthropicSSE
	writeEvent := func(event convert.AnthropicSSE) error {
		if !wrote {
			w.WriteHeader(http.StatusOK)
			wrote = true
		}
		if err := writeSSE(w, event); err != nil {
			return err
		}
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	}
	flushPending := func() error {
		for _, event := range pending {
			if err := writeEvent(event); err != nil {
				return err
			}
		}
		pending = nil
		return nil
	}
	writeConvertedEvent := func(event convert.AnthropicSSE) error {
		if !wrote && event.Event == "message_start" {
			pending = append(pending, event)
			return nil
		}
		if err := flushPending(); err != nil {
			return err
		}
		return writeEvent(event)
	}
	reducer := convert.NewStreamReducerWithOptions(anthropicMessageID(traceBase), model, convert.StreamReducerOptions{
		ToolSchemas:         toolSchemas,
		FallbackInputTokens: fallbackInputTokens,
	})
	var trace responseTrace
	streamStarted := time.Now()
	notifyIdle, stopIdle := s.startStreamIdleTrace(traceBase, streamStarted)
	defer stopIdle()
	eventCount := 0
	toolArgDeltaCount := 0
	toolArgDeltaBytes := 0
	err := codex.ReadSSE(body, func(event codex.SSEEvent) error {
		if !wrote && isPreviousResponseNotFoundEvent(event) {
			return errPreviousResponseNotFound
		}
		eventCount++
		if event.Event == "response.function_call_arguments.delta" {
			toolArgDeltaCount++
			toolArgDeltaBytes += eventStringFieldLen(event.Data, "delta")
		}
		notifyIdle(event.Event)
		if eventCount == 1 {
			s.trace("stream.first_event", mergeTraceFields(traceBase, map[string]any{
				"elapsed_ms":  traceDurationMS(streamStarted),
				"codex_event": event.Event,
			}))
		}
		if shouldTraceCodexEvent(event.Event) {
			fields := map[string]any{
				"elapsed_ms":  traceDurationMS(streamStarted),
				"codex_event": event.Event,
			}
			if msg := eventErrorMessage(event); msg != "" {
				fields["upstream_error"] = msg
			}
			s.trace("stream.event", mergeTraceFields(traceBase, fields))
		}
		trace.observe(event)
		events, err := reducer.ReduceNamed(event.Event, event.Data)
		if err != nil {
			return err
		}
		if reducer.Failed() && retryEarlyUpstreamErrors && !wrote {
			return upstreamStreamEventError{typ: reducer.FailureType(), message: reducer.FailureMessage()}
		}
		for _, anthropicEvent := range events {
			if err := writeConvertedEvent(anthropicEvent); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		s.trace("stream.error", mergeTraceFields(traceBase, map[string]any{
			"elapsed_ms":            traceDurationMS(streamStarted),
			"events":                eventCount,
			"tool_arg_delta_events": toolArgDeltaCount,
			"tool_arg_delta_bytes":  toolArgDeltaBytes,
			"wrote":                 wrote,
			"error":                 err.Error(),
		}))
		if !wrote {
			return err
		}
		_ = writeEvent(streamErrorEvent("Codex upstream stream failed"))
		return nil
	}
	if reducer.Failed() {
		s.trace("stream.upstream_error", mergeTraceFields(traceBase, map[string]any{
			"elapsed_ms":            traceDurationMS(streamStarted),
			"events":                eventCount,
			"tool_arg_delta_events": toolArgDeltaCount,
			"tool_arg_delta_bytes":  toolArgDeltaBytes,
			"error_type":            reducer.FailureType(),
			"error":                 reducer.FailureMessage(),
		}))
		if strings.TrimSpace(chainKey) != "" {
			s.trace("chain.not_recorded", mergeTraceFields(traceBase, map[string]any{
				"reason":       "upstream_error",
				"response_id":  trace.ResponseID,
				"output_items": len(trace.Output),
			}))
		}
		return nil
	}
	if !reducer.Done() {
		s.trace("stream.incomplete", mergeTraceFields(traceBase, map[string]any{
			"elapsed_ms":            traceDurationMS(streamStarted),
			"events":                eventCount,
			"tool_arg_delta_events": toolArgDeltaCount,
			"tool_arg_delta_bytes":  toolArgDeltaBytes,
			"wrote":                 wrote,
		}))
		if !wrote {
			return errors.New("Codex upstream stream ended before completion")
		}
		_ = writeEvent(streamErrorEvent("Codex upstream stream ended before completion"))
		return nil
	}
	if strings.TrimSpace(chainKey) != "" && strings.TrimSpace(trace.ResponseID) == "" {
		s.trace("chain.not_recorded", mergeTraceFields(traceBase, map[string]any{
			"reason":       "missing_response_id",
			"output_items": len(trace.Output),
		}))
	} else if strings.TrimSpace(chainKey) != "" {
		s.trace("chain.recorded", mergeTraceFields(traceBase, map[string]any{
			"response_id":  trace.ResponseID,
			"output_items": len(trace.Output),
		}))
	}
	s.trace("stream.completed", mergeTraceFields(traceBase, map[string]any{
		"elapsed_ms":                           traceDurationMS(streamStarted),
		"events":                               eventCount,
		"tool_arg_delta_events":                toolArgDeltaCount,
		"tool_arg_delta_bytes":                 toolArgDeltaBytes,
		"reported_input_tokens":                reducer.Usage().InputTokens,
		"reported_cache_creation_input_tokens": reducer.Usage().CacheCreationInputTokens,
		"reported_cache_read_input_tokens":     reducer.Usage().CacheReadInputTokens,
		"reported_total_input_tokens":          usageTotalInputTokens(reducer.Usage()),
		"reported_output_tokens":               reducer.Usage().OutputTokens,
	}))
	s.recordResponseChain(chainKey, fullRequest, trace)
	return nil
}

func usageTotalInputTokens(usage convert.Usage) int {
	return usage.InputTokens + usage.CacheCreationInputTokens + usage.CacheReadInputTokens
}

func (s *Server) writeNonStreamingMessage(w http.ResponseWriter, body io.Reader, model string) {
	_ = s.writeNonStreamingMessageWithSchemas(w, body, model, nil, 0, "", codex.Request{}, nil)
}

func (s *Server) writeNonStreamingMessageWithSchemas(w http.ResponseWriter, body io.Reader, model string, toolSchemas map[string]map[string]any, fallbackInputTokens int, chainKey string, fullRequest codex.Request, traceBase map[string]any) error {
	reducer := convert.NewStreamReducerWithOptions(anthropicMessageID(traceBase), model, convert.StreamReducerOptions{
		ToolSchemas:         toolSchemas,
		FallbackInputTokens: fallbackInputTokens,
	})
	var events []convert.AnthropicSSE
	var trace responseTrace
	streamStarted := time.Now()
	notifyIdle, stopIdle := s.startStreamIdleTrace(traceBase, streamStarted)
	defer stopIdle()
	eventCount := 0
	err := codex.ReadSSE(body, func(event codex.SSEEvent) error {
		if isPreviousResponseNotFoundEvent(event) {
			return errPreviousResponseNotFound
		}
		eventCount++
		notifyIdle(event.Event)
		if eventCount == 1 {
			s.trace("stream.first_event", mergeTraceFields(traceBase, map[string]any{
				"elapsed_ms":  traceDurationMS(streamStarted),
				"codex_event": event.Event,
			}))
		}
		if shouldTraceCodexEvent(event.Event) {
			fields := map[string]any{
				"elapsed_ms":  traceDurationMS(streamStarted),
				"codex_event": event.Event,
			}
			if msg := eventErrorMessage(event); msg != "" {
				fields["upstream_error"] = msg
			}
			s.trace("stream.event", mergeTraceFields(traceBase, fields))
		}
		trace.observe(event)
		next, err := reducer.ReduceNamed(event.Event, event.Data)
		if err != nil {
			return err
		}
		events = append(events, next...)
		return nil
	})
	if err != nil {
		s.trace("stream.error", mergeTraceFields(traceBase, map[string]any{
			"elapsed_ms": traceDurationMS(streamStarted),
			"events":     eventCount,
			"error":      err.Error(),
		}))
		return err
	}
	if reducer.Failed() {
		s.trace("stream.upstream_error", mergeTraceFields(traceBase, map[string]any{
			"elapsed_ms": traceDurationMS(streamStarted),
			"events":     eventCount,
			"error_type": reducer.FailureType(),
			"error":      reducer.FailureMessage(),
		}))
		if strings.TrimSpace(chainKey) != "" {
			s.trace("chain.not_recorded", mergeTraceFields(traceBase, map[string]any{
				"reason":       "upstream_error",
				"response_id":  trace.ResponseID,
				"output_items": len(trace.Output),
			}))
		}
		return upstreamStreamEventError{typ: reducer.FailureType(), message: reducer.FailureMessage()}
	}
	if !reducer.Done() {
		s.trace("stream.incomplete", mergeTraceFields(traceBase, map[string]any{
			"elapsed_ms": traceDurationMS(streamStarted),
			"events":     eventCount,
		}))
		return errors.New("Codex upstream stream ended before completion")
	}
	if strings.TrimSpace(chainKey) != "" && strings.TrimSpace(trace.ResponseID) == "" {
		s.trace("chain.not_recorded", mergeTraceFields(traceBase, map[string]any{
			"reason":       "missing_response_id",
			"output_items": len(trace.Output),
		}))
	} else if strings.TrimSpace(chainKey) != "" {
		s.trace("chain.recorded", mergeTraceFields(traceBase, map[string]any{
			"response_id":  trace.ResponseID,
			"output_items": len(trace.Output),
		}))
	}
	s.trace("stream.completed", mergeTraceFields(traceBase, map[string]any{
		"elapsed_ms": traceDurationMS(streamStarted),
		"events":     eventCount,
	}))
	s.recordResponseChain(chainKey, fullRequest, trace)
	message, errEvent := convert.AssembleMessage(events, "", model)
	if errEvent != nil {
		writeJSON(w, http.StatusBadGateway, errEvent.Data)
		return nil
	}
	writeJSON(w, http.StatusOK, message)
	return nil
}

func anthropicMessageID(traceBase map[string]any) string {
	traceID, _ := traceBase["request_id"].(string)
	traceID = strings.TrimSpace(traceID)
	if traceID == "" {
		return ""
	}
	return "msg_claudodex_" + strings.NewReplacer("-", "_", ".", "_").Replace(traceID)
}

func streamErrorEvent(message string) convert.AnthropicSSE {
	return convert.AnthropicSSE{
		Event: "error",
		Data: map[string]any{
			"type": "error",
			"error": map[string]any{
				"type":    "api_error",
				"message": message,
			},
		},
	}
}

func writeSSE(w io.Writer, event convert.AnthropicSSE) error {
	data, err := json.Marshal(event.Data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Event, data)
	return err
}

func isPreviousResponseNotFoundEvent(event codex.SSEEvent) bool {
	message := eventErrorMessage(event)
	normalized := strings.ToLower(message)
	return strings.Contains(normalized, "previous response") && strings.Contains(normalized, "not found")
}

func eventErrorMessage(event codex.SSEEvent) string {
	var payload map[string]any
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return ""
	}
	if errorObj, _ := payload["error"].(map[string]any); errorObj != nil {
		if message, _ := errorObj["message"].(string); message != "" {
			return message
		}
	}
	if response, _ := payload["response"].(map[string]any); response != nil {
		if errorObj, _ := response["error"].(map[string]any); errorObj != nil {
			if message, _ := errorObj["message"].(string); message != "" {
				return message
			}
		}
	}
	if message, _ := payload["message"].(string); message != "" {
		return message
	}
	return ""
}

func eventStringFieldLen(raw json.RawMessage, field string) int {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return 0
	}
	value, _ := payload[field].(string)
	return len(value)
}

func applyRateLimitHeaders(dst http.Header, upstream http.Header, forceRejected bool) {
	snapshot := ratelimit.FromCodexHeaders(upstream)
	if snapshot == nil && forceRejected {
		snapshot = ratelimit.SnapshotFromRetryAfter(upstream, time.Now())
	}
	ratelimit.ApplyAnthropicHeaders(dst, snapshot, forceRejected)
}

func sessionIDFromRequest(r *http.Request) string {
	for _, key := range []string{"X-Claude-Code-Session-Id", "x-claude-code-session-id", "x-client-request-id"} {
		if value := strings.TrimSpace(r.Header.Get(key)); value != "" {
			return sanitizeRouteID(value)
		}
	}
	return ""
}

func sanitizeRouteID(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 128 {
		return value[:128]
	}
	return value
}
