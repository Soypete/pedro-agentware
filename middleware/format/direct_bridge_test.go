package format

import (
	"context"
	"testing"

	"github.com/pedro/agent-middleware/types"
)

type mockExecutor struct {
	tools []types.ToolDefinition
}

func (e *mockExecutor) CallTool(ctx context.Context, name string, args map[string]interface{}) (*types.ToolResult, error) {
	return &types.ToolResult{Content: "result"}, nil
}

func (e *mockExecutor) ListTools() []types.ToolDefinition {
	return e.tools
}

func TestDirectBridge_FormatTools(t *testing.T) {
	executor := &mockExecutor{
		tools: []types.ToolDefinition{
			{Name: "tool1", Description: "desc1"},
			{Name: "tool2", Description: "desc2"},
		},
	}

	bridge := NewDirectBridge(executor, ModelFamilyClaude)

	result := bridge.FormatTools(executor.ListTools())
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestDirectBridge_ParseToolCalls(t *testing.T) {
	executor := &mockExecutor{}
	bridge := NewDirectBridge(executor, ModelFamilyClaude)

	calls, err := bridge.ParseToolCalls(`<tool_call><tool name="test_tool"></tool_call>`)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(calls) == 0 {
		t.Error("expected tool calls to be parsed")
	}
}

func TestIsMCPCapable(t *testing.T) {
	tests := []struct {
		family  ModelFamily
		capable bool
	}{
		{ModelFamilyClaude, true},
		{ModelFamilyOpenAI, true},
		{ModelFamilyGLM4, true},
		{ModelFamilyLlama3, false},
		{ModelFamilyQwen, false},
		{ModelFamilyMistral, false},
	}

	for _, tt := range tests {
		result := IsMCPCapable(tt.family)
		if result != tt.capable {
			t.Errorf("IsMCPCapable(%v) = %v, want %v", tt.family, result, tt.capable)
		}
	}
}

func TestDetectModelFamily(t *testing.T) {
	tests := []struct {
		response string
		expected ModelFamily
	}{
		{"This is Claude", ModelFamilyClaude},
		{"Using GPT-4", ModelFamilyOpenAI},
		{"GLM-4 model", ModelFamilyGLM4},
		{"Llama 3", ModelFamilyLlama3},
		{"Qwen model", ModelFamilyQwen},
		{"Mistral AI", ModelFamilyMistral},
		{"unknown model", ModelFamilyClaude},
	}

	for _, tt := range tests {
		result := DetectModelFamily(tt.response)
		if result != tt.expected {
			t.Errorf("DetectModelFamily(%q) = %v, want %v", tt.response, result, tt.expected)
		}
	}
}

func TestDirectBridge_ExecuteTool(t *testing.T) {
	executor := &mockExecutor{}
	bridge := NewDirectBridge(executor, ModelFamilyClaude)

	result, err := bridge.ExecuteTool(context.Background(), ToolCall{
		Name:      "test_tool",
		Arguments: map[string]interface{}{"key": "value"},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil || result.Content != "result" {
		t.Error("expected result with content 'result'")
	}
}
