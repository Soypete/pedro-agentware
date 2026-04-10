"""Audit logging for middleware."""

from dataclasses import dataclass, field
from datetime import datetime
from typing import Any, Protocol

from .types import Action, Decision


@dataclass
class AuditRecord:
    """Record of a tool call decision."""

    session_id: str
    tool_name: str
    args: dict[str, Any]
    decision: Decision
    timestamp: datetime = field(default_factory=datetime.now)


class AuditFilter:
    """Filter for querying audit records."""

    def __init__(
        self,
        session_id: str = "",
        tool_name: str = "",
        action: Action | None = None,
        since: datetime | None = None,
        limit: int = 0,
    ) -> None:
        self.session_id = session_id
        self.tool_name = tool_name
        self.action = action
        self.since = since
        self.limit = limit


class Auditor(Protocol):
    """Protocol for auditors."""

    def record(self, record: AuditRecord) -> None:
        """Record an audit entry."""
        ...

    def query(self, filter: AuditFilter) -> list[AuditRecord]:
        """Query audit records."""
        ...


class InMemoryAuditor:
    """In-memory auditor for testing and development."""

    def __init__(self) -> None:
        self._records: list[AuditRecord] = []

    def record(self, record: AuditRecord) -> None:
        """Record an audit entry."""
        self._records.append(record)

    def query(self, filter: AuditFilter) -> list[AuditRecord]:
        """Query audit records."""
        results = []
        for r in self._records:
            if filter.session_id and r.session_id != filter.session_id:
                continue
            if filter.tool_name and r.tool_name != filter.tool_name:
                continue
            if filter.action and r.decision.action != filter.action:
                continue
            if filter.since and r.timestamp < filter.since:
                continue
            results.append(r)
            if filter.limit > 0 and len(results) >= filter.limit:
                break
        return results

    def clear(self) -> None:
        """Clear all records."""
        self._records.clear()
