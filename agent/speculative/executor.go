package speculative

import (
	"context"
	"iter"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/tool"
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

// WithFastAgent sets an optional fast agent used for streaming when the
// predictor confidence exceeds the threshold.
func WithFastAgent(a agent.Agent) ExecutorOption {
	return func(c *executorConfig) {
		c.fastAgent = a
	}
}

// SpeculativeExecutor implements agent.Agent by running a predictor and
// ground-truth agent in parallel. If the prediction is validated, the result
// is returned with a speedup. Otherwise, the ground-truth result is used.
type SpeculativeExecutor struct {
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

// Invoke runs the predictor and ground-truth agent in parallel. If the
// predictor finishes first with sufficient confidence and passes validation,
// the prediction is returned. Otherwise, the ground-truth result is used.
func (e *SpeculativeExecutor) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	if e.config.predictor == nil {
		// No predictor configured; fall through to ground truth.
		return e.groundTruth.Invoke(ctx, input, opts...)
	}

	type predResult struct {
		text       string
		confidence float64
		err        error
	}

	type gtResult struct {
		text string
		err  error
	}

	predCh := make(chan predResult, 1)
	gtCh := make(chan gtResult, 1)

	gtCtx, gtCancel := context.WithCancel(ctx)
	defer gtCancel()

	// Run predictor and ground-truth in parallel. Synchronization is done via
	// the buffered channels below — no WaitGroup is needed.
	go func() {
		prediction, confidence, err := e.config.predictor.Predict(ctx, input)
		predCh <- predResult{text: prediction, confidence: confidence, err: err}
	}()

	go func() {
		text, err := e.groundTruth.Invoke(gtCtx, input, opts...)
		gtCh <- gtResult{text: text, err: err}
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

	// Validate prediction against ground truth.
	valid, err := e.config.validator.Validate(ctx, pred.text, gt.text)
	if err != nil {
		// Validation error — use ground truth.
		return gt.text, nil
	}

	speedup := gtDone.Sub(predDone)
	result := Result{
		Prediction:  pred.text,
		GroundTruth: gt.text,
		Validated:   valid,
		Confidence:  pred.confidence,
		Speedup:     speedup,
		Output:      gt.text,
	}

	if valid {
		result.Output = pred.text
		e.config.metrics.RecordHit(speedup)
		if e.config.hooks.OnValidation != nil {
			e.config.hooks.OnValidation(ctx, result)
		}
		return pred.text, nil
	}

	result.WastedTokens = estimateTokens(pred.text)
	e.config.metrics.RecordMiss(result.WastedTokens)
	if e.config.hooks.OnMisprediction != nil {
		e.config.hooks.OnMisprediction(ctx, result)
	}
	return gt.text, nil
}

// Stream pre-classifies via predictor confidence. If the predictor reports
// high confidence and a fast agent is configured, the fast agent's stream
// is used. Otherwise, the ground-truth agent's stream is used.
func (e *SpeculativeExecutor) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		// If no predictor or no fast agent, stream ground truth directly.
		if e.config.predictor == nil || e.config.fastAgent == nil {
			for event, err := range e.groundTruth.Stream(ctx, input, opts...) {
				if !yield(event, err) {
					return
				}
				if err != nil {
					return
				}
			}
			return
		}

		// Pre-classify: check predictor confidence.
		prediction, confidence, err := e.config.predictor.Predict(ctx, input)
		if err != nil {
			if e.config.hooks.OnCancel != nil {
				e.config.hooks.OnCancel(ctx, err)
			}
			// Fall through to ground truth on predictor error.
			for event, err := range e.groundTruth.Stream(ctx, input, opts...) {
				if !yield(event, err) {
					return
				}
				if err != nil {
					return
				}
			}
			return
		}

		if e.config.hooks.OnPrediction != nil {
			e.config.hooks.OnPrediction(ctx, prediction, confidence)
		}

		if confidence >= e.config.confidenceThreshold {
			// High confidence — route to fast agent. This is an unvalidated
			// pre-classification decision; we use a distinct metric so that
			// validated HitRate() is not polluted.
			e.config.metrics.RecordStreamFastRoute()
			for event, err := range e.config.fastAgent.Stream(ctx, input, opts...) {
				if !yield(event, err) {
					return
				}
				if err != nil {
					return
				}
			}
			return
		}

		// Low confidence — use ground truth.
		e.config.metrics.RecordStreamGroundRoute()
		for event, err := range e.groundTruth.Stream(ctx, input, opts...) {
			if !yield(event, err) {
				return
			}
			if err != nil {
				return
			}
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
