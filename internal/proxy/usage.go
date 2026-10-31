package proxy

import (
	"errors"
	"net/http"

	"github.com/bassner/claudodex/internal/auth"
	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/convert"
)

func (s *Server) handleUsage(w http.ResponseWriter, r *http.Request) {
	raw, err := s.fetchCodexUsage(r)
	if err != nil {
		writeMappedUpstreamError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, convert.CodexUsageToClaude(raw))
}

func (s *Server) fetchCodexUsage(r *http.Request) (map[string]any, error) {
	file, err := s.freshAuth(r.Context())
	if err != nil {
		return nil, err
	}
	installationID, err := auth.InstallationID(s.cfg.Home)
	if err != nil {
		return nil, err
	}
	client := codex.Client{BaseURL: s.cfg.CodexBaseURL, HTTPClient: s.cfg.HTTPClient, Version: s.cfg.Version}
	credentials := codex.Credentials{
		AccessToken:    file.Tokens.AccessToken,
		AccountID:      file.Tokens.AccountID,
		InstallationID: installationID,
		FedRAMP:        file.Tokens.ChatGPTAccountIsFedRAMP,
		AllowedOrigin:  file.AllowedOrigin,
	}
	raw, err := client.FetchUsage(r.Context(), credentials)
	if err == nil {
		return raw, nil
	}
	var upstream *codex.UpstreamError
	if !errors.As(err, &upstream) || upstream.Status != http.StatusUnauthorized {
		return nil, err
	}
	file, refreshErr := s.refreshAuth(r.Context(), credentials.AccessToken)
	if refreshErr != nil {
		return nil, refreshErr
	}
	credentials.AccessToken = file.Tokens.AccessToken
	credentials.AccountID = file.Tokens.AccountID
	credentials.FedRAMP = file.Tokens.ChatGPTAccountIsFedRAMP
	credentials.AllowedOrigin = file.AllowedOrigin
	return client.FetchUsage(r.Context(), credentials)
}
