package inmemory

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

func init() {
	// Register inmemory provider with the global registry
	vectorstores.GetRegistry().Register("inmemory", func(ctx context.Context, config vectorstoresiface.Config) (vectorstores.VectorStore, error) {
		// Convert vectorstoresiface.Config to local Config
		localConfig := Config{
			Embedder:       config.Embedder,
			SearchK:        config.SearchK,
			ScoreThreshold: config.ScoreThreshold,
		}
		store, err := NewInMemoryVectorStoreFromConfig(ctx, localConfig)
		if err != nil {
			return nil, err
		}
		// Wrap to ensure Option type compatibility
		return &vectorStoreWrapper{store: store.(*InMemoryVectorStore)}, nil
	})
}

// vectorStoreWrapper wraps InMemoryVectorStore to ensure Option type compatibility
type vectorStoreWrapper struct {
	store *InMemoryVectorStore
}

func (w *vectorStoreWrapper) AddDocuments(ctx context.Context, documents []schema.Document, opts ...vectorstores.Option) ([]string, error) {
	// Convert vectorstores.Option to local Option
	localOpts := make([]Option, len(opts))
	for i, opt := range opts {
		localOpts[i] = func(c *Config) {
			cfg := &vectorstoresiface.Config{
				Embedder:       c.Embedder,
				SearchK:        c.SearchK,
				ScoreThreshold: c.ScoreThreshold,
			}
			opt(cfg)
			c.Embedder = cfg.Embedder
			c.SearchK = cfg.SearchK
			c.ScoreThreshold = cfg.ScoreThreshold
		}
	}
	return w.store.AddDocuments(ctx, documents, localOpts...)
}

func (w *vectorStoreWrapper) DeleteDocuments(ctx context.Context, ids []string, opts ...vectorstores.Option) error {
	localOpts := make([]Option, len(opts))
	for i, opt := range opts {
		localOpts[i] = func(c *Config) {
			cfg := &vectorstoresiface.Config{
				Embedder:       c.Embedder,
				SearchK:        c.SearchK,
				ScoreThreshold: c.ScoreThreshold,
			}
			opt(cfg)
			c.Embedder = cfg.Embedder
			c.SearchK = cfg.SearchK
			c.ScoreThreshold = cfg.ScoreThreshold
		}
	}
	return w.store.DeleteDocuments(ctx, ids, localOpts...)
}

func (w *vectorStoreWrapper) SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...vectorstores.Option) ([]schema.Document, []float32, error) {
	localOpts := make([]Option, len(opts))
	for i, opt := range opts {
		localOpts[i] = func(c *Config) {
			cfg := &vectorstoresiface.Config{
				Embedder:       c.Embedder,
				SearchK:        c.SearchK,
				ScoreThreshold: c.ScoreThreshold,
			}
			opt(cfg)
			c.Embedder = cfg.Embedder
			c.SearchK = cfg.SearchK
			c.ScoreThreshold = cfg.ScoreThreshold
		}
	}
	return w.store.SimilaritySearch(ctx, queryVector, k, localOpts...)
}

func (w *vectorStoreWrapper) SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder vectorstores.Embedder, opts ...vectorstores.Option) ([]schema.Document, []float32, error) {
	localOpts := make([]Option, len(opts))
	for i, opt := range opts {
		localOpts[i] = func(c *Config) {
			cfg := &vectorstoresiface.Config{
				Embedder:       c.Embedder,
				SearchK:        c.SearchK,
				ScoreThreshold: c.ScoreThreshold,
			}
			opt(cfg)
			c.Embedder = cfg.Embedder
			c.SearchK = cfg.SearchK
			c.ScoreThreshold = cfg.ScoreThreshold
		}
	}
	return w.store.SimilaritySearchByQuery(ctx, query, k, embedder, localOpts...)
}

func (w *vectorStoreWrapper) AsRetriever(opts ...vectorstores.Option) vectorstores.Retriever {
	localOpts := make([]Option, len(opts))
	for i, opt := range opts {
		localOpts[i] = func(c *Config) {
			cfg := &vectorstoresiface.Config{
				Embedder:       c.Embedder,
				SearchK:        c.SearchK,
				ScoreThreshold: c.ScoreThreshold,
			}
			opt(cfg)
			c.Embedder = cfg.Embedder
			c.SearchK = cfg.SearchK
			c.ScoreThreshold = cfg.ScoreThreshold
		}
	}
	return w.store.AsRetriever(localOpts...)
}

func (w *vectorStoreWrapper) GetName() string {
	return w.store.GetName()
}
