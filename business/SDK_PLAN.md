# Agent Middleware SDK Plan

Three SDKs in order: **Go → Python → TypeScript**

The Go module is the core implementation (Milestone 1-2 from ENGINEERING_DESIGN.md). The Python and TypeScript SDKs are **clients** that talk to the middleware over MCP (stdio or HTTP+SSE). They don't reimplement the policy engine — they delegate to the Go server.

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│              Go Core (the engine)                │
│  policy engine, audit, context control           │
│  exposed as: Go module + MCP server              │
└──────────────┬──────────────────┬────────────────┘
               │ MCP (stdio)      │ MCP (HTTP+SSE)
               ▼                  ▼
┌──────────────────┐  ┌──────────────────────────┐
│   Python SDK     │  │   TypeScript SDK          │
│   pip install    │  │   npm install             │
│   MCP client     │  │   MCP client              │
│   + native API   │  │   + native API            │
└──────────────────┘  └──────────────────────────┘
```

Each SDK provides:
1. **MCP client** — talks to the Go middleware server over MCP protocol
2. **Native API** — idiomatic wrapper (`middleware.call_tool()`, not raw JSON-RPC)
3. **Framework integrations** — LangGraph, LangChain, CrewAI (Python); LangGraph.js, Vercel AI SDK (TypeScript)
4. **Policy helpers** — build/validate policy YAML programmatically

> **Note:** iam_pedro's `moderation_tools.go` provides a good reference implementation for Python/TS tool definition types. The two-tier tool selection pattern (`GetModerationToolDefinitions(extended bool)`) with 22 tool definitions, `ParseModerationToolCall()`, and `IsModerationTool()` demonstrates a mature tool registration API that the SDK type definitions should mirror.

The Go server runs as either:
- A **subprocess** (SDK spawns it, communicates via stdio MCP) — zero config
- A **sidecar** (already running, SDK connects via HTTP+SSE) — production deployments

---

## Phase 1: Go SDK (the core module)

**This IS the middleware.** Not a client — it's the engine.

### Deliverables

```
agent-middleware/
├── go.mod
├── types.go              # ToolDefinition, ToolResult, CallerContext
├── executor.go           # ToolExecutor interface
├── middleware.go          # Core enforcement (wraps any ToolExecutor)
├── policy/
│   ├── policy.go         # Policy, Rule, Condition types
│   ├── evaluator.go      # Decision logic
│   ├── loader.go         # YAML loading
│   └── matchers.go       # Condition matching (eq, contains, regex, not)
├── audit/
│   ├── audit.go          # Auditor interface + Decision type
│   ├── memory.go         # In-memory ring buffer
│   └── prometheus.go     # Metrics export
├── mcp/
│   ├── server.go         # MCP server (tools/list, tools/call)
│   ├── transport_stdio.go
│   ├── transport_http.go
│   └── protocol.go       # JSON-RPC types
└── cmd/
    └── agent-middleware/
        └── main.go       # Standalone MCP server binary
```

### Key Design Choices

- **Standalone binary** (`cmd/agent-middleware/main.go`) — Python/TS SDKs spawn this as a subprocess
- **Go module import** — Pedro repos import directly, skip MCP overhead
- **Policy hot-reload** — watch YAML file for changes, no restart needed
- **`cmd/agent-middleware` flags:**
  - `--policy policy.yaml` — policy file path
  - `--transport stdio|http` — MCP transport mode
  - `--upstream-mcp <cmd>` — upstream MCP server to wrap
  - `--upstream-http <url>` — upstream HTTP tool server
  - `--port 9090` — HTTP transport port
  - `--metrics-port 9091` — Prometheus metrics

### Milestones (from ENGINEERING_DESIGN.md)

- Milestone 1: Core engine + policy + audit
- Milestone 2: MCP server + standalone binary
- Milestone 3-4: Pedro repo integrations (direct Go import)

---

## Phase 2: Python SDK

### Why Python First

- LangGraph/LangChain ecosystem is Python-first
- PRD wedge use case: "General-purpose tool-calling agent (LangGraph-style)"
- Largest AI agent developer community
- Demo content targeting Python developers

### Deliverables

```
agent-middleware-python/
├── pyproject.toml
├── src/
│   └── agent_middleware/
│       ├── __init__.py
│       ├── client.py          # Core MCP client
│       ├── types.py           # ToolDefinition, ToolResult, CallerContext, Decision
│       ├── policy.py          # Policy builder (create/validate YAML programmatically)
│       ├── middleware.py       # High-level wrapper (spawn Go binary, call tools)
│       ├── exceptions.py      # ToolDeniedError, PolicyViolation, etc.
│       │
│       ├── integrations/
│       │   ├── __init__.py
│       │   ├── langgraph.py   # LangGraph tool node wrapper
│       │   ├── langchain.py   # LangChain tool wrapper
│       │   └── crewai.py      # CrewAI tool wrapper
│       │
│       └── transports/
│           ├── __init__.py
│           ├── stdio.py       # Spawn Go binary, communicate via stdio
│           └── http.py        # Connect to running Go server via HTTP+SSE
│
├── tests/
│   ├── test_client.py
│   ├── test_policy.py
│   ├── test_middleware.py
│   └── test_integrations/
│       ├── test_langgraph.py
│       └── test_langchain.py
│
├── examples/
│   ├── basic_usage.py
│   ├── langgraph_safe_agent.py    # The demo from the PRD
│   ├── langchain_constrained.py
│   └── injection_prevention.py
│
└── bin/                           # Bundled Go binary (optional, for pip install)
    └── .gitkeep
```

### SDK Milestones

#### P2-M1: Core Client

**Goal:** Python can talk to the Go middleware over MCP.

- [ ] MCP JSON-RPC client (stdio transport)
- [ ] `MiddlewareClient` class — `call_tool(name, args)`, `list_tools()`
- [ ] Auto-spawn Go binary from bundled or PATH
- [ ] `ToolResult` and `Decision` dataclasses
- [ ] `ToolDeniedError` exception raised on deny
- [ ] `CallerContext` passed via MCP request metadata
- [ ] Unit tests with mock MCP server

```python
from agent_middleware import Middleware

mw = Middleware(policy="policy.yaml")  # spawns Go binary

result = mw.call_tool("web_search", {"query": "python packaging"})
# or
result = mw.call_tool(
    "bash", {"command": "rm -rf /"},
    context=CallerContext(role="user", trusted=False)
)
# raises ToolDeniedError: "blocked by rule: block-dangerous-bash"
```

#### P2-M2: LangGraph Integration

**Goal:** Drop-in middleware for LangGraph tool nodes.

- [ ] `SafeToolNode` — wraps LangGraph `ToolNode` with middleware enforcement
- [ ] `constrained_tools()` — filters tool list based on policy
- [ ] Works with LangGraph's `StateGraph` and `MessageGraph`
- [ ] Example: safe agent that blocks injection + unsafe tools

```python
from langgraph.prebuilt import ToolNode
from agent_middleware.integrations.langgraph import SafeToolNode

# Before (unsafe):
tool_node = ToolNode(tools)

# After (constrained):
tool_node = SafeToolNode(tools, policy="policy.yaml")

graph = StateGraph(...)
graph.add_node("tools", tool_node)
```

#### P2-M3: LangChain Integration + HTTP Transport

**Goal:** LangChain wrapper + production-ready HTTP transport.

- [ ] `ConstrainedTool` — wraps any LangChain `BaseTool` with middleware
- [ ] `ConstrainedToolkit` — wraps `BaseToolkit`
- [ ] HTTP+SSE transport client (connect to running Go sidecar)
- [ ] Connection pooling and retry logic
- [ ] Health check support

```python
from langchain.tools import BaseTool
from agent_middleware.integrations.langchain import ConstrainedTool

safe_tool = ConstrainedTool(
    tool=my_search_tool,
    policy="policy.yaml"
)
```

#### P2-M4: Policy Builder + CrewAI

**Goal:** Programmatic policy creation + CrewAI integration.

- [ ] `PolicyBuilder` — fluent API to create policies in Python
- [ ] Policy validation (check for conflicts, unreachable rules)
- [ ] Export to YAML
- [ ] CrewAI tool wrapper

```python
from agent_middleware.policy import PolicyBuilder

policy = (
    PolicyBuilder()
    .deny("bash", when={"args.command": {"matches": "rm -rf|DROP TABLE"}})
    .allow("web_search", rate_limit={"count": 10, "window": "1m"})
    .filter("*", redact_fields=["password", "api_key"])
    .build()
)
policy.save("policy.yaml")
```

### Binary Distribution Strategy

The Python SDK needs the Go binary. Options:

1. **PATH lookup** (default) — user installs Go binary separately (`go install github.com/pedro/agent-middleware/cmd/agent-middleware@latest`)
2. **Bundled binaries** (pip install) — pre-compiled for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
3. **Docker** — `docker run ghcr.io/pedro/agent-middleware --policy ...`

For MVP: PATH lookup. Phase 2: bundled binaries via platform-specific wheels.

### Python Packaging

- **Build:** `hatchling` or `setuptools`
- **Publish:** PyPI (`pip install agent-middleware`)
- **Python version:** 3.10+
- **Dependencies (minimal):** `pydantic` (types), `pyyaml` (policy), `httpx` (HTTP transport)
- **Optional deps:** `langgraph`, `langchain-core`, `crewai` (framework integrations)

---

## Phase 3: TypeScript SDK

### Why TypeScript Third

- Growing AI agent ecosystem (Vercel AI SDK, LangGraph.js)
- Web-first developers building agent UIs
- professor_pedro's React frontend could use it
- Complements Python SDK for full market coverage

### Deliverables

```
agent-middleware-ts/
├── package.json
├── tsconfig.json
├── src/
│   ├── index.ts
│   ├── client.ts              # Core MCP client
│   ├── types.ts               # ToolDefinition, ToolResult, CallerContext, Decision
│   ├── policy.ts              # Policy builder
│   ├── middleware.ts           # High-level wrapper
│   ├── errors.ts              # ToolDeniedError, etc.
│   │
│   ├── integrations/
│   │   ├── langgraph.ts       # LangGraph.js tool wrapper
│   │   ├── vercel-ai.ts       # Vercel AI SDK tool wrapper
│   │   └── mastra.ts          # Mastra framework wrapper
│   │
│   └── transports/
│       ├── stdio.ts           # Spawn Go binary via child_process
│       └── http.ts            # HTTP+SSE client (fetch/EventSource)
│
├── tests/
│   ├── client.test.ts
│   ├── policy.test.ts
│   └── integrations/
│       └── langgraph.test.ts
│
├── examples/
│   ├── basic-usage.ts
│   ├── langgraph-safe-agent.ts
│   ├── vercel-ai-constrained.ts
│   └── nextjs-api-route.ts       # Server-side enforcement in Next.js
│
└── bin/                           # Bundled Go binary (optional)
    └── .gitkeep
```

### SDK Milestones

#### P3-M1: Core Client

**Goal:** TypeScript can talk to the Go middleware over MCP.

- [ ] MCP JSON-RPC client (stdio transport via `child_process.spawn`)
- [ ] `MiddlewareClient` class — `callTool(name, args)`, `listTools()`
- [ ] Full TypeScript types for all middleware concepts
- [ ] `ToolDeniedError` thrown on deny
- [ ] Works in Node.js (not browser — Go binary is server-side)
- [ ] Unit tests with mock MCP server

```typescript
import { Middleware } from 'agent-middleware';

const mw = new Middleware({ policy: 'policy.yaml' });

const result = await mw.callTool('web_search', { query: 'typescript agents' });

// throws ToolDeniedError
await mw.callTool('bash', { command: 'rm -rf /' }, {
  context: { role: 'user', trusted: false }
});
```

#### P3-M2: LangGraph.js + Vercel AI SDK Integration

**Goal:** Framework integrations for the two biggest TS agent ecosystems.

- [ ] `createSafeToolNode()` — LangGraph.js wrapper
- [ ] `constrainedTool()` — Vercel AI SDK `tool()` wrapper
- [ ] HTTP+SSE transport (for serverless/edge — can't spawn subprocesses)
- [ ] Example: Next.js API route with constrained agent

```typescript
// LangGraph.js
import { createSafeToolNode } from 'agent-middleware/integrations/langgraph';

const toolNode = createSafeToolNode(tools, { policy: 'policy.yaml' });

// Vercel AI SDK
import { constrainedTool } from 'agent-middleware/integrations/vercel-ai';

const safeTool = constrainedTool(myTool, { policy: 'policy.yaml' });
```

#### P3-M3: Policy Builder + Browser-Compatible Client

**Goal:** Programmatic policies + HTTP-only client for serverless.

- [ ] `PolicyBuilder` — fluent TypeScript API
- [ ] HTTP-only client (no subprocess, connects to Go sidecar)
- [ ] Works in Cloudflare Workers, Vercel Edge, Deno
- [ ] Mastra framework integration

```typescript
import { PolicyBuilder } from 'agent-middleware/policy';

const policy = new PolicyBuilder()
  .deny('bash', { when: { 'args.command': { matches: /rm -rf|DROP TABLE/ } } })
  .allow('web_search', { rateLimit: { count: 10, window: '1m' } })
  .filter('*', { redactFields: ['password', 'api_key'] })
  .build();
```

### TypeScript Packaging

- **Build:** `tsup` (ESM + CJS dual output)
- **Publish:** npm (`npm install agent-middleware`)
- **Node version:** 18+
- **Dependencies (minimal):** none for core (just `child_process` and `fetch`)
- **Optional deps:** `@langchain/langgraph`, `ai` (Vercel AI SDK)

---

## Binary Distribution (Cross-SDK)

All SDKs need the Go binary. Centralized build:

```
agent-middleware/              # Go repo
├── .goreleaser.yml            # Cross-compile for all platforms
├── cmd/agent-middleware/
│   └── main.go               # Standalone binary
└── dist/                      # Release artifacts
    ├── agent-middleware_linux_amd64
    ├── agent-middleware_linux_arm64
    ├── agent-middleware_darwin_amd64
    ├── agent-middleware_darwin_arm64
    └── agent-middleware_windows_amd64.exe
```

**Distribution channels:**
1. `go install` — Go developers
2. GitHub Releases — manual download
3. Homebrew — `brew install agent-middleware`
4. PyPI platform wheels — bundled in Python package
5. npm optional dependency — bundled in TS package
6. Docker — `ghcr.io/pedro/agent-middleware`

---

## Full Timeline (All Phases)

```
Phase 1: Go SDK (IS the middleware)
├── M1: Core engine + policy + audit
├── M2: MCP server + standalone binary (agent-middleware CLI)
├── M3: PedroCLI integration (Go import)
└── M4: professor_pedro integration (Go import)

Phase 2: Python SDK
├── P2-M1: Core MCP client + stdio transport
├── P2-M2: LangGraph integration (THE demo)
├── P2-M3: LangChain integration + HTTP transport
└── P2-M4: Policy builder + CrewAI

Phase 3: TypeScript SDK
├── P3-M1: Core MCP client + stdio transport
├── P3-M2: LangGraph.js + Vercel AI SDK integration
└── P3-M3: Policy builder + HTTP-only client (serverless)

Phase 4: Content & GTM (runs parallel to Phase 2-3)
├── Demo repo: LangGraph agent (Python) with 3 failure modes
├── YouTube: "Your AI agent is unsafe" series
├── Blog: technical walkthrough posts
└── iam_pedro integration (Go import, Milestone 6)
```

---

## Repo Structure (4 repos total)

| Repo | Language | What | Package |
|------|----------|------|---------|
| `agent-middleware` | Go | Core engine + MCP server + CLI binary | `go get`, `brew`, Docker |
| `agent-middleware-python` | Python | MCP client + LangGraph/LangChain/CrewAI | `pip install agent-middleware` |
| `agent-middleware-ts` | TypeScript | MCP client + LangGraph.js/Vercel AI | `npm install agent-middleware` |
| `agent-middleware-demo` | Python + TS | Example agents + failure mode demos | Not published, clone to run |

---

## What's Shared via MCP Protocol (Not Code)

The SDKs share **zero code**. They share the **MCP protocol contract**:

```
tools/list → returns filtered ToolDefinition[]
tools/call → enforces policy, returns ToolResult or error
```

This means:
- Go changes don't break Python or TypeScript
- SDKs can be versioned independently
- Any MCP-compatible client works (even ones we don't build)
- The Go binary is the only thing that evaluates policies

The SDKs are thin clients with framework-specific wrappers. The intelligence lives in Go.
