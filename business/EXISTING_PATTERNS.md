# Existing Patterns Across Pedro Repos

Analysis of current agent, middleware, tool-calling, and MCP patterns across all Pedro repositories that inform the Agent Middleware design.

---

## iam_pedro - Bot Platform

### Constrained Agent Patterns

The Twitch moderation system is a **production-grade constrained agent** with LLM-driven decision-making, tool whitelisting, rate limiting, and audit logging.

| Pattern | Location | What It Does | Middleware Opportunity |
|---------|----------|-------------|----------------------|
| Moderation Tool Definitions | `ai/twitchchat/agent/moderation_tools.go` | **22 moderation tools** in two tiers: core (11) + extended (11) with permission-based selection via `GetModerationToolDefinitions()` | Mature tool registration pattern — extract for middleware |
| LLM Moderation Pipeline | `ai/twitchchat/agent/` | Message → Quick Filter (5 heuristics) → LLM evaluation → Tool call → Execute → Audit trail | Production decision pipeline — model for middleware enforcement |
| Quick Filter Heuristics | `ai/twitchchat/agent/` | Pre-LLM filtering that skips ~90% of messages using 5 heuristic checks | Performance optimization — `quick_filter` concept for middleware |
| Rate Limiting | `ai/twitchchat/agent/` | Per-monitor action counters with auto-reset; per-user escalation (warning → timeout → ban) | Extract as middleware rate limiting primitive |
| Database Audit Trail | `ai/twitchchat/agent/` | `ModAction` table logs every decision with reasoning, tool call, params, and success/failure | Production audit pattern — model for middleware audit system |
| Tool Call Parsing | `ai/twitchchat/agent/moderation_tools.go` | `ParseModerationToolCall()`, `IsModerationTool()` for safe dispatch | Validated tool dispatch pattern |
| Web Search Tool | `ai/agent/tools.go` | WebSearchTool with JSON schema definition + parser + executor | Wrap with constraint enforcement |
| Command Dispatch | `discord/commands.go` | Name-to-handler map for slash commands | Add permission checks |
| OAuth Token Refresh | Middleware | Priority chain with graceful fallback + health endpoint | Auth middleware pattern |
| Message Validation | `discord/ask.go` | `messageValidator()` before processing | Extend to context control |
| Async Tool Execution | `twitch/handleMessage.go` | Channel-based web search dispatch | Intercept point for tool control |

### Key Abstractions to Leverage

- **`ai/twitchchat/agent/moderation_tools.go`**: Two-tier tool definition with `GetModerationToolDefinitions(extended bool)` — permission-based tool selection
- **LLM decision pipeline**: Message → Quick Filter → LLM → Tool Call → Execute → Audit — exactly the middleware enforcement pattern
- **Per-user escalation**: Warning → timeout → ban chain with configurable thresholds
- **`ModAction` audit table**: Logs reasoning, tool call, params, success/failure — production audit trail
- **Channel-based dispatch**: Non-blocking `select` pattern for tool results
- **Config-driven behavior**: YAML configs for moderation rules (`ModerationConfig.AllowedTools`, `SensitivityLevel`, `DryRun`)
- **Interface composition**: `database.ChatResponseWriter` = `MessageWriter` + `ResponseWriter`
- **Metrics middleware**: Prometheus counters/histograms wrapping every handler

### iam_pedro as Proto-Middleware

The moderation system already implements many middleware concepts:
- **Tool whitelisting**: `AllowedTools` config controls which moderation tools are available
- **Dry-run mode**: `DryRun` flag logs decisions without executing actions
- **Rate limiting**: Per-monitor counters prevent action floods
- **Audit trail**: Every decision persisted with full context
- **Escalation**: Graduated response (warning → timeout → ban)

This makes iam_pedro the **most production-ready integration target** — the focus shifts from "building agent capabilities" to "unifying existing moderation middleware into the shared module."

---

## PedroCLI - Agent Framework

### Mature Agent Patterns

| Pattern | Location | What It Does | Middleware Opportunity |
|---------|----------|-------------|----------------------|
| Tool Registry | `pkg/tools/registry.go` | Centralized tool registration, filtering, discovery | Hook enforcement layer here |
| Tool Bridge | `pkg/toolformat/bridge.go` | Unified interface: DirectBridge, HybridBridge, MCPClientAdapter | Insert middleware at bridge level |
| Phased Executor | `pkg/agents/phased_executor.go` | Multi-phase workflows with per-phase tool subsets | Per-phase constraint policies |
| **Pre-Filtering Tool Definitions** | `pkg/agents/phased_executor.go` | `filterToolDefinitions()` removes tools before LLM sees them — **160x token reduction** | Exactly what the middleware should do at the policy layer |
| **Tool Call Validation** | `pkg/agents/phased_executor.go` | `validateToolCalls()` catches invalid tool names and parameter errors BEFORE execution, feeds corrections back to LLM | Pre-execution validation belongs in middleware |
| **Tool Loop Prevention** | `pkg/agents/phased_executor.go` | Tracks `calledTools` and `failedTools` maps — removes tools after success (prevent loops) or 3+ failures (give up) | Safety pattern for middleware iteration control |
| Inference Loop | `pkg/agents/executor.go` | Parse tool calls -> execute -> feed back to LLM | Central interception point |
| Model Formatters | `pkg/toolformat/formatter.go` | Model-specific tool call parsing (Qwen, Llama, Claude, GLM-4, etc.) | Format-agnostic enforcement |
| Bash Safety | Tool config | Allowlist/denylist for shell commands | Extend to all tools |
| Capability Detection | `pkg/tools/capabilities.go` | Tools declare required capabilities | Map to constraint policies |
| **Emergency Context Management** | Inference loop | Periodic forced compaction every N rounds + context budget warnings | Control what survives compaction |
| **Per-Tool Result Truncation** | `ToolResultLimits` config | Different character limits per tool type | Data control via middleware policy |

### Key Abstractions to Leverage

- **`ToolBridge` interface**: `CallTool(ctx, name, args) (*BridgeResult, error)` - perfect middleware insertion point
- **`ToolRegistry`**: Already supports filtering by category, capabilities - extend with permissions
- **`ToolDefinition`**: Has `Category`, `Parameters` (JSON Schema) - add `Constraints` field
- **`filterToolDefinitions()`**: Pre-LLM tool filtering — reduces token usage by 160x; model for middleware tool list filtering
- **`validateToolCalls()`**: Pre-execution validation catches invalid calls before they reach tools; feeds structured corrections back to LLM
- **`calledTools`/`failedTools` tracking**: Loop prevention and failure escalation — mature safety pattern
- **`ToolResultLimits`**: Per-tool-type character limits on results — data scoping via configuration
- **`PhaseCallback`**: Post-phase validation hooks - extend to pre-execution validation
- **`ProgressCallback`**: Event system for tool calls - add violation events
- **Config hierarchy**: `.pedrocli.json` (project) -> `~/.pedrocli.json` (user) -> defaults

### MCP Integration Already Exists

- `MCPClientAdapter` in `pkg/toolformat/bridge.go` for subprocess-based MCP tools
- `HybridBridge` combines direct + MCP execution
- This is the exact pattern the middleware should wrap/replace

---

## professor_pedro - Learning Platform

### Agent Patterns

| Pattern | Location | What It Does | Middleware Opportunity |
|---------|----------|-------------|----------------------|
| Orchestrator | `tui/internal/agent/orchestrator.go` | LLM-driven tool dispatch with conversation management | Wrap tool dispatch with constraints |
| Tool Definitions | `tui/internal/agent/tools.go` | **12 tools** via `GetAgentTools()` (including `show_quiz`) | Enforce per-tool permissions |
| **Interactive Quiz Tool** | `tui/internal/agent/tools.go` | `show_quiz` tool for interactive quiz UI component | Scope quiz access to appropriate PKO steps |
| Workflow Functions | `tui/internal/agent/workflows.go` | Goroutine-based workflows with channel I/O | Add data scoping per workflow |
| **Iteration Limits** | `tui/internal/agent/orchestrator.go` | `maxTurnsPerSession` (default 100) + `maxWorkflowIterations` (default 50) with mutex-protected counters | Model for middleware turn/iteration limiting |
| **LLM Timeout** | `tui/internal/agent/orchestrator.go` | 120-second context-based deadline on all LLM calls | Timeout enforcement pattern |
| **DB State Sync** | `tui/internal/agent/` | `get_current_pko_step` fetches fresh state from database, not stale in-memory cache | Server-authoritative state pattern |
| **Conversational Teaching Prompts** | System prompt | Explicit "NEVER DUMP CONTENT" rules with chunking guidance | Context control in action — shows prompt-level enforcement |
| Auth Middleware | `backend-go/internal/middleware/auth.go` | Session-based authentication | Extend with tool-level auth |
| Qwen Formatter | `tui/internal/llm/qwen_formatter.go` | Model-specific tool embedding | Format-agnostic constraint layer |

### Key Abstractions to Leverage

- **`LLMClient` interface**: `ChatCompletionWithTools(messages, tools, temperature)` - intercept tool list
- **`AgentMessage.Metadata`**: Trigger UI transitions - could carry constraint violation info
- **`Orchestrator.executeToolCall()`**: Single dispatch point for all tool calls
- **Iteration/turn limiting**: `maxTurnsPerSession` and `maxWorkflowIterations` with mutex-protected counters — safety pattern for middleware
- **DB-centric state**: Fresh state from database prevents stale cache issues — model for middleware state management
- **Chi middleware chain**: Already composable - add constraint middleware
- **PKO step types**: `"read"`, `"code"`, `"exercise"`, `"quiz"` - map to permission scopes

---

## pedro-ops - Infrastructure

### Relevant Patterns

| Pattern | Location | What It Does | Middleware Opportunity |
|---------|----------|-------------|----------------------|
| OpenAI Metrics Middleware | `internal/middleware/openai.go` | Extract and track LLM metrics from responses | Add constraint violation metrics |
| Secrets Management | `secrets/` | 1Password -> OpenBAO -> K8s secrets | Model for credential scoping |
| Prometheus Integration | `internal/metrics/client.go` | Dual expvar + Prometheus metrics | Observability for violations |
| Phased Deployment | `scripts/phase*.sh` | Staged rollout with verification | Model for phased middleware adoption |

---

## Cross-Repo Patterns Summary

### What Already Exists

1. **Tool definition format**: JSON Schema-based tool definitions (all repos)
2. **Tool registry**: Centralized registration and discovery (PedroCLI)
3. **Tool bridge**: Abstracted execution layer (PedroCLI)
4. **Tool pre-filtering**: `filterToolDefinitions()` removes tools before LLM sees them (PedroCLI) — 160x token reduction
5. **Tool call validation**: `validateToolCalls()` catches invalid calls before execution (PedroCLI)
6. **Tool loop prevention**: `calledTools`/`failedTools` tracking prevents infinite loops (PedroCLI)
7. **Rate limiting**: Per-monitor action counters with auto-reset (iam_pedro moderation)
8. **Audit trail**: `ModAction` table logs every moderation decision with reasoning (iam_pedro)
9. **Iteration limits**: `maxTurnsPerSession` + `maxWorkflowIterations` with mutex counters (professor_pedro)
10. **Model-agnostic formatting**: Multiple LLM backends supported (PedroCLI, professor_pedro)
11. **Channel-based async**: Goroutine + channel patterns for tool execution (all repos)
12. **Metrics/observability**: Prometheus integration (iam_pedro, pedro-ops)
13. **Config-driven rules**: YAML/JSON config for behavior (all repos)
14. **Interface-based DI**: Easy to swap implementations (all repos)

### What's Missing (The Middleware Gap)

1. **No unified constraint enforcement layer** — iam_pedro has moderation enforcement, PedroCLI has tool validation, but these are separate implementations with no shared abstraction
2. **No cross-repo permission model** — iam_pedro has `AllowedTools` for moderation, PedroCLI has per-phase tool subsets, but no shared permission framework
3. **No context separation** between trusted instructions and untrusted input — none of the repos tag input provenance
4. **No unified data scoping** — PedroCLI has `ToolResultLimits` per tool type, but no general-purpose field-level filtering
5. **No shared audit system** — iam_pedro has `ModAction` in its database, PedroCLI has progress events, but no common audit interface
6. **No shared policy configuration** — each repo has its own config format (YAML moderation rules, JSON pedrocli config, Go constants in professor_pedro)
7. **No shared middleware module** — each repo reinvents its own safety checks independently

### Natural Integration Points

| Repo | Integration Point | How |
|------|-------------------|-----|
| **iam_pedro** | `ai/twitchchat/agent/moderation_tools.go` | Unify 22 moderation tool definitions into middleware tool registration pattern |
| **iam_pedro** | Moderation LLM pipeline | Replace custom enforcement with shared middleware `CallTool()` — already has the right shape |
| **iam_pedro** | Rate limiting + escalation | Extract as middleware primitives — per-user action counters + graduated response |
| **iam_pedro** | `ModAction` audit trail | Model the middleware audit system after this production pattern |
| **iam_pedro** | Discord command handlers | Add permission middleware before handler dispatch |
| **PedroCLI** | `ToolBridge.CallTool()` | Insert middleware bridge that validates before delegating |
| **PedroCLI** | `filterToolDefinitions()` | Move pre-LLM tool filtering into middleware policy layer |
| **PedroCLI** | `validateToolCalls()` | Move pre-execution validation into middleware `ValidateToolCall()` |
| **PedroCLI** | `ToolRegistry` | Filter available tools based on policy |
| **professor_pedro** | `Orchestrator.executeToolCall()` | Validate tool call before workflow launch |
| **professor_pedro** | `LLMClient.ChatCompletionWithTools()` | Filter tool list (including `show_quiz`) based on context/PKO step |
| **professor_pedro** | Iteration limits | Enforce `maxTurnsPerSession`/`maxWorkflowIterations` via middleware policy |
| **professor_pedro** | Backend Chi middleware chain | Add constraint middleware to API routes |
