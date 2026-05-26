# Simple Flow Example

This example demonstrates basic Kitaru flow execution using the Go SDK.

## Prerequisites

- Go 1.21+
- Kitaru server running (local or K8s)
- API key and project configured

## Code

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    kitaru "github.com/zenml-io/kitaru-sdk-go"
)

func main() {
    // Create client - uses KITARU_URL, KITARU_API_KEY, KITARU_PROJECT env vars
    client := kitaru.MustNewClientFromEnv()

    ctx := context.Background()

    // Create a flow reference
    flow := client.Flow("hello-world")

    // Run the flow with inputs
    execID, err := flow.Run(ctx, map[string]any{
        "name": "World",
    })
    if err != nil {
        log.Fatalf("Failed to start flow: %v", err)
    }

    fmt.Printf("Flow started with execution ID: %s\n", execID)

    // Wait for completion
    exec, err := flow.RunWithWait(ctx, nil, 2*time.Second)
    if err != nil {
        log.Fatalf("Flow failed: %v", err)
    }

    fmt.Printf("Flow completed with status: %s\n", exec.Status)
    fmt.Printf("Output: %v\n", exec.Output)
}
```

## Expected Output

```
Flow started with execution ID: exec-abc123
Flow completed with status: completed
Output: map[message:Hello, World!]
```

## Environment Variables

Set these before running:

```bash
export KITARU_URL="http://localhost:8080"
export KITARU_API_KEY="your-api-key"
export KITARU_PROJECT="your-project"
```

Or use explicit configuration:

```go
client := kitaru.NewClient(
    "http://kitaru-service.default.svc.cluster.local:8080",
    "your-api-key",
    "your-project",
)
```

## Running the Example

```bash
go run main.go
```

## With Checkpoints and Artifacts

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    kitaru "github.com/zenml-io/kitaru-sdk-go"
)

func main() {
    client := kitaru.MustNewClientFromEnv()
    ctx := context.Background()

    flow := client.Flow("data-pipeline")

    // Start flow
    execID, _ := flow.Run(ctx, map[string]any{
        "input_file": "data.csv",
    })
    fmt.Printf("Started: %s\n", execID)

    // Save checkpoint after each stage
    flow.Checkpoint("fetch_complete", map[string]any{
        "rows_fetched": 1000,
    })

    flow.Log(kitaru.LogLevelInfo, "Processing data", map[string]any{
        "stage": "transform",
    })

    flow.Checkpoint("transform_complete", map[string]any{
        "rows_transformed": 1000,
        "duration_ms":      500,
    })

    // Save final output as artifact
    flow.SaveArtifact("results", map[string]any{
        "total_rows":    1000,
        "success_count": 995,
        "error_count":   5,
    }, "application/json")

    // Wait for completion
    exec, _ := flow.RunWithWait(ctx, nil, 2*time.Second)

    fmt.Printf("Final status: %s\n", exec.Status)

    // Load and display artifact
    var results map[string]any
    flow.LoadArtifact("results", &results)
    fmt.Printf("Results: %v\n", results)
}
```

## Kubernetes Deployment

Deploy your application to K8s:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kitaru-client-example
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kitaru-client
  template:
    metadata:
      labels:
        app: kitaru-client
    spec:
      containers:
      - name: app
        image: your-app:latest
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
```