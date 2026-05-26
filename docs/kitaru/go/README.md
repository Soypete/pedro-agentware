# Kitaru Go SDK Integration

Documentation for using the Kitaru Go SDK (`kitaru-go`) with pedro-agentware middleware.

## Overview

Kitaru is a durable execution runtime for AI agents. This guide covers integrating the Go SDK (`kitaru-go`) with pedro-agentware when Kitaru is deployed in Kubernetes.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                           │
│                                                                 │
│  ┌─────────────────┐     ┌───────────────────────────────┐    │
│  │  Kitaru Server  │     │   pedro-agentware Middleware  │    │
│  │   (Deployment)  │     │      (Deployment/Sidecar)     │    │
│  │                 │◄────│                               │    │
│  │   Port: 8080    │     │   Policy Enforcement          │    │
│  │   /api/v1/*     │     │   - Rate limiting             │    │
│  └────────┬────────┘     │   - Tool filtering            │    │
│           │              │   - Audit logging             │    │
│           ▼              └───────────────┬───────────────┘    │
│  ┌─────────────────┐                     │                     │
│  │   Go Client     │◄────────────────────┘                     │
│  │  (kitaru-go)    │                                           │
│  │                 │     ┌───────────────────────────────┐    │
│  └─────────────────┘     │   Your Agent Application      │    │
│                          │   - Uses kitaru-go SDK        │    │
│                          │   - Calls middleware          │    │
│                          └───────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

## Contents

- [Installation](./installation.md) - Installing kitaru-go SDK
- [Connecting to K8s](./connecting.md) - Configuring K8s-deployed Kitaru
- [Basic Usage](./basic-usage.md) - Core flow, checkpoint, artifact operations
- [Middleware Integration](./middleware-integration.md) - pedro-agentware integration
  - [Policy Enforcement](./middleware-integration.md#direction-1-middleware-wraps-kitaru-tool-calls) - Middleware wraps Kitaru
  - [Kitaru as Tools](./middleware-integration.md#direction-2-kitaru-flows-as-middleware-tools) - Kitaru flows exposed as tools
  - [Bidirectional](./middleware-integration.md#bidirectional-integration) - Full bidirectional integration
- [Examples](./examples/) - Code examples
  - [Simple Flow](./examples/simple-flow.md) - Basic flow execution
  - [Policy-Enforced Agent](./examples/policy-enforced.md) - Middleware + Kitaru integration
- [Troubleshooting](./troubleshooting.md) - Common issues and solutions

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    kitaru "github.com/zenml-io/kitaru-sdk-go"
)

func main() {
    // Connect to K8s-deployed Kitaru
    client := kitaru.NewClient(
        "http://kitaru-service.namespace.svc.cluster.local:8080",
        "your-api-key",
        "your-project",
    )

    ctx := context.Background()

    // Run a flow
    flow := client.Flow("my-agent-flow")
    execID, err := flow.Run(ctx, map[string]any{
        "task": "process user request",
    })
    if err != nil {
        log.Fatalf("Failed to start flow: %v", err)
    }

    fmt.Printf("Flow started: %s\n", execID)

    // Save checkpoint
    flow.Checkpoint("analysis_complete", map[string]any{
        "files_processed": 10,
    })

    // Check execution status
    exec, _ := flow.GetExecution()
    fmt.Printf("Status: %s\n", exec.Status)
}
```

## Prerequisites

- Go 1.21+
- Access to a Kitaru deployment in Kubernetes
- pedro-agentware middleware (optional, for policy enforcement)