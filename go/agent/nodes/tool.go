package nodes

import (
	"context"

	"github.com/soypete/pedro-agentware/go/agent"
	"github.com/soypete/pedro-agentware/go/middleware"
	"github.com/soypete/pedro-agentware/go/tools"
)

type ToolNodeConfig[T any] struct {
	ToolName     string
	Executor     middleware.ToolExecutor
	InputMapper  func(*agent.State[T]) (map[string]any, error)
	OutputMapper func(*agent.State[T], *tools.Result) error
}

func NewToolNode[T any](cfg ToolNodeConfig[T], nextNode string) agent.NodeFunc[T] {
	return func(ctx context.Context, state *agent.State[T]) (*agent.State[T], string, error) {
		args := map[string]any{}
		if cfg.InputMapper != nil {
			var err error
			args, err = cfg.InputMapper(state)
			if err != nil {
				return state, "", err
			}
		}

		result, err := cfg.Executor.Execute(ctx, cfg.ToolName, args)
		if err != nil {
			return state, "", err
		}

		if cfg.OutputMapper != nil {
			if err := cfg.OutputMapper(state, result); err != nil {
				return state, "", err
			}
		}

		return state, nextNode, nil
	}
}
