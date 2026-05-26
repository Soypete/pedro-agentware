# Basic Usage

This guide covers core Kitaru Go SDK operations: flows, checkpoints, artifacts, and logging.

## Creating a Client

```go
import kitaru "github.com/zenml-io/kitaru-sdk-go"

// From environment variables
client := kitaru.MustNewClientFromEnv()

// Or explicit configuration
client := kitaru.NewClient(
    "http://kitaru-service:8080",
    "your-api-key",
    "your-project",
)
```

## Running Flows

### Basic Flow Execution

```go
ctx := context.Background()

// Create a flow reference
flow := client.Flow("my-agent-flow")

// Run with inputs
execID, err := flow.Run(ctx, map[string]any{
    "task":        "analyze user request",
    "user_id":     "user-123",
    "session_id":  "session-456",
})
if err != nil {
    log.Fatalf("Failed to start flow: %v", err)
}

fmt.Printf("Execution started: %s\n", execID)
```

### Running with Wait

For flows that complete quickly, use `RunWithWait`:

```go
flow := client.Flow("quick-flow")
exec, err := flow.RunWithWait(ctx, map[string]any{
    "input": "data",
}, 2 * time.Second)  // Poll interval
if err != nil {
    log.Fatalf("Flow failed: %v", err)
}

fmt.Printf("Status: %s, Output: %v\n", exec.Status, exec.Output)
```

## Checkpoints

Checkpoints persist state and enable replay from failure points.

### Saving Checkpoints

```go
// After completing a step, save a checkpoint
err := flow.Checkpoint("analysis_complete", map[string]any{
    "files_analyzed":  10,
    "summary":         "Analyzed codebase structure",
    "duration_ms":     1500,
})
if err != nil {
    log.Printf("Warning: Failed to save checkpoint: %v", err)
}
```

### Restoring Checkpoints

```go
// Restore previous checkpoint data
checkpointData, err := flow.RestoreCheckpoint("analysis_complete")
if err != nil {
    log.Printf("Warning: Failed to restore checkpoint: %v", err)
} else {
    filesAnalyzed := checkpointData["files_analyzed"].(int)
    fmt.Printf("Restored: %d files analyzed\n", filesAnalyzed)
}
```

### Replay from Checkpoint

If a flow fails, replay from a checkpoint:

```go
exec, err := flow.GetExecution()
if err != nil {
    log.Fatalf("Failed to get execution: %v", err)
}

if exec.Status == kitaru.StatusFailed {
    // Replay from the last successful checkpoint
    newExecID, err := flow.Replay("analysis_complete", map[string]any{
        "input": "new data",  // Optional: override inputs
    })
    if err != nil {
        log.Printf("Failed to replay: %v", err)
    } else {
        fmt.Printf("Replayed from checkpoint, new execution: %s\n", newExecID)
    }
}
```

## Artifacts

Store and retrieve persistent data across executions.

### Saving Artifacts

```go
// Save any serializable data
err := flow.SaveArtifact("analysis_report", map[string]any{
    "title":       "Code Analysis",
    "findings":    []string{"issue-1", "issue-2"},
    "severity":    "medium",
}, "application/json")
if err != nil {
    log.Printf("Warning: Failed to save artifact: %v", err)
}
```

### Loading Artifacts

```go
// Load artifact into struct
var report struct {
    Title     string   `json:"title"`
    Findings  []string `json:"findings"`
    Severity  string   `json:"severity"`
}

err := flow.LoadArtifact("analysis_report", &report)
if err != nil {
    log.Printf("Warning: Failed to load artifact: %v", err)
} else {
    fmt.Printf("Report: %s (%s)\n", report.Title, report.Severity)
}
```

## Logging

Add structured logs for observability.

```go
// Info level
err := flow.Log(kitaru.LogLevelInfo, "Completed analysis phase", map[string]any{
    "phase":       "analysis",
    "duration_ms": 1500,
})

// Debug level
err := flow.Log(kitaru.LogLevelDebug, "Processing file", map[string]any{
    "file":  "main.go",
    "lines": 150,
})

// Error level
err := flow.Log(kitaru.LogLevelError, "API call failed", map[string]any{
    "endpoint": "/api/analyze",
    "status":   500,
})
```

## Execution Management

### Get Execution Status

```go
exec, err := flow.GetExecution()
if err != nil {
    log.Fatalf("Failed to get execution: %v", err)
}

fmt.Printf("Status: %s\n", exec.Status)
fmt.Printf("Started: %s\n", exec.StartedAt)
fmt.Printf("Updated: %s\n", exec.UpdatedAt)
fmt.Printf("Output: %v\n", exec.Output)
```

### Execution States

| Status | Description |
|--------|-------------|
| `pending` | Flow queued, not started |
| `running` | Currently executing |
| `waiting` | Paused (e.g., human input) |
| `completed` | Successfully finished |
| `failed` | Failed with error |

### Wait for Completion

```go
// Poll for completion
exec, err := flow.RunWithWait(ctx, inputs, 2*time.Second)
if err != nil {
    // Handle error
}

// Or manual polling
execID, _ := flow.Run(ctx, inputs)
for {
    exec, _ := flow.GetExecution()
    if exec.Status == kitaru.StatusCompleted || exec.Status == kitaru.StatusFailed {
        break
    }
    time.Sleep(2 * time.Second)
}
```

## Complete Example

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

    flow := client.Flow("data-processing-agent")

    // Run flow
    execID, err := flow.Run(ctx, map[string]any{
        "input_file": "data.csv",
    })
    if err != nil {
        log.Fatalf("Failed to start flow: %v", err)
    }
    fmt.Printf("Started: %s\n", execID)

    // Checkpoint after validation
    flow.Checkpoint("validation_done", map[string]any{
        "rows_validated": 1000,
        "invalid_rows":   5,
    })

    // Log progress
    flow.Log(kitaru.LogLevelInfo, "Processing records", map[string]any{
        "processed": 500,
        "total":     1000,
    })

    // Save artifact
    flow.SaveArtifact("results", map[string]any{
        "total_processed": 1000,
        "success_rate":    0.995,
    }, "application/json")

    // Wait for completion
    exec, err := flow.RunWithWait(ctx, nil, 2*time.Second)
    if err != nil {
        log.Fatalf("Flow failed: %v", err)
    }

    fmt.Printf("Final status: %s\n", exec.Status)

    // Handle failure with replay
    if exec.Status == kitaru.StatusFailed {
        newID, err := flow.Replay("validation_done", nil)
        if err != nil {
            log.Printf("Replay failed: %v", err)
        } else {
            fmt.Printf("Replayed from checkpoint: %s\n", newID)
        }
    }
}
```

## Best Practices

1. **Save checkpoints after each major step** - Enables efficient replay on failure
2. **Use structured metadata in logs** - Makes debugging easier
3. **Save artifacts for important outputs** - Preserves data across executions
4. **Set appropriate poll intervals** - 1-5 seconds is usually good
5. **Handle errors gracefully** - Checkpoint/log failures shouldn't crash flows