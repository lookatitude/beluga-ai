package trajectory

import (
	"context"
	"sync"
	"time"
)

// RunnerHooks provides optional callback functions invoked during trajectory
// evaluation. All fields are optional; nil hooks are skipped.
type RunnerHooks struct {
	// BeforeRun is called before evaluation starts.
	BeforeRun func(ctx context.Context, trajectories []Trajectory) error
	// AfterRun is called after evaluation completes.
	AfterRun func(ctx context.Context, report *Report)
	// BeforeTrajectory is called before evaluating each trajectory.
	BeforeTrajectory func(ctx context.Context, t Trajectory) error
	// AfterTrajectory is called after evaluating each trajectory.
	AfterTrajectory func(ctx context.Context, result TrajectoryResult)
}

// RunnerOption configures a Runner.
type RunnerOption func(*Runner)

// WithMetrics sets the trajectory metrics to evaluate.
func WithMetrics(metrics ...TrajectoryMetric) RunnerOption {
	return func(r *Runner) {
		r.metrics = metrics
	}
}

// WithTrajectories sets the trajectories to evaluate.
func WithTrajectories(trajectories []Trajectory) RunnerOption {
	return func(r *Runner) {
		r.trajectories = trajectories
	}
}

// WithParallel sets the number of trajectories to evaluate concurrently.
// Values less than 1 are ignored.
func WithParallel(n int) RunnerOption {
	return func(r *Runner) {
		if n > 0 {
			r.parallel = n
		}
	}
}

// WithTimeout sets the maximum duration for the evaluation run.
func WithTimeout(d time.Duration) RunnerOption {
	return func(r *Runner) {
		r.timeout = d
	}
}

// WithRunnerHooks sets the lifecycle hooks for the evaluation run.
func WithRunnerHooks(hooks RunnerHooks) RunnerOption {
	return func(r *Runner) {
		r.hooks = hooks
	}
}

// Runner evaluates a set of trajectories against configured metrics with
// bounded concurrency.
type Runner struct {
	metrics      []TrajectoryMetric
	trajectories []Trajectory
	parallel     int
	timeout      time.Duration
	hooks        RunnerHooks
}

// NewRunner creates a new Runner with the given options.
func NewRunner(opts ...RunnerOption) *Runner {
	r := &Runner{
		parallel: 1,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Run executes all configured metrics against all trajectories and returns
// an aggregate report. Trajectories are evaluated with bounded concurrency.
func (r *Runner) Run(ctx context.Context) (*Report, error) {
	if r.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.timeout)
		defer cancel()
	}

	if r.hooks.BeforeRun != nil {
		if err := r.hooks.BeforeRun(ctx, r.trajectories); err != nil {
			return nil, err
		}
	}

	results := make([]TrajectoryResult, len(r.trajectories))
	sem := make(chan struct{}, r.parallel)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i, traj := range r.trajectories {
		if ctx.Err() != nil {
			mu.Lock()
			if firstErr == nil {
				firstErr = ctx.Err()
			}
			mu.Unlock()
			break
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, t Trajectory) {
			defer wg.Done()
			defer func() { <-sem }()

			if r.hooks.BeforeTrajectory != nil {
				if err := r.hooks.BeforeTrajectory(ctx, t); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					return
				}
			}

			result := r.evaluateTrajectory(ctx, t)

			mu.Lock()
			results[idx] = result
			mu.Unlock()

			if r.hooks.AfterTrajectory != nil {
				r.hooks.AfterTrajectory(ctx, result)
			}
		}(i, traj)
	}

	wg.Wait()

	report := r.buildReport(results)

	if r.hooks.AfterRun != nil {
		r.hooks.AfterRun(ctx, report)
	}

	if firstErr != nil && len(r.trajectories) == 0 {
		return report, firstErr
	}

	return report, nil
}

// evaluateTrajectory runs all metrics against a single trajectory.
func (r *Runner) evaluateTrajectory(ctx context.Context, t Trajectory) TrajectoryResult {
	scores := make(map[string]*TrajectoryScore, len(r.metrics))

	for _, m := range r.metrics {
		if ctx.Err() != nil {
			break
		}

		score, err := m.ScoreTrajectory(ctx, t)
		if err != nil {
			// Record a zero score with the error in details.
			scores[m.Name()] = &TrajectoryScore{
				Overall: 0,
				Details: map[string]any{"error": err.Error()},
			}
			continue
		}
		scores[m.Name()] = score
	}

	return TrajectoryResult{
		TrajectoryID: t.ID,
		Scores:       scores,
	}
}

// buildReport aggregates per-trajectory results into a Report.
func (r *Runner) buildReport(results []TrajectoryResult) *Report {
	report := &Report{
		Trajectories: results,
		Aggregate:    make(map[string]float64),
		Timestamp:    time.Now(),
	}

	sums := make(map[string]float64)
	counts := make(map[string]int)

	for _, res := range results {
		for name, score := range res.Scores {
			if score != nil {
				sums[name] += score.Overall
				counts[name]++
			}
		}
	}

	for name, sum := range sums {
		if counts[name] > 0 {
			report.Aggregate[name] = sum / float64(counts[name])
		}
	}

	return report
}
