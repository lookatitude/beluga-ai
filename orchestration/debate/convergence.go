package debate

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// ConvergenceDetector determines whether a debate has reached a point
// where further rounds are unlikely to produce new insights.
type ConvergenceDetector interface {
	// Check evaluates the current debate state and returns whether
	// convergence has been achieved.
	Check(ctx context.Context, state DebateState) (ConvergenceResult, error)
}

// DetectorFactory creates a ConvergenceDetector from a configuration map.
type DetectorFactory func(cfg map[string]any) (ConvergenceDetector, error)

var (
	detectorMu       sync.RWMutex
	detectorRegistry = make(map[string]DetectorFactory)
)

// RegisterDetector registers a named ConvergenceDetector factory.
// This should be called from init().
func RegisterDetector(name string, f DetectorFactory) {
	detectorMu.Lock()
	defer detectorMu.Unlock()
	detectorRegistry[name] = f
}

// NewDetector creates a ConvergenceDetector by name from the registry.
func NewDetector(name string, cfg map[string]any) (ConvergenceDetector, error) {
	detectorMu.RLock()
	f, ok := detectorRegistry[name]
	detectorMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("debate: unknown detector %q (registered: %v)", name, ListDetectors())
	}
	return f(cfg)
}

// ListDetectors returns the names of all registered detectors in sorted order.
func ListDetectors() []string {
	detectorMu.RLock()
	defer detectorMu.RUnlock()
	names := make([]string, 0, len(detectorRegistry))
	for name := range detectorRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// MaxRoundsDetector is a simple convergence detector that triggers
// when the maximum number of rounds has been reached.
type MaxRoundsDetector struct{}

// Compile-time check.
var _ ConvergenceDetector = (*MaxRoundsDetector)(nil)

func init() {
	RegisterDetector("maxrounds", func(_ map[string]any) (ConvergenceDetector, error) {
		return &MaxRoundsDetector{}, nil
	})
}

// Check returns converged when the current round equals or exceeds MaxRounds.
func (d *MaxRoundsDetector) Check(_ context.Context, state DebateState) (ConvergenceResult, error) {
	if state.CurrentRound >= state.MaxRounds-1 {
		return ConvergenceResult{
			Converged: true,
			Reason:    fmt.Sprintf("reached maximum rounds (%d)", state.MaxRounds),
			Score:     0.0,
		}, nil
	}
	return ConvergenceResult{
		Converged: false,
		Reason:    fmt.Sprintf("round %d of %d", state.CurrentRound+1, state.MaxRounds),
	}, nil
}
