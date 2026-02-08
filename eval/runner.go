package eval

import (
	"context"
	"sync"
	"time"
)

// Hooks provides optional callback functions invoked during evaluation.
// All fields are optional; nil hooks are skipped.
type Hooks struct {
	// BeforeRun is called before the evaluation starts.
	BeforeRun func(ctx context.Context, samples []EvalSample) error
	// AfterRun is called after the evaluation completes.
	AfterRun func(ctx context.Context, report *EvalReport)
	// BeforeSample is called before each sample is evaluated.
	BeforeSample func(ctx context.Context, sample EvalSample) error
	// AfterSample is called after each sample is evaluated.
	AfterSample func(ctx context.Context, result SampleResult)
}

// Config holds the configuration for an EvalRunner.
type Config struct {
	// Parallel is the number of samples to evaluate concurrently.
	// Defaults to 1 (sequential).
	Parallel int
	// Timeout is the maximum duration for the entire evaluation run.
	// Zero means no timeout.
	Timeout time.Duration
	// StopOnError stops evaluation on the first metric error when true.
	StopOnError bool
}

// RunnerOption configures an EvalRunner.
type RunnerOption func(*EvalRunner)

// WithMetrics sets the metrics to evaluate.
func WithMetrics(metrics ...Metric) RunnerOption {
	return func(r *EvalRunner) {
		r.metrics = metrics
	}
}

// WithDataset sets the evaluation samples.
func WithDataset(samples []EvalSample) RunnerOption {
	return func(r *EvalRunner) {
		r.dataset = samples
	}
}

// WithParallel sets the number of samples to evaluate concurrently.
func WithParallel(n int) RunnerOption {
	return func(r *EvalRunner) {
		if n > 0 {
			r.cfg.Parallel = n
		}
	}
}

// WithTimeout sets the maximum duration for the evaluation run.
func WithTimeout(d time.Duration) RunnerOption {
	return func(r *EvalRunner) {
		r.cfg.Timeout = d
	}
}

// WithStopOnError configures the runner to stop on the first error.
func WithStopOnError(stop bool) RunnerOption {
	return func(r *EvalRunner) {
		r.cfg.StopOnError = stop
	}
}

// WithHooks sets the lifecycle hooks for the evaluation run.
func WithHooks(hooks Hooks) RunnerOption {
	return func(r *EvalRunner) {
		r.hooks = hooks
	}
}

// EvalRunner runs a set of metrics against a dataset of samples.
type EvalRunner struct {
	metrics []Metric
	dataset []EvalSample
	cfg     Config
	hooks   Hooks
}

// NewRunner creates a new EvalRunner with the given options.
func NewRunner(opts ...RunnerOption) *EvalRunner {
	r := &EvalRunner{
		cfg: Config{
			Parallel: 1,
		},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Run executes all configured metrics against all samples and returns
// an aggregate report. Samples are evaluated with the configured
// concurrency level.
func (r *EvalRunner) Run(ctx context.Context) (*EvalReport, error) {
	if r.cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.cfg.Timeout)
		defer cancel()
	}

	if r.hooks.BeforeRun != nil {
		if err := r.hooks.BeforeRun(ctx, r.dataset); err != nil {
			return nil, err
		}
	}

	start := time.Now()
	results := make([]SampleResult, len(r.dataset))

	sem := make(chan struct{}, r.cfg.Parallel)
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error
	stopped := false

	for i, sample := range r.dataset {
		mu.Lock()
		if stopped {
			mu.Unlock()
			break
		}
		mu.Unlock()

		if err := ctx.Err(); err != nil {
			mu.Lock()
			if firstErr == nil {
				firstErr = err
			}
			mu.Unlock()
			break
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, s EvalSample) {
			defer wg.Done()
			defer func() { <-sem }()

			if r.hooks.BeforeSample != nil {
				if err := r.hooks.BeforeSample(ctx, s); err != nil {
					mu.Lock()
					results[idx] = SampleResult{Sample: s, Error: err}
					if r.cfg.StopOnError {
						stopped = true
						if firstErr == nil {
							firstErr = err
						}
					}
					mu.Unlock()
					return
				}
			}

			result := r.evaluateSample(ctx, s)
			mu.Lock()
			results[idx] = result
			if result.Error != nil && r.cfg.StopOnError {
				stopped = true
				if firstErr == nil {
					firstErr = result.Error
				}
			}
			mu.Unlock()

			if r.hooks.AfterSample != nil {
				r.hooks.AfterSample(ctx, result)
			}
		}(i, sample)
	}

	wg.Wait()

	report := r.buildReport(results, time.Since(start))

	if r.hooks.AfterRun != nil {
		r.hooks.AfterRun(ctx, report)
	}

	return report, nil
}

// evaluateSample runs all metrics against a single sample.
func (r *EvalRunner) evaluateSample(ctx context.Context, sample EvalSample) SampleResult {
	scores := make(map[string]float64, len(r.metrics))
	var sampleErr error

	for _, m := range r.metrics {
		if ctx.Err() != nil {
			sampleErr = ctx.Err()
			break
		}
		score, err := m.Score(ctx, sample)
		if err != nil {
			sampleErr = err
			if r.cfg.StopOnError {
				break
			}
			continue
		}
		scores[m.Name()] = score
	}

	return SampleResult{
		Sample: sample,
		Scores: scores,
		Error:  sampleErr,
	}
}

// buildReport aggregates per-sample results into an EvalReport.
func (r *EvalRunner) buildReport(results []SampleResult, duration time.Duration) *EvalReport {
	report := &EvalReport{
		Samples:  results,
		Metrics:  make(map[string]float64),
		Duration: duration,
	}

	// Accumulate scores for averaging.
	sums := make(map[string]float64)
	counts := make(map[string]int)

	for _, res := range results {
		if res.Error != nil {
			report.Errors = append(report.Errors, res.Error)
		}
		for name, score := range res.Scores {
			sums[name] += score
			counts[name]++
		}
	}

	for name, sum := range sums {
		if counts[name] > 0 {
			report.Metrics[name] = sum / float64(counts[name])
		}
	}

	return report
}
