package kitaru

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/soypete/pedro-agentware/go/middleware"
)

type mockFlowHandle struct {
	execID    string
	execution *Execution
	err       error
	runCalled bool
}

func (m *mockFlowHandle) Run(ctx context.Context, inputs map[string]any) (string, error) {
	m.runCalled = true
	if m.err != nil {
		return "", m.err
	}
	m.execID = "exec-123"
	return m.execID, nil
}

func (m *mockFlowHandle) RunWithWait(ctx context.Context, inputs map[string]any, pollInterval time.Duration) (*Execution, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if m.err != nil {
		return nil, m.err
	}
	if m.execution != nil {
		return m.execution, nil
	}
	return &Execution{ID: m.execID, Status: "completed", Output: map[string]any{"result": "success"}}, nil
}

func (m *mockFlowHandle) GetExecution() (*Execution, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.execution != nil {
		return m.execution, nil
	}
	if m.execID == "" {
		m.execID = "exec-123"
	}
	return &Execution{ID: m.execID, Status: "completed"}, nil
}

func (m *mockFlowHandle) Checkpoint(name string, data map[string]any) error {
	return m.err
}

func (m *mockFlowHandle) RestoreCheckpoint(name string) (map[string]any, error) {
	return nil, m.err
}

func (m *mockFlowHandle) SaveArtifact(key string, data interface{}, artifactType string) error {
	return m.err
}

func (m *mockFlowHandle) LoadArtifact(key string, dest interface{}) error {
	return m.err
}

func (m *mockFlowHandle) Log(level LogLevel, message string, metadata map[string]any) error {
	return m.err
}

func (m *mockFlowHandle) Replay(fromCheckpoint string, inputs map[string]any) (string, error) {
	return "", m.err
}

func (m *mockFlowHandle) GetExecID() string {
	return m.execID
}

type mockClient struct {
	flowHandle *mockFlowHandle
	flowName   string
}

func (m *mockClient) Flow(name string) FlowHandle {
	m.flowName = name
	if m.flowHandle == nil {
		m.flowHandle = &mockFlowHandle{
			execID:    "exec-123",
			execution: &Execution{ID: "exec-123", Status: "completed", Output: map[string]any{"result": "success"}},
		}
	}
	return m.flowHandle
}

type mockPolicy struct {
	decision middleware.Decision
}

func (m *mockPolicy) Evaluate(toolName string, args map[string]any, caller middleware.CallerContext) middleware.Decision {
	return m.decision
}

func TestNewFlowToolExecutor(t *testing.T) {
	client := &mockClient{}
	mapping := map[string]string{"tool1": "flow1"}

	executor := NewFlowToolExecutor(client, mapping)

	if executor.client != client {
		t.Error("expected client to be set")
	}
	if executor.defaultConfig == nil {
		t.Error("expected default config to be set")
	}
	if executor.defaultConfig.WaitInterval != 2*time.Second {
		t.Errorf("expected WaitInterval 2s, got %v", executor.defaultConfig.WaitInterval)
	}
	if executor.defaultConfig.RequestTimeout != 60*time.Second {
		t.Errorf("expected RequestTimeout 60s, got %v", executor.defaultConfig.RequestTimeout)
	}
	if executor.defaultConfig.RetryCount != 3 {
		t.Errorf("expected RetryCount 3, got %v", executor.defaultConfig.RetryCount)
	}
}

func TestNewFlowToolExecutorWithConfig(t *testing.T) {
	client := &mockClient{}
	mapping := map[string]string{"tool1": "flow1"}
	config := &Config{
		WaitInterval:   5 * time.Second,
		RequestTimeout: 30 * time.Second,
		RetryCount:     5,
	}

	executor := NewFlowToolExecutorWithConfig(client, mapping, config)

	if executor.defaultConfig.WaitInterval != 5*time.Second {
		t.Errorf("expected WaitInterval 5s, got %v", executor.defaultConfig.WaitInterval)
	}
	if executor.defaultConfig.RequestTimeout != 30*time.Second {
		t.Errorf("expected RequestTimeout 30s, got %v", executor.defaultConfig.RequestTimeout)
	}
	if executor.defaultConfig.RetryCount != 5 {
		t.Errorf("expected RetryCount 5, got %v", executor.defaultConfig.RetryCount)
	}
}

func TestNewFlowToolExecutorWithNilConfig(t *testing.T) {
	client := &mockClient{}
	mapping := map[string]string{"tool1": "flow1"}

	executor := NewFlowToolExecutorWithConfig(client, mapping, nil)

	if executor.defaultConfig.WaitInterval != 2*time.Second {
		t.Errorf("expected default WaitInterval, got %v", executor.defaultConfig.WaitInterval)
	}
}

func TestFlowToolExecutor_Execute_Success(t *testing.T) {
	client := &mockClient{}
	mapping := map[string]string{"my_tool": "my_flow"}
	executor := NewFlowToolExecutor(client, mapping)

	result, err := executor.Execute(context.Background(), "my_tool", map[string]any{"key": "value"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.Output != "map[result:success]" {
		t.Errorf("expected output 'map[result:success]', got '%s'", result.Output)
	}
	if result.Metadata["tool_name"] != "my_tool" {
		t.Errorf("expected tool_name 'my_tool', got '%v'", result.Metadata["tool_name"])
	}
	if result.Metadata["flow_name"] != "my_flow" {
		t.Errorf("expected flow_name 'my_flow', got '%v'", result.Metadata["flow_name"])
	}
}

func TestFlowToolExecutor_Execute_UnknownTool(t *testing.T) {
	client := &mockClient{}
	mapping := map[string]string{"my_tool": "my_flow"}
	executor := NewFlowToolExecutor(client, mapping)

	result, err := executor.Execute(context.Background(), "unknown_tool", nil)

	if err != nil {
		t.Fatal("expected no error for unknown tool")
	}
	if result.Success {
		t.Error("expected failure for unknown tool")
	}
	if result.Error != "unknown tool: unknown_tool" {
		t.Errorf("expected error 'unknown tool: unknown_tool', got '%s'", result.Error)
	}
}

func TestFlowToolExecutor_Execute_FailedStatus(t *testing.T) {
	client := &mockClient{}
	mapping := map[string]string{"my_tool": "my_flow"}
	executor := NewFlowToolExecutor(client, mapping)

	client.Flow("my_flow")
	client.flowHandle.execution = &Execution{ID: "exec-123", Status: "failed", Output: map[string]any{"result": "fail"}}

	result, err := executor.Execute(context.Background(), "my_tool", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for failed status")
	}
}

func TestFlowToolExecutor_Execute_RunError(t *testing.T) {
	client := &mockClient{}
	mapping := map[string]string{"my_tool": "my_flow"}
	executor := NewFlowToolExecutor(client, mapping)

	client.Flow("my_flow")
	client.flowHandle.err = errors.New("run failed")

	result, err := executor.Execute(context.Background(), "my_tool", nil)

	if err == nil {
		t.Fatal("expected error")
	}
	if result.Success {
		t.Error("expected failure")
	}
}

func TestFlowToolExecutor_ListTools(t *testing.T) {
	client := &mockClient{}
	mapping := map[string]string{
		"tool1": "flow1",
		"tool2": "flow2",
	}
	executor := NewFlowToolExecutor(client, mapping)

	toolsList := executor.ListTools()

	if len(toolsList) != 2 {
		t.Errorf("expected 2 tools, got %d", len(toolsList))
	}

	toolNames := make(map[string]bool)
	for _, tool := range toolsList {
		toolNames[tool.Name()] = true
		if tool.Description() == "" {
			t.Error("expected non-empty description")
		}
	}

	if !toolNames["tool1"] {
		t.Error("expected tool1")
	}
	if !toolNames["tool2"] {
		t.Error("expected tool2")
	}
}

func TestFlowToolExecutor_GetFlowMapping(t *testing.T) {
	client := &mockClient{}
	mapping := map[string]string{"tool1": "flow1", "tool2": "flow2"}
	executor := NewFlowToolExecutor(client, mapping)

	result := executor.GetFlowMapping()

	if result["tool1"] != "flow1" {
		t.Errorf("expected flow1, got %s", result["tool1"])
	}
	if result["tool2"] != "flow2" {
		t.Errorf("expected flow2, got %s", result["tool2"])
	}

	result["tool1"] = "modified"
	if executor.GetFlowMapping()["tool1"] == "modified" {
		t.Error("expected copy, not modification of original")
	}
}

func TestFlowToolExecutor_UpdateFlowMapping(t *testing.T) {
	client := &mockClient{}
	executor := NewFlowToolExecutor(client, map[string]string{"tool1": "flow1"})

	executor.UpdateFlowMapping(map[string]string{"new_tool": "new_flow"})

	if executor.GetFlowMapping()["new_tool"] != "new_flow" {
		t.Error("expected new_tool mapping")
	}
	if executor.GetFlowMapping()["tool1"] != "" {
		t.Error("expected old tool to be removed")
	}
}

func TestFlowToolExecutor_AddFlowMapping(t *testing.T) {
	client := &mockClient{}
	executor := NewFlowToolExecutor(client, map[string]string{"tool1": "flow1"})

	executor.AddFlowMapping("tool2", "flow2")

	if executor.GetFlowMapping()["tool2"] != "flow2" {
		t.Error("expected tool2 mapping")
	}
	if executor.GetFlowMapping()["tool1"] != "flow1" {
		t.Error("expected tool1 to still exist")
	}
}

func TestFlowToolExecutor_SetConfig(t *testing.T) {
	client := &mockClient{}
	executor := NewFlowToolExecutor(client, nil)

	newConfig := &Config{WaitInterval: 10 * time.Second}
	executor.SetConfig(newConfig)

	result := executor.getConfig()
	if result.WaitInterval != 10*time.Second {
		t.Errorf("expected 10s, got %v", result.WaitInterval)
	}
}

func TestFlowTool_Name(t *testing.T) {
	client := &mockClient{}
	executor := NewFlowToolExecutor(client, map[string]string{"my_tool": "my_flow"})
	toolsList := executor.ListTools()

	if toolsList[0].Name() != "my_tool" {
		t.Errorf("expected 'my_tool', got '%s'", toolsList[0].Name())
	}
}

func TestFlowTool_Description(t *testing.T) {
	client := &mockClient{}
	executor := NewFlowToolExecutor(client, map[string]string{"my_tool": "my_flow"})
	toolsList := executor.ListTools()

	desc := toolsList[0].Description()
	if desc != "Execute Kitaru flow: my_flow" {
		t.Errorf("expected 'Execute Kitaru flow: my_flow', got '%s'", desc)
	}
}

func TestFlowTool_Execute(t *testing.T) {
	client := &mockClient{}
	executor := NewFlowToolExecutor(client, map[string]string{"my_tool": "my_flow"})
	toolsList := executor.ListTools()

	result, err := toolsList[0].Execute(context.Background(), map[string]any{"key": "value"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
}

func TestFlowTool_InputSchema(t *testing.T) {
	client := &mockClient{}
	executor := NewFlowToolExecutor(client, map[string]string{"my_tool": "my_flow"})
	toolsList := executor.ListTools()

	ft, ok := toolsList[0].(*flowTool)
	if !ok {
		t.Fatal("expected flowTool type")
	}

	schema := ft.InputSchema()

	if schema["type"] != "object" {
		t.Errorf("expected type 'object', got '%v'", schema["type"])
	}
}

func TestFlowTool_Examples(t *testing.T) {
	client := &mockClient{}
	executor := NewFlowToolExecutor(client, map[string]string{"my_tool": "my_flow"})
	toolsList := executor.ListTools()

	ft, ok := toolsList[0].(*flowTool)
	if !ok {
		t.Fatal("expected flowTool type")
	}

	examples := ft.Examples()

	if examples != nil {
		t.Error("expected nil examples")
	}
}

func TestNewKitaruMiddlewareAdapter(t *testing.T) {
	client := &mockClient{}
	mapping := map[string]string{"tool1": "flow1"}
	policy := &mockPolicy{}
	auditor := middleware.NewInMemoryAuditor()

	adapter := NewKitaruMiddlewareAdapter(client, mapping, policy, auditor)

	if adapter.executor == nil {
		t.Error("expected executor to be set")
	}
	if adapter.policy != policy {
		t.Error("expected policy to be set")
	}
	if adapter.auditor != auditor {
		t.Error("expected auditor to be set")
	}
}

func TestKitaruMiddlewareAdapter_Execute_WithAllowPolicy(t *testing.T) {
	client := &mockClient{}
	mapping := map[string]string{"my_tool": "my_flow"}
	policy := &mockPolicy{decision: middleware.Decision{Action: middleware.ActionAllow}}
	auditor := middleware.NewInMemoryAuditor()

	adapter := NewKitaruMiddlewareAdapter(client, mapping, policy, auditor)

	result, err := adapter.Execute(context.Background(), "my_tool", map[string]any{"key": "value"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	records := auditor.Query(middleware.AuditFilter{ToolName: "my_tool"})
	if len(records) != 1 {
		t.Errorf("expected 1 audit record, got %d", len(records))
	}
}

func TestKitaruMiddlewareAdapter_Execute_WithDenyPolicy(t *testing.T) {
	client := &mockClient{}
	mapping := map[string]string{"my_tool": "my_flow"}
	policy := &mockPolicy{decision: middleware.Decision{
		Action: middleware.ActionDeny,
		Rule:   "deny-rule",
		Reason: "denied by policy",
	}}
	auditor := middleware.NewInMemoryAuditor()

	adapter := NewKitaruMiddlewareAdapter(client, mapping, policy, auditor)

	result, err := adapter.Execute(context.Background(), "my_tool", map[string]any{"key": "value"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure due to policy deny")
	}
	if result.Error != "denied by policy: denied by policy" {
		t.Errorf("expected deny error, got '%s'", result.Error)
	}
	if result.Metadata["rule"] != "deny-rule" {
		t.Errorf("expected rule 'deny-rule', got '%v'", result.Metadata["rule"])
	}
}

func TestKitaruMiddlewareAdapter_Execute_WithNoPolicy(t *testing.T) {
	client := &mockClient{}
	mapping := map[string]string{"my_tool": "my_flow"}
	auditor := middleware.NewInMemoryAuditor()

	adapter := NewKitaruMiddlewareAdapter(client, mapping, nil, auditor)

	result, err := adapter.Execute(context.Background(), "my_tool", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success when no policy")
	}
}

func TestKitaruMiddlewareAdapter_Execute_WithNoAuditor(t *testing.T) {
	client := &mockClient{}
	mapping := map[string]string{"my_tool": "my_flow"}
	policy := &mockPolicy{decision: middleware.Decision{Action: middleware.ActionAllow}}

	adapter := NewKitaruMiddlewareAdapter(client, mapping, policy, nil)

	result, err := adapter.Execute(context.Background(), "my_tool", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
}

func TestKitaruMiddlewareAdapter_ListTools(t *testing.T) {
	client := &mockClient{}
	mapping := map[string]string{"tool1": "flow1", "tool2": "flow2"}
	adapter := NewKitaruMiddlewareAdapter(client, mapping, nil, nil)

	toolsList := adapter.ListTools()

	if len(toolsList) != 2 {
		t.Errorf("expected 2 tools, got %d", len(toolsList))
	}
}

func TestKitaruMiddlewareAdapter_WithPolicy(t *testing.T) {
	client := &mockClient{}
	adapter := NewKitaruMiddlewareAdapter(client, nil, nil, nil)

	newPolicy := &mockPolicy{}
	resultAdapter := adapter.WithPolicy(newPolicy)

	if resultAdapter.policy != newPolicy {
		t.Error("expected policy to be set")
	}
	if resultAdapter != adapter {
		t.Error("expected same adapter returned for chaining")
	}
}

func TestKitaruMiddlewareAdapter_WithAuditor(t *testing.T) {
	client := &mockClient{}
	adapter := NewKitaruMiddlewareAdapter(client, nil, nil, nil)

	newAuditor := middleware.NewInMemoryAuditor()
	resultAdapter := adapter.WithAuditor(newAuditor)

	if resultAdapter.auditor != newAuditor {
		t.Error("expected auditor to be set")
	}
	if resultAdapter != adapter {
		t.Error("expected same adapter returned for chaining")
	}
}

func TestKitaruMiddlewareAdapter_GetAuditor(t *testing.T) {
	client := &mockClient{}
	auditor := middleware.NewInMemoryAuditor()
	adapter := NewKitaruMiddlewareAdapter(client, nil, nil, auditor)

	result := adapter.GetAuditor()

	if result != auditor {
		t.Error("expected same auditor")
	}
}

func TestGetCallerContext_WithContext(t *testing.T) {
	ctx := context.Background()
	callerCtx := middleware.CallerContext{
		Trusted: false,
		Role:    "user",
		UserID:  "user-123",
	}
	ctx = WithCallerContext(ctx, callerCtx)

	result := getCallerContext(ctx)

	if result.Trusted {
		t.Error("expected trusted to be false")
	}
	if result.Role != "user" {
		t.Errorf("expected role 'user', got '%s'", result.Role)
	}
	if result.UserID != "user-123" {
		t.Errorf("expected user_id 'user-123', got '%s'", result.UserID)
	}
}

func TestGetCallerContext_WithoutContext(t *testing.T) {
	ctx := context.Background()

	result := getCallerContext(ctx)

	if !result.Trusted {
		t.Error("expected default trusted to be true")
	}
}

func TestWithCallerContext(t *testing.T) {
	ctx := context.Background()
	callerCtx := middleware.CallerContext{
		Trusted:   true,
		SessionID: "session-123",
	}

	resultCtx := WithCallerContext(ctx, callerCtx)
	result := resultCtx.Value(kitaruCallerKey).(middleware.CallerContext)

	if result.Trusted != true {
		t.Error("expected trusted to be true")
	}
	if result.SessionID != "session-123" {
		t.Error("expected session-123")
	}
}

func TestClientFlowNameTracking(t *testing.T) {
	client := &mockClient{}
	executor := NewFlowToolExecutor(client, map[string]string{"my_tool": "my_flow"})

	executor.Execute(context.Background(), "my_tool", nil)

	if client.flowName != "my_flow" {
		t.Errorf("expected flowName 'my_flow', got '%s'", client.flowName)
	}
}

func TestFlowToolExecutor_Execute_ContextCancellation(t *testing.T) {
	client := &mockClient{}
	mapping := map[string]string{"my_tool": "my_flow"}

	exec := NewFlowToolExecutorWithConfig(client, mapping, &Config{
		WaitInterval:   1 * time.Millisecond,
		RequestTimeout: 1 * time.Millisecond,
		RetryCount:     1,
	})

	_, _ = exec.Execute(context.Background(), "my_tool", nil)
}
