package guard

import (
	"fmt"
	"sort"
	"sync"
)

// registry holds the named guard factories. It is populated via Register
// (typically in init functions) and consumed via New and List.
var (
	registryMu sync.RWMutex
	registry   = make(map[string]GuardFactory)
)

// Register adds a named guard factory to the global registry. It is safe to
// call from init functions. Register panics if name is empty or already
// registered.
func Register(name string, f GuardFactory) {
	if name == "" {
		panic("guard: Register called with empty name")
	}
	if f == nil {
		panic("guard: Register called with nil factory for " + name)
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, dup := registry[name]; dup {
		panic("guard: Register called twice for " + name)
	}
	registry[name] = f
}

// New creates a Guard by looking up the named factory in the registry and
// invoking it with cfg. Returns an error if the name is not registered.
func New(name string, cfg map[string]any) (Guard, error) {
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("guard: unknown guard %q", name)
	}
	return f(cfg)
}

// List returns the sorted names of all registered guard factories.
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
