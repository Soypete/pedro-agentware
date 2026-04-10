package tools

import (
	"context"
	"sort"
	"testing"
)

type testTool struct {
	name        string
	description string
}

func (t *testTool) Name() string        { return t.name }
func (t *testTool) Description() string { return t.description }
func (t *testTool) Execute(ctx context.Context, args map[string]any) (*Result, error) {
	return &Result{Success: true}, nil
}

type testExtendedTool struct {
	testTool
}

func (t *testExtendedTool) InputSchema() map[string]any { return map[string]any{"type": "object"} }
func (t *testExtendedTool) Examples() []ToolExample     { return nil }

func TestNewToolRegistry(t *testing.T) {
	registry := NewToolRegistry()
	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
	if registry.tools == nil {
		t.Error("expected tools map to be initialized")
	}
}

func TestToolRegistryRegister(t *testing.T) {
	registry := NewToolRegistry()
	tool := &testTool{name: "tool1", description: "Tool 1"}
	registry.Register(tool)

	if len(registry.tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(registry.tools))
	}
}

func TestToolRegistryGet(t *testing.T) {
	registry := NewToolRegistry()
	tool := &testTool{name: "my_tool", description: "My tool"}
	registry.Register(tool)

	t.Run("existing tool", func(t *testing.T) {
		result, ok := registry.Get("my_tool")
		if !ok {
			t.Error("expected to find tool")
		}
		if result.Name() != "my_tool" {
			t.Errorf("expected 'my_tool', got '%s'", result.Name())
		}
	})

	t.Run("non-existing tool", func(t *testing.T) {
		_, ok := registry.Get("nonexistent")
		if ok {
			t.Error("expected not to find tool")
		}
	})
}

func TestToolRegistryAll(t *testing.T) {
	registry := NewToolRegistry()
	registry.Register(&testTool{name: "tool_b", description: "B"})
	registry.Register(&testTool{name: "tool_a", description: "A"})
	registry.Register(&testTool{name: "tool_c", description: "C"})

	tools := registry.All()
	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}

	if tools[0].Name() != "tool_a" {
		t.Errorf("expected first tool to be 'tool_a', got '%s'", tools[0].Name())
	}
	if tools[1].Name() != "tool_b" {
		t.Errorf("expected second tool to be 'tool_b', got '%s'", tools[1].Name())
	}
	if tools[2].Name() != "tool_c" {
		t.Errorf("expected third tool to be 'tool_c', got '%s'", tools[2].Name())
	}
}

func TestToolRegistryNames(t *testing.T) {
	registry := NewToolRegistry()
	registry.Register(&testTool{name: "z_tool", description: "Z"})
	registry.Register(&testTool{name: "a_tool", description: "A"})

	names := registry.Names()
	if len(names) != 2 {
		t.Errorf("expected 2 names, got %d", len(names))
	}

	sortedNames := make([]string, len(names))
	copy(sortedNames, names)
	sort.Strings(sortedNames)

	if names[0] != "a_tool" && names[0] != "z_tool" {
		t.Errorf("unexpected name: %s", names[0])
	}
}

func TestToolRegistrySchemas(t *testing.T) {
	registry := NewToolRegistry()
	registry.Register(&testTool{name: "basic_tool", description: "Basic"})
	registry.Register(&testExtendedTool{
		testTool: testTool{name: "extended_tool", description: "Extended"},
	})
	registry.tools["extended_tool"] = &testExtendedTool{
		testTool: testTool{name: "extended_tool", description: "Extended"},
	}

	schemas := registry.Schemas()
	if len(schemas) != 1 {
		t.Errorf("expected 1 schema, got %d", len(schemas))
	}
	if schemas["extended_tool"]["type"] != "object" {
		t.Errorf("expected schema type 'object', got '%v'", schemas["extended_tool"]["type"])
	}
	if _, ok := schemas["basic_tool"]; ok {
		t.Error("expected basic_tool not to have schema")
	}
}
