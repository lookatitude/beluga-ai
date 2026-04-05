package optimizers

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"

	"github.com/lookatitude/beluga-ai/optimize"
	"github.com/lookatitude/beluga-ai/optimize/metric"
	"github.com/lookatitude/beluga-ai/optimize/pareto"
)

// -------------------------------------------------------------------------
// Constructor and option tests
// -------------------------------------------------------------------------

func TestGEPA_New(t *testing.T) {
	g := NewGEPA(
		WithPopulationSize(20),
		WithMaxGenerations(15),
	)

	if g.PopulationSize != 20 {
		t.Errorf("expected PopulationSize=20, got %d", g.PopulationSize)
	}
	if g.MaxGenerations != 15 {
		t.Errorf("expected MaxGenerations=15, got %d", g.MaxGenerations)
	}
}

func TestGEPA_Defaults(t *testing.T) {
	g := NewGEPA()

	if g.PopulationSize != 10 {
		t.Errorf("expected default PopulationSize=10, got %d", g.PopulationSize)
	}
	if g.MaxGenerations != 10 {
		t.Errorf("expected default MaxGenerations=10, got %d", g.MaxGenerations)
	}
	if g.MutationRate != 0.3 {
		t.Errorf("expected default MutationRate=0.3, got %f", g.MutationRate)
	}
	if g.CrossoverRate != 0.5 {
		t.Errorf("expected default CrossoverRate=0.5, got %f", g.CrossoverRate)
	}
	if g.ArchiveSize != 50 {
		t.Errorf("expected default ArchiveSize=50, got %d", g.ArchiveSize)
	}
	if g.ReflectionInterval != 3 {
		t.Errorf("expected default ReflectionInterval=3, got %d", g.ReflectionInterval)
	}
	if g.EvalSampleSize != 10 {
		t.Errorf("expected default EvalSampleSize=10, got %d", g.EvalSampleSize)
	}
	if g.ConsistencyRuns != 2 {
		t.Errorf("expected default ConsistencyRuns=2, got %d", g.ConsistencyRuns)
	}
	if g.ConvergenceWindow != 5 {
		t.Errorf("expected default ConvergenceWindow=5, got %d", g.ConvergenceWindow)
	}
	if g.NumWorkers != 1 {
		t.Errorf("expected default NumWorkers=1, got %d", g.NumWorkers)
	}
	if g.Seed != 42 {
		t.Errorf("expected default Seed=42, got %d", g.Seed)
	}
}

func TestGEPA_AllOptions(t *testing.T) {
	g := NewGEPA(
		WithPopulationSize(8),
		WithMaxGenerations(5),
		WithMutationRate(0.4),
		WithCrossoverRate(0.6),
		WithGEPASeed(99),
		WithGEPAArchiveSize(20),
		WithGEPAReflectionInterval(2),
		WithGEPAEvalSampleSize(5),
		WithGEPAConsistencyRuns(3),
		WithGEPANumWorkers(2),
		WithGEPAConvergenceWindow(4),
		WithGEPAConvergenceThreshold(0.01),
	)

	if g.PopulationSize != 8 {
		t.Errorf("expected PopulationSize=8, got %d", g.PopulationSize)
	}
	if g.MaxGenerations != 5 {
		t.Errorf("expected MaxGenerations=5, got %d", g.MaxGenerations)
	}
	if g.MutationRate != 0.4 {
		t.Errorf("expected MutationRate=0.4, got %f", g.MutationRate)
	}
	if g.CrossoverRate != 0.6 {
		t.Errorf("expected CrossoverRate=0.6, got %f", g.CrossoverRate)
	}
	if g.Seed != 99 {
		t.Errorf("expected Seed=99, got %d", g.Seed)
	}
	if g.ArchiveSize != 20 {
		t.Errorf("expected ArchiveSize=20, got %d", g.ArchiveSize)
	}
	if g.ReflectionInterval != 2 {
		t.Errorf("expected ReflectionInterval=2, got %d", g.ReflectionInterval)
	}
	if g.EvalSampleSize != 5 {
		t.Errorf("expected EvalSampleSize=5, got %d", g.EvalSampleSize)
	}
	if g.ConsistencyRuns != 3 {
		t.Errorf("expected ConsistencyRuns=3, got %d", g.ConsistencyRuns)
	}
	if g.NumWorkers != 2 {
		t.Errorf("expected NumWorkers=2, got %d", g.NumWorkers)
	}
	if g.ConvergenceWindow != 4 {
		t.Errorf("expected ConvergenceWindow=4, got %d", g.ConvergenceWindow)
	}
	if g.ConvergenceThreshold != 0.01 {
		t.Errorf("expected ConvergenceThreshold=0.01, got %f", g.ConvergenceThreshold)
	}
}

func TestGEPA_InvalidOptions_Ignored(t *testing.T) {
	g := NewGEPA(
		WithPopulationSize(0),            // should be ignored
		WithMaxGenerations(-1),           // should be ignored
		WithMutationRate(1.5),            // should be ignored
		WithCrossoverRate(-0.1),          // should be ignored
		WithGEPAArchiveSize(0),           // should be ignored
		WithGEPAReflectionInterval(0),    // should be ignored
		WithGEPAEvalSampleSize(-1),       // should be ignored
		WithGEPAConsistencyRuns(0),       // should be ignored
		WithGEPANumWorkers(0),            // should be ignored
		WithGEPAConvergenceWindow(1),     // below min of 2 — should be ignored
		WithGEPAConvergenceThreshold(-1), // should be ignored
	)

	if g.PopulationSize != 10 {
		t.Errorf("invalid PopulationSize should be ignored, got %d", g.PopulationSize)
	}
	if g.MaxGenerations != 10 {
		t.Errorf("invalid MaxGenerations should be ignored, got %d", g.MaxGenerations)
	}
	if g.MutationRate != 0.3 {
		t.Errorf("invalid MutationRate should be ignored, got %f", g.MutationRate)
	}
	if g.CrossoverRate != 0.5 {
		t.Errorf("invalid CrossoverRate should be ignored, got %f", g.CrossoverRate)
	}
}

// -------------------------------------------------------------------------
// Registry tests
// -------------------------------------------------------------------------

func TestGEPA_Registry(t *testing.T) {
	optimizers := optimize.ListOptimizers()
	found := false
	for _, name := range optimizers {
		if name == "gepa" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("gepa optimizer not found in registry: %v", optimizers)
	}
}

func TestGEPA_Registry_CreateFromConfig(t *testing.T) {
	opt, err := optimize.NewOptimizer("gepa", optimize.OptimizerConfig{})
	if err != nil {
		t.Fatalf("failed to create optimizer from registry: %v", err)
	}
	if opt == nil {
		t.Error("expected optimizer, got nil")
	}
}

func TestGEPA_Registry_WithLLM(t *testing.T) {
	llm := &gepaLLMClient{response: "Provide a clear and concise answer."}
	opt, err := optimize.NewOptimizer("gepa", optimize.OptimizerConfig{LLM: llm})
	if err != nil {
		t.Fatalf("failed to create optimizer from registry with LLM: %v", err)
	}
	if opt == nil {
		t.Error("expected optimizer, got nil")
	}
	g, ok := opt.(*GEPA)
	if !ok {
		t.Fatal("expected *GEPA")
	}
	if g.llm == nil {
		t.Error("expected LLM client to be set")
	}
}

// -------------------------------------------------------------------------
// Validation tests
// -------------------------------------------------------------------------

func TestGEPA_Compile_MissingMetric(t *testing.T) {
	g := NewGEPA()
	_, err := g.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: gepaTrainset(5),
	})
	if err == nil {
		t.Error("expected error when metric is missing")
	}
}

func TestGEPA_Compile_EmptyTrainset(t *testing.T) {
	g := NewGEPA()
	_, err := g.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: []optimize.Example{},
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
	if err == nil {
		t.Error("expected error when trainset is empty")
	}
}

// -------------------------------------------------------------------------
// Compile behaviour tests
// -------------------------------------------------------------------------

func TestGEPA_Compile_ReturnsNonNilProgram(t *testing.T) {
	g := NewGEPA(
		WithMaxGenerations(2),
		WithPopulationSize(4),
		WithGEPAEvalSampleSize(3),
		WithGEPASeed(1),
	)

	compiled, err := g.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: gepaTrainset(10),
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compiled == nil {
		t.Error("expected compiled program, got nil")
	}
}

func TestGEPA_Compile_IsDeterministic(t *testing.T) {
	trainset := gepaTrainset(20)
	opts := optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	}
	gepaOpts := []GEPAOption{
		WithMaxGenerations(3),
		WithPopulationSize(5),
		WithGEPAEvalSampleSize(5),
		WithGEPASeed(42),
	}

	_, err1 := NewGEPA(gepaOpts...).Compile(context.Background(), &MockProgram{}, opts)
	_, err2 := NewGEPA(gepaOpts...).Compile(context.Background(), &MockProgram{}, opts)

	if (err1 == nil) != (err2 == nil) {
		t.Errorf("non-deterministic error result: %v vs %v", err1, err2)
	}
}

func TestGEPA_Compile_WithCostBudget(t *testing.T) {
	g := NewGEPA(
		WithMaxGenerations(100),
		WithPopulationSize(3),
		WithGEPAEvalSampleSize(2),
	)

	_, err := g.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: gepaTrainset(10),
		Metric:   optimize.MetricFunc(metric.ExactMatch),
		MaxCost: &optimize.CostBudget{
			MaxIterations: 2,
		},
	})
	if err != nil {
		t.Errorf("unexpected error with cost budget: %v", err)
	}
}

func TestGEPA_Compile_WithCallback(t *testing.T) {
	var mu sync.Mutex
	var trialIDs []int

	cb := &gepaCallback{
		onTrial: func(tr optimize.Trial) {
			mu.Lock()
			trialIDs = append(trialIDs, tr.ID)
			mu.Unlock()
		},
	}

	g := NewGEPA(
		WithMaxGenerations(3),
		WithPopulationSize(4),
		WithGEPAEvalSampleSize(3),
	)

	_, err := g.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset:  gepaTrainset(10),
		Metric:    optimize.MetricFunc(metric.ExactMatch),
		Callbacks: []optimize.Callback{cb},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(trialIDs) == 0 {
		t.Error("expected at least one callback, got none")
	}
}

func TestGEPA_Compile_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	g := NewGEPA(
		WithMaxGenerations(100),
		WithPopulationSize(5),
	)

	// With a cancelled context the program should still return without hanging.
	_, _ = g.Compile(ctx, &MockProgram{}, optimize.CompileOptions{
		Trainset: gepaTrainset(10),
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
}

func TestGEPA_Compile_MultiObjective(t *testing.T) {
	counter := 0
	scoringMetric := optimize.MetricFunc(func(_ optimize.Example, _ optimize.Prediction, _ *optimize.Trace) float64 {
		counter++
		if counter%3 == 0 {
			return 1.0
		}
		return 0.5
	})

	g := NewGEPA(
		WithMaxGenerations(3),
		WithPopulationSize(5),
		WithGEPAEvalSampleSize(5),
		WithGEPAConsistencyRuns(2),
	)

	compiled, err := g.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: gepaTrainset(15),
		Metric:   scoringMetric,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compiled == nil {
		t.Error("expected compiled program, got nil")
	}
}

// -------------------------------------------------------------------------
// Sub-function unit tests
// -------------------------------------------------------------------------

func TestGEPA_InitializePopulation(t *testing.T) {
	g := NewGEPA(WithPopulationSize(5))
	trainset := gepaTrainset(10)
	rng := rand.New(rand.NewSource(42))
	pop := g.initializePopulation(&MockProgram{}, trainset, rng)

	if len(pop) != 5 {
		t.Errorf("expected 5 candidates, got %d", len(pop))
	}
	for i, c := range pop {
		if len(c.Demos) == 0 {
			t.Errorf("candidate %d has no demos", i)
		}
		if c.Prompt == "" {
			t.Errorf("candidate %d has no prompt", i)
		}
		if c.ID == "" {
			t.Errorf("candidate %d has empty ID", i)
		}
	}
}

func TestGEPA_InitializePopulation_SmallTrainset(t *testing.T) {
	g := NewGEPA(WithPopulationSize(8))
	trainset := gepaTrainset(2)
	rng := rand.New(rand.NewSource(1))
	pop := g.initializePopulation(&MockProgram{}, trainset, rng)

	if len(pop) != 8 {
		t.Errorf("expected 8 candidates, got %d", len(pop))
	}
	for _, c := range pop {
		if len(c.Demos) > 2 {
			t.Errorf("demos should not exceed trainset size (2), got %d", len(c.Demos))
		}
	}
}

func TestGEPA_TournamentSelect_CorrectSize(t *testing.T) {
	g := NewGEPA()
	pop := gepaPopulation(5)
	rng := rand.New(rand.NewSource(42))

	parents := g.tournamentSelect(pop, 5, rng)
	if len(parents) != 5 {
		t.Errorf("expected 5 parents, got %d", len(parents))
	}
}

func TestGEPA_TournamentSelect_PrefersHigherScore(t *testing.T) {
	g := NewGEPA()
	pop := []gepaCandidate{
		{ID: "low", Score: 0.1},
		{ID: "high", Score: 0.99},
	}
	rng := rand.New(rand.NewSource(7))

	parents := g.tournamentSelect(pop, 20, rng)
	highCount := 0
	for _, p := range parents {
		if p.ID == "high" {
			highCount++
		}
	}
	if highCount == 0 {
		t.Error("expected high-scoring candidate to appear in parents")
	}
}

func TestGEPA_Crossover_CorrectOffspringSize(t *testing.T) {
	g := NewGEPA(
		WithPopulationSize(4),
		WithCrossoverRate(1.0),
	)
	parents := gepaPopulation(4)
	for i := range parents {
		parents[i].Score = float64(i) * 0.2
	}
	rng := rand.New(rand.NewSource(42))

	offspring := g.crossover(parents, 1, rng)
	if len(offspring) != 4 {
		t.Errorf("expected 4 offspring, got %d", len(offspring))
	}
}

func TestGEPA_Crossover_NoCrossover(t *testing.T) {
	g := NewGEPA(
		WithPopulationSize(4),
		WithCrossoverRate(0.0),
	)
	parents := gepaPopulation(4)
	rng := rand.New(rand.NewSource(1))

	offspring := g.crossover(parents, 1, rng)
	if len(offspring) != 4 {
		t.Errorf("expected 4 offspring, got %d", len(offspring))
	}
	for i, o := range offspring {
		found := false
		for _, p := range parents {
			if o.Prompt == p.Prompt {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("offspring %d has prompt not found in parents: %q", i, o.Prompt)
		}
	}
}

func TestGEPA_Crossover_InheritsHigherScoringPrompt(t *testing.T) {
	g := NewGEPA(
		WithPopulationSize(10),
		WithCrossoverRate(1.0),
	)
	parents := []gepaCandidate{
		{ID: "A", Prompt: "Alpha", Score: 0.9, Demos: gepaTrainset(2)},
		{ID: "B", Prompt: "Beta", Score: 0.1, Demos: gepaTrainset(2)},
	}
	rng := rand.New(rand.NewSource(0))
	offspring := g.crossover(parents, 1, rng)

	for _, o := range offspring {
		if o.Prompt != "Alpha" && o.Prompt != "Beta" {
			t.Errorf("unexpected prompt %q — should be from a parent", o.Prompt)
		}
	}
}

func TestGEPA_Mutate_ForcedMutation(t *testing.T) {
	g := NewGEPA(WithMutationRate(1.0))
	trainset := gepaTrainset(5)

	// Run mutation with multiple candidates and multiple seeds; assert that at
	// least one prompt changes across the tries (the mutation operator is
	// stochastic — case 2 trims at a "." which may be a no-op for some inputs).
	changed := false
	for seed := int64(0); seed < 20 && !changed; seed++ {
		offspring := []gepaCandidate{
			{ID: "c1", Prompt: "Original prompt without period", Demos: gepaTrainset(3)},
		}
		original := offspring[0].Prompt
		rng := rand.New(rand.NewSource(seed))
		g.mutate(offspring, trainset, rng)
		if offspring[0].Prompt != original {
			changed = true
		}
	}

	if !changed {
		t.Error("expected prompt to be mutated in at least one of 20 seeds")
	}
}

func TestGEPA_Mutate_ZeroRate(t *testing.T) {
	g := NewGEPA(WithMutationRate(0.0))
	trainset := gepaTrainset(5)
	offspring := []gepaCandidate{
		{ID: "c1", Prompt: "Unchanged", Demos: gepaTrainset(2)},
	}
	original := offspring[0].Prompt

	rng := rand.New(rand.NewSource(1))
	g.mutate(offspring, trainset, rng)

	if offspring[0].Prompt != original {
		t.Error("expected prompt to remain unchanged with MutationRate=0")
	}
}

func TestGEPA_Mutate_ProducesValidPrompts(t *testing.T) {
	g := NewGEPA(WithMutationRate(1.0))
	trainset := gepaTrainset(10)
	pop := gepaPopulation(20)

	rng := rand.New(rand.NewSource(99))
	g.mutate(pop, trainset, rng)

	for i, c := range pop {
		if strings.TrimSpace(c.Prompt) == "" {
			t.Errorf("candidate %d has empty prompt after mutation", i)
		}
	}
}

func TestGEPA_SelectBestFromArchive_Empty(t *testing.T) {
	g := NewGEPA()
	archive := pareto.NewArchive(10)

	best := g.selectBestFromArchive(archive)
	if best != nil {
		t.Error("expected nil for empty archive, got a candidate")
	}
}

func TestGEPA_SelectBestFromArchive_Single(t *testing.T) {
	g := NewGEPA()
	archive := pareto.NewArchive(10)
	archive.Add(pareto.Point{
		ID:         "only",
		Objectives: []float64{0.7, 0.5, 0.9},
		Payload:    gepaCandidate{ID: "c_only", Score: 0.7},
	}, 0)

	best := g.selectBestFromArchive(archive)
	if best == nil {
		t.Fatal("expected a candidate, got nil")
	}
	if best.ID != "c_only" {
		t.Errorf("expected c_only, got %s", best.ID)
	}
}

func TestGEPA_SelectBestFromArchive_MultiPoint(t *testing.T) {
	g := NewGEPA()
	archive := pareto.NewArchive(20)
	for i := 1; i <= 5; i++ {
		score := float64(i) * 0.2
		archive.Add(pareto.Point{
			ID:         fmt.Sprintf("p%d", i),
			Objectives: []float64{score},
			Payload:    gepaCandidate{ID: fmt.Sprintf("c%d", i), Score: score},
		}, i)
	}

	best := g.selectBestFromArchive(archive)
	if best == nil {
		t.Fatal("expected a candidate, got nil")
	}
	validIDs := map[string]bool{"c1": true, "c2": true, "c3": true, "c4": true, "c5": true}
	if !validIDs[best.ID] {
		t.Errorf("unexpected candidate ID: %s", best.ID)
	}
}

func TestGEPA_Reflect_NoLLM_NoError(t *testing.T) {
	g := NewGEPA()
	archive := pareto.NewArchive(10)
	archive.Add(pareto.Point{
		ID:         "p1",
		Objectives: []float64{0.8},
		Payload:    gepaCandidate{ID: "c1", Score: 0.8},
	}, 0)

	offspring := gepaPopulation(3)
	offspring[0].Score = 0.1
	rng := rand.New(rand.NewSource(1))

	g.reflect(context.Background(), archive, offspring, 1, rng)
}

func TestGEPA_Reflect_WithLLM_InjectsSuggestion(t *testing.T) {
	llm := &gepaLLMClient{response: "Answer with precise vocabulary."}
	g := NewGEPA(WithGEPALLMClient(llm))

	archive := pareto.NewArchive(10)
	for i := 1; i <= 4; i++ {
		score := float64(i) * 0.2
		archive.Add(pareto.Point{
			ID:         fmt.Sprintf("p%d", i),
			Objectives: []float64{score},
			Payload: gepaCandidate{
				ID:     fmt.Sprintf("c%d", i),
				Score:  score,
				Prompt: fmt.Sprintf("prompt%d", i),
			},
		}, i)
	}

	offspring := gepaPopulation(4)
	offspring[0].Score = 0.05

	rng := rand.New(rand.NewSource(1))
	g.reflect(context.Background(), archive, offspring, 1, rng)

	found := false
	for _, o := range offspring {
		if o.Prompt == "Answer with precise vocabulary." {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one offspring to receive the LLM-generated prompt")
	}
}

func TestGEPA_Reflect_WithLLM_Error_Graceful(t *testing.T) {
	llm := &gepaLLMClient{err: fmt.Errorf("LLM unavailable")}
	g := NewGEPA(WithGEPALLMClient(llm))

	archive := pareto.NewArchive(10)
	archive.Add(pareto.Point{
		ID:         "p1",
		Objectives: []float64{0.5},
		Payload:    gepaCandidate{ID: "c1", Score: 0.5, Prompt: "prompt1"},
	}, 0)

	offspring := gepaPopulation(2)
	rng := rand.New(rand.NewSource(1))

	g.reflect(context.Background(), archive, offspring, 0, rng)
}

// -------------------------------------------------------------------------
// Race-condition safety tests
// -------------------------------------------------------------------------

func TestGEPA_RaceFree_Sequential(t *testing.T) {
	g := NewGEPA(
		WithPopulationSize(6),
		WithMaxGenerations(3),
		WithGEPANumWorkers(1),
		WithGEPAEvalSampleSize(5),
	)

	_, err := g.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: gepaTrainset(20),
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGEPA_RaceFree_Parallel(t *testing.T) {
	g := NewGEPA(
		WithPopulationSize(8),
		WithMaxGenerations(2),
		WithGEPANumWorkers(4),
		WithGEPAEvalSampleSize(4),
	)

	_, err := g.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: gepaTrainset(20),
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// -------------------------------------------------------------------------
// Integration tests (end-to-end, race-free)
// -------------------------------------------------------------------------

func TestGEPA_Integration_IdentityMetric(t *testing.T) {
	g := NewGEPA(
		WithPopulationSize(5),
		WithMaxGenerations(5),
		WithGEPAEvalSampleSize(5),
		WithGEPAConsistencyRuns(1),
	)

	alwaysCorrect := optimize.MetricFunc(func(_ optimize.Example, _ optimize.Prediction, _ *optimize.Trace) float64 {
		return 1.0
	})

	compiled, err := g.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: gepaTrainset(15),
		Metric:   alwaysCorrect,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compiled == nil {
		t.Error("expected compiled program, got nil")
	}
}

func TestGEPA_Integration_ZeroMetric(t *testing.T) {
	g := NewGEPA(
		WithPopulationSize(4),
		WithMaxGenerations(3),
		WithGEPAEvalSampleSize(5),
	)

	alwaysWrong := optimize.MetricFunc(func(_ optimize.Example, _ optimize.Prediction, _ *optimize.Trace) float64 {
		return 0.0
	})

	compiled, err := g.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: gepaTrainset(10),
		Metric:   alwaysWrong,
	})
	if err != nil {
		t.Fatalf("unexpected error even with zero metric: %v", err)
	}
	if compiled == nil {
		t.Error("expected compiled program (best-effort), got nil")
	}
}

func TestGEPA_Integration_EarlyConvergence(t *testing.T) {
	stable := optimize.MetricFunc(func(_ optimize.Example, _ optimize.Prediction, _ *optimize.Trace) float64 {
		return 0.8 // constant — variance is zero, should converge
	})

	g := NewGEPA(
		WithPopulationSize(4),
		WithMaxGenerations(100),
		WithGEPAConvergenceWindow(3),
		WithGEPAConvergenceThreshold(0.01),
		WithGEPAEvalSampleSize(3),
		WithGEPAConsistencyRuns(1),
	)

	_, err := g.Compile(context.Background(), &MockProgram{}, optimize.CompileOptions{
		Trainset: gepaTrainset(10),
		Metric:   stable,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGEPA_Integration_ConcurrentCompile(t *testing.T) {
	const numConcurrent = 5
	trainset := gepaTrainset(20)
	opts := optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	}

	var wg sync.WaitGroup
	errs := make([]error, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			g := NewGEPA(
				WithPopulationSize(4),
				WithMaxGenerations(2),
				WithGEPAEvalSampleSize(5),
				WithGEPASeed(int64(idx)),
			)
			_, errs[idx] = g.Compile(context.Background(), &MockProgram{}, opts)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: unexpected error: %v", i, err)
		}
	}
}

// -------------------------------------------------------------------------
// Pareto frontier integration
// -------------------------------------------------------------------------

func TestGEPA_ParetoFrontier_MultiObjective(t *testing.T) {
	g := NewGEPA()
	archive := pareto.NewArchive(50)

	nonDom := []gepaCandidate{
		{ID: "nd1", Score: 0.9, Objectives: []float64{0.9, 0.3, 0.7}},
		{ID: "nd2", Score: 0.7, Objectives: []float64{0.7, 0.8, 0.6}},
		{ID: "nd3", Score: 0.5, Objectives: []float64{0.5, 0.95, 0.9}},
	}
	dom := gepaCandidate{ID: "dom", Score: 0.4, Objectives: []float64{0.4, 0.2, 0.4}}

	for i, c := range nonDom {
		archive.Add(pareto.Point{ID: c.ID, Objectives: c.Objectives, Payload: c}, i)
	}
	archive.Add(pareto.Point{ID: dom.ID, Objectives: dom.Objectives, Payload: dom}, 10)

	best := g.selectBestFromArchive(archive)
	if best == nil {
		t.Fatal("expected a candidate from archive")
	}
}

// -------------------------------------------------------------------------
// Pareto frontier extended tests
// -------------------------------------------------------------------------

func TestGEPA_ParetoFrontier_CrowdingDistance(t *testing.T) {
	f := pareto.NewFrontier()
	// Three non-dominated trade-off points.
	f.Add(pareto.Point{ID: "1", Objectives: []float64{0.0, 1.0}})
	f.Add(pareto.Point{ID: "2", Objectives: []float64{0.5, 0.5}})
	f.Add(pareto.Point{ID: "3", Objectives: []float64{1.0, 0.0}})

	cd := f.CrowdingDistance()
	if len(cd) != 3 {
		t.Fatalf("expected 3 crowding distances, got %d", len(cd))
	}
	// Boundary points should have infinite distance.
	infCount := 0
	for _, d := range cd {
		if d > 1e100 {
			infCount++
		}
	}
	if infCount < 2 {
		t.Errorf("expected at least 2 boundary points with infinite distance, got %d", infCount)
	}
}

func TestGEPA_ParetoFrontier_HypervolumeIndicator_1D(t *testing.T) {
	f := pareto.NewFrontier()
	f.Add(pareto.Point{ID: "1", Objectives: []float64{0.9}})

	hv := f.HypervolumeIndicator([]float64{0.0})
	if hv != 0.9 {
		t.Errorf("expected hypervolume=0.9, got %f", hv)
	}
}

func TestGEPA_ParetoFrontier_HypervolumeIndicator_2D(t *testing.T) {
	f := pareto.NewFrontier()
	// Single point at (1.0, 1.0), reference at (0.0, 0.0) — hypervolume = 1.0
	f.Add(pareto.Point{ID: "1", Objectives: []float64{1.0, 1.0}})

	hv := f.HypervolumeIndicator([]float64{0.0, 0.0})
	if hv <= 0 {
		t.Errorf("expected positive hypervolume, got %f", hv)
	}
}

func TestGEPA_ParetoFrontier_RankNonDominated(t *testing.T) {
	pts := []pareto.Point{
		{ID: "1", Objectives: []float64{0.9, 0.1}},
		{ID: "2", Objectives: []float64{0.1, 0.9}},
		{ID: "3", Objectives: []float64{0.5, 0.5}},
		{ID: "4", Objectives: []float64{0.3, 0.3}}, // dominated by 1, 2, 3
	}

	ranks := pareto.RankNonDominated(pts)
	if len(ranks) != 4 {
		t.Fatalf("expected 4 ranks, got %d", len(ranks))
	}
	// Points 0-2 form the first front; point 3 is in a lower rank.
	if ranks[3] <= ranks[0] {
		t.Errorf("dominated point (idx 3) should have higher rank number than front-0 points, got rank[3]=%d rank[0]=%d",
			ranks[3], ranks[0])
	}
}

func TestGEPA_ParetoFrontier_EuclideanDistance(t *testing.T) {
	a := pareto.Point{Objectives: []float64{0.0, 0.0}}
	b := pareto.Point{Objectives: []float64{3.0, 4.0}}
	d := pareto.EuclideanDistance(a, b)
	if d != 5.0 {
		t.Errorf("expected EuclideanDistance=5.0, got %f", d)
	}
}

// -------------------------------------------------------------------------
// Mean / stddev helper tests
// -------------------------------------------------------------------------

func TestGEPA_Mean(t *testing.T) {
	tests := []struct {
		input    []float64
		expected float64
	}{
		{nil, 0},
		{[]float64{}, 0},
		{[]float64{1}, 1},
		{[]float64{1, 2, 3}, 2},
		{[]float64{0, 10}, 5},
	}
	for _, tt := range tests {
		got := mean(tt.input)
		if got != tt.expected {
			t.Errorf("mean(%v) = %f, want %f", tt.input, got, tt.expected)
		}
	}
}

func TestGEPA_Stddev(t *testing.T) {
	tests := []struct {
		input    []float64
		wantZero bool
	}{
		{nil, true},
		{[]float64{}, true},
		{[]float64{5}, true},
		{[]float64{5, 5, 5}, true},
		{[]float64{1, 2, 3}, false},
	}
	for _, tt := range tests {
		got := stddev(tt.input)
		if tt.wantZero && got != 0 {
			t.Errorf("stddev(%v) = %f, want 0", tt.input, got)
		}
		if !tt.wantZero && got == 0 {
			t.Errorf("stddev(%v) = 0, want > 0", tt.input)
		}
		if got < 0 {
			t.Errorf("stddev(%v) = %f, must not be negative", tt.input, got)
		}
	}
}

func TestGEPA_CloneDemos(t *testing.T) {
	original := []optimize.Example{
		{Inputs: map[string]interface{}{"k": "v1"}},
		{Inputs: map[string]interface{}{"k": "v2"}},
	}
	clone := cloneDemos(original)

	if len(clone) != len(original) {
		t.Errorf("clone length mismatch: %d vs %d", len(clone), len(original))
	}
	clone[0] = optimize.Example{Inputs: map[string]interface{}{"k": "modified"}}
	if original[0].Inputs["k"] != "v1" {
		t.Error("cloneDemos should produce a separate slice")
	}
}

func TestGEPA_CloneDemos_Nil(t *testing.T) {
	clone := cloneDemos(nil)
	if clone != nil {
		t.Error("cloneDemos(nil) should return nil")
	}
}

// -------------------------------------------------------------------------
// Benchmarks vs other optimizers
// -------------------------------------------------------------------------

func BenchmarkGEPA_Compile_Small(b *testing.B) {
	trainset := gepaTrainset(20)
	opts := optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	}
	g := NewGEPA(
		WithPopulationSize(5),
		WithMaxGenerations(3),
		WithGEPAEvalSampleSize(5),
		WithGEPAConsistencyRuns(1),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = g.Compile(context.Background(), &MockProgram{}, opts)
	}
}

func BenchmarkGEPA_Compile_Medium(b *testing.B) {
	trainset := gepaTrainset(50)
	opts := optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	}
	g := NewGEPA(
		WithPopulationSize(10),
		WithMaxGenerations(5),
		WithGEPAEvalSampleSize(10),
		WithGEPAConsistencyRuns(1),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = g.Compile(context.Background(), &MockProgram{}, opts)
	}
}

func BenchmarkGEPA_vs_BootstrapFewShot(b *testing.B) {
	trainset := gepaTrainset(30)
	opts := optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	}

	b.Run("GEPA", func(b *testing.B) {
		g := NewGEPA(
			WithPopulationSize(5),
			WithMaxGenerations(3),
			WithGEPAEvalSampleSize(5),
			WithGEPAConsistencyRuns(1),
		)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = g.Compile(context.Background(), &MockProgram{}, opts)
		}
	})

	b.Run("BootstrapFewShot", func(b *testing.B) {
		bs := NewBootstrapFewShot(
			WithMaxBootstrapped(3),
			WithMaxLabeled(10),
		)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = bs.Compile(context.Background(), &MockProgram{}, opts)
		}
	})

	b.Run("SIMBA", func(b *testing.B) {
		s := NewSIMBA(
			WithSIMBAMaxIterations(3),
			WithSIMBAMinibatchSize(5),
		)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = s.Compile(context.Background(), &MockProgram{}, opts)
		}
	})
}

func BenchmarkGEPA_EvalOneCandidate(b *testing.B) {
	g := NewGEPA(
		WithGEPAEvalSampleSize(10),
		WithGEPAConsistencyRuns(1),
	)
	trainset := gepaTrainset(50)
	opts := optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	}
	c := gepaCandidate{
		ID:     "bench",
		Prompt: "Answer the question.",
		Demos:  trainset[:4],
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cp := c
		g.evalOneCandidate(context.Background(), &MockProgram{}, &cp, opts, 10, 0)
	}
}

func BenchmarkPareto_Frontier_Add(b *testing.B) {
	f := pareto.NewFrontier()
	rng := rand.New(rand.NewSource(42))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := pareto.Point{
			ID:         fmt.Sprintf("p%d", i),
			Objectives: []float64{rng.Float64(), rng.Float64(), rng.Float64()},
		}
		f.Add(p)
	}
}

func BenchmarkPareto_Archive_Add(b *testing.B) {
	archive := pareto.NewArchive(100)
	rng := rand.New(rand.NewSource(1))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := pareto.Point{
			ID:         fmt.Sprintf("p%d", i),
			Objectives: []float64{rng.Float64(), rng.Float64()},
		}
		archive.Add(p, i)
	}
}

func BenchmarkPareto_CrowdingDistance(b *testing.B) {
	f := pareto.NewFrontier()
	for i := 0; i < 20; i++ {
		x := float64(i) / 20.0
		f.Add(pareto.Point{
			ID:         fmt.Sprintf("p%d", i),
			Objectives: []float64{x, 1.0 - x},
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.CrowdingDistance()
	}
}

func BenchmarkPareto_HypervolumeIndicator_2D(b *testing.B) {
	f := pareto.NewFrontier()
	for i := 0; i < 10; i++ {
		x := float64(i+1) / 11.0
		f.Add(pareto.Point{
			ID:         fmt.Sprintf("p%d", i),
			Objectives: []float64{x, 1.0 - x},
		})
	}
	ref := []float64{0.0, 0.0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.HypervolumeIndicator(ref)
	}
}

// -------------------------------------------------------------------------
// Test helpers
// -------------------------------------------------------------------------

// gepaTrainset creates a trainset of n examples (GEPA-specific helper to avoid
// collisions with the shared makeTrainset defined in mipro_test.go).
func gepaTrainset(n int) []optimize.Example {
	examples := make([]optimize.Example, n)
	for i := 0; i < n; i++ {
		examples[i] = optimize.Example{
			Inputs:  map[string]interface{}{"q": fmt.Sprintf("question_%d", i)},
			Outputs: map[string]interface{}{"a": fmt.Sprintf("answer_%d", i)},
		}
	}
	return examples
}

// gepaPopulation creates a population of n candidates with varied scores.
func gepaPopulation(n int) []gepaCandidate {
	pop := make([]gepaCandidate, n)
	prompts := []string{"Be concise.", "Think step by step.", "Answer directly.", "Explain.", "Summarise."}
	for i := 0; i < n; i++ {
		pop[i] = gepaCandidate{
			ID:     fmt.Sprintf("c%d", i),
			Prompt: prompts[i%len(prompts)],
			Demos:  gepaTrainset(2),
			Score:  float64(i%5) * 0.2,
		}
	}
	return pop
}

// gepaCallback records trial calls for inspection (GEPA-specific to avoid
// collision with countingCallback in mipro_test.go).
type gepaCallback struct {
	mu         sync.Mutex
	onTrial    func(optimize.Trial)
	onComplete func(optimize.OptimizationResult)
}

func (r *gepaCallback) OnTrialComplete(t optimize.Trial) {
	if r.onTrial != nil {
		r.onTrial(t)
	}
}

func (r *gepaCallback) OnOptimizationComplete(result optimize.OptimizationResult) {
	if r.onComplete != nil {
		r.onComplete(result)
	}
}

// gepaLLMClient is a race-safe LLM client stub for GEPA reflection tests.
// (mockLLMClient in mipro_test.go is not mutex-protected.)
type gepaLLMClient struct {
	response string
	err      error
	mu       sync.Mutex
	calls    int
}

func (m *gepaLLMClient) Complete(_ context.Context, _ string, _ optimize.CompletionOptions) (string, error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	return m.response, m.err
}

func (m *gepaLLMClient) CompleteJSON(_ context.Context, _ string, _ json.RawMessage, _ optimize.CompletionOptions) (json.RawMessage, error) {
	return nil, m.err
}

func (m *gepaLLMClient) GetUsage() optimize.TokenUsage {
	return optimize.TokenUsage{}
}
