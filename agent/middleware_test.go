package agent

import (
	"context"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/tool"
)

// wrappedAgent is a middleware-created wrapper that records calls.
type wrappedAgent struct {
	inner  Agent
	prefix string
}

func (w *wrappedAgent) ID() string            { return w.inner.ID() }
func (w *wrappedAgent) Persona() Persona      { return w.inner.Persona() }
func (w *wrappedAgent) Tools() []tool.Tool    { return w.inner.Tools() }
func (w *wrappedAgent) Children() []Agent     { return w.inner.Children() }
func (w *wrappedAgent) Invoke(ctx context.Context, input string, opts ...Option) (string, error) {
	result, err := w.inner.Invoke(ctx, w.prefix+input, opts...)
	return w.prefix + result, err
}
func (w *wrappedAgent) Stream(ctx context.Context, input string, opts ...Option) iter.Seq2[Event, error] {
	return w.inner.Stream(ctx, w.prefix+input, opts...)
}

func TestApplyMiddleware_SingleMiddleware(t *testing.T) {
	base := &mockAgent{id: "base"}

	mw := func(a Agent) Agent {
		return &wrappedAgent{inner: a, prefix: "[mw]"}
	}

	wrapped := ApplyMiddleware(base, mw)
	result, err := wrapped.Invoke(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// wrappedAgent prepends prefix to input, then prepends prefix to result.
	// inner.Invoke("hello") -> mockAgent returns "mock:[mw]hello"
	// wrappedAgent prepends "[mw]" -> "[mw]mock:[mw]hello"
	if result != "[mw]mock:[mw]hello" {
		t.Errorf("result = %q, want %q", result, "[mw]mock:[mw]hello")
	}
}

func TestApplyMiddleware_MultipleMiddleware(t *testing.T) {
	var order []string
	base := &mockAgent{id: "base"}

	mw1 := func(a Agent) Agent {
		return &orderAgent{inner: a, label: "mw1", order: &order}
	}
	mw2 := func(a Agent) Agent {
		return &orderAgent{inner: a, label: "mw2", order: &order}
	}

	wrapped := ApplyMiddleware(base, mw1, mw2)
	_, _ = wrapped.Invoke(context.Background(), "test")

	// mw1 is outermost, so it should be called first.
	if len(order) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(order))
	}
	if order[0] != "mw1" {
		t.Errorf("first call = %q, want %q", order[0], "mw1")
	}
	if order[1] != "mw2" {
		t.Errorf("second call = %q, want %q", order[1], "mw2")
	}
}

func TestApplyMiddleware_NoMiddleware(t *testing.T) {
	base := &mockAgent{id: "base"}
	result := ApplyMiddleware(base)

	if result != base {
		t.Error("expected same agent when no middleware applied")
	}
}

func TestApplyMiddleware_PreservesID(t *testing.T) {
	base := &mockAgent{id: "original-id"}

	mw := func(a Agent) Agent {
		return &wrappedAgent{inner: a, prefix: ""}
	}

	wrapped := ApplyMiddleware(base, mw)
	if wrapped.ID() != "original-id" {
		t.Errorf("ID() = %q, want %q", wrapped.ID(), "original-id")
	}
}

// orderAgent tracks call order for middleware testing.
type orderAgent struct {
	inner Agent
	label string
	order *[]string
}

func (o *orderAgent) ID() string            { return o.inner.ID() }
func (o *orderAgent) Persona() Persona      { return o.inner.Persona() }
func (o *orderAgent) Tools() []tool.Tool    { return o.inner.Tools() }
func (o *orderAgent) Children() []Agent     { return o.inner.Children() }
func (o *orderAgent) Invoke(ctx context.Context, input string, opts ...Option) (string, error) {
	*o.order = append(*o.order, o.label)
	return o.inner.Invoke(ctx, input, opts...)
}
func (o *orderAgent) Stream(ctx context.Context, input string, opts ...Option) iter.Seq2[Event, error] {
	*o.order = append(*o.order, o.label)
	return o.inner.Stream(ctx, input, opts...)
}
