"""Tests for ContextWindowManager."""

import threading

from pedro_agentware.llm import Message, Role
from pedro_agentware.llmcontext.context_window import ContextWindowManager, default_counter


def make_messages(count: int) -> list[Message]:
    """Create test messages with increasing content length."""
    return [
        Message(
            role=Role.USER if i % 2 == 0 else Role.ASSISTANT,
            content=f"Message {i}: " + ("x" * (i * 100)),
        )
        for i in range(count)
    ]


class TestContextWindowManager_UpdateTokenCount:
    def test_update_token_count_stores_value(self):
        mgr = ContextWindowManager(1000, default_counter)
        mgr.update_token_count(500)

        tokens, _ = mgr.check([])
        assert tokens == 500

    def test_token_count_overrides_estimate(self):
        mgr = ContextWindowManager(1000, default_counter)
        messages = make_messages(3)

        _, needs_compact_before = mgr.check(messages)
        mgr.update_token_count(100)
        tokens, _ = mgr.check(messages)

        assert tokens == 100
        assert not needs_compact_before


class TestContextWindowManager_Check_UsesActualCount:
    def test_check_returns_false_when_under_threshold(self):
        mgr = ContextWindowManager(1000, default_counter)
        mgr.update_token_count(500)

        tokens, should_compact = mgr.check([])

        assert tokens == 500
        assert not should_compact

    def test_check_returns_true_when_at_threshold(self):
        mgr = ContextWindowManager(1000, default_counter)
        mgr.set_compaction_ratio(0.5)
        mgr.update_token_count(500)

        tokens, should_compact = mgr.check([])

        assert tokens == 500
        assert should_compact


class TestContextWindowManager_ShouldCompact_UsesActualCount:
    def test_should_compact_false_when_under(self):
        mgr = ContextWindowManager(1000, default_counter)
        mgr.update_token_count(500)

        assert not mgr.should_compact([])

    def test_should_compact_true_when_over(self):
        mgr = ContextWindowManager(1000, default_counter)
        mgr.set_compaction_ratio(0.5)
        mgr.update_token_count(600)

        assert mgr.should_compact([])


class TestContextWindowManager_Compact_ResetsTokenCount:
    def test_compact_resets_last_known_tokens(self):
        mgr = ContextWindowManager(1000, default_counter)
        mgr.update_token_count(800)
        messages = make_messages(10)

        compacted = mgr.compact(messages)

        assert isinstance(compacted, list)
        tokens, _ = mgr.check([])
        assert tokens is not None
        assert tokens != 800


class TestContextWindowManager_ThreadSafety:
    def test_concurrent_access(self):
        mgr = ContextWindowManager(1000, default_counter)
        errors: list[Exception] = []

        def worker():
            try:
                for i in range(100):
                    mgr.update_token_count(i)
                    mgr.check(make_messages(5))
                    mgr.should_compact(make_messages(5))
            except Exception as e:
                errors.append(e)

        threads = [threading.Thread(target=worker) for _ in range(5)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()

        assert len(errors) == 0


class TestDefaultCounter:
    def test_counts_messages_correctly(self):
        messages = [
            Message(role=Role.USER, content="hello world test content here"),
            Message(role=Role.ASSISTANT, content="response with more content"),
        ]

        count = default_counter(messages)

        assert count > 0

    def test_empty_messages_returns_zero(self):
        count = default_counter([])
        assert count == 0
