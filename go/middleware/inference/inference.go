package inference

import (
	"context"
	"errors"

	"github.com/soypete/pedro-agentware/go/llm"
	"github.com/soypete/pedro-agentware/go/middleware/guardrails"
)

var ErrRetriesExhausted = errors.New("retries exhausted")

type InferenceResult struct {
	Response        llm.Response
	NewMessages     []llm.Message
	ToolCallCounter int
	Attempts        int
}

type InferenceConfig struct {
	Client         llm.Backend
	ContextManager *llm.ContextWindowManager
	Validator      *guardrails.ResponseValidator
	ErrorTracker   *guardrails.ErrorTracker
	StepEnforcer   *guardrails.StepEnforcer
	ToolSpecs      []llm.ToolDefinition
	MaxAttempts    int
	StepIndex      int
}

func RunInference(ctx context.Context, messages []llm.Message, cfg InferenceConfig) (*InferenceResult, error) {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}

	currentMessages := make([]llm.Message, len(messages))
	copy(currentMessages, messages)

	var lastResponse llm.Response
	var toolCallCounter int

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		if cfg.ContextManager != nil {
			if cfg.ContextManager.ShouldCompact(currentMessages) {
				compacted, err := cfg.ContextManager.Compact(currentMessages)
				if err != nil {
					return nil, err
				}
				currentMessages = compacted
			}

			warning := cfg.ContextManager.CheckThresholds(ctx, currentMessages)
			if warning != "" {
				warningMsg := llm.Message{
					Role:    llm.RoleUser,
					Content: warning,
					Meta: llm.MessageMeta{
						Type: llm.MessageTypeContextWarning,
					},
				}
				currentMessages = append(currentMessages, warningMsg)
			}
		}

		req := &llm.Request{
			Messages:    currentMessages,
			Tools:       cfg.ToolSpecs,
			Temperature: 0.7,
			MaxTokens:   4096,
		}

		resp, err := cfg.Client.Complete(ctx, req)
		if err != nil {
			return nil, err
		}

		if cfg.ContextManager != nil && resp.UsageTokens.TotalTokens > 0 {
			cfg.ContextManager.UpdateTokenCount(resp.UsageTokens.TotalTokens)
		}

		var validationResult guardrails.ValidationResult

		if len(resp.ToolCalls) > 0 {
			toolCalls := make([]guardrails.ToolCall, len(resp.ToolCalls))
			for i, tc := range resp.ToolCalls {
				toolCalls[i] = guardrails.ToolCall{
					Tool: tc.Name,
					Args: tc.Args,
				}
			}
			validationResult = cfg.Validator.ValidateToolCalls(toolCalls)
		} else if resp.Content != "" {
			validationResult = cfg.Validator.ValidateTextResponse(resp.Content)
			if !validationResult.NeedsRetry && len(validationResult.ToolCalls) > 0 {
				resp.ToolCalls = make([]llm.ToolCall, len(validationResult.ToolCalls))
				for i, tc := range validationResult.ToolCalls {
					resp.ToolCalls[i] = llm.ToolCall{
						ID:   "",
						Name: tc.Tool,
						Args: tc.Args,
					}
				}
			}
		} else {
			validationResult = guardrails.ValidationResult{
				ToolCalls:  nil,
				Nudge:      guardrails.RetryNudge("empty response", getToolNames(cfg.ToolSpecs)),
				NeedsRetry: true,
			}
		}

		lastResponse = *resp

		if !validationResult.NeedsRetry {
			if cfg.ErrorTracker != nil {
				cfg.ErrorTracker.ResetSession("")
			}

			if cfg.StepEnforcer != nil && len(resp.ToolCalls) > 0 {
				for _, tc := range resp.ToolCalls {
					allowed, missing := cfg.StepEnforcer.CanExecute("", tc.Name)
					if !allowed {
						nudge := guardrails.StepNudge(tc.Name, missing, 1)
						currentMessages = append(currentMessages, llm.Message{
							Role:    llm.RoleUser,
							Content: nudge.Content,
							Meta: llm.MessageMeta{
								Type: llm.MessageTypeStepNudge,
							},
						})
						continue
					}
				}
			}

			toolCallCounter += len(resp.ToolCalls)
			return &InferenceResult{
				Response:        lastResponse,
				NewMessages:     currentMessages,
				ToolCallCounter: toolCallCounter,
				Attempts:        attempt,
			}, nil
		}

		if cfg.ErrorTracker != nil {
			cfg.ErrorTracker.RecordError("", "", nil, errors.New("validation failed"), guardrails.ErrCategoryUnknown)
		}

		if attempt >= cfg.MaxAttempts {
			return nil, ErrRetriesExhausted
		}

		if validationResult.Nudge != nil {
			nudgeMsg := llm.Message{
				Role:    llm.RoleUser,
				Content: validationResult.Nudge.Content,
				Meta: llm.MessageMeta{
					Type: llm.MessageTypeRetryNudge,
				},
			}
			currentMessages = append(currentMessages, nudgeMsg)
		}

		failedMsg := llm.Message{
			Role:    llm.RoleAssistant,
			Content: resp.Content,
			Meta: llm.MessageMeta{
				Type: llm.MessageTypeTextResponse,
			},
		}
		currentMessages = append(currentMessages, failedMsg)
	}

	return nil, ErrRetriesExhausted
}

func getToolNames(specs []llm.ToolDefinition) []string {
	names := make([]string, len(specs))
	for i, spec := range specs {
		names[i] = spec.Name
	}
	return names
}
