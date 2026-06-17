"""Pydantic AI client wrapper."""

from dataclasses import dataclass
from typing import Any, Generic, TypeVar

try:
    from pydantic_ai import Agent
    from pydantic_ai.tools import ToolDefinition
except ImportError:
    Agent = None  # type: ignore
    ToolDefinition = None  # type: ignore

Output = TypeVar("Output")


@dataclass
class PydanticTool:
    """Represents a Pydantic AI tool."""

    name: str
    description: str
    parameters_json_schema: dict[str, Any]


class Client(Generic[Output]):
    """Wrapper for Pydantic AI Agent.

    Extracts tool definitions and handles execution.
    """

    def __init__(self, agent: "Agent[Any, Output]"):
        if Agent is None:
            raise ImportError("pydantic-ai is not installed")
        self._agent = agent

    def list_tools(self) -> list[PydanticTool]:
        """List tools from the Pydantic AI agent."""
        tool_definitions = self._agent._tools  # type: ignore[attr-defined]
        return [
            PydanticTool(
                name=tool.name,
                description=tool.description,
                parameters_json_schema=tool.parameters_json_schema or {},
            )
            for tool in tool_definitions
        ]

    async def execute(
        self,
        tool_name: str,
        args: dict[str, Any],
    ) -> tuple[Any | None, str | None]:
        """Execute a tool by calling the agent with the tool name and args.

        Note: Pydantic AI doesn't expose tools as standalone callable functions.
        This executes the full agent loop - consider using a simpler approach.
        """
        try:
            # For Pydantic AI, we need to run the agent with the tool context
            # This is a simplified implementation - full integration would need
            # more sophisticated tool call handling
            result = await self._agent.run(
                f"Execute tool: {tool_name} with args: {args}"
            )
            return result.data, None
        except Exception as e:
            return None, str(e)

    def get_agent(self) -> "Agent[Any, Output]":
        """Get the underlying Pydantic AI agent."""
        return self._agent