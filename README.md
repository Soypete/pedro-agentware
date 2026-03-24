# pedro-agentware

MCP-compatible middleware for enforcing policies on tool calls in agent frameworks. This repository contains both Go and Python implementations.

## Go Middleware

The Go middleware provides core policy enforcement for tool execution.

## Python Middleware

Python port of the Go middleware with LangGraph integration.

### Features

- **Policy Engine**: Enforce rate limits, max turns, iteration limits, and conditional rules
- **Auditor**: Track and audit all tool call decisions
- **LangGraph Integration**: Wrap LangGraph tool nodes with policy enforcement
- **Flexible Configuration**: Load policies from YAML files

### Installation

```bash
# Using uv (recommended)
uv pip install middleware-py

# Using pip
pip install middleware-py
```

### Quick Start

```python
from middleware_py import Middleware, Policy, Rule, Action, CallerContext
from middleware_py.langgraph import LangGraphToolWrapper

# Define a policy
policy = Policy(
    rules=[
        Rule(
            name="rate-limit-tools",
            tools=["*"],
            action=Action.ALLOW,
            max_rate={"count": 10, "window": 60},  # 10 calls per 60 seconds
        ),
    ],
    default_deny=False,
)

# Create middleware
middleware = Middleware(policy=policy)

# Wrap a LangGraph tool
wrapper = LangGraphToolWrapper(policy=policy)
wrapped_tool = wrapper.wrap(your_langgraph_tool_node)
```

### Architecture

#### Core Components

1. **Types** (`middleware_py.types`)
   - `Action`: Enum for allow/deny/filter actions
   - `ToolDefinition`: Tool metadata including name, description, input schema
   - `ToolResult`: Result of tool execution
   - `CallerContext`: Context about the caller (user, session, role, etc.)
   - `Decision`: Policy decision result

2. **Policy Engine** (`middleware_py.policy`)
   - `Policy`: Collection of rules
   - `Rule`: Individual policy rule with conditions, rate limits, etc.
   - `PolicyEvaluator`: Interface for evaluating policies
   - Supports operators: `eq`, `not_eq`, `contains`, `not_contains`, `matches`, `not_matches`, `exists`, `not_exists`

3. **Auditor** (`middleware_py.audit`)
   - `Auditor`: Interface for recording decisions
   - `InMemoryAuditor`: Thread-safe in-memory storage
   - `NoOpAuditor`: No-op implementation for testing

4. **Middleware** (`middleware_py.middleware`)
   - `Middleware`: Main middleware class wrapping tool executors
   - `CallHistory`: Tracks called/failed tools per session
   - Functional options pattern for configuration

5. **LangGraph Integration** (`middleware_py.langgraph`)
   - `LangGraphToolWrapper`: Wrap LangGraph tool nodes with policy enforcement

### Policy YAML Format

```yaml
rules:
  - name: "rate-limit-read"
    tools:
      - "read_file"
      - "search"
    action: "allow"
    max_rate:
      count: 5
      window: 60  # seconds
    conditions:
      - field: "caller.role"
        operator: "eq"
        value: "user"

  - name: "deny-admin-tools"
    tools:
      - "delete_database"
      - "drop_table"
    action: "deny"
    conditions:
      - field: "caller.trusted"
        operator: "eq"
        value: "false"

  - name: "filter-sensitive"
    tools:
      - "get_user"
    action: "filter"
    redact_fields:
      - "password"
      - "ssn"

default_deny: false
```

### Condition Operators

| Operator | Description |
|----------|-------------|
| `eq` | Field equals value |
| `not_eq` | Field does not equal value |
| `contains` | Field contains value |
| `not_contains` | Field does not contain value |
| `matches` | Field matches regex pattern |
| `not_matches` | Field does not match regex pattern |
| `exists` | Field exists (not nil) |
| `not_exists` | Field does not exist (nil) |
| `not` | Field is empty |

### Field Resolution

Conditions can reference:
- `caller.role` - Caller's role
- `caller.user_id` - User ID
- `caller.session_id` - Session ID
- `caller.source` - Call source
- `context.trusted` - Whether caller is trusted
- `args.<name>` - Tool argument values
- `context.<key>` - Custom context metadata

### LangGraph Integration

```python
from langgraph.prebuilt import ToolNode
from middleware_py.langgraph import LangGraphToolWrapper

# Create wrapper
wrapper = LangGraphToolWrapper(policy=your_policy)

# Wrap a ToolNode
tool_node = ToolNode([your_tool])
wrapped_node = wrapper.wrap_tool_node(tool_node)

# Or wrap a function directly
def my_tool(state):
    return {"result": "done"}

wrapped_my_tool = wrapper.wrap(my_tool)
```

### Development

```bash
# Install dependencies
uv pip install -e .

# Run tests
uv test

# Run with coverage
uv test --cov=middleware_py --cov-report=html

# Format code
ruff format .
```

## License

MIT