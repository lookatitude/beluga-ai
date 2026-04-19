package judge

import (
	"context"
	"math"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/eval"
	"github.com/lookatitude/beluga-ai/v2/llm"
)

// consistencyOptions holds configuration for ConsistencyChecker.
type consistencyOptions struct {
	rubric   *Rubric
	models   []llm.ChatModel
	repeats  int
	parallel int
}

// ConsistencyOption configures a ConsistencyChecker.
type ConsistencyOption func(*consistencyOptions)

// WithConsistencyRubric sets the rubric used for consistency checking.
func WithConsistencyRubric(r *Rubric) ConsistencyOption {
	return func(o *consistencyOptions) {
		o.rubric = r
	}
}

// WithModels sets the LLM models used for cross-model agreement.
func WithModels(models ...llm.ChatModel) ConsistencyOption {
	return func(o *consistencyOptions) {
		o.models = models
	}
}

// WithRepeats sets how many times each model evaluates each sample.
// Defaults to 3.
func WithRepeats(n int) ConsistencyOption {
	return func(o *consistencyOptions) {
		if n > 0 {
			o.repeats = n
		}
	}
}

// WithConsistencyParallel sets the concurrency limit for consistency checking.
// Defaults to 2.
func WithConsistencyParallel(n int) ConsistencyOption {
	return func(o *consistencyOptions) {
		if n > 0 {
			o.parallel = n
		}
	}
}

// ConsistencyResult holds the results of a consistency check for a single sample.
type ConsistencyResult struct {
	// Scores maps model ID to the list of scores from repeated evaluations.
	Scores map[string][]float64

	// MeanScore is the overall mean across all models and repeats.
	MeanScore float64

	// StdDev is the standard deviation across all scores.
	StdDev float64

	// Agreement is the fraction of score pairs within 0.1 of each other,
	// representing cross-model agreement. Range [0.0, 1.0].
	Agreement float64
}

// ConsistencyChecker validates scoring reliability by running repeated
// evaluations with one or more LLM judges and computing agreement metrics.
type ConsistencyChecker struct {
	opts consistencyOptions
}

// NewConsistencyChecker creates a new ConsistencyChecker with the given options.
func NewConsistencyChecker(opts ...ConsistencyOption) (*ConsistencyChecker, error) {
	o := consistencyOptions{
		repeats:  3,
		parallel: 2,
	}
	for _, opt := range opts {
		opt(&o)
	}
	if len(o.models) == 0 {
		return nil, core.NewError("judge.consistency.new", core.ErrInvalidInput, "at least one model is required", nil)
	}
	if o.rubric == nil {
		return nil, core.NewError("judge.consistency.new", core.ErrInvalidInput, "rubric is required", nil)
	}
	if err := o.rubric.Validate(); err != nil {
		return nil, core.NewError("judge.consistency.new", core.ErrInvalidInput, "invalid rubric", err)
	}
	return &ConsistencyChecker{opts: o}, nil
}

// Check evaluates a single sample with all models, each repeated the
// configured number of times, and returns agreement metrics.
func (c *ConsistencyChecker) Check(ctx context.Context, sample eval.EvalSample) (*ConsistencyResult, error) {
	type scoreEntry struct {
		modelID string
		score   float64
		err     error
	}

	// Build all JudgeMetric instances eagerly before launching any goroutines
	// so that a construction failure for a later model does not leave earlier
	// goroutines running unobserved LLM calls.
	metrics := make([]*JudgeMetric, len(c.opts.models))
	for i, model := range c.opts.models {
		m, err := NewJudgeMetric(
			WithModel(model),
			WithRubric(c.opts.rubric),
		)
		if err != nil {
			return nil, err
		}
		metrics[i] = m
	}

	results := make(chan scoreEntry, len(c.opts.models)*c.opts.repeats)
	sem := make(chan struct{}, c.opts.parallel)
	var wg sync.WaitGroup

dispatch:
	for i, model := range c.opts.models {
		metric := metrics[i]
		for r := 0; r < c.opts.repeats; r++ {
			// Acquire a slot, respecting context cancellation so a cancelled
			// context is noticed even when all parallel slots are in use.
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				break dispatch
			}

			wg.Add(1)
			go func(m llm.ChatModel, jm *JudgeMetric) {
				defer wg.Done()
				defer func() { <-sem }()

				score, err := jm.Score(ctx, sample)
				results <- scoreEntry{modelID: m.ModelID(), score: score, err: err}
			}(model, metric)
		}
	}

	wg.Wait()
	close(results)

	scoresByModel := make(map[string][]float64)
	var allScores []float64

	for entry := range results {
		if entry.err != nil {
			continue
		}
		scoresByModel[entry.modelID] = append(scoresByModel[entry.modelID], entry.score)
		allScores = append(allScores, entry.score)
	}

	if len(allScores) == 0 {
		return nil, core.NewError("judge.consistency.check", core.ErrToolFailed, "all evaluations failed", nil)
	}

	mean := computeMean(allScores)
	stddev := computeStdDev(allScores, mean)
	agreement := computeAgreement(allScores)

	return &ConsistencyResult{
		Scores:    scoresByModel,
		MeanScore: mean,
		StdDev:    stddev,
		Agreement: agreement,
	}, nil
}

// computeMean returns the arithmetic mean of scores.
func computeMean(scores []float64) float64 {
	if len(scores) == 0 {
		return 0
	}
	var sum float64
	for _, s := range scores {
		sum += s
	}
	return sum / float64(len(scores))
}

// computeStdDev returns the population standard deviation.
func computeStdDev(scores []float64, mean float64) float64 {
	if len(scores) == 0 {
		return 0
	}
	var sumSq float64
	for _, s := range scores {
		d := s - mean
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(scores)))
}

// computeAgreement returns the fraction of all score pairs within 0.1 of
// each other.
func computeAgreement(scores []float64) float64 {
	n := len(scores)
	if n < 2 {
		return 1.0
	}
	total := 0
	agree := 0
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			total++
			if math.Abs(scores[i]-scores[j]) <= 0.1 {
				agree++
			}
		}
	}
	if total == 0 {
		return 1.0
	}
	return float64(agree) / float64(total)
}
