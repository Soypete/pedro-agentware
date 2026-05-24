"""Context window management for conversation history."""

import threading

from pedro_agentware.llm import Message
from pedro_agentware.llmcontext.strategies import (
    CompactionStrategy,
    TieredCompact,
    TokenCounter,
)


class ContextWindowManager:
    """Manages context window size and compaction for conversation history."""

    def __init__(
        self,
        context_window: int,
        counter: TokenCounter | None = None,
        strategy: CompactionStrategy | None = None,
    ) -> None:
        self._context_window = context_window
        self._compaction_ratio = 0.75
        self._counter: TokenCounter = counter if counter is not None else default_counter
        self._strategy: CompactionStrategy = (
            strategy if strategy is not None else TieredCompact()
        )
        self._last_known_tokens: int | None = None
        self._lock = threading.RLock()

    def set_compaction_ratio(self, ratio: float) -> None:
        """Set the ratio of context window at which compaction triggers."""
        with self._lock:
            self._compaction_ratio = ratio

    def update_token_count(self, total_tokens: int) -> None:
        """Record actual token count from backend API response."""
        with self._lock:
            self._last_known_tokens = total_tokens

    def check(self, messages: list[Message]) -> tuple[int, bool]:
        """Check current token count and if threshold is exceeded.

        Returns:
            Tuple of (current_tokens, should_compact)
        """
        with self._lock:
            current_tokens = self._estimate_tokens(messages)
            threshold = int(self._context_window * self._compaction_ratio)
            return current_tokens, current_tokens >= threshold

    def should_compact(self, messages: list[Message]) -> bool:
        """Determine if compaction should be triggered."""
        with self._lock:
            current_tokens = self._estimate_tokens(messages)
            threshold = int(self._context_window * self._compaction_ratio)
            return current_tokens > threshold

    def compact(self, messages: list[Message]) -> list[Message]:
        """Compact messages to fit within target token count.

        Returns compacted messages. Resets last_known_tokens after compaction.
        """
        with self._lock:
            target_tokens = int(self._context_window * self._compaction_ratio)
            compacted = self._strategy.compact(messages, target_tokens, self._counter)
            self._last_known_tokens = None
            return compacted

    def _estimate_tokens(self, messages: list[Message]) -> int:
        """Estimate tokens, using actual count if available."""
        if self._last_known_tokens is not None:
            return self._last_known_tokens
        return self._counter(messages)


def default_counter(messages: list[Message]) -> int:
    """Default token counter using character-based estimation."""
    total = 0
    for m in messages:
        overhead = len(str(m.role)) + 4
        if m.tool_calls:
            for tc in m.tool_calls:
                overhead += len(tc.name) + 1
        total += (len(m.content) // 4) + overhead
    return total
