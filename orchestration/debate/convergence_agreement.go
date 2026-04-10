package debate

import (
	"context"
	"fmt"
	"strings"
)

// Compile-time check.
var _ ConvergenceDetector = (*AgreementDetector)(nil)

func init() {
	RegisterDetector("agreement", func(cfg map[string]any) (ConvergenceDetector, error) {
		threshold := 0.6
		if t, ok := cfg["threshold"].(float64); ok && t > 0 && t <= 1 {
			threshold = t
		}
		return NewAgreementDetector(threshold), nil
	})
}

// AgreementDetector detects convergence when a majority of agents produce
// similar responses in the most recent round. It groups contributions by
// similarity and checks whether any group exceeds the configured fraction.
type AgreementDetector struct {
	// Threshold is the minimum fraction of agents that must agree (0.0 to 1.0).
	Threshold float64
}

// NewAgreementDetector creates an AgreementDetector with the given threshold.
// The threshold must be between 0.0 and 1.0; values outside this range are
// clamped.
func NewAgreementDetector(threshold float64) *AgreementDetector {
	if threshold < 0 {
		threshold = 0
	}
	if threshold > 1 {
		threshold = 1
	}
	return &AgreementDetector{Threshold: threshold}
}

// Check groups the most recent round's contributions by similarity and
// reports convergence if any group meets the threshold fraction.
func (d *AgreementDetector) Check(_ context.Context, state DebateState) (ConvergenceResult, error) {
	if len(state.Rounds) == 0 {
		return ConvergenceResult{
			Converged: false,
			Reason:    "no rounds completed",
			Score:     0.0,
		}, nil
	}

	lastRound := state.Rounds[len(state.Rounds)-1]
	if len(lastRound.Contributions) == 0 {
		return ConvergenceResult{
			Converged: false,
			Reason:    "no contributions in last round",
			Score:     0.0,
		}, nil
	}

	// Group by similarity: each contribution is compared to existing groups.
	// A contribution joins a group if it is >0.5 similar to the group leader.
	groups := groupBySimilarity(lastRound.Contributions, 0.5)

	maxGroupSize := 0
	for _, g := range groups {
		if len(g) > maxGroupSize {
			maxGroupSize = len(g)
		}
	}

	total := len(lastRound.Contributions)
	fraction := float64(maxGroupSize) / float64(total)
	converged := fraction >= d.Threshold

	reason := fmt.Sprintf("agreement fraction %.3f (threshold: %.3f, largest group: %d/%d)",
		fraction, d.Threshold, maxGroupSize, total)
	if converged {
		reason = "converged: " + reason
	}

	return ConvergenceResult{
		Converged: converged,
		Reason:    reason,
		Score:     fraction,
	}, nil
}

// groupBySimilarity clusters contributions into groups using average-linkage
// bigram similarity. Each candidate is placed into the first group whose
// mean similarity to all existing members meets simThreshold. This avoids
// the leader-only comparison footgun where a contribution close to later
// group members but distant from the leader would be split into a fresh
// group, under-counting agreement.
func groupBySimilarity(contributions []Contribution, simThreshold float64) [][]Contribution {
	var groups [][]Contribution

	for _, c := range contributions {
		placed := false
		content := strings.ToLower(strings.TrimSpace(c.Content))
		for i, g := range groups {
			var sum float64
			for _, member := range g {
				member := strings.ToLower(strings.TrimSpace(member.Content))
				sum += bigramSimilarity(content, member)
			}
			avg := sum / float64(len(g))
			if avg >= simThreshold {
				groups[i] = append(groups[i], c)
				placed = true
				break
			}
		}
		if !placed {
			groups = append(groups, []Contribution{c})
		}
	}

	return groups
}
