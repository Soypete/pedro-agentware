package nodes

import (
	"context"

	"github.com/soypete/pedro-agentware/go/agent"
	"github.com/soypete/pedro-agentware/go/llm"
)

type LLMNodeConfig[T any] struct {
	Backend        llm.Backend
	SystemPrompt   string
	UserPromptFn   func(*agent.State[T]) string
	Tools          []llm.ToolDefinition
	ResponseParser func(llm.Response) (T, error)
	MessageHistory []llm.Message
}

func NewLLMNode[T any](cfg LLMNodeConfig[T], nextNode string) agent.NodeFunc[T] {
	return func(ctx context.Context, state *agent.State[T]) (*agent.State[T], string, error) {
		userPrompt := ""
		if cfg.UserPromptFn != nil {
			userPrompt = cfg.UserPromptFn(state)
		}

		messages := make([]llm.Message, len(cfg.MessageHistory), len(cfg.MessageHistory)+2)
		copy(messages, cfg.MessageHistory)

		if cfg.SystemPrompt != "" {
			messages = append(messages, llm.Message{Role: llm.RoleSystem, Content: cfg.SystemPrompt})
		}
		if userPrompt != "" {
			messages = append(messages, llm.Message{Role: llm.RoleUser, Content: userPrompt})
		}

		req := &llm.Request{
			Messages: messages,
		}

		if len(cfg.Tools) > 0 {
			req.Tools = cfg.Tools
		}

		resp, err := cfg.Backend.Complete(ctx, req)
		if err != nil {
			return state, "", err
		}

		if cfg.ResponseParser != nil {
			parsed, err := cfg.ResponseParser(*resp)
			if err != nil {
				return state, "", err
			}
			state.Set(parsed)
		}

		cfg.MessageHistory = append(cfg.MessageHistory, llm.Message{Role: llm.RoleUser, Content: userPrompt})
		cfg.MessageHistory = append(cfg.MessageHistory, llm.Message{Role: llm.RoleAssistant, Content: resp.Content})

		return state, nextNode, nil
	}
}
