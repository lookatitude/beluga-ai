package loader

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/schema"
)

// DocumentLoader loads content from a source and returns documents.
// Implementations handle specific formats (text, JSON, CSV, etc.) and
// register themselves via init().
type DocumentLoader interface {
	// Load reads content from the given source (typically a file path or URL)
	// and returns a slice of documents.
	Load(ctx context.Context, source string) ([]schema.Document, error)
}

// Transformer applies a transformation to a single document.
// Transformers are used in pipelines to enrich or modify documents
// after loading (e.g., adding metadata, cleaning content).
type Transformer interface {
	// Transform applies a transformation to the given document and returns
	// the modified document.
	Transform(ctx context.Context, doc schema.Document) (schema.Document, error)
}

// TransformerFunc is a convenience type that implements Transformer.
type TransformerFunc func(ctx context.Context, doc schema.Document) (schema.Document, error)

// Transform calls the underlying function.
func (f TransformerFunc) Transform(ctx context.Context, doc schema.Document) (schema.Document, error) {
	return f(ctx, doc)
}

// Factory creates a DocumentLoader from a ProviderConfig.
type Factory func(cfg config.ProviderConfig) (DocumentLoader, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds a loader factory to the global registry. It is intended to
// be called from provider init() functions. Duplicate registrations for the
// same name silently overwrite the previous factory.
func Register(name string, f Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = f
}

// New creates a DocumentLoader by looking up the provider name in the registry
// and calling its factory with the given configuration.
func New(name string, cfg config.ProviderConfig) (DocumentLoader, error) {
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("loader: unknown provider %q (registered: %v)", name, List())
	}
	return f(cfg)
}

// List returns the names of all registered loaders, sorted alphabetically.
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
