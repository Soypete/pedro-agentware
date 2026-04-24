package agent

import (
	"fmt"
	"strings"
)

type Graph[T any] interface {
	AddNode(name string, fn NodeFunc[T]) Graph[T]
	AddEdge(from, to string) Graph[T]
	AddConditionalEdge(from string, fn ConditionalFunc[T], branches []string) Graph[T]
	SetEntryPoint(node string) Graph[T]
	SetFinishPoint(node string) Graph[T]
	Compile() (Runner[T], error)
}

type graph[T any] struct {
	nodes            map[string]NodeFunc[T]
	edges            map[string][]string
	conditionalEdges map[string]conditionalEdge[T]
	entryPoint       string
	finishPoints     map[string]bool
}

type conditionalEdge[T any] struct {
	fn       ConditionalFunc[T]
	branches []string
}

func NewGraph[T any]() Graph[T] {
	return &graph[T]{
		nodes:            make(map[string]NodeFunc[T]),
		edges:            make(map[string][]string),
		conditionalEdges: make(map[string]conditionalEdge[T]),
		finishPoints:     make(map[string]bool),
	}
}

func (g *graph[T]) AddNode(name string, fn NodeFunc[T]) Graph[T] {
	g.nodes[name] = fn
	return g
}

func (g *graph[T]) AddEdge(from, to string) Graph[T] {
	if _, ok := g.nodes[from]; !ok {
		panic(fmt.Errorf("node %s does not exist", from))
	}
	if _, ok := g.nodes[to]; !ok {
		panic(fmt.Errorf("node %s does not exist", to))
	}
	g.edges[from] = append(g.edges[from], to)
	return g
}

func (g *graph[T]) AddConditionalEdge(from string, fn ConditionalFunc[T], branches []string) Graph[T] {
	if _, ok := g.nodes[from]; !ok {
		panic(fmt.Errorf("node %s does not exist", from))
	}
	for _, b := range branches {
		if _, ok := g.nodes[b]; !ok {
			panic(fmt.Errorf("branch node %s does not exist", b))
		}
	}
	g.conditionalEdges[from] = conditionalEdge[T]{
		fn:       fn,
		branches: branches,
	}
	return g
}

func (g *graph[T]) SetEntryPoint(node string) Graph[T] {
	if _, ok := g.nodes[node]; !ok {
		panic(fmt.Errorf("entry point node %s does not exist", node))
	}
	g.entryPoint = node
	return g
}

func (g *graph[T]) SetFinishPoint(node string) Graph[T] {
	if _, ok := g.nodes[node]; !ok {
		panic(fmt.Errorf("finish point node %s does not exist", node))
	}
	g.finishPoints[node] = true
	return g
}

func (g *graph[T]) Compile() (Runner[T], error) {
	if g.entryPoint == "" {
		return nil, fmt.Errorf("no entry point set")
	}
	if len(g.finishPoints) == 0 {
		return nil, fmt.Errorf("no finish points set")
	}
	return &runner[T]{
		graph:        g,
		checkpointer: nil,
	}, nil
}

func (g *graph[T]) Validate() error {
	var missing []string
	for name := range g.edges {
		if _, ok := g.nodes[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("undefined nodes referenced in edges: %s", strings.Join(missing, ", "))
	}
	return nil
}
