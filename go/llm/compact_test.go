package llm

import (
	"testing"
)

func TestTieredCompact_Name(t *testing.T) {
	compact := NewTieredCompact()
	if compact.Name() != "TieredCompact" {
		t.Errorf("expected TieredCompact, got %s", compact.Name())
	}
}

func TestTieredCompact_Phase1_DropsNudges(t *testing.T) {
	compact := NewTieredCompact()
	compact.KeepRecent = 0
	stepIdx := 0

	messages := []Message{
		{Role: RoleSystem, Content: "system", Meta: MessageMeta{Type: MessageTypeSystemPrompt}},
		{Role: RoleUser, Content: "user", Meta: MessageMeta{Type: MessageTypeUserInput}},
		{Role: RoleAssistant, Content: "", Meta: MessageMeta{Type: MessageTypeStepNudge, StepIndex: &stepIdx}},
		{Role: RoleAssistant, Content: "text", Meta: MessageMeta{Type: MessageTypeTextResponse, StepIndex: &stepIdx}},
	}

	counter := func(msgs []Message) int {
		return EstimateMessagesTokens(msgs)
	}

	result, err := compact.Compact(messages, 5, counter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundNudge := false
	for _, m := range result {
		if m.Meta.Type == MessageTypeStepNudge {
			foundNudge = true
			break
		}
	}
	if foundNudge {
		t.Error("expected nudge to be dropped in phase 1")
	}
}

func TestTieredCompact_Phase1_TruncatesToolResults(t *testing.T) {
	compact := NewTieredCompact()
	compact.KeepRecent = 0

	step0 := 0
	step1 := 1

	messages := []Message{
		{Role: RoleSystem, Content: "system", Meta: MessageMeta{Type: MessageTypeSystemPrompt, StepIndex: &step0}},
		{Role: RoleUser, Content: "user", Meta: MessageMeta{Type: MessageTypeUserInput, StepIndex: &step0}},
		{Role: RoleTool, Content: string(make([]byte, 500)), Meta: MessageMeta{Type: MessageTypeToolResult, StepIndex: &step0}},
		{Role: RoleTool, Content: string(make([]byte, 500)), Meta: MessageMeta{Type: MessageTypeToolResult, StepIndex: &step1}},
	}

	counter := func(msgs []Message) int {
		return EstimateMessagesTokens(msgs)
	}

	result, err := compact.Compact(messages, 150, counter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(result))
	}

	for _, m := range result {
		if m.Meta.Type == MessageTypeToolResult && len(m.Content) > 200 {
			t.Errorf("expected tool_result truncated to 200 chars, got %d", len(m.Content))
		}
	}
}

func TestTieredCompact_Phase2_DropsToolResults(t *testing.T) {
	compact := NewTieredCompact()
	compact.KeepRecent = 0
	stepIdx := 0

	messages := []Message{
		{Role: RoleSystem, Content: "system", Meta: MessageMeta{Type: MessageTypeSystemPrompt}},
		{Role: RoleUser, Content: "user", Meta: MessageMeta{Type: MessageTypeUserInput}},
		{Role: RoleTool, Content: "tool result", Meta: MessageMeta{Type: MessageTypeToolResult, StepIndex: &stepIdx}},
		{Role: RoleAssistant, Content: "text", Meta: MessageMeta{Type: MessageTypeTextResponse, StepIndex: &stepIdx}},
	}

	counter := func(msgs []Message) int {
		return EstimateMessagesTokens(msgs)
	}

	result, err := compact.Compact(messages, 5, counter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundToolResult := false
	for _, m := range result {
		if m.Meta.Type == MessageTypeToolResult {
			foundToolResult = true
			break
		}
	}
	if foundToolResult {
		t.Error("expected tool_result to be dropped in phase 2")
	}
}

func TestTieredCompact_Phase3_DropsReasoningAndText(t *testing.T) {
	compact := NewTieredCompact()
	compact.KeepRecent = 0
	stepIdx := 0

	messages := []Message{
		{Role: RoleSystem, Content: "system", Meta: MessageMeta{Type: MessageTypeSystemPrompt}},
		{Role: RoleUser, Content: "user", Meta: MessageMeta{Type: MessageTypeUserInput}},
		{Role: RoleTool, Content: "tool result", Meta: MessageMeta{Type: MessageTypeToolResult, StepIndex: &stepIdx}},
		{Role: RoleAssistant, Content: "reasoning", Meta: MessageMeta{Type: MessageTypeReasoning, StepIndex: &stepIdx}},
		{Role: RoleAssistant, Content: "text response", Meta: MessageMeta{Type: MessageTypeTextResponse, StepIndex: &stepIdx}},
		{Role: RoleAssistant, Content: "", ToolCalls: []ToolCall{{Name: "test"}}, Meta: MessageMeta{Type: MessageTypeToolCall, StepIndex: &stepIdx}},
	}

	counter := func(msgs []Message) int {
		return EstimateMessagesTokens(msgs)
	}

	result, err := compact.Compact(messages, 2, counter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, m := range result {
		if m.Meta.Type == MessageTypeReasoning || m.Meta.Type == MessageTypeTextResponse {
			t.Errorf("expected %s to be dropped in phase 3", m.Meta.Type)
		}
	}
}

func TestTieredCompact_ProtectsSystemAndUser(t *testing.T) {
	compact := NewTieredCompact()
	compact.KeepRecent = 0
	stepIdx := 0

	messages := []Message{
		{Role: RoleSystem, Content: "system prompt", Meta: MessageMeta{Type: MessageTypeSystemPrompt}},
		{Role: RoleUser, Content: "user input", Meta: MessageMeta{Type: MessageTypeUserInput}},
		{Role: RoleAssistant, Content: "", Meta: MessageMeta{Type: MessageTypeStepNudge, StepIndex: &stepIdx}},
	}

	counter := func(msgs []Message) int {
		return EstimateMessagesTokens(msgs)
	}

	result, err := compact.Compact(messages, 1, counter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 messages (system+user protected), got %d", len(result))
	}

	if result[0].Meta.Type != MessageTypeSystemPrompt {
		t.Error("system prompt should be preserved")
	}
	if result[1].Meta.Type != MessageTypeUserInput {
		t.Error("user input should be preserved")
	}
}

func TestTieredCompact_KeepRecent(t *testing.T) {
	compact := NewTieredCompact()
	compact.KeepRecent = 2

	step0 := 0
	step1 := 1
	step2 := 2
	step3 := 3
	step4 := 4

	messages := []Message{
		{Role: RoleSystem, Content: "system", Meta: MessageMeta{Type: MessageTypeSystemPrompt}},
		{Role: RoleUser, Content: "user", Meta: MessageMeta{Type: MessageTypeUserInput}},
		{Role: RoleTool, Content: "tool0", Meta: MessageMeta{Type: MessageTypeToolResult, StepIndex: &step0}},
		{Role: RoleTool, Content: "tool1", Meta: MessageMeta{Type: MessageTypeToolResult, StepIndex: &step1}},
		{Role: RoleTool, Content: "tool2", Meta: MessageMeta{Type: MessageTypeToolResult, StepIndex: &step2}},
		{Role: RoleTool, Content: "tool3", Meta: MessageMeta{Type: MessageTypeToolResult, StepIndex: &step3}},
		{Role: RoleTool, Content: "tool4", Meta: MessageMeta{Type: MessageTypeToolResult, StepIndex: &step4}},
	}

	counter := func(msgs []Message) int {
		return EstimateMessagesTokens(msgs)
	}

	result, err := compact.Compact(messages, 1, counter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hasStep3 := false
	hasStep4 := false
	for _, m := range result {
		if m.Meta.StepIndex != nil {
			if *m.Meta.StepIndex == 3 {
				hasStep3 = true
			}
			if *m.Meta.StepIndex == 4 {
				hasStep4 = true
			}
		}
	}

	if !hasStep3 || !hasStep4 {
		t.Error("expected steps 3 and 4 to be preserved (keepRecent=2)")
	}
}

func TestTieredCompact_EmptyMessages(t *testing.T) {
	compact := NewTieredCompact()

	counter := func(msgs []Message) int {
		return EstimateMessagesTokens(msgs)
	}

	_, err := compact.Compact([]Message{}, 100, counter)
	if err != ErrEmptyMessages {
		t.Errorf("expected ErrEmptyMessages, got %v", err)
	}
}

func TestFindEligibleEnd_NoStepIndex(t *testing.T) {
	messages := []Message{
		{Role: RoleSystem, Content: "system"},
		{Role: RoleUser, Content: "user"},
		{Role: RoleAssistant, Content: "msg1"},
		{Role: RoleAssistant, Content: "msg2"},
		{Role: RoleAssistant, Content: "msg3"},
	}

	result := findEligibleEnd(messages, 2)
	if result != 2 {
		t.Errorf("expected 2, got %d", result)
	}
}

func TestFindEligibleEnd_WithStepIndex(t *testing.T) {
	step0 := 0
	step1 := 1
	step2 := 2
	step3 := 3
	step4 := 4

	messages := []Message{
		{Role: RoleSystem, Content: "system", Meta: MessageMeta{StepIndex: &step0}},
		{Role: RoleUser, Content: "user", Meta: MessageMeta{StepIndex: &step0}},
		{Role: RoleAssistant, Content: "msg1", Meta: MessageMeta{StepIndex: &step1}},
		{Role: RoleAssistant, Content: "msg2", Meta: MessageMeta{StepIndex: &step2}},
		{Role: RoleAssistant, Content: "msg3", Meta: MessageMeta{StepIndex: &step3}},
		{Role: RoleAssistant, Content: "msg4", Meta: MessageMeta{StepIndex: &step4}},
	}

	result := findEligibleEnd(messages, 2)
	if result != 3 {
		t.Errorf("expected 3 (index of step2), got %d", result)
	}
}

func TestIsNudgeType(t *testing.T) {
	tests := []struct {
		msgType MessageType
		want    bool
	}{
		{MessageTypeStepNudge, true},
		{MessageTypeRetryNudge, true},
		{MessageTypePrerequisiteNudge, true},
		{MessageTypeToolResult, false},
		{MessageTypeReasoning, false},
		{MessageTypeTextResponse, false},
		{MessageTypeSystemPrompt, false},
		{MessageTypeUserInput, false},
	}

	for _, tt := range tests {
		if got := isNudgeType(tt.msgType); got != tt.want {
			t.Errorf("isNudgeType(%s) = %v, want %v", tt.msgType, got, tt.want)
		}
	}
}
