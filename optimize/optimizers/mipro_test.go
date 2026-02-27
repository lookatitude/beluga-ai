package optimizers

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/optimize"
	"github.com/lookatitude/beluga-ai/optimize/metric"
)

func TestMIPROv2_New(t *testing.T) {
	m := NewMIPROv2(
		WithNumTrials(50),
		WithMinibatchSize(30),
	)

	if m.NumTrials != 50 {
		t.Errorf("expected NumTrials=50, got %d", m.NumTrials)
	}
	if m.MinibatchSize != 30 {
		t.Errorf("expected MinibatchSize=30, got %d", m.MinibatchSize)
	}
}

func TestMIPROv2_Defaults(t *testing.T) {
	m := NewMIPROv2()

	if m.NumTrials != 30 {
		t.Errorf("expected default NumTrials=30, got %d", m.NumTrials)
	}
	if m.MinibatchSize != 25 {
		t.Errorf("expected default MinibatchSize=25, got %d", m.MinibatchSize)
	}
	if m.NumInstructionCandidates != 5 {
		t.Errorf("expected default NumInstructionCandidates=5, got %d", m.NumInstructionCandidates)
	}
	if m.NumDemoCandidates != 5 {
		t.Errorf("expected default NumDemoCandidates=5, got %d", m.NumDemoCandidates)
	}
}

func TestMIPROv2_Registry(t *testing.T) {
	// Test that the optimizer is registered
	optimizers := optimize.ListOptimizers()
	found := false
	for _, name := range optimizers {
		if name == "mipro" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("mipro optimizer not found in registry: %v", optimizers)
	}

	// Test creating via registry
	opt, err := optimize.NewOptimizer("mipro", optimize.OptimizerConfig{})
	if err != nil {
		t.Fatalf("failed to create optimizer from registry: %v", err)
	}
	if opt == nil {
		t.Error("expected optimizer, got nil")
	}
}

func TestMIPROv2_Compile_MissingMetric(t *testing.T) {
	m := NewMIPROv2()
	student := &MockProgram{}

	_, err := m.Compile(context.Background(), student, optimize.CompileOptions{
		Trainset: []optimize.Example{},
		// No metric provided
	})

	if err == nil {
		t.Error("expected error when metric is missing")
	}
}

func TestMIPROv2_Compile_EmptyTrainset(t *testing.T) {
	m := NewMIPROv2()
	student := &MockProgram{}

	_, err := m.Compile(context.Background(), student, optimize.CompileOptions{
		Trainset: []optimize.Example{},
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})

	if err == nil {
		t.Error("expected error when trainset is empty")
	}
}

func TestMIPROv2_Compile_WithTrainset(t *testing.T) {
	m := NewMIPROv2(
		WithNumTrials(5), // Reduce for test speed
		WithMinibatchSize(2),
	)
	student := &MockProgram{}

	trainset := []optimize.Example{
		{
			Inputs:  map[string]interface{}{"question": "What is 2+2?"},
			Outputs: map[string]interface{}{"answer": "4"},
		},
		{
			Inputs:  map[string]interface{}{"question": "What is 3+3?"},
			Outputs: map[string]interface{}{"answer": "6"},
		},
		{
			Inputs:  map[string]interface{}{"question": "What is 4+4?"},
			Outputs: map[string]interface{}{"answer": "8"},
		},
		{
			Inputs:  map[string]interface{}{"question": "What is 5+5?"},
			Outputs: map[string]interface{}{"answer": "10"},
		},
	}

	compiled, err := m.Compile(context.Background(), student, optimize.CompileOptions{
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

func TestMIPROv2_GenerateInstructionCandidates(t *testing.T) {
	m := NewMIPROv2(
		WithNumInstructionCandidates(3),
	)
	student := &MockProgram{}

	trainset := []optimize.Example{
		{Inputs: map[string]interface{}{"q": "1"}},
	}

	candidates, err := m.generateInstructionCandidates(context.Background(), student, optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(candidates) != 3 {
		t.Errorf("expected 3 candidates, got %d", len(candidates))
	}
}

func TestMIPROv2_GenerateDemoCandidates(t *testing.T) {
	m := NewMIPROv2(
		WithNumDemoCandidates(3),
	)
	student := &MockProgram{}

	trainset := []optimize.Example{
		{Inputs: map[string]interface{}{"q": "1"}},
		{Inputs: map[string]interface{}{"q": "2"}},
		{Inputs: map[string]interface{}{"q": "3"}},
		{Inputs: map[string]interface{}{"q": "4"}},
		{Inputs: map[string]interface{}{"q": "5"}},
	}

	candidates, err := m.generateDemoCandidates(context.Background(), student, optimize.CompileOptions{
		Trainset: trainset,
		Metric:   optimize.MetricFunc(metric.ExactMatch),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(candidates) != 3 {
		t.Errorf("expected 3 candidates, got %d", len(candidates))
	}

	// Each candidate should have 4 demos
	for i, c := range candidates {
		if len(c) != 4 {
			t.Errorf("expected candidate %d to have 4 demos, got %d", i, len(c))
		}
	}
}
