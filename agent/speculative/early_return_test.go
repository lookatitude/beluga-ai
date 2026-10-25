package speculative

import (
	"context"
	"testing"
	"time"
)

// TestSpeculativeExecutor_DefaultWaitsForGroundTruth documents the default
// (FWC-18): a confident prediction is only returned after validation, so
// Invoke is not faster than the ground truth.
func TestSpeculativeExecutor_DefaultWaitsForGroundTruth(t *testing.T) {
	gt := &mockAgent{id: "gt", delay: 100 * time.Millisecond, invokeFn: func(context.Context, string) (string, error) { return "4", nil }}
	pred := &mockPredictor{prediction: "4", confidence: 0.9}
	e := NewSpeculativeExecutor(gt, WithPredictor(pred))
	start := time.Now()
	out, err := e.Invoke(context.Background(), "2+2")
	if err != nil || out != "4" {
		t.Fatalf("Invoke = %q, %v", out, err)
	}
	if time.Since(start) < 100*time.Millisecond {
		t.Error("default mode returned before the ground truth finished")
	}
}

// TestSpeculativeExecutor_EarlyReturn is a regression test for FWC-18: there
// was no way to get the documented speedup; WithEarlyReturn returns the
// confident prediction before the ground truth and validates afterwards.
func TestSpeculativeExecutor_EarlyReturn(t *testing.T) {
	gt := &mockAgent{id: "gt", delay: 200 * time.Millisecond, invokeFn: func(context.Context, string) (string, error) { return "5", nil }}
	pred := &mockPredictor{prediction: "4", confidence: 0.9}
	missed := make(chan Result, 1)
	e := NewSpeculativeExecutor(gt, WithPredictor(pred), WithEarlyReturn(), WithHooks(Hooks{
		OnMisprediction: func(_ context.Context, r Result) { missed <- r },
	}))

	ctx, cancel := context.WithCancel(context.Background())
	start := time.Now()
	out, err := e.Invoke(ctx, "2+2")
	elapsed := time.Since(start)
	cancel() // the background validation must survive caller cancellation
	if err != nil || out != "4" {
		t.Fatalf("Invoke = %q, %v", out, err)
	}
	if elapsed >= 150*time.Millisecond {
		t.Errorf("early return took %v, want well under the 200ms ground truth", elapsed)
	}
	select {
	case r := <-missed:
		if r.GroundTruth != "5" || r.Prediction != "4" {
			t.Errorf("misprediction = %+v", r)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("background validation never reported the misprediction")
	}
}

func TestSpeculativeExecutor_EarlyReturnHitRecordsSpeedup(t *testing.T) {
	gt := &mockAgent{id: "gt", delay: 50 * time.Millisecond, invokeFn: func(context.Context, string) (string, error) { return "4", nil }}
	validated := make(chan Result, 1)
	e := NewSpeculativeExecutor(gt, WithPredictor(&mockPredictor{prediction: "4", confidence: 0.9}), WithEarlyReturn(),
		WithHooks(Hooks{OnValidation: func(_ context.Context, r Result) { validated <- r }}))
	if out, err := e.Invoke(context.Background(), "q"); err != nil || out != "4" {
		t.Fatalf("Invoke = %q, %v", out, err)
	}
	select {
	case r := <-validated:
		if r.Speedup <= 0 {
			t.Errorf("Speedup = %v, want > 0", r.Speedup)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no validation")
	}
	if e.Metrics().TotalSpeedup() <= 0 {
		t.Error("TotalSpeedup not recorded")
	}

	// Ground-truth error in the background is swallowed (no hook, no panic).
	failing := &mockAgent{id: "gt", invokeFn: func(context.Context, string) (string, error) { return "", context.DeadlineExceeded }}
	e2 := NewSpeculativeExecutor(failing, WithPredictor(&mockPredictor{prediction: "x", confidence: 1}), WithEarlyReturn())
	if out, _ := e2.Invoke(context.Background(), "q"); out != "x" {
		t.Errorf("out = %q", out)
	}
}
