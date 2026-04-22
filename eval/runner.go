package eval

import (
	"context"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/v2/o11y"
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

// WithDatasetName attaches the dataset identifier that is emitted as the
// beluga.eval.dataset attribute on the eval.run and eval.row spans and as a
// label dimension on the beluga.eval.metric.score Histogram. Empty names are
// accepted and emitted unchanged — callers must set this when the metric
// aggregation label matters.
func WithDatasetName(name string) RunnerOption {
	return func(r *EvalRunner) {
		r.datasetName = name
	}
}

// EvalRunner runs a set of metrics against a dataset of samples.
type EvalRunner struct {
	metrics     []Metric
	dataset     []EvalSample
	datasetName string
	cfg         Config
	hooks       Hooks
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
// concurrency level. The entire run is wrapped in an eval.run span; each
// sample in an eval.row child span.
func (r *EvalRunner) Run(ctx context.Context) (*EvalReport, error) {
	if r.cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.cfg.Timeout)
		defer cancel()
	}

	ctx, runSpan := startRunSpan(ctx, r.datasetName, len(r.dataset), len(r.metrics))
	defer runSpan.End()

	if r.hooks.BeforeRun != nil {
		if err := r.hooks.BeforeRun(ctx, r.dataset); err != nil {
			runSpan.RecordError(err)
			runSpan.SetStatus(o11y.StatusError, err.Error())
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
			r.processSample(ctx, idx, s, results, &mu, &stopped, &firstErr)
		}(i, sample)
	}

	wg.Wait()

	report := r.buildReport(results, time.Since(start))

	if firstErr != nil {
		runSpan.RecordError(firstErr)
		runSpan.SetStatus(o11y.StatusError, firstErr.Error())
	} else if len(report.Errors) > 0 {
		runSpan.SetStatus(o11y.StatusError, "one or more samples failed")
	} else {
		runSpan.SetStatus(o11y.StatusOK, "")
	}

	if r.hooks.AfterRun != nil {
		r.hooks.AfterRun(ctx, report)
	}

	return report, nil
}

// processSample evaluates a single sample with hooks and records the result.
// Wraps the evaluation in an eval.row span so that per-metric
// gen_ai.evaluation.result events attach to it.
func (r *EvalRunner) processSample(ctx context.Context, idx int, s EvalSample, results []SampleResult, mu *sync.Mutex, stopped *bool, firstErr *error) {
	rowCtx, rowSpan := startRowSpan(ctx, r.datasetName, idx)
	defer rowSpan.End()

	if r.hooks.BeforeSample != nil {
		if err := r.hooks.BeforeSample(rowCtx, s); err != nil {
			rowSpan.RecordError(err)
			rowSpan.SetStatus(o11y.StatusError, err.Error())
			r.recordResult(idx, SampleResult{Sample: s, Error: err}, results, mu, stopped, firstErr)
			return
		}
	}

	result := r.evaluateSample(rowCtx, s)
	r.recordResult(idx, result, results, mu, stopped, firstErr)

	if result.Error != nil {
		rowSpan.RecordError(result.Error)
		rowSpan.SetStatus(o11y.StatusError, result.Error.Error())
	} else {
		rowSpan.SetStatus(o11y.StatusOK, "")
	}

	if r.hooks.AfterSample != nil {
		r.hooks.AfterSample(rowCtx, result)
	}
}

// recordResult stores a sample result and updates the stopped/error state under lock.
func (r *EvalRunner) recordResult(idx int, result SampleResult, results []SampleResult, mu *sync.Mutex, stopped *bool, firstErr *error) {
	mu.Lock()
	results[idx] = result
	if result.Error != nil && r.cfg.StopOnError {
		*stopped = true
		if *firstErr == nil {
			*firstErr = result.Error
		}
	}
	mu.Unlock()
}

// evaluateSample runs all metrics against a single sample. On success, each
// metric emits a gen_ai.evaluation.result event on the enclosing eval.row span
// and a beluga.eval.metric.score Histogram sample.
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
		recordEvalResult(ctx, m.Name(), score)
		recordMetricScore(ctx, m.Name(), r.datasetName, score)
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
