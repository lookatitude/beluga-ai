package cost

import (
	"context"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/eval"
	"github.com/lookatitude/beluga-ai/eval/metrics"
)

// Compile-time interface check.
var _ eval.Metric = (*CostMetric)(nil)

// ModelPricing defines per-token pricing for a model. It is a type alias for
// metrics.ModelPricing so that callers can share a single pricing map between
// the eval/metrics.Cost metric and the eval/cost package without duplication.
type ModelPricing = metrics.ModelPricing

// defaultQPDReference is the default quality-per-dollar value that maps to a
// normalized score of 1.0. It is intentionally conservative; callers with
// different cost/quality profiles should use WithQPDReference to override it.
const defaultQPDReference = 1000.0

// defaultCostReference is the default dollar cost that maps to a normalized
// score of 0.0 when no quality metric is configured. Costs at or above this
// value yield a score of 0.0; a cost of 0 yields 1.0. Callers with different
// budget profiles should use WithCostReference to override it.
const defaultCostReference = 1.0

// costOptions holds configuration for CostMetric.
type costOptions struct {
	pricing       map[string]ModelPricing
	qualityMetric eval.Metric
	metricName    string
	qpdReference  float64
	costReference float64
}

// CostOption configures a CostMetric.
type CostOption func(*costOptions)

// WithPricing sets the model pricing map.
func WithPricing(pricing map[string]ModelPricing) CostOption {
	return func(o *costOptions) {
		o.pricing = pricing
	}
}

// WithQualityMetric sets the quality metric used for quality-per-dollar
// computation. If nil, the CostMetric returns a cost-only normalized score.
func WithQualityMetric(m eval.Metric) CostOption {
	return func(o *costOptions) {
		o.qualityMetric = m
	}
}

// WithMetricName sets the name returned by Name(). Defaults to "cost_quality".
func WithMetricName(name string) CostOption {
	return func(o *costOptions) {
		o.metricName = name
	}
}

// WithQPDReference sets the quality-per-dollar reference value that maps to a
// normalized score of 1.0. Values above the reference are clamped to 1.0.
// Must be positive. Defaults to 1000.
func WithQPDReference(ref float64) CostOption {
	return func(o *costOptions) {
		if ref > 0 {
			o.qpdReference = ref
		}
	}
}

// WithCostReference sets the dollar cost that maps to a normalized score of
// 0.0 when no quality metric is configured. Must be positive. Defaults to 1.0.
func WithCostReference(ref float64) CostOption {
	return func(o *costOptions) {
		if ref > 0 {
			o.costReference = ref
		}
	}
}

// CostMetric computes a normalized cost-aware score for evaluation samples.
// It reads Metadata["model"], Metadata["input_tokens"], and
// Metadata["output_tokens"] from the sample.
//
// When a quality metric is configured, Score returns quality-per-dollar
// normalized by the configured QPD reference and clamped to [0, 1].
//
// When no quality metric is configured, Score returns a cost-only normalized
// score (1 - cost/costReference, clamped to [0, 1]): a zero-cost sample scores
// 1.0 and a sample at or above the cost reference scores 0.0. Use
// ComputeRawCost for the raw dollar amount (e.g. for budget tracking).
type CostMetric struct {
	opts costOptions
}

// NewCostMetric creates a new CostMetric with the given options.
func NewCostMetric(opts ...CostOption) *CostMetric {
	o := costOptions{
		pricing:       make(map[string]ModelPricing),
		metricName:    "cost_quality",
		qpdReference:  defaultQPDReference,
		costReference: defaultCostReference,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &CostMetric{opts: o}
}

// Name returns the metric name.
func (c *CostMetric) Name() string { return c.opts.metricName }

// Score computes a normalized cost-aware score for a sample, always in
// [0.0, 1.0] as required by the eval.Metric contract.
//
// If a quality metric is configured, returns quality/cost normalized by the
// QPD reference. Otherwise returns a cost-only normalized score
// (1 - cost/costReference).
func (c *CostMetric) Score(ctx context.Context, sample eval.EvalSample) (float64, error) {
	cost, err := c.computeCost(sample)
	if err != nil {
		return 0, err
	}

	if c.opts.qualityMetric == nil {
		// Cost-only normalized score: 1.0 at zero cost, 0.0 at or above the
		// reference. Always in [0, 1] to honor the eval.Metric contract.
		ref := c.opts.costReference
		if ref <= 0 {
			ref = defaultCostReference
		}
		score := 1.0 - (cost / ref)
		return clamp01(score), nil
	}

	quality, err := c.opts.qualityMetric.Score(ctx, sample)
	if err != nil {
		return 0, core.Errorf(core.ErrInvalidInput, "cost_quality: quality metric: %w", err)
	}

	if cost <= 0 {
		if quality > 0 {
			return 1.0, nil
		}
		return 0, nil
	}

	// Quality per dollar, normalized to [0, 1] using the configured reference.
	ref := c.opts.qpdReference
	if ref <= 0 {
		ref = defaultQPDReference
	}
	qpd := quality / cost
	return clamp01(qpd / ref), nil
}

// ComputeRawCost returns the raw dollar cost for a sample without any
// quality normalization. Useful for budget tracking.
func (c *CostMetric) ComputeRawCost(sample eval.EvalSample) (float64, error) {
	return c.computeCost(sample)
}

// computeCost calculates dollar cost from sample metadata.
func (c *CostMetric) computeCost(sample eval.EvalSample) (float64, error) {
	modelRaw, ok := sample.Metadata["model"]
	if !ok {
		return 0, core.Errorf(core.ErrInvalidInput, "cost: missing metadata key %q", "model")
	}
	model, ok := modelRaw.(string)
	if !ok {
		return 0, core.Errorf(core.ErrInvalidInput, "cost: metadata %q must be a string, got %T", "model", modelRaw)
	}

	pricing, ok := c.opts.pricing[model]
	if !ok {
		return 0, core.Errorf(core.ErrNotFound, "cost: no pricing for model %q", model)
	}

	inputRaw, ok := sample.Metadata["input_tokens"]
	if !ok {
		return 0, core.Errorf(core.ErrInvalidInput, "cost: missing metadata key %q", "input_tokens")
	}
	inputTokens, err := toFloat64(inputRaw)
	if err != nil {
		return 0, core.Errorf(core.ErrInvalidInput, "cost: input_tokens: %w", err)
	}

	outputRaw, ok := sample.Metadata["output_tokens"]
	if !ok {
		return 0, core.Errorf(core.ErrInvalidInput, "cost: missing metadata key %q", "output_tokens")
	}
	outputTokens, err := toFloat64(outputRaw)
	if err != nil {
		return 0, core.Errorf(core.ErrInvalidInput, "cost: output_tokens: %w", err)
	}

	return (inputTokens*pricing.InputTokenPrice + outputTokens*pricing.OutputTokenPrice) / 1_000_000, nil
}

// clamp01 clamps a value into [0, 1].
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// toFloat64 converts numeric interface values to float64.
func toFloat64(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case int32:
		return float64(n), nil
	default:
		return 0, core.Errorf(core.ErrInvalidInput, "unsupported numeric type %T", v)
	}
}
