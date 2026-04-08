package rl

import (
	"context"
	"testing"
)

func TestHeuristicPolicy_Decide(t *testing.T) {
	policy := NewHeuristicPolicy()
	ctx := context.Background()

	tests := []struct {
		name       string
		features   PolicyFeatures
		wantAction MemoryAction
		wantConfGt float64 // confidence > this value
		wantConfLe float64 // confidence <= this value
	}{
		{
			name: "novel content is added",
			features: PolicyFeatures{
				MaxSimilarity: 0.1,
			},
			wantAction: ActionAdd,
			wantConfGt: 0.0,
			wantConfLe: 1.0,
		},
		{
			name: "zero similarity is added with max confidence",
			features: PolicyFeatures{
				MaxSimilarity: 0.0,
			},
			wantAction: ActionAdd,
			wantConfGt: 0.9,
			wantConfLe: 1.0,
		},
		{
			name: "highly similar with match is updated",
			features: PolicyFeatures{
				MaxSimilarity:    0.9,
				HasMatchingEntry: true,
			},
			wantAction: ActionUpdate,
			wantConfGt: 0.0,
			wantConfLe: 1.0,
		},
		{
			name: "large store old low utility is deleted",
			features: PolicyFeatures{
				MaxSimilarity:      0.5,
				StoreSize:          200,
				EntryAge:           0.95,
				RetrievalFrequency: 0,
			},
			wantAction: ActionDelete,
			wantConfGt: 0.0,
			wantConfLe: 1.0,
		},
		{
			name: "medium similarity no match is noop",
			features: PolicyFeatures{
				MaxSimilarity:    0.5,
				MeanSimilarity:   0.4,
				HasMatchingEntry: false,
			},
			wantAction: ActionNoop,
			wantConfGt: -0.1,
			wantConfLe: 1.0,
		},
		{
			name: "high similarity but no matching entry is noop",
			features: PolicyFeatures{
				MaxSimilarity:    0.8,
				MeanSimilarity:   0.6,
				HasMatchingEntry: false,
			},
			wantAction: ActionNoop,
			wantConfGt: -0.1,
			wantConfLe: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, confidence, err := policy.Decide(ctx, tt.features)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if action != tt.wantAction {
				t.Errorf("action = %v, want %v", action, tt.wantAction)
			}
			if confidence <= tt.wantConfGt || confidence > tt.wantConfLe {
				t.Errorf("confidence = %v, want (%v, %v]", confidence, tt.wantConfGt, tt.wantConfLe)
			}
		})
	}
}

func TestHeuristicPolicy_ConfidenceClamped(t *testing.T) {
	policy := NewHeuristicPolicy()
	ctx := context.Background()

	// Test with edge case values that might produce out-of-range confidence.
	features := PolicyFeatures{MaxSimilarity: -1.0}
	_, confidence, err := policy.Decide(ctx, features)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if confidence < 0 || confidence > 1 {
		t.Errorf("confidence %v out of [0,1] range", confidence)
	}
}

func TestHeuristicPolicy_ContextCancelled(t *testing.T) {
	policy := NewHeuristicPolicy()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// HeuristicPolicy doesn't use context, so should still work.
	action, _, err := policy.Decide(ctx, PolicyFeatures{MaxSimilarity: 0.1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != ActionAdd {
		t.Errorf("expected ActionAdd, got %v", action)
	}
}

func TestNewHeuristicPolicy_Defaults(t *testing.T) {
	p := NewHeuristicPolicy()
	if p.AddThreshold != 0.3 {
		t.Errorf("AddThreshold = %v, want 0.3", p.AddThreshold)
	}
	if p.UpdateThreshold != 0.7 {
		t.Errorf("UpdateThreshold = %v, want 0.7", p.UpdateThreshold)
	}
	if p.DeleteUtilityThreshold != 1 {
		t.Errorf("DeleteUtilityThreshold = %v, want 1", p.DeleteUtilityThreshold)
	}
	if p.MaxStoreSize != 100 {
		t.Errorf("MaxStoreSize = %v, want 100", p.MaxStoreSize)
	}
}
