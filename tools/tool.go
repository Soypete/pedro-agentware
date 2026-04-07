package tools

import "context"

type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, args map[string]any) (*Result, error)
}

type ExtendedTool interface {
	Tool
	InputSchema() map[string]any
	Examples() []ToolExample
}

type ToolExample struct {
	Input       map[string]any
	Output      string
	Explanation string
}
