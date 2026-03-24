package windowmanager

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrEmptyMessages = errors.New("messages cannot be empty")
var ErrInvalidReservedTokens = errors.New("reserved_tokens must be less than max_tokens")

type Message struct {
	Role    string
	Content string
	Name    string
}

type ModelSpec struct {
	Name            string
	MaxTokens       int
	ReservedTokens  int
	TokenMultiplier float64
}

type WarningLevel string

const (
	WarningLevelNone     WarningLevel = "none"
	WarningLevelLow      WarningLevel = "low"
	WarningLevelMedium   WarningLevel = "medium"
	WarningLevelHigh     WarningLevel = "high"
	WarningLevelCritical WarningLevel = "critical"
)

type ContextStatus struct {
	UsedTokens      int
	RemainingTokens int
	MaxTokens       int
	ReservedTokens  int
	WarningLevel    WarningLevel
	MessageCount    int
}

type CompactionStrategy interface {
	Compact(messages []Message, targetTokens int, counter TokenCounter) ([]Message, error)
	Name() string
}

type TokenCounter interface {
	Count(messages []Message) (int, error)
	CountMessage(msg Message) (int, error)
}

type ContextWindowManager struct {
	model     ModelSpec
	strategy  CompactionStrategy
	counter   TokenCounter
	mu        sync.RWMutex
	lastCheck time.Time
}

func NewContextWindowManager(model ModelSpec, strategy CompactionStrategy, counter TokenCounter) (*ContextWindowManager, error) {
	if model.MaxTokens <= 0 {
		model.MaxTokens = 4096
	}
	if model.ReservedTokens < 0 {
		model.ReservedTokens = 0
	}
	if model.ReservedTokens >= model.MaxTokens {
		return nil, ErrInvalidReservedTokens
	}
	if model.TokenMultiplier == 0 {
		model.TokenMultiplier = 4.0
	}

	if counter == nil {
		counter = NewDefaultCounter()
	}

	return &ContextWindowManager{
		model:    model,
		strategy: strategy,
		counter:  counter,
	}, nil
}

func (m *ContextWindowManager) Check(ctx context.Context, messages []Message) (*ContextStatus, error) {
	if len(messages) == 0 {
		return &ContextStatus{
			UsedTokens:      0,
			RemainingTokens: m.model.MaxTokens - m.model.ReservedTokens,
			MaxTokens:       m.model.MaxTokens,
			ReservedTokens:  m.model.ReservedTokens,
			WarningLevel:    WarningLevelNone,
			MessageCount:    0,
		}, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	tokenCount, err := m.counter.Count(messages)
	if err != nil {
		return nil, err
	}

	available := m.model.MaxTokens - m.model.ReservedTokens
	remaining := available - tokenCount

	return &ContextStatus{
		UsedTokens:      tokenCount,
		RemainingTokens: remaining,
		MaxTokens:       m.model.MaxTokens,
		ReservedTokens:  m.model.ReservedTokens,
		WarningLevel:    m.calculateWarningLevel(remaining, available),
		MessageCount:    len(messages),
	}, nil
}

func (m *ContextWindowManager) ShouldCompact(ctx context.Context, messages []Message) (bool, error) {
	status, err := m.Check(ctx, messages)
	if err != nil {
		return false, err
	}

	threshold := m.model.MaxTokens / 4
	return status.RemainingTokens < threshold, nil
}

func (m *ContextWindowManager) Compact(ctx context.Context, messages []Message) ([]Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	available := m.model.MaxTokens - m.model.ReservedTokens
	targetTokens := int(float64(available) * 0.75)

	return m.strategy.Compact(messages, targetTokens, m.counter)
}

func (m *ContextWindowManager) CompactIfNeeded(ctx context.Context, messages []Message) ([]Message, bool, error) {
	shouldCompact, err := m.ShouldCompact(ctx, messages)
	if err != nil {
		return nil, false, err
	}

	if !shouldCompact {
		return messages, false, nil
	}

	compacted, err := m.Compact(ctx, messages)
	if err != nil {
		return nil, false, err
	}

	return compacted, true, nil
}

func (m *ContextWindowManager) calculateWarningLevel(remaining, available int) WarningLevel {
	if available == 0 {
		return WarningLevelCritical
	}

	ratio := float64(remaining) / float64(available)

	switch {
	case ratio <= 0.1:
		return WarningLevelCritical
	case ratio <= 0.25:
		return WarningLevelHigh
	case ratio <= 0.5:
		return WarningLevelMedium
	case ratio <= 0.75:
		return WarningLevelLow
	default:
		return WarningLevelNone
	}
}

type LastNCompaction struct {
	KeepCount int
}

func (c *LastNCompaction) Name() string {
	return "last_n"
}

func (c *LastNCompaction) Compact(messages []Message, targetTokens int, counter TokenCounter) ([]Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	keepCount := c.KeepCount
	if keepCount <= 0 {
		keepCount = len(messages) / 2
	}
	if keepCount < 1 {
		keepCount = 1
	}

	if keepCount >= len(messages) {
		count, _ := counter.Count(messages)
		if count <= targetTokens {
			return messages, nil
		}
		keepCount = len(messages) / 2
	}

	for i := keepCount; i >= 1; i-- {
		subset := messages[len(messages)-i:]
		count, err := counter.Count(subset)
		if err != nil {
			continue
		}
		if count <= targetTokens {
			return subset, nil
		}
	}

	return messages[len(messages)-1:], nil
}

type SummaryCompaction struct {
	summaryPrompt string
	maxSummaryLen int
}

func NewSummaryCompaction() *SummaryCompaction {
	return &SummaryCompaction{
		summaryPrompt: "Summarize the following conversation concisely, preserving key information:",
		maxSummaryLen: 500,
	}
}

func (c *SummaryCompaction) Name() string {
	return "summary"
}

func (c *SummaryCompaction) Compact(messages []Message, targetTokens int, counter TokenCounter) ([]Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	var systemMsg *Message
	var otherMsgs []Message

	for i, msg := range messages {
		if msg.Role == "system" {
			systemMsg = &messages[i]
		} else {
			otherMsgs = append(otherMsgs, msg)
		}
	}

	if len(otherMsgs) == 0 {
		return messages, nil
	}

	count, _ := counter.Count(otherMsgs)
	if count <= targetTokens {
		if systemMsg != nil {
			result := append([]Message{*systemMsg}, otherMsgs...)
			return result, nil
		}
		return messages, nil
	}

	var result []Message
	if systemMsg != nil {
		result = append(result, *systemMsg)
	}

	summaryTokens := targetTokens / 4
	summaryChars := int(float64(summaryTokens) * 3)

	summary := c.summarizeMessages(otherMsgs, summaryChars)
	summaryMsg := Message{
		Role:    "system",
		Content: "[Previous conversation summarized: " + summary + "]",
	}
	result = append(result, summaryMsg)

	currentCount, _ := counter.Count(result)
	if currentCount > targetTokens {
		keepCount := len(otherMsgs) / 3
		if keepCount < 1 {
			keepCount = 1
		}
		recentMsgs := otherMsgs[len(otherMsgs)-keepCount:]
		result = append(result, recentMsgs...)
	}

	return result, nil
}

func (c *SummaryCompaction) summarizeMessages(messages []Message, maxLen int) string {
	var content string
	for _, msg := range messages {
		prefix := msg.Role
		if msg.Name != "" {
			prefix = msg.Name
		}
		content += prefix + ": " + msg.Content + "\n"
	}

	if len(content) <= maxLen {
		return content
	}

	truncated := content[:maxLen-3]
	lastSpace := -1
	for i := len(truncated) - 1; i >= 0; i-- {
		if truncated[i] == ' ' || truncated[i] == '\n' {
			lastSpace = i
			break
		}
	}

	if lastSpace > 0 {
		return truncated[:lastSpace] + "..."
	}

	return truncated + "..."
}

type Priority int

const (
	PrioritySystem Priority = iota
	PriorityHigh
	PriorityMedium
	PriorityLow
)

type PriorityMessage struct {
	Message
	Priority Priority
}

type PriorityBasedCompaction struct {
	keepSystem bool
}

func NewPriorityBasedCompaction() *PriorityBasedCompaction {
	return &PriorityBasedCompaction{
		keepSystem: true,
	}
}

func (c *PriorityBasedCompaction) Name() string {
	return "priority"
}

func (c *PriorityBasedCompaction) Compact(messages []Message, targetTokens int, counter TokenCounter) ([]Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	var priorityMsgs []PriorityMessage
	for _, msg := range messages {
		priority := PriorityMedium
		switch msg.Role {
		case "system":
			if c.keepSystem {
				priority = PrioritySystem
			} else {
				priority = PriorityLow
			}
		case "assistant":
			priority = PriorityHigh
		case "user":
			priority = PriorityMedium
		default:
			priority = PriorityLow
		}
		priorityMsgs = append(priorityMsgs, PriorityMessage{Message: msg, Priority: priority})
	}

	sorted := make([]PriorityMessage, len(priorityMsgs))
	copy(sorted, priorityMsgs)

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Priority > sorted[j].Priority {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	var result []Message
	for _, pm := range sorted {
		result = append(result, pm.Message)
		count, _ := counter.Count(result)
		if count > targetTokens {
			result = result[:len(result)-1]
			break
		}
	}

	if len(result) == 0 && len(messages) > 0 {
		result = append(result, messages[len(messages)-1])
	}

	return result, nil
}

type DefaultCounter struct{}

func NewDefaultCounter() *DefaultCounter {
	return &DefaultCounter{}
}

func (c *DefaultCounter) Count(messages []Message) (int, error) {
	total := 0
	for _, msg := range messages {
		count, err := c.CountMessage(msg)
		if err != nil {
			return 0, err
		}
		total += count
	}
	return total, nil
}

func (c *DefaultCounter) CountMessage(msg Message) (int, error) {
	content := msg.Role
	if msg.Name != "" {
		content += msg.Name
	}
	content += msg.Content

	tokens := len(content) / 4
	return tokens + 4, nil
}
