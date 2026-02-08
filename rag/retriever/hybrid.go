package retriever

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
)

// BM25Searcher performs keyword-based BM25 search. Implementations may use
// an external search engine (Elasticsearch, Typesense) or an in-memory index.
type BM25Searcher interface {
	// Search returns documents matching the query ranked by BM25 score.
	Search(ctx context.Context, query string, k int) ([]schema.Document, error)
}

// HybridRetriever combines dense vector retrieval with sparse BM25 keyword
// search using Reciprocal Rank Fusion. This is the default retrieval strategy
// in the Beluga framework.
type HybridRetriever struct {
	store    vectorstore.VectorStore
	embedder embedding.Embedder
	bm25     BM25Searcher
	rrfK     int
	hooks    Hooks
}

// HybridOption configures a HybridRetriever.
type HybridOption func(*HybridRetriever)

// WithHybridRRFK sets the RRF k parameter. Defaults to 60.
func WithHybridRRFK(k int) HybridOption {
	return func(r *HybridRetriever) {
		if k > 0 {
			r.rrfK = k
		}
	}
}

// WithHybridHooks sets hooks on the HybridRetriever.
func WithHybridHooks(h Hooks) HybridOption {
	return func(r *HybridRetriever) {
		r.hooks = h
	}
}

// NewHybridRetriever creates a hybrid retriever that combines vector search
// and BM25 keyword search using RRF fusion. If rrfK is 0, the default of 60
// is used.
func NewHybridRetriever(store vectorstore.VectorStore, embedder embedding.Embedder, bm25 BM25Searcher, opts ...HybridOption) *HybridRetriever {
	r := &HybridRetriever{
		store:    store,
		embedder: embedder,
		bm25:     bm25,
		rrfK:     60,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Retrieve performs both vector and BM25 search, then fuses results using RRF.
func (r *HybridRetriever) Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
	cfg := ApplyOptions(opts...)

	if r.hooks.BeforeRetrieve != nil {
		if err := r.hooks.BeforeRetrieve(ctx, query); err != nil {
			return nil, err
		}
	}

	// Dense retrieval: embed query and search vector store.
	vec, err := r.embedder.EmbedSingle(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("retriever: hybrid embed: %w", err)
	}

	var searchOpts []vectorstore.SearchOption
	if cfg.Metadata != nil {
		searchOpts = append(searchOpts, vectorstore.WithFilter(cfg.Metadata))
	}

	// Fetch more candidates than requested for better fusion.
	vectorK := cfg.TopK * 2
	if vectorK < 20 {
		vectorK = 20
	}

	vectorDocs, err := r.store.Search(ctx, vec, vectorK, searchOpts...)
	if err != nil {
		return nil, fmt.Errorf("retriever: hybrid vector search: %w", err)
	}

	// Sparse retrieval: BM25 search.
	bm25K := cfg.TopK * 2
	if bm25K < 20 {
		bm25K = 20
	}

	bm25Docs, err := r.bm25.Search(ctx, query, bm25K)
	if err != nil {
		return nil, fmt.Errorf("retriever: hybrid bm25 search: %w", err)
	}

	// Fuse using RRF.
	rrf := NewRRFStrategy(r.rrfK)
	fused, err := rrf.Fuse(ctx, [][]schema.Document{vectorDocs, bm25Docs})
	if err != nil {
		return nil, fmt.Errorf("retriever: hybrid fuse: %w", err)
	}

	if cfg.TopK > 0 && len(fused) > cfg.TopK {
		fused = fused[:cfg.TopK]
	}

	if r.hooks.AfterRetrieve != nil {
		r.hooks.AfterRetrieve(ctx, fused, nil)
	}

	return fused, nil
}
