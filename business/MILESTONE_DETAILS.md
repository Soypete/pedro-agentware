# Agent Middleware - Milestone Details

Detailed breakdown of each milestone with acceptance criteria, dependencies, and estimated scope.

---

## Milestone 1: Core Enforcement Engine

### Context

This is the foundation. Everything else builds on the policy evaluation engine and the `ToolExecutor` wrapping pattern. The design mirrors PedroCLI's `ToolBridge` interface so that integration is a drop-in.

### Tasks

1. **Define core types** - `ToolExecutor`, `ToolResult`, `ToolDefinition`, `CallerContext`
2. **Implement `Middleware` struct** - wraps any `ToolExecutor`, intercepts `CallTool()`
3. **Implement `ValidateToolCall()` method** - pre-execution validation inspired by PedroCLI's `validateToolCalls()` pattern (catches invalid tool names and parameter errors BEFORE execution)
4. **Implement `Policy` evaluator** - loads rules, evaluates conditions against tool name + args + caller context
5. **Implement condition matchers** - `eq`, `contains`, `matches`, `not`, `not_matches`
6. **Implement `max_turns` and `iteration_limit` rule types** - turn/iteration limiting inspired by professor_pedro's `maxTurnsPerSession` (100) and `maxWorkflowIterations` (50)
7. **Implement `quick_filter` concept** - heuristic pre-evaluation skip inspired by iam_pedro's quick filter (~90% message skip rate)
8. **Implement YAML policy loader** - parse `agent-policy.yaml` into `Policy` struct
9. **Implement `Auditor` interface** - in-memory ring buffer modeled after iam_pedro's `ModAction` table pattern (reasoning, tool call, params, success/failure)
10. **Implement data filtering** - `RedactFields` support to strip fields from `ToolResult`
11. **Write unit tests** - cover all 3 decision types (allow, deny, filter) + validation + iteration limits
12. **Build 3 failure mode examples:**
    - Prompt injection: untrusted input tries to call `bash rm -rf /` -> denied
    - Unsafe tool: agent calls `file` on `.env` -> denied
    - Data leak: tool result contains `password` field -> redacted

### Acceptance Criteria

- [ ] `middleware.New(executor, policy, auditor)` returns a working `ToolExecutor`
- [ ] `ValidateToolCall()` catches unknown tools and invalid parameters
- [ ] Policy file loaded from YAML
- [ ] Allow/deny decisions based on tool name and argument matching
- [ ] Regex-based argument validation
- [ ] `max_turns` and `iteration_limit` enforcement
- [ ] `quick_filter` heuristic skip evaluation
- [ ] Result field redaction
- [ ] All decisions recorded in auditor (with reasoning and success fields)
- [ ] 90%+ test coverage on policy evaluation
- [ ] 3 runnable example programs demonstrating failure modes

### Dependencies

- None (standalone module)

---

## Milestone 2: MCP Server

### Context

MCP is the emerging standard for exposing tools to LLMs. Wrapping the middleware as an MCP server means any MCP-compatible client can use it. PedroCLI already has an `MCPClientAdapter`, making this a natural bridge.

### Tasks

1. **Define MCP protocol types** - `ListToolsRequest/Response`, `CallToolRequest/Response`, JSON-RPC envelope
2. **Implement `MCPServer`** - wraps `Middleware`, handles `tools/list` and `tools/call`
3. **Implement stdio transport** - for subprocess-based MCP (matches PedroCLI pattern)
4. **Implement HTTP+SSE transport** - for network-based MCP
5. **Add tool list filtering** - `tools/list` returns only policy-permitted tools
6. **Integration test** - MCP client -> middleware MCP server -> mock tool MCP server

### Acceptance Criteria

- [ ] MCP server responds to `tools/list` and `tools/call` over stdio
- [ ] MCP server responds over HTTP+SSE
- [ ] Tool list filtered by caller policy
- [ ] Tool calls fully enforced (deny returns MCP error, filter modifies result)
- [ ] Can chain: client -> middleware MCP -> upstream MCP server

### Dependencies

- Milestone 1 (core engine)

---

## Milestone 3: PedroCLI Integration

### Context

PedroCLI has the most mature agent architecture with `ToolBridge`, `ToolRegistry`, and phased execution. The middleware plugs in at the `ToolBridge` layer, which is already an abstraction boundary.

### Key Integration Points

1. **`pkg/toolformat/bridge.go`** - `ToolBridge` interface matches our `ToolExecutor`
2. **`pkg/tools/registry.go`** - Can filter tools before passing to LLM
3. **`pkg/agents/phased_executor.go`** - `Phase.Tools` already restricts per-phase tools; `filterToolDefinitions()` and `validateToolCalls()` are middleware candidates
4. **`pkg/agents/executor.go`** - Inference loop emits `ProgressEventToolCall`

### Tasks

1. **Create `ToolBridge` adapter** - wraps `DirectBridge` or `HybridBridge` with middleware
2. **Add policy config** - `"middleware"` section in `.pedrocli.json`
3. **Wire into agent initialization** - `NewAppContext()` wraps bridge with middleware
4. **Integrate tool loop prevention** - migrate `calledTools`/`failedTools` tracking from phased executor to middleware (remove tools after success to prevent loops, give up after 3+ failures)
5. **Integrate per-tool result truncation** - migrate `ToolResultLimits` config to middleware data control policy
6. **Add violation progress events** - `ProgressEventViolation` type
7. **Per-phase policy scoping** - different rules per `Phase.Name`
8. **Bash tool enforcement** - migrate existing allowlist/denylist to middleware policy
9. **Demo** - Builder agent that can't access `.env` files or run destructive commands

### Acceptance Criteria

- [ ] PedroCLI `go build` succeeds with middleware dependency
- [ ] Policy loaded from `.pedrocli.json` or standalone YAML file
- [ ] Tool calls go through middleware before execution
- [ ] Tool loop prevention works through middleware (calledTools/failedTools)
- [ ] Per-tool result truncation enforced via middleware data control
- [ ] Violations appear in progress output (SSE events)
- [ ] Existing bash allowlist/denylist works through middleware
- [ ] Phased execution respects per-phase tool restrictions via middleware

### Dependencies

- Milestone 1 (core engine)
- Milestone 2 (MCP server, optional - only if using MCP transport)

---

## Milestone 4: professor_pedro Integration

### Context

professor_pedro has an LLM-orchestrated TUI agent and a Go backend with Chi middleware. The middleware needs to integrate at two levels: the TUI orchestrator (tool dispatch) and the backend API (HTTP middleware).

### Key Integration Points

1. **`tui/internal/agent/orchestrator.go`** - `executeToolCall()` is the single dispatch point; `maxTurnsPerSession`/`maxWorkflowIterations` are middleware candidates
2. **`tui/internal/agent/tools.go`** - `GetAgentTools()` returns 12 tools (including `show_quiz`) to LLM
3. **`tui/internal/llm/client.go`** - `ChatCompletionWithTools()` receives tool list
4. **`backend-go/internal/middleware/`** - Chi middleware chain
5. **`backend-go/internal/handlers/code.go`** - Code execution endpoints

### Tasks

1. **Create orchestrator adapter** - wraps `executeToolCall()` with middleware check
2. **Filter tool list** - middleware removes tools the current user/context shouldn't see
3. **Quiz tool scoping** - `show_quiz` only available during quiz-type PKO steps via middleware policy
4. **Iteration limit enforcement** - migrate `maxTurnsPerSession` (100) and `maxWorkflowIterations` (50) to middleware `max_turns`/`iteration_limit` rules
5. **DB state sync pattern** - middleware reads authoritative state from database (following `get_current_pko_step` pattern)
6. **Surface violations in UI** - `AgentMessage` with type `"violation"` and metadata
7. **PKO-step-type scoping** - `"read"` steps only allow `explain_concept`, `"exercise"` steps allow `show_code_editor`, `"quiz"` steps allow `show_quiz`
8. **Chi middleware adapter** - HTTP middleware that enforces policies on API endpoints
9. **Code execution constraints** - resource limits enforced via middleware, not just Docker
10. **Demo** - Teaching agent that shows violations when student tries to bypass exercise flow

### Acceptance Criteria

- [ ] professor_pedro `go build` succeeds with middleware dependency
- [ ] Tool calls in TUI agent go through middleware
- [ ] `show_quiz` scoped to quiz-type PKO steps only
- [ ] Iteration limits enforced via middleware policy (not just Go constants)
- [ ] LLM only sees tools it's permitted to use for current step type
- [ ] Violations displayed in chat with explanation
- [ ] Backend API code execution endpoints have middleware enforcement
- [ ] PKO step progression can't be bypassed via direct tool calls

### Dependencies

- Milestone 1 (core engine)
- Milestone 3 (PedroCLI integration, for shared learnings)

---

## Milestone 5: Observability & Content

### Context

Observability makes the middleware visible and debuggable. Content drives GTM. Both are necessary before the middleware is "production-ready" itself.

### Tasks

1. **Prometheus metrics** - counters for decisions (allow/deny/filter by tool and rule)
2. **Duration histogram** - policy evaluation latency
3. **Grafana dashboard** - template JSON for all middleware metrics
4. **Violation log API** - HTTP endpoint to query audit log
5. **Demo repository** - Standalone repo with README, examples, and walkthrough
6. **YouTube content** - Script and record 3 failure mode demos
7. **Blog post** - "Your AI agent is unsafe" with code examples

### Acceptance Criteria

- [ ] Prometheus metrics exported and scrapeable
- [ ] Grafana dashboard shows decisions, violations, latency
- [ ] Violation log queryable via HTTP API (time range, tool, action)
- [ ] Demo repo has working examples that anyone can clone and run
- [ ] YouTube video published
- [ ] Blog post published

### Dependencies

- Milestone 3 or 4 (need at least one real integration for content)

---

## Milestone 6: iam_pedro Integration — Unify Existing Moderation Middleware

### Context

iam_pedro is **no longer a future integration target** — it's the most production-ready consumer. The Twitch moderation system already IS a constrained agent with:
- **22 moderation tool definitions** in two tiers (core 11 + extended 11) with permission-based selection
- **LLM-driven decision pipeline**: Message → Quick Filter → LLM evaluation → Tool call → Execute → Audit
- **Rate limiting**: Per-monitor action counters with auto-reset
- **Per-user escalation**: Warning → timeout → ban
- **Database audit trail**: `ModAction` table with reasoning, tool call, params, success/failure
- **Dry-run mode**: Log decisions without executing actions

This milestone focuses on **migration and unification**, not building from scratch.

### Key Integration Points

1. **`ai/twitchchat/agent/moderation_tools.go`** - 22 moderation tool definitions (`GetModerationToolDefinitions()`, `ParseModerationToolCall()`, `IsModerationTool()`)
2. **`ai/twitchchat/agent/`** - LLM moderation pipeline with quick filter heuristics
3. **Moderation monitor** - Rate limiting with per-monitor counters and per-user escalation
4. **`ModAction` database table** - Production audit trail
5. **`ai/agent/tools.go`** - Discord bot WebSearchTool execution
6. **`discord/ask.go`** - Command handler with tool call detection
7. **`ModerationConfig`** - AllowedTools, SensitivityLevel, DryRun settings

### Tasks

1. **Migrate 22 moderation tool definitions** - `GetModerationToolDefinitions()` → middleware `ToolDefinition` format with two-tier policy (core vs extended)
2. **Replace custom enforcement** - Moderation pipeline enforcement → middleware `CallTool()` with same decision flow
3. **Migrate rate limiting** - Per-monitor action counters → middleware `MaxRate` rules with auto-reset windows
4. **Migrate per-user escalation** - Warning → timeout → ban chain → middleware escalation policy
5. **Migrate audit trail** - `ModAction` table → middleware `Auditor` with database backend adapter
6. **Migrate config to policy** - `AllowedTools` + `DryRun` + `SensitivityLevel` → middleware policy YAML
7. **Preserve quick filter** - Keep heuristic pre-LLM filtering in iam_pedro (platform-specific Twitch chat patterns)
8. **Context tagging** - Tag Discord/Twitch user input as untrusted, system prompts as trusted
9. **Wrap Discord bot tools** - All tool calls (web search + future) go through middleware
10. **Demo** - Moderation system running through shared middleware with full audit trail visible

### Acceptance Criteria

- [ ] iam_pedro `go build` succeeds with middleware dependency
- [ ] All 22 moderation tools registered in middleware format
- [ ] Moderation LLM pipeline uses middleware `CallTool()` for enforcement
- [ ] Rate limiting works through middleware (per-monitor counters preserved)
- [ ] Per-user escalation works through middleware (warning → timeout → ban)
- [ ] Audit trail flows through middleware `Auditor` to database
- [ ] Quick filter heuristics still work (skipping ~90% of messages before middleware)
- [ ] Discord bot tool calls go through middleware
- [ ] DryRun mode works through middleware policy (log without execute)
- [ ] Demo shows moderation decisions + audit trail through shared middleware

### Dependencies

- Milestone 1 (core engine)
- Milestone 5 (observability, for demo content)
