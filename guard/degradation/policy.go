package degradation

// DegradationPolicy evaluates a severity score and determines the
// appropriate autonomy level for agent operation.
type DegradationPolicy interface {
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
type LevelThresholds struct {
	// Restricted is the minimum severity to enter restricted mode.
	Restricted float64

	// ReadOnly is the minimum severity to enter read-only mode.
	ReadOnly float64

	// Sequestered is the minimum severity to enter sequestered mode.
	Sequestered float64
}

// ThresholdPolicy is a DegradationPolicy that maps severity ranges to
// autonomy levels using configurable thresholds.
type ThresholdPolicy struct {
	thresholds LevelThresholds
}

// Compile-time interface check.
var _ DegradationPolicy = (*ThresholdPolicy)(nil)

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
func NewThresholdPolicy(opts ...PolicyOption) *ThresholdPolicy {
	p := &ThresholdPolicy{
		thresholds: DefaultThresholds,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
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
