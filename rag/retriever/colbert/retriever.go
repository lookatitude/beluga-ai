package colbert

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/rag/embedding"
	"github.com/lookatitude/beluga-ai/v2/rag/retriever"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

func init() {
	retriever.Register("colbert", func(_ config.ProviderConfig) (retriever.Retriever, error) {
		return nil, core.Errorf(core.ErrInvalidInput, "colbert: use colbert.NewColBERTRetriever() with WithEmbedder and WithIndex options")
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
		return nil, core.Errorf(core.ErrInvalidInput, "colbert: embedder is required (use WithEmbedder)")
	}
	if r.index == nil {
		return nil, core.Errorf(core.ErrInvalidInput, "colbert: index is required (use WithIndex)")
	}
	return r, nil
}

// Retrieve encodes the query into per-token embeddings and searches the
// ColBERT index for the most similar documents by MaxSim score. Results are
// returned in decreasing order of relevance.
func (r *ColBERTRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
	cfg := retriever.ApplyOptions(opts...)
	// Prefer the per-call TopK when explicitly positive; only fall back to the
	// retriever's configured default when no usable value was provided.
	topK := r.topK
	if cfg.TopK > 0 {
		topK = cfg.TopK
	}

	if r.hooks.BeforeRetrieve != nil {
		if err := r.hooks.BeforeRetrieve(ctx, query); err != nil {
			if r.hooks.AfterRetrieve != nil {
				r.hooks.AfterRetrieve(ctx, nil, err)
			}
			return nil, err
		}
	}

	// Encode query into per-token embeddings.
	queryEmbeddings, err := r.embedder.EmbedMulti(ctx, []string{query})
	if err != nil {
		err = core.Errorf(core.ErrProviderDown, "colbert: embed query: %w", err)
		if r.hooks.AfterRetrieve != nil {
			r.hooks.AfterRetrieve(ctx, nil, err)
		}
		return nil, err
	}
	if len(queryEmbeddings) == 0 {
		err := core.Errorf(core.ErrProviderDown, "colbert: embedder returned no embeddings for query")
		if r.hooks.AfterRetrieve != nil {
			r.hooks.AfterRetrieve(ctx, nil, err)
		}
		return nil, err
	}
	queryVecs := queryEmbeddings[0]

	// Search the index.
	results, err := r.index.Search(ctx, queryVecs, topK)
	if err != nil {
		err = core.Errorf(core.ErrProviderDown, "colbert: index search: %w", err)
		if r.hooks.AfterRetrieve != nil {
			r.hooks.AfterRetrieve(ctx, nil, err)
		}
		return nil, err
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
