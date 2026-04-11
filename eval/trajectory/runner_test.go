package trajectory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// staticMetric is a test metric that returns a fixed score.
type staticMetric struct {
	name  string
	score float64
	err   error
}

var _ TrajectoryMetric = (*staticMetric)(nil)

func (m *staticMetric) Name() string { return m.name }
func (m *staticMetric) ScoreTrajectory(_ context.Context, _ Trajectory) (*TrajectoryScore, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &TrajectoryScore{Overall: m.score}, nil
}

func TestRunner_BasicRun(t *testing.T) {
	trajectories := []Trajectory{
		{ID: "t1", Input: "hello", Output: "world"},
		{ID: "t2", Input: "foo", Output: "bar"},
	}

	runner := NewRunner(
		WithMetrics(&staticMetric{name: "m1", score: 0.8}, &staticMetric{name: "m2", score: 0.6}),
		WithTrajectories(trajectories),
	)

	report, err := runner.Run(context.Background())
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Len(t, report.Trajectories, 2)
	assert.Equal(t, 0.8, report.Aggregate["m1"])
	assert.Equal(t, 0.6, report.Aggregate["m2"])

	// Per-trajectory results.
	assert.Equal(t, "t1", report.Trajectories[0].TrajectoryID)
	assert.Equal(t, 0.8, report.Trajectories[0].Scores["m1"].Overall)
}

func TestRunner_EmptyTrajectories(t *testing.T) {
	runner := NewRunner(
		WithMetrics(&staticMetric{name: "m1", score: 1.0}),
		WithTrajectories(nil),
	)

	report, err := runner.Run(context.Background())
	require.NoError(t, err)
	assert.Empty(t, report.Trajectories)
	assert.Empty(t, report.Aggregate)
}

func TestRunner_MetricError(t *testing.T) {
	runner := NewRunner(
		WithMetrics(&staticMetric{name: "m1", err: fmt.Errorf("metric failed")}),
		WithTrajectories([]Trajectory{{ID: "t1"}}),
	)

	report, err := runner.Run(context.Background())
	require.NoError(t, err)

	// Error should be recorded in details.
	score := report.Trajectories[0].Scores["m1"]
	assert.Equal(t, 0.0, score.Overall)
	assert.Equal(t, "metric failed", score.Details["error"])
}

func TestRunner_Parallel(t *testing.T) {
	trajectories := make([]Trajectory, 10)
	for i := range trajectories {
		trajectories[i] = Trajectory{ID: fmt.Sprintf("t%d", i)}
	}

	runner := NewRunner(
		WithMetrics(&staticMetric{name: "m1", score: 0.5}),
		WithTrajectories(trajectories),
		WithParallel(4),
	)

	report, err := runner.Run(context.Background())
	require.NoError(t, err)
	assert.Len(t, report.Trajectories, 10)
	assert.Equal(t, 0.5, report.Aggregate["m1"])
}

func TestRunner_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	runner := NewRunner(
		WithMetrics(&staticMetric{name: "m1", score: 1.0}),
		WithTrajectories([]Trajectory{{ID: "t1"}, {ID: "t2"}}),
	)

	report, err := runner.Run(ctx)
	// Should surface the cancellation as an error and still return the report.
	require.ErrorIs(t, err, context.Canceled)
	require.NotNil(t, report)
}

func TestRunner_Timeout(t *testing.T) {
	runner := NewRunner(
		WithMetrics(&staticMetric{name: "m1", score: 1.0}),
		WithTrajectories([]Trajectory{{ID: "t1"}}),
		WithTimeout(5*time.Second),
	)

	report, err := runner.Run(context.Background())
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Len(t, report.Trajectories, 1)
}

func TestRunner_Hooks(t *testing.T) {
	var beforeCalled, afterCalled bool
	var beforeTrajCalled, afterTrajCalled int

	runner := NewRunner(
		WithMetrics(&staticMetric{name: "m1", score: 1.0}),
		WithTrajectories([]Trajectory{{ID: "t1"}, {ID: "t2"}}),
		WithRunnerHooks(RunnerHooks{
			BeforeRun: func(_ context.Context, trajs []Trajectory) error {
				beforeCalled = true
				assert.Len(t, trajs, 2)
				return nil
			},
			AfterRun: func(_ context.Context, report *Report) {
				afterCalled = true
				assert.NotNil(t, report)
			},
			BeforeTrajectory: func(_ context.Context, traj Trajectory) error {
				beforeTrajCalled++
				return nil
			},
			AfterTrajectory: func(_ context.Context, result TrajectoryResult) {
				afterTrajCalled++
			},
		}),
	)

	_, err := runner.Run(context.Background())
	require.NoError(t, err)

	assert.True(t, beforeCalled)
	assert.True(t, afterCalled)
	assert.Equal(t, 2, beforeTrajCalled)
	assert.Equal(t, 2, afterTrajCalled)
}

func TestRunner_BeforeRunError(t *testing.T) {
	runner := NewRunner(
		WithMetrics(&staticMetric{name: "m1", score: 1.0}),
		WithTrajectories([]Trajectory{{ID: "t1"}}),
		WithRunnerHooks(RunnerHooks{
			BeforeRun: func(_ context.Context, _ []Trajectory) error {
				return fmt.Errorf("hook error")
			},
		}),
	)

	_, err := runner.Run(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hook error")
}

func TestRunnerOption_InvalidParallel(t *testing.T) {
	runner := NewRunner(WithParallel(-1))
	assert.Equal(t, 1, runner.parallel) // Should keep default.

	runner = NewRunner(WithParallel(0))
	assert.Equal(t, 1, runner.parallel) // Should keep default.
}
