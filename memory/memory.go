// Package memory provides the MemGPT-inspired 3-tier memory system for Beluga AI agents.
//
// The three tiers follow the MemGPT model:
//   - Core: always-in-context persona and human blocks (editable by the agent)
//   - Recall: searchable conversation history (message-level persistence)
//   - Archival: vector-based long-term storage (embedding + retrieval)
//
// Additionally, a graph memory tier provides entity-relationship storage for
// structured knowledge representation.
//
// The package follows Beluga's registry pattern — providers register via init()
// and are instantiated with New:
//
//	mem, err := memory.New("composite", cfg)
//	err = mem.Save(ctx, input, output)
//	msgs, err := mem.Load(ctx, "search query")
//
// A CompositeMemory combines all tiers into a unified Memory implementation
// that delegates to the appropriate tier based on the operation.
package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/schema"
)

// Memory is the primary interface for agent memory. Implementations provide
// the ability to save conversation turns, load relevant history, and search
// over stored documents. All methods must be safe for concurrent use.
type Memory interface {
	// Save persists an input/output message pair from a conversation turn.
	Save(ctx context.Context, input, output schema.Message) error

	// Load retrieves messages relevant to the given query. The returned
	// messages are ordered by relevance or recency depending on the
	// implementation.
	Load(ctx context.Context, query string) ([]schema.Message, error)

	// Search finds documents relevant to the given query, returning at most
	// k results. This is primarily used by the archival tier.
	Search(ctx context.Context, query string, k int) ([]schema.Document, error)

	// Clear removes all stored data from this memory instance.
	Clear(ctx context.Context) error
}

// Factory creates a Memory from a ProviderConfig. Each provider registers a
// Factory via Register in its init() function.
type Factory func(cfg config.ProviderConfig) (Memory, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds a provider factory to the global registry. It is intended to
// be called from provider init() functions. Duplicate registrations for the
// same name silently overwrite the previous factory.
func Register(name string, f Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = f
}

// New creates a Memory by looking up the provider name in the registry and
// calling its factory with the given configuration.
func New(name string, cfg config.ProviderConfig) (Memory, error) {
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("memory: unknown provider %q (registered: %v)", name, List())
	}
	return f(cfg)
}

// List returns the names of all registered providers, sorted alphabetically.
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
