package agent

import "sync"

type State[T any] struct {
	data     T
	mu       sync.RWMutex
	reducers map[string]ReducerFunc[T]
}

type ReducerFunc[T any] func(existing, incoming T) T

func NewState[T any](initial T) *State[T] {
	return &State[T]{
		data:     initial,
		reducers: make(map[string]ReducerFunc[T]),
	}
}

func (s *State[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}

func (s *State[T]) Set(data T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = data
}

func (s *State[T]) Update(fn func(T) T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = fn(s.data)
}

func (s *State[T]) RegisterReducer(name string, reducer ReducerFunc[T]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reducers[name] = reducer
}

func (s *State[T]) ApplyReducer(name string, incoming T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if reducer, ok := s.reducers[name]; ok {
		s.data = reducer(s.data, incoming)
	}
}

func (s *State[T]) Clone() *State[T] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return &State[T]{
		data:     s.data,
		reducers: s.reducers,
	}
}

func AppendReducer[T any](existing, incoming []T) []T {
	return append(existing, incoming...)
}

func MergeMapReducer[K comparable, V any](existing, incoming map[K]V) map[K]V {
	result := make(map[K]V, len(existing)+len(incoming))
	for k, v := range existing {
		result[k] = v
	}
	for k, v := range incoming {
		result[k] = v
	}
	return result
}
