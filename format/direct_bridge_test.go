package format

import (
	"context"
	"errors"
	"testing"

	"github.com/soypete/pedro-agentware/types"
)

func TestDirectBridge_FormatTools(t *testing.T) {
	bridge := NewDirectBridge(nil, ModelFamilyLlama3)
	tools := []types.ToolDefinition{
		{Name: "echo", Description: "Echoes input"},
		{Name: "read_file", Description: "Reads a file"},
	}
	result := bridge.FormatTools(tools)
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestLlama3Formatter_FormatTools(t *testing.T) {
	f := NewLlama3Formatter()
	tools := []types.ToolDefinition{
		{Name: "tool1", Description: "desc1"},
		{Name: "tool2", Description: "desc2"},
	}
	result := f.FormatTools(tools)
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestLlama3Formatter_FormatToolResult(t *testing.T) {
	f := NewLlama3Formatter()

	result := f.FormatToolResult(&types.ToolResult{Content: "test content"})
	if result != "test content" {
		t.Errorf("expected 'test content', got %s", result)
	}

	result = f.FormatToolResult(&types.ToolResult{Error: errors.New("test error")})
	if result != "Error: test error" {
		t.Errorf("expected 'Error: test error', got %s", result)
	}
}

func TestQwenFormatter_FormatTools(t *testing.T) {
	f := NewQwenFormatter()
	tools := []types.ToolDefinition{
		{Name: "tool1", Description: "desc1"},
	}
	result := f.FormatTools(tools)
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestQwenFormatter_FormatToolResult(t *testing.T) {
	f := NewQwenFormatter()

	result := f.FormatToolResult(&types.ToolResult{Content: "test"})
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}

	result = f.FormatToolResult(&types.ToolResult{Error: errors.New("err")})
	if result != "Error: err" {
		t.Errorf("expected 'Error: err', got %s", result)
	}
}

func TestMistralFormatter_FormatTools(t *testing.T) {
	f := NewMistralFormatter()
	tools := []types.ToolDefinition{
		{Name: "tool1", Description: "desc1"},
	}
	result := f.FormatTools(tools)
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestMistralFormatter_FormatToolResult(t *testing.T) {
	f := NewMistralFormatter()

	result := f.FormatToolResult(&types.ToolResult{Content: "test"})
	if result != "<result>test</result>" {
		t.Errorf("expected '<result>test</result>', got %s", result)
	}

	result = f.FormatToolResult(&types.ToolResult{Error: errors.New("err")})
	if result != "<error>err</error>" {
		t.Errorf("expected '<error>err</error>', got %s", result)
	}
}

func TestClaudeFormatter_FormatTools(t *testing.T) {
	f := NewClaudeFormatter()
	tools := []types.ToolDefinition{
		{Name: "tool1", Description: "desc1"},
	}
	result := f.FormatTools(tools)
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestClaudeFormatter_FormatToolResult(t *testing.T) {
	f := NewClaudeFormatter()

	result := f.FormatToolResult(&types.ToolResult{Content: "test"})
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}

	result = f.FormatToolResult(&types.ToolResult{Error: errors.New("err")})
	if result != "Error: err" {
		t.Errorf("expected 'Error: err', got %s", result)
	}
}

func TestOpenAIFormatter_FormatTools(t *testing.T) {
	f := NewOpenAIFormatter()
	tools := []types.ToolDefinition{
		{Name: "tool1", Description: "desc1"},
	}
	result := f.FormatTools(tools)
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestOpenAIFormatter_FormatToolResult(t *testing.T) {
	f := NewOpenAIFormatter()

	result := f.FormatToolResult(&types.ToolResult{Content: "test"})
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}

	result = f.FormatToolResult(&types.ToolResult{Error: errors.New("err")})
	if result != "err" {
		t.Errorf("expected 'err', got %s", result)
	}
}

func TestGLM4Formatter_FormatTools(t *testing.T) {
	f := NewGLM4Formatter()
	tools := []types.ToolDefinition{
		{Name: "tool1", Description: "desc1"},
	}
	result := f.FormatTools(tools)
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestGLM4Formatter_FormatToolResult(t *testing.T) {
	f := NewGLM4Formatter()

	result := f.FormatToolResult(&types.ToolResult{Content: "test"})
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}

	result = f.FormatToolResult(&types.ToolResult{Error: errors.New("err")})
	if result != "err" {
		t.Errorf("expected 'err', got %s", result)
	}
}

func TestNewFormatter_Default(t *testing.T) {
	f := NewFormatter("unknown")
	if f.ModelFamily() != ModelFamilyClaude {
		t.Error("expected default to be claude")
	}
}

func TestParseToolCallsByFamily_Default(t *testing.T) {
	p := ParseToolCallsByFamily("unknown")
	calls, err := p("test")
	if err != nil {
		t.Error(err)
	}
	if len(calls) != 0 {
		t.Error("expected no calls for unknown family")
	}
}

func TestParseMistralToolCalls(t *testing.T) {
	calls, err := ParseMistralToolCalls(`<tool>echo</tool> <tool>read</tool>`)
	if err != nil {
		t.Error(err)
	}
	if len(calls) != 2 {
		t.Errorf("expected 2 calls, got %d", len(calls))
	}
}

func TestParseGLM4ToolCalls(t *testing.T) {
	calls, err := ParseGLM4ToolCalls(`[{"function":{"name":"echo","arguments":"{}"}}]`)
	if err != nil {
		t.Error(err)
	}
	if len(calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "echo" {
		t.Errorf("expected 'echo', got %s", calls[0].Name)
	}
}

func TestHybridBridge_FormatTools(t *testing.T) {
	mockExecutor := &mockExecutor{
		tools: []types.ToolDefinition{{Name: "echo", Description: "test"}},
	}
	bridge := NewHybridBridge(mockExecutor, nil, ModelFamilyLlama3)
	result := bridge.FormatTools([]types.ToolDefinition{{Name: "test", Description: "desc"}})
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestHybridBridge_FormatTools_MCP(t *testing.T) {
	mockExecutor := &mockExecutor{
		tools: []types.ToolDefinition{{Name: "echo", Description: "test"}},
	}
	bridge := NewHybridBridge(mockExecutor, &mockMCPBridge{}, ModelFamilyClaude)
	bridge.SwitchToMCP()
	result := bridge.FormatTools([]types.ToolDefinition{{Name: "test", Description: "desc"}})
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestHybridBridge_FormatToolResult(t *testing.T) {
	mockExecutor := &mockExecutor{
		tools: []types.ToolDefinition{{Name: "echo", Description: "test"}},
	}
	bridge := NewHybridBridge(mockExecutor, nil, ModelFamilyLlama3)
	result := bridge.FormatToolResult(&types.ToolResult{Content: "test"})
	if result != "test" {
		t.Errorf("expected 'test', got %s", result)
	}
}

func TestHybridBridge_ExecuteTool_Direct(t *testing.T) {
	mockExecutor := &mockExecutor{
		tools:  []types.ToolDefinition{{Name: "echo", Description: "test"}},
		result: &types.ToolResult{Content: "result"},
	}
	bridge := NewHybridBridge(mockExecutor, nil, ModelFamilyLlama3)

	result, err := bridge.ExecuteTool(context.Background(), ToolCall{Name: "echo", Arguments: map[string]interface{}{"x": "y"}})
	if err != nil {
		t.Error(err)
	}
	if result.Content != "result" {
		t.Errorf("expected 'result', got %v", result.Content)
	}
}

func TestHybridBridge_ExecuteTool_MCP(t *testing.T) {
	mockMCP := &mockMCPBridge{
		result: &types.ToolResult{Content: "mcp result"},
	}
	bridge := NewHybridBridge(nil, mockMCP, ModelFamilyClaude)
	bridge.SwitchToMCP()

	result, err := bridge.ExecuteTool(context.Background(), ToolCall{Name: "server/echo", Arguments: map[string]interface{}{"x": "y"}})
	if err != nil {
		t.Error(err)
	}
	if result.Content != "mcp result" {
		t.Errorf("expected 'mcp result', got %v", result.Content)
	}
}

func TestHybridBridge_ExecuteTool_MCP_InvalidFormat(t *testing.T) {
	mockMCP := &mockMCPBridge{result: &types.ToolResult{}}
	mockExecutor := &mockExecutor{}
	bridge := NewHybridBridge(mockExecutor, mockMCP, ModelFamilyClaude)
	bridge.SwitchToMCP()

	_, err := bridge.ExecuteTool(context.Background(), ToolCall{Name: "invalid"})
	if err == nil {
		t.Error("expected error for invalid tool format")
	}
}

func TestHybridBridge_ExecuteTool_MCP_NoBridge(t *testing.T) {
	mockExecutor := &mockExecutor{
		tools:  []types.ToolDefinition{{Name: "echo", Description: "test"}},
		result: &types.ToolResult{Content: "direct"},
	}
	bridge := NewHybridBridge(mockExecutor, nil, ModelFamilyClaude)
	bridge.SwitchToMCP()

	result, err := bridge.ExecuteTool(context.Background(), ToolCall{Name: "echo"})
	if err != nil {
		t.Error(err)
	}
	if result.Content != "direct" {
		t.Errorf("expected 'direct', got %v", result.Content)
	}
}

func TestHybridBridge_ListTools(t *testing.T) {
	mockExecutor := &mockExecutor{
		tools: []types.ToolDefinition{{Name: "echo", Description: "test"}},
	}
	bridge := NewHybridBridge(mockExecutor, nil, ModelFamilyLlama3)
	tools := bridge.ListTools()
	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}
}

func TestHybridBridge_UseMCP(t *testing.T) {
	bridge := NewHybridBridge(nil, nil, ModelFamilyClaude)
	if bridge.UseMCP() != true {
		t.Error("expected Claude to be MCP capable")
	}

	bridge2 := NewHybridBridge(nil, nil, ModelFamilyLlama3)
	if bridge2.UseMCP() != false {
		t.Error("expected Llama3 to not be MCP capable")
	}
}

func TestHybridBridge_SwitchToDirect(t *testing.T) {
	mockMCP := &mockMCPBridge{}
	bridge := NewHybridBridge(nil, mockMCP, ModelFamilyClaude)
	bridge.SwitchToDirect()
	if bridge.UseMCP() != false {
		t.Error("expected UseMCP to be false after SwitchToDirect")
	}
}

func TestHybridBridge_SwitchToMCP(t *testing.T) {
	mockExecutor := &mockExecutor{}
	bridge := NewHybridBridge(mockExecutor, nil, ModelFamilyLlama3)
	bridge.SwitchToMCP()
	if bridge.UseMCP() != true {
		t.Error("expected UseMCP to be true after SwitchToMCP")
	}
}

func TestHybridBridge_ModelFamily(t *testing.T) {
	mockExecutor := &mockExecutor{}
	bridge := NewHybridBridge(mockExecutor, nil, ModelFamilyQwen)
	if bridge.ModelFamily() != ModelFamilyQwen {
		t.Error("expected ModelFamily to be qwen")
	}
}

func TestSplitToolName(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"server/tool", []string{"server", "tool"}},
		{"tool", []string{"default", "tool"}},
	}

	for _, tc := range tests {
		result := splitToolName(tc.input)
		if len(result) != len(tc.expected) || result[0] != tc.expected[0] || result[1] != tc.expected[1] {
			t.Errorf("expected %v, got %v", tc.expected, result)
		}
	}
}

func TestInvalidToolFormatError(t *testing.T) {
	err := ErrInvalidToolFormat
	if err.Error() != "tool name must be in format 'server/tool' for MCP calls" {
		t.Error("unexpected error message")
	}
}

func TestFormatResultAsString(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"test", "test"},
		{[]byte("bytes"), "bytes"},
		{map[string]interface{}{"key": "value"}, `{"key":"value"}`},
	}

	for _, tc := range tests {
		result := formatResultAsString(tc.input)
		if result != tc.expected {
			t.Errorf("expected %s, got %s", tc.expected, result)
		}
	}
}

type mockExecutor struct {
	tools  []types.ToolDefinition
	result *types.ToolResult
}

func (m *mockExecutor) CallTool(ctx context.Context, name string, args map[string]interface{}) (*types.ToolResult, error) {
	return m.result, nil
}

func (m *mockExecutor) ListTools() []types.ToolDefinition {
	return m.tools
}

type mockMCPBridge struct {
	result *types.ToolResult
}

func (m *mockMCPBridge) Connect(ctx context.Context, serverName string) error {
	return nil
}

func (m *mockMCPBridge) CallTool(ctx context.Context, server string, tool string, args map[string]interface{}) (*types.ToolResult, error) {
	return m.result, nil
}

func (m *mockMCPBridge) ListTools(server string) []types.ToolDefinition {
	return nil
}

func TestDirectBridge_ExecuteToolByName(t *testing.T) {
	mockExec := &mockExecutor{result: &types.ToolResult{Content: "test result"}}
	bridge := NewDirectBridge(mockExec, ModelFamilyClaude)

	result, err := bridge.ExecuteToolByName(context.Background(), "test_tool", map[string]interface{}{"arg": "value"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Content != "test result" {
		t.Errorf("expected 'test result', got %v", result.Content)
	}
}

func TestDirectBridge_SetToolFilter(t *testing.T) {
	mockExec := &mockExecutor{tools: []types.ToolDefinition{{Name: "tool1"}, {Name: "tool2"}}}
	bridge := NewDirectBridge(mockExec, ModelFamilyClaude)

	bridge.SetToolFilter(func(tools []types.ToolDefinition) []types.ToolDefinition {
		return tools[:1]
	})

	tools := bridge.ListTools()
	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}
}

func TestNewDirectBridgeForNonMCP(t *testing.T) {
	mockExec := &mockExecutor{tools: []types.ToolDefinition{{Name: "tool"}}}
	bridge, err := NewDirectBridgeForNonMCP(mockExec, "llama 3 model")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if bridge == nil {
		t.Error("expected bridge, got nil")
	}
}

func TestNewDirectBridgeForNonMCP_MCPCapable(t *testing.T) {
	mockExec := &mockExecutor{tools: []types.ToolDefinition{{Name: "tool"}}}
	_, err := NewDirectBridgeForNonMCP(mockExec, "claude model")
	if err != ErrMCPRequired {
		t.Errorf("expected ErrMCPRequired, got %v", err)
	}
}

func TestParseToolCallsByFamily_Llama3(t *testing.T) {
	parser := ParseToolCallsByFamily(ModelFamilyLlama3)
	calls, err := parser("test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	_ = calls
}

func TestParseToolCallsByFamily_Qwen(t *testing.T) {
	parser := ParseToolCallsByFamily(ModelFamilyQwen)
	calls, err := parser("test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	_ = calls
}

func TestParseToolCallsByFamily_Unknown(t *testing.T) {
	parser := ParseToolCallsByFamily("unknown")
	calls, err := parser("test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	_ = calls
}

func TestHybridBridge_ParseToolCalls(t *testing.T) {
	mockExec := &mockExecutor{}
	bridge := NewHybridBridge(mockExec, nil, ModelFamilyLlama3)

	calls, err := bridge.ParseToolCalls("test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	_ = calls
}

func TestHybridBridge_ListTools_MCP(t *testing.T) {
	mockMCP := &mockMCPBridge{}
	mockExec := &mockExecutor{}
	bridge := NewHybridBridge(mockExec, mockMCP, ModelFamilyClaude)
	bridge.SwitchToMCP()

	tools := bridge.ListTools()
	_ = tools
}

func TestHybridBridge_SetUseMCP(t *testing.T) {
	mockExec := &mockExecutor{}
	bridge := NewHybridBridge(mockExec, nil, ModelFamilyLlama3)
	bridge.SetUseMCP(true)
	if !bridge.UseMCP() {
		t.Error("expected UseMCP to be true")
	}
}

func TestDirectBridge_ParseToolCalls(t *testing.T) {
	executor := &mockExecutor{}
	bridge := NewDirectBridge(executor, ModelFamilyClaude)

	calls, err := bridge.ParseToolCalls(`<tool_call><tool name="test_tool"></tool_call>`)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(calls) == 0 {
		t.Error("expected tool calls to be parsed")
	}
}

func TestIsMCPCapable(t *testing.T) {
	tests := []struct {
		family  ModelFamily
		capable bool
	}{
		{ModelFamilyClaude, true},
		{ModelFamilyOpenAI, true},
		{ModelFamilyGLM4, true},
		{ModelFamilyLlama3, false},
		{ModelFamilyQwen, false},
		{ModelFamilyMistral, false},
	}

	for _, tt := range tests {
		result := IsMCPCapable(tt.family)
		if result != tt.capable {
			t.Errorf("IsMCPCapable(%v) = %v, want %v", tt.family, result, tt.capable)
		}
	}
}

func TestDetectModelFamily(t *testing.T) {
	tests := []struct {
		response string
		expected ModelFamily
	}{
		{"This is Claude", ModelFamilyClaude},
		{"Using GPT-4", ModelFamilyOpenAI},
		{"GLM-4 model", ModelFamilyGLM4},
		{"Llama 3", ModelFamilyLlama3},
		{"Qwen model", ModelFamilyQwen},
		{"Mistral AI", ModelFamilyMistral},
		{"unknown model", ModelFamilyClaude},
	}

	for _, tt := range tests {
		result := DetectModelFamily(tt.response)
		if result != tt.expected {
			t.Errorf("DetectModelFamily(%q) = %v, want %v", tt.response, result, tt.expected)
		}
	}
}

func TestDirectBridge_ExecuteTool(t *testing.T) {
	executor := &mockExecutor{result: &types.ToolResult{Content: "result"}}
	bridge := NewDirectBridge(executor, ModelFamilyClaude)

	result, err := bridge.ExecuteTool(context.Background(), ToolCall{
		Name:      "test_tool",
		Arguments: map[string]interface{}{"key": "value"},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil || result.Content != "result" {
		t.Error("expected result with content 'result'")
	}
}

func TestNewFormatter_Mistral(t *testing.T) {
	f := NewFormatter(ModelFamilyMistral)
	if f == nil {
		t.Error("expected non-nil formatter")
	}
	if f.ModelFamily() != ModelFamilyMistral {
		t.Errorf("expected Mistral, got %s", f.ModelFamily())
	}
}

func TestNewFormatter_OpenAI(t *testing.T) {
	f := NewFormatter(ModelFamilyOpenAI)
	if f == nil {
		t.Error("expected non-nil formatter")
	}
	if f.ModelFamily() != ModelFamilyOpenAI {
		t.Errorf("expected OpenAI, got %s", f.ModelFamily())
	}
}

func TestNewFormatter_GLM4(t *testing.T) {
	f := NewFormatter(ModelFamilyGLM4)
	if f == nil {
		t.Error("expected non-nil formatter")
	}
	if f.ModelFamily() != ModelFamilyGLM4 {
		t.Errorf("expected GLM4, got %s", f.ModelFamily())
	}
}

func TestDirectBridge_FormatTools_WithFilter(t *testing.T) {
	mockExec := &mockExecutor{}
	bridge := NewDirectBridge(mockExec, ModelFamilyLlama3)
	bridge.SetToolFilter(func(tools []types.ToolDefinition) []types.ToolDefinition {
		return []types.ToolDefinition{tools[0]}
	})
	tools := []types.ToolDefinition{
		{Name: "echo", Description: "Echoes input"},
		{Name: "read_file", Description: "Reads a file"},
	}
	result := bridge.FormatTools(tools)
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestParseToolCallsByFamily_Mistral(t *testing.T) {
	parser := ParseToolCallsByFamily(ModelFamilyMistral)
	calls, err := parser("<tool>test</tool>")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(calls) != 1 || calls[0].Name != "test" {
		t.Error("expected one tool call with name 'test'")
	}
}

func TestParseToolCallsByFamily_Claude(t *testing.T) {
	parser := ParseToolCallsByFamily(ModelFamilyClaude)
	calls, err := parser("test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	_ = calls
}

func TestParseToolCallsByFamily_OpenAI(t *testing.T) {
	parser := ParseToolCallsByFamily(ModelFamilyOpenAI)
	calls, err := parser("[]")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(calls) != 0 {
		t.Error("expected no tool calls for empty array")
	}
}

func TestParseToolCallsByFamily_GLM4(t *testing.T) {
	parser := ParseToolCallsByFamily(ModelFamilyGLM4)
	calls, err := parser("[]")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(calls) != 0 {
		t.Error("expected no tool calls for empty array")
	}
}
