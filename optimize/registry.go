package optimize

import (
	"fmt"
	"sort"
	"sync"
)

// OptimizerConfig holds configuration for creating an optimizer via the registry.
type OptimizerConfig struct {
	// LLM is the language model the optimizer will use for generation.
	LLM LLMClient
	// Extra holds optimizer-specific configuration.
	Extra map[string]any
}

// OptimizerFactory is a constructor function for creating an Optimizer from config.
type OptimizerFactory func(cfg OptimizerConfig) (Optimizer, error)

var (
	optimizerMu       sync.RWMutex
	optimizerRegistry = make(map[string]OptimizerFactory)
)

// RegisterOptimizer registers an optimizer factory under the given name.
// This is typically called from init() in optimizer implementation files.
func RegisterOptimizer(name string, factory OptimizerFactory) {
	optimizerMu.Lock()
	defer optimizerMu.Unlock()
	optimizerRegistry[name] = factory
}

// NewOptimizer creates a new optimizer by looking up the registered factory.
func NewOptimizer(name string, cfg OptimizerConfig) (Optimizer, error) {
	optimizerMu.RLock()
	factory, ok := optimizerRegistry[name]
	optimizerMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("optimizer %q not registered (available: %v)", name, ListOptimizers())
	}
	return factory(cfg)
}

// ListOptimizers returns the sorted names of all registered optimizers.
func ListOptimizers() []string {
	optimizerMu.RLock()
	defer optimizerMu.RUnlock()

	names := make([]string, 0, len(optimizerRegistry))
	for name := range optimizerRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// MustOptimizer creates an optimizer or panics if registration fails.
func MustOptimizer(name string, cfg OptimizerConfig) Optimizer {
	opt, err := NewOptimizer(name, cfg)
	if err != nil {
		panic(err)
	}
	return opt
}
