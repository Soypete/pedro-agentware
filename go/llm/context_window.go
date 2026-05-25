package llm

import (
	"context"
	"sort"
	"sync"
)

type ThresholdCallback func(tokens, budget int, pct float64) string

type CompactCallback func(CompactEvent)

type ContextWindowOption func(*ContextWindowManager)

type ContextWindowManager struct {
	mu              sync.RWMutex
	contextWindow   int
	compactionRatio float64
	counter         TokenCounter
	lastKnownTokens *int
	thresholds      []float64
	onThreshold     ThresholdCallback
	onCompact       CompactCallback
	firedThresholds map[float64]struct{}
}

func NewContextWindowManager(contextWindow int, counter TokenCounter, opts ...ContextWindowOption) *ContextWindowManager {
	if counter == nil {
		counter = DefaultCounter
	}
	m := &ContextWindowManager{
		contextWindow:   contextWindow,
		compactionRatio: 0.75,
		counter:         counter,
		lastKnownTokens: nil,
		thresholds:      []float64{0.65, 0.80},
		onThreshold:     DefaultThresholdCallback,
		firedThresholds: make(map[float64]struct{}),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func WithThresholds(thresholds []float64, cb ThresholdCallback) ContextWindowOption {
	return func(m *ContextWindowManager) {
		m.mu.Lock()
		defer m.mu.Unlock()
		if len(thresholds) > 0 {
			thresholdsCopy := make([]float64, len(thresholds))
			copy(thresholdsCopy, thresholds)
			sort.Float64s(thresholdsCopy)
			m.thresholds = thresholdsCopy
		}
		if cb != nil {
			m.onThreshold = cb
		}
	}
}

func WithOnCompact(cb CompactCallback) ContextWindowOption {
	return func(m *ContextWindowManager) {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.onCompact = cb
	}
}

func DefaultThresholdCallback(tokens, budget int, pct float64) string {
	switch {
	case pct >= 0.80:
		return "Context is nearly full. Summarize critical findings now and prioritize completing the current task."
	case pct >= 0.65:
		return "Context is filling up. Be concise and front-load important information."
	default:
		return ""
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

	tokensBefore := m.counter(messages)
	messagesBefore := len(messages)
	budgetTokens := m.contextWindow
	targetTokens := int(float64(budgetTokens) * m.compactionRatio)
	strategy := NewTieredCompact()
	compacted, err := strategy.Compact(messages, targetTokens, m.counter)
	if err != nil {
		return nil, err
	}
	tokensAfter := m.counter(compacted)
	messagesAfter := len(compacted)

	phaseReached := 1
	if strategy != nil {
		phaseReached = strategy.LastPhase()
	}

	stepIndex := 0
	if len(messages) > 0 && messages[len(messages)-1].Meta.StepIndex != nil {
		stepIndex = *messages[len(messages)-1].Meta.StepIndex
	}

	if m.onCompact != nil {
		m.onCompact(CompactEvent{
			StepIndex:      stepIndex,
			TokensBefore:   tokensBefore,
			TokensAfter:    tokensAfter,
			BudgetTokens:   budgetTokens,
			MessagesBefore: messagesBefore,
			MessagesAfter:  messagesAfter,
			PhaseReached:   phaseReached,
			StrategyName:   strategy.Name(),
		})
	}

	m.lastKnownTokens = nil
	m.firedThresholds = make(map[float64]struct{})
	return compacted, nil
}

func (m *ContextWindowManager) CheckThresholds(ctx context.Context, messages []Message) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	currentTokens := m.estimateTokensLocked(messages)
	budget := m.contextWindow
	if currentTokens == 0 || budget == 0 {
		return ""
	}

	pct := float64(currentTokens) / float64(budget)

	for _, threshold := range m.thresholds {
		if pct >= threshold {
			if _, fired := m.firedThresholds[threshold]; !fired {
				m.firedThresholds[threshold] = struct{}{}
				return m.onThreshold(currentTokens, budget, pct)
			}
		}
	}
	return ""
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
