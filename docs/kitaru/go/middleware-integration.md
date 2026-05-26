# Middleware Integration

This guide covers integrating kitaru-go with pedro-agentware middleware for policy enforcement.

## Architecture Overview

The integration supports two directions:

1. **Middleware wraps Kitaru**: Policy enforcement on every tool call made by Kitaru agents
2. **Kitaru as tools**: Expose Kitaru flows as tools that other agents can call via middleware
3. **Bidirectional**: Both directions working together

## Prerequisites

- Kitaru Go SDK installed (`go get github.com/zenml-io/kitaru-sdk-go`)
- pedro-agentware middleware configured
- Policy YAML file

## Direction 1: Middleware Wraps Kitaru Tool Calls

Wrap Kitaru tool calls with pedro-agentware middleware to enforce policies.

### Setup

```go
package main

import (
    "context"
    "fmt"
    "log"

    kitaru "github.com/zenml-io/kitaru-sdk-go"
    "github.com/soypete/pedro-agentware/middleware"
    "github.com/soypete/pedro-agentware/middleware/types"
)
```

### Create Kitaru Tool Executor

```go
type KitaruToolExecutor struct {
    client    *kitaru.Client
    flowName  string
}

func NewKitaruToolExecutor(client *kitaru.Client, flowName string) *KitaruToolExecutor {
    return &KitaruToolExecutor{
        client:   client,
        flowName: flowName,
    }
}

func (e *KitaruToolExecutor) CallTool(ctx context.Context, name string, args map[string]interface{}) (*types.ToolResult, error) {
    flow := e.client.Flow(e.flowName)

    // Run the Kitaru flow with tool arguments
    exec, err := flow.RunWithWait(ctx, map[string]any{
        "tool_name": name,
        "tool_args": args,
    }, 2*time.Second)

    if err != nil {
        return &types.ToolResult{
            Success: false,
            Error:   err.Error(),
        }, err
    }

    return &types.ToolResult{
        Success:   exec.Status == kitaru.StatusCompleted,
        Content:   fmt.Sprintf("%v", exec.Output),
        Metadata:  map[string]interface{}{
            "execution_id": exec.ID,
            "status":       exec.Status,
        },
    }, nil
}

func (e *KitaruToolExecutor) ListTools() []types.ToolDefinition {
    // Return list of available Kitaru flows as tools
    return []types.ToolDefinition{
        {Name: "research", Description: "Research topic using web search"},
        {Name: "analyze", Description: "Analyze data and generate insights"},
        {Name: "report", Description: "Generate formatted report"},
    }
}
```

### Wrap with Middleware

```go
func main() {
    // Create Kitaru client
    kitaruClient := kitaru.NewClient(
        "http://kitaru-service:8080",
        "api-key",
        "project",
    )

    // Create Kitaru executor
    executor := NewKitaruToolExecutor(kitaruClient, "agent-workflow")

    // Load policy
    policy, err := middleware.LoadPolicyFromFile("policy.yaml")
    if err != nil {
        log.Fatalf("Failed to load policy: %v", err)
    }

    // Create middleware
    mw := middleware.New(executor, *policy)

    // Create caller context
    callerCtx := types.CallerContext{
        Trusted:   true,
        Role:      "user",
        UserID:    "user-123",
        SessionID: "session-456",
        Source:    "kitaru-agent",
    }
    ctx := middleware.WithCallerContext(context.Background(), callerCtx)

    // Call tool through middleware (policy enforced)
    result, err := mw.CallTool(ctx, "research", map[string]interface{}{
        "query": "Go middleware patterns",
    })
    if err != nil {
        log.Fatalf("Tool call failed: %v", err)
    }

    fmt.Printf("Result: %s\n", result.Content)
}
```

### Example Policy (policy.yaml)

```yaml
rules:
  - name: "rate-limit-research"
    tools:
      - "research"
    action: "allow"
    max_rate:
      count: 10
      window: 60

  - name: "deny-destructive"
    tools:
      - "delete_*"
      - "drop_*"
    action: "deny"
    conditions:
      - field: "caller.trusted"
        operator: "eq"
        value: false

  - name: "log-all-calls"
    tools:
      - "*"
    action: "allow"
    audit: true

default_deny: false
```

## Direction 2: Kitaru Flows as Middleware Tools

Expose Kitaru flows as tools that can be called by other agents through the middleware.

### Create Tool Adapter

```go
type KitaruFlowToolExecutor struct {
    client *kitaru.Client
    flows  map[string]string // tool name -> flow name
}

func NewKitaruFlowToolExecutor(client *kitaru.Client) *KitaruFlowToolExecutor {
    return &KitaruFlowToolExecutor{
        client: client,
        flows: map[string]string{
            "kitaru_research": "research-flow",
            "kitaru_analyze":  "analyze-flow",
            "kitaru_report":   "report-flow",
        },
    }
}

func (e *KitaruFlowToolExecutor) CallTool(ctx context.Context, name string, args map[string]interface{}) (*types.ToolResult, error) {
    flowName, ok := e.flows[name]
    if !ok {
        return nil, fmt.Errorf("unknown tool: %s", name)
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
        Success:   exec.Status == kitaru.StatusCompleted,
        Content:   fmt.Sprintf("%v", exec.Output),
        Metadata:  map[string]interface{}{
            "execution_id": exec.ID,
        },
    }, nil
}

func (e *KitaruFlowToolExecutor) ListTools() []types.ToolDefinition {
    tools := make([]types.ToolDefinition, 0, len(e.flows))
    for name, flow := range e.flows {
        tools = append(tools, types.ToolDefinition{
            Name:        name,
            Description: fmt.Sprintf("Execute Kitaru flow: %s", flow),
        })
    }
    return tools
}
```

### Use with Middleware

```go
func main() {
    client := kitaru.NewClient(
        "http://kitaru-service:8080",
        "api-key",
        "project",
    )

    executor := NewKitaruFlowToolExecutor(client)

    policy, _ := middleware.LoadPolicyFromFile("policy.yaml")
    mw := middleware.New(executor, *policy)

    // Other agents can now call Kitaru flows through the middleware
    result, _ := mw.CallTool(context.Background(), "kitaru_research", map[string]interface{}{
        "query": "agent middleware patterns",
    })

    fmt.Printf("Kitaru flow result: %s\n", result.Content)
}
```

## Bidirectional Integration

Full integration where both directions work together.

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Your Agent Application                   │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              pedro-agentware Middleware              │   │
│  │  ┌───────────────────────────────────────────────┐  │   │
│  │  │  Policy Engine                                 │  │   │
│  │  │  - Rate limiting                               │  │   │
│  │  │  - Tool filtering                              │  │   │
│  │  │  - Audit logging                               │  │   │
│  │  └───────────────────────────────────────────────┘  │   │
│  │                       │                              │   │
│  │           ┌───────────┴───────────┐                 │   │
│  │           ▼                       ▼                  │   │
│  │  ┌──────────────────┐   ┌──────────────────────┐   │   │
│  │  │ Kitaru Executor  │   │ External Tools       │   │   │
│  │  │ (wraps Kitaru    │   │ (web, filesystem,    │   │   │
│  │  │  tool calls)     │   │  etc.)               │   │   │
│  │  └────────┬─────────┘   └──────────┬───────────┘   │   │
│  └───────────┼─────────────────────────┼───────────────┘   │
│              │                         │                    │
│              ▼                         ▼                    │
│  ┌─────────────────────┐   ┌──────────────────────────┐   │
│  │   Kitaru Server     │   │  External Tool Services  │   │
│  │   (K8s Deployment)  │   │                          │   │
│  └─────────────────────┘   └──────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Complete Integration Example

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
)

type IntegratedExecutor struct {
    kitaruClient   *kitaru.Client
    externalTools map[string]func(ctx context.Context, args map[string]interface{}) (*types.ToolResult, error)
}

func NewIntegratedExecutor(kitaruURL, apiKey, project string) *IntegratedExecutor {
    client := kitaru.NewClient(kitaruURL, apiKey, project)

    return &IntegratedExecutor{
        kitaruClient: client,
        externalTools: map[string]func(ctx context.Context, args map[string]interface{}) (*types.ToolResult, error){
            "web_search": func(ctx context.Context, args map[string]interface{}) (*types.ToolResult, error) {
                query := args["query"].(string)
                return &types.ToolResult{
                    Success: true,
                    Content: fmt.Sprintf("Search results for: %s", query),
                }, nil
            },
            "save_file": func(ctx context.Context, args map[string]interface{}) (*types.ToolResult, error) {
                return &types.ToolResult{
                    Success: true,
                    Content: "File saved successfully",
                }, nil
            },
        },
    }
}

func (e *IntegratedExecutor) CallTool(ctx context.Context, name string, args map[string]interface{}) (*types.ToolResult, error) {
    // Check external tools first
    if tool, ok := e.externalTools[name]; ok {
        return tool(ctx, args)
    }

    // Fall back to Kitaru
    flow := e.kitaruClient.Flow(name)
    exec, err := flow.RunWithWait(ctx, args, 2*time.Second)
    if err != nil {
        return &types.ToolResult{Success: false, Error: err.Error()}, err
    }

    return &types.ToolResult{
        Success:  exec.Status == kitaru.StatusCompleted,
        Content:  fmt.Sprintf("%v", exec.Output),
        Metadata: map[string]interface{}{"execution_id": exec.ID},
    }, nil
}

func (e *IntegratedExecutor) ListTools() []types.ToolDefinition {
    tools := []types.ToolDefinition{
        {Name: "web_search", Description: "Search the web"},
        {Name: "save_file", Description: "Save content to file"},
        {Name: "research_flow", Description: "Run Kitaru research flow"},
        {Name: "analyze_flow", Description: "Run Kitaru analysis flow"},
    }
    return tools
}

func main() {
    executor := NewIntegratedExecutor(
        "http://kitaru-service:8080",
        "api-key",
        "project",
    )

    policy, err := middleware.LoadPolicyFromFile("policy.yaml")
    if err != nil {
        log.Fatalf("Failed to load policy: %v", err)
    }

    mw := middleware.New(executor, *policy)

    // Create caller context
    ctx := middleware.WithCallerContext(context.Background(), types.CallerContext{
        Trusted:   true,
        Role:      "agent",
        UserID:    "system",
        SessionID: "session-1",
    })

    // Call external tool through middleware
    result, err := mw.CallTool(ctx, "web_search", map[string]interface{}{
        "query": "Go middleware patterns",
    })
    if err != nil {
        log.Printf("Tool denied: %v", err)
    } else {
        fmt.Printf("External tool result: %s\n", result.Content)
    }

    // Call Kitaru flow through middleware
    result, err = mw.CallTool(ctx, "research_flow", map[string]interface{}{
        "topic": "agent middleware",
    })
    if err != nil {
        log.Printf("Kitaru flow denied: %v", err)
    } else {
        fmt.Printf("Kitaru flow result: %s\n", result.Content)
    }
}
```

## Configuration for Kubernetes

### Deploy Middleware as Sidecar

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agent-service
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: agent
        image: your-agent:latest
        env:
        - name: KITARU_URL
          value: "http://kitaru-service:8080"
        - name: KITARU_API_KEY
          valueFrom:
            secretKeyRef:
              name: kitaru-credentials
              key: api-key
      - name: middleware
        image: pedro-agentware:latest
        ports:
        - containerPort: 8081
        env:
        - name: POLICY_PATH
          value: "/config/policy.yaml"
```

### Deploy Middleware as Separate Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: agent-middleware
spec:
  selector:
    app: agent-middleware
  ports:
  - port: 8081
    targetPort: 8081
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agent-middleware
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: middleware
        image: pedro-agentware:latest
        ports:
        - containerPort: 8081
        volumeMounts:
        - name: policy
          mountPath: /config
      volumes:
      - name: policy
        configMap:
          name: agent-policy
```

## Policy Examples

### Rate Limiting Kitaru Flows

```yaml
rules:
  - name: "limit-research-flow"
    tools:
      - "research_flow"
    action: "allow"
    max_rate:
      count: 5
      window: 60

  - name: "limit-analysis-flow"
    tools:
      - "analyze_flow"
    action: "allow"
    max_rate:
      count: 3
      window: 300
```

### Trust-Based Access Control

```yaml
rules:
  - name: "trusted-agents-full-access"
    tools:
      - "*"
    action: "allow"
    conditions:
      - field: "caller.trusted"
        operator: "eq"
        value: true

  - name: "untrusted-read-only"
    tools:
      - "research_flow"
      - "web_search"
    action: "allow"
    conditions:
      - field: "caller.trusted"
        operator: "eq"
        value: false

  - name: "deny-destructive-untrusted"
    tools:
      - "delete_*"
      - "kitaru_*"
    action: "deny"
    conditions:
      - field: "caller.trusted"
        operator: "eq"
        value: false
```

### Audit All Kitaru Calls

```yaml
rules:
  - name: "audit-kitaru-flows"
    tools:
      - "*"
    action: "allow"
    audit: true

  - name: "alert-on-failures"
    tools:
      - "*"
    action: "allow"
    conditions:
      - field: "context.last_error"
        operator: "exists"
    alert: true
```