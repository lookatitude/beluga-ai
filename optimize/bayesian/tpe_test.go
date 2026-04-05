package bayesian

import (
	"math"
	"testing"
)

func TestNewTPE(t *testing.T) {
	tpe := NewTPE()

	if tpe.Gamma != 0.25 {
		t.Errorf("expected default Gamma=0.25, got %f", tpe.Gamma)
	}
	if tpe.Rand == nil {
		t.Error("expected Rand to be initialized")
	}
}

func TestTPE_WithGamma(t *testing.T) {
	tpe := NewTPE().WithGamma(0.5)

	if tpe.Gamma != 0.5 {
		t.Errorf("expected Gamma=0.5, got %f", tpe.Gamma)
	}
}

func TestTPE_WithSeed(t *testing.T) {
	tpe := NewTPE().WithSeed(123)

	// Sample two values with the same seed
	val1 := tpe.Rand.Intn(100)

	tpe2 := NewTPE().WithSeed(123)
	val2 := tpe2.Rand.Intn(100)

	if val1 != val2 {
		t.Errorf("expected same values with same seed, got %d and %d", val1, val2)
	}
}

func TestTPE_SampleInt_NoTrials(t *testing.T) {
	tpe := NewTPE().WithSeed(42)

	// With no trials, should sample uniformly
	val := tpe.SampleInt("test", 0, 10, nil)

	if val < 0 || val > 10 {
		t.Errorf("expected value in [0, 10], got %d", val)
	}
}

func TestTPE_SampleInt_WithTrials(t *testing.T) {
	tpe := NewTPE().WithSeed(42)

	// Create some trials
	trials := []Trial{
		{Params: map[string]interface{}{"value": 5}, Score: 0.9},
		{Params: map[string]interface{}{"value": 3}, Score: 0.8},
		{Params: map[string]interface{}{"value": 7}, Score: 0.7},
		{Params: map[string]interface{}{"value": 1}, Score: 0.6},
		{Params: map[string]interface{}{"value": 9}, Score: 0.5},
	}

	val := tpe.SampleInt("test", 0, 10, trials)

	if val < 0 || val > 10 {
		t.Errorf("expected value in [0, 10], got %d", val)
	}
}

func TestTPE_SampleFloat_NoTrials(t *testing.T) {
	tpe := NewTPE().WithSeed(42)

	// With no trials, should sample uniformly
	val := tpe.SampleFloat("test", 0.0, 1.0, false, nil)

	if val < 0.0 || val > 1.0 {
		t.Errorf("expected value in [0.0, 1.0], got %f", val)
	}
}

func TestTPE_SampleFloat_LogScale(t *testing.T) {
	tpe := NewTPE().WithSeed(42)

	// Test log scale sampling
	val := tpe.SampleFloat("test", 0.001, 1.0, true, nil)

	if val < 0.001 || val > 1.0 {
		t.Errorf("expected value in [0.001, 1.0], got %f", val)
	}
}

func TestTPE_SampleCategorical(t *testing.T) {
	tpe := NewTPE().WithSeed(42)

	choices := CategoricalChoice{
		Options: []string{"a", "b", "c"},
	}

	val := tpe.SampleCategorical("test", choices, nil)

	found := false
	for _, opt := range choices.Options {
		if val == opt {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected value in %v, got %s", choices.Options, val)
	}
}

func TestTPE_SampleCategorical_Empty(t *testing.T) {
	tpe := NewTPE().WithSeed(42)

	choices := CategoricalChoice{
		Options: []string{},
	}

	val := tpe.SampleCategorical("test", choices, nil)

	if val != "" {
		t.Errorf("expected empty string for empty options, got %s", val)
	}
}

func TestTPE_SplitTrials(t *testing.T) {
	tpe := NewTPE().WithGamma(0.4) // Top 40%

	trials := []Trial{
		{Params: map[string]interface{}{}, Score: 0.9},
		{Params: map[string]interface{}{}, Score: 0.8},
		{Params: map[string]interface{}{}, Score: 0.7},
		{Params: map[string]interface{}{}, Score: 0.6},
		{Params: map[string]interface{}{}, Score: 0.5},
	}

	good, bad := tpe.splitTrials(trials, "test")

	// With 5 trials and gamma=0.4, should have 2 good trials
	if len(good) != 2 {
		t.Errorf("expected 2 good trials, got %d", len(good))
	}
	if len(bad) != 3 {
		t.Errorf("expected 3 bad trials, got %d", len(bad))
	}

	// Good trials should be the top scores
	if good[0].Score != 0.9 {
		t.Errorf("expected top score 0.9, got %f", good[0].Score)
	}
	if good[1].Score != 0.8 {
		t.Errorf("expected second score 0.8, got %f", good[1].Score)
	}
}

func TestGaussianPDF(t *testing.T) {
	// Test at mean
	pdf := gaussianPDF(0.0, 0.0, 1.0)
	expected := 1.0 / math.Sqrt(2*math.Pi)
	if math.Abs(pdf-expected) > 0.0001 {
		t.Errorf("expected PDF at mean to be %f, got %f", expected, pdf)
	}

	// Test at 1 std
	pdf = gaussianPDF(1.0, 0.0, 1.0)
	expected = math.Exp(-0.5) / math.Sqrt(2*math.Pi)
	if math.Abs(pdf-expected) > 0.0001 {
		t.Errorf("expected PDF at 1 std to be %f, got %f", expected, pdf)
	}

	// Test with zero std
	pdf = gaussianPDF(5.0, 5.0, 0.0)
	if pdf != 1.0 {
		t.Errorf("expected PDF at mean with zero std to be 1.0, got %f", pdf)
	}

	pdf = gaussianPDF(5.0, 6.0, 0.0)
	if pdf != 0.0 {
		t.Errorf("expected PDF away from mean with zero std to be 0.0, got %f", pdf)
	}
}
