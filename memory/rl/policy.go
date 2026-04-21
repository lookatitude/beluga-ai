package rl

import (
	"context"
	"math"
)

// Decider decides which memory action to take given the current
// observation features. Implementations range from rule-based heuristics
// to trained neural network models.
type Decider interface {
	// Decide selects a MemoryAction and returns a confidence score in [0, 1].
	// The confidence indicates how certain the policy is about the chosen action.
	Decide(ctx context.Context, features PolicyFeatures) (MemoryAction, float64, error)
}

// HeuristicPolicy is a rule-based Decider that uses similarity thresholds
// and memory state to choose actions. It serves as a reasonable baseline and
// fallback when no trained model is available.
type HeuristicPolicy struct {
	// AddThreshold is the maximum similarity below which new content is
	// considered novel and should be added. Default: 0.3.
	AddThreshold float64

	// UpdateThreshold is the minimum similarity above which an existing entry
	// should be updated with the new content. Default: 0.7.
	UpdateThreshold float64

	// DeleteUtilityThreshold is the maximum retrieval frequency below which
	// a highly similar but rarely used entry should be deleted. Default: 1.
	DeleteUtilityThreshold int

	// MaxStoreSize is the store size above which low-utility entries become
	// candidates for deletion. Default: 100.
	MaxStoreSize int
}

// NewHeuristicPolicy creates a HeuristicPolicy with sensible defaults.
func NewHeuristicPolicy() *HeuristicPolicy {
	return &HeuristicPolicy{
		AddThreshold:           0.3,
		UpdateThreshold:        0.7,
		DeleteUtilityThreshold: 1,
		MaxStoreSize:           100,
	}
}

// Decide implements Decider using rule-based heuristics.
//
// Decision logic:
//   - If MaxSimilarity < AddThreshold: ActionAdd (novel content)
//   - If MaxSimilarity >= UpdateThreshold and HasMatchingEntry: ActionUpdate
//   - If store is large, entry is old, and retrieval frequency is low: ActionDelete
//   - Otherwise: ActionNoop (content is somewhat similar but not worth updating)
func (p *HeuristicPolicy) Decide(_ context.Context, f PolicyFeatures) (MemoryAction, float64, error) {
	// Novel content: add it.
	if f.MaxSimilarity < p.AddThreshold {
		confidence := 1.0 - f.MaxSimilarity/p.AddThreshold
		return ActionAdd, clampConfidence(confidence), nil
	}

	// Highly similar existing entry: update it.
	if f.MaxSimilarity >= p.UpdateThreshold && f.HasMatchingEntry {
		confidence := 0.0
		if denom := 1.0 - p.UpdateThreshold; denom > 0 {
			confidence = (f.MaxSimilarity - p.UpdateThreshold) / denom
		} else {
			// UpdateThreshold is 1.0 (or higher); f.MaxSimilarity must equal it
			// to reach this branch, so confidence is maximum.
			confidence = 1.0
		}
		return ActionUpdate, clampConfidence(confidence), nil
	}

	// Store is getting large and the matching entry has low utility: delete.
	if int(f.StoreSize) > p.MaxStoreSize && f.RetrievalFrequency <= p.DeleteUtilityThreshold && f.EntryAge > 0.8 {
		confidence := f.EntryAge * (1.0 - float64(f.RetrievalFrequency)/float64(p.DeleteUtilityThreshold+1))
		return ActionDelete, clampConfidence(confidence), nil
	}

	// Default: content is somewhat similar, not worth any action.
	confidence := f.MeanSimilarity
	return ActionNoop, clampConfidence(confidence), nil
}

// clampConfidence ensures the confidence value is within [0, 1].
func clampConfidence(c float64) float64 {
	return math.Max(0, math.Min(1, c))
}

// Compile-time interface check.
var _ Decider = (*HeuristicPolicy)(nil)
