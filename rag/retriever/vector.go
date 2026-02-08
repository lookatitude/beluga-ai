package retriever

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
)

// VectorStoreRetriever retrieves documents by embedding the query and
// performing similarity search against a VectorStore.
type VectorStoreRetriever struct {
	store    vectorstore.VectorStore
	embedder embedding.Embedder
	hooks    Hooks
}

// VectorStoreOption configures a VectorStoreRetriever.
type VectorStoreOption func(*VectorStoreRetriever)

// WithVectorStoreHooks sets hooks on the VectorStoreRetriever.
func WithVectorStoreHooks(h Hooks) VectorStoreOption {
	return func(r *VectorStoreRetriever) {
		r.hooks = h
	}
}

// NewVectorStoreRetriever creates a retriever that embeds queries and searches
// the given vector store.
func NewVectorStoreRetriever(store vectorstore.VectorStore, embedder embedding.Embedder, opts ...VectorStoreOption) *VectorStoreRetriever {
	r := &VectorStoreRetriever{
		store:    store,
		embedder: embedder,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Retrieve embeds the query, searches the vector store, and returns matching
// documents ordered by similarity.
func (r *VectorStoreRetriever) Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
	cfg := ApplyOptions(opts...)

	if r.hooks.BeforeRetrieve != nil {
		if err := r.hooks.BeforeRetrieve(ctx, query); err != nil {
			return nil, err
		}
	}

	vec, err := r.embedder.EmbedSingle(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("retriever: embed query: %w", err)
	}

	var searchOpts []vectorstore.SearchOption
	if cfg.Threshold > 0 {
		searchOpts = append(searchOpts, vectorstore.WithThreshold(cfg.Threshold))
	}
	if cfg.Metadata != nil {
		searchOpts = append(searchOpts, vectorstore.WithFilter(cfg.Metadata))
	}

	docs, err := r.store.Search(ctx, vec, cfg.TopK, searchOpts...)

	if r.hooks.AfterRetrieve != nil {
		r.hooks.AfterRetrieve(ctx, docs, err)
	}

	return docs, err
}
