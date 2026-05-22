"""Middleware types."""

from dataclasses import dataclass, field
from datetime import datetime
from enum import Enum
from typing import Any


class MessageType(str, Enum):
    """Semantic type of a message for smart compaction and guardrails."""

    SYSTEM_PROMPT = "system_prompt"
    USER_INPUT = "user_input"
    TOOL_CALL = "tool_call"
    TOOL_RESULT = "tool_result"
    REASONING = "reasoning"
    TEXT_RESPONSE = "text_response"
    STEP_NUDGE = "step_nudge"
    PREREQUISITE_NUDGE = "prerequisite_nudge"
    RETRY_NUDGE = "retry_nudge"
    CONTEXT_WARNING = "context_warning"
    SUMMARY = "summary"


@dataclass(frozen=True)
class MessageMeta:
    """Metadata for a message used in smart compaction and guardrails."""

    type: MessageType = MessageType.USER_INPUT
    step_index: int | None = None
    original_type: MessageType | None = None
    token_estimate: int | None = None


class Action(str, Enum):
    """Action to take on a tool call."""

    ALLOW = "allow"
    DENY = "deny"
    FILTER = "filter"


@dataclass
class CallerContext:
    """Context about the caller making the tool call."""

    user_id: str = ""
    session_id: str = ""
    role: str = ""
    source: str = ""
    trusted: bool = True
    metadata: dict[str, str] = field(default_factory=dict)


@dataclass
class Decision:
    """Decision made by policy evaluator."""

    action: Action
    rule: str = ""
    reason: str = ""
    redacted_args: dict[str, Any] = field(default_factory=dict)
    timestamp: datetime = field(default_factory=datetime.now)

    def to_dict(self) -> dict[str, Any]:
        return {
            "action": self.action.value,
            "rule": self.rule,
            "reason": self.reason,
            "redacted_args": self.redacted_args,
            "timestamp": self.timestamp.isoformat(),
        }
