package tools

import (
	"context"
	"testing"
)

type mockTool struct {
	name        string
	description string
	executeFn   func(ctx context.Context, args map[string]any) (*Result, error)
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return m.description }
func (m *mockTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	return m.executeFn(ctx, args)
}

func TestToolInterface(t *testing.T) {
	tool := &mockTool{
		name:        "test_tool",
		description: "A test tool",
		executeFn: func(ctx context.Context, args map[string]any) (*Result, error) {
			return &Result{Success: true, Output: "executed"}, nil
		},
	}

	if tool.Name() != "test_tool" {
		t.Errorf("expected name 'test_tool', got '%s'", tool.Name())
	}
	if tool.Description() != "A test tool" {
		t.Errorf("expected description 'A test tool', got '%s'", tool.Description())
	}

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]any{"key": "value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success to be true")
	}
	if result.Output != "executed" {
		t.Errorf("expected output 'executed', got '%s'", result.Output)
	}
}

func TestToolExampleStruct(t *testing.T) {
	example := ToolExample{
		Input:       map[string]any{"arg1": "value1"},
		Output:      "expected output",
		Explanation: "how it works",
	}

	if example.Input["arg1"] != "value1" {
		t.Errorf("expected Input['arg1'] to be 'value1'")
	}
	if example.Output != "expected output" {
		t.Errorf("expected Output to be 'expected output'")
	}
	if example.Explanation != "how it works" {
		t.Errorf("expected Explanation to be 'how it works'")
	}
}
