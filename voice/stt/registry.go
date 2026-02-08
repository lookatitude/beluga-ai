package stt

import (
	"fmt"
	"sort"
	"sync"
)

// Factory creates an STT engine from a Config. Each provider registers a
// Factory via Register in its init() function.
type Factory func(cfg Config) (STT, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds a named STT factory to the global registry. It is intended to
// be called from provider init() functions. Register panics if name is empty
// or already registered.
func Register(name string, f Factory) {
	if name == "" {
		panic("stt: Register called with empty name")
	}
	if f == nil {
		panic("stt: Register called with nil factory for " + name)
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, dup := registry[name]; dup {
		panic("stt: Register called twice for " + name)
	}
	registry[name] = f
}

// New creates an STT engine by looking up the named factory in the registry.
func New(name string, cfg Config) (STT, error) {
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("stt: unknown provider %q (registered: %v)", name, List())
	}
	return f(cfg)
}

// List returns the sorted names of all registered STT providers.
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
