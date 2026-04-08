package cost

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/eval"
)

// Compile-time interface check.
var _ eval.Metric = (*CostMetric)(nil)

// ModelPricing defines per-token pricing for a model.
type ModelPricing struct {
	// InputTokenPrice is the price per 1 million input tokens in dollars.
	InputTokenPrice float64
	// OutputTokenPrice is the price per 1 million output tokens in dollars.
	OutputTokenPrice float64
}

// costOptions holds configuration for CostMetric.
type costOptions struct {
	pricing       map[string]ModelPricing
	qualityMetric eval.Metric
	metricName    string
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
// computation. If nil, the CostMetric returns only the raw cost.
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

// CostMetric computes quality-per-dollar by combining a quality metric score
// with token cost. It reads Metadata["model"], Metadata["input_tokens"], and
// Metadata["output_tokens"] from the sample.
//
// When a quality metric is set, the score is quality / cost (higher is better,
// clamped to [0, 1] by dividing by a reference of 1000 quality-per-dollar).
// When no quality metric is set, the score is 1 - normalized_cost.
type CostMetric struct {
	opts costOptions
}

// NewCostMetric creates a new CostMetric with the given options.
func NewCostMetric(opts ...CostOption) *CostMetric {
	o := costOptions{
		pricing:    make(map[string]ModelPricing),
		metricName: "cost_quality",
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &CostMetric{opts: o}
}

// Name returns the metric name.
func (c *CostMetric) Name() string { return c.opts.metricName }

// Score computes the quality-per-dollar score for a sample. If a quality
// metric is configured, returns quality/cost normalized to [0, 1]. Otherwise
// returns the raw dollar cost as the score (not normalized).
func (c *CostMetric) Score(ctx context.Context, sample eval.EvalSample) (float64, error) {
	cost, err := c.computeCost(sample)
	if err != nil {
		return 0, err
	}

	if c.opts.qualityMetric == nil {
		return cost, nil
	}

	quality, err := c.opts.qualityMetric.Score(ctx, sample)
	if err != nil {
		return 0, fmt.Errorf("cost_quality: quality metric: %w", err)
	}

	if cost <= 0 {
		if quality > 0 {
			return 1.0, nil
		}
		return 0, nil
	}

	// Quality per dollar, normalized to [0, 1] using 1000 as reference.
	qpd := quality / cost
	score := qpd / 1000.0
	if score > 1.0 {
		score = 1.0
	}
	return score, nil
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
		return 0, fmt.Errorf("cost: missing metadata key %q", "model")
	}
	model, ok := modelRaw.(string)
	if !ok {
		return 0, fmt.Errorf("cost: metadata %q must be a string", "model")
	}

	pricing, ok := c.opts.pricing[model]
	if !ok {
		return 0, fmt.Errorf("cost: no pricing for model %q", model)
	}

	inputTokens, err := toFloat64(sample.Metadata["input_tokens"])
	if err != nil {
		return 0, fmt.Errorf("cost: input_tokens: %w", err)
	}

	outputTokens, err := toFloat64(sample.Metadata["output_tokens"])
	if err != nil {
		return 0, fmt.Errorf("cost: output_tokens: %w", err)
	}

	return (inputTokens*pricing.InputTokenPrice + outputTokens*pricing.OutputTokenPrice) / 1_000_000, nil
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
		return 0, fmt.Errorf("unsupported numeric type %T", v)
	}
}
