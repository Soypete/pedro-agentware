package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ServerConfig holds configuration for the OpenAI-compatible HTTP backend.
type ServerConfig struct {
	BaseURL       string
	APIKey        string
	Model         string
	ContextWindow int
	Timeout       time.Duration
}

// serverBackend implements Backend for OpenAI-compatible APIs.
type serverBackend struct {
	config ServerConfig
	client *http.Client
}

// NewServerBackend creates a new OpenAI-compatible backend.
func NewServerBackend(config ServerConfig) Backend {
	return &serverBackend{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Complete sends a completion request to the OpenAI-compatible server.
func (b *serverBackend) Complete(ctx context.Context, req *Request) (*Response, error) {
	payload := map[string]any{
		"model":       b.config.Model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
		"stop":        req.Stop,
	}

	if len(req.Tools) > 0 {
		tools := make([]map[string]any, 0, len(req.Tools))
		for _, t := range req.Tools {
			tools = append(tools, map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  t.InputSchema,
				},
			})
		}
		payload["tools"] = tools
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", b.config.BaseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if b.config.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+b.config.APIKey)
	}

	resp, err := b.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message      Message `json:"message"`
			FinishReason string  `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return &Response{
			Content:      "",
			FinishReason: "empty",
		}, nil
	}

	choice := result.Choices[0]
	return &Response{
		Content:      choice.Message.Content,
		ToolCalls:    choice.Message.ToolCalls,
		FinishReason: choice.FinishReason,
		UsageTokens: TokenUsage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
	}, nil
}

// SupportsNativeToolCalling returns true since OpenAI-compatible APIs support tools.
func (b *serverBackend) SupportsNativeToolCalling() bool {
	return true
}

// ModelName returns the configured model name.
func (b *serverBackend) ModelName() string {
	return b.config.Model
}

// ContextWindowSize returns the configured context window size.
func (b *serverBackend) ContextWindowSize() int {
	return b.config.ContextWindow
}
