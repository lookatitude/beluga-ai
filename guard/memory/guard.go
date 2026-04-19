package memory

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// MemoryGuard orchestrates multiple AnomalyDetectors in a pipeline to detect
// memory poisoning attempts. It runs all detectors and aggregates their
// results, flagging content when the maximum score exceeds a configurable
// threshold.
type MemoryGuard struct {
	detectors []AnomalyDetector
	threshold float64
	hooks     Hooks
}

// GuardOption configures a MemoryGuard.
type GuardOption func(*MemoryGuard)

// WithDetectors sets the anomaly detectors for the guard pipeline.
func WithDetectors(detectors ...AnomalyDetector) GuardOption {
	return func(g *MemoryGuard) {
		g.detectors = detectors
	}
}

// WithHooks sets the hooks for guard events.
func WithHooks(hooks Hooks) GuardOption {
	return func(g *MemoryGuard) {
		g.hooks = hooks
	}
}

// WithThreshold sets the anomaly score threshold (0.0 to 1.0) above which
// content is flagged as poisoned. Default is 0.5.
func WithThreshold(t float64) GuardOption {
	return func(g *MemoryGuard) {
		if t > 0 && t <= 1.0 {
			g.threshold = t
		}
	}
}

// NewMemoryGuard creates a MemoryGuard with the given options. If no
// detectors are provided, a default set is used.
func NewMemoryGuard(opts ...GuardOption) *MemoryGuard {
	g := &MemoryGuard{
		threshold: 0.5,
	}
	for _, opt := range opts {
		opt(g)
	}
	if len(g.detectors) == 0 {
		g.detectors = []AnomalyDetector{
			&EntropyDetector{},
			&PatternDetector{},
			&SizeDetector{},
		}
	}
	return g
}

// GuardResult contains the aggregated result of running all detectors.
type GuardResult struct {
	// Blocked is true when the content should be rejected.
	Blocked bool

	// MaxScore is the highest anomaly score across all detectors.
	MaxScore float64

	// Results contains individual detector results.
	Results []AnomalyResult
}

// Check runs all configured detectors against the content and returns an
// aggregated result. Context cancellation is respected between detector runs.
func (g *MemoryGuard) Check(ctx context.Context, content string) (GuardResult, error) {
	var results []AnomalyResult
	var maxScore float64

	for _, d := range g.detectors {
		// Respect context cancellation.
		select {
		case <-ctx.Done():
			return GuardResult{}, ctx.Err()
		default:
		}

		result, err := d.Detect(ctx, content)
		if err != nil {
			return GuardResult{}, core.NewError(
				"guard/memory.Check",
				core.ErrToolFailed,
				"detector "+d.Name()+" failed",
				err,
			)
		}

		results = append(results, result)
		if result.Score > maxScore {
			maxScore = result.Score
		}
	}

	blocked := maxScore >= g.threshold

	// Collect flagged results for the hook.
	if blocked && g.hooks.OnPoisoningDetected != nil {
		var flagged []AnomalyResult
		for _, r := range results {
			if r.Detected {
				flagged = append(flagged, r)
			}
		}
		if len(flagged) > 0 {
			g.hooks.OnPoisoningDetected(ctx, content, flagged)
		}
	}

	return GuardResult{
		Blocked:  blocked,
		MaxScore: maxScore,
		Results:  results,
	}, nil
}

// Detectors returns the configured detectors. This is useful for inspection
// and testing.
func (g *MemoryGuard) Detectors() []AnomalyDetector {
	return g.detectors
}

// Threshold returns the configured score threshold.
func (g *MemoryGuard) Threshold() float64 {
	return g.threshold
}
