# Agent Middleware

MCP-compatible middleware SDK for Go that enforces policies on context, tools, and data. The middleware sits between the agent (LLM orchestrator) and tool execution, intercepting every tool call to enforce policies before allowing execution.

## Features

- **Tool Policy Enforcement** - Allow/deny tools based on rules
- **Rate Limiting** - Per-tool, per-user rate limits
- **Iteration Limits** - Control max iterations per tool
- **Response Filtering** - Filter sensitive fields from results
- **Audit Logging** - Track all tool calls and decisions
- **Context Control** - Mark contexts as trusted/untrusted

## Installation

```bash
go get github.com/pedro/agent-middleware
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"

	"github.com/pedro/agent-middleware"
)

func main() {
	// Create a policy
	policy := &middleware.Policy{
		DefaultDeny: true,
		Rules: []middleware.Rule{
			{
				Name:  "allow-read",
				Tools: []string{"read", "list"},
				Action: middleware.ActionAllow,
			},
		},
	}

	// Create evaluator and auditor
	evaluator := middleware.NewPolicyEvaluator(policy)
	auditor := middleware.NewInMemoryAuditor(100)

	// Create middleware
	mw := middleware.New(
		middleware.WithPolicyEvaluator(evaluator),
		middleware.WithAuditor(auditor),
	)

	// Wrap your tool executor
	executor := &MyToolExecutor{}
	wrapped := mw.Wrap(executor)

	// Use the wrapped executor
	ctx := context.Background()
	result, err := wrapped.CallTool(ctx, "read", map[string]interface{}{"path": "/foo"})
	fmt.Println(result, err)
}

// Implement ToolExecutor interface
type MyToolExecutor struct{}

func (e *MyToolExecutor) CallTool(ctx context.Context, name string, args map[string]interface{}) (*middleware.ToolResult, error) {
	return &middleware.ToolResult{Content: "result"}, nil
}

func (e *MyToolExecutor) ListTools() []middleware.ToolDefinition {
	return []middleware.ToolDefinition{
		{Name: "read", Description: "Read a file"},
		{Name: "write", Description: "Write a file"},
	}
}
```

## Policy Configuration

Policies can be defined programmatically or loaded from YAML:

```go
// From YAML
policy, err := middleware.LoadPolicyFromString(`
rules:
  - name: allow-read
    tools:
      - "read"
      - "list"
    action: allow
  - name: deny-dangerous
    tools:
      - "delete"
      - "drop"
    action: deny
default_deny: true
`)
```

## Rule Conditions

Rules support conditional evaluation:

```go
Rules: []middleware.Rule{
	{
		Name:  "rate-limited-read",
		Tools: []string{"read"},
		Action: middleware.ActionAllow,
		Conditions: []middleware.Condition{
			{
				Type:  middleware.ConditionTypeRateLimit,
				Key:   "user_id",
				Value: 10, // 10 requests per minute
			},
		},
	},
}
```

## Response Filtering

Filter sensitive data from tool responses:

```go
policy := &middleware.Policy{
	Rules: []middleware.Rule{
		{
			Name:  "filter-sensitive",
			Tools: []string{"get-user"},
			Action: middleware.ActionAllow,
			ResultFilters: []middleware.ResultFilter{
				{
					Fields: []string{"password", "api_key", "token"},
					Action: middleware.FilterActionRemove,
				},
			},
		},
	},
}
```

## Audit Logging

Track all tool calls and policy decisions:

```go
auditor := middleware.NewInMemoryAuditor(1000)

// After tool execution
violations := auditor.GetViolations()
decisions := auditor.GetDecisions()

for _, d := range decisions {
	fmt.Printf("Tool: %s, Action: %s, Caller: %s\n", 
		d.ToolName, d.Action, d.CallerID)
}
```

## Context Control

Mark contexts as trusted/untrusted for additional security:

```go
ctx := context.Background()
trustedCtx := middleware.WithCallerContext(ctx, middleware.CallerContext{
	CallerID:    "user-123",
	IsTrusted:   true,
	TrustLevel:  "high",
	Metadata:    map[string]interface{}{"source": "internal"},
})
```

## Examples

See `examples/main.go` for complete usage examples.

## License

MIT