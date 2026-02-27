package optimizers

import (
	"context"
	"math"
	"math/rand"
	"testing"

	"github.com/lookatitude/beluga-ai/optimize"
	"github.com/lookatitude/beluga-ai/optimize/metric"
)

func TestSIMBA_New(t *testing.T) {
	s := NewSIMBA(
		WithSIMBAMaxIterations(20),
		WithSIMBAMinibatchSize(30),
		WithSIMBACandidatePoolSize(12),
	)

	if s.MaxIterations != 20 {
		t.Errorf("expected MaxIterations=20, got %d", s.MaxIterations)
	}
	if s.MinibatchSize != 30 {
		t.Errorf("expected MinibatchSize=30, got %d", s.MinibatchSize)
	}
	if s.CandidatePoolSize != 12 {
		t.Errorf("expected CandidatePoolSize=12, got %d", s.CandidatePoolSize)
	}
}

func TestSIMBA_Defaults(t *testing.T) {
	s := NewSIMBA()

	if s.MaxIterations != 15 {
		t.Errorf("expected default MaxIterations=15, got %d", s.MaxIterations)
	}
	if s.MinibatchSize != 20 {
		t.Errorf("expected default MinibatchSize=20, got %d", s.MinibatchSize)
	}
	if s.CandidatePoolSize != 8 {
		t.Errorf("expected default CandidatePoolSize=8, got %d", s.CandidatePoolSize)
	}
	if s.SamplingTemperature != 0.2 {
		t.Errorf("expected default SamplingTemperature=0.2, got %f", s.SamplingTemperature)
	}
	if s.ConvergenceThreshold != 0.001 {
		t.Errorf("expected default ConvergenceThreshold=0.001, got %f", s.ConvergenceThreshold)
	}
	if s.MinVariabilityThreshold != 0.3 {
		t.Errorf("expected default MinVariabilityThreshold=0.3, got %f", s.MinVariabilityThreshold)
	}
	if s.Seed != 42 {
		t.Errorf("expected default Seed=42, got %d", s.Seed)
	}
}

func TestSIMBA_Options(t *testing.T) {
	s := NewSIMBA(
		WithSIMBASamplingTemperature(0.5),
		WithSIMBAConvergenceThreshold(0.01),
		WithSIMBAMinVariabilityThreshold(0.5),
		WithSIMBASeed(123),
	)

	if s.SamplingTemperature != 0.5 {
		t.Errorf("expected SamplingTemperature=0.5, got %f", s.SamplingTemperature)
	}
	if s.ConvergenceThreshold != 0.01 {
		t.Errorf("expected ConvergenceThreshold=0.01, got %f", s.ConvergenceThreshold)
	}
	if s.MinVariabilityThreshold != 0.5 {
		t.Errorf("expected MinVariabilityThreshold=0.5, got %f", s.MinVariabilityThreshold)
	}
	if s.Seed != 123 {
		t.Errorf("expected Seed=123, got %d", s.Seed)
	}
}

func TestSIMBA_OptionValidation(t *testing.T) {
	// Invalid values should be ignored
	s := NewSIMBA(
		WithSIMBAMaxIterations(-1),
		WithSIMBAMinibatchSize(0),
		WithSIMBACandidatePoolSize(-5),
		WithSIMBASamplingTemperature(2.0), // > 1.0
		WithSIMBAConvergenceThreshold(-1),
	)

	if s.MaxIterations != 15 {
		t.Errorf("expected default MaxIterations=15, got %d", s.MaxIterations)
	}
	if s.MinibatchSize != 20 {
		t.Errorf("expected default MinibatchSize=20, got %d", s.MinibatchSize)
	}
	if s.CandidatePoolSize != 8 {
		t.Errorf("expected default CandidatePoolSize=8, got %d", s.CandidatePoolSize)
	}
	if s.SamplingTemperature != 0.2 {
		t.Errorf("expected default SamplingTemperature=0.2, got %f", s.SamplingTemperature)
	}
	if s.ConvergenceThreshold != 0.001 {
		t.Errorf("expected default ConvergenceThreshold=0.001, got %f", s.ConvergenceThreshold)
	}
}

func TestSIMBA_Registry(t *testing.T) {
	optimizers := optimize.ListOptimizers()
	found := false
	for _, name := range optimizers {
		if name == "simba" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("simba optimizer not found in registry: %v", optimizers)
	}

	opt, err := optimize.NewOptimizer("simba", optimize.OptimizerConfig{})
	if err != nil {
		t.Fatalf("failed to create optimizer from registry: %v", err)
	}
	if opt == nil {
		t.Error("expected optimizer, got nil")
	}
}

func TestSIMBA_Compile_MissingMetric(t *testing.T) {
	s := NewSIMBA()
	student := &MockProgram{}

	_, err := s.Compile(context.Background(), student, optimize.CompileOptions{
		Trainset: []optimize.Example{{Inputs: map[string]interface{}{"q": "1"}}},
	})

	if err == nil {
		t.Error("expected error when metric is missing")
	}
}

func TestSIMBA_Compile_EmptyTrainset(t *testing.T) {
	s := NewSIMBA()
	student := &MockProgram{}

	_, err := s.Compile(context.Background(), student, optimize.CompileOptions{
		Trainset: []optimize.Example{},
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})

	if err == nil {
		t.Error("expected error when trainset is empty")
	}
}

func TestSIMBA_Compile_WithTrainset(t *testing.T) {
	s := NewSIMBA(
		WithSIMBAMaxIterations(3),
		WithSIMBACandidatePoolSize(4),
		WithSIMBAMinibatchSize(3),
	)
	student := &MockProgram{}

	trainset := []optimize.Example{
		{Inputs: map[string]interface{}{"q": "1"}, Outputs: map[string]interface{}{"a": "1"}},
		{Inputs: map[string]interface{}{"q": "2"}, Outputs: map[string]interface{}{"a": "2"}},
		{Inputs: map[string]interface{}{"q": "3"}, Outputs: map[string]interface{}{"a": "3"}},
		{Inputs: map[string]interface{}{"q": "4"}, Outputs: map[string]interface{}{"a": "4"}},
		{Inputs: map[string]interface{}{"q": "5"}, Outputs: map[string]interface{}{"a": "5"}},
	}

	compiled, err := s.Compile(context.Background(), student, optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compiled == nil {
		t.Error("expected compiled program, got nil")
	}
}

func TestSIMBA_Compile_WithCostBudget(t *testing.T) {
	s := NewSIMBA(
		WithSIMBAMaxIterations(100), // High max, but budget should stop us
		WithSIMBACandidatePoolSize(4),
		WithSIMBAMinibatchSize(2),
	)
	student := &MockProgram{}

	trainset := []optimize.Example{
		{Inputs: map[string]interface{}{"q": "1"}, Outputs: map[string]interface{}{"a": "1"}},
		{Inputs: map[string]interface{}{"q": "2"}, Outputs: map[string]interface{}{"a": "2"}},
	}

	compiled, err := s.Compile(context.Background(), student, optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
		MaxCost:  &optimize.CostBudget{MaxIterations: 2},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compiled == nil {
		t.Error("expected compiled program, got nil")
	}
}

func TestSIMBA_InitializeCandidatePool(t *testing.T) {
	s := NewSIMBA(WithSIMBACandidatePoolSize(5))

	trainset := []optimize.Example{
		{Inputs: map[string]interface{}{"q": "1"}},
		{Inputs: map[string]interface{}{"q": "2"}},
		{Inputs: map[string]interface{}{"q": "3"}},
		{Inputs: map[string]interface{}{"q": "4"}},
		{Inputs: map[string]interface{}{"q": "5"}},
	}

	rng := rand.New(rand.NewSource(42))
	pool := s.initializeCandidatePool(trainset, rng)

	if len(pool) != 5 {
		t.Errorf("expected 5 candidates, got %d", len(pool))
	}

	for i, c := range pool {
		if c.Prompt == "" {
			t.Errorf("candidate %d has no prompt", i)
		}
		if len(c.Demos) == 0 {
			t.Errorf("candidate %d has no demos", i)
		}
		if c.ID == "" {
			t.Errorf("candidate %d has no ID", i)
		}
	}
}

func TestSIMBA_SampleMinibatch(t *testing.T) {
	s := NewSIMBA(WithSIMBAMinibatchSize(3))

	trainset := []optimize.Example{
		{Inputs: map[string]interface{}{"q": "1"}},
		{Inputs: map[string]interface{}{"q": "2"}},
		{Inputs: map[string]interface{}{"q": "3"}},
		{Inputs: map[string]interface{}{"q": "4"}},
		{Inputs: map[string]interface{}{"q": "5"}},
	}

	rng := rand.New(rand.NewSource(42))
	batch := s.sampleMinibatch(trainset, rng)

	if len(batch) != 3 {
		t.Errorf("expected 3 examples, got %d", len(batch))
	}
}

func TestSIMBA_SampleMinibatch_LargerThanTrainset(t *testing.T) {
	s := NewSIMBA(WithSIMBAMinibatchSize(10))

	trainset := []optimize.Example{
		{Inputs: map[string]interface{}{"q": "1"}},
		{Inputs: map[string]interface{}{"q": "2"}},
	}

	rng := rand.New(rand.NewSource(42))
	batch := s.sampleMinibatch(trainset, rng)

	if len(batch) != 2 {
		t.Errorf("expected 2 examples (capped at trainset size), got %d", len(batch))
	}
}

func TestSoftmax(t *testing.T) {
	tests := []struct {
		name        string
		scores      []float64
		temperature float64
		wantLen     int
	}{
		{
			name:        "basic scores",
			scores:      []float64{1.0, 2.0, 3.0},
			temperature: 1.0,
			wantLen:     3,
		},
		{
			name:        "low temperature (exploit)",
			scores:      []float64{0.1, 0.9},
			temperature: 0.01,
			wantLen:     2,
		},
		{
			name:        "high temperature (explore)",
			scores:      []float64{0.1, 0.9},
			temperature: 1.0,
			wantLen:     2,
		},
		{
			name:        "equal scores",
			scores:      []float64{0.5, 0.5, 0.5},
			temperature: 0.2,
			wantLen:     3,
		},
		{
			name:        "empty",
			scores:      []float64{},
			temperature: 0.2,
			wantLen:     0,
		},
		{
			name:        "single element",
			scores:      []float64{1.0},
			temperature: 0.2,
			wantLen:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			probs := softmax(tt.scores, tt.temperature)
			if len(probs) != tt.wantLen {
				t.Errorf("expected %d probabilities, got %d", tt.wantLen, len(probs))
				return
			}
			if tt.wantLen == 0 {
				return
			}

			// Probabilities should sum to ~1.0
			sum := 0.0
			for _, p := range probs {
				sum += p
				if p < 0 || p > 1 {
					t.Errorf("probability %f outside [0,1]", p)
				}
			}
			if math.Abs(sum-1.0) > 1e-6 {
				t.Errorf("probabilities sum to %f, expected ~1.0", sum)
			}
		})
	}
}

func TestSoftmax_LowTemperatureConcentrates(t *testing.T) {
	scores := []float64{0.1, 0.5, 0.9}

	lowTemp := softmax(scores, 0.01)
	highTemp := softmax(scores, 1.0)

	// Low temperature should concentrate probability on the highest score
	if lowTemp[2] < highTemp[2] {
		t.Error("expected low temperature to give higher probability to best score")
	}

	// Low temperature: best score should dominate
	if lowTemp[2] < 0.9 {
		t.Errorf("expected low-temp prob for best score > 0.9, got %f", lowTemp[2])
	}
}

func TestSoftmax_EqualScoresUniform(t *testing.T) {
	scores := []float64{0.5, 0.5, 0.5, 0.5}
	probs := softmax(scores, 0.2)

	expected := 0.25
	for i, p := range probs {
		if math.Abs(p-expected) > 1e-6 {
			t.Errorf("prob[%d]=%f, expected %f for equal scores", i, p, expected)
		}
	}
}

func TestVariance(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		want   float64
	}{
		{"empty", []float64{}, 0},
		{"single", []float64{5.0}, 0},
		{"identical", []float64{3.0, 3.0, 3.0}, 0},
		{"simple", []float64{1.0, 3.0}, 1.0},     // mean=2, var=((1-2)^2+(3-2)^2)/2=1
		{"another", []float64{2.0, 4.0, 4.0, 4.0, 5.0, 5.0, 7.0, 9.0}, 4.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := variance(tt.values)
			if math.Abs(got-tt.want) > 1e-6 {
				t.Errorf("variance(%v) = %f, want %f", tt.values, got, tt.want)
			}
		})
	}
}

func TestWeightedSample(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	// With deterministic probs, sampling should be biased
	probs := []float64{0.0, 0.0, 1.0} // Always pick index 2
	for i := 0; i < 10; i++ {
		idx := weightedSample(probs, rng)
		if idx != 2 {
			t.Errorf("expected index 2, got %d", idx)
		}
	}
}

func TestSIMBA_SoftmaxSelect(t *testing.T) {
	s := NewSIMBA(WithSIMBASamplingTemperature(0.2))

	candidates := []simbaCandidate{
		{ID: "a", Score: 0.1},
		{ID: "b", Score: 0.5},
		{ID: "c", Score: 0.9},
		{ID: "d", Score: 0.3},
		{ID: "e", Score: 0.7},
	}

	rng := rand.New(rand.NewSource(42))
	selected := s.softmaxSelect(candidates, 3, rng)

	if len(selected) != 3 {
		t.Errorf("expected 3 selected, got %d", len(selected))
	}

	// Ensure no duplicates
	seen := make(map[string]bool)
	for _, c := range selected {
		if seen[c.ID] {
			t.Errorf("duplicate candidate selected: %s", c.ID)
		}
		seen[c.ID] = true
	}
}

func TestSIMBA_SoftmaxSelect_FewerThanN(t *testing.T) {
	s := NewSIMBA()

	candidates := []simbaCandidate{
		{ID: "a", Score: 0.5},
		{ID: "b", Score: 0.9},
	}

	rng := rand.New(rand.NewSource(42))
	selected := s.softmaxSelect(candidates, 5, rng)

	if len(selected) != 2 {
		t.Errorf("expected 2 (all candidates), got %d", len(selected))
	}
}

func TestSIMBA_IdentifyChallengingExamples(t *testing.T) {
	s := NewSIMBA(WithSIMBAMinVariabilityThreshold(0.1))

	// Create candidates with different demo sets to produce varied scores
	pool := []simbaCandidate{
		{ID: "a", Demos: []optimize.Example{{Inputs: map[string]interface{}{"q": "1"}, Outputs: map[string]interface{}{"a": "1"}}}},
		{ID: "b", Demos: []optimize.Example{{Inputs: map[string]interface{}{"q": "2"}, Outputs: map[string]interface{}{"a": "2"}}}},
	}

	examples := []optimize.Example{
		{Inputs: map[string]interface{}{"q": "1"}, Outputs: map[string]interface{}{"a": "1"}},
	}

	// With MockProgram, all candidates return same output, so variability is 0
	ctx := context.Background()
	challenging := s.identifyChallengingExamples(ctx, &MockProgram{}, pool, examples, optimize.MetricFunc(metric.ExactMatch))

	// MockProgram always returns input as output, so all candidates score the same
	// → variance is 0 → no challenging examples
	if len(challenging) != 0 {
		t.Errorf("expected 0 challenging examples with uniform mock, got %d", len(challenging))
	}
}

func TestSIMBA_Reflect(t *testing.T) {
	s := NewSIMBA()

	// Pool with low scores → should suggest diversify_prompts
	pool := []simbaCandidate{
		{ID: "a", Score: 0.1},
		{ID: "b", Score: 0.2},
	}
	challenging := []optimize.Example{
		{Inputs: map[string]interface{}{"q": "1"}},
	}

	rules := s.reflect(context.Background(), pool, challenging, 0)

	if !containsRule(rules, "focus_challenging") {
		t.Error("expected focus_challenging rule when challenging examples exist")
	}
	if !containsRule(rules, "diversify_prompts") {
		t.Error("expected diversify_prompts rule when best score < 0.5")
	}
}

func TestSIMBA_Reflect_ClusteredScores(t *testing.T) {
	s := NewSIMBA()

	pool := []simbaCandidate{
		{ID: "a", Score: 0.8},
		{ID: "b", Score: 0.8},
		{ID: "c", Score: 0.8},
	}

	rules := s.reflect(context.Background(), pool, nil, 0)

	if !containsRule(rules, "increase_exploration") {
		t.Error("expected increase_exploration rule when scores are tightly clustered")
	}
}

func TestSIMBA_GenerateImprovedCandidates(t *testing.T) {
	s := NewSIMBA(WithSIMBACandidatePoolSize(6))

	pool := []simbaCandidate{
		{ID: "a", Score: 0.5, Prompt: "Base prompt"},
		{ID: "b", Score: 0.8, Prompt: "Better prompt"},
	}

	trainset := []optimize.Example{
		{Inputs: map[string]interface{}{"q": "1"}},
		{Inputs: map[string]interface{}{"q": "2"}},
		{Inputs: map[string]interface{}{"q": "3"}},
		{Inputs: map[string]interface{}{"q": "4"}},
	}

	rng := rand.New(rand.NewSource(42))
	newCandidates := s.generateImprovedCandidates(pool, nil, trainset, nil, 0, rng)

	if len(newCandidates) != 3 { // CandidatePoolSize/2
		t.Errorf("expected 3 new candidates, got %d", len(newCandidates))
	}

	for i, c := range newCandidates {
		if c.Prompt == "" {
			t.Errorf("new candidate %d has no prompt", i)
		}
		if len(c.Demos) == 0 {
			t.Errorf("new candidate %d has no demos", i)
		}
	}
}

func TestSIMBA_GenerateImprovedCandidates_WithChallenging(t *testing.T) {
	s := NewSIMBA(WithSIMBACandidatePoolSize(4))

	pool := []simbaCandidate{
		{ID: "a", Score: 0.5, Prompt: "Base"},
	}

	challenging := []optimize.Example{
		{Inputs: map[string]interface{}{"q": "hard1"}},
	}

	trainset := []optimize.Example{
		{Inputs: map[string]interface{}{"q": "1"}},
		{Inputs: map[string]interface{}{"q": "2"}},
		{Inputs: map[string]interface{}{"q": "3"}},
		{Inputs: map[string]interface{}{"q": "4"}},
	}

	rules := []string{"focus_challenging"}

	rng := rand.New(rand.NewSource(42))
	newCandidates := s.generateImprovedCandidates(pool, challenging, trainset, rules, 0, rng)

	if len(newCandidates) != 2 { // CandidatePoolSize/2
		t.Errorf("expected 2 new candidates, got %d", len(newCandidates))
	}
}

func TestContainsRule(t *testing.T) {
	rules := []string{"focus_challenging", "increase_exploration"}

	if !containsRule(rules, "focus_challenging") {
		t.Error("expected to find focus_challenging")
	}
	if containsRule(rules, "nonexistent") {
		t.Error("expected not to find nonexistent rule")
	}
	if containsRule(nil, "anything") {
		t.Error("expected nil rules to not contain anything")
	}
}

func TestSortCandidatesByScore(t *testing.T) {
	candidates := []simbaCandidate{
		{ID: "a", Score: 0.3},
		{ID: "b", Score: 0.9},
		{ID: "c", Score: 0.1},
		{ID: "d", Score: 0.7},
	}

	sortCandidatesByScore(candidates)

	if candidates[0].ID != "b" || candidates[1].ID != "d" || candidates[2].ID != "a" || candidates[3].ID != "c" {
		t.Errorf("unexpected sort order: %v", candidates)
	}
}
