"""Tests for middleware_py.types."""

from middleware_py.middleware_types import (
    Action,
    CallerContext,
    ToolDefinition,
    ToolResult,
    ToolCall,
    Decision,
    RateLimitConfig,
    Condition,
)


class TestAction:
    def test_action_values(self):
        assert Action.ALLOW.value == "allow"
        assert Action.DENY.value == "deny"
        assert Action.FILTER.value == "filter"


class TestCallerContext:
    def test_default_context(self):
        ctx = CallerContext()
        assert ctx.user_id is None
        assert ctx.session_id is None
        assert ctx.trusted is False

    def test_get_method(self):
        ctx = CallerContext(user_id="user1", session_id="sess1", role="admin")
        assert ctx.get("user_id") == "user1"
        assert ctx.get("session_id") == "sess1"
        assert ctx.get("role") == "admin"
        assert ctx.get("nonexistent") is None
        assert ctx.get("nonexistent", "default") == "default"

    def test_metadata_access(self):
        ctx = CallerContext(metadata={"custom_key": "custom_value"})
        assert ctx.get("custom_key") == "custom_value"


class TestToolDefinition:
    def test_minimal_definition(self):
        tool = ToolDefinition(name="test_tool")
        assert tool.name == "test_tool"
        assert tool.description == ""
        assert tool.input_schema == {}

    def test_full_definition(self):
        tool = ToolDefinition(
            name="test_tool",
            description="A test tool",
            input_schema={"type": "object", "properties": {}},
        )
        assert tool.name == "test_tool"
        assert tool.description == "A test tool"


class TestToolResult:
    def test_successful_result(self):
        result = ToolResult(tool_name="test", success=True, result={"data": "value"})
        assert result.success is True
        assert result.result == {"data": "value"}
        assert result.error is None

    def test_failed_result(self):
        result = ToolResult(tool_name="test", success=False, error="Something went wrong")
        assert result.success is False
        assert result.error == "Something went wrong"


class TestToolCall:
    def test_minimal_tool_call(self):
        call = ToolCall(tool_name="test", args={})
        assert call.tool_name == "test"
        assert call.args == {}

    def test_tool_call_with_context(self):
        ctx = CallerContext(user_id="user1")
        call = ToolCall(tool_name="test", args={"arg1": "val1"}, caller_context=ctx)
        assert call.caller_context.user_id == "user1"


class TestDecision:
    def test_allow_decision(self):
        decision = Decision(action=Action.ALLOW)
        assert decision.action == Action.ALLOW

    def test_deny_with_message(self):
        decision = Decision(
            action=Action.DENY,
            rule_name="test_rule",
            message="Denied by policy",
        )
        assert decision.action == Action.DENY
        assert decision.rule_name == "test_rule"
        assert decision.message == "Denied by policy"


class TestCondition:
    def test_condition_creation(self):
        cond = Condition(field="caller.role", operator="eq", value="admin")
        assert cond.field == "caller.role"
        assert cond.operator == "eq"
        assert cond.value == "admin"


class TestRateLimitConfig:
    def test_rate_limit_config(self):
        config = RateLimitConfig(count=10, window=60)
        assert config.count == 10
        assert config.window == 60