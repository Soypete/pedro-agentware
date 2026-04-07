package middleware

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	counters map[string]*rateCounter
	windows  map[string]time.Duration
}

type rateCounter struct {
	count     int
	firstCall time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		counters: make(map[string]*rateCounter),
		windows:  make(map[string]time.Duration),
	}
}

func (r *RateLimiter) SetWindow(key string, window time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.windows[key] = window
}

func (r *RateLimiter) Allow(key string, limit int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	window := r.windows[key]
	if window == 0 {
		window = time.Minute
	}

	now := time.Now()
	counter, exists := r.counters[key]

	if !exists || now.Sub(counter.firstCall) > window {
		r.counters[key] = &rateCounter{
			count:     1,
			firstCall: now,
		}
		return true
	}

	if counter.count >= limit {
		return false
	}

	counter.count++
	return true
}

func (r *RateLimiter) Reset(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.counters, key)
}
