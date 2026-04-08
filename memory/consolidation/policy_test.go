package consolidation

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThresholdPolicy_Evaluate(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		policy  *ThresholdPolicy
		records []Record
		want    []Action
	}{
		{
			name:   "high utility kept",
			policy: NewThresholdPolicy(),
			records: []Record{
				{
					ID: "1", CreatedAt: now,
					Utility: UtilityScore{Importance: 1.0, Relevance: 1.0, EmotionalSalience: 1.0},
				},
			},
			want: []Action{ActionKeep},
		},
		{
			name:   "low utility pruned",
			policy: NewThresholdPolicy(),
			records: []Record{
				{
					ID: "1", CreatedAt: now.Add(-365 * 24 * time.Hour),
					Utility: UtilityScore{Importance: 0.0, Relevance: 0.0, EmotionalSalience: 0.0},
				},
			},
			want: []Action{ActionPrune},
		},
		{
			name: "mid utility compressed",
			policy: &ThresholdPolicy{
				Threshold:         0.5,
				CompressThreshold: 0.2,
				Weights:           DefaultWeights(),
			},
			records: []Record{
				{
					ID: "1", CreatedAt: now,
					Utility: UtilityScore{Importance: 0.1, Relevance: 0.1, EmotionalSalience: 0.1},
				},
			},
			want: []Action{ActionCompress},
		},
		{
			name:    "empty records",
			policy:  NewThresholdPolicy(),
			records: nil,
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decisions, err := tt.policy.Evaluate(context.Background(), tt.records)
			require.NoError(t, err)
			if tt.want == nil {
				assert.Empty(t, decisions)
				return
			}
			require.Len(t, decisions, len(tt.want))
			for i, d := range decisions {
				assert.Equal(t, tt.want[i], d.Action, "record %d", i)
			}
		})
	}
}

func TestThresholdPolicy_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	records := make([]Record, 100)
	for i := range records {
		records[i] = Record{ID: "r"}
	}

	_, err := NewThresholdPolicy().Evaluate(ctx, records)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestFrequencyPolicy_Evaluate(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		policy  *FrequencyPolicy
		records []Record
		want    []Action
	}{
		{
			name:   "accessed record kept",
			policy: NewFrequencyPolicy(),
			records: []Record{
				{
					ID: "1", CreatedAt: now.Add(-60 * 24 * time.Hour),
					Utility: UtilityScore{AccessCount: 5},
				},
			},
			want: []Action{ActionKeep},
		},
		{
			name:   "never accessed old record pruned",
			policy: NewFrequencyPolicy(),
			records: []Record{
				{
					ID: "1", CreatedAt: now.Add(-60 * 24 * time.Hour),
					Utility: UtilityScore{AccessCount: 0},
				},
			},
			want: []Action{ActionPrune},
		},
		{
			name:   "never accessed recent record kept",
			policy: NewFrequencyPolicy(),
			records: []Record{
				{
					ID: "1", CreatedAt: now.Add(-time.Hour),
					Utility: UtilityScore{AccessCount: 0},
				},
			},
			want: []Action{ActionKeep},
		},
		{
			name:   "zero TTL uses default 30 days",
			policy: &FrequencyPolicy{TTL: 0},
			records: []Record{
				{
					ID: "1", CreatedAt: now.Add(-31 * 24 * time.Hour),
					Utility: UtilityScore{AccessCount: 0},
				},
			},
			want: []Action{ActionPrune},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decisions, err := tt.policy.Evaluate(context.Background(), tt.records)
			require.NoError(t, err)
			require.Len(t, decisions, len(tt.want))
			for i, d := range decisions {
				assert.Equal(t, tt.want[i], d.Action, "record %d", i)
			}
		})
	}
}

func TestFrequencyPolicy_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	records := make([]Record, 100)
	for i := range records {
		records[i] = Record{ID: "r"}
	}

	_, err := NewFrequencyPolicy().Evaluate(ctx, records)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestCompositePolicy_Evaluate(t *testing.T) {
	now := time.Now()

	t.Run("most severe action wins", func(t *testing.T) {
		// ThresholdPolicy will keep (high utility), FrequencyPolicy will prune (old, no access).
		threshold := NewThresholdPolicy()
		freq := &FrequencyPolicy{TTL: 24 * time.Hour}

		composite := NewCompositePolicy(threshold, freq)
		records := []Record{
			{
				ID: "1", CreatedAt: now.Add(-48 * time.Hour),
				Utility: UtilityScore{
					Importance: 1.0, Relevance: 1.0, EmotionalSalience: 1.0,
					AccessCount: 0,
				},
			},
		}

		decisions, err := composite.Evaluate(context.Background(), records)
		require.NoError(t, err)
		require.Len(t, decisions, 1)
		assert.Equal(t, ActionPrune, decisions[0].Action)
	})

	t.Run("empty records", func(t *testing.T) {
		composite := NewCompositePolicy(NewThresholdPolicy())
		decisions, err := composite.Evaluate(context.Background(), nil)
		require.NoError(t, err)
		assert.Nil(t, decisions)
	})

	t.Run("no policies keeps all", func(t *testing.T) {
		composite := NewCompositePolicy()
		records := []Record{{ID: "1", CreatedAt: now}}
		decisions, err := composite.Evaluate(context.Background(), records)
		require.NoError(t, err)
		require.Len(t, decisions, 1)
		assert.Equal(t, ActionKeep, decisions[0].Action)
	})
}
