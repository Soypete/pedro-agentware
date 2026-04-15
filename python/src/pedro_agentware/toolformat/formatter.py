"""Tool formatter - Model-specific tool formatting."""

from dataclasses import dataclass
from typing import Any, Protocol

from ..tools import Result, Tool


@dataclass
class ParsedToolCall:
    """A parsed tool call from model output."""

    id: str
    name: str
    args: dict[str, Any]
    raw: str


class ToolFormatter(Protocol):
    """Protocol for tool formatters."""

    def format_tool_definitions(self, tools: list[Tool]) -> str:
        """Format tool definitions for model prompt."""
        ...

    def parse_tool_calls(self, response: str) -> list[ParsedToolCall]:
        """Parse tool calls from model output."""
        ...

    def format_tool_result(self, name: str, result: Result) -> str:
        """Format tool result for model input."""
        ...

    def model_family(self) -> str:
        """Return the model family name."""
        ...


class GenericFormatter:
    """Generic tool formatter for unknown models."""

    def format_tool_definitions(self, tools: list[Tool]) -> str:
        """Format tools as JSON schema."""
        if not tools:
            return "No tools available."

        lines = ["Available tools:"]
        for tool in tools:
            lines.append(f"- {tool.name}: {tool.description}")
        return "\n".join(lines)

    def parse_tool_calls(self, response: str) -> list[ParsedToolCall]:
        """Parse tool calls from response."""
        return []

    def format_tool_result(self, name: str, result: Result) -> str:
        """Format tool result."""
        if result.success:
            return f"Tool {name} result: {result.data}"
        return f"Tool {name} error: {result.error}"

    def model_family(self) -> str:
        return "generic"
