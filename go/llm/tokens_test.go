package llm

import (
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"hello", 1},
		{"four chars", 2},
		{"exactly four", 3},
		{"five chars", 2},
		{"a b c d e f g h", 3},
	}

	for _, tt := range tests {
		result := EstimateTokens(tt.input)
		if result != tt.expected {
			t.Errorf("EstimateTokens('%s'): expected %d, got %d", tt.input, tt.expected, result)
		}
	}
}

func TestEstimateMessagesTokens(t *testing.T) {
	t.Run("empty messages", func(t *testing.T) {
		result := EstimateMessagesTokens([]Message{})
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}
	})

	t.Run("single message", func(t *testing.T) {
		messages := []Message{
			{Role: RoleUser, Content: "hello"},
		}
		result := EstimateMessagesTokens(messages)
		if result < 4 { // 4 overhead + tokens
			t.Errorf("expected at least 4, got %d", result)
		}
	})

	t.Run("message with tool calls", func(t *testing.T) {
		messages := []Message{
			{
				Role:    RoleAssistant,
				Content: "using tool",
				ToolCalls: []ToolCall{
					{Name: "my_tool", Args: map[string]any{"key": 123}},
				},
			},
		}
		result := EstimateMessagesTokens(messages)
		if result < 5 { // overhead + content + tool call
			t.Errorf("expected at least 5, got %d", result)
		}
	})
}

func TestGetModelContextWindow(t *testing.T) {
	tests := []struct {
		model    string
		expected int
	}{
		{"gpt-4o", 128000},
		{"gpt-4o-mini", 128000},
		{"gpt-4-turbo", 128000},
		{"gpt-4", 8192},
		{"gpt-3.5-turbo", 16385},
		{"claude-3-opus", 200000},
		{"claude-3-sonnet", 200000},
		{"claude-3-haiku", 200000},
		{"qwen2.5", 32768},
		{"llama-3", 8192},
		{"mistral-large", 32768},
		{"unknown-model", 4096},
		{"", 4096},
	}

	for _, tt := range tests {
		result := GetModelContextWindow(tt.model)
		if result != tt.expected {
			t.Errorf("GetModelContextWindow('%s'): expected %d, got %d", tt.model, tt.expected, result)
		}
	}
}
