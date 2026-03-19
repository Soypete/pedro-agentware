# Agent Middleware - Engineering Design Document

## Overview

MCP-compatible agent middleware that enforces contracts on context, tools, and data. Designed as a shared Go module that integrates into both `professor_pedro` and `PedroCLI`, with future integration into `iam_pedro`.

The middleware sits between the agent (LLM orchestrator) and tool execution, intercepting every tool call to enforce policies before allowing execution.

---

## Architecture

```
┌─────────────────────────────────────┐
│           Agent / LLM               │
│  (professor_pedro orchestrator,     │
│   PedroCLI inference loop,          │
│   iam_pedro discord bot)            │
└──────────────┬──────────────────────┘
               │ Tool Call Request
               ▼
┌─────────────────────────────────────┐
│        Agent Middleware              │
│  ┌───────────────────────────────┐  │
│  │  Context Control              │  │
│  │  - trusted/untrusted tagging  │  │
│  │  - input sanitization         │  │
│  └───────────────────────────────┘  │
│  ┌───────────────────────────────┐  │
│  │  Tool Control                 │  │
│  │  - permission check           │  │
│  │  - argument validation        │  │
│  │  - rate limiting              │  │
│  └───────────────────────────────┘  │
│  ┌───────────────────────────────┐  │
│  │  Data Control                 │  │
│  │  - response filtering         │  │
│  │  - field redaction            │  │
│  └───────────────────────────────┘  │
│  ┌───────────────────────────────┐  │
│  │  Audit / Observability        │  │
│  │  - violation log              │  │
│  │  - decision trace             │  │
│  └───────────────────────────────┘  │
└──────────────┬──────────────────────┘
               │ Allowed / Blocked
               ▼
┌─────────────────────────────────────┐
│        Tool Execution               │
│  (MCP server, direct bridge,        │
│   API call, shell command)          │
└─────────────────────────────────────┘
```

---

## Core Interface

The middleware wraps any tool execution backend via a single interface:

```go
package middleware

// ToolExecutor is the interface the middleware wraps.
// Matches PedroCLI's ToolBridge pattern.
type ToolExecutor interface {
    CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error)
    ListTools() []ToolDefinition
}

// Middleware is the enforcement layer.
type Middleware struct {
    executor ToolExecutor
    policy   Policy
    auditor  Auditor
}

// ValidateToolCall checks a tool call without executing it.
// Inspired by PedroCLI's validateToolCalls() pattern — catches invalid
// tool names and parameter errors BEFORE execution.
func (m *Middleware) ValidateToolCall(ctx context.Context, name string, args map[string]interface{}) (*Decision, error) {
    callerCtx := m.extractCallerContext(ctx)

    // Check tool exists in catalog
    if _, ok := m.catalog.Get(name); !ok {
        return &Decision{Action: Deny, Reason: fmt.Sprintf("unknown tool: %s", name)}, nil
    }

    // Evaluate policy
    decision := m.policy.Evaluate(callerCtx, name, args)
    m.auditor.Record(decision)
    return &decision, nil
}

// CallTool intercepts every tool call.
func (m *Middleware) CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error) {
    // 1. Context control - check caller identity, tag trusted/untrusted
    callerCtx := m.extractCallerContext(ctx)

    // 2. Tool control - check permissions, validate args
    decision := m.policy.Evaluate(callerCtx, name, args)
    m.auditor.Record(decision)

    if decision.Action == Deny {
        return &ToolResult{Error: decision.Reason}, nil
    }

    // 3. Execute
    result, err := m.executor.CallTool(ctx, name, args)
    if err != nil {
        return nil, err
    }

    // 4. Data control - filter response
    filtered := m.policy.FilterResult(callerCtx, name, result)
    return filtered, nil
}
```

The middleware itself implements `ToolExecutor`, so it's a drop-in replacement anywhere a `ToolExecutor` is used.

---

## Policy System

```go
// Policy defines what's allowed.
type Policy struct {
    Rules []Rule `yaml:"rules" json:"rules"`
}

// Rule is a single constraint.
type Rule struct {
    Name          string            `yaml:"name"`
    Tools         []string          `yaml:"tools"`           // tool names or "*"
    Action        Action            `yaml:"action"`          // allow, deny, filter
    Conditions    []Condition       `yaml:"conditions"`      // when to apply
    MaxRate       *RateLimit        `yaml:"max_rate"`        // optional rate limit
    MaxTurns      *int              `yaml:"max_turns"`       // max tool calls per session (inspired by professor_pedro's maxTurnsPerSession)
    IterationLimit *int             `yaml:"iteration_limit"` // max iterations per workflow (inspired by professor_pedro's maxWorkflowIterations)
    QuickFilter   *QuickFilter     `yaml:"quick_filter"`    // pre-evaluation heuristic skip (inspired by iam_pedro's quick filter)
    ArgSchema     map[string]Schema `yaml:"arg_schema"`      // argument constraints
    RedactFields  []string          `yaml:"redact_fields"`   // fields to strip from results
}

// QuickFilter defines heuristic checks that skip full policy evaluation.
// Inspired by iam_pedro's pre-LLM filtering that skips ~90% of messages.
type QuickFilter struct {
    SkipWhen []Condition `yaml:"skip_when"` // skip full evaluation when ALL conditions match
}

type Condition struct {
    Field    string `yaml:"field"`    // "caller.role", "args.path", "context.trusted"
    Operator string `yaml:"operator"` // "eq", "contains", "matches", "not"
    Value    string `yaml:"value"`
}
```

Example policy file:

```yaml
# agent-policy.yaml
rules:
  - name: block-dangerous-bash
    tools: ["bash"]
    action: deny
    conditions:
      - field: "args.command"
        operator: "matches"
        value: "rm -rf|DROP TABLE|curl.*\\|.*sh"

  - name: restrict-file-access
    tools: ["file", "code_edit"]
    action: allow
    conditions:
      - field: "args.path"
        operator: "not_matches"
        value: ".*\\.(env|key|pem|secret)$"

  - name: rate-limit-web-search
    tools: ["web_search", "webscraper"]
    action: allow
    max_rate:
      count: 10
      window: "1m"

  - name: redact-credentials
    tools: ["*"]
    action: filter
    redact_fields: ["password", "api_key", "token", "secret"]
```

---

## Audit System

Modeled after iam_pedro's production `ModAction` table, which logs every moderation decision with reasoning, tool call, params, and success/failure.

```go
type Auditor interface {
    Record(decision Decision)
    GetViolations(since time.Time) []Decision
}

type Decision struct {
    Timestamp  time.Time
    Tool       string
    Args       map[string]interface{}
    Action     Action   // allow, deny, filter
    Rule       string   // which rule matched
    Reason     string   // human-readable explanation (follows iam_pedro's reasoning field pattern)
    CallerCtx  CallerContext
    Success    *bool    // post-execution success/failure tracking (follows iam_pedro's ModAction.Success)
}
```

MVP: In-memory ring buffer with Prometheus metrics export.
Phase 2: Persistent storage with query API (following iam_pedro's `ModAction` table pattern — already proven in production).

---

## MCP Integration

The middleware exposes itself as an MCP server that wraps upstream MCP servers or direct tool execution:

```
Client (LLM) → Agent Middleware (MCP Server) → Upstream Tools (MCP Server / Direct)
```

MCP protocol support:
- `tools/list` - Returns filtered tool list based on caller policy
- `tools/call` - Intercepts, validates, executes, filters
- Standard MCP JSON-RPC transport (stdio or HTTP+SSE)

```go
// MCPServer wraps the middleware as an MCP-compatible server.
type MCPServer struct {
    middleware *Middleware
    transport  Transport // stdio or http
}

func (s *MCPServer) HandleToolsList(req ListToolsRequest) ListToolsResponse {
    allTools := s.middleware.executor.ListTools()
    // Filter based on caller context
    return ListToolsResponse{Tools: s.middleware.policy.FilterToolList(allTools)}
}

func (s *MCPServer) HandleToolsCall(req CallToolRequest) CallToolResponse {
    result, err := s.middleware.CallTool(req.Context(), req.Name, req.Arguments)
    // ... format as MCP response
}
```

---

## Integration Points

### professor_pedro

**TUI Agent** - Wrap `Orchestrator.executeToolCall()`:

```go
// Before (current):
func (o *Orchestrator) executeToolCall(toolName string, args map[string]interface{}) {
    // direct workflow dispatch
}

// After (with middleware):
func (o *Orchestrator) executeToolCall(toolName string, args map[string]interface{}) {
    result, err := o.middleware.CallTool(ctx, toolName, args)
    if result.Denied() {
        o.sendMessage(AgentMessage{Content: result.Reason, Type: "error"})
        return
    }
    // proceed with workflow
}
```

**Quiz tool scoping** - `show_quiz` only available during quiz-type PKO steps:

```yaml
# professor_pedro policy
rules:
  - name: quiz-step-only
    tools: ["show_quiz"]
    action: deny
    conditions:
      - field: "context.pko_step_type"
        operator: "not"
        value: "quiz"
```

**Iteration limit enforcement** - Middleware enforces existing limits via policy:

```yaml
  - name: session-turn-limit
    tools: ["*"]
    max_turns: 100        # matches existing maxTurnsPerSession
  - name: workflow-iteration-limit
    tools: ["*"]
    iteration_limit: 50   # matches existing maxWorkflowIterations
```

**DB state sync pattern** - Middleware reads authoritative state from database (following `get_current_pko_step` pattern that fetches fresh state from DB, not stale in-memory cache).

**Backend API** - Add as Chi middleware for tool-related endpoints:

```go
router.Route("/api/code", func(r chi.Router) {
    r.Use(agentMiddleware.HTTPMiddleware()) // constraint enforcement
    r.Post("/execute", codeHandler.Execute)
    r.Post("/evaluate", codeHandler.Evaluate)
})
```

### PedroCLI

**Tool Bridge** - Replace or wrap the existing `ToolBridge`:

```go
// The middleware implements ToolBridge interface
directBridge := tools.NewDirectBridge(registry)
constrainedBridge := middleware.New(directBridge, policy, auditor)

// Use constrainedBridge everywhere directBridge was used
agent := agents.NewBuilder(constrainedBridge, ...)
```

This is the cleanest integration because PedroCLI already has the `ToolBridge` abstraction.

### iam_pedro (Production-Ready Integration Target)

iam_pedro already has the most production-ready agent patterns: 22+ moderation tools, LLM decision pipeline, rate limiting, and database audit trail. The integration focuses on **unifying the existing moderation middleware** into the shared module, not building from scratch.

**Moderation Pipeline** - Replace custom enforcement with shared middleware:

```go
// The moderation system already follows the middleware pattern:
// Message → Quick Filter → LLM → Tool Call → Execute → Audit
// Integration replaces the custom enforcement layer with middleware.CallTool()

func (m *ModerationMonitor) handleMessage(msg ChatMessage) {
    // Quick filter stays in iam_pedro (platform-specific heuristics)
    if m.quickFilter.ShouldSkip(msg) {
        return
    }

    // LLM evaluation → tool call decision
    toolCall := m.llm.EvaluateMessage(msg)

    // Middleware enforcement (replaces custom validation)
    result, err := m.middleware.CallTool(ctx, toolCall.Name, toolCall.Args)
    if result.Denied() {
        m.auditDenied(msg, result)
        return
    }
    // Execute action...
}
```

**Discord Bot** - Wrap all tool execution (web search + future tools):

```go
func (d Client) askPedro(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // All tool calls go through middleware
    result, err := d.middleware.CallTool(ctx, toolCall.Name, toolCall.Args)
    if result.Denied() {
        // respond with constraint violation message
        return
    }
    // proceed with tool result
}
```

**Key migration items:**
- 22 moderation tool definitions → middleware `ToolDefinition` format
- Rate limiting (per-monitor counters) → middleware `MaxRate` rules
- Per-user escalation (warning → timeout → ban) → middleware escalation policy
- `ModAction` audit trail → middleware `Auditor` with database backend
- `AllowedTools` config → middleware policy YAML
```

---

## Module Structure

```
agent-middleware/          # New Go module (separate repo or shared module)
├── go.mod                 # module github.com/pedro/agent-middleware
├── middleware.go           # Core Middleware struct + CallTool
├── policy.go              # Policy evaluation engine
├── policy_loader.go       # YAML/JSON policy file loading
├── audit.go               # Audit interface + in-memory implementation
├── context.go             # CallerContext extraction + trusted/untrusted tagging
├── mcp/
│   ├── server.go          # MCP server wrapper
│   ├── transport.go       # stdio + HTTP transport
│   └── protocol.go        # MCP JSON-RPC message types
├── integrations/
│   ├── chi.go             # Chi HTTP middleware adapter
│   ├── toolbridge.go      # PedroCLI ToolBridge adapter
│   └── orchestrator.go    # professor_pedro Orchestrator adapter
├── metrics/
│   └── prometheus.go      # Prometheus metrics for violations/decisions
├── testutil/
│   └── mock.go            # Test helpers and mock implementations
└── examples/
    ├── basic/             # Minimal usage example
    ├── injection-demo/    # Prompt injection prevention demo
    └── policy-examples/   # Example policy files
```

---

## Milestones

### Milestone 1: Core Enforcement Engine

**Goal:** Working middleware that intercepts tool calls and evaluates policies.

**Deliverables:**
- [ ] `Middleware` struct implementing `ToolExecutor` interface
- [ ] `Policy` evaluation engine with rule matching
- [ ] YAML policy file loading
- [ ] `Auditor` interface with in-memory implementation
- [ ] Unit tests for allow/deny/filter decisions
- [ ] Basic example demonstrating the 3 failure modes (injection, unsafe tool, data leak)

**Integration:** None yet - standalone module with examples.

### Milestone 2: MCP Server

**Goal:** Middleware exposed as an MCP-compatible server.

**Deliverables:**
- [ ] MCP JSON-RPC protocol types
- [ ] `tools/list` with policy-filtered tool list
- [ ] `tools/call` with full enforcement pipeline
- [ ] stdio transport (for subprocess MCP)
- [ ] HTTP+SSE transport (for network MCP)
- [ ] Integration test: MCP client -> middleware -> mock tools

**Integration:** Can be used as a standalone MCP server wrapping any upstream MCP server.

### Milestone 3: PedroCLI Integration

**Goal:** Agent middleware enforcing constraints in PedroCLI's inference loop.

**Deliverables:**
- [ ] `ToolBridge` adapter wrapping `DirectBridge` with middleware
- [ ] Policy file support in `.pedrocli.json` config
- [ ] Violation events emitted via `ProgressCallback`
- [ ] Per-phase tool subset enforcement (leverage existing `Phase.Tools`)
- [ ] Demo: builder agent with constrained file access

**Integration:** PedroCLI imports `agent-middleware` module.

### Milestone 4: professor_pedro Integration

**Goal:** Agent middleware enforcing constraints in the TUI agent and backend API.

**Deliverables:**
- [ ] Orchestrator adapter wrapping `executeToolCall()` with middleware
- [ ] Chi middleware adapter for backend API tool endpoints
- [ ] `LLMClient` wrapper that filters tool list based on policy
- [ ] Violation messages surfaced in `AgentMessage` with metadata
- [ ] Per-PKO-step-type tool scoping (e.g., "read" steps can't access code editor)
- [ ] Demo: teaching agent with constrained tool access

**Integration:** professor_pedro imports `agent-middleware` module.

### Milestone 5: Observability & Content

**Goal:** Production observability and demo content for GTM.

**Deliverables:**
- [ ] Prometheus metrics: `agent_middleware_decisions_total{action,tool,rule}`
- [ ] Prometheus metrics: `agent_middleware_decision_duration_seconds`
- [ ] Grafana dashboard template
- [ ] Violation log query API (GET /violations)
- [ ] Demo repo: "Unsafe Agent -> Production-Ready Agent" walkthrough
- [ ] YouTube content: 3 failure mode demonstrations
- [ ] Blog post: "Your AI agent is unsafe"

**Integration:** Metrics exported from all integrated repos.

### Milestone 6: iam_pedro Integration — Unify Existing Moderation Middleware

**Goal:** Migrate iam_pedro's existing moderation agent infrastructure into the shared middleware module. iam_pedro already has 22+ moderation tools, rate limiting, LLM-driven decisions, and a database audit trail — this milestone unifies them, not builds from scratch.

**Deliverables:**
- [ ] Migrate 22 moderation tool definitions (`GetModerationToolDefinitions()`) to middleware `ToolDefinition` format
- [ ] Replace custom moderation enforcement with middleware `CallTool()` pipeline
- [ ] Migrate rate limiting (per-monitor action counters) to middleware `MaxRate` rules
- [ ] Migrate per-user escalation (warning → timeout → ban) to middleware escalation policy
- [ ] Migrate `ModAction` audit trail to middleware `Auditor` with database backend
- [ ] Migrate `AllowedTools` + `DryRun` config to middleware policy YAML
- [ ] Preserve quick filter heuristics in iam_pedro (platform-specific, not middleware)
- [ ] Middleware wrapping Discord bot tool execution (web search + future tools)
- [ ] Demo: moderation system running through shared middleware with full audit trail

**Integration:** iam_pedro imports `agent-middleware` module. Existing moderation pipeline preserved but enforcement layer replaced.

---

## Design Decisions

### Why a Shared Go Module?

All three repos are Go. A shared module means:
- Single implementation, no drift
- Type-safe integration
- Go module versioning for compatibility
- Can vendor if needed for stability

### Why MCP?

- Emerging standard for tool exposure
- PedroCLI already has `MCPClientAdapter`
- Enables third-party tool wrapping without code changes
- Composable: middleware MCP server wraps upstream MCP servers

### Why Policy Files (Not Code)?

- Non-developers can author policies (Phase 2 ICP)
- Policies can be version-controlled separately from code
- Enables dynamic policy reloading without restart
- Maps naturally to YAML/JSON config patterns already used in all repos

### Why Interface-Based Design?

Matches existing patterns in all repos:
- `ToolBridge` in PedroCLI
- `LLMClient` in professor_pedro
- `ChatResponseWriter` in iam_pedro

Easy to mock, easy to swap, easy to test.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Performance overhead per tool call | Latency in inference loop | In-memory policy evaluation, no I/O in hot path |
| Policy misconfiguration blocks legitimate calls | Agent stops working | Dry-run mode (log violations, don't block) |
| MCP spec changes | Breaking changes | Abstract transport layer, pin to stable spec version |
| Adoption friction | Nobody uses it | Drop-in adapters for each repo, minimal config to start |
| Over-engineering before validation | Wasted effort | MVP is just the core engine + 1 integration |
