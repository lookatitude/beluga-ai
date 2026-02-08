package orchestration

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/core"
)

func TestScatterGather_FanOutFanIn(t *testing.T) {
	w1 := newStep(func(input any) (any, error) { return fmt.Sprintf("w1(%v)", input), nil })
	w2 := newStep(func(input any) (any, error) { return fmt.Sprintf("w2(%v)", input), nil })
	w3 := newStep(func(input any) (any, error) { return fmt.Sprintf("w3(%v)", input), nil })

	aggregator := func(results []any) (any, error) {
		s := ""
		for i, r := range results {
			if i > 0 {
				s += "+"
			}
			s += fmt.Sprintf("%v", r)
		}
		return s, nil
	}

	sg := NewScatterGather(aggregator, w1, w2, w3)
	result, err := sg.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "w1(x)+w2(x)+w3(x)"
	if result != expected {
		t.Fatalf("expected %q, got %v", expected, result)
	}
}

func TestScatterGather_EmptyWorkers(t *testing.T) {
	aggregator := func(results []any) (any, error) {
		return len(results), nil
	}

	sg := NewScatterGather(aggregator)
	result, err := sg.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 0 {
		t.Fatalf("expected 0 results for nil input, got %v", result)
	}
}

func TestScatterGather_WorkerError(t *testing.T) {
	errWorker := errors.New("worker failed")
	w1 := newStep(func(_ any) (any, error) { return "ok", nil })
	w2 := newStep(func(_ any) (any, error) { return nil, errWorker })

	aggregator := func(results []any) (any, error) { return results, nil }

	sg := NewScatterGather(aggregator, w1, w2)
	_, err := sg.Invoke(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errWorker) {
		t.Fatalf("expected worker error, got %v", err)
	}
}

func TestScatterGather_AggregatorError(t *testing.T) {
	errAgg := errors.New("aggregation failed")
	w1 := newStep(func(_ any) (any, error) { return "ok", nil })

	aggregator := func(_ []any) (any, error) { return nil, errAgg }

	sg := NewScatterGather(aggregator, w1)
	_, err := sg.Invoke(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errAgg) {
		t.Fatalf("expected aggregator error, got %v", err)
	}
}

func TestScatterGather_Timeout(t *testing.T) {
	slow := &mockRunnable{
		invokeFn: func(ctx context.Context, _ any, _ ...core.Option) (any, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(5 * time.Second):
				return "slow-result", nil
			}
		},
	}
	fast := newStep(func(_ any) (any, error) { return "fast", nil })

	aggregator := func(results []any) (any, error) { return results, nil }

	sg := NewScatterGather(aggregator, fast, slow).WithTimeout(50 * time.Millisecond)
	_, err := sg.Invoke(context.Background(), "x")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestScatterGather_Stream(t *testing.T) {
	w1 := newStep(func(_ any) (any, error) { return "a", nil })
	w2 := newStep(func(_ any) (any, error) { return "b", nil })

	aggregator := func(results []any) (any, error) {
		return fmt.Sprintf("%v+%v", results[0], results[1]), nil
	}

	sg := NewScatterGather(aggregator, w1, w2)
	var results []any
	for val, err := range sg.Stream(context.Background(), "x") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		results = append(results, val)
	}
	if len(results) != 1 || results[0] != "a+b" {
		t.Fatalf("expected [a+b], got %v", results)
	}
}

func TestScatterGather_OrderPreserved(t *testing.T) {
	// Workers should appear in order in results.
	var workers []core.Runnable
	for i := 0; i < 5; i++ {
		idx := i
		workers = append(workers, newStep(func(_ any) (any, error) { return idx, nil }))
	}

	aggregator := func(results []any) (any, error) { return results, nil }

	sg := NewScatterGather(aggregator, workers...)
	result, err := sg.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	results := result.([]any)
	for i, r := range results {
		if r != i {
			t.Fatalf("expected result[%d]=%d, got %v", i, i, r)
		}
	}
}
