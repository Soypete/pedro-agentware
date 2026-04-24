package checkpointer

import (
	"context"
	"sync"

	"github.com/soypete/pedro-agentware/go/agent"
)

type MemoryCheckpointer[T any] struct {
	mu    sync.RWMutex
	state map[string]*memoryState[T]
}

type memoryState[T any] struct {
	state       *agent.State[T]
	currentNode string
}

func NewMemory[T any]() *MemoryCheckpointer[T] {
	return &MemoryCheckpointer[T]{
		state: make(map[string]*memoryState[T]),
	}
}

func (c *MemoryCheckpointer[T]) Save(ctx context.Context, threadID string, state *agent.State[T], node string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state[threadID] = &memoryState[T]{
		state:       state.Clone(),
		currentNode: node,
	}
	return nil
}

func (c *MemoryCheckpointer[T]) Get(ctx context.Context, threadID string) (*agent.State[T], string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if s, ok := c.state[threadID]; ok {
		return s.state, s.currentNode, nil
	}
	return nil, "", nil
}

func (c *MemoryCheckpointer[T]) List(ctx context.Context) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ids := make([]string, 0, len(c.state))
	for id := range c.state {
		ids = append(ids, id)
	}
	return ids, nil
}

func (c *MemoryCheckpointer[T]) Delete(ctx context.Context, threadID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.state, threadID)
	return nil
}
