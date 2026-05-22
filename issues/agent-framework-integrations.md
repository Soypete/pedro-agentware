# Research: Integration with Kitaru, Pydantic Agents, and Hermes Agent

## Summary

Research and plan integrations for agent-middleware with external agent frameworks and durable execution systems.

## Research Findings

### 1. Kitaru (Durable Execution)

**Location:** https://github.com/zenml-io/kitaru (Python, not Go)
**Stars:** N/A (new project)

**What it does:**
- Durable execution runtime for AI agents
- Provides `@flow` and `@checkpoint` decorators for persistent, replayable workflows
- Checkpoints persist outputs, enabling crash recovery and replay
- Built-in `wait()` for human-in-the-loop pauses
- Memory system for durable state across runs
- Deployments via versioned snapshots

**Key features:**
- Works with PydanticAI, OpenAI Agents SDK, or raw Python
- Self-hosted runtime layer
- KitaruAgent wrapper for PydanticAI integration

**Integration opportunity with pedro-agentware:**
- The middleware is Go-based, Kitaru is Python
- Could expose Go middleware as MCP server that wraps Python tools
- Kitaru's Python agents could call Go middleware tools via MCP
- Alternative: Create Go SDK for Kitaru (not currently available)

---

### 2. Pydantic Agents

**Location:** https://ai.pydantic.dev/agents
**Docs:** https://docs.pydantic.ai/ai/core-concepts/agent/

**What it does:**
- Type-safe agent framework for Python
- Generic `Agent[Deps, Output]` type system
- Built-in tool definitions with `@agent.tool` decorator
- Multiple run modes: `run()`, `run_sync()`, `run_stream()`, `run_stream_events()`, `iter()`
- Graph-based execution using pydantic-graph
- MCP client and server support
- Durable execution integrations: Temporal, DBOS, Prefect, Restate

**Key features:**
- Type safety via Pydantic models
- Multi-agent patterns support
- Streamed events and output
- Capabilities (reusable bundles of tools/hooks)
- Agent specs (load from config files)

**Integration opportunity:**
- Add as Python tool provider for Go middleware
- Wrap Pydantic agent as tool via MCP
- Use Pydantic AI's MCP server to expose tools to Go middleware

---

### 3. Hermes Agent (Nous Research)

**Location:** https://github.com/nousresearch/hermes-agent
**Stars:** 138k
**Language:** Python (88.5%), TypeScript (8.1%)

**What it does:**
- Self-improving AI agent built by Nous Research
- Built-in learning loop — creates skills from experience
- Improves skills during use
- Persistent memory and user modeling
- Multi-platform: Telegram, Discord, Slack, WhatsApp, Signal, Email
- 40+ built-in tools
- MCP integration support

**Key features:**
- Skill creation from successful task completions
- FTS5 session search with LLM summarization
- Cron scheduler for automated tasks
- Multiple terminal backends: local, Docker, SSH, Singularity, Modal, Daytona, Vercel Sandbox
- Works with many LLM providers: OpenRouter (200+ models), OpenAI, Anthropic, NVIDIA NIM, etc.

**Integration opportunity:**
- Wrap Hermes as MCP tool server for Go middleware
- Use Hermes as external agent for complex tasks
- Hermes skills could be exposed as tools through middleware

---

## Recommended Actions

### High Priority

1. **Add Pydantic Agents Python tool adapter**
   - Create Python package that wraps Pydantic agent as MCP tool server
   - Allows Go middleware to call Pydantic agents as tools
   - Similar to existing MCP client adapters

2. **Explore Hermes Agent integration**
   - Evaluate Hermes as external agent for complex workflows
   - Wrap Hermes skills as MCP tools
   - Consider Hermes as messaging gateway for agent middleware

3. **Explore Kitaru for durable execution**
   - Evaluate if Kitaru fits professor_pedro's workflow needs
   - Consider wrapping Kitaru flows as MCP tools
   - Potential Go SDK development if Kitaru proves valuable

### Medium Priority

4. **Add more durable execution adapters**
   - Explore Temporal SDK for Go
   - Evaluate DBOS (DB OS) for Go integration

---

## Architecture Notes

```
Python Agent (Kitaru/PydanticAI/Hermes) <--MCP--> Go Middleware (pedro-agentware)
                                                          |
                                                          v
                                                 Tool Execution (MCP/CLI/API)
```

The MCP protocol serves as the interoperability layer between Python agent frameworks and Go middleware.

---

## References

- Kitaru: https://github.com/zenml-io/kitaru
- Pydantic AI: https://ai.pydantic.dev/agents
- Pydantic AI MCP: https://docs.pydantic.ai/ai/mcp/overview/
- Hermes Agent: https://github.com/nousresearch/hermes-agent
- Hermes Docs: https://hermes-agent.nousresearch.com/docs/
- Claude Agent SDK (TS): https://github.com/anthropics/claude-agent-sdk-typescript