package provider

import (
	"encoding/json"
	"time"
)

type Config struct {
	Model   string
	APIKey  string
	Token   any
	BaseURL string

	// * cloudflare
	AccountID string
	GatewayID string
}

type Models struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type CopilotModels struct {
	Data []struct {
		ID                 string `json:"id"`
		ModelPickerEnabled bool   `json:"model_picker_enabled"`
		Policy             struct {
			State string `json:"state"`
		} `json:"policy"`
	} `json:"data"`
}

type GeminiModels struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

type CloudFlareModels struct {
	Result []struct {
		Name string `json:"name"`
		Task struct {
			Name string `json:"name"`
		} `json:"task"`
	} `json:"result"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type Message struct {
	Role             string     `json:"role"`
	Content          any        `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
}

type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type CopilotToken struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	Scope       string    `json:"scope"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type CopilotRefreshToken struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

type GrokToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func (t *GrokToken) Expired() bool {
	return time.Now().After(t.ExpiresAt.Add(-60 * time.Second))
}

type CodexToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	IDToken      string    `json:"id_token"`
	AccountID    string    `json:"account_id"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func (t *CodexToken) Expired() bool {
	return time.Now().After(t.ExpiresAt.Add(-60 * time.Second))
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
	ThoughtSignature string `json:"thought_signature,omitempty"` // Gemini thinking models
}

type Output struct {
	Choices []OutputChoices `json:"choices"`
	Usage   Usage           `json:"usage"`
	Error   *struct {
		Message string      `json:"message"`
		Type    string      `json:"type"`
		Code    json.Number `json:"code"`
	} `json:"error,omitempty"`
}

type OutputChoices struct {
	Message      Message `json:"message"`
	Delta        Message `json:"delta"`
	FinishReason string  `json:"finish_reason,omitempty"`
}

type Usage struct {
	Input       int `json:"input_tokens"`
	Output      int `json:"output_tokens"`
	CacheCreate int `json:"cache_creation_input_tokens,omitempty"`
	CacheRead   int `json:"cache_read_input_tokens,omitempty"`
}

func (u *Usage) UnmarshalJSON(data []byte) error {
	var raw struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		PromptTokens             int `json:"prompt_tokens"`
		CompletionTokens         int `json:"completion_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		PromptTokensDetails      struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	cached := raw.CacheReadInputTokens + raw.PromptTokensDetails.CachedTokens
	u.Input = raw.InputTokens + raw.PromptTokens - raw.PromptTokensDetails.CachedTokens
	u.Output = raw.OutputTokens + raw.CompletionTokens
	u.CacheCreate = raw.CacheCreationInputTokens
	u.CacheRead = cached
	return nil
}
