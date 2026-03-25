"""LangGraph integration for middleware."""

from typing import Any, Callable, Sequence

from middleware_py.types import CallerContext, ToolCall, ToolResult
from middleware_py.middleware import Middleware


def create_middleware_tool(
    tool_runnable,
    middleware: Middleware,
    tool_name: str | None = None,
    tool_description: str = "",
):
    """Wrap a LangGraph tool runnable with middleware policy enforcement.

    Args:
        tool_runnable: A LangGraph Runnable (typically a tool function)
        middleware: Middleware instance for policy enforcement
        tool_name: Name of the tool (defaults to runnable's name)
        tool_description: Description of the tool

    Returns:
        A wrapped runnable that enforces policies before execution
    """
    name = tool_name or getattr(tool_runnable, "name", None) or "tool"
    description = tool_description or getattr(tool_runnable, "description", "") or ""

    def invoke_with_middleware(input_data: Any, config: dict | None = None) -> ToolResult:
        caller = _extract_caller_context(config)
        return middleware.call(name, _normalize_input(input_data), caller)

    async def ainvoke_with_middleware(input_data: Any, config: dict | None = None) -> ToolResult:
        caller = _extract_caller_context(config)
        return middleware.call(name, _normalize_input(input_data), caller)

    class MiddlewareWrappedTool:
        def __init__(self):
            self.name = name
            self.description = description
            self._tool = tool_runnable

        def invoke(self, input_data: Any, config: dict | None = None) -> Any:
            result = invoke_with_middleware(input_data, config)
            if not result.success:
                return {"error": result.error, "success": False}
            return result.result

        async def ainvoke(self, input_data: Any, config: dict | None = None) -> Any:
            result = await ainvoke_with_middleware(input_data, config)
            if not result.success:
                return {"error": result.error, "success": False}
            return result.result

        def batch(self, inputs: list[Any], configs: list[dict] | None = None) -> list[Any]:
            configs = configs or [None] * len(inputs)
            return [self.invoke(inp, cfg) for inp, cfg in zip(inputs, configs)]

        async def abatch(self, inputs: list[Any], configs: list[dict] | None = None) -> list[Any]:
            configs = configs or [None] * len(inputs)
            results = []
            for inp, cfg in zip(inputs, configs):
                result = await self.ainvoke(inp, cfg)
                results.append(result)
            return results

    return MiddlewareWrappedTool()


def _extract_caller_context(config: dict | None) -> CallerContext:
    """Extract CallerContext from LangGraph config."""
    if not config:
        return CallerContext()

    config_dict = dict(config) if not isinstance(config, dict) else config
    configurable = config_dict.get("configurable", {})

    return CallerContext(
        user_id=configurable.get("user_id"),
        session_id=configurable.get("session_id"),
        role=configurable.get("role"),
        source=configurable.get("source"),
        trusted=configurable.get("trusted", False),
        metadata=configurable.get("metadata", {}),
    )


def _normalize_input(input_data: Any) -> dict[str, Any]:
    """Normalize input to args dict."""
    if input_data is None:
        return {}
    if isinstance(input_data, dict):
        return input_data
    if hasattr(input_data, "model_dump"):
        return input_data.model_dump()
    if hasattr(input_data, "dict"):
        return input_data.dict()
    return {"input": input_data}


def create_middleware_node(
    middleware: Middleware,
    tool_executor: Callable[[str, dict[str, Any]], Any],
    node_name: str = "middleware_node",
) -> Callable:
    """Create a LangGraph node that enforces middleware policies.

    Args:
        middleware: Middleware instance for policy enforcement
        tool_executor: Function to execute the actual tool
        node_name: Name of the node

    Returns:
        A node function for use in LangGraph StateGraph
    """

    def node(state: dict[str, Any]) -> dict[str, Any]:
        tool_call = state.get("tool_call")
        if not tool_call:
            return {"error": "No tool_call in state"}

        tool_name = tool_call.get("name")
        args = tool_call.get("args", {})

        caller = _state_to_caller_context(state)

        result = middleware.call(tool_name, args, caller)

        return {
            "tool_result": result.result if result.success else None,
            "tool_error": result.error if not result.success else None,
            "tool_success": result.success,
        }

    return node


def _state_to_caller_context(state: dict[str, Any]) -> CallerContext:
    """Convert LangGraph state to CallerContext."""
    return CallerContext(
        user_id=state.get("user_id"),
        session_id=state.get("session_id"),
        role=state.get("role"),
        source=state.get("source"),
        trusted=state.get("trusted", False),
        metadata=state.get("metadata", {}),
    )


def middleware_on(
    middleware: Middleware,
    tools: Sequence[Any],
    tool_executor: Callable[[str, dict[str, Any]], Any],
) -> list[Any]:
    """Apply middleware to a list of LangGraph tools.

    Args:
        middleware: Middleware instance
        tools: List of LangGraph tool runnables
        tool_executor: Function to execute tools

    Returns:
        List of middleware-wrapped tools
    """
    wrapped = []
    for tool in tools:
        tool_name = getattr(tool, "name", None) or str(tool)
        wrapped_tool = create_middleware_tool(
            tool,
            middleware,
            tool_name=tool_name,
            tool_description=getattr(tool, "description", ""),
        )
        wrapped.append(wrapped_tool)

    return wrapped


def policy_decision_node(
    middleware: Middleware,
    tool_call_key: str = "tool_call",
) -> Callable:
    """Create a node that makes policy decisions without executing tools.

    This is useful for filtering tools before they're even called.

    Args:
        middleware: Middleware instance
        tool_call_key: Key in state containing the tool call info

    Returns:
        A node function that returns a decision
    """

    def node(state: dict[str, Any]) -> dict[str, Any]:
        tool_call_data = state.get(tool_call_key)
        if not tool_call_data:
            return {"decision": "allow", "reason": "No tool call"}

        tool_name = tool_call_data.get("name") if isinstance(tool_call_data, dict) else tool_call_data
        args = tool_call_data.get("args", {}) if isinstance(tool_call_data, dict) else {}

        caller = _state_to_caller_context(state)

        tool_call = ToolCall(tool_name=tool_name, args=args, caller_context=caller)
        decision = middleware._evaluator.evaluate(tool_call)

        return {
            "decision": decision.action.value,
            "decision_reason": decision.message,
            "decision_metadata": decision.metadata,
            "redacted_fields": decision.redacted_fields,
        }

    return node