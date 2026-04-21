package degradation

import (
	"log/slog"
	"sort"
)

// PolicyEvaluator evaluates a severity score and determines the
// appropriate autonomy level for agent operation.
type PolicyEvaluator interface {
	// Evaluate maps a severity score in [0.0, 1.0] to an AutonomyLevel.
	Evaluate(severity float64) AutonomyLevel
}

// DefaultThresholds defines the default severity thresholds for level
// transitions. These are tuned for a balance between safety and usability.
var DefaultThresholds = LevelThresholds{
	Restricted:  0.3,
	ReadOnly:    0.6,
	Sequestered: 0.85,
}

// LevelThresholds maps each degraded autonomy level to the minimum severity
// score that triggers it. The Full level is implicit when severity is below
// the Restricted threshold.
//
// Thresholds must be monotonically non-decreasing:
//
//	0 <= Restricted <= ReadOnly <= Sequestered <= 1
//
// NewThresholdPolicy validates this invariant at construction time.
type LevelThresholds struct {
	// Restricted is the minimum severity to enter restricted mode.
	Restricted float64

	// ReadOnly is the minimum severity to enter read-only mode.
	ReadOnly float64

	// Sequestered is the minimum severity to enter sequestered mode.
	Sequestered float64
}

// ThresholdPolicy is a PolicyEvaluator that maps severity ranges to
// autonomy levels using configurable thresholds.
type ThresholdPolicy struct {
	thresholds LevelThresholds
}

// Compile-time interface check.
var _ PolicyEvaluator = (*ThresholdPolicy)(nil)

// PolicyOption configures a ThresholdPolicy.
type PolicyOption func(*ThresholdPolicy)

// WithLevelThresholds sets custom severity thresholds for level transitions.
func WithLevelThresholds(t LevelThresholds) PolicyOption {
	return func(p *ThresholdPolicy) {
		p.thresholds = t
	}
}

// NewThresholdPolicy creates a ThresholdPolicy with the given options.
// Without options, DefaultThresholds are used.
//
// Thresholds are normalised at construction time: values outside [0,1] are
// clamped, and non-monotonic configurations (e.g. Restricted=0.8,
// ReadOnly=0.3) are sorted into ascending order so that Evaluate always
// tests thresholds in a well-defined sequence. A warning is logged when
// normalisation adjusts the provided values so misconfiguration is visible
// without breaking callers.
func NewThresholdPolicy(opts ...PolicyOption) *ThresholdPolicy {
	p := &ThresholdPolicy{
		thresholds: DefaultThresholds,
	}
	for _, opt := range opts {
		opt(p)
	}
	p.thresholds = normalizeThresholds(p.thresholds)
	return p
}

// normalizeThresholds clamps values to [0,1] and sorts them into ascending
// order so that the invariant Restricted <= ReadOnly <= Sequestered holds
// regardless of how the LevelThresholds struct was populated.
func normalizeThresholds(t LevelThresholds) LevelThresholds {
	orig := t
	clamp := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
	}
	values := []float64{clamp(t.Restricted), clamp(t.ReadOnly), clamp(t.Sequestered)}
	sort.Float64s(values)
	normalized := LevelThresholds{
		Restricted:  values[0],
		ReadOnly:    values[1],
		Sequestered: values[2],
	}
	if normalized != orig {
		slog.Default().Warn("degradation: threshold configuration normalized",
			"original", orig,
			"normalized", normalized,
		)
	}
	return normalized
}

// Evaluate returns the autonomy level corresponding to the given severity
// score. Higher severity values produce more restrictive levels.
func (p *ThresholdPolicy) Evaluate(severity float64) AutonomyLevel {
	switch {
	case severity >= p.thresholds.Sequestered:
		return Sequestered
	case severity >= p.thresholds.ReadOnly:
		return ReadOnly
	case severity >= p.thresholds.Restricted:
		return Restricted
	default:
		return Full
	}
}
