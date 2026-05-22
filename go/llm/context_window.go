package llm

import (
	"sync"
)

type ContextWindowManager struct {
	mu              sync.RWMutex
	contextWindow   int
	compactionRatio float64
	counter         TokenCounter
	lastKnownTokens *int
}

func NewContextWindowManager(contextWindow int, counter TokenCounter) *ContextWindowManager {
	if counter == nil {
		counter = DefaultCounter
	}
	return &ContextWindowManager{
		contextWindow:   contextWindow,
		compactionRatio: 0.75,
		counter:         counter,
		lastKnownTokens: nil,
	}
}

func (m *ContextWindowManager) SetCompactionRatio(ratio float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.compactionRatio = ratio
}

func (m *ContextWindowManager) UpdateTokenCount(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastKnownTokens = &n
}

func (m *ContextWindowManager) Check(messages []Message) (int, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	currentTokens := m.estimateTokensLocked(messages)
	threshold := int(float64(m.contextWindow) * m.compactionRatio)
	return currentTokens, currentTokens >= threshold
}

func (m *ContextWindowManager) ShouldCompact(messages []Message) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	currentTokens := m.estimateTokensLocked(messages)
	threshold := int(float64(m.contextWindow) * m.compactionRatio)
	return currentTokens > threshold
}

func (m *ContextWindowManager) Compact(messages []Message) ([]Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	targetTokens := int(float64(m.contextWindow) * m.compactionRatio)
	strategy := NewTieredCompact()
	compacted, err := strategy.Compact(messages, targetTokens, m.counter)
	if err != nil {
		return nil, err
	}
	m.lastKnownTokens = nil
	return compacted, nil
}

func (m *ContextWindowManager) estimateTokensLocked(messages []Message) int {
	if m.lastKnownTokens != nil {
		return *m.lastKnownTokens
	}
	return m.counter(messages)
}

func DefaultCounter(messages []Message) int {
	total := 0
	for _, m := range messages {
		overhead := len(string(m.Role)) + 4
		for _, tc := range m.ToolCalls {
			overhead += len(tc.Name) + 1
		}
		total += (len(m.Content) / 4) + overhead
	}
	return total
}
