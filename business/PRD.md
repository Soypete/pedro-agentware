# Agent Middleware PRD (MVP)

## Product Name (Working)

**Agent Middleware**
(*placeholder - can refine later*)

---

## 1. Product Overview

### Summary

Agent middleware that enforces contracts on context, tools, and data - making agents production-ready.

### Why This Exists

AI agents today are:

* unpredictable
* over-permissioned
* unsafe to connect to real systems

The problem is not model capability.

> The problem is lack of enforcement.

This product introduces a **constraint-enforced runtime** that ensures:

* agents only see what they should
* agents only do what they are allowed
* agents cannot be manipulated by untrusted input

### Core Outcome

> Developers can confidently connect agents to real systems - because those agents are production-ready.

---

## 2. Problem

### Problem Statement

AI agents lack enforceable boundaries.

They:

* mix instructions with untrusted data
* have unrestricted access to tools
* operate without permission constraints
* expose more data than necessary

### Failure Modes

1. **Prompt Injection** - Untrusted input overrides behavior.
2. **Unsafe Tool Execution** - Agents perform actions outside intended limits.
3. **Over-Scoped Data** - Agents access more data than required (data leakage risk).

### Why Existing Tools Fail

| Tool | Does | Doesn't |
|------|------|---------|
| Agent Frameworks (LangGraph, etc.) | Orchestrate workflows | Enforce constraints |
| Tool-Calling Systems (MCP, etc.) | Expose tools | Govern access or behavior |
| Prompt Engineering | Suggest behavior | Enforce anything |

### Root Problem

> There is no runtime that enforces how agents access data, use tools, and behave.

---

## 3. User & ICP

### Primary User

**AI / Backend / Fullstack Engineers** building agents, integrating tools, experimenting quickly.

> Mindset: "This works... but I don't trust it."

### ICP - Phase 1 (MVP)

Indie builders / early adopters building agent workflows, experimenting with LangGraph or similar, comfortable with OSS.

### ICP - Phase 2

Startups to Enterprise - platform teams, security-conscious orgs, internal AI adoption.

---

## 4. Use Case Wedge

### Wedge

> General-purpose tool-calling agent (LangGraph-style)

### Demo Structure: "Unsafe Agent -> Production-Ready Agent"

1. **Prompt Injection** - untrusted input attempts override -> middleware blocks
2. **Unsafe Tool Calls** - agent attempts invalid action -> middleware blocks
3. **Over-Scoped Data** - agent retrieves excess data -> middleware restricts

### Teaching Outcome

> Developers learn how to build production-ready agents - not just working ones.

---

## 5. Product Definition

### Core Experience

| Without Product | With Product |
|----------------|-------------|
| Unsafe | Constrained behavior |
| Unpredictable | Scoped access |
| Not production-ready | Reliable execution |

### Core Capabilities

- **Context Control** - Separates trusted vs untrusted input, prevents injection
- **Tool Control** - Restricts tool access, enforces safe usage
- **Data Control** - Limits visible data, prevents overexposure

### Runtime Behavior

- Evaluates actions
- Enforces constraints
- Allows / blocks decisions

### Observability

- Shows violations
- Explains behavior
- Supports debugging

### MVP Scope

**Included:** runtime enforcement, constraint validation, violation visibility

**Not Included:** control plane, enterprise features, advanced policy systems

---

## 6. Go-To-Market (GTM)

### Strategy

> Education-led, demo-driven, OSS distribution

### GTM Loop

1. Developer sees content
2. Sees agent break
3. Sees fix
4. Tries repo
5. Gets "aha"
6. Shares / adopts

### Distribution

- **Primary:** GitHub, YouTube, Substack
- **Secondary:** LinkedIn / Twitter, Discord

### Monetization Path

1. **Phase 1 - OSS:** adoption, feedback
2. **Phase 2 - Control Plane:** visibility, policy management
3. **Phase 3 - Advanced:** efficiency, long-running agents

---

## Final Positioning

> The missing layer that makes agents production-ready.
