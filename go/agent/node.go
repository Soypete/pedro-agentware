package agent

import (
	"context"
	"errors"
)

var (
	ErrStop      = errors.New("graph completed")
	ErrInterrupt = errors.New("execution interrupted")
)

type NodeFunc[T any] func(context.Context, *State[T]) (*State[T], string, error)

type NodeResult[T any] struct {
	State      *State[T]
	NextNode   string
	Error      error
	Interrupts []Interrupt
}

type Interrupt struct {
	Type     string
	Message  string
	Payload  map[string]any
	Approved bool
}

func END() string {
	return "__end__"
}

type ConditionalFunc[T any] func(context.Context, *State[T]) (string, error)
