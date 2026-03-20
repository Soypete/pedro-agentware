package middleware

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/soypete/pedro-agentware/middleware/types"
)

var ErrToolNotFound = errors.New("tool not found")
var ErrPolicyDenied = errors.New("policy denied")

type CallHistory struct {
	CalledTools map[string]bool
	FailedTools map[string]int
	mu          sync.RWMutex
}

func NewCallHistory() *CallHistory {
	return &CallHistory{
		CalledTools: make(map[string]bool),
		FailedTools: make(map[string]int),
	}
}

func (ch *CallHistory) RecordToolCall(toolName string, success bool) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if success {
		ch.CalledTools[toolName] = true
		delete(ch.FailedTools, toolName)
	} else {
		ch.FailedTools[toolName]++
	}
}

func (ch *CallHistory) GetCalledTools() map[string]bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	result := make(map[string]bool)
	for k, v := range ch.CalledTools {
		result[k] = v
	}
	return result
}

func (ch *CallHistory) GetFailedTools() map[string]int {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	result := make(map[string]int)
	for k, v := range ch.FailedTools {
		result[k] = v
	}
	return result
}

func (ch *CallHistory) Reset() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.CalledTools = make(map[string]bool)
	ch.FailedTools = make(map[string]int)
}

func (ch *CallHistory) ResetPhase(phase string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	for tool := range ch.CalledTools {
		if phase == "" || strings.HasPrefix(tool, phase+":") {
			delete(ch.CalledTools, tool)
		}
	}
}

const MaxFailures = 3

func (ch *CallHistory) HasToolFailedTooManyTimes(toolName string) bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	return ch.FailedTools[toolName] >= MaxFailures
}

func (ch *CallHistory) WasToolCalled(toolName string) bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	return ch.CalledTools[toolName]
}

type Middleware struct {
	executor    types.ToolExecutor
	auditor     Auditor
	policy      PolicyEvaluator
	tools       []types.ToolDefinition
	toolMap     map[string]types.ToolDefinition
	callHistory *CallHistory
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
		executor:    executor,
		auditor:     NoOpAuditor{},
		policy:      evaluator,
		tools:       executor.ListTools(),
		toolMap:     make(map[string]ToolDefinition),
		callHistory: NewCallHistory(),
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

	success := err == nil
	if result != nil && result.Error != nil {
		success = false
	}

	toolKey := name
	if callerCtx.Phase != "" {
		toolKey = callerCtx.Phase + ":" + name
	}
	m.callHistory.RecordToolCall(toolKey, success)

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
		return m.filterFailedTools(m.tools, "")
	}
	tools := m.policy.FilterToolList(m.tools)
	return m.filterFailedTools(tools, "")
}

func (m *Middleware) GetToolsWithContext(ctx context.Context) []ToolDefinition {
	callerCtx := extractCallerContext(ctx)
	return m.GetToolsForPhase("", callerCtx)
}

func (m *Middleware) GetAuditor() Auditor {
	return m.auditor
}

func (m *Middleware) GetPolicy() PolicyEvaluator {
	return m.policy
}

func (m *Middleware) GetCallHistory() *CallHistory {
	return m.callHistory
}

// GetToolsForPhase returns the list of tools available for a given phase.
// A "phase" represents a stage in a multi-phase agent workflow (from PedroCLI's
// phased executor). Each phase can have different tools available.
// This method filters tools to exclude:
//   - Tools already called in this phase (prevents loops)
//   - Tools that have failed 3+ times in this phase (gives up after repeated failures)
//
// The phase is either passed directly as the phase argument or extracted from
// callerCtx.Phase. Tools are tracked with keys like "phaseName:toolName" to
// maintain per-phase call history.
func (m *Middleware) GetToolsForPhase(phase string, callerCtx CallerContext) []ToolDefinition {
	if m.policy == nil {
		return m.tools
	}

	tools := m.policy.FilterToolList(m.tools)

	if phase == "" && callerCtx.Phase == "" {
		return tools
	}

	activePhase := phase
	if activePhase == "" {
		activePhase = callerCtx.Phase
	}

	if activePhase == "" {
		return tools
	}

	var filtered []ToolDefinition
	for _, tool := range tools {
		key := activePhase + ":" + tool.Name
		if m.callHistory.WasToolCalled(key) {
			continue
		}
		if m.callHistory.HasToolFailedTooManyTimes(tool.Name) {
			continue
		}
		filtered = append(filtered, tool)
	}

	return filtered
}

func (m *Middleware) filterFailedTools(tools []ToolDefinition, phase string) []ToolDefinition {
	var filtered []ToolDefinition
	for _, tool := range tools {
		checkKey := tool.Name
		if phase != "" {
			checkKey = phase + ":" + tool.Name
		}
		if m.callHistory.HasToolFailedTooManyTimes(checkKey) {
			continue
		}
		filtered = append(filtered, tool)
	}
	return filtered
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
