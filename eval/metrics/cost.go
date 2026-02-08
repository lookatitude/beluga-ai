package metrics

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/eval"
)

// ModelPricing defines per-token pricing for a model.
type ModelPricing struct {
	// InputTokenPrice is the price per 1 million input tokens in dollars.
	InputTokenPrice float64
	// OutputTokenPrice is the price per 1 million output tokens in dollars.
	OutputTokenPrice float64
}

// Cost calculates the dollar cost of a generation by reading token counts and
// model name from sample metadata. It reads Metadata["input_tokens"],
// Metadata["output_tokens"], and Metadata["model"].
//
// Unlike other metrics, Cost returns the raw dollar amount rather than a
// normalized 0-1 score.
type Cost struct {
	pricing map[string]ModelPricing
}

// CostOption configures a Cost metric.
type CostOption func(*Cost)

// WithPricing sets the model pricing map.
func WithPricing(pricing map[string]ModelPricing) CostOption {
	return func(c *Cost) {
		c.pricing = pricing
	}
}

// NewCost creates a new Cost metric with the given options.
func NewCost(opts ...CostOption) *Cost {
	c := &Cost{
		pricing: make(map[string]ModelPricing),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Name returns "cost".
func (c *Cost) Name() string { return "cost" }

// Score calculates the cost in dollars for the sample based on token usage
// and model pricing. Returns the dollar cost as the score value.
func (c *Cost) Score(_ context.Context, sample eval.EvalSample) (float64, error) {
	modelRaw, ok := sample.Metadata["model"]
	if !ok {
		return 0, fmt.Errorf("cost: missing metadata key %q", "model")
	}
	model, ok := modelRaw.(string)
	if !ok {
		return 0, fmt.Errorf("cost: metadata %q must be a string, got %T", "model", modelRaw)
	}

	pricing, ok := c.pricing[model]
	if !ok {
		return 0, fmt.Errorf("cost: no pricing for model %q", model)
	}

	inputRaw, ok := sample.Metadata["input_tokens"]
	if !ok {
		return 0, fmt.Errorf("cost: missing metadata key %q", "input_tokens")
	}
	inputTokens, err := toFloat64(inputRaw)
	if err != nil {
		return 0, fmt.Errorf("cost: input_tokens: %w", err)
	}

	outputRaw, ok := sample.Metadata["output_tokens"]
	if !ok {
		return 0, fmt.Errorf("cost: missing metadata key %q", "output_tokens")
	}
	outputTokens, err := toFloat64(outputRaw)
	if err != nil {
		return 0, fmt.Errorf("cost: output_tokens: %w", err)
	}

	cost := (inputTokens * pricing.InputTokenPrice / 1_000_000) +
		(outputTokens * pricing.OutputTokenPrice / 1_000_000)

	return cost, nil
}
