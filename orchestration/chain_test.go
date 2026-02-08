package orchestration

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
)

// mockRunnable is a test double for core.Runnable.
type mockRunnable struct {
	invokeFn func(ctx context.Context, input any, opts ...core.Option) (any, error)
	streamFn func(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error]
}

func (m *mockRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	if m.invokeFn != nil {
		return m.invokeFn(ctx, input, opts...)
	}
	return input, nil
}

func (m *mockRunnable) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	if m.streamFn != nil {
		return m.streamFn(ctx, input, opts...)
	}
	return func(yield func(any, error) bool) {
		yield(input, nil)
	}
}

func newStep(transform func(any) (any, error)) *mockRunnable {
	return &mockRunnable{
		invokeFn: func(_ context.Context, input any, _ ...core.Option) (any, error) {
			return transform(input)
		},
		streamFn: func(_ context.Context, input any, _ ...core.Option) iter.Seq2[any, error] {
			return func(yield func(any, error) bool) {
				result, err := transform(input)
				yield(result, err)
			}
		},
	}
}

func TestChain_EmptyChain(t *testing.T) {
	c := Chain()
	result, err := c.Invoke(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello" {
		t.Fatalf("expected input passthrough, got %v", result)
	}
}

func TestChain_SingleStep(t *testing.T) {
	step := newStep(func(input any) (any, error) {
		return fmt.Sprintf("[%v]", input), nil
	})
	c := Chain(step)

	result, err := c.Invoke(context.Background(), "data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "[data]" {
		t.Fatalf("expected [data], got %v", result)
	}
}

func TestChain_TwoSteps(t *testing.T) {
	step1 := newStep(func(input any) (any, error) {
		return fmt.Sprintf("(%v)", input), nil
	})
	step2 := newStep(func(input any) (any, error) {
		return fmt.Sprintf("[%v]", input), nil
	})
	c := Chain(step1, step2)

	result, err := c.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "[(x)]" {
		t.Fatalf("expected [(x)], got %v", result)
	}
}

func TestChain_ThreeSteps(t *testing.T) {
	step1 := newStep(func(input any) (any, error) { return fmt.Sprintf("a(%v)", input), nil })
	step2 := newStep(func(input any) (any, error) { return fmt.Sprintf("b(%v)", input), nil })
	step3 := newStep(func(input any) (any, error) { return fmt.Sprintf("c(%v)", input), nil })
	c := Chain(step1, step2, step3)

	result, err := c.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "c(b(a(x)))"
	if result != expected {
		t.Fatalf("expected %q, got %v", expected, result)
	}
}

func TestChain_ErrorPropagation(t *testing.T) {
	errBoom := errors.New("boom")
	step1 := newStep(func(input any) (any, error) { return input, nil })
	step2 := newStep(func(_ any) (any, error) { return nil, errBoom })
	step3 := newStep(func(input any) (any, error) { return input, nil })
	c := Chain(step1, step2, step3)

	_, err := c.Invoke(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errBoom) {
		t.Fatalf("expected wrapped boom error, got %v", err)
	}
}

func TestChain_Stream_EmptyChain(t *testing.T) {
	c := Chain()
	var results []any
	for val, err := range c.Stream(context.Background(), "hello") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		results = append(results, val)
	}
	if len(results) != 1 || results[0] != "hello" {
		t.Fatalf("expected [hello], got %v", results)
	}
}

func TestChain_Stream_MultiStep(t *testing.T) {
	step1 := newStep(func(input any) (any, error) { return fmt.Sprintf("(%v)", input), nil })
	step2 := &mockRunnable{
		invokeFn: func(_ context.Context, input any, _ ...core.Option) (any, error) {
			return input, nil
		},
		streamFn: func(_ context.Context, input any, _ ...core.Option) iter.Seq2[any, error] {
			return func(yield func(any, error) bool) {
				s := fmt.Sprintf("%v", input)
				for _, ch := range s {
					if !yield(string(ch), nil) {
						return
					}
				}
			}
		},
	}
	c := Chain(step1, step2)

	var results []any
	for val, err := range c.Stream(context.Background(), "ab") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		results = append(results, val)
	}
	// step1 wraps in parens: (ab), step2 streams each char
	expected := []string{"(", "a", "b", ")"}
	if len(results) != len(expected) {
		t.Fatalf("expected %d results, got %d: %v", len(expected), len(results), results)
	}
	for i, v := range results {
		if v != expected[i] {
			t.Fatalf("result[%d]: expected %q, got %v", i, expected[i], v)
		}
	}
}

func TestChain_Stream_ErrorPropagation(t *testing.T) {
	errBoom := errors.New("boom")
	step1 := newStep(func(_ any) (any, error) { return nil, errBoom })
	step2 := newStep(func(input any) (any, error) { return input, nil })
	c := Chain(step1, step2)

	for _, err := range c.Stream(context.Background(), "x") {
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errBoom) {
			t.Fatalf("expected wrapped boom error, got %v", err)
		}
		return
	}
	t.Fatal("expected at least one stream result")
}

func TestChain_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	step := &mockRunnable{
		invokeFn: func(ctx context.Context, input any, _ ...core.Option) (any, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return input, nil
		},
	}
	c := Chain(step)

	_, err := c.Invoke(ctx, "x")
	if err == nil {
		t.Fatal("expected context error, got nil")
	}
}
