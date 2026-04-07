package tools

import "context"

// RegistryExecutor is a ToolExecutor that dispatches to tools in a registry.
type RegistryExecutor struct {
	registry *ToolRegistry
}

// NewRegistryExecutor creates a new executor that dispatches to a registry.
func NewRegistryExecutor(registry *ToolRegistry) *RegistryExecutor {
	return &RegistryExecutor{registry: registry}
}

// Execute looks up the tool in the registry and executes it.
func (e *RegistryExecutor) Execute(ctx context.Context, toolName string, args map[string]any) (*Result, error) {
	tool, ok := e.registry.Get(toolName)
	if !ok {
		return nil, ErrToolNotFound
	}
	return tool.Execute(ctx, args)
}

// ErrToolNotFound is returned when a tool is not found in the registry.
var ErrToolNotFound = &ToolError{
	Code:    "TOOL_NOT_FOUND",
	Message: "tool not found in registry",
}

type ToolError struct {
	Code    string
	Message string
	Err     error
}

func (e *ToolError) Error() string {
	if e.Err != nil {
		return e.Code + ": " + e.Message + ": " + e.Err.Error()
	}
	return e.Code + ": " + e.Message
}

func (e *ToolError) Unwrap() error {
	return e.Err
}
