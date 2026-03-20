package middleware

import (
	"testing"
	"time"
)

func TestInMemoryAuditor_NewInMemoryAuditor(t *testing.T) {
	aud := NewInMemoryAuditor(0)
	if aud == nil {
		t.Error("NewInMemoryAuditor returned nil")
	}
}

func TestInMemoryAuditor_NewInMemoryAuditor_Negative(t *testing.T) {
	aud := NewInMemoryAuditor(-5)
	if aud == nil {
		t.Error("NewInMemoryAuditor returned nil")
	}
}

func TestInMemoryAuditor_Record(t *testing.T) {
	aud := NewInMemoryAuditor(100)
	decision := Decision{
		Timestamp: time.Now(),
		Tool:      "test-tool",
		Action:    ActionAllow,
	}
	aud.Record(decision)
}

func TestInMemoryAuditor_Record_Rollover(t *testing.T) {
	aud := NewInMemoryAuditor(2)
	for i := 0; i < 5; i++ {
		aud.Record(Decision{
			Timestamp: time.Now(),
			Tool:      "tool",
			Action:    ActionAllow,
		})
	}
}

func TestInMemoryAuditor_GetViolations(t *testing.T) {
	aud := NewInMemoryAuditor(100)
	now := time.Now()

	aud.Record(Decision{
		Timestamp: now,
		Tool:      "tool1",
		Action:    ActionDeny,
	})
	aud.Record(Decision{
		Timestamp: now,
		Tool:      "tool2",
		Action:    ActionAllow,
	})
	aud.Record(Decision{
		Timestamp: now.Add(-time.Hour),
		Tool:      "tool3",
		Action:    ActionDeny,
	})

	violations := aud.GetViolations(now.Add(-time.Minute))
	if len(violations) != 1 {
		t.Errorf("expected 1 violation, got %d", len(violations))
	}
}

func TestInMemoryAuditor_GetDecisions(t *testing.T) {
	aud := NewInMemoryAuditor(100)
	now := time.Now()

	aud.Record(Decision{
		Timestamp: now,
		Tool:      "tool1",
		Action:    ActionAllow,
	})
	aud.Record(Decision{
		Timestamp: now,
		Tool:      "tool1",
		Action:    ActionDeny,
	})
	aud.Record(Decision{
		Timestamp: now.Add(-time.Hour),
		Tool:      "tool1",
		Action:    ActionAllow,
	})

	decisions := aud.GetDecisions("tool1", now.Add(-time.Minute))
	if len(decisions) != 2 {
		t.Errorf("expected 2 decisions, got %d", len(decisions))
	}
}

func TestInMemoryAuditor_Close(t *testing.T) {
	aud := NewInMemoryAuditor(100)
	if err := aud.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestNoOpAuditor(t *testing.T) {
	aud := NoOpAuditor{}

	aud.Record(Decision{Action: ActionDeny})

	violations := aud.GetViolations(time.Now())
	if violations != nil {
		t.Error("expected nil violations")
	}

	decisions := aud.GetDecisions("tool", time.Now())
	if decisions != nil {
		t.Error("expected nil decisions")
	}

	if err := aud.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
