package orchestration

import (
	"context"
	"fmt"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
)

func TestApplyMiddleware_Order(t *testing.T) {
	// Middleware wraps outside-in: first middleware in list is outermost.
	var order []string

	mw1 := func(next core.Runnable) core.Runnable {
		return &mockRunnable{
			invokeFn: func(ctx context.Context, input any, opts ...core.Option) (any, error) {
				order = append(order, "mw1-before")
				result, err := next.Invoke(ctx, input, opts...)
				order = append(order, "mw1-after")
				return result, err
			},
		}
	}

	mw2 := func(next core.Runnable) core.Runnable {
		return &mockRunnable{
			invokeFn: func(ctx context.Context, input any, opts ...core.Option) (any, error) {
				order = append(order, "mw2-before")
				result, err := next.Invoke(ctx, input, opts...)
				order = append(order, "mw2-after")
				return result, err
			},
		}
	}

	base := newStep(func(input any) (any, error) {
		order = append(order, "base")
		return input, nil
	})

	wrapped := ApplyMiddleware(base, mw1, mw2)
	_, _ = wrapped.Invoke(context.Background(), "x")

	expected := []string{"mw1-before", "mw2-before", "base", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, order)
	}
	for i, v := range order {
		if v != expected[i] {
			t.Fatalf("order[%d]: expected %q, got %q", i, expected[i], v)
		}
	}
}

func TestApplyMiddleware_NoMiddleware(t *testing.T) {
	base := newStep(func(input any) (any, error) { return fmt.Sprintf("(%v)", input), nil })
	wrapped := ApplyMiddleware(base)

	result, err := wrapped.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "(x)" {
		t.Fatalf("expected (x), got %v", result)
	}
}

func TestApplyMiddleware_Transform(t *testing.T) {
	// A middleware that transforms input.
	prefixer := func(next core.Runnable) core.Runnable {
		return &mockRunnable{
			invokeFn: func(ctx context.Context, input any, opts ...core.Option) (any, error) {
				return next.Invoke(ctx, fmt.Sprintf("prefixed:%v", input), opts...)
			},
			streamFn: func(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
				return next.Stream(ctx, fmt.Sprintf("prefixed:%v", input), opts...)
			},
		}
	}

	base := newStep(func(input any) (any, error) { return input, nil })
	wrapped := ApplyMiddleware(base, prefixer)

	result, err := wrapped.Invoke(context.Background(), "data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "prefixed:data" {
		t.Fatalf("expected prefixed:data, got %v", result)
	}
}

func TestApplyMiddleware_SingleMiddleware(t *testing.T) {
	mw := func(next core.Runnable) core.Runnable {
		return &mockRunnable{
			invokeFn: func(ctx context.Context, input any, opts ...core.Option) (any, error) {
				result, err := next.Invoke(ctx, input, opts...)
				if err != nil {
					return nil, err
				}
				return fmt.Sprintf("[%v]", result), nil
			},
		}
	}

	base := newStep(func(input any) (any, error) { return input, nil })
	wrapped := ApplyMiddleware(base, mw)

	result, err := wrapped.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "[x]" {
		t.Fatalf("expected [x], got %v", result)
	}
}
