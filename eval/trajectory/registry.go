package trajectory

import (
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/core"
)

// Factory creates a TrajectoryMetric from configuration.
type Factory func(cfg map[string]any) (TrajectoryMetric, error)

var (
	mu       sync.RWMutex
	registry = make(map[string]Factory)
)

// Register adds a named factory to the trajectory metric registry.
// It is intended to be called from init() functions.
func Register(name string, f Factory) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = f
}

// New creates a TrajectoryMetric by looking up the named factory in the
// registry. Returns an error if the name is not registered.
func New(name string, cfg map[string]any) (TrajectoryMetric, error) {
	mu.RLock()
	f, ok := registry[name]
	mu.RUnlock()
	if !ok {
		return nil, core.Errorf(core.ErrNotFound, "trajectory: unknown metric %q (registered: %v)", name, List())
	}
	return f(cfg)
}

// List returns the sorted names of all registered trajectory metrics.
func List() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
