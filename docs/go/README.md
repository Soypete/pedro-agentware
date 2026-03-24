# Go Middleware Usage Examples

This document provides examples of how to use the Go middleware for policy enforcement.

## Basic Usage

### Creating a Policy

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/soypete/pedro-agentware/middleware"
    "github.com/soypete/pedro-agentware/middleware/types"
)

func main() {
    policy := types.Policy{
        DefaultDeny: false,
        Rules: []types.Rule{
            {
                Name:  "rate-limit-tools",
                Tools: []string{"*"},
                Action: types.ActionAllow,
                MaxRate: &types.RateLimit{
                    Count:  10,
                    Window: 60 * time.Second,
                },
            },
            {
                Name:  "deny-admin",
                Tools: []string{"delete_database", "drop_table"},
                Action: types.ActionDeny,
                Conditions: []types.Condition{
                    {
                        Field:    "caller.trusted",
                        Operator: "eq",
                        Value:    "false",
                    },
                },
            },
        },
    }
}
```

### Wrapping a Tool Executor

```go
// Create a custom tool executor (implementing types.ToolExecutor)
type MyToolExecutor struct{}

func (e *MyToolExecutor) CallTool(ctx context.Context, name string, args map[string]interface{}) (*types.ToolResult, error) {
    // Your tool execution logic here
    return &types.ToolResult{
        Content:  fmt.Sprintf("Executed %s", name),
        Metadata: map[string]interface{}{},
    }, nil
}

func (e *MyToolExecutor) ListTools() []types.ToolDefinition {
    return []types.ToolDefinition{
        {Name: "read_file", Description: "Read a file"},
        {Name: "write_file", Description: "Write a file"},
    }
}

// Create middleware
executor := &MyToolExecutor{}
mw := middleware.New(executor, policy)

// Call a tool through middleware
result, err := mw.CallTool(context.Background(), "read_file", map[string]interface{}{
    "path": "/tmp/test.txt",
})
```

### Using Caller Context

```go
// Create caller context with user information
callerCtx := types.CallerContext{
    Trusted:   true,
    Role:      "user",
    UserID:    "user-123",
    SessionID: "session-456",
    Source:    "cli",
}

// Add context to request
ctx := middleware.WithCallerContext(context.Background(), callerCtx)

result, err := mw.CallTool(ctx, "read_file", map[string]interface{}{
    "path": "/tmp/test.txt",
})
```

### Using Audit

```go
import "github.com/soypete/pedro-agentware/middleware/audit"

// Create in-memory auditor
auditor := audit.NewInMemoryAuditor()

// Configure middleware with auditor
mw := middleware.New(executor, policy, middleware.WithAuditor(auditor))

// After tool calls, get audit log
log := auditor.GetLog()
for _, entry := range log {
    fmt.Printf("Decision: %s, Tool: %s, Rule: %s\n",
        entry.Decision.Action, entry.Decision.Tool, entry.Decision.Rule)
}
```

## Loading Policy from YAML

```go
import "github.com/soypete/pedro-agentware/middleware"

func main() {
    // Load policy from YAML file
    policy, err := middleware.LoadPolicyFromFile("policy.yaml")
    if err != nil {
        panic(err)
    }

    mw := middleware.New(executor, *policy)
}
```

Example `policy.yaml`:

```yaml
rules:
  - name: "rate-limit-read"
    tools:
      - "read_file"
      - "search"
    action: "allow"
    max_rate:
      count: 5
      window: 60

  - name: "deny-admin-tools"
    tools:
      - "delete_database"
    action: "deny"
    conditions:
      - field: "caller.trusted"
        operator: "eq"
        value: "false"

default_deny: false
```

## Filtering Tool List

The middleware can filter available tools based on policy rules:

```go
// Get list of allowed tools for a caller
tools := mw.ListTools()
for _, tool := range tools {
    fmt.Println(tool.Name)
}
```

### Phase-based Tool Filtering

For multi-phase workflows (like PedroCLI's phased executor):

```go
callerCtx := types.CallerContext{
    Phase: "planning", // Phase name
}

// Get tools available for a specific phase
tools := mw.GetToolsForPhase("planning", callerCtx)
```

This filters out:
- Tools already called in this phase
- Tools that have failed 3+ times in this phase

## Condition Operators

| Operator | Description |
|----------|-------------|
| `eq` | Field equals value |
| `not_eq` | Field does not equal value |
| `contains` | Field contains value |
| `not_contains` | Field does not contain value |
| `matches` | Field matches regex pattern |
| `not_matches` | Field does not match regex pattern |
| `exists` | Field exists |
| `not_exists` | Field does not exist |
| `not` | Field is empty |

## Field Resolution

Conditions can reference:
- `caller.role` - Caller's role
- `caller.user_id` - User ID
- `caller.session_id` - Session ID
- `caller.source` - Call source
- `caller.trusted` - Whether caller is trusted
- `args.<name>` - Tool argument values
- `context.<key>` - Custom context metadata

## Using with PedroCLI

The middleware integrates with PedroCLI's ToolBridge pattern:

```go
import "github.com/soypete/pedro-cli/bridge"

func main() {
    // Create middleware wrapping your executor
    mw := middleware.New(executor, policy)

    // Use with PedroCLI bridge
    b := bridge.NewBridge(mw)
    // ... continue with PedroCLI setup
}
```