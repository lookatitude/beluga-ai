package optimizers

import (
	"context"
	"math/rand"
	"testing"

	"github.com/lookatitude/beluga-ai/optimize"
	"github.com/lookatitude/beluga-ai/optimize/metric"
	"github.com/lookatitude/beluga-ai/optimize/pareto"
)

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
}

func TestGEPA_Registry(t *testing.T) {
	// Test that the optimizer is registered
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

	// Test creating via registry
	opt, err := optimize.NewOptimizer("gepa", optimize.OptimizerConfig{})
	if err != nil {
		t.Fatalf("failed to create optimizer from registry: %v", err)
	}
	if opt == nil {
		t.Error("expected optimizer, got nil")
	}
}

func TestGEPA_Compile_MissingMetric(t *testing.T) {
	g := NewGEPA()
	student := &MockProgram{}

	_, err := g.Compile(context.Background(), student, optimize.CompileOptions{
		Trainset: []optimize.Example{},
		// No metric provided
	})

	if err == nil {
		t.Error("expected error when metric is missing")
	}
}

func TestGEPA_Compile_EmptyTrainset(t *testing.T) {
	g := NewGEPA()
	student := &MockProgram{}

	_, err := g.Compile(context.Background(), student, optimize.CompileOptions{
		Trainset: []optimize.Example{},
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})

	if err == nil {
		t.Error("expected error when trainset is empty")
	}
}

func TestGEPA_Compile_WithTrainset(t *testing.T) {
	g := NewGEPA(
		WithMaxGenerations(2), // Reduce for test speed
		WithPopulationSize(5),
	)
	student := &MockProgram{}

	trainset := []optimize.Example{
		{Inputs: map[string]interface{}{"q": "1"}, Outputs: map[string]interface{}{"a": "1"}},
		{Inputs: map[string]interface{}{"q": "2"}, Outputs: map[string]interface{}{"a": "2"}},
		{Inputs: map[string]interface{}{"q": "3"}, Outputs: map[string]interface{}{"a": "3"}},
		{Inputs: map[string]interface{}{"q": "4"}, Outputs: map[string]interface{}{"a": "4"}},
		{Inputs: map[string]interface{}{"q": "5"}, Outputs: map[string]interface{}{"a": "5"}},
	}

	compiled, err := g.Compile(context.Background(), student, optimize.CompileOptions{
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

func TestGEPA_InitializePopulation(t *testing.T) {
	g := NewGEPA(
		WithPopulationSize(5),
	)

	trainset := []optimize.Example{
		{Inputs: map[string]interface{}{"q": "1"}},
		{Inputs: map[string]interface{}{"q": "2"}},
		{Inputs: map[string]interface{}{"q": "3"}},
		{Inputs: map[string]interface{}{"q": "4"}},
		{Inputs: map[string]interface{}{"q": "5"}},
	}

	rng := rand.New(rand.NewSource(42))
	population := g.initializePopulation(&MockProgram{}, trainset, rng)

	if len(population) != 5 {
		t.Errorf("expected 5 candidates, got %d", len(population))
	}

	// Check each candidate has demos
	for i, c := range population {
		if len(c.Demos) == 0 {
			t.Errorf("candidate %d has no demos", i)
		}
		if c.Prompt == "" {
			t.Errorf("candidate %d has no prompt", i)
		}
	}
}

func TestGEPA_TournamentSelect(t *testing.T) {
	g := NewGEPA()

	population := []gepaCandidate{
		{ID: "1", Score: 0.5},
		{ID: "2", Score: 0.9},
		{ID: "3", Score: 0.3},
		{ID: "4", Score: 0.7},
		{ID: "5", Score: 0.8},
	}

	rng := rand.New(rand.NewSource(42))
	// This is probabilistic, so we just check it returns the right size
	parents := g.tournamentSelect(population, 5, rng)
	if len(parents) != 5 {
		t.Errorf("expected 5 parents, got %d", len(parents))
	}
}

func TestGEPA_Crossover(t *testing.T) {
	g := NewGEPA(
		WithPopulationSize(4),
		WithCrossoverRate(1.0), // Force crossover
	)

	parents := []gepaCandidate{
		{ID: "p1", Prompt: "Prompt A", Demos: []optimize.Example{{Inputs: map[string]interface{}{"a": "1"}}}},
		{ID: "p2", Prompt: "Prompt B", Demos: []optimize.Example{{Inputs: map[string]interface{}{"b": "1"}}}},
		{ID: "p3", Prompt: "Prompt C", Demos: []optimize.Example{{Inputs: map[string]interface{}{"c": "1"}}}},
		{ID: "p4", Prompt: "Prompt D", Demos: []optimize.Example{{Inputs: map[string]interface{}{"d": "1"}}}},
	}

	rng := rand.New(rand.NewSource(42))
	offspring := g.crossover(parents, rng)

	if len(offspring) != 4 {
		t.Errorf("expected 4 offspring, got %d", len(offspring))
	}
}

func TestGEPA_Mutate(t *testing.T) {
	g := NewGEPA(
		WithMutationRate(1.0), // Force mutation
	)

	trainset := []optimize.Example{
		{Inputs: map[string]interface{}{"q": "new"}},
	}

	offspring := []gepaCandidate{
		{ID: "c1", Prompt: "Original prompt", Demos: []optimize.Example{{Inputs: map[string]interface{}{"q": "old"}}}},
	}

	originalPrompt := offspring[0].Prompt
	originalDemo := offspring[0].Demos[0].Inputs["q"]

	rng := rand.New(rand.NewSource(42))
	g.mutate(offspring, trainset, rng)

	// Check something changed
	if offspring[0].Prompt == originalPrompt {
		t.Error("expected prompt to be mutated")
	}
	if offspring[0].Demos[0].Inputs["q"] == originalDemo {
		t.Error("expected demo to be mutated")
	}
}

func TestGEPA_SelectBestFromArchive(t *testing.T) {
	g := NewGEPA()

	// Test empty archive
	archive := pareto.NewArchive(10)
	best := g.selectBestFromArchive(archive)
	if best != nil {
		t.Error("expected nil for empty archive")
	}

	// Test with points
	archive.Add(pareto.Point{
		ID:         "1",
		Objectives: []float64{0.5},
		Payload:    gepaCandidate{ID: "c1", Score: 0.5},
	}, 1)
	archive.Add(pareto.Point{
		ID:         "2",
		Objectives: []float64{0.9},
		Payload:    gepaCandidate{ID: "c2", Score: 0.9},
	}, 1)
	archive.Add(pareto.Point{
		ID:         "3",
		Objectives: []float64{0.7},
		Payload:    gepaCandidate{ID: "c3", Score: 0.7},
	}, 1)

	best = g.selectBestFromArchive(archive)
	if best == nil {
		t.Error("expected best candidate, got nil")
	}
	// Should return one of the candidates
	if best.ID != "c1" && best.ID != "c2" && best.ID != "c3" {
		t.Errorf("unexpected candidate ID: %s", best.ID)
	}
}
