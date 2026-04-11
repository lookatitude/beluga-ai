package audit

import (
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/core"
)

// Config holds configuration used to construct a Store via the registry.
type Config struct {
	// Extra carries provider-specific configuration.
	Extra map[string]any
}

// Factory is a constructor function that creates a Store from Config.
type Factory func(cfg Config) (Store, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds a named Store factory to the global registry. It is typically
// called from an init() function. Register panics if name is empty or if the
// same name is registered twice.
func Register(name string, f Factory) {
	if name == "" {
		panic("audit: Register called with empty name")
	}
	if f == nil {
		panic("audit: Register called with nil factory for " + name)
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, dup := registry[name]; dup {
		panic("audit: Register called twice for " + name)
	}
	registry[name] = f
}

// New creates a Store by looking up the named factory in the registry and
// invoking it with cfg. It returns an error if no factory is registered under
// name.
func New(name string, cfg Config) (Store, error) {
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()

	if !ok {
		return nil, core.Errorf(core.ErrNotFound, "audit: store %q not registered (registered: %v)", name, List())
	}
	return f(cfg)
}

// List returns the sorted names of all registered Store factories.
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
