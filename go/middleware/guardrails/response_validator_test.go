package guardrails

import (
	"testing"
)

func TestNewResponseValidator(t *testing.T) {
	tools := []string{"tool1", "tool2"}
	rv := NewResponseValidator(tools, true)

	if len(rv.ToolNames) != 2 {
		t.Errorf("expected 2 tool names, got %d", len(rv.ToolNames))
	}
	if !rv.RescueEnabled {
		t.Error("expected rescue to be enabled")
	}
}

func TestValidateToolCalls_Valid(t *testing.T) {
	tools := []string{"tool1", "tool2"}
	rv := NewResponseValidator(tools, false)

	calls := []ToolCall{
		{Tool: "tool1", Args: map[string]interface{}{"key": "value"}},
	}

	result := rv.ValidateToolCalls(calls)

	if result.NeedsRetry {
		t.Error("expected no retry for valid tool calls")
	}
	if result.Nudge != nil {
		t.Error("expected no nudge for valid tool calls")
	}
	if len(result.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(result.ToolCalls))
	}
}

func TestValidateToolCalls_Invalid(t *testing.T) {
	tools := []string{"tool1", "tool2"}
	rv := NewResponseValidator(tools, false)

	calls := []ToolCall{
		{Tool: "unknown_tool", Args: map[string]interface{}{}},
	}

	result := rv.ValidateToolCalls(calls)

	if !result.NeedsRetry {
		t.Error("expected retry for unknown tool")
	}
	if result.Nudge == nil {
		t.Error("expected nudge for unknown tool")
	}
	if result.Nudge.Kind != NudgeKindUnknownTool {
		t.Errorf("expected unknown_tool nudge, got %s", result.Nudge.Kind)
	}
}

func TestValidateTextResponse_WithRescue(t *testing.T) {
	tools := []string{"tool1", "tool2"}
	rv := NewResponseValidator(tools, true)

	response := `{"tool": "tool1", "args": {"key": "value"}}`

	result := rv.ValidateTextResponse(response)

	if result.NeedsRetry {
		t.Error("expected no retry when tool call is rescued")
	}
	if len(result.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(result.ToolCalls))
	}
}

func TestValidateTextResponse_WithoutRescue(t *testing.T) {
	tools := []string{"tool1", "tool2"}
	rv := NewResponseValidator(tools, false)

	response := "This is just text, not a tool call."

	result := rv.ValidateTextResponse(response)

	if !result.NeedsRetry {
		t.Error("expected retry for text response")
	}
	if result.Nudge == nil {
		t.Error("expected nudge for text response")
	}
}

func TestValidateTextResponse_Empty(t *testing.T) {
	tools := []string{"tool1", "tool2"}
	rv := NewResponseValidator(tools, true)

	result := rv.ValidateTextResponse("")

	if !result.NeedsRetry {
		t.Error("expected retry for empty response")
	}
}

func TestRescueToolCall_StripsThinkTags(t *testing.T) {
	tools := []string{"tool1", "tool2"}
	rv := NewResponseValidator(tools, true)

	response := `[THINK] thinking [/THINK]
{"tool": "tool1", "args": {}}`

	calls := rv.RescueToolCall(response)

	if len(calls) != 1 {
		t.Errorf("expected 1 rescued call, got %d", len(calls))
	}
}

func TestRescueToolCall_StripsPythonTag(t *testing.T) {
	tools := []string{"tool1", "tool2"}
	rv := NewResponseValidator(tools, true)

	response := `<|python_tag|>{"tool": "tool1", "args": {}}`

	calls := rv.RescueToolCall(response)

	if len(calls) != 1 {
		t.Errorf("expected 1 rescued call, got %d", len(calls))
	}
}

func TestExtractJSONToolCalls_WithCodeFence(t *testing.T) {
	tools := []string{"tool1", "tool2"}
	rv := NewResponseValidator(tools, true)

	response := "```json\n{\"tool\": \"tool1\", \"args\": {\"key\": \"value\"}}\n```"

	calls := rv.extractJSONToolCalls(response)

	if len(calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(calls))
	}
}

func TestExtractRehearsalToolCalls(t *testing.T) {
	tools := []string{"tool1", "tool2"}
	rv := NewResponseValidator(tools, true)

	response := "tool1[ARGS]{" + `"` + "key" + `"` + ": " + `"` + "value" + `"` + "}"

	calls := rv.extractRehearsalToolCalls(response)

	if len(calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(calls))
	}
}

func TestIsValidTool(t *testing.T) {
	tools := []string{"tool1", "tool2"}
	rv := NewResponseValidator(tools, false)

	if !rv.isValidTool("tool1") {
		t.Error("expected tool1 to be valid")
	}
	if rv.isValidTool("tool3") {
		t.Error("expected tool3 to be invalid")
	}
}

func TestTryParseToolCall_Valid(t *testing.T) {
	tools := []string{"tool1", "tool2"}
	rv := NewResponseValidator(tools, true)

	jsonStr := `{"tool": "tool1", "args": {"key": "value"}}`

	call := rv.tryParseToolCall(jsonStr)

	if call == nil {
		t.Fatal("expected non-nil call")
	}
	if call.Tool != "tool1" {
		t.Errorf("expected tool1, got %s", call.Tool)
	}
}

func TestTryParseToolCall_InvalidJSON(t *testing.T) {
	tools := []string{"tool1", "tool2"}
	rv := NewResponseValidator(tools, true)

	call := rv.tryParseToolCall("not valid json")

	if call != nil {
		t.Error("expected nil for invalid json")
	}
}

func TestTryParseToolCall_UnknownTool(t *testing.T) {
	tools := []string{"tool1"}
	rv := NewResponseValidator(tools, true)

	jsonStr := `{"tool": "unknown", "args": {}}`

	call := rv.tryParseToolCall(jsonStr)

	if call != nil {
		t.Error("expected nil for unknown tool")
	}
}

func TestExtractQwenXMLToolCalls(t *testing.T) {
	tools := []string{"tool1", "tool2"}
	rv := NewResponseValidator(tools, true)

	calls := rv.extractQwenXMLToolCalls("<function=tool1>content</function>")

	if len(calls) != 0 {
		t.Errorf("expected 0 calls (not implemented), got %d", len(calls))
	}
}
