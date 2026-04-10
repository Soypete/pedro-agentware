package llm

import (
	"testing"
)

func TestRequest(t *testing.T) {
	req := Request{
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
			{Role: RoleAssistant, Content: "Hi there"},
		},
		Tools: []ToolDefinition{
			{
				Name:        "test_tool",
				Description: "A test tool",
				InputSchema: map[string]any{"type": "object"},
			},
		},
		Temperature: 0.7,
		MaxTokens:   1000,
		Stop:        []string{"STOP", "DONE"},
	}

	if len(req.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(req.Messages))
	}
	if len(req.Tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(req.Tools))
	}
	if req.Temperature != 0.7 {
		t.Errorf("expected 0.7, got %f", req.Temperature)
	}
}

func TestMessage(t *testing.T) {
	msg := Message{
		Role:       RoleAssistant,
		Content:    "Using a tool",
		ToolCallID: "call_123",
		ToolCalls: []ToolCall{
			{ID: "call_1", Name: "tool1", Args: map[string]any{"arg": "value"}},
		},
	}

	if msg.Role != RoleAssistant {
		t.Errorf("expected RoleAssistant, got '%s'", msg.Role)
	}
	if msg.ToolCallID != "call_123" {
		t.Errorf("expected 'call_123', got '%s'", msg.ToolCallID)
	}
	if len(msg.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(msg.ToolCalls))
	}
}

func TestRoleConstants(t *testing.T) {
	if RoleSystem != "system" {
		t.Errorf("expected 'system', got '%s'", RoleSystem)
	}
	if RoleUser != "user" {
		t.Errorf("expected 'user', got '%s'", RoleUser)
	}
	if RoleAssistant != "assistant" {
		t.Errorf("expected 'assistant', got '%s'", RoleAssistant)
	}
	if RoleTool != "tool" {
		t.Errorf("expected 'tool', got '%s'", RoleTool)
	}
}

func TestToolDefinition(t *testing.T) {
	def := ToolDefinition{
		Name:        "my_tool",
		Description: "Does something",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"arg1": map[string]any{"type": "string"},
			},
		},
	}

	if def.Name != "my_tool" {
		t.Errorf("expected 'my_tool', got '%s'", def.Name)
	}
	if def.InputSchema["type"] != "object" {
		t.Errorf("expected 'object', got '%v'", def.InputSchema["type"])
	}
}
