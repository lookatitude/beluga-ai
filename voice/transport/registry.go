package transport

import (
	"fmt"
	"sort"
	"sync"
)

// Factory creates an AudioTransport from a Config. Each provider registers a
// Factory via Register in its init() function.
type Factory func(cfg Config) (AudioTransport, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds a named transport factory to the global registry. It is
// intended to be called from provider init() functions. Register panics if
// name is empty or already registered.
func Register(name string, f Factory) {
	if name == "" {
		panic("transport: Register called with empty name")
	}
	if f == nil {
		panic("transport: Register called with nil factory for " + name)
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, dup := registry[name]; dup {
		panic("transport: Register called twice for " + name)
	}
	registry[name] = f
}

// New creates an AudioTransport by looking up the named factory in the registry.
func New(name string, cfg Config) (AudioTransport, error) {
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("transport: unknown provider %q (registered: %v)", name, List())
	}
	return f(cfg)
}

// List returns the sorted names of all registered transport providers.
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
