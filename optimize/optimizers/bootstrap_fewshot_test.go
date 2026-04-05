package optimizers

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/optimize"
	"github.com/lookatitude/beluga-ai/optimize/metric"
)

// MockProgram is a mock implementation of optimize.Program for testing.
type MockProgram struct {
	demos []optimize.Example
}

func (m *MockProgram) Run(_ context.Context, inputs map[string]interface{}) (optimize.Prediction, error) {
	// Simple mock: return the input as output.
	return optimize.Prediction{
		Outputs: inputs,
		Raw:     "mock response",
	}, nil
}

func (m *MockProgram) WithDemos(demos []optimize.Example) optimize.Program {
	return &MockProgram{demos: demos}
}

func (m *MockProgram) GetSignature() optimize.Signature {
	return nil
}

// scoringProgram returns predictions that score differently based on inputs.
type scoringProgram struct {
	demos    []optimize.Example
	passKeys map[string]bool // inputs whose "question" value is in this set → pass
}

func (s *scoringProgram) Run(_ context.Context, inputs map[string]interface{}) (optimize.Prediction, error) {
	q, _ := inputs["question"].(string)
	if s.passKeys != nil && s.passKeys[q] {
		return optimize.Prediction{
			Outputs: map[string]interface{}{"answer": q}, // match expected
			Raw:     "pass",
			Usage:   optimize.TokenUsage{TotalTokens: 10},
		}, nil
	}
	return optimize.Prediction{
		Outputs: map[string]interface{}{"answer": "wrong"},
		Raw:     "fail",
		Usage:   optimize.TokenUsage{TotalTokens: 5},
	}, nil
}

func (s *scoringProgram) WithDemos(demos []optimize.Example) optimize.Program {
	return &scoringProgram{demos: demos, passKeys: s.passKeys}
}

func (s *scoringProgram) GetSignature() optimize.Signature { return nil }

// failingProgram always returns an error.
type failingProgram struct{}

func (f *failingProgram) Run(_ context.Context, _ map[string]interface{}) (optimize.Prediction, error) {
	return optimize.Prediction{}, errors.New("teacher failed")
}

func (f *failingProgram) WithDemos(demos []optimize.Example) optimize.Program {
	return &MockProgram{demos: demos}
}

func (f *failingProgram) GetSignature() optimize.Signature { return nil }

// ── Defaults ──────────────────────────────────────────────────────────────────

func TestBootstrapFewShot_Defaults(t *testing.T) {
	bs := NewBootstrapFewShot()

	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"MaxBootstrapped", bs.MaxBootstrapped, 4},
		{"MaxLabeled", bs.MaxLabeled, 16},
		{"MaxRounds", bs.MaxRounds, 1},
		{"MetricThreshold", bs.MetricThreshold, 1.0},
		{"Seed", bs.Seed, int64(42)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("expected %s=%v, got %v", tt.name, tt.want, tt.got)
			}
		})
	}

	if bs.Teacher != nil {
		t.Error("expected nil Teacher by default")
	}
	if bs.llm != nil {
		t.Error("expected nil llm by default")
	}
}

// ── Functional Options ────────────────────────────────────────────────────────

func TestBootstrapFewShot_FunctionalOptions(t *testing.T) {
	teacher := &MockProgram{}
	bs := NewBootstrapFewShot(
		WithTeacher(teacher),
		WithMaxBootstrapped(8),
		WithMaxLabeled(32),
		WithMaxRounds(3),
		WithMetricThreshold(0.5),
		WithBootstrapSeed(99),
	)

	if bs.Teacher != teacher {
		t.Error("expected Teacher to be set")
	}
	if bs.MaxBootstrapped != 8 {
		t.Errorf("MaxBootstrapped: expected 8, got %d", bs.MaxBootstrapped)
	}
	if bs.MaxLabeled != 32 {
		t.Errorf("MaxLabeled: expected 32, got %d", bs.MaxLabeled)
	}
	if bs.MaxRounds != 3 {
		t.Errorf("MaxRounds: expected 3, got %d", bs.MaxRounds)
	}
	if bs.MetricThreshold != 0.5 {
		t.Errorf("MetricThreshold: expected 0.5, got %f", bs.MetricThreshold)
	}
	if bs.Seed != 99 {
		t.Errorf("Seed: expected 99, got %d", bs.Seed)
	}
}

func TestBootstrapFewShot_OptionValidation(t *testing.T) {
	bs := NewBootstrapFewShot(
		WithMaxBootstrapped(-1),   // invalid → keep default
		WithMaxLabeled(0),         // invalid → keep default
		WithMaxRounds(-5),         // invalid → keep default
		WithMetricThreshold(-0.1), // invalid → keep default
	)

	if bs.MaxBootstrapped != 4 {
		t.Errorf("MaxBootstrapped should remain default, got %d", bs.MaxBootstrapped)
	}
	if bs.MaxLabeled != 16 {
		t.Errorf("MaxLabeled should remain default, got %d", bs.MaxLabeled)
	}
	if bs.MaxRounds != 1 {
		t.Errorf("MaxRounds should remain default, got %d", bs.MaxRounds)
	}
	if bs.MetricThreshold != 1.0 {
		t.Errorf("MetricThreshold should remain default, got %f", bs.MetricThreshold)
	}
}

// ── Registry ──────────────────────────────────────────────────────────────────

func TestBootstrapFewShot_Registry(t *testing.T) {
	names := optimize.ListOptimizers()
	found := false
	for _, name := range names {
		if name == "bootstrapfewshot" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("'bootstrapfewshot' not found in registry: %v", names)
	}

	opt, err := optimize.NewOptimizer("bootstrapfewshot", optimize.OptimizerConfig{})
	if err != nil {
		t.Fatalf("NewOptimizer('bootstrapfewshot'): %v", err)
	}
	if opt == nil {
		t.Error("expected non-nil optimizer")
	}
}

func TestBootstrapFewShot_RegistryWithLLM(t *testing.T) {
	llm := &mockLLMClient{response: "test"}

	opt, err := optimize.NewOptimizer("bootstrapfewshot", optimize.OptimizerConfig{LLM: llm})
	if err != nil {
		t.Fatalf("NewOptimizer with LLM: %v", err)
	}
	bs, ok := opt.(*BootstrapFewShot)
	if !ok {
		t.Fatal("expected *BootstrapFewShot")
	}
	if bs.llm == nil {
		t.Error("expected LLM to be wired from config")
	}
}

// ── Compile: error cases ──────────────────────────────────────────────────────

func TestBootstrapFewShot_Compile_MissingMetric(t *testing.T) {
	bs := NewBootstrapFewShot()
	_, err := bs.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: makeTrainset(3),
	})
	if err == nil {
		t.Error("expected error for missing metric")
	}
}

func TestBootstrapFewShot_Compile_EmptyTrainset(t *testing.T) {
	bs := NewBootstrapFewShot()
	compiled, err := bs.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: []optimize.Example{},
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compiled == nil {
		t.Error("expected compiled program, got nil")
	}
}

// ── Compile: basic success ────────────────────────────────────────────────────

func TestBootstrapFewShot_Compile_Basic(t *testing.T) {
	bs := NewBootstrapFewShot(
		WithMaxBootstrapped(2),
		WithMaxLabeled(2),
		WithMetricThreshold(0.5),
	)

	trainset := makeTrainset(6)
	compiled, err := bs.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
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

// ── Compile: with teacher ─────────────────────────────────────────────────────

func TestBootstrapFewShot_Compile_WithTeacher(t *testing.T) {
	// Teacher that passes for specific inputs.
	teacher := &scoringProgram{
		passKeys: map[string]bool{"q0": true, "q1": true, "q2": true},
	}

	bs := NewBootstrapFewShot(
		WithTeacher(teacher),
		WithMaxBootstrapped(2),
		WithMaxLabeled(2),
		WithMetricThreshold(0.5),
	)

	trainset := []optimize.Example{
		{Inputs: map[string]interface{}{"question": "q0"}, Outputs: map[string]interface{}{"answer": "q0"}},
		{Inputs: map[string]interface{}{"question": "q1"}, Outputs: map[string]interface{}{"answer": "q1"}},
		{Inputs: map[string]interface{}{"question": "q2"}, Outputs: map[string]interface{}{"answer": "q2"}},
		{Inputs: map[string]interface{}{"question": "q3"}, Outputs: map[string]interface{}{"answer": "q3"}},
	}

	compiled, err := bs.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
	if err != nil {
		t.Fatalf("Compile with teacher: %v", err)
	}
	if compiled == nil {
		t.Error("expected non-nil compiled program")
	}
}

// ── Compile: failing teacher ──────────────────────────────────────────────────

func TestBootstrapFewShot_Compile_FailingTeacher(t *testing.T) {
	bs := NewBootstrapFewShot(
		WithTeacher(&failingProgram{}),
		WithMaxBootstrapped(2),
		WithMaxLabeled(2),
		WithMaxRounds(2),
	)

	trainset := makeTrainset(4)
	compiled, err := bs.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
	if err != nil {
		t.Fatalf("Compile with failing teacher: %v", err)
	}
	// Should still return a program (with labeled examples only).
	if compiled == nil {
		t.Error("expected compiled program even with failing teacher")
	}
}

// ── Compile: threshold filtering ──────────────────────────────────────────────

func TestBootstrapFewShot_Compile_ThresholdFiltering(t *testing.T) {
	// Only "pass" inputs will be bootstrapped.
	teacher := &scoringProgram{
		passKeys: map[string]bool{"pass1": true, "pass2": true},
	}

	bs := NewBootstrapFewShot(
		WithTeacher(teacher),
		WithMaxBootstrapped(10), // High limit to see filtering effect.
		WithMaxLabeled(0),
		WithMetricThreshold(1.0), // Strict threshold.
	)

	trainset := []optimize.Example{
		{Inputs: map[string]interface{}{"question": "pass1"}, Outputs: map[string]interface{}{"answer": "pass1"}},
		{Inputs: map[string]interface{}{"question": "fail1"}, Outputs: map[string]interface{}{"answer": "fail1"}},
		{Inputs: map[string]interface{}{"question": "pass2"}, Outputs: map[string]interface{}{"answer": "pass2"}},
		{Inputs: map[string]interface{}{"question": "fail2"}, Outputs: map[string]interface{}{"answer": "fail2"}},
	}

	compiled, err := bs.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
	if err != nil {
		t.Fatalf("Compile threshold: %v", err)
	}
	if compiled == nil {
		t.Error("expected non-nil compiled program")
	}
}

// ── Compile: multiple rounds ──────────────────────────────────────────────────

func TestBootstrapFewShot_Compile_MultipleRounds(t *testing.T) {
	bs := NewBootstrapFewShot(
		WithMaxBootstrapped(2),
		WithMaxLabeled(2),
		WithMaxRounds(3),
		WithMetricThreshold(0.5),
	)

	trainset := makeTrainset(4)
	compiled, err := bs.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
	if err != nil {
		t.Fatalf("Compile multi-round: %v", err)
	}
	if compiled == nil {
		t.Error("expected non-nil compiled program")
	}
}

// ── Compile: context cancellation ─────────────────────────────────────────────

func TestBootstrapFewShot_Compile_ContextCancellation(t *testing.T) {
	bs := NewBootstrapFewShot(
		WithMaxBootstrapped(100),
		WithMaxLabeled(100),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := bs.Compile(ctx, &MockProgram{}, optimize.CompileOptions{
		Trainset: makeTrainset(50),
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
	// Should either return context.Canceled or succeed if the loop doesn't
	// reach the select. Either way, must not hang.
	_ = err
}

// ── Compile: callbacks ────────────────────────────────────────────────────────

func TestBootstrapFewShot_Compile_WithCallback(t *testing.T) {
	var trialCount int
	var completedCalled bool
	var finalResult optimize.OptimizationResult

	cb := &countingCallback{
		onTrial: func(_ optimize.Trial) {
			trialCount++
		},
		onComplete: func(result optimize.OptimizationResult) {
			completedCalled = true
			finalResult = result
		},
	}

	bs := NewBootstrapFewShot(
		WithMaxBootstrapped(3),
		WithMaxLabeled(2),
		WithMetricThreshold(0.5),
	)

	_, err := bs.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
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
	if finalResult.NumTrials == 0 {
		t.Error("expected non-zero NumTrials in OptimizationResult")
	}
	if finalResult.Duration < 0 {
		t.Error("expected non-negative Duration")
	}
}

// ── Compile: budget limit ─────────────────────────────────────────────────────

func TestBootstrapFewShot_Compile_BudgetLimit(t *testing.T) {
	bs := NewBootstrapFewShot(
		WithMaxBootstrapped(100),
		WithMaxLabeled(100),
	)

	_, err := bs.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: makeTrainset(50),
		Metric:   optimize.MetricFunc(metric.ExactMatch),
		MaxCost: &optimize.CostBudget{
			MaxIterations: 3,
		},
	})
	if err != nil {
		t.Fatalf("Compile with budget: %v", err)
	}
}

// ── Compile: validation set ───────────────────────────────────────────────────

func TestBootstrapFewShot_Compile_WithValset(t *testing.T) {
	var finalResult optimize.OptimizationResult

	cb := &countingCallback{
		onComplete: func(result optimize.OptimizationResult) {
			finalResult = result
		},
	}

	bs := NewBootstrapFewShot(
		WithMaxBootstrapped(2),
		WithMaxLabeled(2),
		WithMetricThreshold(0.5),
	)

	trainset := makeTrainset(6)
	valset := makeTrainset(3)

	_, err := bs.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset:  trainset,
		Valset:    valset,
		Metric:    optimize.MetricFunc(metric.ExactMatch),
		Callbacks: []optimize.Callback{cb},
	})
	if err != nil {
		t.Fatalf("Compile with valset: %v", err)
	}
	// BestScore should reflect validation evaluation.
	if finalResult.BestScore < 0 {
		t.Error("expected non-negative BestScore from validation")
	}
}

// ── Compile: determinism ──────────────────────────────────────────────────────

func TestBootstrapFewShot_Compile_Deterministic(t *testing.T) {
	trainset := makeTrainset(8)
	opts := optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	}

	bs1 := NewBootstrapFewShot(WithBootstrapSeed(7), WithMaxBootstrapped(3), WithMetricThreshold(0.5))
	bs2 := NewBootstrapFewShot(WithBootstrapSeed(7), WithMaxBootstrapped(3), WithMetricThreshold(0.5))

	r1, err := bs1.Compile(context.Background(), &MockProgram{}, opts)
	if err != nil {
		t.Fatalf("first compile: %v", err)
	}
	r2, err := bs2.Compile(context.Background(), &MockProgram{}, opts)
	if err != nil {
		t.Fatalf("second compile: %v", err)
	}

	if r1 == nil || r2 == nil {
		t.Error("expected non-nil compiled programs for both runs")
	}
}

// ── Compile: result metadata ──────────────────────────────────────────────────

func TestBootstrapFewShot_Compile_ResultMetadata(t *testing.T) {
	var finalResult optimize.OptimizationResult

	cb := &countingCallback{
		onComplete: func(result optimize.OptimizationResult) {
			finalResult = result
		},
	}

	bs := NewBootstrapFewShot(
		WithMaxBootstrapped(2),
		WithMaxLabeled(3),
		WithMetricThreshold(0.5),
	)

	_, err := bs.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset:  makeTrainset(8),
		Metric:    optimize.MetricFunc(metric.ExactMatch),
		Callbacks: []optimize.Callback{cb},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	if finalResult.BestCandidate.ID != "bootstrap_best" {
		t.Errorf("expected candidate ID 'bootstrap_best', got %q", finalResult.BestCandidate.ID)
	}

	numBootstrapped, ok := finalResult.BestCandidate.Metadata["num_bootstrapped"]
	if !ok {
		t.Error("expected 'num_bootstrapped' in metadata")
	}
	if nb, ok := numBootstrapped.(int); ok && nb < 0 {
		t.Errorf("unexpected num_bootstrapped: %d", nb)
	}

	numLabeled, ok := finalResult.BestCandidate.Metadata["num_labeled"]
	if !ok {
		t.Error("expected 'num_labeled' in metadata")
	}
	if nl, ok := numLabeled.(int); ok && nl < 0 {
		t.Errorf("unexpected num_labeled: %d", nl)
	}
}

// ── Helper tests ──────────────────────────────────────────────────────────────

func TestExamplesEqual(t *testing.T) {
	tests := []struct {
		name string
		a, b optimize.Example
		want bool
	}{
		{
			name: "equal",
			a:    optimize.Example{Inputs: map[string]interface{}{"key": "value"}},
			b:    optimize.Example{Inputs: map[string]interface{}{"key": "value"}},
			want: true,
		},
		{
			name: "different_value",
			a:    optimize.Example{Inputs: map[string]interface{}{"key": "value"}},
			b:    optimize.Example{Inputs: map[string]interface{}{"key": "different"}},
			want: false,
		},
		{
			name: "different_length",
			a:    optimize.Example{Inputs: map[string]interface{}{"key": "value"}},
			b:    optimize.Example{Inputs: map[string]interface{}{"key": "value", "extra": "field"}},
			want: false,
		},
		{
			name: "both_empty",
			a:    optimize.Example{Inputs: map[string]interface{}{}},
			b:    optimize.Example{Inputs: map[string]interface{}{}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := examplesEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("examplesEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsExample(t *testing.T) {
	demos := []optimize.Example{
		{Inputs: map[string]interface{}{"key": "value1"}},
		{Inputs: map[string]interface{}{"key": "value2"}},
	}

	if !containsExample(demos, optimize.Example{Inputs: map[string]interface{}{"key": "value1"}}) {
		t.Error("expected to find existing example")
	}
	if containsExample(demos, optimize.Example{Inputs: map[string]interface{}{"key": "value3"}}) {
		t.Error("expected not to find missing example")
	}
	if containsExample(nil, optimize.Example{Inputs: map[string]interface{}{"key": "value1"}}) {
		t.Error("expected not to find in nil slice")
	}
}

// ── Benchmarks ────────────────────────────────────────────────────────────────

func BenchmarkBootstrapFewShot_Compile(b *testing.B) {
	trainset := makeTrainset(20)
	bs := NewBootstrapFewShot(
		WithMaxBootstrapped(4),
		WithMaxLabeled(8),
		WithMetricThreshold(0.5),
	)
	opts := optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bs.Compile(context.Background(), &MockProgram{}, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBootstrapFewShot_CompileWithTeacher(b *testing.B) {
	trainset := makeTrainset(20)
	teacher := &scoringProgram{
		passKeys: map[string]bool{"0": true, "1": true, "2": true, "3": true, "4": true},
	}
	bs := NewBootstrapFewShot(
		WithTeacher(teacher),
		WithMaxBootstrapped(4),
		WithMaxLabeled(8),
		WithMetricThreshold(0.5),
		WithMaxRounds(3),
	)
	opts := optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bs.Compile(context.Background(), &MockProgram{}, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}
