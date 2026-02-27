package optimizers

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/optimize"
	"github.com/lookatitude/beluga-ai/optimize/metric"
)

// MockProgram is a mock implementation of optimize.Program for testing.
type MockProgram struct {
	 demos []optimize.Example
}

func (m *MockProgram) Run(ctx context.Context, inputs map[string]interface{}) (optimize.Prediction, error) {
	// Simple mock: return the input as output
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

func TestBootstrapFewShot_New(t *testing.T) {
	config := BootstrapFewShotConfig{
		MaxBootstrapped: 4,
		MaxLabeled:      16,
	}
	
	bs := NewBootstrapFewShot(config)
	
	if bs.MaxBootstrapped != 4 {
		t.Errorf("expected MaxBootstrapped=4, got %d", bs.MaxBootstrapped)
	}
	if bs.MaxLabeled != 16 {
		t.Errorf("expected MaxLabeled=16, got %d", bs.MaxLabeled)
	}
	if bs.Temperature != 1.0 {
		t.Errorf("expected default Temperature=1.0, got %f", bs.Temperature)
	}
}

func TestBootstrapFewShot_Defaults(t *testing.T) {
	bs := NewBootstrapFewShot(BootstrapFewShotConfig{})
	
	if bs.MaxBootstrapped != 4 {
		t.Errorf("expected default MaxBootstrapped=4, got %d", bs.MaxBootstrapped)
	}
	if bs.MaxLabeled != 16 {
		t.Errorf("expected default MaxLabeled=16, got %d", bs.MaxLabeled)
	}
	if bs.MaxRounds != 1 {
		t.Errorf("expected default MaxRounds=1, got %d", bs.MaxRounds)
	}
	if bs.MetricThreshold != 1.0 {
		t.Errorf("expected default MetricThreshold=1.0, got %f", bs.MetricThreshold)
	}
}

func TestBootstrapFewShot_Compile_MissingMetric(t *testing.T) {
	bs := NewBootstrapFewShot(BootstrapFewShotConfig{})
	student := &MockProgram{}
	
	_, err := bs.Compile(context.Background(), student, optimize.CompileOptions{
		Trainset: []optimize.Example{},
		// No metric provided
	})
	
	if err == nil {
		t.Error("expected error when metric is missing")
	}
}

func TestBootstrapFewShot_Compile_EmptyTrainset(t *testing.T) {
	bs := NewBootstrapFewShot(BootstrapFewShotConfig{})
	student := &MockProgram{}
	
	compiled, err := bs.Compile(context.Background(), student, optimize.CompileOptions{
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

func TestBootstrapFewShot_Compile_WithTrainset(t *testing.T) {
	// Create a teacher that returns the expected output
	teacher := &MockProgram{}
	
	config := BootstrapFewShotConfig{
		Teacher:           teacher,
		MaxBootstrapped:   2,
		MaxLabeled:        2,
		MetricThreshold:   0.5, // Lower threshold for testing
	}
	
	bs := NewBootstrapFewShot(config)
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
	}
	
	compiled, err := bs.Compile(context.Background(), student, optimize.CompileOptions{
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

func TestExamplesEqual(t *testing.T) {
	a := optimize.Example{
		Inputs: map[string]interface{}{"key": "value"},
	}
	b := optimize.Example{
		Inputs: map[string]interface{}{"key": "value"},
	}
	c := optimize.Example{
		Inputs: map[string]interface{}{"key": "different"},
	}
	
	if !examplesEqual(a, b) {
		t.Error("expected a and b to be equal")
	}
	
	if examplesEqual(a, c) {
		t.Error("expected a and c to be different")
	}
}

func TestContainsExample(t *testing.T) {
	demos := []optimize.Example{
		{Inputs: map[string]interface{}{"key": "value1"}},
		{Inputs: map[string]interface{}{"key": "value2"}},
	}
	
	existing := optimize.Example{
		Inputs: map[string]interface{}{"key": "value1"},
	}
	newEx := optimize.Example{
		Inputs: map[string]interface{}{"key": "value3"},
	}
	
	if !containsExample(demos, existing) {
		t.Error("expected to find existing example")
	}
	
	if containsExample(demos, newEx) {
		t.Error("expected not to find new example")
	}
}
