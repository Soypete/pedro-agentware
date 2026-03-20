package middleware

import (
	"sync"
	"time"
)

type Auditor interface {
	Record(decision Decision)
	GetViolations(since time.Time) []Decision
	GetDecisions(tool string, since time.Time) []Decision
	Close() error
}

type inMemoryAuditor struct {
	decisions []Decision
	mu        sync.RWMutex
	maxSize   int
}

func NewInMemoryAuditor(maxSize int) Auditor {
	if maxSize <= 0 {
		maxSize = 10000
	}
	return &inMemoryAuditor{
		decisions: make([]Decision, 0, maxSize),
		maxSize:   maxSize,
	}
}

func (a *inMemoryAuditor) Record(decision Decision) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.decisions) >= a.maxSize {
		a.decisions = a.decisions[1:]
	}

	a.decisions = append(a.decisions, decision)
}

func (a *inMemoryAuditor) GetViolations(since time.Time) []Decision {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var violations []Decision
	for _, d := range a.decisions {
		if d.Timestamp.After(since) && d.Action == ActionDeny {
			violations = append(violations, d)
		}
	}
	return violations
}

func (a *inMemoryAuditor) GetDecisions(tool string, since time.Time) []Decision {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var results []Decision
	for _, d := range a.decisions {
		if d.Timestamp.After(since) && d.Tool == tool {
			results = append(results, d)
		}
	}
	return results
}

func (a *inMemoryAuditor) Close() error {
	return nil
}

type NoOpAuditor struct{}

func (a NoOpAuditor) Record(decision Decision)                             {}
func (a NoOpAuditor) GetViolations(since time.Time) []Decision             { return nil }
func (a NoOpAuditor) GetDecisions(tool string, since time.Time) []Decision { return nil }
func (a NoOpAuditor) Close() error                                         { return nil }
