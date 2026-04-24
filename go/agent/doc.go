// Package agent provides a graph-based agent framework for building
// workflow-driven AI agents.
//
// # Core Concepts
//
// State: Generic state management with optional reducers for accumulating data.
//
//	state := agent.NewState(MyState{Task: "hello"})
//	state.Update(func(s MyState) MyState { s.Messages = append(s.Messages, "hi"); return s })
//
// Node: A step in the agent workflow. Returns updated state and the next node to run.
//
//	type NodeFunc[T any] func(context.Context, *State[T]) (*State[T], string, error)
//
// Graph: A directed graph of nodes with edges connecting them.
//
//	g := agent.NewGraph[MyState]().
//		AddNode("think", thinkNode).
//		AddNode("act", actNode).
//		AddEdge("think", "act").
//		SetEntryPoint("think").
//		SetFinishPoint("act")
//
// Runner: Compiled graph ready for execution.
//
//	runner, _ := g.Compile()
//	result, _ := runner.Run(ctx, initialState)
//
// # Built-in Nodes
//
// The nodes subpackage provides common node implementations:
//
//   - LLMNode: Calls an LLM and updates state with the response
//   - ToolNode: Executes a tool via middleware and merges result into state
//
// # Checkpointing
//
// Use a checkpointer to persist state across runs for resumability:
//
//	checkpointer := checkpointer.NewMemory[MyState]()
//	runner, _ := g.Compile(agent.WithCheckpointer(checkpointer))
//
// # Example
//
// See examples/simple.go for a complete usage example.
package agent
