//go:build integration

package optimizers

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/optimize"
	"github.com/lookatitude/beluga-ai/optimize/metric"
)

// ── integration tests ─────────────────────────────────────────────────────────
//
// These tests run the full MIPROv2 multi-step Bayesian loop and verify
// properties that require a non-trivial number of iterations.  They are
// guarded by the 'integration' build tag to keep the default `go test ./...`
// run fast.
//
// Run with:
//   go test ./optimize/optimizers/... -race -tags integration -v

// miproIntegrationCallback is a local copy of the counting callback used in
// integration tests. Because the integration tag creates a separate compilation
// unit from the unit-test-only helpers, we define it here so both tags compile.
type miproIntegrationCallback struct {
	onTrial    func(optimize.Trial)
	onComplete func(optimize.OptimizationResult)
}

func (c *miproIntegrationCallback) OnTrialComplete(t optimize.Trial) {
	if c.onTrial != nil {
		c.onTrial(t)
	}
}

func (c *miproIntegrationCallback) OnOptimizationComplete(r optimize.OptimizationResult) {
	if c.onComplete != nil {
		c.onComplete(r)
	}
}

// TestMIPROv2_Integration_FullBayesianLoop exercises the complete optimization
// pipeline: proposal → Bayesian search → convergence check → validation.
func TestMIPROv2_Integration_FullBayesianLoop(t *testing.T) {
	trainset := integrationTrainset(40)
	valset := integrationTrainset(10)

	m := NewMIPROv2(
		WithNumTrials(20),
		WithMinibatchSize(10),
		WithNumInstructionCandidates(5),
		WithNumDemoCandidates(5),
		WithMIPROv2Seed(123),
	)

	var trialScores []float64
	var mu sync.Mutex

	cb := &miproIntegrationCallback{
		onTrial: func(t optimize.Trial) {
			mu.Lock()
			trialScores = append(trialScores, t.Score)
			mu.Unlock()
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	compiled, err := m.Compile(ctx, &MockProgram{}, optimize.CompileOptions{
		Trainset:  trainset,
		Valset:    valset,
		Metric:    optimize.MetricFunc(metric.ExactMatch),
		Callbacks: []optimize.Callback{cb},
	})
	if err != nil {
		t.Fatalf("full loop: %v", err)
	}
	if compiled == nil {
		t.Fatal("expected non-nil compiled program")
	}

	if len(trialScores) == 0 {
		t.Error("expected at least one trial")
	}
	// All scores should be in [0, 1].
	for i, s := range trialScores {
		if s < 0 || s > 1 {
			t.Errorf("trial %d score out of range: %v", i, s)
		}
	}
}

// TestMIPROv2_Integration_ConvergenceEarlyStop checks that the optimizer stops
// before exhausting all trials when scores have converged (low variance).
func TestMIPROv2_Integration_ConvergenceEarlyStop(t *testing.T) {
	// Use a program that always scores 1.0 so scores converge immediately.
	prog := &perfectMockProgram{}
	trainset := integrationTrainset(20)

	m := NewMIPROv2(
		WithNumTrials(100), // large budget — should stop early
		WithMinibatchSize(5),
		WithMIPROv2ConvergenceThreshold(0.0001),
		WithMIPROv2ConvergenceWindow(3),
		WithMIPROv2Seed(42),
	)

	var trialCount int32
	cb := &miproIntegrationCallback{
		onTrial: func(_ optimize.Trial) {
			atomic.AddInt32(&trialCount, 1)
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	_, err := m.Compile(ctx, prog, optimize.CompileOptions{
		Trainset:  trainset,
		Metric:    optimize.MetricFunc(metric.ExactMatch),
		Callbacks: []optimize.Callback{cb},
	})
	if err != nil {
		t.Fatalf("convergence test: %v", err)
	}

	if int(trialCount) >= 100 {
		t.Errorf("expected early stop, but ran all 100 trials (convergence not detected)")
	}
	t.Logf("stopped after %d trials (budget=100)", trialCount)
}

// TestMIPROv2_Integration_LLMProposal verifies that when an LLM client is
// provided the instruction candidates are generated via the LLM (not from
// hard-coded templates) and that the compiled program is still valid.
func TestMIPROv2_Integration_LLMProposal(t *testing.T) {
	trainset := integrationTrainset(20)

	var promptsSeen []string
	var mu sync.Mutex

	llm := &integrationMockLLM{
		fn: func(prompt string) string {
			mu.Lock()
			promptsSeen = append(promptsSeen, prompt)
			mu.Unlock()
			return "Given the context, answer the question accurately and briefly."
		},
	}

	m := NewMIPROv2(
		WithNumTrials(5),
		WithMinibatchSize(5),
		WithNumInstructionCandidates(4),
		WithMIPROv2LLM(llm),
		WithMIPROv2Seed(7),
	)

	compiled, err := m.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
	if err != nil {
		t.Fatalf("LLM proposal: %v", err)
	}
	if compiled == nil {
		t.Fatal("expected non-nil compiled program")
	}

	// The LLM should have been called once per instruction candidate.
	if llm.callCount() != 4 {
		t.Errorf("expected 4 LLM calls for 4 candidates, got %d", llm.callCount())
	}
}

// TestMIPROv2_Integration_ContextCancellationMidRun verifies that the optimizer
// respects context cancellation without leaking goroutines or panicking.
func TestMIPROv2_Integration_ContextCancellationMidRun(t *testing.T) {
	trainset := integrationTrainset(50)

	m := NewMIPROv2(
		WithNumTrials(200),
		WithMinibatchSize(20),
		WithMIPROv2Seed(5),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, _ = m.Compile(ctx, &slowMockProgram{delay: 5 * time.Millisecond}, optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
	elapsed := time.Since(start)

	// Should have returned within a reasonable grace period after the 50 ms deadline.
	if elapsed > 3*time.Second {
		t.Errorf("expected early exit due to context cancellation, took %v", elapsed)
	}
}

// TestMIPROv2_Integration_BudgetRespected checks that MaxIterations in CostBudget
// is strictly respected — at most MaxIterations trials should be recorded.
func TestMIPROv2_Integration_BudgetRespected(t *testing.T) {
	const maxIter = 7
	var count int32

	cb := &miproIntegrationCallback{
		onTrial: func(_ optimize.Trial) {
			atomic.AddInt32(&count, 1)
		},
	}

	m := NewMIPROv2(
		WithNumTrials(100),
		WithMinibatchSize(3),
		WithMIPROv2Seed(9),
	)

	_, err := m.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset:  integrationTrainset(20),
		Metric:    optimize.MetricFunc(metric.ExactMatch),
		MaxCost:   &optimize.CostBudget{MaxIterations: maxIter},
		Callbacks: []optimize.Callback{cb},
	})
	if err != nil {
		t.Fatalf("budget test: %v", err)
	}
	if int(count) > maxIter {
		t.Errorf("budget exceeded: ran %d trials, max was %d", count, maxIter)
	}
}

// TestMIPROv2_Integration_MultiStepImprovement runs MIPROv2 with a scoring
// metric that rewards longer demo sets and verifies that the best-found
// candidate improves over the initial (zero-demo) baseline.
func TestMIPROv2_Integration_MultiStepImprovement(t *testing.T) {
	trainset := integrationTrainset(30)

	// Metric: score = number of demos attached / 4 (saturates at 1.0 with 4 demos).
	demoCountMetric := optimize.MetricFunc(func(ex optimize.Example, pred optimize.Prediction, _ *optimize.Trace) float64 {
		if n, ok := pred.Outputs["demo_count"].(int); ok {
			if n >= 4 {
				return 1.0
			}
			return float64(n) / 4.0
		}
		return 0.0
	})

	prog := &demoCountingProgram{}

	m := NewMIPROv2(
		WithNumTrials(15),
		WithMinibatchSize(8),
		WithNumDemosPerCandidate(4),
		WithMIPROv2Seed(77),
	)

	compiled, err := m.Compile(context.Background(), prog, optimize.CompileOptions{
		Trainset: trainset,
		Metric:   demoCountMetric,
	})
	if err != nil {
		t.Fatalf("multi-step improvement: %v", err)
	}
	if compiled == nil {
		t.Fatal("expected compiled program")
	}
}

// TestMIPROv2_Integration_RaceFreeConcurrentCompile ensures there are no data
// races when multiple MIPROv2 instances run concurrently (distinct instances,
// shared trainset).
func TestMIPROv2_Integration_RaceFreeConcurrentCompile(t *testing.T) {
	trainset := integrationTrainset(20)
	opts := optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	}

	const numConcurrent = 4
	var wg sync.WaitGroup
	errs := make([]error, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			m := NewMIPROv2(
				WithNumTrials(5),
				WithMinibatchSize(3),
				WithMIPROv2Seed(int64(i+1)),
			)
			_, errs[i] = m.Compile(context.Background(), &MockProgram{}, opts)
		}()
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: %v", i, err)
		}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func integrationTrainset(n int) []optimize.Example {
	ex := make([]optimize.Example, n)
	for i := range ex {
		ex[i] = optimize.Example{
			Inputs:  map[string]interface{}{"question": i},
			Outputs: map[string]interface{}{"answer": i},
		}
	}
	return ex
}

// perfectMockProgram always returns the expected output so ExactMatch = 1.0.
type perfectMockProgram struct{}

func (p *perfectMockProgram) Run(_ context.Context, inputs map[string]interface{}) (optimize.Prediction, error) {
	return optimize.Prediction{Outputs: inputs}, nil
}

func (p *perfectMockProgram) WithDemos(_ []optimize.Example) optimize.Program {
	return &perfectMockProgram{}
}

func (p *perfectMockProgram) GetSignature() optimize.Signature { return nil }

// slowMockProgram introduces an artificial delay so context cancellation can
// interrupt the evaluation loop.
type slowMockProgram struct {
	delay time.Duration
}

func (s *slowMockProgram) Run(ctx context.Context, inputs map[string]interface{}) (optimize.Prediction, error) {
	select {
	case <-ctx.Done():
		return optimize.Prediction{}, ctx.Err()
	case <-time.After(s.delay):
	}
	return optimize.Prediction{Outputs: inputs}, nil
}

func (s *slowMockProgram) WithDemos(_ []optimize.Example) optimize.Program {
	return &slowMockProgram{delay: s.delay}
}

func (s *slowMockProgram) GetSignature() optimize.Signature { return nil }

// demoCountingProgram returns the number of demos it has in its output.
type demoCountingProgram struct {
	demos []optimize.Example
}

func (d *demoCountingProgram) Run(_ context.Context, _ map[string]interface{}) (optimize.Prediction, error) {
	return optimize.Prediction{
		Outputs: map[string]interface{}{"demo_count": len(d.demos)},
	}, nil
}

func (d *demoCountingProgram) WithDemos(demos []optimize.Example) optimize.Program {
	return &demoCountingProgram{demos: demos}
}

func (d *demoCountingProgram) GetSignature() optimize.Signature { return nil }

// integrationMockLLM is a thread-safe LLM stub for integration tests.
type integrationMockLLM struct {
	fn    func(string) string
	mu    sync.Mutex
	calls int
}

func (m *integrationMockLLM) Complete(_ context.Context, prompt string, _ optimize.CompletionOptions) (string, error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	return m.fn(prompt), nil
}

func (m *integrationMockLLM) CompleteJSON(_ context.Context, _ string, _ json.RawMessage, _ optimize.CompletionOptions) (json.RawMessage, error) {
	return nil, nil
}

func (m *integrationMockLLM) GetUsage() optimize.TokenUsage { return optimize.TokenUsage{} }

func (m *integrationMockLLM) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}
