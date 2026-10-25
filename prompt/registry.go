package prompt

import (
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// Config configures a PromptManager created through the registry.
type Config struct {
	// Extra holds provider-specific settings (for example "dir" for the
	// file provider).
	Extra map[string]any
}

// Factory creates a PromptManager from a Config.
type Factory func(cfg Config) (PromptManager, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds a named PromptManager factory. It is intended to be called
// from provider init() functions. Registering a name that is already
// registered replaces the previous factory (the framework-wide duplicate
// policy).
func Register(name string, f Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = f
}

// New creates a PromptManager using the factory registered under name.
func New(name string, cfg Config) (PromptManager, error) {
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, core.Errorf(core.ErrNotFound, "prompt: unknown provider %q (registered: %v)", name, List())
	}
	return f(cfg)
}

// List returns the sorted names of all registered providers.
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
