"""Tests for LangGraph integration."""

from unittest.mock import MagicMock

from middleware_py.middleware_types import Action, ToolResult
from middleware_py.middleware import Middleware
from middleware_py.policy import Policy, Rule
from middleware_py.langgraph import (
    create_middleware_tool,
    create_middleware_node,
    middleware_on,
    policy_decision_node,
    _extract_caller_context,
    _normalize_input,
)


class TestExtractCallerContext:
    """Tests for _extract_caller_context helper."""

    def test_empty_config(self):
        result = _extract_caller_context(None)
        assert result.user_id is None
        assert result.session_id is None

    def test_extract_from_config(self):
        config = {
            "configurable": {
                "user_id": "user123",
                "session_id": "sess456",
                "role": "admin",
                "source": "api",
                "trusted": True,
                "metadata": {"key": "value"},
            }
        }
        result = _extract_caller_context(config)
        assert result.user_id == "user123"
        assert result.session_id == "sess456"
        assert result.role == "admin"
        assert result.source == "api"
        assert result.trusted is True
        assert result.metadata == {"key": "value"}

    def test_missing_configurable(self):
        config = {"other": "data"}
        result = _extract_caller_context(config)
        assert result.user_id is None


class TestNormalizeInput:
    """Tests for _normalize_input helper."""

    def test_none_input(self):
        result = _normalize_input(None)
        assert result == {}

    def test_dict_input(self):
        result = _normalize_input({"key": "value"})
        assert result == {"key": "value"}

    def test_object_with_model_dump(self):
        obj = MagicMock()
        obj.model_dump.return_value = {"foo": "bar"}
        result = _normalize_input(obj)
        assert result == {"foo": "bar"}

    def test_object_with_dict(self):
        class HasDict:
            def dict(self):
                return {"baz": "qux"}
        obj = HasDict()
        result = _normalize_input(obj)
        assert result == {"baz": "qux"}

    def test_primitive_input(self):
        result = _normalize_input("string")
        assert result == {"input": "string"}


class TestCreateMiddlewareTool:
    """Tests for create_middleware_tool function."""

    def test_basic_tool_wrap(self):
        policy = Policy()
        middleware = Middleware(policy=policy)

        mock_tool = MagicMock()
        mock_tool.name = "test_tool"
        mock_tool.description = "A test tool"

        wrapped = create_middleware_tool(mock_tool, middleware, "test_tool")

        assert wrapped.name == "test_tool"
        assert wrapped.description == "A test tool"

    def test_tool_invoke_allowed(self):
        policy = Policy()
        middleware = Middleware(policy=policy)
        middleware.set_executor(lambda name, args: ToolResult(
            tool_name=name, success=True, result={"output": "ok"}
        ))

        mock_tool = MagicMock()
        wrapped = create_middleware_tool(mock_tool, middleware, "test_tool")

        result = wrapped.invoke({"arg1": "value1"})

        assert result == {"output": "ok"}

    def test_tool_invoke_denied(self):
        deny_rule = Rule(
            name="deny_test",
            tools=["test_tool"],
            conditions=[],
            action=Action.DENY,
        )
        policy = Policy(rules=[deny_rule])
        middleware = Middleware(policy=policy)

        mock_tool = MagicMock()
        wrapped = create_middleware_tool(mock_tool, middleware, "test_tool")

        result = wrapped.invoke({"arg1": "value1"})

        assert result == {"error": "Denied by policy", "success": False}

    def test_tool_invoke_with_config(self):
        policy = Policy()
        middleware = Middleware(policy=policy)
        middleware.set_executor(lambda name, args: ToolResult(
            tool_name=name, success=True, result=args
        ))

        mock_tool = MagicMock()
        wrapped = create_middleware_tool(mock_tool, middleware, "test_tool")

        config = {
            "configurable": {
                "user_id": "user123",
                "session_id": "sess456",
            }
        }

        result = wrapped.invoke({"input": "test"}, config)

        assert result == {"input": "test"}


class TestCreateMiddlewareNode:
    """Tests for create_middleware_node function."""

    def test_node_execution(self):
        policy = Policy()
        middleware = Middleware(policy=policy)
        middleware.set_executor(lambda name, args: ToolResult(
            tool_name=name, success=True, result={"executed": True}
        ))

        node = create_middleware_node(middleware, lambda n, a: None)

        state = {
            "tool_call": {"name": "test_tool", "args": {"key": "value"}},
            "user_id": "user123",
        }

        result = node(state)

        assert result["tool_success"] is True
        assert result["tool_result"] == {"executed": True}

    def test_node_no_tool_call(self):
        policy = Policy()
        middleware = Middleware(policy=policy)

        node = create_middleware_node(middleware, lambda n, a: None)

        result = node({})

        assert "error" in result


class TestMiddlewareOn:
    """Tests for middleware_on function."""

    def test_wrap_multiple_tools(self):
        policy = Policy()
        middleware = Middleware(policy=policy)

        tool1 = MagicMock()
        tool1.name = "tool1"
        tool1.description = "First tool"

        tool2 = MagicMock()
        tool2.name = "tool2"
        tool2.description = "Second tool"

        wrapped = middleware_on(middleware, [tool1, tool2], lambda n, a: None)

        assert len(wrapped) == 2
        assert wrapped[0].name == "tool1"
        assert wrapped[1].name == "tool2"


class TestPolicyDecisionNode:
    """Tests for policy_decision_node function."""

    def test_allow_decision(self):
        policy = Policy()
        middleware = Middleware(policy=policy)

        node = policy_decision_node(middleware)

        state = {
            "tool_call": {"name": "test_tool", "args": {}},
            "user_id": "user123",
        }

        result = node(state)

        assert result["decision"] == "allow"

    def test_deny_decision(self):
        deny_rule = Rule(
            name="deny_all",
            tools=["*"],
            conditions=[],
            action=Action.DENY,
        )
        policy = Policy(rules=[deny_rule], default_deny=True)
        middleware = Middleware(policy=policy)

        node = policy_decision_node(middleware)

        state = {
            "tool_call": {"name": "test_tool", "args": {}},
        }

        result = node(state)

        assert result["decision"] == "deny"
        assert result["decision_reason"] is None