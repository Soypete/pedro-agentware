package middleware

import (
	"testing"
	"time"
)

func TestPolicyEvaluate(t *testing.T) {
	policy := &Policy{
		Rules: []Rule{
			{
				Name:   "allow_read",
				Tools:  []string{"read", "list"},
				Action: ActionAllow,
			},
			{
				Name:   "deny_delete",
				Tools:  []string{"delete"},
				Action: ActionDeny,
			},
		},
		DefaultDeny: false,
	}

	t.Run("allow matching rule", func(t *testing.T) {
		decision := policy.Evaluate("read", map[string]any{}, CallerContext{})
		if decision.Action != ActionAllow {
			t.Errorf("expected ActionAllow, got '%s'", decision.Action)
		}
		if decision.Rule != "allow_read" {
			t.Errorf("expected rule 'allow_read', got '%s'", decision.Rule)
		}
	})

	t.Run("deny matching rule", func(t *testing.T) {
		decision := policy.Evaluate("delete", map[string]any{}, CallerContext{})
		if decision.Action != ActionDeny {
			t.Errorf("expected ActionDeny, got '%s'", decision.Action)
		}
	})

	t.Run("no matching rule with default allow", func(t *testing.T) {
		decision := policy.Evaluate("unknown", map[string]any{}, CallerContext{})
		if decision.Action != ActionAllow {
			t.Errorf("expected ActionAllow, got '%s'", decision.Action)
		}
	})

	t.Run("no matching rule with default deny", func(t *testing.T) {
		policy.DefaultDeny = true
		decision := policy.Evaluate("unknown", map[string]any{}, CallerContext{})
		if decision.Action != ActionDeny {
			t.Errorf("expected ActionDeny, got '%s'", decision.Action)
		}
	})
}

func TestRuleMatchesTool(t *testing.T) {
	rule := Rule{
		Tools: []string{"read", "write", "*"},
	}

	tests := []struct {
		toolName string
		expected bool
	}{
		{"read", true},
		{"write", true},
		{"delete", true},
		{"execute", true},
	}

	for _, tt := range tests {
		result := rule.matchesTool(tt.toolName)
		if result != tt.expected {
			t.Errorf("expected matchesTool('%s') = %v, got %v", tt.toolName, tt.expected, result)
		}
	}
}

func TestRuleMatchesToolWildcard(t *testing.T) {
	rule := Rule{Tools: []string{"*"}}

	if !rule.matchesTool("any_tool") {
		t.Error("expected wildcard to match any tool")
	}
}

func TestRuleEvaluateConditions(t *testing.T) {
	rule := Rule{
		Conditions: []Condition{
			{Field: "caller.role", Operator: OperatorEq, Value: "admin"},
		},
	}

	args := map[string]any{}
	caller := CallerContext{Role: "admin"}
	if !rule.evaluateConditions(args, caller) {
		t.Error("expected conditions to evaluate to true")
	}

	caller.Role = "user"
	if rule.evaluateConditions(args, caller) {
		t.Error("expected conditions to evaluate to false")
	}
}

func TestConditionEvaluate(t *testing.T) {
	t.Run("caller.role eq", func(t *testing.T) {
		cond := Condition{Field: "caller.role", Operator: OperatorEq, Value: "admin"}
		args := map[string]any{}
		caller := CallerContext{Role: "admin"}

		if !cond.evaluate(args, caller) {
			t.Error("expected condition to evaluate to true")
		}
	})

	t.Run("caller.role not_eq", func(t *testing.T) {
		cond := Condition{Field: "caller.role", Operator: OperatorNotEq, Value: "admin"}
		args := map[string]any{}
		caller := CallerContext{Role: "user"}

		if !cond.evaluate(args, caller) {
			t.Error("expected condition to evaluate to true")
		}
	})

	t.Run("caller.trusted", func(t *testing.T) {
		cond := Condition{Field: "caller.trusted", Operator: OperatorEq, Value: "true"}
		args := map[string]any{}
		caller := CallerContext{Trusted: true}

		if !cond.evaluate(args, caller) {
			t.Error("expected condition to evaluate to true")
		}
	})

	t.Run("args.field", func(t *testing.T) {
		cond := Condition{Field: "args.filename", Operator: OperatorEq, Value: "secret.txt"}
		args := map[string]any{"filename": "secret.txt"}
		caller := CallerContext{}

		if !cond.evaluate(args, caller) {
			t.Error("expected condition to evaluate to true")
		}
	})

	t.Run("unknown field", func(t *testing.T) {
		cond := Condition{Field: "unknown", Operator: OperatorEq, Value: "value"}
		args := map[string]any{}
		caller := CallerContext{}

		if cond.evaluate(args, caller) {
			t.Error("expected condition to evaluate to false for unknown field")
		}
	})
}

func TestOperatorConstants(t *testing.T) {
	if OperatorEq != "eq" {
		t.Errorf("expected 'eq', got '%s'", OperatorEq)
	}
	if OperatorNotEq != "not_eq" {
		t.Errorf("expected 'not_eq', got '%s'", OperatorNotEq)
	}
	if OperatorContains != "contains" {
		t.Errorf("expected 'contains', got '%s'", OperatorContains)
	}
	if OperatorExists != "exists" {
		t.Errorf("expected 'exists', got '%s'", OperatorExists)
	}
}

func TestRateLimit(t *testing.T) {
	rl := RateLimit{
		Count:  10,
		Window: time.Minute,
	}

	if rl.Count != 10 {
		t.Errorf("expected Count 10, got %d", rl.Count)
	}
	if rl.Window != time.Minute {
		t.Errorf("expected Window 1m, got %v", rl.Window)
	}
}
