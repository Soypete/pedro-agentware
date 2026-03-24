"""Core middleware implementation."""

from typing import Any, Callable
import threading
from collections import defaultdict

from middleware_py.types import (
    Action,
    CallerContext,
    ToolCall,
    ToolDefinition,
    ToolResult,
)
from middleware_py.policy import Policy, PolicyEvaluator
from middleware_py.audit import Auditor, NoOpAuditor


CallHistoryValue = tuple[list[str], list[str]]


class CallHistory:
    """Tracks called/failed tools per session."""

    def __init__(self):
        self._called: dict[str, list[str]] = defaultdict(list)
        self._failed: dict[str, list[str]] = defaultdict(list)
        self._lock = threading.RLock()

    def record(self, session_id: str, tool_name: str, success: bool):
        """Record a tool call."""
        with self._lock:
            if success:
                self._called[session_id].append(tool_name)
            else:
                self._failed[session_id].append(tool_name)

    def get_called(self, session_id: str) -> list[str]:
        """Get list of called tools for a session."""
        with self._lock:
            return list(self._called.get(session_id, []))

    def get_failed(self, session_id: str) -> list[str]:
        """Get list of failed tools for a session."""
        with self._lock:
            return list(self._failed.get(session_id, []))

    def clear(self, session_id: str | None = None):
        """Clear history for a session or all."""
        with self._lock:
            if session_id is None:
                self._called.clear()
                self._failed.clear()
            else:
                self._called.pop(session_id, None)
                self._failed.pop(session_id, None)


MiddlewareOption = Callable[["Middleware"], None]


def with_auditor(auditor: Auditor) -> MiddlewareOption:
    """Option to set a custom auditor."""

    def apply(m: "Middleware"):
        m._auditor = auditor

    return apply


def with_tools(tools: list[ToolDefinition]) -> MiddlewareOption:
    """Option to set available tools."""

    def apply(m: "Middleware"):
        m._tools = {tool.name: tool for tool in tools}

    return apply


class Middleware:
    """Middleware wrapping tool execution with policy enforcement."""

    def __init__(
        self,
        executor: Callable[[str, dict[str, Any]], ToolResult] | None = None,
        policy: Policy | None = None,
        options: list[MiddlewareOption] | None = None,
    ):
        self._executor = executor
        self._policy = policy or Policy()
        self._evaluator = PolicyEvaluator(self._policy)
        self._auditor: Auditor = NoOpAuditor()
        self._tools: dict[str, ToolDefinition] = {}
        self._history = CallHistory()

        if options:
            for option in options:
                option(self)

    def set_executor(self, executor: Callable[[str, dict[str, Any]], ToolResult]):
        """Set the tool executor function."""
        self._executor = executor

    def set_policy(self, policy: Policy):
        """Set the policy."""
        self._policy = policy
        self._evaluator = PolicyEvaluator(policy)

    def call(self, tool_name: str, args: dict[str, Any], caller_context: CallerContext | None = None) -> ToolResult:
        """Execute a tool call through the middleware."""
        caller = caller_context or CallerContext()
        tool_call = ToolCall(tool_name=tool_name, args=args, caller_context=caller)

        decision = self._evaluator.evaluate(tool_call)

        if decision.action == Action.DENY:
            self._auditor.record(tool_call, decision)
            self._history.record(caller.session_id or "default", tool_name, success=False)
            return ToolResult(
                tool_name=tool_name,
                success=False,
                error=decision.message or "Denied by policy",
            )

        if decision.action == Action.FILTER:
            result = self._execute_tool(tool_name, args, caller)
            redacted_result = self._redact_result(result, decision.redacted_fields)
            self._auditor.record(tool_call, decision, redacted_result)
            self._history.record(caller.session_id or "default", tool_name, success=redacted_result.success)
            return redacted_result

        result = self._execute_tool(tool_name, args, caller)
        self._auditor.record(tool_call, decision, result)
        self._history.record(caller.session_id or "default", tool_name, result.success)
        return result

    def _execute_tool(self, tool_name: str, args: dict[str, Any], caller: CallerContext) -> ToolResult:
        """Execute the tool using the wrapped executor."""
        if not self._executor:
            return ToolResult(
                tool_name=tool_name,
                success=False,
                error="No executor configured",
            )

        try:
            return self._executor(tool_name, args)
        except Exception as e:
            return ToolResult(
                tool_name=tool_name,
                success=False,
                error=str(e),
            )

    def _redact_result(self, result: ToolResult, fields: list[str]) -> ToolResult:
        """Redact sensitive fields from the result."""
        if not fields or result.result is None:
            return result

        if isinstance(result.result, dict):
            redacted = dict(result.result)
            for field in fields:
                if field in redacted:
                    redacted[field] = "[REDACTED]"
            return ToolResult(
                tool_name=result.tool_name,
                success=result.success,
                result=redacted,
                error=result.error,
                metadata=result.metadata,
            )

        return result

    def list_tools(self) -> list[ToolDefinition]:
        """List available tools."""
        return list(self._tools.values())

    def filter_tools(self, caller: CallerContext) -> list[ToolDefinition]:
        """Filter tools based on policy for a caller."""
        allowed = []
        for tool in self._tools.values():
            tool_call = ToolCall(tool_name=tool.name, args={}, caller_context=caller)
            decision = self._evaluator.evaluate(tool_call)
            if decision.action != Action.DENY:
                allowed.append(tool)
        return allowed

    def get_history(self, session_id: str) -> CallHistoryValue:
        """Get call history for a session."""
        return (self._history.get_called(session_id), self._history.get_failed(session_id))

    def clear_history(self, session_id: str | None = None):
        """Clear call history."""
        self._history.clear(session_id)