package middleware

import "time"

type PolicyEvaluator interface {
	Evaluate(toolName string, args map[string]any, caller CallerContext) Decision
}

type Policy struct {
	Rules       []Rule
	DefaultDeny bool
}

type Rule struct {
	Name         string
	Tools        []string
	Action       Action
	Conditions   []Condition
	MaxRate      *RateLimit
	RedactFields []string
}

type RateLimit struct {
	Count  int
	Window time.Duration
}

type Condition struct {
	Field    string
	Operator Operator
	Value    string
}

type Operator string

const (
	OperatorEq          Operator = "eq"
	OperatorNotEq       Operator = "not_eq"
	OperatorContains    Operator = "contains"
	OperatorNotContains Operator = "not_contains"
	OperatorMatches     Operator = "matches"
	OperatorNotMatches  Operator = "not_matches"
	OperatorExists      Operator = "exists"
	OperatorNotExists   Operator = "not_exists"
)

func (p *Policy) Evaluate(toolName string, args map[string]any, caller CallerContext) Decision {
	for _, rule := range p.Rules {
		if !rule.matchesTool(toolName) {
			continue
		}
		if !rule.evaluateConditions(args, caller) {
			continue
		}
		return Decision{
			Action:    rule.Action,
			Rule:      rule.Name,
			Reason:    "matched rule " + rule.Name,
			Timestamp: time.Now(),
		}
	}

	if p.DefaultDeny {
		return Decision{
			Action:    ActionDeny,
			Rule:      "default",
			Reason:    "no matching rules and default deny is enabled",
			Timestamp: time.Now(),
		}
	}

	return Decision{
		Action:    ActionAllow,
		Rule:      "default",
		Reason:    "no matching rules and default allow is enabled",
		Timestamp: time.Now(),
	}
}

func (r *Rule) matchesTool(toolName string) bool {
	for _, t := range r.Tools {
		if t == "*" || t == toolName {
			return true
		}
	}
	return false
}

func (r *Rule) evaluateConditions(args map[string]any, caller CallerContext) bool {
	if len(r.Conditions) == 0 {
		return true
	}
	for _, cond := range r.Conditions {
		if !cond.evaluate(args, caller) {
			return false
		}
	}
	return true
}

func (c *Condition) evaluate(args map[string]any, caller CallerContext) bool {
	var value string
	switch {
	case c.Field == "caller.role":
		value = caller.Role
	case c.Field == "caller.source":
		value = caller.Source
	case c.Field == "caller.trusted":
		if caller.Trusted {
			value = "true"
		} else {
			value = "false"
		}
	case c.Field == "caller.user_id":
		value = caller.UserID
	case c.Field == "caller.session_id":
		value = caller.SessionID
	case len(c.Field) > 5 && c.Field[:5] == "args.":
		argKey := c.Field[5:]
		if v, ok := args[argKey]; ok {
			if sv, ok := v.(string); ok {
				value = sv
			}
		}
	default:
		return false
	}

	switch c.Operator {
	case OperatorEq:
		return value == c.Value
	case OperatorNotEq:
		return value != c.Value
	case OperatorContains:
		return len(value) > 0 && len(c.Value) > 0 && contains(value, c.Value)
	case OperatorNotContains:
		return !contains(value, c.Value)
	case OperatorMatches:
		return matches(value, c.Value)
	case OperatorNotMatches:
		return !matches(value, c.Value)
	case OperatorExists:
		return value != ""
	case OperatorNotExists:
		return value == ""
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func matches(s, pattern string) bool {
	return simpleMatch(s, pattern)
}

func simpleMatch(s, pattern string) bool {
	if len(pattern) == 0 {
		return len(s) == 0
	}
	if len(s) == 0 {
		return false
	}
	return s == pattern
}
