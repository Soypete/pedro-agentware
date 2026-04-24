# Agent Framework Design Document

## Overview

A Go-based agent building framework providing graph-based workflow orchestration, state management, and tool execution. Inspired by LangGraph and Pydantic AI patterns, designed to integrate with existing Pedro middleware, executor, and tool components.

---

## Motivation

The existing `go/executor` package provides a linear inference loop (send prompt → parse tool calls → execute → repeat). Agent workflows often require:

- **Branching**: Different paths based on state (conditional edges)
- **Parallelism**: Multiple nodes can run concurrently
- **Cycles**: Loops for retry, refinement, or iterative tasks
- **Checkpointing**: Persist state across runs for resumability
- **Interrupts**: Human-in-the-loop approval before dangerous operations
- **Typed state**: Type-safe state management beyond `map[string]any`

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Agent Framework                         │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │  Graph   │→ │  Runner  │→ │  State   │→ │ Checkpointer│  │
│  │ Builder  │  │          │  │  [T]     │  │            │  │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
│       │             │              │              │         │
│       ▼             ▼              ▼              ▼         │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                    Nodes                             │    │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐   │    │
│  │  │ LLM Node│ │Tool Node│ │Cond Node│ │  Custom │   │    │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘   │    │
│  └─────────────────────────────────────────────────────┘    │
│                         │                                    │
│                         ▼                                    │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                  Middleware (existing)              │    │
│  │   Policy | Audit | Context Control | Rate Limiting  │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

---

## Core Interfaces

### State

```go
// State represents the working memory of an agent graph.
type State[T any] struct {
    data T
}

// Get returns the current state value.
func (s *State[T]) Get() T {
    return s.data
}

// Update replaces the state entirely.
func (s *State[T]) Update(data T) {
    s.data = data
}

// UpdateField updates a single field using a reducer.
func (s *State[T]) UpdateField(field string, reducer ReducerFunc[T]) error
```

### Reducers

```go
// ReducerFunc merges new values into existing state.
// Used for accumulating values (e.g., messages += [new message]).
type ReducerFunc[T any] func(existing, incoming T) T

// Common reducers for common patterns.
var Reducers = struct {
    // AppendReducer adds incoming to existing slice.
    AppendReducer[T any] ReducerFunc[[]T]
    // MergeMapReducer merges incoming map into existing.
    MergeMapReducer[K comparable, V any] ReducerFunc[map[K]V]
}{}
```

### Node

```go
// NodeFunc is a step in the agent workflow.
// Returns updated state and the name of the next node to run.
type NodeFunc[T any] func(context.Context, *State[T]) (*State[T], string, error)

// NodeResult contains the output of a node execution.
type NodeResult[T any] struct {
    State     *State[T]
    NextNode  string
    Error     error
    Interrupts []Interrupt
}

// Interrupt pauses execution for human approval.
type Interrupt struct {
    Type      string
    Message   string
    Payload   map[string]any
    ResumeCh  chan any  // Channel to send approval/denial
}
```

### Graph

```go
// Graph defines the structure of an agent workflow.
type Graph[T any] interface {
    // AddNode adds a named node to the graph.
    AddNode(name string, fn NodeFunc[T]) Graph[T]

    // AddEdge adds a directed edge from one node to another.
    AddEdge(from, to string) Graph[T]

    // AddConditionalEdge adds an edge that selects the next node
    // based on the return value of the conditional function.
    AddConditionalEdge(from string, fn ConditionalFunc[T], branches []string) Graph[T]

    // SetEntryPoint sets the first node to execute.
    SetEntryPoint(node string) Graph[T]

    // SetFinishPoint marks a node as terminal (ends the graph).
    SetFinishPoint(node string) Graph[T]

    // Compile builds the executable graph.
    Compile() (Runner[T], error)
}

// ConditionalFunc determines the next node based on current state.
type ConditionalFunc[T any] func(context.Context, *State[T]) (string, error)
```

### Runner

```go
// Runner is a compiled graph ready for execution.
type Runner[T any] interface {
    // Run executes the graph from the entry point.
    Run(ctx context.Context, initialState T) (<-chan StateUpdate[T], error)

    // RunUntil runs the graph until a condition is met.
    RunUntil(ctx context.Context, initialState T, until ConditionFunc[T]) (T, error)

    // GetState retrieves the current state (requires checkpointer).
    GetState(ctx context.Context, threadID string) (*State[T], error)

    // UpdateState updates state for a thread (requires checkpointer).
    UpdateState(ctx context.Context, threadID string, state T) error
}

// StateUpdate is a streaming update from the runner.
type StateUpdate[T any] struct {
    Node    string
    State   *State[T]
    IsDelta bool  // true if this is a partial update (reducer applied)
}
```

### Checkpointer

```go
// Checkpointer persists and restores graph state.
type Checkpointer[T any] interface {
    // Save saves the current state for a thread.
    Save(ctx context.Context, threadID string, state *State[T], node string) error

    // Get retrieves state for a thread.
    Get(ctx context.Context, threadID string) (*State[T], string, error)

    // List lists all thread IDs.
    List(ctx context.Context) ([]string, error)

    // Delete removes state for a thread.
    Delete(ctx context.Context, threadID error) error
}
```

---

## Built-in Nodes

### LLM Node

```go
// LLMNode calls an LLM and updates state with the response.
type LLMNode[T any] struct {
    Backend   llm.Backend
    Prompt    string // or func(*State[T]) string
    Tools     []tools.Tool
    Extractor func(llm.Response) (T, error) // Parse LLM output into state
}

// NewLLMNode creates a node that calls the LLM.
func NewLLMNode[T any](backend llm.Backend, prompt string, extractor func(llm.Response) (T, error)) *LLMNode[T]
```

### Tool Node

```go
// ToolNode executes a tool via middleware.
type ToolNode[T any] struct {
    ToolName string
    Middleware middleware.ToolExecutor
    // InputMapper extracts tool args from state.
    InputMapper func(*State[T]) (map[string]any, error)
    // OutputMapper merges tool result into state.
    OutputMapper func(*State[T], *tools.Result) error
}

// NewToolNode creates a node that executes a tool.
func NewToolNode[T any](middleware middleware.ToolExecutor, toolName string) *ToolNode[T]
```

### Conditional Node

```go
// ConditionalNode selects the next node based on state.
type ConditionalNode[T any] struct {
    Condition func(*State[T]) (string, error)
    Branches  map[string]string // condition value -> node name
}

// NewConditionalNode creates a routing node.
func NewConditionalNode[T any](condition func(*State[T]) (string, error)) *ConditionalNode[T]
```

---

## Directory Structure

```
go/agent/
├── go.mod
├── state.go           # State[T] with reducers
├── node.go            # NodeFunc, NodeResult, Interrupt
├── graph.go           # Graph builder implementation
├── runner.go          # Compiled runner implementation
├── checkpointer.go    # Checkpointer interface
├── checkpointer/
│   └── memory.go      # In-memory checkpointer implementation
├── nodes/
│   ├── doc.go
│   ├── llm.go         # LLM node
│   └── tool.go        # Tool node
├── edges/
│   ├── doc.go
│   └── conditional.go # Conditional edge helpers
└── examples/
    ├── doc.go
    └── simple.go      # Basic graph example
```

---

## Integration with Existing Components

### Middleware Integration

Tool nodes use the existing middleware for policy enforcement:

```go
toolNode := nodes.NewToolNode[T](middlewareInstance, "web_search")
```

The middleware is already wired in `executor/dispatcher.go` - agents can use that pattern.

### LLM Integration

LLM nodes use the existing `llm.Backend` interface from `go/llm/backend.go`:

```go
llmNode := nodes.NewLLMNode(backend, prompt, func(resp llm.Response) (T, error) {
    // Parse response
})
```

### Tool Integration

Tools are registered in `tools.ToolRegistry` - agents can either:
1. Pass the registry to the agent for dynamic tool discovery
2. Pre-filter tools using middleware policy

---

## Usage Examples

### Simple Sequential Agent

```go
// Define state
type AgentState struct {
    Task      string   `json:"task"`
    Messages  []string `json:"messages"`
    Result    string   `json:"result"`
}

// Reducers
func addMessage(existing, incoming []string) []string {
    return append(existing, incoming...)
}

// Build graph
g := agent.NewGraph[AgentState]().
    AddNode("ask_llm", func(ctx context.Context, state *agent.State[AgentState]) (*agent.State[AgentState], string, error) {
        resp, err := llm.Call(ctx, state.Get().Task)
        state.UpdateField("messages", func(s AgentState) AgentState {
            s.Messages = append(s.Messages, resp)
            return s
        })
        return state, "respond", nil
    }).
    AddNode("respond", func(ctx context.Context, state *agent.State[AgentState]) (*agent.State[AgentState], string, error) {
        // Process response
        return state, agent.END, nil
    }).
    SetEntryPoint("ask_llm").
    SetFinishPoint("respond")

runner, _ := g.Compile()
result, _ := runner.RunUntil(ctx, AgentState{Task: "Hello"}, func(s *agent.State[AgentState]) bool {
    return s.Get().Result != ""
})
```

### Graph with Conditional Branching

```go
g := agent.NewGraph[AgentState]().
    AddNode("classify", func(ctx context.Context, state *agent.State[AgentState]) (*agent.State[AgentState], string, error) {
        intent := classifyIntent(state.Get().Task)
        return state, intent, nil
    }).
    AddNode("search", searchNode).
    AddNode("calculate", calcNode).
    AddNode("respond", respondNode).
    AddConditionalEdge("classify", func(ctx context.Context, state *agent.State[AgentState]) (string, error) {
        return state.Get().Intent, nil
    }, []string{"search", "calculate"}).
    AddEdge("search", "respond").
    AddEdge("calculate", "respond").
    SetFinishPoint("respond")
```

### Tool-Enabled Agent with Middleware

```go
// Create middleware-wrapped tool executor
registry := tools.NewRegistry()
registry.Register(&WebSearchTool{})
registry.Register(&FileReadTool{})

exec := tools.NewRegistryExecutor(registry)
mw := middleware.NewMiddleware(exec)
mw = mw.WithPolicy(policyEvaluator)
mw = mw.WithAuditor(auditor)

// Build graph with tool nodes
g := agent.NewGraph[AgentState]().
    AddNode("think", llmNode).
    AddNode("search", nodes.NewToolNode[AgentState](mw, "web_search")).
    AddNode("read", nodes.NewToolNode[AgentState](mw, "file_read")).
    AddEdge("think", "search").
    SetEntryPoint("think").
    SetFinishPoint("search")
```

### Checkpointed Agent (Resumable)

```go
// Create runner with checkpointer
runner, _ := g.Compile(agent.WithCheckpointer(checkpointer))

// Start execution (saves state to thread-1)
updates, _ := runner.Run(ctx, AgentState{Task: "research"})

// Later, resume from where we left off
state, _ := runner.GetState(ctx, "thread-1")
// ... continue execution
```

---

## Design Decisions

### Why Generics for State?

LangGraph uses `map[string]any` which loses type safety. Using Go generics:

```go
type State[MyState any] struct {
    data MyState
}
```

Benefits:
- Compile-time type checking
- IDE autocomplete
- Refactoring safety

### Why Interface-Based Runner?

Allows for different execution strategies:
- `LinearRunner` - Sequential node execution
- `ParallelRunner` - Concurrent node execution
- `AsyncRunner` - Streaming with async updates

### Why Checkpointer Interface?

Enables different persistence strategies:
- `MemoryCheckpointer` - In-memory (dev/fast iteration)
- `PostgresCheckpointer` - Production persistence
- `RedisCheckpointer` - Distributed deployments

### Why Interrupt Support?

LangGraph's killer feature is human-in-the-loop. Without it:
- Dangerous operations can't be approved
- Production agents lack governance
- No way to correct agent mistakes

---

## Existing Patterns in PedroCLI

The `pedro-agentware/go` module already has components that can be leveraged or adapted. Additionally, PedroCLI has a mature agent harness system in `pkg/agents/`:

### What PedroCLI Has (Extract/Adapt)

| Component | Location | What It Does | Framework Opportunity |
|-----------|----------|-------------|----------------------|
| **BaseAgent** | `pkg/agents/base.go` | Common agent base with tools, registry, system prompts | Extract as `agent.Agent` base type |
| **InferenceExecutor** | `pkg/agents/executor.go` | Single-phase inference loop with tool calls | Adapt as `LinearRunner` |
| **PhasedExecutor** | `pkg/agents/phased_executor.go` | Multi-phase workflows with per-phase tool subsets, validators | This IS a graph-like pattern! |
| **Phase** | `pkg/agents/phased_executor.go:23` | `Name`, `SystemPrompt`, `Tools[]`, `MaxRounds`, `Validator` | Already a node definition |
| **ProgressCallback** | `pkg/agents/executor.go:42` | Event streaming during execution | Use for runner state updates |
| **SubagentManager** | `pkg/orchestration/` | Spawns parallel subagents | Supports parallel node execution |
| **ArtifactStore** | `pkg/artifacts/` | Passes data between phases | Used for state passing between nodes |
| **JobManager** | `pkg/jobs/` | Async job execution with state | Checkpointer can use this |
| **ModeConstraints** | `pkg/agents/mode_constraints.go` | Per-mode tool restrictions | Policy-based tool filtering |
| **ContextManager** | `pkg/llmcontext/` | Conversation history management | State/history persistence |

### What's Missing from PedroCLI

1. **Generic State** - Currently uses `map[string]interface{}`; need typed `State[T]`
2. **Explicit Graph** - PhasedExecutor is sequential, not a DAG; need graph builder
3. **Conditional Edges** - Phases are linear; need router nodes
4. **State Reducers** - Artifacts pass data but no merge/reduce logic
5. **Checkpointer** - Jobs are persisted but not graph state checkpoints
6. **Interrupts** - No human-in-the-loop support

### What To Extract

```go
// From PedroCLI's agents package → go/agent

// 1. Agent interface (already exists in base.go)
// Current: Agent interface with Execute()
// New: Add state type parameter

// 2. PhasedExecutor → Graph Runner
// Current: Sequential phases with tool subsets
// New: Graph with nodes + edges

// 3. Phase → Node
// Current: Phase{ Name, SystemPrompt, Tools, MaxRounds, Validator }
// New: NodeFunc[T] with config

// 4. ProgressCallback → StateUpdate channel
// Current: ProgressEvent{ Type, Message, Data }
// New: StateUpdate[T]{ Node, State, IsDelta }
```

## Comparison to LangGraph

| Feature | LangGraph (Python) | This Framework (Go) | PedroCLI Has |
|---------|-------------------|---------------------|--------------|
| State | `dict` (map) | `State[T]` generic | `map[string]interface{}` (partial) |
| Nodes | Functions | `NodeFunc[T]` | `Phase` (sequential only) |
| Edges | Strings | Strings + Conditionals | Sequential phases |
| Checkpointer | Optional | Pluggable interface | JobManager (partial) |
| Interrupts | Yes | Yes | No |
| Time Travel | Yes | Via checkpointer | No |
| Streaming | Yes | Via channel | ProgressCallback |
| Subagents | Yes | Via SubagentSpawner | SubagentManager |
| Phases | Yes | Via Graph | PhasedExecutor |

---

## Unknowns / Future Exploration

1. **Streaming format** - LangGraph streams state deltas; need to define delta format
2. **Subgraphs** - Nested graphs for reusability
3. **Parallel branches** - `add_parallel` for concurrent node execution
4. **Retry policies** - Node-level retry configuration
5. **Timeout handling** - Per-node and global timeouts
6. **Observability** - Tracing integration for debugging
7. **MCP integration** - Graph exposed as MCP server for Python/TS clients

---

## Implementation Strategy

Given the existing PedroCLI patterns, the implementation should be incremental:

### Phase 1: Core Abstractions (Minimal)

Focus on the graph model itself, reusing existing execution patterns:

```go
go/agent/
├── state.go           // State[T] with reducers
├── node.go            // NodeFunc[T], NodeResult
├── graph.go           // Graph builder
└── runner.go          // Simple sequential runner
```

Key decisions:
- **Don't rewrite InferenceExecutor** - Adapt it as the default runner
- **Keep PhasedExecutor** - It's already a valid workflow pattern, just add graph support
- **Add state typing** - Wrap `map[string]interface{}` with generics

### Phase 2: Adapters for PedroCLI

Create adapters to use the framework within PedroCLI:

```go
go/agent/
├── adapters/
│   ├── executor.go    // Wrap InferenceExecutor as Runner
│   └── phased.go      // Wrap PhasedExecutor as Graph
```

### Phase 3: Graph Features

Add what's missing from LangGraph:
- Conditional edges
- Checkpointer
- Interrupts

---

## Implementation Order

1. **Phase 1**: Core types (State, Node, Graph, Runner)
2. **Phase 2**: Built-in nodes (LLM, Tool)
3. **Phase 3**: Checkpointer (Memory first)
4. **Phase 4**: Conditional edges
5. **Phase 5**: Interrupts
6. **Phase 6**: Streaming
7. **Phase 7**: Persistence checkpointer (Postgres/Redis)

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Generics complexity | Hard to use | Clear examples, typed wrapper helpers |
| Performance overhead | Slower than direct code | Benchmark, allow bypassing graph for hot paths |
| State serialization | Hard to persist | Support both typed and `map[string]any` modes |
| Over-engineering | Feature bloat | Start with minimal viable graph |