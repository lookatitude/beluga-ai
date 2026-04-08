package cognitive

import (
	"fmt"
	"sort"
	"sync"
)

// ScorerConfig holds configuration for creating a ComplexityScorer via the
// registry.
type ScorerConfig struct {
	// Extra holds scorer-specific configuration.
	Extra map[string]any
}

// ScorerFactory is a constructor function for creating a ComplexityScorer.
type ScorerFactory func(cfg ScorerConfig) (ComplexityScorer, error)

var (
	scorerMu       sync.RWMutex
	scorerRegistry = make(map[string]ScorerFactory)
)

// RegisterScorer registers a scorer factory under the given name.
// This is typically called from init() in scorer implementation files.
func RegisterScorer(name string, factory ScorerFactory) {
	scorerMu.Lock()
	defer scorerMu.Unlock()
	scorerRegistry[name] = factory
}

// NewScorer creates a new ComplexityScorer by looking up the registered factory.
func NewScorer(name string, cfg ScorerConfig) (ComplexityScorer, error) {
	scorerMu.RLock()
	factory, ok := scorerRegistry[name]
	scorerMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("cognitive: scorer %q not registered (registered: %v)", name, ListScorers())
	}
	return factory(cfg)
}

// ListScorers returns the sorted names of all registered scorers.
func ListScorers() []string {
	scorerMu.RLock()
	defer scorerMu.RUnlock()

	names := make([]string, 0, len(scorerRegistry))
	for name := range scorerRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func init() {
	RegisterScorer("heuristic", func(_ ScorerConfig) (ComplexityScorer, error) {
		return NewHeuristicScorer(), nil
	})
}
