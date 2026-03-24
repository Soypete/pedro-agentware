package windowmanager

import (
	"context"
	"testing"
)

func TestNewContextWindowManager(t *testing.T) {
	tests := []struct {
		name      string
		model     ModelSpec
		strategy  CompactionStrategy
		counter   TokenCounter
		expectErr bool
	}{
		{
			name: "valid config uses defaults",
			model: ModelSpec{
				MaxTokens:      4096,
				ReservedTokens: 512,
			},
			strategy:  &LastNCompaction{KeepCount: 10},
			counter:   nil,
			expectErr: false,
		},
		{
			name: "zero max tokens uses default",
			model: ModelSpec{
				MaxTokens:      0,
				ReservedTokens: 100,
			},
			strategy:  &LastNCompaction{},
			counter:   nil,
			expectErr: false,
		},
		{
			name: "negative reserved tokens uses default",
			model: ModelSpec{
				MaxTokens:      4096,
				ReservedTokens: -1,
			},
			strategy:  &LastNCompaction{},
			counter:   nil,
			expectErr: false,
		},
		{
			name: "reserved tokens >= max tokens returns error",
			model: ModelSpec{
				MaxTokens:      4096,
				ReservedTokens: 4096,
			},
			strategy:  &LastNCompaction{},
			counter:   nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewContextWindowManager(tt.model, tt.strategy, tt.counter)
			if (err != nil) != tt.expectErr {
				t.Errorf("NewContextWindowManager() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestContextWindowManager_Check(t *testing.T) {
	manager, _ := NewContextWindowManager(
		ModelSpec{MaxTokens: 1000, ReservedTokens: 200},
		&LastNCompaction{},
		nil,
	)

	tests := []struct {
		name           string
		messages       []Message
		expectedTokens int
		expectError    bool
	}{
		{
			name:           "empty messages returns zero tokens",
			messages:       []Message{},
			expectedTokens: 0,
			expectError:    false,
		},
		{
			name: "single user message",
			messages: []Message{
				{Role: "user", Content: "hello"},
			},
			expectError: false,
		},
		{
			name: "multiple messages",
			messages: []Message{
				{Role: "system", Content: "You are helpful."},
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there!"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := manager.Check(context.Background(), tt.messages)
			if (err != nil) != tt.expectError {
				t.Errorf("Check() error = %v, expectError %v", err, tt.expectError)
			}
			if !tt.expectError && status == nil {
				t.Error("Check() returned nil status")
			}
		})
	}
}

func TestContextWindowManager_ShouldCompact(t *testing.T) {
	manager, _ := NewContextWindowManager(
		ModelSpec{MaxTokens: 1000, ReservedTokens: 200},
		&LastNCompaction{},
		nil,
	)

	smallMessages := []Message{
		{Role: "user", Content: "Hi"},
	}

	largeMessages := make([]Message, 100)
	for i := range largeMessages {
		largeMessages[i] = Message{Role: "user", Content: "This is a test message with some content to fill up space."}
	}

	shouldCompact, err := manager.ShouldCompact(context.Background(), smallMessages)
	if err != nil {
		t.Errorf("ShouldCompact() error = %v", err)
	}
	if shouldCompact {
		t.Error("ShouldCompact() should not compact small messages")
	}

	_, err = manager.ShouldCompact(context.Background(), largeMessages)
	if err != nil {
		t.Errorf("ShouldCompact() error = %v", err)
	}
}

func TestContextWindowManager_Compact(t *testing.T) {
	manager, _ := NewContextWindowManager(
		ModelSpec{MaxTokens: 1000, ReservedTokens: 200},
		&LastNCompaction{KeepCount: 2},
		nil,
	)

	messages := []Message{
		{Role: "system", Content: "System prompt"},
		{Role: "user", Content: "Message 1"},
		{Role: "assistant", Content: "Response 1"},
		{Role: "user", Content: "Message 2"},
		{Role: "assistant", Content: "Response 2"},
	}

	compacted, err := manager.Compact(context.Background(), messages)
	if err != nil {
		t.Errorf("Compact() error = %v", err)
	}
	if len(compacted) == 0 {
		t.Error("Compact() returned empty messages")
	}
}

func TestContextWindowManager_CompactIfNeeded(t *testing.T) {
	manager, _ := NewContextWindowManager(
		ModelSpec{MaxTokens: 1000, ReservedTokens: 200},
		&LastNCompaction{KeepCount: 10},
		nil,
	)

	smallMessages := []Message{
		{Role: "user", Content: "Hello"},
	}

	compacted, didCompact, err := manager.CompactIfNeeded(context.Background(), smallMessages)
	if err != nil {
		t.Errorf("CompactIfNeeded() error = %v", err)
	}
	if didCompact {
		t.Error("CompactIfNeeded() should not compact small messages")
	}
	if len(compacted) != len(smallMessages) {
		t.Errorf("CompactIfNeeded() returned %d messages, want %d", len(compacted), len(smallMessages))
	}
}

func TestContextWindowManager_calculateWarningLevel(t *testing.T) {
	manager, _ := NewContextWindowManager(
		ModelSpec{MaxTokens: 1000, ReservedTokens: 0},
		&LastNCompaction{},
		nil,
	)

	tests := []struct {
		name          string
		remaining     int
		available     int
		expectedLevel WarningLevel
	}{
		{"critical 10%", 100, 1000, WarningLevelCritical},
		{"high 25%", 250, 1000, WarningLevelHigh},
		{"medium 50%", 500, 1000, WarningLevelMedium},
		{"low 75%", 750, 1000, WarningLevelLow},
		{"none 90%", 900, 1000, WarningLevelNone},
		{"zero available", 0, 0, WarningLevelCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := manager.calculateWarningLevel(tt.remaining, tt.available)
			if level != tt.expectedLevel {
				t.Errorf("calculateWarningLevel() = %v, want %v", level, tt.expectedLevel)
			}
		})
	}
}

func TestLastNCompaction_Compact(t *testing.T) {
	strategy := &LastNCompaction{KeepCount: 2}
	counter := &DefaultCounter{}

	messages := []Message{
		{Role: "system", Content: "System"},
		{Role: "user", Content: "User 1"},
		{Role: "assistant", Content: "Assistant 1"},
		{Role: "user", Content: "User 2"},
		{Role: "assistant", Content: "Assistant 2"},
	}

	compacted, err := strategy.Compact(messages, 100, counter)
	if err != nil {
		t.Errorf("Compact() error = %v", err)
	}
	if len(compacted) > len(messages) {
		t.Errorf("Compact() returned more messages than input: %d > %d", len(compacted), len(messages))
	}
}

func TestLastNCompaction_CompactZeroKeepCount(t *testing.T) {
	strategy := &LastNCompaction{KeepCount: 0}
	counter := &DefaultCounter{}

	messages := []Message{
		{Role: "user", Content: "Message 1"},
		{Role: "user", Content: "Message 2"},
	}

	compacted, err := strategy.Compact(messages, 1000, counter)
	if err != nil {
		t.Errorf("Compact() error = %v", err)
	}
	if len(compacted) == 0 {
		t.Error("Compact() should return at least one message")
	}
}

func TestLastNCompaction_CompactEmpty(t *testing.T) {
	strategy := &LastNCompaction{}
	counter := &DefaultCounter{}

	compacted, err := strategy.Compact([]Message{}, 100, counter)
	if err != nil {
		t.Errorf("Compact() error = %v", err)
	}
	if len(compacted) != 0 {
		t.Error("Compact() should return empty for empty input")
	}
}

func TestSummaryCompaction_Compact(t *testing.T) {
	strategy := NewSummaryCompaction()
	counter := &DefaultCounter{}

	messages := []Message{
		{Role: "system", Content: "System prompt here"},
		{Role: "user", Content: "User message 1"},
		{Role: "assistant", Content: "Assistant response 1"},
		{Role: "user", Content: "User message 2"},
		{Role: "assistant", Content: "Assistant response 2"},
	}

	compacted, err := strategy.Compact(messages, 100, counter)
	if err != nil {
		t.Errorf("Compact() error = %v", err)
	}
	if len(compacted) == 0 {
		t.Error("Compact() should return messages")
	}
}

func TestSummaryCompaction_CompactEmpty(t *testing.T) {
	strategy := NewSummaryCompaction()
	counter := &DefaultCounter{}

	compacted, err := strategy.Compact([]Message{}, 100, counter)
	if err != nil {
		t.Errorf("Compact() error = %v", err)
	}
	if len(compacted) != 0 {
		t.Error("Compact() should return empty for empty input")
	}
}

func TestSummaryCompaction_summarizeMessages(t *testing.T) {
	strategy := NewSummaryCompaction()

	messages := []Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
	}

	summary := strategy.summarizeMessages(messages, 100)
	if summary == "" {
		t.Error("summarizeMessages() should return non-empty string")
	}

	shortSummary := strategy.summarizeMessages(messages, 5)
	if len(shortSummary) > 5 {
		t.Errorf("summarizeMessages() should truncate to maxLen, got %d", len(shortSummary))
	}
}

func TestPriorityBasedCompaction_Compact(t *testing.T) {
	strategy := NewPriorityBasedCompaction()
	counter := &DefaultCounter{}

	messages := []Message{
		{Role: "system", Content: "System"},
		{Role: "user", Content: "User"},
		{Role: "assistant", Content: "Assistant"},
	}

	compacted, err := strategy.Compact(messages, 1000, counter)
	if err != nil {
		t.Errorf("Compact() error = %v", err)
	}
	if len(compacted) == 0 {
		t.Error("Compact() should return messages")
	}
}

func TestPriorityBasedCompaction_CompactEmpty(t *testing.T) {
	strategy := NewPriorityBasedCompaction()
	counter := &DefaultCounter{}

	compacted, err := strategy.Compact([]Message{}, 100, counter)
	if err != nil {
		t.Errorf("Compact() error = %v", err)
	}
	if len(compacted) != 0 {
		t.Error("Compact() should return empty for empty input")
	}
}

func TestPriorityBasedCompaction_Name(t *testing.T) {
	strategy := NewPriorityBasedCompaction()
	if strategy.Name() != "priority" {
		t.Errorf("Name() = %v, want priority", strategy.Name())
	}
}

func TestDefaultCounter_Count(t *testing.T) {
	counter := &DefaultCounter{}

	messages := []Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
	}

	count, err := counter.Count(messages)
	if err != nil {
		t.Errorf("Count() error = %v", err)
	}
	if count <= 0 {
		t.Error("Count() should return positive count")
	}
}

func TestDefaultCounter_CountMessage(t *testing.T) {
	counter := &DefaultCounter{}

	msg := Message{Role: "user", Content: "Test message", Name: "User1"}

	count, err := counter.CountMessage(msg)
	if err != nil {
		t.Errorf("CountMessage() error = %v", err)
	}
	if count <= 0 {
		t.Error("CountMessage() should return positive count")
	}
}

func TestDefaultCounter_CountEmptyMessage(t *testing.T) {
	counter := &DefaultCounter{}

	msg := Message{Role: "", Content: ""}

	count, err := counter.CountMessage(msg)
	if err != nil {
		t.Errorf("CountMessage() error = %v", err)
	}
	if count < 4 {
		t.Errorf("CountMessage() should return at least 4 for overhead, got %d", count)
	}
}

func TestWarningLevel_Constants(t *testing.T) {
	if WarningLevelNone != "none" {
		t.Errorf("WarningLevelNone = %v, want none", WarningLevelNone)
	}
	if WarningLevelLow != "low" {
		t.Errorf("WarningLevelLow = %v, want low", WarningLevelLow)
	}
	if WarningLevelMedium != "medium" {
		t.Errorf("WarningLevelMedium = %v, want medium", WarningLevelMedium)
	}
	if WarningLevelHigh != "high" {
		t.Errorf("WarningLevelHigh = %v, want high", WarningLevelHigh)
	}
	if WarningLevelCritical != "critical" {
		t.Errorf("WarningLevelCritical = %v, want critical", WarningLevelCritical)
	}
}

func TestLastNCompaction_Name(t *testing.T) {
	strategy := &LastNCompaction{KeepCount: 5}
	if strategy.Name() != "last_n" {
		t.Errorf("Name() = %v, want last_n", strategy.Name())
	}
}

func TestSummaryCompaction_Name(t *testing.T) {
	strategy := NewSummaryCompaction()
	if strategy.Name() != "summary" {
		t.Errorf("Name() = %v, want summary", strategy.Name())
	}
}

func TestCompactionStrategy_Interface(t *testing.T) {
	var _ CompactionStrategy = (*LastNCompaction)(nil)
	var _ CompactionStrategy = (*SummaryCompaction)(nil)
	var _ CompactionStrategy = (*PriorityBasedCompaction)(nil)
}

func TestTokenCounter_Interface(t *testing.T) {
	var _ TokenCounter = (*DefaultCounter)(nil)
}
