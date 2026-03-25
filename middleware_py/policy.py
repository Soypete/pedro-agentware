"""Policy engine for evaluating rules against tool calls."""

from dataclasses import dataclass, field
from typing import Any
import re
import time
import threading

from middleware_py.middleware_types import (
    Action,
    Condition,
    Decision,
    RateLimitConfig,
    ToolCall,
)


@dataclass
class Rule:
    """Individual policy rule."""

    name: str
    tools: list[str]
    action: Action
    conditions: list[Condition] = field(default_factory=list)
    max_rate: RateLimitConfig | None = None
    max_turns: int | None = None
    max_iterations: int | None = None
    redact_fields: list[str] = field(default_factory=list)

    def matches_tool(self, tool_name: str) -> bool:
        """Check if this rule applies to the given tool."""
        for pattern in self.tools:
            if pattern == "*":
                return True
            if pattern == tool_name:
                return True
        return False


@dataclass
class Policy:
    """Collection of policy rules."""

    rules: list[Rule] = field(default_factory=list)
    default_deny: bool = False

    def get_rules_for_tool(self, tool_name: str) -> list[Rule]:
        """Get all rules that apply to a given tool."""
        return [rule for rule in self.rules if rule.matches_tool(tool_name)]


class PolicyEvaluator:
    """Interface for evaluating policies."""

    def __init__(self, policy: Policy):
        self.policy = policy
        self._rate_limit_cache: dict[str, list[float]] = {}
        self._cache_lock = threading.Lock()
        self._turn_count: dict[str, int] = {}
        self._iteration_count: dict[str, int] = {}

    def evaluate(self, tool_call: ToolCall) -> Decision:
        """Evaluate policy for a tool call."""
        rules = self.policy.get_rules_for_tool(tool_call.tool_name)

        for rule in rules:
            if self._check_conditions(rule, tool_call):
                if not self._check_rate_limit(rule, tool_call):
                    return Decision(
                        action=Action.DENY,
                        rule_name=rule.name,
                        message=f"Rate limit exceeded for tool {tool_call.tool_name}",
                    )

                if not self._check_max_turns(rule, tool_call):
                    return Decision(
                        action=Action.DENY,
                        rule_name=rule.name,
                        message="Maximum turns exceeded",
                    )

                if not self._check_max_iterations(rule, tool_call):
                    return Decision(
                        action=Action.DENY,
                        rule_name=rule.name,
                        message="Maximum iterations exceeded",
                    )

                return Decision(
                    action=rule.action,
                    rule_name=rule.name,
                    redacted_fields=rule.redact_fields,
                )

        if self.policy.default_deny:
            return Decision(
                action=Action.DENY,
                message=f"No matching rule for tool {tool_call.tool_name}",
            )

        return Decision(action=Action.ALLOW)

    def _check_conditions(self, rule: Rule, tool_call: ToolCall) -> bool:
        """Check if all conditions in a rule match."""
        if not rule.conditions:
            return True

        for condition in rule.conditions:
            if not self._match_condition(condition, tool_call):
                return False
        return True

    def _match_condition(self, condition: Condition, tool_call: ToolCall) -> bool:
        """Match a single condition against the tool call."""
        value = self._resolve_field(condition.field, tool_call)
        return match_operator(condition.operator, value, condition.value)

    def _resolve_field(self, field: str, tool_call: ToolCall) -> Any:
        """Resolve a field path to its value."""
        if field.startswith("caller."):
            key = field[7:]
            return tool_call.caller_context.get(key)

        if field.startswith("args."):
            key = field[5:]
            return tool_call.args.get(key)

        if field.startswith("context."):
            key = field[8:]
            return tool_call.caller_context.metadata.get(key)

        return None

    def _check_rate_limit(self, rule: Rule, tool_call: ToolCall) -> bool:
        """Check rate limit for a rule."""
        if not rule.max_rate:
            return True

        session_id = tool_call.caller_context.session_id or "default"
        tool_name = tool_call.tool_name
        key = f"{session_id}:{tool_name}"

        with self._cache_lock:
            if key not in self._rate_limit_cache:
                self._rate_limit_cache[key] = []

            now = time.time()
            window = rule.max_rate.window

            self._rate_limit_cache[key] = [
                t for t in self._rate_limit_cache[key] if now - t < window
            ]

            if len(self._rate_limit_cache[key]) >= rule.max_rate.count:
                return False

            self._rate_limit_cache[key].append(now)
            return True

    def _check_max_turns(self, rule: Rule, tool_call: ToolCall) -> bool:
        """Check max turns limit."""
        if not rule.max_turns:
            return True

        session_id = tool_call.caller_context.session_id or "default"
        key = f"{session_id}:turns"

        count = self._turn_count.get(key, 0)
        if count >= rule.max_turns:
            return False

        self._turn_count[key] = count + 1
        return True

    def _check_max_iterations(self, rule: Rule, tool_call: ToolCall) -> bool:
        """Check max iterations limit."""
        if not rule.max_iterations:
            return True

        session_id = tool_call.caller_context.session_id or "default"
        tool_name = tool_call.tool_name
        key = f"{session_id}:{tool_name}:iterations"

        count = self._iteration_count.get(key, 0)
        if count >= rule.max_iterations:
            return False

        self._iteration_count[key] = count + 1
        return True


def match_operator(operator: str, field_value: Any, condition_value: Any) -> bool:
    """Match a field value against a condition using an operator."""
    if operator == "eq":
        return field_value == condition_value
    elif operator == "not_eq":
        return field_value != condition_value
    elif operator == "contains":
        return (
            field_value is not None
            and condition_value is not None
            and str(condition_value) in str(field_value)
        )
    elif operator == "not_contains":
        return (
            field_value is None
            or condition_value is None
            or str(condition_value) not in str(field_value)
        )
    elif operator == "matches":
        if field_value is None or condition_value is None:
            return False
        try:
            return bool(re.match(str(condition_value), str(field_value)))
        except re.error:
            return False
    elif operator == "not_matches":
        if field_value is None or condition_value is None:
            return True
        try:
            return not bool(re.match(str(condition_value), str(field_value)))
        except re.error:
            return False
    elif operator == "exists":
        return field_value is not None
    elif operator == "not_exists":
        return field_value is None
    elif operator == "not":
        return not field_value

    return False