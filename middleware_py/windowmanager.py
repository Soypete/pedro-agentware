"""Context window management for preemptive context monitoring."""

from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from datetime import datetime
from enum import Enum
from typing import Protocol


class WarningLevel(str, Enum):
    NONE = "none"
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"
    CRITICAL = "critical"


@dataclass
class Message:
    role: str
    content: str
    name: str = ""


@dataclass
class ModelSpec:
    name: str = ""
    max_tokens: int = 4096
    reserved_tokens: int = 0
    token_multiplier: float = 4.0


@dataclass
class ContextStatus:
    used_tokens: int
    remaining_tokens: int
    max_tokens: int
    reserved_tokens: int
    warning_level: WarningLevel
    message_count: int


class CompactionStrategy(Protocol):
    """Protocol for compaction strategies."""

    @abstractmethod
    def compact(self, messages: list[Message], target_tokens: int, counter: "TokenCounter") -> list[Message]:
        """Compact messages to fit within target token limit."""
        ...

    @abstractmethod
    def name(self) -> str:
        """Return strategy name."""
        ...


class TokenCounter(Protocol):
    """Protocol for token counting."""

    @abstractmethod
    def count(self, messages: list[Message]) -> int:
        """Count total tokens in messages."""
        ...

    @abstractmethod
    def count_message(self, message: Message) -> int:
        """Count tokens in a single message."""
        ...


class ContextWindowManager:
    """Manages context window with preemptive compaction."""

    def __init__(
        self,
        model: ModelSpec,
        strategy: CompactionStrategy | None = None,
        counter: TokenCounter | None = None,
    ):
        if model.max_tokens <= 0:
            model.max_tokens = 4096
        if model.reserved_tokens < 0:
            model.reserved_tokens = 0
        if model.reserved_tokens >= model.max_tokens:
            raise ValueError("reserved_tokens must be less than max_tokens")
        if model.token_multiplier == 0:
            model.token_multiplier = 4.0

        self._model = model
        self._strategy = strategy or LastNCompaction()
        self._counter = counter or DefaultCounter()
        self._last_check: datetime | None = None

    def check(self, messages: list[Message]) -> ContextStatus:
        """Check current context status."""
        if not messages:
            return ContextStatus(
                used_tokens=0,
                remaining_tokens=self._model.max_tokens - self._model.reserved_tokens,
                max_tokens=self._model.max_tokens,
                reserved_tokens=self._model.reserved_tokens,
                warning_level=WarningLevel.NONE,
                message_count=0,
            )

        token_count = self._counter.count(messages)
        available = self._model.max_tokens - self._model.reserved_tokens
        remaining = available - token_count

        return ContextStatus(
            used_tokens=token_count,
            remaining_tokens=remaining,
            max_tokens=self._model.max_tokens,
            reserved_tokens=self._model.reserved_tokens,
            warning_level=self._calculate_warning_level(remaining, available),
            message_count=len(messages),
        )

    def should_compact(self, messages: list[Message]) -> bool:
        """Check if compaction is needed."""
        status = self.check(messages)
        threshold = self._model.max_tokens // 4
        return status.remaining_tokens < threshold

    def compact(self, messages: list[Message]) -> list[Message]:
        """Compact messages to free up space."""
        if not messages:
            return messages

        available = self._model.max_tokens - self._model.reserved_tokens
        target_tokens = int(available * 0.75)

        return self._strategy.compact(messages, target_tokens, self._counter)

    def compact_if_needed(self, messages: list[Message]) -> tuple[list[Message], bool]:
        """Compact messages if needed, returns (messages, did_compact)."""
        if not self.should_compact(messages):
            return messages, False

        compacted = self.compact(messages)
        return compacted, True

    def _calculate_warning_level(self, remaining: int, available: int) -> WarningLevel:
        """Calculate warning level based on remaining tokens."""
        if available == 0:
            return WarningLevel.CRITICAL

        ratio = remaining / available

        if ratio <= 0.1:
            return WarningLevel.CRITICAL
        elif ratio <= 0.25:
            return WarningLevel.HIGH
        elif ratio <= 0.5:
            return WarningLevel.MEDIUM
        elif ratio <= 0.75:
            return WarningLevel.LOW
        else:
            return WarningLevel.NONE


class LastNCompaction:
    """Compaction strategy that keeps the last N messages."""

    def __init__(self, keep_count: int = 0):
        self._keep_count = keep_count

    def name(self) -> str:
        return "last_n"

    def compact(self, messages: list[Message], target_tokens: int, counter: TokenCounter) -> list[Message]:
        if not messages:
            return messages

        keep_count = self._keep_count
        if keep_count <= 0:
            keep_count = len(messages) // 2
        if keep_count < 1:
            keep_count = 1

        if keep_count >= len(messages):
            count = counter.count(messages)
            if count <= target_tokens:
                return messages
            keep_count = len(messages) // 2

        for i in range(keep_count, 0, -1):
            subset = messages[len(messages) - i:]
            count = counter.count(subset)
            if count <= target_tokens:
                return subset

        return [messages[-1]]


class Priority(str, Enum):
    SYSTEM = "system"
    HIGH = "high"
    MEDIUM = "medium"
    LOW = "low"


@dataclass
class PriorityMessage:
    message: Message
    priority: Priority


class PriorityBasedCompaction:
    """Compaction strategy that keeps messages by priority."""

    def __init__(self, keep_system: bool = True):
        self._keep_system = keep_system

    def name(self) -> str:
        return "priority"

    def compact(self, messages: list[Message], target_tokens: int, counter: TokenCounter) -> list[Message]:
        if not messages:
            return messages

        priority_msgs: list[PriorityMessage] = []
        for msg in messages:
            if msg.role == "system":
                priority = Priority.SYSTEM if self._keep_system else Priority.LOW
            elif msg.role == "assistant":
                priority = Priority.HIGH
            elif msg.role == "user":
                priority = Priority.MEDIUM
            else:
                priority = Priority.LOW
            priority_msgs.append(PriorityMessage(message=msg, priority=priority))

        priority_order = [Priority.SYSTEM, Priority.HIGH, Priority.MEDIUM, Priority.LOW]
        sorted_msgs = sorted(priority_msgs, key=lambda pm: priority_order.index(pm.priority))

        result: list[Message] = []
        for pm in sorted_msgs:
            result.append(pm.message)
            if counter.count(result) > target_tokens:
                result = result[:-1]
                break

        if not result and messages:
            result = [messages[-1]]

        return result


class SummaryCompaction:
    """Compaction strategy that summarizes older messages."""

    def __init__(self, max_summary_len: int = 500):
        self._max_summary_len = max_summary_len

    def name(self) -> str:
        return "summary"

    def compact(self, messages: list[Message], target_tokens: int, counter: TokenCounter) -> list[Message]:
        if not messages:
            return messages

        system_msg: Message | None = None
        other_msgs: list[Message] = []

        for msg in messages:
            if msg.role == "system":
                system_msg = msg
            else:
                other_msgs.append(msg)

        if not other_msgs:
            return messages

        count = counter.count(other_msgs)
        if count <= target_tokens:
            if system_msg:
                return [system_msg] + other_msgs
            return messages

        result: list[Message] = []
        if system_msg:
            result.append(system_msg)

        summary_tokens = target_tokens // 4
        summary_chars = int(summary_tokens * 3)

        summary = self._summarize_messages(other_msgs, summary_chars)
        summary_msg = Message(
            role="system",
            content=f"[Previous conversation summarized: {summary}]",
        )
        result.append(summary_msg)

        if counter.count(result) > target_tokens:
            keep_count = len(other_msgs) // 3
            if keep_count < 1:
                keep_count = 1
            result.extend(other_msgs[-keep_count:])

        return result

    def _summarize_messages(self, messages: list[Message], max_len: int) -> str:
        """Create a simple summary of messages."""
        content = ""
        for msg in messages:
            prefix = msg.name if msg.name else msg.role
            content += f"{prefix}: {msg.content}\n"

        if len(content) <= max_len:
            return content

        truncated = content[: max_len - 3]
        last_space = truncated.rfind(" ")
        if last_space > 0:
            return truncated[:last_space] + "..."

        return truncated + "..."


class DefaultCounter:
    """Default token counter using character-based estimation."""

    def count(self, messages: list[Message]) -> int:
        total = 0
        for msg in messages:
            total += self.count_message(msg)
        return total

    def count_message(self, message: Message) -> int:
        content = message.role
        if message.name:
            content += message.name
        content += message.content

        tokens = len(content) // 4
        return tokens + 4