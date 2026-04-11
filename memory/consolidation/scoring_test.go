package consolidation

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRecencyScore(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		created  time.Time
		halfLife time.Duration
		wantMin  float64
		wantMax  float64
	}{
		{
			name:     "just created",
			created:  now,
			halfLife: DefaultHalfLife,
			wantMin:  0.99,
			wantMax:  1.0,
		},
		{
			name:     "one half-life ago",
			created:  now.Add(-DefaultHalfLife),
			halfLife: DefaultHalfLife,
			wantMin:  0.49,
			wantMax:  0.51,
		},
		{
			name:     "two half-lives ago",
			created:  now.Add(-2 * DefaultHalfLife),
			halfLife: DefaultHalfLife,
			wantMin:  0.24,
			wantMax:  0.26,
		},
		{
			name:     "future timestamp returns 1",
			created:  now.Add(time.Hour),
			halfLife: DefaultHalfLife,
			wantMin:  1.0,
			wantMax:  1.0,
		},
		{
			name:     "zero half-life uses default",
			created:  now.Add(-DefaultHalfLife),
			halfLife: 0,
			wantMin:  0.49,
			wantMax:  0.51,
		},
		{
			name:     "negative half-life uses default",
			created:  now.Add(-DefaultHalfLife),
			halfLife: -time.Hour,
			wantMin:  0.49,
			wantMax:  0.51,
		},
		{
			name:     "very old record approaches zero",
			created:  now.Add(-100 * DefaultHalfLife),
			halfLife: DefaultHalfLife,
			wantMin:  0.0,
			wantMax:  1e-10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RecencyScore(tt.created, now, tt.halfLife)
			assert.GreaterOrEqual(t, got, tt.wantMin, "score too low")
			assert.LessOrEqual(t, got, tt.wantMax, "score too high")
		})
	}
}

func TestCompositeScore(t *testing.T) {
	tests := []struct {
		name    string
		utility UtilityScore
		weights Weights
		want    float64
		epsilon float64
	}{
		{
			name: "all ones with default weights",
			utility: UtilityScore{
				Recency: 1.0, Importance: 1.0,
				Relevance: 1.0, EmotionalSalience: 1.0,
			},
			weights: DefaultWeights(),
			want:    1.0,
			epsilon: 1e-9,
		},
		{
			name: "all zeros",
			utility: UtilityScore{
				Recency: 0, Importance: 0,
				Relevance: 0, EmotionalSalience: 0,
			},
			weights: DefaultWeights(),
			want:    0.0,
			epsilon: 1e-9,
		},
		{
			name: "only recency with default weights",
			utility: UtilityScore{
				Recency: 1.0, Importance: 0,
				Relevance: 0, EmotionalSalience: 0,
			},
			weights: DefaultWeights(),
			want:    0.4,
			epsilon: 1e-9,
		},
		{
			name: "zero total weight returns zero",
			utility: UtilityScore{
				Recency: 1.0, Importance: 1.0,
				Relevance: 1.0, EmotionalSalience: 1.0,
			},
			weights: Weights{0, 0, 0, 0},
			want:    0.0,
			epsilon: 1e-9,
		},
		{
			name: "custom weights",
			utility: UtilityScore{
				Recency: 0.5, Importance: 0.8,
				Relevance: 0.3, EmotionalSalience: 0.9,
			},
			weights: Weights{1, 1, 1, 1},
			want:    0.625, // (0.5+0.8+0.3+0.9)/4
			epsilon: 1e-9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompositeScore(tt.utility, tt.weights)
			assert.InDelta(t, tt.want, got, tt.epsilon)
		})
	}
}

func TestClamp01(t *testing.T) {
	tests := []struct {
		in   float64
		want float64
	}{
		{-1, 0},
		{0, 0},
		{0.5, 0.5},
		{1.0, 1.0},
		{2.0, 1.0},
		{math.Inf(1), 1.0},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, clamp01(tt.in))
	}
}
