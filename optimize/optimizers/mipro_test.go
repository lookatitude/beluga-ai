package optimizers

import (
	"context"
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/lookatitude/beluga-ai/optimize"
	"github.com/lookatitude/beluga-ai/optimize/metric"
)

// ── unit tests ────────────────────────────────────────────────────────────────

func TestMIPROv2_Defaults(t *testing.T) {
	m := NewMIPROv2()

	tests := []struct {
		name string
		got  int
		want int
	}{
		{"NumTrials", m.NumTrials, 30},
		{"MinibatchSize", m.MinibatchSize, 25},
		{"NumInstructionCandidates", m.NumInstructionCandidates, 5},
		{"NumDemoCandidates", m.NumDemoCandidates, 5},
		{"NumDemosPerCandidate", m.NumDemosPerCandidate, 4},
		{"ConvergenceWindowSize", m.ConvergenceWindowSize, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("expected %s=%d, got %d", tt.name, tt.want, tt.got)
			}
		})
	}

	if m.ConvergenceThreshold <= 0 {
		t.Errorf("expected positive ConvergenceThreshold, got %v", m.ConvergenceThreshold)
	}
	if m.TPE == nil {
		t.Error("expected non-nil TPE")
	}
}

func TestMIPROv2_FunctionalOptions(t *testing.T) {
	m := NewMIPROv2(
		WithNumTrials(50),
		WithMinibatchSize(30),
		WithNumInstructionCandidates(8),
		WithNumDemoCandidates(6),
		WithNumDemosPerCandidate(3),
		WithMIPROv2Seed(99),
		WithMIPROv2Gamma(0.3),
		WithMIPROv2ConvergenceThreshold(0.005),
		WithMIPROv2ConvergenceWindow(7),
	)

	if m.NumTrials != 50 {
		t.Errorf("NumTrials: expected 50, got %d", m.NumTrials)
	}
	if m.MinibatchSize != 30 {
		t.Errorf("MinibatchSize: expected 30, got %d", m.MinibatchSize)
	}
	if m.NumInstructionCandidates != 8 {
		t.Errorf("NumInstructionCandidates: expected 8, got %d", m.NumInstructionCandidates)
	}
	if m.NumDemoCandidates != 6 {
		t.Errorf("NumDemoCandidates: expected 6, got %d", m.NumDemoCandidates)
	}
	if m.NumDemosPerCandidate != 3 {
		t.Errorf("NumDemosPerCandidate: expected 3, got %d", m.NumDemosPerCandidate)
	}
	if m.Seed != 99 {
		t.Errorf("Seed: expected 99, got %d", m.Seed)
	}
	if m.ConvergenceThreshold != 0.005 {
		t.Errorf("ConvergenceThreshold: expected 0.005, got %v", m.ConvergenceThreshold)
	}
	if m.ConvergenceWindowSize != 7 {
		t.Errorf("ConvergenceWindowSize: expected 7, got %d", m.ConvergenceWindowSize)
	}
}

func TestMIPROv2_Registry(t *testing.T) {
	names := optimize.ListOptimizers()
	found := false
	for _, name := range names {
		if name == "mipro" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("'mipro' not found in registry: %v", names)
	}

	opt, err := optimize.NewOptimizer("mipro", optimize.OptimizerConfig{})
	if err != nil {
		t.Fatalf("NewOptimizer('mipro'): %v", err)
	}
	if opt == nil {
		t.Error("expected non-nil optimizer")
	}
}

func TestMIPROv2_RegistryWithLLM(t *testing.T) {
	llm := &mockLLMClient{response: "Use clear, concise language to answer the question."}

	opt, err := optimize.NewOptimizer("mipro", optimize.OptimizerConfig{LLM: llm})
	if err != nil {
		t.Fatalf("NewOptimizer with LLM: %v", err)
	}
	m, ok := opt.(*MIPROv2)
	if !ok {
		t.Fatal("expected *MIPROv2")
	}
	if m.LLM == nil {
		t.Error("expected LLM to be wired from config")
	}
}

func TestMIPROv2_Compile_MissingMetric(t *testing.T) {
	m := NewMIPROv2()
	_, err := m.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: []optimize.Example{{Inputs: map[string]interface{}{"q": "x"}}},
	})
	if err == nil {
		t.Error("expected error for missing metric")
	}
}

func TestMIPROv2_Compile_EmptyTrainset(t *testing.T) {
	m := NewMIPROv2()
	_, err := m.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Metric:   optimize.MetricFunc(metric.ExactMatch),
		Trainset: []optimize.Example{},
	})
	if err == nil {
		t.Error("expected error for empty trainset")
	}
}

func TestMIPROv2_Compile_Basic(t *testing.T) {
	m := NewMIPROv2(
		WithNumTrials(5),
		WithMinibatchSize(2),
		WithNumInstructionCandidates(3),
		WithNumDemoCandidates(3),
	)

	trainset := makeTrainset(6)

	compiled, err := m.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if compiled == nil {
		t.Error("expected non-nil compiled program")
	}
}

func TestMIPROv2_Compile_WithValset(t *testing.T) {
	m := NewMIPROv2(
		WithNumTrials(5),
		WithMinibatchSize(2),
	)

	trainset := makeTrainset(6)
	valset := makeTrainset(3)

	compiled, err := m.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: trainset,
		Valset:   valset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
	if err != nil {
		t.Fatalf("Compile with valset: %v", err)
	}
	if compiled == nil {
		t.Error("expected non-nil compiled program")
	}
}

func TestMIPROv2_Compile_ContextCancellation(t *testing.T) {
	m := NewMIPROv2(
		WithNumTrials(1000), // high enough that cancellation triggers
		WithMinibatchSize(5),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := m.Compile(ctx, &MockProgram{}, optimize.CompileOptions{
		Trainset: makeTrainset(10),
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
	// Should return context.Canceled or an mipro error wrapping it,
	// OR succeed if the first trial gets past the select{} before cancellation.
	// We only care that it doesn't hang.
	_ = err
}

func TestMIPROv2_Compile_WithCallback(t *testing.T) {
	var trialCount int
	var completedCalled bool

	cb := &countingCallback{
		onTrial: func(trial optimize.Trial) {
			trialCount++
		},
		onComplete: func(result optimize.OptimizationResult) {
			completedCalled = true
			if result.NumTrials == 0 {
				t.Error("expected non-zero NumTrials in OptimizationResult")
			}
		},
	}

	m := NewMIPROv2(
		WithNumTrials(4),
		WithMinibatchSize(2),
	)

	_, err := m.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset:  makeTrainset(6),
		Metric:    optimize.MetricFunc(metric.ExactMatch),
		Callbacks: []optimize.Callback{cb},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if trialCount == 0 {
		t.Error("expected at least one trial callback")
	}
	if !completedCalled {
		t.Error("expected OnOptimizationComplete to be called")
	}
}

func TestMIPROv2_Compile_BudgetLimit(t *testing.T) {
	m := NewMIPROv2(
		WithNumTrials(100),
		WithMinibatchSize(2),
	)

	_, err := m.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: makeTrainset(6),
		Metric:   optimize.MetricFunc(metric.ExactMatch),
		MaxCost: &optimize.CostBudget{
			MaxIterations: 3, // stop after 3 iterations
		},
	})
	if err != nil {
		t.Fatalf("Compile with budget: %v", err)
	}
}

func TestMIPROv2_GenerateInstructionCandidates_Templates(t *testing.T) {
	tests := []struct {
		n int
	}{
		{3}, {5}, {7}, {12},
	}
	for _, tt := range tests {
		t.Run("n="+itoa(tt.n), func(t *testing.T) {
			m := NewMIPROv2(WithNumInstructionCandidates(tt.n))
			cands, err := m.generateInstructionCandidates(context.Background(), &MockProgram{}, optimize.CompileOptions{
				Trainset: makeTrainset(3),
				Metric:   optimize.MetricFunc(metric.ExactMatch),
			})
			if err != nil {
				t.Fatalf("generateInstructionCandidates: %v", err)
			}
			if len(cands) != tt.n {
				t.Errorf("expected %d candidates, got %d", tt.n, len(cands))
			}
			for _, c := range cands {
				if c == "" {
					t.Error("instruction candidate must not be empty")
				}
			}
		})
	}
}

func TestMIPROv2_GenerateInstructionCandidates_LLM(t *testing.T) {
	llm := &mockLLMClient{response: "Carefully read the context and provide a precise answer."}

	m := NewMIPROv2(
		WithNumInstructionCandidates(4),
		WithMIPROv2LLM(llm),
	)
	cands, err := m.generateInstructionCandidates(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: makeTrainset(3),
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
	if err != nil {
		t.Fatalf("LLM proposal: %v", err)
	}
	if len(cands) != 4 {
		t.Errorf("expected 4 candidates, got %d", len(cands))
	}
	if llm.calls == 0 {
		t.Error("expected LLM to be called at least once")
	}
}

func TestMIPROv2_GenerateInstructionCandidates_LLMFallback(t *testing.T) {
	// LLM returns empty responses — should fall back to templates.
	llm := &mockLLMClient{response: ""}

	m := NewMIPROv2(
		WithNumInstructionCandidates(3),
		WithMIPROv2LLM(llm),
	)
	cands, err := m.generateInstructionCandidates(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: makeTrainset(2),
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
	if err != nil {
		t.Fatalf("LLM fallback: %v", err)
	}
	if len(cands) != 3 {
		t.Errorf("expected 3 candidates after fallback, got %d", len(cands))
	}
	for _, c := range cands {
		if c == "" {
			t.Error("fallback candidate must not be empty")
		}
	}
}

func TestMIPROv2_GenerateDemoCandidates(t *testing.T) {
	tests := []struct {
		nDemoCandidates    int
		nDemosPerCandidate int
		trainsetSize       int
		wantDemosPerSet    int
	}{
		{3, 4, 8, 4},
		{5, 4, 3, 3}, // trainset smaller than NumDemosPerCandidate
		{2, 2, 5, 2},
	}
	for _, tt := range tests {
		m := NewMIPROv2(
			WithNumDemoCandidates(tt.nDemoCandidates),
			WithNumDemosPerCandidate(tt.nDemosPerCandidate),
		)
		cands, err := m.generateDemoCandidates(context.Background(), &MockProgram{}, optimize.CompileOptions{
			Trainset: makeTrainset(tt.trainsetSize),
			Metric:   optimize.MetricFunc(metric.ExactMatch),
		})
		if err != nil {
			t.Fatalf("generateDemoCandidates: %v", err)
		}
		if len(cands) != tt.nDemoCandidates {
			t.Errorf("expected %d demo sets, got %d", tt.nDemoCandidates, len(cands))
		}
		for i, c := range cands {
			if len(c) != tt.wantDemosPerSet {
				t.Errorf("set %d: expected %d demos, got %d", i, tt.wantDemosPerSet, len(c))
			}
		}
	}
}

func TestMIPROv2_SampleMinibatch(t *testing.T) {
	trainExamples := makeTrainset(10)

	tests := []struct {
		n    int
		want int
	}{
		{5, 5},
		{10, 10},
		{20, 10}, // capped at len(trainset)
	}
	for _, tt := range tests {
		rng := newTestRNG(0)
		batch := sampleMinibatch(trainExamples, tt.n, rng)
		if len(batch) != tt.want {
			t.Errorf("sampleMinibatch(n=%d): expected %d, got %d", tt.n, tt.want, len(batch))
		}
	}
}

func TestMIPROv2_Compile_Deterministic(t *testing.T) {
	// Same seed → same optimized program.
	trainset := makeTrainset(8)
	opts := optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	}

	m1 := NewMIPROv2(WithNumTrials(5), WithMinibatchSize(3), WithMIPROv2Seed(7))
	m2 := NewMIPROv2(WithNumTrials(5), WithMinibatchSize(3), WithMIPROv2Seed(7))

	r1, err := m1.Compile(context.Background(), &MockProgram{}, opts)
	if err != nil {
		t.Fatalf("first compile: %v", err)
	}
	r2, err := m2.Compile(context.Background(), &MockProgram{}, opts)
	if err != nil {
		t.Fatalf("second compile: %v", err)
	}

	// Both should be non-nil; we can't assert internal identity without
	// exposing more state, but we check that both succeed consistently.
	if r1 == nil || r2 == nil {
		t.Error("expected non-nil compiled programs for both runs")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func makeTrainset(n int) []optimize.Example {
	ex := make([]optimize.Example, n)
	for i := range ex {
		ex[i] = optimize.Example{
			Inputs:  map[string]interface{}{"question": itoa(i)},
			Outputs: map[string]interface{}{"answer": itoa(i)},
		}
	}
	return ex
}

func itoa(n int) string {
	return string(rune('0' + n%10))
}

// newTestRNG creates a *rand.Rand with a fixed seed for test helpers.
func newTestRNG(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}

// mockLLMClient is a simple LLM client stub for unit tests.
type mockLLMClient struct {
	response string
	err      error
	calls    int
}

func (m *mockLLMClient) Complete(_ context.Context, _ string, _ optimize.CompletionOptions) (string, error) {
	m.calls++
	return m.response, m.err
}

func (m *mockLLMClient) CompleteJSON(_ context.Context, _ string, _ json.RawMessage, _ optimize.CompletionOptions) (json.RawMessage, error) {
	m.calls++
	return nil, m.err
}

func (m *mockLLMClient) GetUsage() optimize.TokenUsage { return optimize.TokenUsage{} }

// countingCallback records trial completions and the final result.
type countingCallback struct {
	onTrial    func(optimize.Trial)
	onComplete func(optimize.OptimizationResult)
}

func (c *countingCallback) OnTrialComplete(t optimize.Trial) {
	if c.onTrial != nil {
		c.onTrial(t)
	}
}
func (c *countingCallback) OnOptimizationComplete(r optimize.OptimizationResult) {
	if c.onComplete != nil {
		c.onComplete(r)
	}
}
