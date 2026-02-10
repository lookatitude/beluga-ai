package vectorstore

import (
	"context"

	"github.com/lookatitude/beluga-ai/schema"
)

// SearchStrategy defines the distance metric used for similarity search.
type SearchStrategy int

const (
	// Cosine uses cosine similarity (default). Range: [-1, 1] where 1 is most similar.
	Cosine SearchStrategy = iota
	// DotProduct uses dot-product similarity.
	DotProduct
	// Euclidean uses Euclidean (L2) distance. Lower values indicate more similarity.
	Euclidean
)

// String returns the string representation of the search strategy.
func (s SearchStrategy) String() string {
	switch s {
	case Cosine:
		return "cosine"
	case DotProduct:
		return "dot_product"
	case Euclidean:
		return "euclidean"
	default:
		return "unknown"
	}
}

// SearchConfig holds configuration for a similarity search operation.
type SearchConfig struct {
	// Filter restricts results to documents whose metadata matches all
	// key-value pairs in the map. A nil filter matches all documents.
	Filter map[string]any

	// Threshold sets the minimum similarity score for results. Documents
	// with scores below this threshold are excluded. Zero disables filtering.
	Threshold float64

	// Strategy selects the distance metric. Defaults to Cosine.
	Strategy SearchStrategy
}

// SearchOption configures a SearchConfig.
type SearchOption func(*SearchConfig)

// WithFilter restricts search results to documents whose metadata matches
// all key-value pairs in the filter map.
func WithFilter(filter map[string]any) SearchOption {
	return func(cfg *SearchConfig) {
		cfg.Filter = filter
	}
}

// WithThreshold sets the minimum similarity score for search results.
func WithThreshold(threshold float64) SearchOption {
	return func(cfg *SearchConfig) {
		cfg.Threshold = threshold
	}
}

// WithStrategy selects the distance metric for similarity search.
func WithStrategy(strategy SearchStrategy) SearchOption {
	return func(cfg *SearchConfig) {
		cfg.Strategy = strategy
	}
}

// VectorStore stores document embeddings and supports similarity search.
// Implementations must be safe for concurrent use.
type VectorStore interface {
	// Add inserts documents with their corresponding embeddings into the store.
	// The docs and embeddings slices must have the same length.
	Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error

	// Search finds the k most similar documents to the query vector.
	// Results are returned in descending order of similarity with their
	// Score field populated.
	Search(ctx context.Context, query []float32, k int, opts ...SearchOption) ([]schema.Document, error)

	// Delete removes documents with the given IDs from the store.
	Delete(ctx context.Context, ids []string) error
}

// Hooks provides optional callback functions invoked around vector store
// operations. All fields are optional; nil hooks are skipped.
type Hooks struct {
	// BeforeAdd is called before each Add call with the documents.
	// Returning an error aborts the call.
	BeforeAdd func(ctx context.Context, docs []schema.Document) error

	// AfterSearch is called after Search completes with the results and
	// any error.
	AfterSearch func(ctx context.Context, results []schema.Document, err error)
}

// ComposeHooks merges multiple Hooks into a single Hooks value.
// Callbacks are called in the order the hooks were provided.
// For BeforeAdd, the first error returned short-circuits.
func ComposeHooks(hooks ...Hooks) Hooks {
	return Hooks{
		BeforeAdd: func(ctx context.Context, docs []schema.Document) error {
			for _, h := range hooks {
				if h.BeforeAdd != nil {
					if err := h.BeforeAdd(ctx, docs); err != nil {
						return err
					}
				}
			}
			return nil
		},
		AfterSearch: func(ctx context.Context, results []schema.Document, err error) {
			for _, h := range hooks {
				if h.AfterSearch != nil {
					h.AfterSearch(ctx, results, err)
				}
			}
		},
	}
}
