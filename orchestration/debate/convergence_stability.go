package debate

import (
	"context"
	"fmt"
	"strings"
)

// Compile-time check.
var _ ConvergenceDetector = (*StabilityDetector)(nil)

func init() {
	RegisterDetector("stability", func(cfg map[string]any) (ConvergenceDetector, error) {
		threshold := 0.8
		if t, ok := cfg["threshold"].(float64); ok && t > 0 && t <= 1 {
			threshold = t
		}
		return NewStabilityDetector(threshold), nil
	})
}

// StabilityDetector detects convergence by comparing text similarity
// between consecutive rounds. When the similarity exceeds the configured
// threshold, the debate is considered converged.
type StabilityDetector struct {
	// Threshold is the minimum similarity (0.0 to 1.0) to declare convergence.
	Threshold float64
}

// NewStabilityDetector creates a StabilityDetector with the given threshold.
// The threshold must be between 0.0 and 1.0; values outside this range are
// clamped.
func NewStabilityDetector(threshold float64) *StabilityDetector {
	if threshold < 0 {
		threshold = 0
	}
	if threshold > 1 {
		threshold = 1
	}
	return &StabilityDetector{Threshold: threshold}
}

// Check compares the contributions of the last two rounds using bigram
// similarity. Returns converged if the average similarity exceeds the threshold.
func (d *StabilityDetector) Check(_ context.Context, state DebateState) (ConvergenceResult, error) {
	if len(state.Rounds) < 2 {
		return ConvergenceResult{
			Converged: false,
			Reason:    "not enough rounds for stability comparison",
			Score:     0.0,
		}, nil
	}

	prev := state.Rounds[len(state.Rounds)-2]
	curr := state.Rounds[len(state.Rounds)-1]

	totalSim := 0.0
	count := 0

	for _, pc := range prev.Contributions {
		for _, cc := range curr.Contributions {
			if pc.AgentID == cc.AgentID {
				totalSim += bigramSimilarity(pc.Content, cc.Content)
				count++
				break
			}
		}
	}

	if count == 0 {
		return ConvergenceResult{
			Converged: false,
			Reason:    "no matching agent contributions between rounds",
			Score:     0.0,
		}, nil
	}

	avgSim := totalSim / float64(count)
	converged := avgSim >= d.Threshold

	reason := fmt.Sprintf("stability score %.3f (threshold: %.3f)", avgSim, d.Threshold)
	if converged {
		reason = "converged: " + reason
	}

	return ConvergenceResult{
		Converged: converged,
		Reason:    reason,
		Score:     avgSim,
	}, nil
}

// bigramSimilarity computes the Dice coefficient of character bigrams
// between two strings. Returns 1.0 for identical strings and 0.0 for
// completely different strings.
func bigramSimilarity(a, b string) float64 {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))

	if len(a) < 2 || len(b) < 2 {
		return 0.0
	}
	if a == b {
		return 1.0
	}

	bigramsA := makeBigrams(a)
	bigramsB := makeBigrams(b)

	intersection := 0
	for bg, countA := range bigramsA {
		if countB, ok := bigramsB[bg]; ok {
			if countA < countB {
				intersection += countA
			} else {
				intersection += countB
			}
		}
	}

	totalA := 0
	for _, c := range bigramsA {
		totalA += c
	}
	totalB := 0
	for _, c := range bigramsB {
		totalB += c
	}

	if totalA+totalB == 0 {
		return 0.0
	}

	return 2.0 * float64(intersection) / float64(totalA+totalB)
}

// makeBigrams returns a frequency map of character bigrams in s.
func makeBigrams(s string) map[string]int {
	runes := []rune(s)
	bigrams := make(map[string]int, len(runes)-1)
	for i := 0; i < len(runes)-1; i++ {
		bg := string(runes[i : i+2])
		bigrams[bg]++
	}
	return bigrams
}
