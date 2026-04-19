package cost

import (
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// Config holds the configuration passed to a Tracker factory when creating a
// new instance via New.
type Config struct {
	// MaxEntries limits the number of usage records retained by in-memory
	// implementations. Zero means unlimited.
	MaxEntries int

	// Options holds provider-specific configuration key-value pairs.
	Options map[string]any
}

// Factory is a constructor function that creates a Tracker from a Config.
type Factory func(cfg Config) (Tracker, error)

var (
	mu       sync.RWMutex
	registry = make(map[string]Factory)
)

func init() {
	Register("inmemory", func(cfg Config) (Tracker, error) {
		var opts []Option
		if cfg.MaxEntries > 0 {
			opts = append(opts, WithMaxEntries(cfg.MaxEntries))
		}
		return NewInMemoryTracker(opts...), nil
	})
}

// Register adds a named Tracker factory to the global registry. It is intended
// to be called from provider init() functions. Registering a duplicate name
// overwrites the previous factory.
func Register(name string, f Factory) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = f
}

// New creates a Tracker by looking up the named factory in the registry and
// calling it with the provided Config. It returns an error if no factory is
// registered under the given name.
func New(name string, cfg Config) (Tracker, error) {
	mu.RLock()
	f, ok := registry[name]
	mu.RUnlock()
	if !ok {
		return nil, core.Errorf(core.ErrNotFound, "cost: unknown tracker %q (registered: %v)", name, List())
	}
	return f(cfg)
}

// List returns the sorted names of all registered Tracker providers.
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
