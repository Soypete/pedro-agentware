package middleware

import (
	"testing"
	"time"
)

func TestCallerContext(t *testing.T) {
	ctx := CallerContext{
		UserID:    "user123",
		SessionID: "session456",
		Role:      "admin",
		Source:    "cli",
		Trusted:   true,
		Metadata:  map[string]string{"key": "value"},
	}

	if ctx.UserID != "user123" {
		t.Errorf("expected UserID 'user123', got '%s'", ctx.UserID)
	}
	if ctx.Trusted != true {
		t.Error("expected Trusted to be true")
	}
}

func TestDecision(t *testing.T) {
	decision := Decision{
		Action:       ActionAllow,
		Rule:         "test_rule",
		Reason:       "matched test rule",
		RedactedArgs: map[string]any{"secret": "redacted"},
		Timestamp:    time.Now(),
	}

	if decision.Action != ActionAllow {
		t.Errorf("expected ActionAllow, got '%s'", decision.Action)
	}
	if decision.Rule != "test_rule" {
		t.Errorf("expected Rule 'test_rule', got '%s'", decision.Rule)
	}
}

func TestActionConstants(t *testing.T) {
	if ActionAllow != "allow" {
		t.Errorf("expected 'allow', got '%s'", ActionAllow)
	}
	if ActionDeny != "deny" {
		t.Errorf("expected 'deny', got '%s'", ActionDeny)
	}
	if ActionFilter != "filter" {
		t.Errorf("expected 'filter', got '%s'", ActionFilter)
	}
}
