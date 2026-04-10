"""Tool registry for managing available tools."""

from typing import Any


class ToolRegistry:
    """Registry for managing tools."""

    def __init__(self) -> None:
        self._tools: dict[str, Any] = {}

    def register(self, tool: Any) -> None:
        """Register a tool."""
        self._tools[tool.name] = tool

    def get(self, name: str) -> tuple[Any, bool]:
        """Get a tool by name. Returns (tool, found)."""
        tool = self._tools.get(name)
        return tool, tool is not None

    def all(self) -> list[Any]:
        """Get all tools, sorted by name."""
        names = sorted(self._tools.keys())
        return [self._tools[name] for name in names]

    def names(self) -> list[str]:
        """Get all tool names, sorted."""
        return sorted(self._tools.keys())

    def schemas(self) -> dict[str, dict[str, Any]]:
        """Get input schemas for tools that support them."""
        schemas = {}
        for name, tool in self._tools.items():
            if hasattr(tool, "input_schema"):
                schemas[name] = tool.input_schema()
        return schemas

    def clear(self) -> None:
        """Clear all tools."""
        self._tools.clear()
