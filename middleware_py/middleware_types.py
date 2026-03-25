"""Core types for middleware."""

from dataclasses import dataclass, field
from enum import Enum
from typing import Any, Optional


class Action(Enum):
    """Policy action types."""

    ALLOW = "allow"
    DENY = "deny"
    FILTER = "filter"


@dataclass
class CallerContext:
    """Context about the caller making a tool call."""

    user_id: Optional[str] = None
    session_id: Optional[str] = None
    role: Optional[str] = None
    source: Optional[str] = None
    trusted: bool = False
    metadata: dict[str, Any] = field(default_factory=dict)

    def get(self, key: str, default: Any = None) -> Any:
        """Get a context value by key, supporting nested access."""
        if key == "user_id":
            return self.user_id
        elif key == "session_id":
            return self.session_id
        elif key == "role":
            return self.role
        elif key == "source":
            return self.source
        elif key == "trusted":
            return self.trusted
        return self.metadata.get(key, default)


@dataclass
class ToolDefinition:
    """Definition of a tool available for execution."""

    name: str
    description: str = ""
    input_schema: dict[str, Any] = field(default_factory=dict)


@dataclass
class ToolResult:
    """Result of a tool execution."""

    tool_name: str
    success: bool
    result: Any = None
    error: str | None = None
    metadata: dict[str, Any] = field(default_factory=dict)


@dataclass
class ToolCall:
    """Represents a tool call request."""

    tool_name: str
    args: dict[str, Any] = field(default_factory=dict)
    caller_context: CallerContext = field(default_factory=CallerContext)


@dataclass
class Decision:
    """Policy decision result."""

    action: Action
    rule_name: str | None = None
    message: str | None = None
    redacted_fields: list[str] = field(default_factory=list)
    metadata: dict[str, Any] = field(default_factory=dict)


@dataclass
class RateLimitConfig:
    """Rate limit configuration."""

    count: int
    window: int


@dataclass
class Condition:
    """A condition for policy rule matching."""

    field: str
    operator: str
    value: Any = None