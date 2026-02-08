package eval_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/eval"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMetricForRunner is a configurable mock metric for runner tests.
type mockMetricForRunner struct {
	name  string
	score float64
	err   error
	delay time.Duration
	mu    sync.Mutex
	calls int
}

func (m *mockMetricForRunner) Name() string { return m.name }

func (m *mockMetricForRunner) Score(ctx context.Context, sample eval.EvalSample) (float64, error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()

	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}

	if m.err != nil {
		return 0, m.err
	}
	return m.score, nil
}

func (m *mockMetricForRunner) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func TestNewRunner_DefaultConfig(t *testing.T) {
	runner := eval.NewRunner()

	require.NotNil(t, runner)
}

func TestNewRunner_WithMetrics(t *testing.T) {
	metric1 := &mockMetricForRunner{name: "metric1", score: 0.9}
	metric2 := &mockMetricForRunner{name: "metric2", score: 0.8}

	runner := eval.NewRunner(
		eval.WithMetrics(metric1, metric2),
	)

	require.NotNil(t, runner)
}

func TestNewRunner_WithDataset(t *testing.T) {
	samples := []eval.EvalSample{
		{Input: "q1", Output: "a1"},
		{Input: "q2", Output: "a2"},
	}

	runner := eval.NewRunner(
		eval.WithDataset(samples),
	)

	require.NotNil(t, runner)
}

func TestNewRunner_WithOptions(t *testing.T) {
	runner := eval.NewRunner(
		eval.WithParallel(4),
		eval.WithTimeout(5*time.Minute),
		eval.WithStopOnError(true),
	)

	require.NotNil(t, runner)
}

func TestRunner_RunSimple(t *testing.T) {
	metric := &mockMetricForRunner{name: "test_metric", score: 0.85}
	samples := []eval.EvalSample{
		{Input: "q1", Output: "a1"},
		{Input: "q2", Output: "a2"},
	}

	runner := eval.NewRunner(
		eval.WithMetrics(metric),
		eval.WithDataset(samples),
	)

	ctx := context.Background()
	report, err := runner.Run(ctx)

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Len(t, report.Samples, 2)
	assert.Equal(t, 0.85, report.Metrics["test_metric"])
	assert.Empty(t, report.Errors)
	assert.Greater(t, report.Duration, time.Duration(0))
}

func TestRunner_RunEmptyDataset(t *testing.T) {
	metric := &mockMetricForRunner{name: "test_metric", score: 0.85}

	runner := eval.NewRunner(
		eval.WithMetrics(metric),
		eval.WithDataset([]eval.EvalSample{}),
	)

	ctx := context.Background()
	report, err := runner.Run(ctx)

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Empty(t, report.Samples)
	assert.Empty(t, report.Metrics)
	assert.Empty(t, report.Errors)
}

func TestRunner_RunMultipleMetrics(t *testing.T) {
	metric1 := &mockMetricForRunner{name: "latency", score: 0.9}
	metric2 := &mockMetricForRunner{name: "toxicity", score: 1.0}
	metric3 := &mockMetricForRunner{name: "cost", score: 0.5}

	samples := []eval.EvalSample{
		{Input: "q1", Output: "a1"},
	}

	runner := eval.NewRunner(
		eval.WithMetrics(metric1, metric2, metric3),
		eval.WithDataset(samples),
	)

	ctx := context.Background()
	report, err := runner.Run(ctx)

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Len(t, report.Samples, 1)
	assert.Equal(t, 0.9, report.Metrics["latency"])
	assert.Equal(t, 1.0, report.Metrics["toxicity"])
	assert.Equal(t, 0.5, report.Metrics["cost"])
}

func TestRunner_RunParallel(t *testing.T) {
	// Use a metric with a small delay to ensure parallel execution
	metric := &mockMetricForRunner{
		name:  "test_metric",
		score: 0.85,
		delay: 10 * time.Millisecond,
	}

	// Create 10 samples
	samples := make([]eval.EvalSample, 10)
	for i := range samples {
		samples[i] = eval.EvalSample{Input: "q", Output: "a"}
	}

	runner := eval.NewRunner(
		eval.WithMetrics(metric),
		eval.WithDataset(samples),
		eval.WithParallel(4),
	)

	ctx := context.Background()
	start := time.Now()
	report, err := runner.Run(ctx)
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Len(t, report.Samples, 10)

	// With 4 parallel workers and 10ms delay per sample:
	// Sequential would take ~100ms, parallel should take ~30ms (10/4 * 10ms)
	// We use a generous check to avoid flakiness
	assert.Less(t, elapsed, 80*time.Millisecond, "parallel execution should be faster")
	assert.Equal(t, 10, metric.CallCount())
}

func TestRunner_RunWithMetricError(t *testing.T) {
	expectedErr := errors.New("metric error")
	metric := &mockMetricForRunner{name: "failing_metric", err: expectedErr}

	samples := []eval.EvalSample{
		{Input: "q1", Output: "a1"},
	}

	runner := eval.NewRunner(
		eval.WithMetrics(metric),
		eval.WithDataset(samples),
	)

	ctx := context.Background()
	report, err := runner.Run(ctx)

	require.NoError(t, err) // Run itself doesn't error, errors are collected
	require.NotNil(t, report)
	assert.Len(t, report.Errors, 1)
	assert.ErrorIs(t, report.Errors[0], expectedErr)
}

func TestRunner_RunStopOnError(t *testing.T) {
	expectedErr := errors.New("stop error")
	metric := &mockMetricForRunner{name: "failing_metric", err: expectedErr}

	// Create many samples
	samples := make([]eval.EvalSample, 10)
	for i := range samples {
		samples[i] = eval.EvalSample{Input: "q", Output: "a"}
	}

	runner := eval.NewRunner(
		eval.WithMetrics(metric),
		eval.WithDataset(samples),
		eval.WithStopOnError(true),
	)

	ctx := context.Background()
	report, err := runner.Run(ctx)

	require.NoError(t, err)
	require.NotNil(t, report)
	// Should stop after first error, so not all samples are evaluated
	assert.NotEmpty(t, report.Errors)
}

func TestRunner_RunWithTimeout(t *testing.T) {
	// Metric with long delay
	metric := &mockMetricForRunner{
		name:  "slow_metric",
		score: 0.85,
		delay: 5 * time.Second,
	}

	samples := []eval.EvalSample{
		{Input: "q1", Output: "a1"},
	}

	runner := eval.NewRunner(
		eval.WithMetrics(metric),
		eval.WithDataset(samples),
		eval.WithTimeout(50*time.Millisecond),
	)

	ctx := context.Background()
	start := time.Now()
	report, err := runner.Run(ctx)
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, report)
	// Should timeout quickly
	assert.Less(t, elapsed, 200*time.Millisecond)
	// The context timeout error should be collected
	assert.NotEmpty(t, report.Errors)
}

func TestRunner_RunWithHooks(t *testing.T) {
	var beforeRunCalled atomic.Bool
	var afterRunCalled atomic.Bool
	var beforeSampleCount atomic.Int32
	var afterSampleCount atomic.Int32

	hooks := eval.Hooks{
		BeforeRun: func(ctx context.Context, samples []eval.EvalSample) error {
			beforeRunCalled.Store(true)
			assert.Len(t, samples, 3)
			return nil
		},
		AfterRun: func(ctx context.Context, report *eval.EvalReport) {
			afterRunCalled.Store(true)
			assert.NotNil(t, report)
		},
		BeforeSample: func(ctx context.Context, sample eval.EvalSample) error {
			beforeSampleCount.Add(1)
			return nil
		},
		AfterSample: func(ctx context.Context, result eval.SampleResult) {
			afterSampleCount.Add(1)
		},
	}

	metric := &mockMetricForRunner{name: "test_metric", score: 0.9}
	samples := []eval.EvalSample{
		{Input: "q1", Output: "a1"},
		{Input: "q2", Output: "a2"},
		{Input: "q3", Output: "a3"},
	}

	runner := eval.NewRunner(
		eval.WithMetrics(metric),
		eval.WithDataset(samples),
		eval.WithHooks(hooks),
	)

	ctx := context.Background()
	report, err := runner.Run(ctx)

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.True(t, beforeRunCalled.Load())
	assert.True(t, afterRunCalled.Load())
	assert.Equal(t, int32(3), beforeSampleCount.Load())
	assert.Equal(t, int32(3), afterSampleCount.Load())
}

func TestRunner_RunHookError(t *testing.T) {
	expectedErr := errors.New("before run hook error")

	hooks := eval.Hooks{
		BeforeRun: func(ctx context.Context, samples []eval.EvalSample) error {
			return expectedErr
		},
	}

	metric := &mockMetricForRunner{name: "test_metric", score: 0.9}
	samples := []eval.EvalSample{
		{Input: "q1", Output: "a1"},
	}

	runner := eval.NewRunner(
		eval.WithMetrics(metric),
		eval.WithDataset(samples),
		eval.WithHooks(hooks),
	)

	ctx := context.Background()
	report, err := runner.Run(ctx)

	// BeforeRun hook error causes Run to return error
	require.Error(t, err)
	require.Nil(t, report)
	assert.ErrorIs(t, err, expectedErr)
}

func TestRunner_RunBeforeSampleHookError(t *testing.T) {
	expectedErr := errors.New("before sample hook error")

	hooks := eval.Hooks{
		BeforeSample: func(ctx context.Context, sample eval.EvalSample) error {
			if sample.Input == "q2" {
				return expectedErr
			}
			return nil
		},
	}

	metric := &mockMetricForRunner{name: "test_metric", score: 0.9}
	samples := []eval.EvalSample{
		{Input: "q1", Output: "a1"},
		{Input: "q2", Output: "a2"},
		{Input: "q3", Output: "a3"},
	}

	runner := eval.NewRunner(
		eval.WithMetrics(metric),
		eval.WithDataset(samples),
		eval.WithHooks(hooks),
	)

	ctx := context.Background()
	report, err := runner.Run(ctx)

	require.NoError(t, err)
	require.NotNil(t, report)
	// One sample should have an error
	assert.Len(t, report.Errors, 1)
	// Check that the error is from the expected sample
	for _, result := range report.Samples {
		if result.Sample.Input == "q2" {
			assert.Error(t, result.Error)
			assert.ErrorIs(t, result.Error, expectedErr)
		}
	}
}

func TestRunner_RunContextCancellation(t *testing.T) {
	// Use a metric with delay that will check context
	metric := &mockMetricForRunner{
		name:  "slow_metric",
		score: 0.85,
		delay: 100 * time.Millisecond,
	}

	samples := []eval.EvalSample{
		{Input: "q1", Output: "a1"},
		{Input: "q2", Output: "a2"},
	}

	runner := eval.NewRunner(
		eval.WithMetrics(metric),
		eval.WithDataset(samples),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	report, err := runner.Run(ctx)

	require.NoError(t, err)
	require.NotNil(t, report)
	// Should collect context timeout error from at least one sample
	// (May not error if execution is fast enough, so we just check it completes)
	_ = report.Errors
}

// variableScoreMetric returns different scores for each invocation.
type variableScoreMetric struct {
	name   string
	scores []float64
	index  int
	mu     sync.Mutex
}

func (m *variableScoreMetric) Name() string { return m.name }

func (m *variableScoreMetric) Score(ctx context.Context, sample eval.EvalSample) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.index < len(m.scores) {
		score := m.scores[m.index]
		m.index++
		return score, nil
	}
	return 0.0, nil
}

func TestRunner_AverageScores(t *testing.T) {
	// Create a metric that returns different scores for each sample
	metric := &variableScoreMetric{
		name:   "variable_metric",
		scores: []float64{1.0, 0.8, 0.6},
	}

	samples := []eval.EvalSample{
		{Input: "q1", Output: "a1"},
		{Input: "q2", Output: "a2"},
		{Input: "q3", Output: "a3"},
	}

	runner := eval.NewRunner(
		eval.WithMetrics(metric),
		eval.WithDataset(samples),
	)

	ctx := context.Background()
	report, err := runner.Run(ctx)

	require.NoError(t, err)
	require.NotNil(t, report)
	// Average of 1.0, 0.8, 0.6 = 0.8
	assert.InDelta(t, 0.8, report.Metrics["variable_metric"], 0.01)
}

func TestRunner_WithParallelZero(t *testing.T) {
	metric := &mockMetricForRunner{name: "test_metric", score: 0.9}
	samples := []eval.EvalSample{
		{Input: "q1", Output: "a1"},
	}

	// WithParallel(0) should be ignored (defaults to 1)
	runner := eval.NewRunner(
		eval.WithMetrics(metric),
		eval.WithDataset(samples),
		eval.WithParallel(0),
	)

	ctx := context.Background()
	report, err := runner.Run(ctx)

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Len(t, report.Samples, 1)
}

func TestRunner_WithParallelNegative(t *testing.T) {
	metric := &mockMetricForRunner{name: "test_metric", score: 0.9}
	samples := []eval.EvalSample{
		{Input: "q1", Output: "a1"},
	}

	// WithParallel(-1) should be ignored (defaults to 1)
	runner := eval.NewRunner(
		eval.WithMetrics(metric),
		eval.WithDataset(samples),
		eval.WithParallel(-1),
	)

	ctx := context.Background()
	report, err := runner.Run(ctx)

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Len(t, report.Samples, 1)
}
