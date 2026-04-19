package embedding

import (
	"context"
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
)

// MultiVectorEmbedder produces per-token embeddings for late interaction
// retrieval models such as ColBERT and ColPali. Unlike [Embedder], which
// produces a single vector per text, MultiVectorEmbedder returns a slice of
// token-level vectors for each input text.
//
// Implementations must be safe for concurrent use.
type MultiVectorEmbedder interface {
	// EmbedMulti produces per-token embeddings for a batch of texts. The
	// returned slice has the same length as texts, where each element is a
	// slice of token-level float32 vectors.
	EmbedMulti(ctx context.Context, texts []string) ([][][]float32, error)

	// TokenDimensions returns the dimensionality of the per-token embedding
	// vectors produced by this embedder.
	TokenDimensions() int
}

// MultiVectorFactory creates a MultiVectorEmbedder from a ProviderConfig.
// Each provider registers a MultiVectorFactory via RegisterMultiVector in its
// init() function.
type MultiVectorFactory func(cfg config.ProviderConfig) (MultiVectorEmbedder, error)

var (
	mvRegistryMu sync.RWMutex
	mvRegistry   = make(map[string]MultiVectorFactory)
)

// RegisterMultiVector adds a multi-vector embedder factory to the global
// registry. It is intended to be called from provider init() functions.
// Duplicate registrations for the same name silently overwrite the previous
// factory.
func RegisterMultiVector(name string, f MultiVectorFactory) {
	mvRegistryMu.Lock()
	defer mvRegistryMu.Unlock()
	mvRegistry[name] = f
}

// NewMultiVector creates a MultiVectorEmbedder by looking up the provider name
// in the registry and calling its factory with the given configuration.
func NewMultiVector(name string, cfg config.ProviderConfig) (MultiVectorEmbedder, error) {
	mvRegistryMu.RLock()
	f, ok := mvRegistry[name]
	mvRegistryMu.RUnlock()
	if !ok {
		return nil, core.Errorf(core.ErrNotFound, "embedding: unknown multi-vector provider %q (registered: %v)", name, ListMultiVector())
	}
	return f(cfg)
}

// ListMultiVector returns the names of all registered multi-vector embedding
// providers, sorted alphabetically.
func ListMultiVector() []string {
	mvRegistryMu.RLock()
	defer mvRegistryMu.RUnlock()
	names := make([]string, 0, len(mvRegistry))
	for name := range mvRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
