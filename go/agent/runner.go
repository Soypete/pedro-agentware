package agent

import (
	"context"
	"fmt"
)

type Runner[T any] interface {
	Run(ctx context.Context, initialState T) (*State[T], error)
	RunWithUpdates(ctx context.Context, initialState T, updates chan<- StateUpdate[T]) error
	GetState(ctx context.Context, threadID string) (*State[T], error)
	UpdateState(ctx context.Context, threadID string, state T) error
}

type StateUpdate[T any] struct {
	Node      string
	State     *State[T]
	IsDelta   bool
	Completed bool
}

type Checkpointer[T any] interface {
	Save(ctx context.Context, threadID string, state *State[T], node string) error
	Get(ctx context.Context, threadID string) (*State[T], string, error)
	List(ctx context.Context) ([]string, error)
	Delete(ctx context.Context, threadID string) error
}

type runner[T any] struct {
	graph        *graph[T]
	checkpointer Checkpointer[T]
}

func (r *runner[T]) Run(ctx context.Context, initialState T) (*State[T], error) {
	state := NewState(initialState)
	currentNode := r.graph.entryPoint

	for {
		select {
		case <-ctx.Done():
			return state, ctx.Err()
		default:
		}

		if r.graph.finishPoints[currentNode] {
			return state, nil
		}

		nodeFn, ok := r.graph.nodes[currentNode]
		if !ok {
			return state, fmt.Errorf("node %s not found", currentNode)
		}

		newState, nextNode, err := nodeFn(ctx, state)
		if err != nil {
			return newState, err
		}

		state = newState

		if nextNode == END() {
			return state, nil
		}

		if nextNode == "" {
			return state, fmt.Errorf("node %s returned empty next node", currentNode)
		}

		currentNode = nextNode
	}
}

func (r *runner[T]) RunWithUpdates(ctx context.Context, initialState T, updates chan<- StateUpdate[T]) error {
	state := NewState(initialState)
	currentNode := r.graph.entryPoint

	updates <- StateUpdate[T]{
		Node:      currentNode,
		State:     state,
		Completed: false,
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if r.graph.finishPoints[currentNode] {
			updates <- StateUpdate[T]{
				Node:      currentNode,
				State:     state,
				Completed: true,
			}
			close(updates)
			return nil
		}

		nodeFn, ok := r.graph.nodes[currentNode]
		if !ok {
			return fmt.Errorf("node %s not found", currentNode)
		}

		newState, nextNode, err := nodeFn(ctx, state)
		if err != nil {
			return err
		}

		state = newState

		updates <- StateUpdate[T]{
			Node:      currentNode,
			State:     state,
			IsDelta:   true,
			Completed: false,
		}

		if nextNode == END() {
			close(updates)
			return nil
		}

		if nextNode == "" {
			return fmt.Errorf("node %s returned empty next node", currentNode)
		}

		currentNode = nextNode
	}
}

func (r *runner[T]) GetState(ctx context.Context, threadID string) (*State[T], error) {
	if r.checkpointer == nil {
		return nil, fmt.Errorf("checkpointer not configured")
	}
	state, _, err := r.checkpointer.Get(ctx, threadID)
	return state, err
}

func (r *runner[T]) UpdateState(ctx context.Context, threadID string, state T) error {
	if r.checkpointer == nil {
		return fmt.Errorf("checkpointer not configured")
	}
	return r.checkpointer.Save(ctx, threadID, NewState(state), "")
}

type RunnerOption[T any] func(*runner[T])

func WithCheckpointer[T any](cp Checkpointer[T]) RunnerOption[T] {
	return func(r *runner[T]) {
		r.checkpointer = cp
	}
}
