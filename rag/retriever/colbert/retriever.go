package colbert

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/rag/retriever"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	retriever.Register("colbert", func(_ config.ProviderConfig) (retriever.Retriever, error) {
		return nil, fmt.Errorf("colbert: use colbert.NewColBERTRetriever() with WithEmbedder and WithIndex options")
	})
}

// ColBERTRetriever implements [retriever.Retriever] using late interaction
// retrieval. It encodes queries into per-token embeddings via a
// [embedding.MultiVectorEmbedder] and searches a [ColBERTIndex] using MaxSim
// scoring.
type ColBERTRetriever struct {
	embedder embedding.MultiVectorEmbedder
	index    ColBERTIndex
	topK     int
	hooks    retriever.Hooks
}

// Compile-time check that ColBERTRetriever implements retriever.Retriever.
var _ retriever.Retriever = (*ColBERTRetriever)(nil)

// ColBERTOption configures a ColBERTRetriever.
type ColBERTOption func(*ColBERTRetriever)

// WithEmbedder sets the multi-vector embedder used to encode queries into
// per-token embeddings.
func WithEmbedder(e embedding.MultiVectorEmbedder) ColBERTOption {
	return func(r *ColBERTRetriever) {
		r.embedder = e
	}
}

// WithIndex sets the ColBERT index used to search pre-indexed document
// embeddings.
func WithIndex(idx ColBERTIndex) ColBERTOption {
	return func(r *ColBERTRetriever) {
		r.index = idx
	}
}

// WithTopK sets the default maximum number of documents to return. This can be
// overridden per-call via [retriever.WithTopK].
func WithTopK(k int) ColBERTOption {
	return func(r *ColBERTRetriever) {
		r.topK = k
	}
}

// WithHooks sets retriever hooks on the ColBERTRetriever.
func WithHooks(h retriever.Hooks) ColBERTOption {
	return func(r *ColBERTRetriever) {
		r.hooks = h
	}
}

// NewColBERTRetriever creates a new ColBERT-style late interaction retriever.
// At minimum, [WithEmbedder] and [WithIndex] must be provided.
func NewColBERTRetriever(opts ...ColBERTOption) (*ColBERTRetriever, error) {
	r := &ColBERTRetriever{
		topK: 10,
	}
	for _, o := range opts {
		o(r)
	}
	if r.embedder == nil {
		return nil, fmt.Errorf("colbert: embedder is required (use WithEmbedder)")
	}
	if r.index == nil {
		return nil, fmt.Errorf("colbert: index is required (use WithIndex)")
	}
	return r, nil
}

// Retrieve encodes the query into per-token embeddings and searches the
// ColBERT index for the most similar documents by MaxSim score. Results are
// returned in decreasing order of relevance.
func (r *ColBERTRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
	cfg := retriever.ApplyOptions(opts...)
	topK := cfg.TopK
	if topK == 0 || topK > r.topK {
		topK = r.topK
	}

	if r.hooks.BeforeRetrieve != nil {
		if err := r.hooks.BeforeRetrieve(ctx, query); err != nil {
			return nil, err
		}
	}

	// Encode query into per-token embeddings.
	queryEmbeddings, err := r.embedder.EmbedMulti(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("colbert: embed query: %w", err)
	}
	if len(queryEmbeddings) == 0 {
		return nil, fmt.Errorf("colbert: embedder returned no embeddings for query")
	}
	queryVecs := queryEmbeddings[0]

	// Search the index.
	results, err := r.index.Search(ctx, queryVecs, topK)
	if err != nil {
		return nil, fmt.Errorf("colbert: index search: %w", err)
	}

	// Convert index results to schema.Document. The index only stores IDs
	// and scores; content must be populated by a downstream enrichment step
	// or by the caller.
	docs := make([]schema.Document, 0, len(results))
	for _, res := range results {
		if cfg.Threshold > 0 && res.Score < cfg.Threshold {
			continue
		}
		docs = append(docs, schema.Document{
			ID:    res.ID,
			Score: res.Score,
		})
	}

	if r.hooks.AfterRetrieve != nil {
		r.hooks.AfterRetrieve(ctx, docs, nil)
	}

	return docs, nil
}
