package codex

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

const DefaultBaseURL = "https://chatgpt.com/backend-api"

type Request struct {
	Model       string `json:"model"`
	Stream      bool   `json:"stream"`
	ServiceTier string `json:"service_tier,omitempty"`
	// Instructions is retained for internal compatibility checks. Codex base
	// instructions are serialized as a leading developer input item.
	Instructions       string            `json:"-"`
	PreviousResponseID string            `json:"previous_response_id,omitempty"`
	Input              []InputItem       `json:"input"`
	Tools              []Tool            `json:"tools,omitempty"`
	ToolChoice         any               `json:"tool_choice,omitempty"`
	ParallelToolCalls  bool              `json:"parallel_tool_calls"`
	Store              bool              `json:"store"`
	Include            []string          `json:"include,omitempty"`
	Reasoning          *Reasoning        `json:"reasoning,omitempty"`
	Text               *TextConfig       `json:"text,omitempty"`
	PromptCacheKey     string            `json:"prompt_cache_key,omitempty"`
	ClientMetadata     map[string]string `json:"client_metadata,omitempty"`
}

func (r Request) MarshalJSON() ([]byte, error) {
	type wire Request
	return json.Marshal(wire(r.withDeveloperInstructions()))
}

func (r Request) withDeveloperInstructions() Request {
	if instructions := strings.TrimSpace(r.Instructions); instructions != "" {
		hash := sha256.New()
		_, _ = hash.Write([]byte(strings.TrimSpace(r.PromptCacheKey)))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(instructions))
		developer := InputItem{
			ID:   "msg_" + hex.EncodeToString(hash.Sum(nil)[:16]),
			Type: "message",
			Role: "developer",
			Content: []ContentPart{{
				Type: "input_text",
				Text: instructions,
			}},
		}
		r.Input = append([]InputItem{developer}, r.Input...)
	}
	return r
}

type Reasoning struct {
	Effort  string `json:"effort"`
	Summary string `json:"summary,omitempty"`
}

type TextConfig struct {
	Format *TextFormat `json:"format,omitempty"`
}

type TextFormat struct {
	Type   string         `json:"type"`
	Name   string         `json:"name,omitempty"`
	Schema map[string]any `json:"schema,omitempty"`
	Strict *bool          `json:"strict,omitempty"`
}

type InputItem struct {
	ID                                     string                                  `json:"id,omitempty"`
	Type                                   string                                  `json:"type"`
	Role                                   string                                  `json:"role,omitempty"`
	Content                                []ContentPart                           `json:"content,omitempty"`
	CallID                                 string                                  `json:"call_id,omitempty"`
	Name                                   string                                  `json:"name,omitempty"`
	Arguments                              string                                  `json:"arguments,omitempty"`
	Output                                 any                                     `json:"output,omitempty"`
	InternalChatMessageMetadataPassthrough *InternalChatMessageMetadataPassthrough `json:"internal_chat_message_metadata_passthrough,omitempty"`
	Raw                                    map[string]json.RawMessage              `json:"-"`
}

type InternalChatMessageMetadataPassthrough struct {
	CreateTime float64 `json:"create_time,omitempty"`
}

func (i *InputItem) SetCreateTimeIfMissing(createTime float64) {
	if i == nil || createTime <= 0 || len(i.Raw) > 0 {
		return
	}
	eligible := i.Type == "function_call_output" ||
		(i.Type == "message" && (i.Role == "user" || i.Role == "developer"))
	if !eligible {
		return
	}
	if i.InternalChatMessageMetadataPassthrough == nil {
		i.InternalChatMessageMetadataPassthrough = &InternalChatMessageMetadataPassthrough{}
	}
	if i.InternalChatMessageMetadataPassthrough.CreateTime == 0 {
		i.InternalChatMessageMetadataPassthrough.CreateTime = createTime
	}
}

func (i *InputItem) UnmarshalJSON(data []byte) error {
	type wire struct {
		ID                                     string                                  `json:"id,omitempty"`
		Type                                   string                                  `json:"type"`
		Role                                   string                                  `json:"role,omitempty"`
		Content                                []ContentPart                           `json:"content,omitempty"`
		CallID                                 string                                  `json:"call_id,omitempty"`
		Name                                   string                                  `json:"name,omitempty"`
		Arguments                              string                                  `json:"arguments,omitempty"`
		Output                                 any                                     `json:"output,omitempty"`
		InternalChatMessageMetadataPassthrough *InternalChatMessageMetadataPassthrough `json:"internal_chat_message_metadata_passthrough,omitempty"`
	}
	var decoded wire
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*i = InputItem{
		ID:                                     decoded.ID,
		Type:                                   decoded.Type,
		Role:                                   decoded.Role,
		Content:                                decoded.Content,
		CallID:                                 decoded.CallID,
		Name:                                   decoded.Name,
		Arguments:                              decoded.Arguments,
		Output:                                 decoded.Output,
		InternalChatMessageMetadataPassthrough: decoded.InternalChatMessageMetadataPassthrough,
		Raw:                                    raw,
	}
	return nil
}

func (i InputItem) MarshalJSON() ([]byte, error) {
	if len(i.Raw) > 0 {
		raw := make(map[string]json.RawMessage, len(i.Raw)+1)
		for key, value := range i.Raw {
			raw[key] = append(json.RawMessage(nil), value...)
		}
		if i.Arguments != "" {
			encoded, err := json.Marshal(i.Arguments)
			if err != nil {
				return nil, err
			}
			raw["arguments"] = encoded
		}
		return json.Marshal(raw)
	}
	type wire struct {
		ID                                     string                                  `json:"id,omitempty"`
		Type                                   string                                  `json:"type"`
		Role                                   string                                  `json:"role,omitempty"`
		Content                                []ContentPart                           `json:"content,omitempty"`
		CallID                                 string                                  `json:"call_id,omitempty"`
		Name                                   string                                  `json:"name,omitempty"`
		Arguments                              string                                  `json:"arguments,omitempty"`
		Output                                 any                                     `json:"output,omitempty"`
		InternalChatMessageMetadataPassthrough *InternalChatMessageMetadataPassthrough `json:"internal_chat_message_metadata_passthrough,omitempty"`
	}
	return json.Marshal(wire{
		ID:                                     i.ID,
		Type:                                   i.Type,
		Role:                                   i.Role,
		Content:                                i.Content,
		CallID:                                 i.CallID,
		Name:                                   i.Name,
		Arguments:                              i.Arguments,
		Output:                                 i.Output,
		InternalChatMessageMetadataPassthrough: i.InternalChatMessageMetadataPassthrough,
	})
}

type ContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

func (p ContentPart) MarshalJSON() ([]byte, error) {
	switch p.Type {
	case "input_text", "output_text", "text":
		return json.Marshal(struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{
			Type: p.Type,
			Text: p.Text,
		})
	case "input_image":
		out := struct {
			Type     string `json:"type"`
			ImageURL string `json:"image_url,omitempty"`
			Detail   string `json:"detail,omitempty"`
		}{
			Type:     p.Type,
			ImageURL: p.ImageURL,
			Detail:   p.Detail,
		}
		return json.Marshal(out)
	default:
		type alias ContentPart
		return json.Marshal(alias(p))
	}
}

type Tool struct {
	Type              string             `json:"type"`
	Name              string             `json:"name,omitempty"`
	Description       string             `json:"description,omitempty"`
	Parameters        map[string]any     `json:"parameters,omitempty"`
	ExternalWebAccess *bool              `json:"external_web_access,omitempty"`
	IndexedWebAccess  *bool              `json:"indexed_web_access,omitempty"`
	Filters           *WebSearchFilters  `json:"filters,omitempty"`
	UserLocation      *WebSearchLocation `json:"user_location,omitempty"`
}

type WebSearchFilters struct {
	AllowedDomains []string `json:"allowed_domains,omitempty"`
}

type WebSearchLocation struct {
	Type     string `json:"type"`
	Country  string `json:"country,omitempty"`
	Region   string `json:"region,omitempty"`
	City     string `json:"city,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

type Credentials struct {
	AccessToken    string
	AccountID      string
	InstallationID string
	FedRAMP        bool
}

type Route struct {
	SessionID      string
	ThreadID       string
	TurnID         string
	RootTurnID     string
	ParentThreadID string
	Subagent       string
}

type SSEEvent struct {
	Event string
	Data  json.RawMessage
}
