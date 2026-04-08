package consolidation

import (
	"math"
	"time"
)

// DefaultHalfLife is the default half-life for recency decay: 7 days.
const DefaultHalfLife = 7 * 24 * time.Hour

// RecencyScore computes an exponential decay score for a record based on
// elapsed time since creation and a configurable half-life. The returned
// value is in [0, 1], where 1 means "just created" and values approach 0
// as age increases.
//
//	score = exp(-ln(2) * elapsed / halfLife)
func RecencyScore(createdAt time.Time, now time.Time, halfLife time.Duration) float64 {
	if halfLife <= 0 {
		halfLife = DefaultHalfLife
	}
	elapsed := now.Sub(createdAt)
	if elapsed <= 0 {
		return 1.0
	}
	return math.Exp(-math.Ln2 * float64(elapsed) / float64(halfLife))
}

// CompositeScore computes a weighted sum of the utility dimensions using the
// given weights. Weights are normalised so the result is in [0, 1] when all
// individual dimensions are in [0, 1].
func CompositeScore(u UtilityScore, w Weights) float64 {
	total := w.Recency + w.Importance + w.Relevance + w.EmotionalSalience
	if total <= 0 {
		return 0
	}
	score := (w.Recency*u.Recency +
		w.Importance*u.Importance +
		w.Relevance*u.Relevance +
		w.EmotionalSalience*u.EmotionalSalience) / total
	return clamp01(score)
}

// clamp01 restricts v to the range [0, 1].
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
