package server

import (
	"context"
	"net/http"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
)

// wrappingAdapter wraps a ServerAdapter and records the wrap order.
type wrappingAdapter struct {
	inner ServerAdapter
	name  string
	order *[]string
}

func (w *wrappingAdapter) RegisterAgent(path string, a agent.Agent) error {
	*w.order = append(*w.order, "RegisterAgent:"+w.name)
	return w.inner.RegisterAgent(path, a)
}

func (w *wrappingAdapter) RegisterHandler(path string, handler http.Handler) error {
	*w.order = append(*w.order, "RegisterHandler:"+w.name)
	return w.inner.RegisterHandler(path, handler)
}

func (w *wrappingAdapter) Serve(ctx context.Context, addr string) error {
	*w.order = append(*w.order, "Serve:"+w.name)
	return nil
}

func (w *wrappingAdapter) Shutdown(ctx context.Context) error {
	*w.order = append(*w.order, "Shutdown:"+w.name)
	return nil
}

func TestApplyMiddleware(t *testing.T) {
	t.Run("right-to-left application, first middleware is outermost", func(t *testing.T) {
		var order []string
		mw1 := func(s ServerAdapter) ServerAdapter {
			return &wrappingAdapter{inner: s, name: "mw1", order: &order}
		}
		mw2 := func(s ServerAdapter) ServerAdapter {
			return &wrappingAdapter{inner: s, name: "mw2", order: &order}
		}

		base := NewStdlibAdapter(Config{})
		wrapped := ApplyMiddleware(base, mw1, mw2)

		a := &mockAgent{id: "test", result: "ok"}
		_ = wrapped.RegisterAgent("/test", a)

		// mw1 is outermost (applied last), so it should record first.
		if len(order) < 1 || order[0] != "RegisterAgent:mw1" {
			t.Errorf("expected mw1 to execute first, got %v", order)
		}
		if len(order) < 2 || order[1] != "RegisterAgent:mw2" {
			t.Errorf("expected mw2 to execute second, got %v", order)
		}
	})

	t.Run("no middlewares returns original", func(t *testing.T) {
		base := NewStdlibAdapter(Config{})
		result := ApplyMiddleware(base)
		if result != base {
			t.Error("expected same adapter when no middlewares applied")
		}
	})

	t.Run("single middleware wraps correctly", func(t *testing.T) {
		var order []string
		mw := func(s ServerAdapter) ServerAdapter {
			return &wrappingAdapter{inner: s, name: "single", order: &order}
		}

		base := NewStdlibAdapter(Config{})
		wrapped := ApplyMiddleware(base, mw)

		_ = wrapped.Shutdown(context.Background())

		if len(order) != 1 || order[0] != "Shutdown:single" {
			t.Errorf("expected [Shutdown:single], got %v", order)
		}
	})
}
