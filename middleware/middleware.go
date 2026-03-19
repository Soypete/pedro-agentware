package middleware

import (
	"context"
	"errors"
)

var ErrToolNotFound = errors.New("tool not found")
var ErrPolicyDenied = errors.New("policy denied")

type Middleware struct {
	executor ToolExecutor
	auditor  Auditor
	policy   PolicyEvaluator
	tools    []ToolDefinition
	toolMap  map[string]ToolDefinition
}

type MiddlewareOption func(*Middleware)

func WithAuditor(auditor Auditor) MiddlewareOption {
	return func(m *Middleware) {
		m.auditor = auditor
	}
}

func WithPolicyEvaluator(policy PolicyEvaluator) MiddlewareOption {
	return func(m *Middleware) {
		m.policy = policy
	}
}

func New(executor ToolExecutor, policy Policy, opts ...MiddlewareOption) *Middleware {
	evaluator := NewPolicyEvaluator(policy)
	mw := &Middleware{
		executor: executor,
		auditor:  NoOpAuditor{},
		policy:   evaluator,
		tools:    executor.ListTools(),
		toolMap:  make(map[string]ToolDefinition),
	}

	for _, tool := range mw.tools {
		mw.toolMap[tool.Name] = tool
	}

	for _, opt := range opts {
		opt(mw)
	}

	return mw
}

func (m *Middleware) CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error) {
	callerCtx := extractCallerContext(ctx)

	decision := m.policy.Evaluate(callerCtx, name, args)
	decision.Success = new(bool)

	if m.auditor != nil {
		m.auditor.Record(decision)
	}

	if decision.IsDenied() {
		*decision.Success = false
		return &ToolResult{
			Error:    ErrPolicyDenied,
			Metadata: map[string]interface{}{"decision": decision},
		}, ErrPolicyDenied
	}

	*decision.Success = true

	tool, exists := m.toolMap[name]
	if !exists {
		*decision.Success = false
		return &ToolResult{
			Error:    ErrToolNotFound,
			Metadata: map[string]interface{}{"decision": decision},
		}, ErrToolNotFound
	}

	result, err := m.executor.CallTool(ctx, name, args)

	if result == nil {
		result = &ToolResult{Error: err}
	}

	filtered := m.policy.FilterResult(callerCtx, name, result)
	filtered.Metadata = mergeMetadata(filtered.Metadata, map[string]interface{}{
		"decision":    decision,
		"tool_schema": tool.InputSchema,
	})

	return filtered, err
}

func (m *Middleware) ListTools() []ToolDefinition {
	if m.policy == nil {
		return m.tools
	}
	return m.policy.FilterToolList(m.tools)
}

func (m *Middleware) GetAuditor() Auditor {
	return m.auditor
}

func (m *Middleware) GetPolicy() PolicyEvaluator {
	return m.policy
}

func extractCallerContext(ctx context.Context) CallerContext {
	if ctx == nil {
		return CallerContext{}
	}

	if cc, ok := ctx.Value(callerContextKey).(CallerContext); ok {
		return cc
	}

	return CallerContext{}
}

type callerContextKeyType string

var callerContextKey callerContextKeyType = "caller-context"

func WithCallerContext(ctx context.Context, callerCtx CallerContext) context.Context {
	return context.WithValue(ctx, callerContextKey, callerCtx)
}

func mergeMetadata(a, b map[string]interface{}) map[string]interface{} {
	if a == nil {
		a = make(map[string]interface{})
	}
	for k, v := range b {
		a[k] = v
	}
	return a
}
