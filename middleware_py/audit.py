"""Auditor for recording policy decisions."""

from dataclasses import dataclass
from datetime import datetime
from typing import Any
import threading

from middleware_py.types import Decision, ToolCall


@dataclass
class AuditEntry:
    """Single audit log entry."""

    timestamp: datetime
    tool_call: ToolCall
    decision: Decision
    tool_result: Any = None


class Auditor:
    """Interface for recording audit logs."""

    def record(self, tool_call: ToolCall, decision: Decision, tool_result: Any = None):
        """Record an audit entry."""
        raise NotImplementedError

    def get_all(self) -> list[AuditEntry]:
        """Get all audit entries."""
        raise NotImplementedError

    def clear(self):
        """Clear all audit entries."""
        raise NotImplementedError


class InMemoryAuditor(Auditor):
    """Thread-safe in-memory audit storage."""

    def __init__(self, max_size: int = 1000):
        self._entries: list[AuditEntry] = []
        self._max_size = max_size
        self._lock = threading.Lock()

    def record(self, tool_call: ToolCall, decision: Decision, tool_result: Any = None):
        """Record an audit entry."""
        entry = AuditEntry(
            timestamp=datetime.now(),
            tool_call=tool_call,
            decision=decision,
            tool_result=tool_result,
        )

        with self._lock:
            if len(self._entries) >= self._max_size:
                self._entries.pop(0)
            self._entries.append(entry)

    def get_all(self) -> list[AuditEntry]:
        """Get all audit entries."""
        with self._lock:
            return list(self._entries)

    def clear(self):
        """Clear all audit entries."""
        with self._lock:
            self._entries.clear()


class NoOpAuditor(Auditor):
    """No-op auditor for testing."""

    def record(self, tool_call: ToolCall, decision: Decision, tool_result: Any = None):
        """No-op record."""
        pass

    def get_all(self) -> list[AuditEntry]:
        """Return empty list."""
        return []

    def clear(self):
        """No-op clear."""
        pass