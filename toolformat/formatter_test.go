package toolformat

import (
	"context"
	"testing"

	"github.com/soypete/pedro-agentware/tools"
)

type mockToolForFormat struct {
	name        string
	description string
}

func (m *mockToolForFormat) Name() string        { return m.name }
func (m *mockToolForFormat) Description() string { return m.description }
func (m *mockToolForFormat) Execute(ctx context.Context, args map[string]any) (*tools.Result, error) {
	return &tools.Result{Success: true}, nil
}

func TestToolFormatterInterface(t *testing.T) {
	formatter := &QwenFormatter{}

	toolList := []tools.Tool{
		&mockToolForFormat{name: "test_tool", description: "A test tool"},
	}

	defs := formatter.FormatToolDefinitions(toolList)
	if defs == "" {
		t.Error("expected non-empty tool definitions")
	}
}

func TestQwenFormatToolDefinitions(t *testing.T) {
	formatter := &QwenFormatter{}

	t.Run("empty tools list", func(t *testing.T) {
		result := formatter.FormatToolDefinitions([]tools.Tool{})
		if result != "" {
			t.Errorf("expected empty string, got '%s'", result)
		}
	})

	t.Run("single tool", func(t *testing.T) {
		tool := &mockToolForFormat{name: "my_tool", description: "Does something"}
		result := formatter.FormatToolDefinitions([]tools.Tool{tool})
		if result == "" {
			t.Error("expected non-empty result")
		}
	})
}

func TestQwenParseToolCalls(t *testing.T) {
	formatter := &QwenFormatter{}

	t.Run("empty response", func(t *testing.T) {
		calls, err := formatter.ParseToolCalls("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls != nil {
			t.Error("expected nil calls for empty response")
		}
	})

	t.Run("valid tool call", func(t *testing.T) {
		response := `<tool_call><tool name="my_tool">{"arg1": "value1"}</tool></tool_call>`
		calls, err := formatter.ParseToolCalls(response)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(calls) != 1 {
			t.Errorf("expected 1 call, got %d", len(calls))
		}
		if calls[0].Name != "my_tool" {
			t.Errorf("expected tool name 'my_tool', got '%s'", calls[0].Name)
		}
	})

	t.Run("no tool calls in response", func(t *testing.T) {
		response := "Just some regular text response"
		calls, err := formatter.ParseToolCalls(response)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls != nil {
			t.Error("expected nil calls")
		}
	})
}

func TestQwenFormatToolResult(t *testing.T) {
	formatter := &QwenFormatter{}

	t.Run("success result", func(t *testing.T) {
		result := &tools.Result{Success: true, Output: "executed successfully"}
		output := formatter.FormatToolResult("my_tool", result)
		if output == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("error result", func(t *testing.T) {
		result := &tools.Result{Success: false, Error: "something failed"}
		output := formatter.FormatToolResult("my_tool", result)
		if output == "" {
			t.Error("expected non-empty output")
		}
	})
}

func TestQwenModelFamily(t *testing.T) {
	formatter := &QwenFormatter{}
	if formatter.ModelFamily() != "qwen" {
		t.Errorf("expected 'qwen', got '%s'", formatter.ModelFamily())
	}
}

func TestGetFormatter(t *testing.T) {
	tests := []struct {
		modelName string
		expected  string
	}{
		{"qwen2.5", "qwen"},
		{"Qwen2.5", "qwen"},
		{"llama-3", "llama"},
		{"Llama-3", "llama"},
		{"mistral-large", "mistral"},
		{"Mistral-Large", "mistral"},
		{"gpt-4", "generic"},
		{"claude-3", "generic"},
	}

	for _, tt := range tests {
		formatter := GetFormatter(tt.modelName)
		if formatter.ModelFamily() != tt.expected {
			t.Errorf("expected '%s' for model '%s', got '%s'", tt.expected, tt.modelName, formatter.ModelFamily())
		}
	}
}
