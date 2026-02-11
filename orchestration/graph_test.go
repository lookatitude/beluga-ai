package orchestration

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
)

func TestGraph_Linear(t *testing.T) {
	g := NewGraph()
	g.AddNode("a", newStep(func(input any) (any, error) { return fmt.Sprintf("a(%v)", input), nil }))
	g.AddNode("b", newStep(func(input any) (any, error) { return fmt.Sprintf("b(%v)", input), nil }))
	g.AddNode("c", newStep(func(input any) (any, error) { return fmt.Sprintf("c(%v)", input), nil }))
	g.AddEdge(Edge{From: "a", To: "b"})
	g.AddEdge(Edge{From: "b", To: "c"})
	g.SetEntry("a")

	result, err := g.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "c(b(a(x)))"
	if result != expected {
		t.Fatalf("expected %q, got %v", expected, result)
	}
}

func TestGraph_Branching(t *testing.T) {
	g := NewGraph()
	g.AddNode("start", newStep(func(input any) (any, error) { return input, nil }))
	g.AddNode("left", newStep(func(input any) (any, error) { return "left", nil }))
	g.AddNode("right", newStep(func(input any) (any, error) { return "right", nil }))

	g.AddEdge(Edge{From: "start", To: "left", Condition: func(v any) bool { return v == "go-left" }})
	g.AddEdge(Edge{From: "start", To: "right", Condition: func(v any) bool { return v == "go-right" }})
	g.SetEntry("start")

	// Test left branch.
	result, err := g.Invoke(context.Background(), "go-left")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "left" {
		t.Fatalf("expected left, got %v", result)
	}

	// Test right branch.
	result, err = g.Invoke(context.Background(), "go-right")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "right" {
		t.Fatalf("expected right, got %v", result)
	}
}

func TestGraph_ConditionalEdges_NoMatch(t *testing.T) {
	g := NewGraph()
	g.AddNode("start", newStep(func(input any) (any, error) { return input, nil }))
	g.AddNode("next", newStep(func(input any) (any, error) { return "next", nil }))

	// Only matches "go".
	g.AddEdge(Edge{From: "start", To: "next", Condition: func(v any) bool { return v == "go" }})
	g.SetEntry("start")

	// No matching condition — terminal at start.
	result, err := g.Invoke(context.Background(), "stop")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "stop" {
		t.Fatalf("expected stop, got %v", result)
	}
}

func TestGraph_UnconditionalEdge(t *testing.T) {
	g := NewGraph()
	g.AddNode("a", newStep(func(input any) (any, error) { return "a", nil }))
	g.AddNode("b", newStep(func(input any) (any, error) { return "b", nil }))

	// nil condition = unconditional.
	g.AddEdge(Edge{From: "a", To: "b"})
	g.SetEntry("a")

	result, err := g.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "b" {
		t.Fatalf("expected b, got %v", result)
	}
}

func TestGraph_NoEntry(t *testing.T) {
	g := NewGraph()
	g.AddNode("a", newStep(func(input any) (any, error) { return input, nil }))

	_, err := g.Invoke(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error for no entry node")
	}
}

func TestGraph_DuplicateNode(t *testing.T) {
	g := NewGraph()
	g.AddNode("a", newStep(func(input any) (any, error) { return input, nil }))
	err := g.AddNode("a", newStep(func(input any) (any, error) { return input, nil }))
	if err == nil {
		t.Fatal("expected error for duplicate node")
	}
}

func TestGraph_EmptyNodeName(t *testing.T) {
	g := NewGraph()
	err := g.AddNode("", newStep(func(input any) (any, error) { return input, nil }))
	if err == nil {
		t.Fatal("expected error for empty node name")
	}
}

func TestGraph_UnknownSourceEdge(t *testing.T) {
	g := NewGraph()
	g.AddNode("b", newStep(func(input any) (any, error) { return input, nil }))

	err := g.AddEdge(Edge{From: "a", To: "b"})
	if err == nil {
		t.Fatal("expected error for unknown source node")
	}
}

func TestGraph_UnknownTargetEdge(t *testing.T) {
	g := NewGraph()
	g.AddNode("a", newStep(func(input any) (any, error) { return input, nil }))

	err := g.AddEdge(Edge{From: "a", To: "b"})
	if err == nil {
		t.Fatal("expected error for unknown target node")
	}
}

func TestGraph_UnknownEntry(t *testing.T) {
	g := NewGraph()
	err := g.SetEntry("unknown")
	if err == nil {
		t.Fatal("expected error for unknown entry node")
	}
}

func TestGraph_MaxDepth(t *testing.T) {
	g := NewGraph()
	g.AddNode("a", newStep(func(input any) (any, error) { return input, nil }))
	g.AddNode("b", newStep(func(input any) (any, error) { return input, nil }))

	// Create a cycle: a → b → a.
	g.AddEdge(Edge{From: "a", To: "b"})
	g.AddEdge(Edge{From: "b", To: "a"})
	g.SetEntry("a")

	_, err := g.Invoke(context.Background(), "x")
	if err == nil {
		t.Fatal("expected max depth error")
	}
	if !errors.Is(err, err) { // Just check it's not nil.
		t.Logf("got expected error: %v", err)
	}
}

func TestGraph_NodeError(t *testing.T) {
	errBoom := errors.New("boom")
	g := NewGraph()
	g.AddNode("a", newStep(func(_ any) (any, error) { return nil, errBoom }))
	g.SetEntry("a")

	_, err := g.Invoke(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errBoom) {
		t.Fatalf("expected boom error, got %v", err)
	}
}

func TestGraph_Stream_Linear(t *testing.T) {
	g := NewGraph()
	g.AddNode("a", newStep(func(input any) (any, error) { return fmt.Sprintf("a(%v)", input), nil }))
	g.AddNode("b", newStep(func(input any) (any, error) { return fmt.Sprintf("b(%v)", input), nil }))
	g.AddEdge(Edge{From: "a", To: "b"})
	g.SetEntry("a")

	var results []any
	for val, err := range g.Stream(context.Background(), "x") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		results = append(results, val)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	// The last node (b) is terminal and gets streamed.
	last := results[len(results)-1]
	if last != "b(a(x))" {
		t.Fatalf("expected b(a(x)), got %v", last)
	}
}

func TestGraph_Stream_NoEntry(t *testing.T) {
	g := NewGraph()
	g.AddNode("a", newStep(func(input any) (any, error) { return input, nil }))

	for _, err := range g.Stream(context.Background(), "x") {
		if err == nil {
			t.Fatal("expected error for no entry node")
		}
		return
	}
	t.Fatal("expected at least one stream result")
}

func TestGraph_Stream_NodeError(t *testing.T) {
	errBoom := errors.New("boom")
	g := NewGraph()
	g.AddNode("a", newStep(func(_ any) (any, error) { return "a", nil }))
	g.AddNode("b", newStep(func(_ any) (any, error) { return nil, errBoom }))
	g.AddEdge(Edge{From: "a", To: "b"})
	g.SetEntry("a")

	for _, err := range g.Stream(context.Background(), "x") {
		if err != nil {
			// Error from node invoke in non-terminal path.
			return
		}
	}
}

func TestGraph_Stream_MaxDepth(t *testing.T) {
	g := NewGraph()
	g.AddNode("a", newStep(func(input any) (any, error) { return input, nil }))
	g.AddNode("b", newStep(func(input any) (any, error) { return input, nil }))
	g.AddEdge(Edge{From: "a", To: "b"})
	g.AddEdge(Edge{From: "b", To: "a"})
	g.SetEntry("a")

	for _, err := range g.Stream(context.Background(), "x") {
		if err != nil {
			// Max depth exceeded error.
			return
		}
	}
}

func TestGraph_Stream_NoMatchingCondition(t *testing.T) {
	g := NewGraph()
	g.AddNode("start", newStep(func(input any) (any, error) { return input, nil }))
	g.AddNode("next", newStep(func(input any) (any, error) { return "next", nil }))
	g.AddEdge(Edge{From: "start", To: "next", Condition: func(v any) bool { return v == "go" }})
	g.SetEntry("start")

	// "stop" doesn't match any condition, but start has outgoing edges so it
	// will invoke, then find no matching next — return value.
	var results []any
	for val, err := range g.Stream(context.Background(), "stop") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		results = append(results, val)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
}

func TestGraph_Stream_TerminalNode(t *testing.T) {
	g := NewGraph()
	g.AddNode("only", newStep(func(input any) (any, error) { return fmt.Sprintf("done(%v)", input), nil }))
	g.SetEntry("only")

	var results []any
	for val, err := range g.Stream(context.Background(), "x") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		results = append(results, val)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	if results[0] != "done(x)" {
		t.Fatalf("expected done(x), got %v", results[0])
	}
}

func TestGraph_Stream_StreamError(t *testing.T) {
	errStream := errors.New("stream error")
	// Create a terminal node whose stream returns an error.
	errorStep := &mockRunnable{
		invokeFn: func(_ context.Context, input any, _ ...core.Option) (any, error) {
			return input, nil
		},
		streamFn: func(_ context.Context, _ any, _ ...core.Option) iter.Seq2[any, error] {
			return func(yield func(any, error) bool) {
				yield(nil, errStream)
			}
		},
	}

	g := NewGraph()
	g.AddNode("terminal", errorStep)
	g.SetEntry("terminal")

	for _, err := range g.Stream(context.Background(), "x") {
		if err != nil {
			if !errors.Is(err, errStream) {
				t.Fatalf("expected stream error, got %v", err)
			}
			return
		}
	}
}

func TestGraph_FirstMatchWins(t *testing.T) {
	g := NewGraph()
	g.AddNode("start", newStep(func(input any) (any, error) { return input, nil }))
	g.AddNode("first", newStep(func(input any) (any, error) { return "first", nil }))
	g.AddNode("second", newStep(func(input any) (any, error) { return "second", nil }))

	// Both conditions match, first edge wins.
	g.AddEdge(Edge{From: "start", To: "first", Condition: func(any) bool { return true }})
	g.AddEdge(Edge{From: "start", To: "second", Condition: func(any) bool { return true }})
	g.SetEntry("start")

	result, err := g.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "first" {
		t.Fatalf("expected first (first match wins), got %v", result)
	}
}
