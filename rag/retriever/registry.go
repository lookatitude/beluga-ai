package retriever

import (
	"fmt"
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/config"
)

// Factory creates a Retriever from a ProviderConfig. Each implementation
// registers a Factory via Register in its init() function.
type Factory func(cfg config.ProviderConfig) (Retriever, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds a retriever factory to the global registry. It is intended to
// be called from init() functions. Duplicate registrations for the same name
// silently overwrite the previous factory.
func Register(name string, f Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = f
}

// New creates a Retriever by looking up the name in the registry and calling
// its factory with the given configuration.
func New(name string, cfg config.ProviderConfig) (Retriever, error) {
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("retriever: unknown provider %q (registered: %v)", name, List())
	}
	return f(cfg)
}

// List returns the names of all registered retriever factories, sorted
// alphabetically.
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
