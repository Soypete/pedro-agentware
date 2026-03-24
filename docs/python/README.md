# Python Middleware Usage Examples

This document provides examples of how to use the Python middleware for policy enforcement.

## Installation

```bash
pip install middleware-py
```

## Basic Usage

### Creating a Policy

```python
from middleware_py import Policy, Rule, Action, RateLimitConfig

policy = Policy(
    default_deny=False,
    rules=[
        Rule(
            name="rate-limit-tools",
            tools=["*"],
            action=Action.ALLOW,
            max_rate=RateLimitConfig(count=10, window=60),
        ),
        Rule(
            name="deny-admin",
            tools=["delete_database", "drop_table"],
            action=Action.DENY,
            conditions=[
                {
                    "field": "caller.trusted",
                    "operator": "eq",
                    "value": False,
                }
            ],
        ),
    ],
)
```

### Creating Middleware

```python
from middleware_py import Middleware, CallerContext

def my_tool_executor(tool_name: str, args: dict) -> ToolResult:
    # Your tool execution logic here
    return ToolResult(
        tool_name=tool_name,
        success=True,
        result={"output": f"Executed {tool_name}"},
    )

# Create middleware
mw = Middleware(
    executor=my_tool_executor,
    policy=policy,
)

# Call a tool through middleware
result = mw.call("read_file", {"path": "/tmp/test.txt"})
```

### Using Caller Context

```python
from middleware_py import CallerContext

# Create caller context with user information
caller_ctx = CallerContext(
    trusted=True,
    role="user",
    user_id="user-123",
    session_id="session-456",
    source="cli",
)

# Call tool with caller context
result = mw.call("read_file", {"path": "/tmp/test.txt"}, caller_context=caller_ctx)
```

### Using Audit

```python
from middleware_py import InMemoryAuditor

# Create in-memory auditor
auditor = InMemoryAuditor()

# Configure middleware with auditor
mw = Middleware(
    executor=my_tool_executor,
    policy=policy,
    options=[with_auditor(auditor)],
)

# After tool calls, get audit log
log = auditor.get_log()
for entry in log:
    print(f"Decision: {entry.decision.action}, Tool: {entry.tool_call.tool_name}")
```

## Loading Policy from YAML

```python
from middleware_py import load_policy_from_file

# Load policy from YAML file
policy = load_policy_from_file("policy.yaml")

mw = Middleware(executor=my_tool_executor, policy=policy)
```

Example `policy.yaml`:

```yaml
rules:
  - name: "rate-limit-read"
    tools:
      - "read_file"
      - "search"
    action: "allow"
    max_rate:
      count: 5
      window: 60

  - name: "deny-admin-tools"
    tools:
      - "delete_database"
    action: "deny"
    conditions:
      - field: "caller.trusted"
        operator: "eq"
        value: false

default_deny: false
```

## Filtering Tool List

```python
# Get list of allowed tools for a caller
tools = mw.filter_tools(caller_ctx)
for tool in tools:
    print(tool.name)
```

## Condition Operators

| Operator | Description |
|----------|-------------|
| `eq` | Field equals value |
| `not_eq` | Field does not equal value |
| `contains` | Field contains value |
| `not_contains` | Field does not contain value |
| `matches` | Field matches regex pattern |
| `not_matches` | Field does not match regex pattern |
| `exists` | Field exists |
| `not_exists` | Field does not exist |
| `not` | Field is empty |

## Field Resolution

Conditions can reference:
- `caller.role` - Caller's role
- `caller.user_id` - User ID
- `caller.session_id` - Session ID
- `caller.source` - Call source
- `caller.trusted` - Whether caller is trusted
- `args.<name>` - Tool argument values
- `context.<key>` - Custom context metadata

## LangGraph Integration

The Python middleware includes LangGraph integration for use with LangChain/LangGraph agents.

### Wrapping LangGraph Tools

```python
from middleware_py import Middleware
from middleware_py.langgraph import create_middleware_tool
from langgraph.prebuilt import ToolNode

# Create your LangGraph tools
def my_tool(input: str) -> str:
    return f"Processed: {input}"

# Wrap with middleware
wrapped_tool = create_middleware_tool(
    tool_runnable=my_tool,
    middleware=mw,
    tool_name="my_tool",
    tool_description="My custom tool",
)

# Use in LangGraph
result = wrapped_tool.invoke({"input": "hello"})
```

### Using Middleware Nodes in LangGraph

```python
from middleware_py.langgraph import create_middleware_node, policy_decision_node
from langgraph.graph import StateGraph

# Create a middleware node
middleware_node = create_middleware_node(
    middleware=mw,
    tool_executor=my_tool_executor,
    node_name="enforce_policy",
)

# Create a policy decision node (without execution)
decision_node = policy_decision_node(
    middleware=mw,
    tool_call_key="tool_call",
)

# Build graph
graph = StateGraph(dict)
graph.add_node("check_policy", decision_node)
graph.add_node("execute", middleware_node)
```

### Applying Middleware to Multiple Tools

```python
from middleware_py.langgraph import middleware_on

# List of LangGraph tools
langgraph_tools = [tool1, tool2, tool3]

# Apply middleware to all tools
wrapped_tools = middleware_on(
    middleware=mw,
    tools=langgraph_tools,
    tool_executor=my_tool_executor,
)

# Now each tool enforces policies before execution
for tool in wrapped_tools:
    result = tool.invoke({"input": "test"})
```

### Using with LangChain Agents

```python
from langchain.agents import AgentExecutor
from middleware_py.langgraph import create_middleware_tool

# Create tools
tools = [create_middleware_tool(t, mw, tool_name=t.name) for t in langgraph_tools]

# Create agent with middleware-wrapped tools
agent = AgentExecutor.from_agent_and_tools(
    agent=agent,
    tools=tools,
)
```

## API Reference

### Core Classes

- `Middleware` - Main middleware class for policy enforcement
- `Policy` - Policy container with rules
- `Rule` - Individual policy rule
- `CallerContext` - Context about the caller
- `ToolCall` - Represents a tool call request
- `ToolResult` - Represents a tool execution result

### Auditors

- `InMemoryAuditor` - Stores audit logs in memory
- `NoOpAuditor` - No-op auditor for performance

### Utilities

- `load_policy_from_file(path)` - Load policy from YAML file
- `load_policy_from_yaml(yaml_string)` - Load policy from YAML string

### LangGraph Integration

- `create_middleware_tool()` - Wrap a LangGraph tool with middleware
- `create_middleware_node()` - Create a LangGraph node with middleware
- `policy_decision_node()` - Create a policy-only decision node
- `middleware_on()` - Apply middleware to multiple tools