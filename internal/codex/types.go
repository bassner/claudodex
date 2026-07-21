package codex

import "encoding/json"

const DefaultBaseURL = "https://chatgpt.com/backend-api"

type Request struct {
	Model              string            `json:"model"`
	Instructions       string            `json:"instructions,omitempty"`
	PreviousResponseID string            `json:"previous_response_id,omitempty"`
	Input              []InputItem       `json:"input"`
	Tools              []Tool            `json:"tools,omitempty"`
	ToolChoice         any               `json:"tool_choice,omitempty"`
	ParallelToolCalls  bool              `json:"parallel_tool_calls"`
	Store              bool              `json:"store"`
	Stream             bool              `json:"stream"`
	Include            []string          `json:"include,omitempty"`
	ServiceTier        string            `json:"service_tier,omitempty"`
	Reasoning          *Reasoning        `json:"reasoning,omitempty"`
	Text               *TextConfig       `json:"text,omitempty"`
	PromptCacheKey     string            `json:"prompt_cache_key,omitempty"`
	ClientMetadata     map[string]string `json:"client_metadata,omitempty"`
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
	Type      string                     `json:"type"`
	Role      string                     `json:"role,omitempty"`
	Content   []ContentPart              `json:"content,omitempty"`
	CallID    string                     `json:"call_id,omitempty"`
	Name      string                     `json:"name,omitempty"`
	Arguments string                     `json:"arguments,omitempty"`
	Output    any                        `json:"output,omitempty"`
	Raw       map[string]json.RawMessage `json:"-"`
}

func (i *InputItem) UnmarshalJSON(data []byte) error {
	type wire struct {
		Type      string        `json:"type"`
		Role      string        `json:"role,omitempty"`
		Content   []ContentPart `json:"content,omitempty"`
		CallID    string        `json:"call_id,omitempty"`
		Name      string        `json:"name,omitempty"`
		Arguments string        `json:"arguments,omitempty"`
		Output    any           `json:"output,omitempty"`
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
		Type:      decoded.Type,
		Role:      decoded.Role,
		Content:   decoded.Content,
		CallID:    decoded.CallID,
		Name:      decoded.Name,
		Arguments: decoded.Arguments,
		Output:    decoded.Output,
		Raw:       raw,
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
		Type      string        `json:"type"`
		Role      string        `json:"role,omitempty"`
		Content   []ContentPart `json:"content,omitempty"`
		CallID    string        `json:"call_id,omitempty"`
		Name      string        `json:"name,omitempty"`
		Arguments string        `json:"arguments,omitempty"`
		Output    any           `json:"output,omitempty"`
	}
	return json.Marshal(wire{
		Type:      i.Type,
		Role:      i.Role,
		Content:   i.Content,
		CallID:    i.CallID,
		Name:      i.Name,
		Arguments: i.Arguments,
		Output:    i.Output,
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
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
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
	ParentThreadID string
	Subagent       string
}

type SSEEvent struct {
	Event string
	Data  json.RawMessage
}
