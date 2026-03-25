"""Tests for middleware_py.windowmanager."""

import pytest

from middleware_py.windowmanager import (
    WarningLevel,
    Message,
    ModelSpec,
    ContextStatus,
    CompactionStrategy,
    TokenCounter,
    ContextWindowManager,
    LastNCompaction,
    Priority,
    PriorityMessage,
    PriorityBasedCompaction,
    SummaryCompaction,
    DefaultCounter,
)


class TestModelSpec:
    def test_default_values(self):
        spec = ModelSpec()
        assert spec.name == ""
        assert spec.max_tokens == 4096
        assert spec.reserved_tokens == 0
        assert spec.token_multiplier == 4.0

    def test_custom_values(self):
        spec = ModelSpec(name="gpt-4", max_tokens=8192, reserved_tokens=1024)
        assert spec.name == "gpt-4"
        assert spec.max_tokens == 8192
        assert spec.reserved_tokens == 1024


class TestMessage:
    def test_basic_message(self):
        msg = Message(role="user", content="Hello")
        assert msg.role == "user"
        assert msg.content == "Hello"
        assert msg.name == ""

    def test_message_with_name(self):
        msg = Message(role="user", content="Hello", name="user1")
        assert msg.name == "user1"


class TestWarningLevel:
    def test_warning_levels(self):
        assert WarningLevel.NONE.value == "none"
        assert WarningLevel.LOW.value == "low"
        assert WarningLevel.MEDIUM.value == "medium"
        assert WarningLevel.HIGH.value == "high"
        assert WarningLevel.CRITICAL.value == "critical"


class TestContextWindowManager:
    def test_init_with_defaults(self):
        manager = ContextWindowManager(ModelSpec())
        assert manager._model.max_tokens == 4096
        assert isinstance(manager._strategy, LastNCompaction)
        assert isinstance(manager._counter, DefaultCounter)

    def test_init_invalid_max_tokens(self):
        model = ModelSpec(max_tokens=0)
        manager = ContextWindowManager(model)
        assert manager._model.max_tokens == 4096

    def test_init_invalid_reserved_tokens(self):
        model = ModelSpec(reserved_tokens=-1)
        manager = ContextWindowManager(model)
        assert manager._model.reserved_tokens == 0

    def test_init_reserved_exceeds_max(self):
        model = ModelSpec(max_tokens=1000, reserved_tokens=2000)
        with pytest.raises(ValueError, match="reserved_tokens must be less than max_tokens"):
            ContextWindowManager(model)

    def test_init_zero_multiplier(self):
        model = ModelSpec(token_multiplier=0)
        manager = ContextWindowManager(model)
        assert manager._model.token_multiplier == 4.0

    def test_check_empty_messages(self):
        manager = ContextWindowManager(ModelSpec(max_tokens=4096, reserved_tokens=512))
        status = manager.check([])
        assert status.used_tokens == 0
        assert status.remaining_tokens == 4096 - 512
        assert status.warning_level == WarningLevel.NONE
        assert status.message_count == 0

    def test_check_with_messages(self):
        manager = ContextWindowManager(ModelSpec(max_tokens=4096))
        messages = [
            Message(role="user", content="Hello world this is a test"),
        ]
        status = manager.check(messages)
        assert status.used_tokens > 0
        assert status.message_count == 1

    def test_check_remaining_tokens(self):
        manager = ContextWindowManager(ModelSpec(max_tokens=1000, reserved_tokens=200))
        messages = [Message(role="user", content="a" * 100)]
        status = manager.check(messages)
        assert status.remaining_tokens == 800 - status.used_tokens

    def test_warning_level_calculation_critical(self):
        manager = ContextWindowManager(ModelSpec(max_tokens=1000, reserved_tokens=0))
        status = manager.check([Message(role="user", content="a" * 3584)])
        assert status.warning_level == WarningLevel.CRITICAL

    def test_warning_level_calculation_high(self):
        manager = ContextWindowManager(ModelSpec(max_tokens=1000, reserved_tokens=0))
        status = manager.check([Message(role="user", content="a" * 2984)])
        assert status.warning_level == WarningLevel.HIGH

    def test_warning_level_calculation_medium(self):
        manager = ContextWindowManager(ModelSpec(max_tokens=1000, reserved_tokens=0))
        status = manager.check([Message(role="user", content="a" * 1984)])
        assert status.warning_level == WarningLevel.MEDIUM

    def test_warning_level_calculation_low(self):
        manager = ContextWindowManager(ModelSpec(max_tokens=1000, reserved_tokens=0))
        status = manager.check([Message(role="user", content="a" * 984)])
        assert status.warning_level == WarningLevel.LOW

    def test_warning_level_calculation_none(self):
        manager = ContextWindowManager(ModelSpec(max_tokens=1000, reserved_tokens=0))
        status = manager.check([Message(role="user", content="a" * 50)])
        assert status.warning_level == WarningLevel.NONE

    def test_should_compact_true(self):
        manager = ContextWindowManager(ModelSpec(max_tokens=1000))
        messages = [Message(role="user", content="a" * 3584)]
        assert manager.should_compact(messages) is True

    def test_should_compact_false(self):
        manager = ContextWindowManager(ModelSpec(max_tokens=1000))
        messages = [Message(role="user", content="a" * 100)]
        assert manager.should_compact(messages) is False

    def test_compact_empty_messages(self):
        manager = ContextWindowManager(ModelSpec())
        result = manager.compact([])
        assert result == []

    def test_compact_needed(self):
        manager = ContextWindowManager(ModelSpec(max_tokens=1000))
        messages = [Message(role="user", content="a" * 4001)]
        result = manager.compact(messages)
        assert len(result) <= len(messages)

    def test_compact_if_needed_no_compact(self):
        manager = ContextWindowManager(ModelSpec(max_tokens=1000))
        messages = [Message(role="user", content="a" * 100)]
        result, did_compact = manager.compact_if_needed(messages)
        assert result == messages
        assert did_compact is False

    def test_compact_if_needed_with_compact(self):
        manager = ContextWindowManager(ModelSpec(max_tokens=1000))
        messages = [Message(role="user", content="a" * 4001)]
        result, did_compact = manager.compact_if_needed(messages)
        assert did_compact is True
        assert len(result) <= len(messages)


class TestLastNCompaction:
    def test_default_keep_count(self):
        strategy = LastNCompaction()
        messages = [
            Message(role="user", content="a"),
            Message(role="assistant", content="b"),
            Message(role="user", content="c"),
            Message(role="assistant", content="d"),
        ]
        counter = DefaultCounter()
        result = strategy.compact(messages, 11, counter)
        assert len(result) == 2

    def test_custom_keep_count(self):
        strategy = LastNCompaction(keep_count=1)
        messages = [
            Message(role="user", content="a"),
            Message(role="assistant", content="b"),
        ]
        counter = DefaultCounter()
        result = strategy.compact(messages, 100, counter)
        assert len(result) == 1

    def test_compact_empty_messages(self):
        strategy = LastNCompaction()
        counter = DefaultCounter()
        result = strategy.compact([], 100, counter)
        assert result == []

    def test_compact_already_fits(self):
        strategy = LastNCompaction()
        messages = [Message(role="user", content="a")]
        counter = DefaultCounter()
        result = strategy.compact(messages, 100, counter)
        assert result == messages

    def test_compact_returns_last_message(self):
        strategy = LastNCompaction(keep_count=0)
        messages = [
            Message(role="user", content="a"),
            Message(role="assistant", content="b"),
            Message(role="user", content="c"),
        ]
        counter = DefaultCounter()
        result = strategy.compact(messages, 1, counter)
        assert len(result) == 1


class TestPriorityBasedCompaction:
    def test_name(self):
        strategy = PriorityBasedCompaction()
        assert strategy.name() == "priority"

    def test_keep_system_by_default(self):
        strategy = PriorityBasedCompaction(keep_system=True)
        messages = [
            Message(role="system", content="system prompt"),
            Message(role="user", content="user message"),
            Message(role="assistant", content="assistant message"),
        ]
        counter = DefaultCounter()
        result = strategy.compact(messages, 100, counter)
        assert result[0].role == "system"

    def test_compact_empty(self):
        strategy = PriorityBasedCompaction()
        counter = DefaultCounter()
        result = strategy.compact([], 100, counter)
        assert result == []

    def test_priority_order(self):
        strategy = PriorityBasedCompaction(keep_system=False)
        messages = [
            Message(role="system", content="system"),
            Message(role="user", content="user"),
            Message(role="assistant", content="assistant"),
            Message(role="unknown", content="unknown"),
        ]
        counter = DefaultCounter()
        result = strategy.compact(messages, 100, counter)
        assert result[0].role == "assistant"


class TestSummaryCompaction:
    def test_name(self):
        strategy = SummaryCompaction()
        assert strategy.name() == "summary"

    def test_compact_empty(self):
        strategy = SummaryCompaction()
        counter = DefaultCounter()
        result = strategy.compact([], 100, counter)
        assert result == []

    def test_no_system_message(self):
        strategy = SummaryCompaction()
        messages = [
            Message(role="user", content="hello"),
            Message(role="assistant", content="hi there"),
        ]
        counter = DefaultCounter()
        result = strategy.compact(messages, 100, counter)
        assert len(result) > 0

    def test_with_system_message(self):
        strategy = SummaryCompaction()
        messages = [
            Message(role="system", content="You are helpful."),
            Message(role="user", content="hello"),
            Message(role="assistant", content="hi there"),
        ]
        counter = DefaultCounter()
        result = strategy.compact(messages, 100, counter)
        assert result[0].role == "system"

    def test_summary_contains_prefix(self):
        strategy = SummaryCompaction()
        messages = [
            Message(role="user", content="test"),
            Message(role="assistant", content="result"),
        ]
        counter = DefaultCounter()
        result = strategy.compact(messages, 1, counter)
        assert len(result) > 0


class TestDefaultCounter:
    def test_count_empty(self):
        counter = DefaultCounter()
        assert counter.count([]) == 0

    def test_count_single_message(self):
        counter = DefaultCounter()
        msg = Message(role="user", content="hello world")
        count = counter.count_message(msg)
        assert count > 0

    def test_count_multiple_messages(self):
        counter = DefaultCounter()
        messages = [
            Message(role="user", content="hello"),
            Message(role="assistant", content="world"),
        ]
        total = counter.count(messages)
        assert total > 0

    def test_count_message_with_name(self):
        counter = DefaultCounter()
        msg = Message(role="user", content="hello", name="user1")
        count = counter.count_message(msg)
        assert count > 0