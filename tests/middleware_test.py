"""Tests for middleware module."""

from middleware_py.middleware import Middleware, with_auditor, with_tools, CallHistory
from middleware_py.types import Action, CallerContext, ToolDefinition, ToolResult
from middleware_py.policy import Policy, Rule
from middleware_py.audit import InMemoryAuditor


def mock_executor(tool_name: str, args: dict) -> ToolResult:
    """Mock executor for testing."""
    if tool_name == "fail_tool":
        return ToolResult(tool_name=tool_name, success=False, error="mock error")
    return ToolResult(tool_name=tool_name, success=True, result={"output": "ok"})


def test_middleware_init():
    """Test middleware initialization."""
    policy = Policy(rules=[Rule(name="allow-all", tools=["*"], action=Action.ALLOW)])
    mw = Middleware(executor=mock_executor, policy=policy)

    assert mw is not None
    assert mw._executor == mock_executor
    assert mw._policy is not None


def test_middleware_call_allowed():
    """Test tool call that is allowed by policy."""
    policy = Policy(rules=[Rule(name="allow-tool1", tools=["tool1"], action=Action.ALLOW)])
    mw = Middleware(executor=mock_executor, policy=policy)

    result = mw.call("tool1", {})

    assert result.success is True
    assert result.tool_name == "tool1"


def test_middleware_call_denied():
    """Test tool call that is denied by policy."""
    policy = Policy(rules=[Rule(name="deny-tool1", tools=["tool1"], action=Action.DENY)], default_deny=False)
    mw = Middleware(executor=mock_executor, policy=policy)

    result = mw.call("tool1", {})

    assert result.success is False
    assert "Denied by policy" in result.error


def test_middleware_call_executor_error():
    """Test tool call that fails in executor."""
    policy = Policy(rules=[Rule(name="allow-fail", tools=["fail_tool"], action=Action.ALLOW)])
    mw = Middleware(executor=mock_executor, policy=policy)

    result = mw.call("fail_tool", {})

    assert result.success is False
    assert result.error == "mock error"


def test_middleware_no_executor():
    """Test middleware without executor configured."""
    policy = Policy(rules=[Rule(name="allow-tool1", tools=["tool1"], action=Action.ALLOW)])
    mw = Middleware(policy=policy)

    result = mw.call("tool1", {})

    assert result.success is False
    assert "No executor configured" in result.error


def test_middleware_set_executor():
    """Test setting executor after initialization."""
    mw = Middleware()
    mw.set_executor(mock_executor)

    result = mw.call("tool1", {})
    assert result.success is True


def test_middleware_set_policy():
    """Test setting policy after initialization."""
    mw = Middleware()
    policy = Policy(rules=[Rule(name="allow-all", tools=["*"], action=Action.ALLOW)])
    mw.set_policy(policy)

    assert mw._policy == policy


def test_middleware_filter_tools():
    """Test filtering tools based on policy."""
    tools = [ToolDefinition(name="tool1"), ToolDefinition(name="tool2"), ToolDefinition(name="dangerous")]
    policy = Policy(
        rules=[Rule(name="allow-safe", tools=["tool1", "tool2"], action=Action.ALLOW)],
        default_deny=True,
    )
    mw = Middleware(policy=policy, options=[with_tools(tools)])

    allowed = mw.filter_tools(CallerContext())

    assert len(allowed) == 2
    tool_names = [t.name for t in allowed]
    assert "tool1" in tool_names
    assert "tool2" in tool_names
    assert "dangerous" not in tool_names


def test_middleware_list_tools():
    """Test listing tools."""
    tools = [ToolDefinition(name="tool1"), ToolDefinition(name="tool2")]
    mw = Middleware(options=[with_tools(tools)])

    listed = mw.list_tools()

    assert len(listed) == 2


def test_middleware_with_auditor_option():
    """Test with_auditor option."""
    auditor = InMemoryAuditor()
    mw = Middleware(options=[with_auditor(auditor)])

    assert mw._auditor == auditor


def test_middleware_with_tools_option():
    """Test with_tools option."""
    tools = [ToolDefinition(name="tool1")]
    mw = Middleware(options=[with_tools(tools)])

    assert "tool1" in mw._tools


def test_middleware_call_history():
    """Test call history recording."""
    policy = Policy(rules=[Rule(name="allow-all", tools=["*"], action=Action.ALLOW)])
    mw = Middleware(executor=mock_executor, policy=policy)

    mw.call("tool1", {})
    mw.call("tool2", {}, CallerContext(session_id="sess1"))

    called, failed = mw.get_history("default")
    assert "tool1" in called

    called, failed = mw.get_history("sess1")
    assert "tool2" in called


def test_middleware_call_history_failure():
    """Test call history records failures."""
    policy = Policy(rules=[Rule(name="allow-fail", tools=["fail_tool"], action=Action.ALLOW)])
    mw = Middleware(executor=mock_executor, policy=policy)

    mw.call("fail_tool", {})

    called, failed = mw.get_history("default")
    assert "fail_tool" in failed


def test_middleware_clear_history():
    """Test clearing history."""
    policy = Policy(rules=[Rule(name="allow-all", tools=["*"], action=Action.ALLOW)])
    mw = Middleware(executor=mock_executor, policy=policy)

    mw.call("tool1", {})
    mw.clear_history()

    called, failed = mw.get_history("default")
    assert len(called) == 0


def test_middleware_filter():
    """Test filter action with redaction."""
    policy = Policy(
        rules=[
            Rule(
                name="filter-sensitive",
                tools=["secret_tool"],
                action=Action.FILTER,
                redact_fields=["password"],
            )
        ]
    )
    mw = Middleware(executor=mock_executor, policy=policy)

    result = mw.call("secret_tool", {}, CallerContext())

    assert result.success is True
    assert result.result is not None


def test_call_history_record():
    """Test CallHistory.record()."""
    ch = CallHistory()

    ch.record("sess1", "tool1", True)
    ch.record("sess1", "tool2", False)

    called = ch.get_called("sess1")
    failed = ch.get_failed("sess1")

    assert "tool1" in called
    assert "tool2" in failed


def test_call_history_clear():
    """Test CallHistory.clear()."""
    ch = CallHistory()

    ch.record("sess1", "tool1", True)
    ch.record("sess2", "tool2", True)

    ch.clear("sess1")

    assert ch.get_called("sess1") == []
    assert "tool2" in ch.get_called("sess2")

    ch.clear()

    assert ch.get_called("sess2") == []


def test_call_history_empty_session():
    """Test CallHistory with empty session returns empty lists."""
    ch = CallHistory()

    called = ch.get_called("nonexistent")
    failed = ch.get_failed("nonexistent")

    assert called == []
    assert failed == []