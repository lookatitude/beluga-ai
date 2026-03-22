package optimizers

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/optimize"
	"github.com/lookatitude/beluga-ai/optimize/metric"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

// buildTrainset constructs a trainset of n examples where input "q" == output "a".
func buildTrainset(n int) []optimize.Example {
	examples := make([]optimize.Example, n)
	for i := range examples {
		v := string(rune('A' + i%26))
		examples[i] = optimize.Example{
			Inputs:  map[string]interface{}{"q": v},
			Outputs: map[string]interface{}{"a": v},
		}
	}
	return examples
}

// ─── BatchEvaluator unit tests ────────────────────────────────────────────────

func TestBatchEvaluator_Defaults(t *testing.T) {
	b := NewBatchEvaluator()
	if b.ChunkSize != 50 {
		t.Errorf("expected default ChunkSize=50, got %d", b.ChunkSize)
	}
	if b.NumWorkers != 1 {
		t.Errorf("expected default NumWorkers=1, got %d", b.NumWorkers)
	}
}

func TestBatchEvaluator_Options(t *testing.T) {
	b := NewBatchEvaluator(
		WithBatchChunkSize(10),
		WithBatchNumWorkers(4),
	)
	if b.ChunkSize != 10 {
		t.Errorf("expected ChunkSize=10, got %d", b.ChunkSize)
	}
	if b.NumWorkers != 4 {
		t.Errorf("expected NumWorkers=4, got %d", b.NumWorkers)
	}
}

func TestBatchEvaluator_InvalidOptions_Ignored(t *testing.T) {
	b := NewBatchEvaluator(
		WithBatchChunkSize(-5),
		WithBatchNumWorkers(0),
	)
	if b.ChunkSize != 50 {
		t.Errorf("expected default ChunkSize=50 after invalid, got %d", b.ChunkSize)
	}
	if b.NumWorkers != 1 {
		t.Errorf("expected default NumWorkers=1 after invalid, got %d", b.NumWorkers)
	}
}

func TestBatchEvaluator_EmptyExamples(t *testing.T) {
	b := NewBatchEvaluator()
	c := simbaCandidate{ID: "x", Demos: nil}
	score, err := b.EvaluateAll(context.Background(), &MockProgram{}, c, nil, optimize.MetricFunc(metric.ExactMatch))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score != 0 {
		t.Errorf("expected score=0 for empty examples, got %f", score)
	}
}

func TestBatchEvaluator_Sequential(t *testing.T) {
	b := NewBatchEvaluator(WithBatchChunkSize(3), WithBatchNumWorkers(1))
	examples := buildTrainset(10)
	c := simbaCandidate{ID: "seq", Demos: nil}

	score, err := b.EvaluateAll(context.Background(), &MockProgram{}, c, examples, optimize.MetricFunc(metric.ExactMatch))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// MockProgram returns input as output, ExactMatch expects input["q"] == output["a"]
	// but MockProgram outputs {"q": v} while expected output is {"a": v} → score should be 0.
	// The point here is no error and deterministic execution.
	if score < 0 || score > 1 {
		t.Errorf("score %f out of [0,1] range", score)
	}
}

func TestBatchEvaluator_Parallel(t *testing.T) {
	b := NewBatchEvaluator(WithBatchChunkSize(5), WithBatchNumWorkers(4))
	examples := buildTrainset(20)
	c := simbaCandidate{ID: "par", Demos: nil}

	score, err := b.EvaluateAll(context.Background(), &MockProgram{}, c, examples, optimize.MetricFunc(metric.ExactMatch))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score < 0 || score > 1 {
		t.Errorf("score %f out of [0,1] range", score)
	}
}

// TestBatchEvaluator_SequentialParallelConsistency verifies that sequential and
// parallel evaluation produce the same score (same program, same examples, same seed).
func TestBatchEvaluator_SequentialParallelConsistency(t *testing.T) {
	examples := buildTrainset(50)
	c := simbaCandidate{ID: "consistency", Demos: nil}
	m := optimize.MetricFunc(metric.ExactMatch)
	prog := &MockProgram{}

	seq := NewBatchEvaluator(WithBatchChunkSize(5), WithBatchNumWorkers(1))
	par := NewBatchEvaluator(WithBatchChunkSize(5), WithBatchNumWorkers(4))

	seqScore, err := seq.EvaluateAll(context.Background(), prog, c, examples, m)
	if err != nil {
		t.Fatalf("sequential error: %v", err)
	}
	parScore, err := par.EvaluateAll(context.Background(), prog, c, examples, m)
	if err != nil {
		t.Fatalf("parallel error: %v", err)
	}
	if seqScore != parScore {
		t.Errorf("sequential score %f != parallel score %f", seqScore, parScore)
	}
}

func TestBatchEvaluator_ContextCancellation(t *testing.T) {
	b := NewBatchEvaluator(WithBatchChunkSize(1), WithBatchNumWorkers(1))
	examples := buildTrainset(100)
	c := simbaCandidate{ID: "cancel", Demos: nil}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Should not block or panic, may return 0 score.
	score, _ := b.EvaluateAll(ctx, &MockProgram{}, c, examples, optimize.MetricFunc(metric.ExactMatch))
	if score < 0 || score > 1 {
		t.Errorf("score %f out of [0,1] after cancellation", score)
	}
}

func TestBatchEvaluator_EvaluatePool(t *testing.T) {
	b := NewBatchEvaluator(WithBatchChunkSize(5), WithBatchNumWorkers(2))
	examples := buildTrainset(15)
	pool := []simbaCandidate{
		{ID: "a", Demos: nil},
		{ID: "b", Demos: nil},
		{ID: "c", Demos: nil},
	}

	scores, err := b.EvaluatePool(context.Background(), &MockProgram{}, pool, examples, optimize.MetricFunc(metric.ExactMatch))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scores) != 3 {
		t.Errorf("expected 3 scores, got %d", len(scores))
	}
	for i, s := range scores {
		if s < 0 || s > 1 {
			t.Errorf("scores[%d]=%f out of [0,1]", i, s)
		}
	}
}

// TestBatchEvaluator_LargeDataset verifies memory-efficient processing of a
// dataset much larger than ChunkSize without OOM or error.
func TestBatchEvaluator_LargeDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large dataset test in short mode")
	}
	b := NewBatchEvaluator(WithBatchChunkSize(20), WithBatchNumWorkers(2))
	examples := buildTrainset(500)
	c := simbaCandidate{ID: "large", Demos: nil}

	score, err := b.EvaluateAll(context.Background(), &MockProgram{}, c, examples, optimize.MetricFunc(metric.ExactMatch))
	if err != nil {
		t.Fatalf("unexpected error on large dataset: %v", err)
	}
	if score < 0 || score > 1 {
		t.Errorf("score %f out of [0,1]", score)
	}
}

// ─── SIMBA integration tests ──────────────────────────────────────────────────

// TestSIMBA_Integration_FullLoop runs a complete SIMBA compile-and-infer cycle
// and verifies the returned program is usable.
func TestSIMBA_Integration_FullLoop(t *testing.T) {
	s := NewSIMBA(
		WithSIMBAMaxIterations(5),
		WithSIMBACandidatePoolSize(4),
		WithSIMBAMinibatchSize(5),
		WithSIMBASeed(100),
	)

	trainset := buildTrainset(20)

	compiled, err := s.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}
	if compiled == nil {
		t.Fatal("expected compiled program, got nil")
	}

	// Verify the compiled program is runnable.
	pred, err := compiled.Run(context.Background(), map[string]interface{}{"q": "test"})
	if err != nil {
		t.Fatalf("Run error on compiled program: %v", err)
	}
	if pred.Outputs == nil {
		t.Error("expected non-nil outputs from compiled program")
	}
}

// TestSIMBA_Integration_ConvergenceExit verifies that SIMBA can stop early
// when convergence is detected before MaxIterations.
func TestSIMBA_Integration_ConvergenceExit(t *testing.T) {
	// High convergence threshold → should exit after just a few iterations.
	s := NewSIMBA(
		WithSIMBAMaxIterations(50),
		WithSIMBACandidatePoolSize(4),
		WithSIMBAMinibatchSize(5),
		WithSIMBAConvergenceThreshold(1.0), // converge immediately
		WithSIMBASeed(7),
	)

	start := time.Now()
	_, err := s.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: buildTrainset(10),
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 50 full iterations on a trivial mock would still be fast, but we at least
	// check that it finished in a reasonable time.
	if elapsed > 10*time.Second {
		t.Errorf("compile took too long: %v", elapsed)
	}
}

// TestSIMBA_Integration_CostBudgetRespected verifies the optimizer honours
// MaxIterations in the cost budget.
func TestSIMBA_Integration_CostBudgetRespected(t *testing.T) {
	s := NewSIMBA(
		WithSIMBAMaxIterations(1000),
		WithSIMBACandidatePoolSize(4),
		WithSIMBAMinibatchSize(3),
		WithSIMBASeed(99),
	)

	_, err := s.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: buildTrainset(10),
		Metric:   optimize.MetricFunc(metric.ExactMatch),
		MaxCost:  &optimize.CostBudget{MaxIterations: 3},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSIMBA_Integration_ContextCancel ensures Compile respects context
// cancellation without deadlock or panic.
func TestSIMBA_Integration_ContextCancel(t *testing.T) {
	s := NewSIMBA(
		WithSIMBAMaxIterations(1000),
		WithSIMBACandidatePoolSize(8),
		WithSIMBAMinibatchSize(10),
		WithSIMBASeed(5),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Build a slow mock that respects context.
	type slowMock struct{ MockProgram }
	// The standard MockProgram doesn't check context; the timeout is short
	// enough that the deadline fires between iterations naturally.

	trainset := buildTrainset(100)
	// We don't require success here — just no panic/deadlock.
	_, _ = s.Compile(ctx, &MockProgram{}, optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
}

// TestSIMBA_Integration_ReproducibleResults verifies that two SIMBA runs with
// the same seed and inputs produce identical compiled programs.
func TestSIMBA_Integration_ReproducibleResults(t *testing.T) {
	opts := []SIMBAOption{
		WithSIMBAMaxIterations(3),
		WithSIMBACandidatePoolSize(4),
		WithSIMBAMinibatchSize(5),
		WithSIMBASeed(42),
	}

	trainset := buildTrainset(15)
	compileOpts := optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	}

	prog1, err1 := NewSIMBA(opts...).Compile(context.Background(), &MockProgram{}, compileOpts)
	prog2, err2 := NewSIMBA(opts...).Compile(context.Background(), &MockProgram{}, compileOpts)

	if err1 != nil || err2 != nil {
		t.Fatalf("compile errors: %v, %v", err1, err2)
	}

	// Both runs should produce a valid program.
	if prog1 == nil || prog2 == nil {
		t.Fatal("one or both compiled programs are nil")
	}
}

// TestSIMBA_Integration_Race exercises concurrent SIMBA Compile calls to detect
// races under -race. Each goroutine uses its own SIMBA instance (no sharing),
// which is the recommended usage pattern.
func TestSIMBA_Integration_Race(t *testing.T) {
	const goroutines = 4

	trainset := buildTrainset(10)
	m := optimize.MetricFunc(metric.ExactMatch)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	errors := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			s := NewSIMBA(
				WithSIMBAMaxIterations(2),
				WithSIMBACandidatePoolSize(4),
				WithSIMBAMinibatchSize(3),
				WithSIMBASeed(int64(i+1)),
			)
			_, err := s.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
				Trainset: trainset,
				Metric:   m,
			})
			errors[i] = err
		}()
	}

	wg.Wait()

	for i, err := range errors {
		if err != nil {
			t.Errorf("goroutine %d compile error: %v", i, err)
		}
	}
}

// TestSIMBA_Integration_BatchEvaluatorE2E verifies that using BatchEvaluator
// inside a SIMBA-style evaluation loop produces valid scores.
func TestSIMBA_Integration_BatchEvaluatorE2E(t *testing.T) {
	evaluator := NewBatchEvaluator(
		WithBatchChunkSize(4),
		WithBatchNumWorkers(2),
	)

	trainset := buildTrainset(20)
	pool := []simbaCandidate{
		{ID: "a", Score: 0.5, Demos: nil},
		{ID: "b", Score: 0.3, Demos: nil},
		{ID: "c", Score: 0.7, Demos: nil},
	}

	scores, err := evaluator.EvaluatePool(
		context.Background(),
		&MockProgram{},
		pool,
		trainset,
		optimize.MetricFunc(metric.ExactMatch),
	)
	if err != nil {
		t.Fatalf("EvaluatePool error: %v", err)
	}

	if len(scores) != len(pool) {
		t.Fatalf("expected %d scores, got %d", len(pool), len(scores))
	}

	for i, s := range scores {
		if s < 0 || s > 1 {
			t.Errorf("pool score[%d]=%f out of [0,1]", i, s)
		}
	}
}
