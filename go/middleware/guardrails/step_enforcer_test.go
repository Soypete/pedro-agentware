package guardrails

import (
	"errors"
	"testing"
)

func TestStepEnforcer_AddStep(t *testing.T) {
	se := NewStepEnforcer()
	se.AddStep("deploy", []string{"build", "test"})

	allowed, missing := se.CanExecute("session1", "deploy")
	if allowed {
		t.Error("expected not allowed without prerequisites")
	}
	if len(missing) != 2 {
		t.Errorf("expected 2 missing steps, got %d", len(missing))
	}
}

func TestStepEnforcer_MarkStepComplete(t *testing.T) {
	se := NewStepEnforcer()
	se.AddStep("deploy", []string{"build", "test"})

	se.MarkStepComplete("session1", "build")
	allowed, missing := se.CanExecute("session1", "deploy")

	if allowed {
		t.Error("expected not allowed with only one prerequisite")
	}
	if len(missing) != 1 || missing[0] != "test" {
		t.Errorf("expected missing ['test'], got %v", missing)
	}
}

func TestStepEnforcer_ValidateExecution_Success(t *testing.T) {
	se := NewStepEnforcer()
	se.AddStep("deploy", []string{"build"})

	se.MarkStepComplete("session1", "build")
	err := se.ValidateExecution("session1", "deploy")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestStepEnforcer_ValidateExecution_Failure(t *testing.T) {
	se := NewStepEnforcer()
	se.AddStep("deploy", []string{"build"})

	err := se.ValidateExecution("session1", "deploy")

	if err == nil {
		t.Error("expected error")
	}
	if !errors.Is(err, ErrStepNotAllowed) {
		t.Errorf("expected ErrStepNotAllowed, got %v", err)
	}
}

func TestStepEnforcer_ResetSession(t *testing.T) {
	se := NewStepEnforcer()
	se.AddStep("deploy", []string{"build"})

	se.MarkStepComplete("session1", "build")
	se.ResetSession("session1")

	allowed, _ := se.CanExecute("session1", "deploy")
	if allowed {
		t.Error("expected not allowed after reset")
	}
}

func TestStepEnforcer_IsTerminalAllowed(t *testing.T) {
	se := NewStepEnforcer()
	se.AddStep("deploy", []string{"build"})
	se.AddStep("build", nil)

	se.MarkStepComplete("session1", "build")
	allowed := se.IsTerminalAllowed("session1", "deploy")

	if !allowed {
		t.Error("expected terminal allowed")
	}
}

func TestStepEnforcer_GetAllowedTerminals(t *testing.T) {
	se := NewStepEnforcer()
	se.AddStep("deploy", []string{"build"})
	se.AddStep("test", nil)

	allowed := se.GetAllowedTerminals("session1")

	if len(allowed) != 1 || allowed[0] != "test" {
		t.Errorf("expected ['test'], got %v", allowed)
	}
}

func TestStepEnforcer_NoPrerequisites(t *testing.T) {
	se := NewStepEnforcer()
	se.AddStep("build", nil)

	allowed, _ := se.CanExecute("session1", "build")
	if !allowed {
		t.Error("expected allowed with no prerequisites")
	}
}

func TestStepEnforcer_InvalidSession(t *testing.T) {
	se := NewStepEnforcer()
	se.AddStep("deploy", []string{"build"})

	allowed, _ := se.CanExecute("nonexistent", "deploy")
	if allowed {
		t.Error("expected not allowed for invalid session")
	}
}
