package main

import (
	"context"
	"fmt"
	"log"
	"time"

	kitaruadapter "github.com/soypete/pedro-agentware/go/adapters/kitaru"
	"github.com/soypete/pedro-agentware/go/middleware"
	"github.com/soypete/pedro-agentware/go/tools"
)

func main() {
	ctx := context.Background()

	client := kitaruadapter.NewHTTPClient(
		"http://localhost:8080",
		"your-api-key",
		"your-project",
	)

	flowMapping := map[string]string{
		"research_flow": "research-agent",
		"analyze_flow":  "data-analyzer",
		"process_data":  "data-processor",
	}

	executor := kitaruadapter.NewFlowToolExecutor(client, flowMapping)

	policy := &middleware.Policy{
		Rules: []middleware.Rule{
			{
				Name:   "rate-limit-flows",
				Tools:  []string{"*"},
				Action: middleware.ActionAllow,
				MaxRate: &middleware.RateLimit{
					Count:  10,
					Window: 60 * time.Second,
				},
			},
			{
				Name:   "deny-destructive",
				Tools:  []string{"delete_*", "drop_*"},
				Action: middleware.ActionDeny,
				Conditions: []middleware.Condition{
					{
						Field:    "caller.trusted",
						Operator: middleware.OperatorEq,
						Value:    "false",
					},
				},
			},
		},
		DefaultDeny: false,
	}

	mw := middleware.NewMiddleware(executor).WithPolicy(policy)

	callerCtx := middleware.CallerContext{
		Trusted:   true,
		Role:      "user",
		UserID:    "user-123",
		SessionID: "session-456",
	}
	ctx = kitaruadapter.WithCallerContext(ctx, callerCtx)

	result, err := mw.Execute(ctx, "research_flow", map[string]any{
		"topic": "Go middleware patterns",
	})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if result.Success {
		fmt.Printf("Success: %s\n", result.Output)
	} else {
		fmt.Printf("Failed: %s\n", result.Error)
	}

	toolsList := mw.(interface {
		ListTools() []tools.Tool
	}).ListTools()
	fmt.Printf("Available tools: %d\n", len(toolsList))
	for _, t := range toolsList {
		fmt.Printf("  - %s: %s\n", t.Name(), t.Description())
	}
}
