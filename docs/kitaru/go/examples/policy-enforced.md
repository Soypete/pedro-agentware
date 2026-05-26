# Policy-Enforced Agent Example

This example demonstrates integrating kitaru-go with pedro-agentware middleware for policy enforcement.

## Overview

The agent:
1. Uses Kitaru for durable execution (checkpoints, replay on failure)
2. Calls tools through pedro-agentware middleware
3. Middleware enforces rate limits, denies destructive operations, and audits all calls

## Project Structure

```
my-agent/
├── main.go
├── policy.yaml
├── go.mod
└── go.sum
```

## Policy Configuration (policy.yaml)

```yaml
# Policy for Kitaru agent with policy enforcement

rules:
  # Rate limit web searches
  - name: "rate-limit-search"
    tools:
      - "web_search"
    action: "allow"
    max_rate:
      count: 10
      window: 60

  # Rate limit data processing
  - name: "rate-limit-process"
    tools:
      - "process_data"
    action: "allow"
    max_rate:
      count: 5
      window: 300

  # Deny dangerous operations for untrusted callers
  - name: "deny-destructive-untrusted"
    tools:
      - "delete_*"
      - "drop_*"
      - "truncate_*"
    action: "deny"
    conditions:
      - field: "caller.trusted"
        operator: "eq"
        value: false

  # Allow research flows for all
  - name: "allow-research"
    tools:
      - "research_flow"
      - "analyze_flow"
    action: "allow"

  # Audit all tool calls
  - name: "audit-all"
    tools:
      - "*"
    action: "allow"
    audit: true

default_deny: false
```

## Main Application (main.go)

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    kitaru "github.com/zenml-io/kitaru-sdk-go"
    "github.com/soypete/pedro-agentware/middleware"
    "github.com/soypete/pedro-agentware/middleware/types"
    "github.com/soypete/pedro-agentware/middleware/audit"
)

// KitaruToolExecutor wraps Kitaru flows as middleware tools
type KitaruToolExecutor struct {
    client   *kitaru.Client
    toolMap  map[string]string // tool name -> flow name
}

func NewKitaruToolExecutor(client *kitaru.Client) *KitaruToolExecutor {
    return &KitaruToolExecutor{
        client: client,
        toolMap: map[string]string{
            "research_flow":  "research-agent",
            "analyze_flow":   "data-analyzer",
            "process_data":   "data-processor",
            "generate_report": "report-generator",
        },
    }
}

func (e *KitaruToolExecutor) CallTool(ctx context.Context, name string, args map[string]interface{}) (*types.ToolResult, error) {
    flowName, ok := e.toolMap[name]
    if !ok {
        // Handle built-in tools
        return e.callBuiltInTool(ctx, name, args)
    }

    flow := e.client.Flow(flowName)
    exec, err := flow.RunWithWait(ctx, args, 2*time.Second)
    if err != nil {
        return &types.ToolResult{
            Success: false,
            Error:   err.Error(),
        }, err
    }

    return &types.ToolResult{
        Success:  exec.Status == kitaru.StatusCompleted,
        Content:  fmt.Sprintf("%v", exec.Output),
        Metadata: map[string]interface{}{
            "execution_id": exec.ID,
            "status":       exec.Status,
        },
    }, nil
}

func (e *KitaruToolExecutor) callBuiltInTool(ctx context.Context, name string, args map[string]interface{}) (*types.ToolResult, error) {
    switch name {
    case "web_search":
        query := args["query"].(string)
        return &types.ToolResult{
            Success: true,
            Content: fmt.Sprintf("Search results for: %s", query),
            Metadata: map[string]interface{}{"query": query},
        }, nil

    case "save_file":
        return &types.ToolResult{
            Success: true,
            Content: "File saved successfully",
        }, nil

    default:
        return nil, fmt.Errorf("unknown tool: %s", name)
    }
}

func (e *KitaruToolExecutor) ListTools() []types.ToolDefinition {
    tools := []types.ToolDefinition{
        // Kitaru flows
        {Name: "research_flow", Description: "Research topic via Kitaru flow"},
        {Name: "analyze_flow", Description: "Analyze data via Kitaru flow"},
        {Name: "process_data", Description: "Process data via Kitaru flow"},
        {Name: "generate_report", Description: "Generate report via Kitaru flow"},
        // Built-in tools
        {Name: "web_search", Description: "Search the web"},
        {Name: "save_file", Description: "Save content to file"},
    }
    return tools
}

func main() {
    // 1. Create Kitaru client
    kitaruClient := kitaru.NewClient(
        getEnv("KITARU_URL", "http://localhost:8080"),
        getEnv("KITARU_API_KEY", ""),
        getEnv("KITARU_PROJECT", "default"),
    )

    // 2. Create executor
    executor := NewKitaruToolExecutor(kitaruClient)

    // 3. Load policy
    policy, err := middleware.LoadPolicyFromFile("policy.yaml")
    if err != nil {
        log.Fatalf("Failed to load policy: %v", err)
    }

    // 4. Create auditor for logging
    auditor := audit.NewInMemoryAuditor()

    // 5. Create middleware with policy and audit
    mw := middleware.New(executor, *policy, middleware.WithAuditor(auditor))

    // 6. Create caller context (e.g., from authenticated user)
    ctx := middleware.WithCallerContext(context.Background(), types.CallerContext{
        Trusted:   true,        // Set based on authentication
        Role:      "user",
        UserID:    "user-123",
        SessionID: "session-456",
        Source:    "api",
    })

    // 7. Run agent tasks through middleware
    runAgentTasks(mw, ctx)
}

func runAgentTasks(mw *middleware.Middleware, ctx context.Context) {
    // Task 1: Web search (rate limited)
    fmt.Println("\n--- Task 1: Web Search ---")
    for i := 0; i < 3; i++ {
        result, err := mw.CallTool(ctx, "web_search", map[string]interface{}{
            "query": fmt.Sprintf("Go middleware patterns #%d", i+1),
        })
        if err != nil {
            log.Printf("Search %d denied: %v", i+1, err)
        } else {
            log.Printf("Search %d result: %s", i+1, result.Content)
        }
        time.Sleep(100 * time.Millisecond)
    }

    // Task 2: Run research flow through Kitaru
    fmt.Println("\n--- Task 2: Kitaru Research Flow ---")
    result, err := mw.CallTool(ctx, "research_flow", map[string]interface{}{
        "topic": "agent middleware patterns",
    })
    if err != nil {
        log.Printf("Research flow error: %v", err)
    } else {
        log.Printf("Research result: %s", result.Content)
    }

    // Task 3: Try destructive operation (should be denied for untrusted)
    fmt.Println("\n--- Task 3: Attempt Destructive Operation ---")
    untrustedCtx := middleware.WithCallerContext(context.Background(), types.CallerContext{
        Trusted:   false,
        Role:      "guest",
        UserID:    "guest-456",
        Source:    "public-api",
    })

    result, err = mw.CallTool(untrustedCtx, "delete_database", map[string]interface{}{
        "confirm": true,
    })
    if err != nil {
        log.Printf("Delete denied (expected): %v", err)
    } else {
        log.Printf("Delete result: %s", result.Content)
    }

    // 8. Print audit log
    fmt.Println("\n--- Audit Log ---")
    logEntries := auditor.GetLog()
    for _, entry := range logEntries {
        fmt.Printf("Tool: %-20s | Decision: %-6s | Rule: %s\n",
            entry.ToolCall.ToolName,
            entry.Decision.Action,
            entry.Decision.Rule,
        )
    }
}

func getEnv(key, defaultValue string) string {
    if value := getEnvInternal(key); value != "" {
        return value
    }
    return defaultValue
}

func getEnvInternal(key string) string {
    // Simple env getter for example
    // In production, use os.Getenv
    return ""
}
```

## Running the Example

```bash
# Install dependencies
go mod init agent-example
go get github.com/zenml-io/kitaru-sdk-go
go get github.com/soypete/pedro-agentware/middleware

# Run
go run main.go
```

## Expected Output

```
--- Task 1: Web Search ---
2026/05/26 10:00:00 Search 1 result: Search results for: Go middleware patterns #1
2026/05/26 10:00:00 Search 2 result: Search results for: Go middleware patterns #2
2026/05/26 10:00:00 Search 3 result: Search results for: Go middleware patterns #3

--- Task 2: Kitaru Research Flow ---
2026/05/26 10:00:01 Research result: <kitaru flow output>

--- Task 3: Attempt Destructive Operation ---
2026/05/26 10:00:02 Delete denied (expected): tool call denied by rule: deny-destructive-untrusted

--- Audit Log ---
Tool: web_search         | Decision: allow  | Rule: rate-limit-search
Tool: web_search         | Decision: allow  | Rule: rate-limit-search
Tool: web_search         | Decision: allow  | Rule: rate-limit-search
Tool: research_flow      | Decision: allow  | Rule: allow-research
Tool: delete_database    | Decision: deny   | Rule: deny-destructive-untrusted
```

## Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: policy-agent
spec:
  replicas: 2
  selector:
    matchLabels:
      app: policy-agent
  template:
    metadata:
      labels:
        app: policy-agent
    spec:
      containers:
      - name: agent
        image: your-agent:latest
        env:
        - name: KITARU_URL
          value: "http://kitaru-service.default.svc.cluster.local:8080"
        - name: KITARU_API_KEY
          valueFrom:
            secretKeyRef:
              name: kitaru-credentials
              key: api-key
        - name: KITARU_PROJECT
          value: "production"
        volumeMounts:
        - name: policy
          mountPath: /app/policy.yaml
          subPath: policy.yaml
      volumes:
      - name: policy
        configMap:
          name: agent-policy
```

## Key Features Demonstrated

1. **Rate Limiting** - Middleware enforces per-tool rate limits
2. **Trust-Based Access** - Different policies for trusted vs untrusted callers
3. **Audit Logging** - All tool calls are logged for compliance
4. **Durable Execution** - Kitaru provides checkpoint/replay on failure
5. **Tool Abstraction** - Kitaru flows exposed seamlessly as middleware tools