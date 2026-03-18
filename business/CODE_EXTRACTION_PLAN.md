# Code Extraction Plan: What Goes Into the New Repo

## Principle

**Extract shared abstractions and enforcement logic. Leave platform-specific implementations in their repos.**

The new repo (`agent-middleware`) should contain:
1. Code that is **duplicated or near-duplicated** across repos today
2. Code that **defines the middleware contracts** (interfaces, types, policy engine)
3. Code that is **generic enough** to work across all three consumers

It should NOT contain:
- Platform-specific tool implementations (web search, code editor, bash)
- LLM client code (each repo has its own model integration)
- UI/handler code

---

## What Moves Into the New Repo

### 1. Unified Tool Types (NEW - resolves inconsistencies)

**Source:** Fragmented across all 3 repos with conflicts

| Current Location | Type | Issue |
|-----------------|------|-------|
| PedroCLI `pkg/tools/interface.go` | `Tool` interface, `Result` struct | Rich but PedroCLI-specific |
| PedroCLI `pkg/toolformat/definition.go` | `ToolDefinition`, `ParameterSchema`, `PropertySchema` | Duplicates interface.go categories |
| professor_pedro `tui/internal/llm/client.go` | `Tool`, `FunctionDef`, `ToolCall`, `FunctionCall` | OpenAI-format only, no execution interface |
| iam_pedro `ai/agent/tools.go` | Uses langchaingo `llms.Tool` | External library types |

**What goes in the new repo:**

```go
// types.go - Canonical tool types for the middleware
package middleware

// ToolDefinition describes a tool for both LLM consumption and policy enforcement.
type ToolDefinition struct {
    Name        string          `json:"name" yaml:"name"`
    Description string          `json:"description" yaml:"description"`
    Category    string          `json:"category" yaml:"category"`
    Parameters  ParameterSchema `json:"parameters" yaml:"parameters"`
}

type ParameterSchema struct {
    Type       string                     `json:"type"`
    Properties map[string]PropertySchema   `json:"properties"`
    Required   []string                   `json:"required"`
}

type PropertySchema struct {
    Type        string          `json:"type"`
    Description string          `json:"description"`
    Enum        []string        `json:"enum,omitempty"`
    Items       *PropertySchema `json:"items,omitempty"`
    Default     interface{}     `json:"default,omitempty"`
}

// ToolResult is the canonical result type.
type ToolResult struct {
    Success       bool                   `json:"success"`
    Output        string                 `json:"output"`
    Error         string                 `json:"error,omitempty"`
    ModifiedFiles []string               `json:"modified_files,omitempty"`
    Data          map[string]interface{} `json:"data,omitempty"`
}
```

**Impact on existing repos:**
- **PedroCLI**: Replace `pkg/toolformat/definition.go` types with imports from middleware. Keep `pkg/tools/interface.go` `Tool` interface as-is (it's the execution interface, not the definition). Add adapter functions `ToMiddlewareDefinition()` / `FromMiddlewareDefinition()`.
- **professor_pedro**: Replace inline `Tool`/`FunctionDef` in `tui/internal/llm/client.go` with middleware types or add thin conversion layer. The LLM-specific format (OpenAI, Qwen) stays in each repo.
- **iam_pedro**: Add adapter from `llms.Tool` (langchaingo) to middleware `ToolDefinition`. Langchaingo dependency stays in iam_pedro.

---

### 2. ToolBridge Interface (EXTRACTED from PedroCLI)

**Source:** `PedroCLI/pkg/toolformat/bridge.go`

```go
// This is already the right abstraction. Move it as-is.
type ToolExecutor interface {
    CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error)
    ListTools() []ToolDefinition
}
```

**What stays in PedroCLI:**
- `DirectBridge` implementation (calls tools directly via registry)
- `HybridBridge` implementation (direct + MCP fallback)
- `MCPClientAdapter` implementation (MCP subprocess)
- Job ID/status extraction utilities

**What changes in PedroCLI:**
- `ToolBridge` interface replaced by `middleware.ToolExecutor` import
- `BridgeResult` replaced by `middleware.ToolResult` import
- Existing bridge implementations now satisfy `middleware.ToolExecutor`

---

### 3. Capability System (EXTRACTED from PedroCLI)

**Source:** `PedroCLI/pkg/tools/capabilities.go`

```go
// Move the interface and checker. Capability constants can be extended per-repo.
type Capability string

type CapabilityChecker interface {
    Check(cap Capability) bool
    CheckAll(caps []Capability) []Capability
    Available() []Capability
}
```

**What stays in PedroCLI:**
- Concrete capability constants (`CapabilityGit`, `CapabilityNetwork`, etc.)
- Runtime detection logic (checking if git exists, if network is available)

**What goes in the new repo:**
- `CapabilityChecker` interface
- `Capability` type
- Integration with policy engine (capabilities as conditions)

---

### 4. Tool Registry (PARTIALLY EXTRACTED from PedroCLI)

**Source:** `PedroCLI/pkg/tools/registry.go`

The registry is powerful but tightly coupled to PedroCLI's `ExtendedTool` interface. Extract the **filtering and discovery** abstractions, leave the implementation.

**What goes in the new repo:**

```go
// registry.go - Minimal tool catalog for policy evaluation
type ToolCatalog interface {
    Get(name string) (*ToolDefinition, bool)
    List() []ToolDefinition
    ListByCategory(category string) []ToolDefinition
}
```

**What stays in PedroCLI:**
- Full `ToolRegistry` struct with `Register()`, `Unregister()`, event listeners, GBNF grammar generation, `Clone()`, `Merge()`
- PedroCLI's registry implements `ToolCatalog` via adapter

**What stays in professor_pedro:**
- Static `GetAgentTools()` function, but it returns `[]middleware.ToolDefinition` instead of custom types

---

### 5. Policy Engine (NEW)

**Source:** No existing equivalent - this is the core new code.

Inspired by patterns already in the repos:
- iam_pedro's `ModerationConfig.AllowedTools` + `DryRun` mode
- PedroCLI's bash allowlist/denylist in config
- PedroCLI's `Phase.Tools` (per-phase tool subsets)

```go
// policy.go - Constraint evaluation engine
type Policy struct { Rules []Rule }
type Rule struct { Name, Action string; Tools []string; Conditions []Condition; ... }

// policy_loader.go - YAML/JSON loading
func LoadPolicy(path string) (*Policy, error)

// evaluator.go - Decision logic
func (p *Policy) Evaluate(ctx CallerContext, tool string, args map[string]interface{}) Decision
```

---

### 6. Context Control (NEW)

**Source:** No existing equivalent - addresses the trusted/untrusted input gap.

Informed by:
- professor_pedro's system prompt vs user message distinction
- iam_pedro's `ModerationContext` struct (recent messages + channel rules)
- PedroCLI's `Phase.SystemPrompt` (different context per phase)

```go
// context.go
type CallerContext struct {
    Identity    string            // user ID, bot name, agent name
    Role        string            // "admin", "user", "agent"
    Trusted     bool              // is this a system-initiated call?
    Tags        map[string]string // arbitrary metadata
    Source      string            // "discord", "tui", "cli", "api"
}
```

---

### 7. Audit System (NEW)

**Source:** No existing equivalent, but follows patterns from:
- pedro-ops `internal/metrics/client.go` (dual Prometheus + expvar)
- iam_pedro's moderation `InsertModAction()` database logging
- PedroCLI's `ProgressCallback` event system

```go
// audit.go
type Auditor interface {
    Record(decision Decision)
    GetViolations(since time.Time) []Decision
}

// audit_memory.go - In-memory ring buffer (MVP)
// audit_prometheus.go - Prometheus metrics export
```

---

### 8. MCP Server Wrapper (NEW)

**Source:** PedroCLI has `MCPClientAdapter` (client side). The new repo adds the **server side**.

```go
// mcp/server.go - Expose middleware as MCP server
// mcp/transport.go - stdio + HTTP+SSE
// mcp/protocol.go - JSON-RPC message types
```

---

### 9. Moderation Tool Registration Pattern (EXTRACTED from iam_pedro)

**Source:** `iam_pedro/ai/twitchchat/agent/moderation_tools.go`

A mature tool registration pattern with:
- `GetModerationToolDefinitions(extended bool)` — two-tier tool selection (core 11 + extended 11)
- `ParseModerationToolCall()` — validated tool call parsing
- `IsModerationTool()` — safe dispatch verification

```go
// This pattern maps directly to middleware ToolCatalog
// Two-tier definitions → policy-based tool filtering
// ParseModerationToolCall() → middleware ValidateToolCall()
// IsModerationTool() → middleware tool catalog lookup
```

**What goes in the new repo:**
- Tool definition registration pattern (generalized from moderation-specific to universal)
- Two-tier permission-based tool selection → middleware policy rules

**What stays in iam_pedro:**
- Moderation-specific tool implementations (timeout_user, ban_user, etc.)
- Moderation-specific tool call parsing logic

---

### 10. Pre-Execution Validation (EXTRACTED from PedroCLI)

**Source:** `PedroCLI/pkg/agents/phased_executor.go`

Two critical validation functions:
- `validateToolCalls()` — catches invalid tool names and parameter errors BEFORE execution, feeds structured corrections back to LLM
- `filterToolDefinitions()` — removes tools before LLM sees them (160x token reduction)

```go
// These belong in the middleware:
// validateToolCalls() → middleware.ValidateToolCall()
// filterToolDefinitions() → middleware.FilterToolList() via policy
```

**What goes in the new repo:**
- `ValidateToolCall()` method on `Middleware` — pre-execution validation
- `FilterToolList()` — policy-based tool list filtering before sending to LLM

**What stays in PedroCLI:**
- LLM correction feedback loop (platform-specific UX)
- Token reduction measurement/reporting

---

### 11. Rate Limiting (EXTRACTED from iam_pedro)

**Source:** iam_pedro moderation monitor

Per-monitor action counters with auto-reset and per-user escalation (warning → timeout → ban).

**What goes in the new repo:**
- Generic rate limiting primitive with configurable windows and counters
- Escalation policy support (graduated response chains)

**What stays in iam_pedro:**
- Moderation-specific escalation thresholds
- Platform-specific action execution (Twitch API calls)

---

### 12. Prometheus Metrics (EXTRACTED from pedro-ops + iam_pedro)

**Source:**
- pedro-ops `internal/metrics/client.go` - dual metric system pattern
- iam_pedro `metrics/server.go` - Prometheus counters/histograms

**What goes in the new repo:**

```go
// metrics/prometheus.go
var (
    DecisionsTotal = prometheus.NewCounterVec(...)    // {action, tool, rule}
    DecisionDuration = prometheus.NewHistogramVec(...) // {tool}
    ViolationsTotal = prometheus.NewCounterVec(...)    // {tool, rule}
)
```

**What stays in source repos:**
- Their existing metrics (command counts, LLM latency, etc.)
- They register middleware metrics alongside their own

---

## What Does NOT Move

| Code | Stays In | Reason |
|------|----------|--------|
| `WebSearchTool` implementation | iam_pedro | Platform-specific tool |
| Moderation tool implementations (timeout_user, ban_user, etc.) | iam_pedro | Platform-specific Twitch API tools |
| Moderation monitor goroutine | iam_pedro | Platform-specific concurrency pattern |
| Quick filter heuristics | iam_pedro | Platform-specific pre-LLM optimization (Twitch chat patterns) |
| `DirectBridge`, `HybridBridge` | PedroCLI | Implementation, not interface |
| `ToolRegistry` full implementation | PedroCLI | Too coupled to PedroCLI's `ExtendedTool` |
| LLM correction feedback loop | PedroCLI | Platform-specific UX for tool validation errors |
| `Orchestrator` | professor_pedro | Platform-specific agent logic |
| `LLMClient` / Qwen formatter | professor_pedro | Model-specific integration |
| Tool call parsing (Qwen XML, Llama, GLM-4, etc.) | PedroCLI | Model-specific formatting |
| Auth middleware | professor_pedro | HTTP-specific, not tool-related |
| Bash allowlist/denylist config | PedroCLI | Migrates to middleware policy format |
| `ModerationConfig` struct | iam_pedro | Migrates to middleware policy format |

---

## Migration Strategy Per Repo

### PedroCLI

**Phase 1 (Milestone 3):**
1. Add `agent-middleware` as Go module dependency
2. Create adapter: `PedroCLI ToolBridge` -> `middleware.ToolExecutor`
3. Replace `BridgeResult` with `middleware.ToolResult` (or add conversion)
4. Migrate bash allowlist/denylist from config to middleware policy YAML
5. Wire `middleware.New(bridge, policy, auditor)` into `NewAppContext()`

**What breaks:** Nothing if adapters are done right. The `DirectBridge` still works, it just gets wrapped.

**Lines of code affected:** ~50-100 lines of adapter code. No deletions needed initially.

### professor_pedro

**Phase 1 (Milestone 4):**
1. Add `agent-middleware` as Go module dependency
2. Create adapter: `executeToolCall()` calls `middleware.CallTool()` first
3. `GetAgentTools()` returns `[]middleware.ToolDefinition` (or converts)
4. Add `AgentMessage` type `"violation"` for denied calls
5. Add policy YAML to `~/.professor-pedro/` config directory

**What breaks:** Nothing. The orchestrator still dispatches workflows, middleware just validates before dispatch.

**Lines of code affected:** ~30-50 lines of adapter code in `orchestrator.go`.

### iam_pedro

**Phase 1 (Milestone 6) — Significantly larger scope than originally estimated:**

iam_pedro now has a mature moderation agent system that needs migration, not just a single web search tool wrapper.

1. Add `agent-middleware` as Go module dependency
2. Migrate 22 moderation tool definitions (`GetModerationToolDefinitions()`) to middleware `ToolDefinition` format
3. Create adapter: `llms.Tool` <-> `middleware.ToolDefinition` (for Discord bot)
4. Replace custom moderation enforcement with middleware `CallTool()` pipeline
5. Migrate rate limiting (per-monitor action counters with auto-reset) to middleware `MaxRate` rules
6. Migrate per-user escalation config (warning → timeout → ban) to middleware escalation policy
7. Migrate `ModAction` audit trail to middleware `Auditor` with database backend adapter
8. Migrate `AllowedTools` + `DryRun` config to middleware policy YAML
9. Tag Discord/Twitch user input as untrusted in `CallerContext`
10. Preserve quick filter heuristics in iam_pedro (platform-specific, not middleware)

**What breaks:** Nothing if adapters are done right. Moderation pipeline shape stays the same, enforcement layer swaps to middleware.

**Lines of code affected:** ~150-250 lines of adapter code + policy YAML files. `ModerationConfig` fields partially deprecated (AllowedTools, DryRun move to policy; platform-specific fields stay).

---

## New Repo Structure

```
agent-middleware/
├── go.mod                      # module github.com/pedro/agent-middleware
├── go.sum
├── LICENSE
├── README.md
│
├── types.go                    # ToolDefinition, ToolResult, ParameterSchema
├── executor.go                 # ToolExecutor interface
├── middleware.go               # Middleware struct (core enforcement)
├── context.go                  # CallerContext, trusted/untrusted tagging
├── catalog.go                  # ToolCatalog interface
├── capability.go               # Capability type, CapabilityChecker interface
│
├── policy/
│   ├── policy.go               # Policy, Rule, Condition types
│   ├── evaluator.go            # Rule matching and decision logic
│   ├── loader.go               # YAML/JSON policy file loading
│   └── matchers.go             # Condition matchers (eq, contains, matches, not)
│
├── audit/
│   ├── audit.go                # Auditor interface, Decision type
│   ├── memory.go               # In-memory ring buffer implementation
│   └── prometheus.go           # Prometheus metrics export
│
├── mcp/
│   ├── server.go               # MCP server wrapper
│   ├── transport_stdio.go      # stdio transport
│   ├── transport_http.go       # HTTP+SSE transport
│   └── protocol.go             # JSON-RPC message types
│
├── adapters/
│   ├── langchain.go            # llms.Tool <-> ToolDefinition (for iam_pedro)
│   ├── openai.go               # OpenAI function format <-> ToolDefinition (for professor_pedro)
│   └── chi.go                  # Chi HTTP middleware adapter (for professor_pedro backend)
│
├── testutil/
│   ├── mock_executor.go        # Mock ToolExecutor for testing
│   ├── mock_auditor.go         # Mock Auditor for testing
│   └── policy_builder.go       # Test helper to build policies programmatically
│
└── examples/
    ├── basic/                  # Minimal usage
    ├── injection-demo/         # Prompt injection prevention
    ├── mcp-proxy/              # MCP server wrapping upstream MCP
    └── policy-examples/        # Example policy YAML files
```

---

## Dependency Direction

```
agent-middleware (new repo)
    ├── no dependencies on any Pedro repo
    ├── depends on: gopkg.in/yaml.v3, prometheus client, standard library
    │
    ▼ imported by:
    │
    ├── PedroCLI          (imports agent-middleware)
    ├── professor_pedro   (imports agent-middleware)
    └── iam_pedro         (imports agent-middleware)
```

The middleware module has **zero dependencies on consuming repos**. Each repo imports it and writes thin adapters to bridge their existing types.

---

## Summary: Lines of New vs Extracted Code

| Category | Estimated Lines | Source |
|----------|----------------|--------|
| Unified types (`types.go`, `executor.go`, `catalog.go`) | ~150 | Extracted/unified from all 3 repos |
| Capability interface (`capability.go`) | ~30 | Extracted from PedroCLI |
| Policy engine (`policy/`) | ~400 | **New code** |
| Context control (`context.go`) | ~60 | **New code**, informed by existing patterns |
| Audit system (`audit/`) | ~200 | **New code**, follows pedro-ops metrics patterns |
| MCP server (`mcp/`) | ~350 | **New code**, mirrors PedroCLI MCP client |
| Adapters (`adapters/`) | ~150 | **New code**, thin conversion layers |
| Test utilities (`testutil/`) | ~100 | **New code** |
| Examples (`examples/`) | ~200 | **New code** |
| **Total** | **~1,640** | ~180 extracted, ~1,460 new |

The middleware is mostly **new code**. The extraction is primarily about **type unification** and **interface standardization**, not moving large chunks of implementation.
