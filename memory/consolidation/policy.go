package consolidation

import (
	"context"
	"time"
)

// ConsolidationPolicy evaluates a set of records and returns a decision for
// each one. Implementations must be safe for concurrent use.
type ConsolidationPolicy interface {
	// Evaluate scores the given records and returns a decision (keep, prune,
	// or compress) for each record. The returned slice must have the same
	// length as the input.
	Evaluate(ctx context.Context, records []Record) ([]Decision, error)
}

// ThresholdPolicy prunes records whose composite utility score falls below
// a configurable threshold (default 0.25).
type ThresholdPolicy struct {
	// Threshold is the minimum composite score to keep a record.
	// Records scoring below this value are marked for pruning.
	Threshold float64

	// CompressThreshold is the score below which records are compressed
	// rather than pruned. Records scoring between CompressThreshold and
	// Threshold are compressed. If zero, compression is disabled.
	CompressThreshold float64

	// Weights controls the composite score computation.
	Weights Weights

	// HalfLife controls recency decay. Zero means DefaultHalfLife.
	HalfLife time.Duration
}

// Compile-time interface check.
var _ ConsolidationPolicy = (*ThresholdPolicy)(nil)

// NewThresholdPolicy creates a ThresholdPolicy with the default threshold of
// 0.25, no compression, and default weights.
func NewThresholdPolicy() *ThresholdPolicy {
	return &ThresholdPolicy{
		Threshold: 0.25,
		Weights:   DefaultWeights(),
	}
}

// Evaluate scores each record and returns prune/compress/keep decisions based
// on the configured thresholds.
func (p *ThresholdPolicy) Evaluate(ctx context.Context, records []Record) ([]Decision, error) {
	now := time.Now()
	decisions := make([]Decision, len(records))
	for i, r := range records {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Update recency from creation time.
		r.Utility.Recency = RecencyScore(r.CreatedAt, now, p.HalfLife)
		score := CompositeScore(r.Utility, p.Weights)

		action := ActionKeep
		if score < p.Threshold {
			if p.CompressThreshold > 0 && score >= p.CompressThreshold {
				action = ActionCompress
			} else {
				action = ActionPrune
			}
		}
		decisions[i] = Decision{Record: r, Action: action}
	}
	return decisions, nil
}

// FrequencyPolicy prunes records that have never been accessed after a
// configurable time-to-live (TTL) has elapsed since creation.
type FrequencyPolicy struct {
	// TTL is the duration after creation beyond which zero-access records
	// are pruned. Zero means 30 days.
	TTL time.Duration
}

// Compile-time interface check.
var _ ConsolidationPolicy = (*FrequencyPolicy)(nil)

// NewFrequencyPolicy creates a FrequencyPolicy with a default TTL of 30 days.
func NewFrequencyPolicy() *FrequencyPolicy {
	return &FrequencyPolicy{TTL: 30 * 24 * time.Hour}
}

// Evaluate checks each record and prunes those with AccessCount == 0 whose
// age exceeds the TTL.
func (p *FrequencyPolicy) Evaluate(ctx context.Context, records []Record) ([]Decision, error) {
	ttl := p.TTL
	if ttl <= 0 {
		ttl = 30 * 24 * time.Hour
	}
	now := time.Now()
	decisions := make([]Decision, len(records))
	for i, r := range records {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		action := ActionKeep
		if r.Utility.AccessCount == 0 && now.Sub(r.CreatedAt) > ttl {
			action = ActionPrune
		}
		decisions[i] = Decision{Record: r, Action: action}
	}
	return decisions, nil
}

// CompositePolicy applies multiple policies in sequence. A more severe action
// from any policy wins (Prune > Compress > Keep).
type CompositePolicy struct {
	policies []ConsolidationPolicy
}

// Compile-time interface check.
var _ ConsolidationPolicy = (*CompositePolicy)(nil)

// NewCompositePolicy creates a policy that combines the given policies.
// An empty composite always returns ActionKeep for all records.
func NewCompositePolicy(policies ...ConsolidationPolicy) *CompositePolicy {
	return &CompositePolicy{policies: policies}
}

// Evaluate runs all inner policies and merges decisions. The most severe
// action for each record wins.
func (c *CompositePolicy) Evaluate(ctx context.Context, records []Record) ([]Decision, error) {
	if len(records) == 0 {
		return nil, nil
	}

	merged := make([]Decision, len(records))
	for i, r := range records {
		merged[i] = Decision{Record: r, Action: ActionKeep}
	}

	for _, pol := range c.policies {
		decisions, err := pol.Evaluate(ctx, records)
		if err != nil {
			return nil, err
		}
		for i, d := range decisions {
			if d.Action > merged[i].Action {
				merged[i].Action = d.Action
			}
		}
	}
	return merged, nil
}
