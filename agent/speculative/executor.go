package speculative

import (
	"context"
	"iter"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/tool"
)

// compile-time check that SpeculativeExecutor implements agent.Agent.
var _ agent.Agent = (*SpeculativeExecutor)(nil)

// ExecutorOption configures a SpeculativeExecutor.
type ExecutorOption func(*executorConfig)

type executorConfig struct {
	predictor           Predictor
	validator           Validator
	confidenceThreshold float64
	hooks               Hooks
	metrics             *Metrics
	fastAgent           agent.Agent
	earlyReturn         bool
}

func defaultExecutorConfig() executorConfig {
	return executorConfig{
		validator:           NewExactValidator(),
		confidenceThreshold: 0.7,
	}
}

// WithPredictor sets the predictor for speculative execution.
func WithPredictor(p Predictor) ExecutorOption {
	return func(c *executorConfig) {
		c.predictor = p
	}
}

// WithValidator sets the validator for checking prediction accuracy.
func WithValidator(v Validator) ExecutorOption {
	return func(c *executorConfig) {
		c.validator = v
	}
}

// WithConfidenceThreshold sets the minimum confidence required to use a
// prediction. Default is 0.7.
func WithConfidenceThreshold(threshold float64) ExecutorOption {
	return func(c *executorConfig) {
		if threshold >= 0 && threshold <= 1 {
			c.confidenceThreshold = threshold
		}
	}
}

// WithHooks sets lifecycle hooks for speculative execution events.
func WithHooks(h Hooks) ExecutorOption {
	return func(c *executorConfig) {
		c.hooks = h
	}
}

// WithMetrics sets the metrics tracker. If nil, a new one is created.
func WithMetrics(m *Metrics) ExecutorOption {
	return func(c *executorConfig) {
		c.metrics = m
	}
}

// WithEarlyReturn makes Invoke return a confident prediction as soon as the
// predictor finishes, without waiting for the ground-truth agent. The ground
// truth keeps running in the background (detached from the caller's
// cancellation) and the prediction is validated against it afterwards; a
// wrong prediction is reported through Hooks.OnMisprediction, which is the
// caller's chance to correct or roll back. Only enable this when acting on an
// occasionally wrong answer is acceptable.
//
// Without WithEarlyReturn, Invoke always waits for the ground truth and only
// returns a prediction that has been validated, so its latency is that of the
// slower of the two.
func WithEarlyReturn() ExecutorOption {
	return func(c *executorConfig) {
		c.earlyReturn = true
	}
}

// WithFastAgent sets an optional fast agent used for streaming when the
// predictor confidence exceeds the threshold.
func WithFastAgent(a agent.Agent) ExecutorOption {
	return func(c *executorConfig) {
		c.fastAgent = a
	}
}

// SpeculativeExecutor implements agent.Agent by running a predictor and
// ground-truth agent in parallel.
//
// By default Invoke waits for both and returns the prediction only when the
// validator accepts it (a "validated prediction"); otherwise the ground-truth
// result is used. This never returns sooner than the ground truth alone. With
// [WithEarlyReturn], a confident prediction is returned immediately and
// validated in the background, which is where the latency saving comes from.
type SpeculativeExecutor struct { //nolint:revive // stutter: renaming would break the exported API
	groundTruth agent.Agent
	config      executorConfig
}

// NewSpeculativeExecutor creates a new SpeculativeExecutor wrapping the given
// ground-truth agent.
func NewSpeculativeExecutor(groundTruth agent.Agent, opts ...ExecutorOption) *SpeculativeExecutor {
	cfg := defaultExecutorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.metrics == nil {
		cfg.metrics = NewMetrics()
	}
	return &SpeculativeExecutor{
		groundTruth: groundTruth,
		config:      cfg,
	}
}

// ID returns the ground-truth agent's ID prefixed with "speculative-".
func (e *SpeculativeExecutor) ID() string {
	return "speculative-" + e.groundTruth.ID()
}

// Persona returns the ground-truth agent's persona.
func (e *SpeculativeExecutor) Persona() agent.Persona {
	return e.groundTruth.Persona()
}

// Tools returns the ground-truth agent's tools.
func (e *SpeculativeExecutor) Tools() []tool.Tool {
	return e.groundTruth.Tools()
}

// Children returns the ground-truth agent as the only child.
func (e *SpeculativeExecutor) Children() []agent.Agent {
	return []agent.Agent{e.groundTruth}
}

// Metrics returns the metrics tracker for this executor.
func (e *SpeculativeExecutor) Metrics() *Metrics {
	return e.config.metrics
}

// Invoke runs the predictor and ground-truth agent in parallel. A confident
// prediction that passes validation is returned; otherwise the ground-truth
// result is used. See [WithEarlyReturn] for returning before validation.
func (e *SpeculativeExecutor) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	if e.config.predictor == nil {
		// No predictor configured; fall through to ground truth.
		return e.groundTruth.Invoke(ctx, input, opts...)
	}

	predCh := make(chan predictionOutcome, 1)
	gtCh := make(chan groundTruthOutcome, 1)

	gtBase := ctx
	if e.config.earlyReturn {
		// The ground truth may outlive this call to validate an early answer.
		gtBase = context.WithoutCancel(ctx)
	}
	gtCtx, gtCancel := context.WithCancel(gtBase)
	detached := false
	defer func() {
		if !detached {
			gtCancel()
		}
	}()

	// Run predictor and ground-truth in parallel. Synchronization is done via
	// the buffered channels below — no WaitGroup is needed.
	go func() {
		prediction, confidence, err := e.config.predictor.Predict(ctx, input)
		predCh <- predictionOutcome{text: prediction, confidence: confidence, err: err}
	}()

	go func() {
		text, err := e.groundTruth.Invoke(gtCtx, input, opts...)
		gtCh <- groundTruthOutcome{text: text, err: err}
	}()

	// Wait for predictor result first.
	pred := <-predCh
	predDone := time.Now()

	// Fire OnPrediction immediately (regardless of branch) so that hook
	// consumers observing predictor latency get consistent timing.
	if pred.err == nil && e.config.hooks.OnPrediction != nil {
		e.config.hooks.OnPrediction(ctx, pred.text, pred.confidence)
	}

	if pred.err != nil || pred.confidence < e.config.confidenceThreshold {
		// Prediction failed or low confidence — use ground truth.
		return e.fallbackToGroundTruth(ctx, pred, gtCh)
	}

	if e.config.earlyReturn {
		// Return now and validate in the background.
		detached = true
		bgCtx := context.WithoutCancel(ctx)
		go func() {
			defer gtCancel()
			gt := <-gtCh
			if gt.err != nil {
				return
			}
			e.resolveWithGroundTruth(bgCtx, pred, gt.text, time.Since(predDone))
		}()
		return pred.text, nil
	}

	// Wait for ground truth to validate.
	gt := <-gtCh
	gtDone := time.Now()

	if gt.err != nil {
		// Ground truth failed but we have a prediction. Return the prediction
		// only if the error is retryable (ground truth might be temporarily down).
		if core.IsRetryable(gt.err) {
			return pred.text, nil
		}
		return "", gt.err
	}

	return e.resolveWithGroundTruth(ctx, pred, gt.text, gtDone.Sub(predDone)), nil
}

// predictionOutcome carries the predictor's result from its goroutine.
type predictionOutcome struct {
	text       string
	confidence float64
	err        error
}

// groundTruthOutcome carries the ground-truth agent's result from its goroutine.
type groundTruthOutcome struct {
	text string
	err  error
}

// fallbackToGroundTruth handles a failed or low-confidence prediction: it
// fires OnCancel on predictor error, waits for the ground truth, and records a
// misprediction when a low-confidence prediction was available.
func (e *SpeculativeExecutor) fallbackToGroundTruth(ctx context.Context, pred predictionOutcome, gtCh <-chan groundTruthOutcome) (string, error) {
	if pred.err != nil && e.config.hooks.OnCancel != nil {
		e.config.hooks.OnCancel(ctx, pred.err)
	}
	gt := <-gtCh
	if gt.err != nil {
		return "", gt.err
	}
	if pred.err == nil {
		// We got a prediction but confidence was too low.
		e.config.metrics.RecordMiss(estimateTokens(pred.text))
		if e.config.hooks.OnMisprediction != nil {
			e.config.hooks.OnMisprediction(ctx, Result{
				Prediction:  pred.text,
				GroundTruth: gt.text,
				Validated:   false,
				Confidence:  pred.confidence,
				Output:      gt.text,
			})
		}
	}
	return gt.text, nil
}

// resolveWithGroundTruth validates a confident prediction against the ground
// truth text, records hit/miss metrics, fires the matching hook, and returns
// the output to use. A validator error falls back to the ground truth.
func (e *SpeculativeExecutor) resolveWithGroundTruth(ctx context.Context, pred predictionOutcome, gtText string, speedup time.Duration) string {
	valid, err := e.config.validator.Validate(ctx, pred.text, gtText)
	if err != nil {
		// Validation error — use ground truth.
		return gtText
	}

	result := Result{
		Prediction:  pred.text,
		GroundTruth: gtText,
		Validated:   valid,
		Confidence:  pred.confidence,
		Speedup:     speedup,
		Output:      gtText,
	}

	if valid {
		result.Output = pred.text
		e.config.metrics.RecordHit(speedup)
		if e.config.hooks.OnValidation != nil {
			e.config.hooks.OnValidation(ctx, result)
		}
		return pred.text
	}

	result.WastedTokens = estimateTokens(pred.text)
	e.config.metrics.RecordMiss(result.WastedTokens)
	if e.config.hooks.OnMisprediction != nil {
		e.config.hooks.OnMisprediction(ctx, result)
	}
	return gtText
}

// Stream pre-classifies via predictor confidence. If the predictor reports
// high confidence and a fast agent is configured, the fast agent's stream
// is used. Otherwise, the ground-truth agent's stream is used.
func (e *SpeculativeExecutor) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		// If no predictor or no fast agent, stream ground truth directly.
		if e.config.predictor == nil || e.config.fastAgent == nil {
			relayStream(e.groundTruth.Stream(ctx, input, opts...), yield)
			return
		}

		// Pre-classify: check predictor confidence.
		prediction, confidence, err := e.config.predictor.Predict(ctx, input)
		if err != nil {
			if e.config.hooks.OnCancel != nil {
				e.config.hooks.OnCancel(ctx, err)
			}
			// Fall through to ground truth on predictor error.
			relayStream(e.groundTruth.Stream(ctx, input, opts...), yield)
			return
		}

		if e.config.hooks.OnPrediction != nil {
			e.config.hooks.OnPrediction(ctx, prediction, confidence)
		}

		relayStream(e.streamTarget(confidence).Stream(ctx, input, opts...), yield)
	}
}

// streamTarget picks the agent to stream from for a successful prediction
// and records the routing metric. High confidence routes to the fast agent;
// this is an unvalidated pre-classification decision, so it uses a distinct
// metric and does not pollute the validated HitRate(). Low confidence routes
// to the ground truth.
func (e *SpeculativeExecutor) streamTarget(confidence float64) agent.Agent {
	if confidence >= e.config.confidenceThreshold {
		e.config.metrics.RecordStreamFastRoute()
		return e.config.fastAgent
	}
	e.config.metrics.RecordStreamGroundRoute()
	return e.groundTruth
}

// relayStream forwards every (event, err) pair from src to yield, stopping
// after the first error or when the consumer stops iteration.
func relayStream(src iter.Seq2[agent.Event, error], yield func(agent.Event, error) bool) {
	for event, err := range src {
		if !yield(event, err) || err != nil {
			return
		}
	}
}

// estimateTokens provides a rough token count estimate based on word count.
// A common heuristic is ~1.3 tokens per word for English text.
func estimateTokens(text string) int {
	words := len(splitWords(text))
	return int(float64(words) * 1.3)
}

// splitWords splits text into words using simple whitespace splitting.
func splitWords(text string) []string {
	var words []string
	start := -1
	for i, r := range text {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if start >= 0 {
				words = append(words, text[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		words = append(words, text[start:])
	}
	return words
}
