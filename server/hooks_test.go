package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestComposeHooks_BeforeRequest(t *testing.T) {
	t.Run("all called in order", func(t *testing.T) {
		var order []int
		h1 := Hooks{BeforeRequest: func(_ context.Context, _ *http.Request) error {
			order = append(order, 1)
			return nil
		}}
		h2 := Hooks{BeforeRequest: func(_ context.Context, _ *http.Request) error {
			order = append(order, 2)
			return nil
		}}

		composed := ComposeHooks(h1, h2)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if err := composed.BeforeRequest(context.Background(), r); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(order) != 2 || order[0] != 1 || order[1] != 2 {
			t.Errorf("expected order [1,2], got %v", order)
		}
	})

	t.Run("first error short-circuits", func(t *testing.T) {
		called := false
		h1 := Hooks{BeforeRequest: func(_ context.Context, _ *http.Request) error {
			return errors.New("denied")
		}}
		h2 := Hooks{BeforeRequest: func(_ context.Context, _ *http.Request) error {
			called = true
			return nil
		}}

		composed := ComposeHooks(h1, h2)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if err := composed.BeforeRequest(context.Background(), r); err == nil {
			t.Fatal("expected error")
		}
		if called {
			t.Error("h2 should not have been called")
		}
	})

	t.Run("nil hooks are skipped", func(t *testing.T) {
		h1 := Hooks{} // BeforeRequest is nil
		h2 := Hooks{BeforeRequest: func(_ context.Context, _ *http.Request) error {
			return nil
		}}

		composed := ComposeHooks(h1, h2)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if err := composed.BeforeRequest(context.Background(), r); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestComposeHooks_AfterRequest(t *testing.T) {
	t.Run("all called in order", func(t *testing.T) {
		var codes []int
		h1 := Hooks{AfterRequest: func(_ context.Context, _ *http.Request, code int) {
			codes = append(codes, code)
		}}
		h2 := Hooks{AfterRequest: func(_ context.Context, _ *http.Request, code int) {
			codes = append(codes, code+1000)
		}}

		composed := ComposeHooks(h1, h2)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		composed.AfterRequest(context.Background(), r, 200)

		if len(codes) != 2 || codes[0] != 200 || codes[1] != 1200 {
			t.Errorf("expected [200, 1200], got %v", codes)
		}
	})

	t.Run("nil hooks are skipped", func(t *testing.T) {
		h1 := Hooks{} // AfterRequest is nil
		called := false
		h2 := Hooks{AfterRequest: func(_ context.Context, _ *http.Request, _ int) {
			called = true
		}}

		composed := ComposeHooks(h1, h2)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		composed.AfterRequest(context.Background(), r, 200)

		if !called {
			t.Error("h2 should have been called")
		}
	})
}

func TestComposeHooks_OnError(t *testing.T) {
	t.Run("non-nil return short-circuits", func(t *testing.T) {
		replacement := errors.New("replaced")
		called := false
		h1 := Hooks{OnError: func(_ context.Context, _ error) error {
			return replacement
		}}
		h2 := Hooks{OnError: func(_ context.Context, _ error) error {
			called = true
			return nil
		}}

		composed := ComposeHooks(h1, h2)
		err := composed.OnError(context.Background(), errors.New("original"))
		if !errors.Is(err, replacement) {
			t.Errorf("expected replacement error, got %v", err)
		}
		if called {
			t.Error("h2 should not have been called")
		}
	})

	t.Run("nil return passes to next hook", func(t *testing.T) {
		original := errors.New("original")
		h1 := Hooks{OnError: func(_ context.Context, _ error) error {
			return nil // suppresses, passes through
		}}
		h2 := Hooks{OnError: func(_ context.Context, err error) error {
			return nil // also suppresses
		}}

		composed := ComposeHooks(h1, h2)
		err := composed.OnError(context.Background(), original)
		if !errors.Is(err, original) {
			t.Errorf("expected original error passthrough, got %v", err)
		}
	})

	t.Run("nil hooks are skipped", func(t *testing.T) {
		original := errors.New("original")
		h1 := Hooks{} // OnError is nil
		composed := ComposeHooks(h1)
		err := composed.OnError(context.Background(), original)
		if !errors.Is(err, original) {
			t.Errorf("expected original error, got %v", err)
		}
	})
}
