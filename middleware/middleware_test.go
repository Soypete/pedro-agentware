package middleware

import (
	"context"
	"errors"
	"testing"
)

type mockExecutor struct {
	tools     []ToolDefinition
	toolCalls map[string]int
	failTool  string
}

func newMockExecutor(tools []string) *mockExecutor {
	defs := make([]ToolDefinition, len(tools))
	for i, t := range tools {
		defs[i] = ToolDefinition{Name: t}
	}
	return &mockExecutor{
		tools:     defs,
		toolCalls: make(map[string]int),
	}
}

func (m *mockExecutor) CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error) {
	m.toolCalls[name]++
	if name == m.failTool {
		return nil, errors.New("mock error")
	}
	return &ToolResult{Content: "ok"}, nil
}

func (m *mockExecutor) ListTools() []ToolDefinition {
	return m.tools
}

func TestNew(t *testing.T) {
	executor := newMockExecutor([]string{"tool1", "tool2"})
	policy := Policy{
		Rules: []Rule{{Name: "allow-all", Tools: []string{"*"}, Action: ActionAllow}},
	}

	mw := New(executor, policy)

	if mw == nil {
		t.Fatal("New returned nil")
	}
	if mw.executor != executor {
		t.Error("executor not set correctly")
	}
	if mw.policy == nil {
		t.Error("policy not set")
	}
}

func TestMiddleware_CallTool_Allowed(t *testing.T) {
	executor := newMockExecutor([]string{"tool1"})
	policy := Policy{
		Rules: []Rule{{Name: "allow-tool1", Tools: []string{"tool1"}, Action: ActionAllow}},
	}

	mw := New(executor, policy)
	result, err := mw.CallTool(context.Background(), "tool1", nil)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("result should not be nil")
	}
}

func TestMiddleware_CallTool_Denied(t *testing.T) {
	executor := newMockExecutor([]string{"tool1"})
	policy := Policy{
		Rules:       []Rule{{Name: "deny-tool1", Tools: []string{"tool1"}, Action: ActionDeny}},
		DefaultDeny: false,
	}

	mw := New(executor, policy)
	result, err := mw.CallTool(context.Background(), "tool1", nil)

	if !errors.Is(err, ErrPolicyDenied) {
		t.Errorf("expected ErrPolicyDenied, got: %v", err)
	}
	if result == nil {
		t.Error("result should not be nil")
	} else if result.Error != ErrPolicyDenied {
		t.Errorf("expected ErrPolicyDenied in result, got: %v", result.Error)
	}
}

func TestMiddleware_CallTool_NotFound(t *testing.T) {
	executor := newMockExecutor([]string{"tool1"})
	policy := Policy{
		Rules: []Rule{{Name: "allow-all", Tools: []string{"*"}, Action: ActionAllow}},
	}

	mw := New(executor, policy)
	result, err := mw.CallTool(context.Background(), "nonexistent", nil)

	if !errors.Is(err, ErrToolNotFound) {
		t.Errorf("expected ErrToolNotFound, got: %v", err)
	}
	if result.Error != ErrToolNotFound {
		t.Errorf("expected ErrToolNotFound in result, got: %v", result.Error)
	}
}

func TestMiddleware_CallTool_ExecutorError(t *testing.T) {
	executor := newMockExecutor([]string{"tool1"})
	executor.failTool = "tool1"
	policy := Policy{
		Rules: []Rule{{Name: "allow-tool1", Tools: []string{"tool1"}, Action: ActionAllow}},
	}

	mw := New(executor, policy)
	result, err := mw.CallTool(context.Background(), "tool1", nil)

	if err == nil {
		t.Error("expected error from executor")
	}
	_ = result
}

func TestMiddleware_ListTools(t *testing.T) {
	executor := newMockExecutor([]string{"tool1", "tool2", "tool3"})
	policy := Policy{
		Rules:       []Rule{{Name: "allow-some", Tools: []string{"tool1", "tool2"}, Action: ActionAllow}},
		DefaultDeny: true,
	}

	mw := New(executor, policy)
	tools := mw.ListTools()

	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}
}

func TestMiddleware_ListTools_NoPolicy(t *testing.T) {
	executor := newMockExecutor([]string{"tool1", "tool2"})
	policy := Policy{}

	mw := New(executor, policy, WithPolicyEvaluator(nil))
	tools := mw.ListTools()

	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}
}

func TestMiddleware_GetAuditor(t *testing.T) {
	executor := newMockExecutor([]string{"tool1"})
	auditor := NewInMemoryAuditor(100)
	policy := Policy{
		Rules: []Rule{{Name: "allow-all", Tools: []string{"*"}, Action: ActionAllow}},
	}

	mw := New(executor, policy, WithAuditor(auditor))
	if mw.GetAuditor() != auditor {
		t.Error("GetAuditor returned wrong auditor")
	}
}

func TestMiddleware_GetPolicy(t *testing.T) {
	executor := newMockExecutor([]string{"tool1"})
	policy := Policy{
		Rules: []Rule{{Name: "allow-all", Tools: []string{"*"}, Action: ActionAllow}},
	}

	mw := New(executor, policy)
	if mw.GetPolicy() == nil {
		t.Error("GetPolicy returned nil")
	}
}

func TestExtractCallerContext(t *testing.T) {
	ctx := context.Background()

	cc := extractCallerContext(ctx)
	if cc.Role != "" || cc.UserID != "" || cc.SessionID != "" {
		t.Error("expected empty CallerContext for nil context")
	}

	cc = extractCallerContext(context.TODO())
	if cc.Role != "" || cc.UserID != "" || cc.SessionID != "" {
		t.Error("expected empty CallerContext for nil ctx")
	}

	callerCtx := CallerContext{Role: "admin", SessionID: "sess1"}
	ctx = WithCallerContext(context.Background(), callerCtx)
	cc = extractCallerContext(ctx)
	if cc.Role != "admin" {
		t.Errorf("expected role admin, got %s", cc.Role)
	}
	if cc.SessionID != "sess1" {
		t.Errorf("expected session sess1, got %s", cc.SessionID)
	}
}

func TestWithCallerContext(t *testing.T) {
	ctx := context.Background()
	callerCtx := CallerContext{Role: "admin"}

	newCtx := WithCallerContext(ctx, callerCtx)
	extracted := extractCallerContext(newCtx)

	if extracted.Role != "admin" {
		t.Errorf("expected admin, got %s", extracted.Role)
	}
}

func TestMergeMetadata(t *testing.T) {
	a := map[string]interface{}{"a": 1}
	b := map[string]interface{}{"b": 2}

	result := mergeMetadata(a, b)
	if result["a"] != 1 {
		t.Error("a not preserved")
	}
	if result["b"] != 2 {
		t.Error("b not added")
	}

	result = mergeMetadata(nil, b)
	if result["b"] != 2 {
		t.Error("b not added to nil map")
	}
}
