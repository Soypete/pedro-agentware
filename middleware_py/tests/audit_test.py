"""Tests for middleware_py.audit."""

from datetime import datetime
from middleware_py.audit import AuditEntry, InMemoryAuditor, NoOpAuditor
from middleware_py.types import Action, CallerContext, Decision, ToolCall


class TestAuditEntry:
    def test_audit_entry_creation(self):
        ctx = CallerContext(user_id="user1")
        tool_call = ToolCall(tool_name="test_tool", args={}, caller_context=ctx)
        decision = Decision(action=Action.ALLOW)

        entry = AuditEntry(
            timestamp=datetime.now(),
            tool_call=tool_call,
            decision=decision,
            tool_result={"data": "value"},
        )

        assert entry.tool_call.tool_name == "test_tool"
        assert entry.decision.action == Action.ALLOW


class TestNoOpAuditor:
    def test_record_noop(self):
        auditor = NoOpAuditor()
        ctx = CallerContext(user_id="user1")
        tool_call = ToolCall(tool_name="test_tool", args={}, caller_context=ctx)
        decision = Decision(action=Action.ALLOW)

        auditor.record(tool_call, decision)

    def test_get_all_empty(self):
        auditor = NoOpAuditor()
        assert auditor.get_all() == []

    def test_clear_noop(self):
        auditor = NoOpAuditor()
        auditor.clear()


class TestInMemoryAuditor:
    def test_record_entry(self):
        auditor = InMemoryAuditor()
        ctx = CallerContext(user_id="user1")
        tool_call = ToolCall(tool_name="test_tool", args={}, caller_context=ctx)
        decision = Decision(action=Action.ALLOW)

        auditor.record(tool_call, decision)

        entries = auditor.get_all()
        assert len(entries) == 1
        assert entries[0].tool_call.tool_name == "test_tool"

    def test_get_all_returns_copy(self):
        auditor = InMemoryAuditor()
        ctx = CallerContext(user_id="user1")
        tool_call = ToolCall(tool_name="test_tool", args={}, caller_context=ctx)
        decision = Decision(action=Action.ALLOW)

        auditor.record(tool_call, decision)

        entries1 = auditor.get_all()
        entries2 = auditor.get_all()

        assert entries1 == entries2
        assert entries1 is not entries2

    def test_clear(self):
        auditor = InMemoryAuditor()
        ctx = CallerContext(user_id="user1")
        tool_call = ToolCall(tool_name="test_tool", args={}, caller_context=ctx)
        decision = Decision(action=Action.ALLOW)

        auditor.record(tool_call, decision)
        assert len(auditor.get_all()) == 1

        auditor.clear()
        assert len(auditor.get_all()) == 0

    def test_max_size_eviction(self):
        auditor = InMemoryAuditor(max_size=2)
        ctx = CallerContext(user_id="user1")

        for i in range(3):
            tool_call = ToolCall(tool_name=f"tool_{i}", args={}, caller_context=ctx)
            decision = Decision(action=Action.ALLOW)
            auditor.record(tool_call, decision)

        entries = auditor.get_all()
        assert len(entries) == 2

    def test_thread_safety(self):
        import threading

        auditor = InMemoryAuditor()
        ctx = CallerContext(user_id="user1")

        def record_entries():
            for i in range(100):
                tool_call = ToolCall(tool_name=f"tool_{i}", args={}, caller_context=ctx)
                decision = Decision(action=Action.ALLOW)
                auditor.record(tool_call, decision)

        threads = [threading.Thread(target=record_entries) for _ in range(5)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()

        entries = auditor.get_all()
        assert len(entries) == 500