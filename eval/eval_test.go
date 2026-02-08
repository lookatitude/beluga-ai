package eval_test

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/eval"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
)

// Verify interface compliance.
var (
	_ eval.Metric = (*mockMetric)(nil)
)

// mockMetric is a simple mock for testing.
type mockMetric struct {
	name  string
	score float64
	err   error
}

func (m *mockMetric) Name() string { return m.name }

func (m *mockMetric) Score(_ context.Context, _ eval.EvalSample) (float64, error) {
	return m.score, m.err
}

func TestEvalSample_Creation(t *testing.T) {
	sample := eval.EvalSample{
		Input:          "What is AI?",
		Output:         "AI stands for Artificial Intelligence.",
		ExpectedOutput: "Artificial Intelligence is...",
		RetrievedDocs: []schema.Document{
			{ID: "doc1", Content: "AI context"},
		},
		Metadata: map[string]any{
			"latency_ms": 150.0,
			"model":      "gpt-4",
		},
	}

	assert.Equal(t, "What is AI?", sample.Input)
	assert.Equal(t, "AI stands for Artificial Intelligence.", sample.Output)
	assert.Equal(t, "Artificial Intelligence is...", sample.ExpectedOutput)
	assert.Len(t, sample.RetrievedDocs, 1)
	assert.Equal(t, "doc1", sample.RetrievedDocs[0].ID)
	assert.Equal(t, 150.0, sample.Metadata["latency_ms"])
}

func TestSampleResult_Creation(t *testing.T) {
	sample := eval.EvalSample{
		Input:  "test input",
		Output: "test output",
	}

	result := eval.SampleResult{
		Sample: sample,
		Scores: map[string]float64{
			"latency":  0.85,
			"toxicity": 1.0,
		},
		Error: nil,
	}

	assert.Equal(t, sample, result.Sample)
	assert.Len(t, result.Scores, 2)
	assert.Equal(t, 0.85, result.Scores["latency"])
	assert.Equal(t, 1.0, result.Scores["toxicity"])
	assert.NoError(t, result.Error)
}

func TestEvalReport_Structure(t *testing.T) {
	samples := []eval.SampleResult{
		{
			Sample: eval.EvalSample{Input: "q1", Output: "a1"},
			Scores: map[string]float64{"latency": 0.9},
		},
		{
			Sample: eval.EvalSample{Input: "q2", Output: "a2"},
			Scores: map[string]float64{"latency": 0.8},
		},
	}

	report := eval.EvalReport{
		Samples: samples,
		Metrics: map[string]float64{
			"latency": 0.85,
		},
		Duration: 2 * time.Second,
		Errors:   nil,
	}

	assert.Len(t, report.Samples, 2)
	assert.Equal(t, 0.85, report.Metrics["latency"])
	assert.Equal(t, 2*time.Second, report.Duration)
	assert.Empty(t, report.Errors)
}

func TestEvalReport_WithErrors(t *testing.T) {
	report := eval.EvalReport{
		Samples: []eval.SampleResult{},
		Metrics: map[string]float64{},
		Duration: 1 * time.Second,
		Errors: []error{
			assert.AnError,
		},
	}

	assert.Len(t, report.Errors, 1)
	assert.Error(t, report.Errors[0])
}

func TestConfig_Defaults(t *testing.T) {
	cfg := eval.Config{}

	// Zero values should be the defaults
	assert.Equal(t, 0, cfg.Parallel)
	assert.Equal(t, time.Duration(0), cfg.Timeout)
	assert.False(t, cfg.StopOnError)
}

func TestConfig_CustomValues(t *testing.T) {
	cfg := eval.Config{
		Parallel:    4,
		Timeout:     5 * time.Minute,
		StopOnError: true,
	}

	assert.Equal(t, 4, cfg.Parallel)
	assert.Equal(t, 5*time.Minute, cfg.Timeout)
	assert.True(t, cfg.StopOnError)
}

func TestHooks_AllOptional(t *testing.T) {
	// All hooks are optional and can be nil
	hooks := eval.Hooks{
		BeforeRun:    nil,
		AfterRun:     nil,
		BeforeSample: nil,
		AfterSample:  nil,
	}

	// Should not panic with nil hooks
	assert.Nil(t, hooks.BeforeRun)
	assert.Nil(t, hooks.AfterRun)
	assert.Nil(t, hooks.BeforeSample)
	assert.Nil(t, hooks.AfterSample)
}

func TestHooks_WithCallbacks(t *testing.T) {
	var beforeRunCalled bool
	var afterRunCalled bool
	var beforeSampleCalled bool
	var afterSampleCalled bool

	hooks := eval.Hooks{
		BeforeRun: func(_ context.Context, _ []eval.EvalSample) error {
			beforeRunCalled = true
			return nil
		},
		AfterRun: func(_ context.Context, _ *eval.EvalReport) {
			afterRunCalled = true
		},
		BeforeSample: func(_ context.Context, _ eval.EvalSample) error {
			beforeSampleCalled = true
			return nil
		},
		AfterSample: func(_ context.Context, _ eval.SampleResult) {
			afterSampleCalled = true
		},
	}

	// Verify hooks are set
	assert.NotNil(t, hooks.BeforeRun)
	assert.NotNil(t, hooks.AfterRun)
	assert.NotNil(t, hooks.BeforeSample)
	assert.NotNil(t, hooks.AfterSample)

	// Call them to verify they work
	_ = hooks.BeforeRun(nil, nil)
	hooks.AfterRun(nil, nil)
	_ = hooks.BeforeSample(nil, eval.EvalSample{})
	hooks.AfterSample(nil, eval.SampleResult{})

	assert.True(t, beforeRunCalled)
	assert.True(t, afterRunCalled)
	assert.True(t, beforeSampleCalled)
	assert.True(t, afterSampleCalled)
}
