package guardrails

import (
	"testing"
)

func TestRetryNudge(t *testing.T) {
	tools := []string{"get_weather", "echo", "search"}

	nudge := RetryNudge("some text response", tools)

	if nudge.Role != "user" {
		t.Errorf("expected role 'user', got '%s'", nudge.Role)
	}
	if nudge.Kind != NudgeKindRetry {
		t.Errorf("expected kind 'retry', got '%s'", nudge.Kind)
	}
	if nudge.Tier != 0 {
		t.Errorf("expected tier 0, got %d", nudge.Tier)
	}
	if nudge.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestUnknownToolNudge(t *testing.T) {
	tools := []string{"echo", "search"}

	nudge := UnknownToolNudge("nonexistent_tool", tools)

	if nudge.Kind != NudgeKindUnknownTool {
		t.Errorf("expected kind 'unknown_tool', got '%s'", nudge.Kind)
	}
	if nudge.Tier != 0 {
		t.Errorf("expected tier 0, got %d", nudge.Tier)
	}
}

func TestStepNudge_Tier1(t *testing.T) {
	pending := []string{"validate", "prepare"}
	nudge := StepNudge("submit", pending, 1)

	if nudge.Kind != NudgeKindStep {
		t.Errorf("expected kind 'step', got '%s'", nudge.Kind)
	}
	if nudge.Tier != 1 {
		t.Errorf("expected tier 1, got %d", nudge.Tier)
	}
}

func TestStepNudge_Tier2(t *testing.T) {
	pending := []string{"validate", "prepare"}
	nudge := StepNudge("submit", pending, 2)

	if nudge.Tier != 2 {
		t.Errorf("expected tier 2, got %d", nudge.Tier)
	}
}

func TestStepNudge_Tier3(t *testing.T) {
	pending := []string{"validate", "prepare"}
	nudge := StepNudge("submit", pending, 3)

	if nudge.Tier != 3 {
		t.Errorf("expected tier 3, got %d", nudge.Tier)
	}
}

func TestStepNudge_TierClamping(t *testing.T) {
	pending := []string{"validate"}

	nudgeLow := StepNudge("submit", pending, 0)
	if nudgeLow.Tier != 1 {
		t.Errorf("expected tier 1 for input 0, got %d", nudgeLow.Tier)
	}

	nudgeHigh := StepNudge("submit", pending, 5)
	if nudgeHigh.Tier != 3 {
		t.Errorf("expected tier 3 for input 5, got %d", nudgeHigh.Tier)
	}
}

func TestPrerequisiteNudge(t *testing.T) {
	missing := []string{"authenticate", "validate"}

	nudge := PrerequisiteNudge("submit", missing)

	if nudge.Kind != NudgeKindPrerequisite {
		t.Errorf("expected kind 'prerequisite', got '%s'", nudge.Kind)
	}
	if nudge.Tier != 0 {
		t.Errorf("expected tier 0, got %d", nudge.Tier)
	}
}

func TestJoinToolNames(t *testing.T) {
	tests := []struct {
		input    []string
		expected string
	}{
		{[]string{}, "(no tools available)"},
		{[]string{"echo"}, "echo"},
		{[]string{"echo", "search"}, "echo and search"},
		{[]string{"echo", "search", "weather"}, "echo, search, and weather"},
	}

	for _, tt := range tests {
		result := joinToolNames(tt.input)
		if result != tt.expected {
			t.Errorf("joinToolNames(%v) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
