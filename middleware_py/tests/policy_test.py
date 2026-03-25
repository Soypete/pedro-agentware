"""Tests for middleware_py.policy."""

from middleware_py.policy import (
    Policy,
    PolicyEvaluator,
    Rule,
    match_operator,
)
from middleware_py.middleware_types import (
    Action,
    CallerContext,
    Condition,
    RateLimitConfig,
    ToolCall,
)


class TestRule:
    def test_matches_tool_exact(self):
        rule = Rule(name="test", tools=["tool1", "tool2"], action=Action.ALLOW)
        assert rule.matches_tool("tool1") is True
        assert rule.matches_tool("tool2") is True
        assert rule.matches_tool("tool3") is False

    def test_matches_tool_wildcard(self):
        rule = Rule(name="test", tools=["*"], action=Action.ALLOW)
        assert rule.matches_tool("any_tool") is True
        assert rule.matches_tool("another_tool") is True


class TestPolicy:
    def test_get_rules_for_tool(self):
        rule1 = Rule(name="rule1", tools=["tool1"], action=Action.ALLOW)
        rule2 = Rule(name="rule2", tools=["tool2"], action=Action.DENY)
        rule3 = Rule(name="rule3", tools=["*"], action=Action.ALLOW)

        policy = Policy(rules=[rule1, rule2, rule3])

        rules_for_tool1 = policy.get_rules_for_tool("tool1")
        assert len(rules_for_tool1) == 2
        assert rule1 in rules_for_tool1
        assert rule3 in rules_for_tool1

        rules_for_tool2 = policy.get_rules_for_tool("tool2")
        assert len(rules_for_tool2) == 2
        assert rule2 in rules_for_tool2

        rules_for_tool3 = policy.get_rules_for_tool("tool3")
        assert len(rules_for_tool3) == 1
        assert rule3 in rules_for_tool3


class TestPolicyEvaluator:
    def test_allow_by_default(self):
        policy = Policy()
        evaluator = PolicyEvaluator(policy)

        call = ToolCall(tool_name="any_tool", args={}, caller_context=CallerContext())
        decision = evaluator.evaluate(call)

        assert decision.action == Action.ALLOW

    def test_default_deny(self):
        policy = Policy(default_deny=True)
        evaluator = PolicyEvaluator(policy)

        call = ToolCall(tool_name="any_tool", args={}, caller_context=CallerContext())
        decision = evaluator.evaluate(call)

        assert decision.action == Action.DENY

    def test_deny_rule(self):
        rule = Rule(name="deny_all", tools=["*"], action=Action.DENY)
        policy = Policy(rules=[rule])
        evaluator = PolicyEvaluator(policy)

        call = ToolCall(tool_name="any_tool", args={}, caller_context=CallerContext())
        decision = evaluator.evaluate(call)

        assert decision.action == Action.DENY
        assert decision.rule_name == "deny_all"

    def test_conditional_rule_match(self):
        rule = Rule(
            name="admin_only",
            tools=["*"],
            action=Action.DENY,
            conditions=[Condition(field="caller.role", operator="eq", value="admin")],
        )
        policy = Policy(rules=[rule])
        evaluator = PolicyEvaluator(policy)

        call_admin = ToolCall(
            tool_name="any_tool",
            args={},
            caller_context=CallerContext(role="admin"),
        )
        decision_admin = evaluator.evaluate(call_admin)
        assert decision_admin.action == Action.DENY

        call_user = ToolCall(
            tool_name="any_tool",
            args={},
            caller_context=CallerContext(role="user"),
        )
        decision_user = evaluator.evaluate(call_user)
        assert decision_user.action == Action.ALLOW

    def test_rate_limiting(self):
        rule = Rule(
            name="rate_limited",
            tools=["tool1"],
            action=Action.ALLOW,
            max_rate=RateLimitConfig(count=2, window=60),
        )
        policy = Policy(rules=[rule])
        evaluator = PolicyEvaluator(policy)

        ctx = CallerContext(session_id="session1")

        call1 = ToolCall(tool_name="tool1", args={}, caller_context=ctx)
        decision1 = evaluator.evaluate(call1)
        assert decision1.action == Action.ALLOW

        call2 = ToolCall(tool_name="tool1", args={}, caller_context=ctx)
        decision2 = evaluator.evaluate(call2)
        assert decision2.action == Action.ALLOW

        call3 = ToolCall(tool_name="tool1", args={}, caller_context=ctx)
        decision3 = evaluator.evaluate(call3)
        assert decision3.action == Action.DENY

    def test_max_turns(self):
        rule = Rule(
            name="max_turns",
            tools=["*"],
            action=Action.ALLOW,
            max_turns=2,
        )
        policy = Policy(rules=[rule])
        evaluator = PolicyEvaluator(policy)

        ctx = CallerContext(session_id="session1")

        call1 = ToolCall(tool_name="tool1", args={}, caller_context=ctx)
        decision1 = evaluator.evaluate(call1)
        assert decision1.action == Action.ALLOW

        call2 = ToolCall(tool_name="tool2", args={}, caller_context=ctx)
        decision2 = evaluator.evaluate(call2)
        assert decision2.action == Action.ALLOW

        call3 = ToolCall(tool_name="tool3", args={}, caller_context=ctx)
        decision3 = evaluator.evaluate(call3)
        assert decision3.action == Action.DENY

    def test_max_iterations(self):
        rule = Rule(
            name="max_iterations",
            tools=["tool1"],
            action=Action.ALLOW,
            max_iterations=2,
        )
        policy = Policy(rules=[rule])
        evaluator = PolicyEvaluator(policy)

        ctx = CallerContext(session_id="session1")

        call1 = ToolCall(tool_name="tool1", args={}, caller_context=ctx)
        decision1 = evaluator.evaluate(call1)
        assert decision1.action == Action.ALLOW

        call2 = ToolCall(tool_name="tool1", args={}, caller_context=ctx)
        decision2 = evaluator.evaluate(call2)
        assert decision2.action == Action.ALLOW

        call3 = ToolCall(tool_name="tool1", args={}, caller_context=ctx)
        decision3 = evaluator.evaluate(call3)
        assert decision3.action == Action.DENY


class TestMatchOperator:
    def test_eq(self):
        assert match_operator("eq", "value", "value") is True
        assert match_operator("eq", "value", "other") is False

    def test_not_eq(self):
        assert match_operator("not_eq", "value", "other") is True
        assert match_operator("not_eq", "value", "value") is False

    def test_contains(self):
        assert match_operator("contains", "hello world", "world") is True
        assert match_operator("contains", "hello", "world") is False

    def test_not_contains(self):
        assert match_operator("not_contains", "hello", "world") is True
        assert match_operator("not_contains", "hello world", "world") is False

    def test_matches(self):
        assert match_operator("matches", "test123", r"test\d+") is True
        assert match_operator("matches", "testabc", r"test\d+") is False

    def test_not_matches(self):
        assert match_operator("not_matches", "testabc", r"test\d+") is True
        assert match_operator("not_matches", "test123", r"test\d+") is False

    def test_exists(self):
        assert match_operator("exists", "value", None) is True
        assert match_operator("exists", None, None) is False

    def test_not_exists(self):
        assert match_operator("not_exists", None, None) is True
        assert match_operator("not_exists", "value", None) is False

    def test_not(self):
        assert match_operator("not", False, None) is True
        assert match_operator("not", True, None) is False