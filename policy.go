package middleware

import (
	"fmt"
	"regexp"
	"sync"
	"time"
)

type PolicyEvaluator interface {
	Evaluate(callerCtx CallerContext, toolName string, args map[string]interface{}) Decision
	FilterResult(callerCtx CallerContext, toolName string, result *ToolResult) *ToolResult
	FilterToolList(tools []ToolDefinition) []ToolDefinition
}

type policyEngine struct {
	policy           Policy
	rateLimiters     map[string]*rateLimiter
	turnCounters     map[string]int
	iterationCounter int
	mu               sync.RWMutex
}

type rateLimiter struct {
	count     int
	window    time.Duration
	resetTime time.Time
	mu        sync.Mutex
}

func NewPolicyEvaluator(policy Policy) PolicyEvaluator {
	pe := &policyEngine{
		policy:       policy,
		rateLimiters: make(map[string]*rateLimiter),
		turnCounters: make(map[string]int),
	}
	return pe
}

func (pe *policyEngine) Evaluate(callerCtx CallerContext, toolName string, args map[string]interface{}) Decision {
	for _, rule := range pe.policy.Rules {
		if !ruleMatchesTool(rule, toolName) {
			continue
		}

		if pe.shouldSkipQuickFilter(rule, callerCtx, args) {
			continue
		}

		if !ruleConditionsMatch(rule.Conditions, callerCtx, args) {
			continue
		}

		if !pe.checkRateLimit(rule, callerCtx, toolName) {
			return Decision{
				Timestamp: time.Now(),
				Tool:      toolName,
				Args:      args,
				Action:    ActionDeny,
				Rule:      rule.Name,
				Reason:    fmt.Sprintf("rate limit exceeded for tool: %s", toolName),
				CallerCtx: callerCtx,
			}
		}

		if !pe.checkMaxTurns(rule, callerCtx) {
			return Decision{
				Timestamp: time.Now(),
				Tool:      toolName,
				Args:      args,
				Action:    ActionDeny,
				Rule:      rule.Name,
				Reason:    "max turns exceeded for session",
				CallerCtx: callerCtx,
			}
		}

		if !pe.checkIterationLimit(rule) {
			return Decision{
				Timestamp: time.Now(),
				Tool:      toolName,
				Args:      args,
				Action:    ActionDeny,
				Rule:      rule.Name,
				Reason:    "max iterations exceeded",
				CallerCtx: callerCtx,
			}
		}

		if rule.Action == ActionDeny {
			return Decision{
				Timestamp: time.Now(),
				Tool:      toolName,
				Args:      args,
				Action:    ActionDeny,
				Rule:      rule.Name,
				Reason:    fmt.Sprintf("denied by rule: %s", rule.Name),
				CallerCtx: callerCtx,
			}
		}

		if rule.Action == ActionFilter {
			return Decision{
				Timestamp: time.Now(),
				Tool:      toolName,
				Args:      args,
				Action:    ActionFilter,
				Rule:      rule.Name,
				Reason:    fmt.Sprintf("filtered by rule: %s", rule.Name),
				CallerCtx: callerCtx,
			}
		}

		return Decision{
			Timestamp: time.Now(),
			Tool:      toolName,
			Args:      args,
			Action:    ActionAllow,
			Rule:      rule.Name,
			Reason:    fmt.Sprintf("allowed by rule: %s", rule.Name),
			CallerCtx: callerCtx,
		}
	}

	defaultAction := ActionAllow
	if pe.policy.DefaultDeny {
		defaultAction = ActionDeny
	}
	return Decision{
		Timestamp: time.Now(),
		Tool:      toolName,
		Args:      args,
		Action:    defaultAction,
		Rule:      "default",
		Reason:    fmt.Sprintf("no matching rules, default %s", defaultAction),
		CallerCtx: callerCtx,
	}
}

func (pe *policyEngine) FilterResult(callerCtx CallerContext, toolName string, result *ToolResult) *ToolResult {
	if result == nil {
		return result
	}

	for _, rule := range pe.policy.Rules {
		if !ruleMatchesTool(rule, toolName) {
			continue
		}

		if rule.Action != ActionFilter {
			continue
		}

		if len(rule.RedactFields) > 0 {
			result = pe.filterFields(result, rule.RedactFields)
		}
	}

	return result
}

func (pe *policyEngine) FilterToolList(tools []ToolDefinition) []ToolDefinition {
	var allowed []ToolDefinition

	for _, tool := range tools {
		decision := pe.Evaluate(CallerContext{Trusted: true}, tool.Name, nil)
		if decision.Action == ActionAllow {
			allowed = append(allowed, tool)
		}
	}

	return allowed
}

func (pe *policyEngine) shouldSkipQuickFilter(rule Rule, callerCtx CallerContext, args map[string]interface{}) bool {
	if rule.QuickFilter == nil {
		return false
	}

	return ruleConditionsMatch(rule.QuickFilter.SkipWhen, callerCtx, args)
}

func (pe *policyEngine) checkRateLimit(rule Rule, callerCtx CallerContext, toolName string) bool {
	if rule.MaxRate == nil {
		return true
	}

	key := fmt.Sprintf("%s:%s", callerCtx.SessionID, toolName)
	pe.mu.Lock()
	defer pe.mu.Unlock()

	limiter, exists := pe.rateLimiters[key]
	if !exists {
		limiter = &rateLimiter{
			count:     0,
			window:    rule.MaxRate.Window,
			resetTime: time.Now().Add(rule.MaxRate.Window),
		}
		pe.rateLimiters[key] = limiter
	}

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	if time.Now().After(limiter.resetTime) {
		limiter.count = 0
		limiter.resetTime = time.Now().Add(limiter.window)
	}

	if limiter.count >= rule.MaxRate.Count {
		return false
	}

	limiter.count++
	return true
}

func (pe *policyEngine) checkMaxTurns(rule Rule, callerCtx CallerContext) bool {
	if rule.MaxTurns == nil {
		return true
	}

	pe.mu.Lock()
	defer pe.mu.Unlock()

	count := pe.turnCounters[callerCtx.SessionID]
	if count >= *rule.MaxTurns {
		return false
	}

	pe.turnCounters[callerCtx.SessionID] = count + 1
	return true
}

func (pe *policyEngine) checkIterationLimit(rule Rule) bool {
	if rule.IterationLimit == nil {
		return true
	}

	pe.mu.Lock()
	defer pe.mu.Unlock()

	if pe.iterationCounter >= *rule.IterationLimit {
		return false
	}

	pe.iterationCounter++
	return true
}

func (pe *policyEngine) filterFields(result *ToolResult, fields []string) *ToolResult {
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}

	for _, field := range fields {
		delete(extractMetadataMap(result.Content), field)
		delete(result.Metadata, field)
	}

	return result
}

func extractMetadataMap(content interface{}) map[string]interface{} {
	if m, ok := content.(map[string]interface{}); ok {
		return m
	}
	return nil
}

func ruleMatchesTool(rule Rule, toolName string) bool {
	for _, t := range rule.Tools {
		if t == "*" {
			return true
		}
		if t == toolName {
			return true
		}
	}
	return false
}

func ruleConditionsMatch(conditions []Condition, callerCtx CallerContext, args map[string]interface{}) bool {
	if len(conditions) == 0 {
		return true
	}

	for _, cond := range conditions {
		if !conditionMatches(cond, callerCtx, args) {
			return false
		}
	}
	return true
}

func conditionMatches(cond Condition, callerCtx CallerContext, args map[string]interface{}) bool {
	value := resolveField(cond.Field, callerCtx, args)
	valueStr := fmt.Sprintf("%v", value)

	switch cond.Operator {
	case "eq":
		return valueStr == cond.Value
	case "not_eq":
		return valueStr != cond.Value
	case "contains":
		return len(valueStr) > 0 && len(cond.Value) > 0 && contains(valueStr, cond.Value)
	case "not_contains":
		return !contains(valueStr, cond.Value)
	case "matches":
		matched, _ := regexp.MatchString(cond.Value, valueStr)
		return matched
	case "not_matches":
		matched, _ := regexp.MatchString(cond.Value, valueStr)
		return !matched
	case "not":
		return valueStr == ""
	case "exists":
		return value != nil
	case "not_exists":
		return value == nil
	default:
		return false
	}
}

func resolveField(field string, callerCtx CallerContext, args map[string]interface{}) interface{} {
	switch field {
	case "caller.role":
		return callerCtx.Role
	case "caller.user_id":
		return callerCtx.UserID
	case "caller.session_id":
		return callerCtx.SessionID
	case "caller.source":
		return callerCtx.Source
	case "context.trusted":
		return callerCtx.Trusted
	default:
		if len(field) > 5 && field[:5] == "args." {
			argKey := field[5:]
			if args != nil {
				return args[argKey]
			}
		}
		if len(field) > 8 && field[:8] == "context." {
			ctxKey := field[8:]
			if callerCtx.Metadata != nil {
				return callerCtx.Metadata[ctxKey]
			}
		}
	}
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
