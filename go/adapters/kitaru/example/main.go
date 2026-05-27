package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"

	kitaruadapter "github.com/soypete/pedro-agentware/go/adapters/kitaru"
	"github.com/soypete/pedro-agentware/go/middleware"
	"github.com/soypete/pedro-agentware/go/tools"
)

func deployFlow(flowName string) error {
	cmd := exec.Command("kitaru", "deploy", flowName)
	cmd.Dir = "/path/to/your/python/project"
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("deploy failed: %v\n%s", err, output)
	}
	log.Printf("Deployed %s: %s", flowName, output)
	return nil
}

func main() {
	ctx := context.Background()

	client := kitaruadapter.NewClient(
		"http://localhost:8080",
		"",
		"default",
		kitaruadapter.WithUsernamePassword("admin", "admin"),
	)

	if err := client.Login(); err != nil {
		log.Fatalf("Login failed: %v", err)
	}

	snapshots, err := client.ListSnapshots("my-flow")
	if err != nil {
		log.Fatalf("Failed to list snapshots: %v", err)
	}

	if len(snapshots) == 0 {
		log.Println("No deployed flows found. Deploy a flow first:")
		log.Println("  kitaru deploy my-flow --inputs topic=your-topic")
		log.Println("")
		log.Println("Or from Python:")
		log.Println("  from my_flow_module import my_flow")
		log.Println("  my_flow.deploy()")
		return
	}

	log.Printf("Found %d deployed snapshot(s): %s", len(snapshots), snapshots[0].ID)

	flowMapping := map[string]string{
		"research_flow": "my-flow",
		"analyze_flow":  "my-flow",
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
