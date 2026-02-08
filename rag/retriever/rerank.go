package retriever

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/schema"
)

// Reranker re-scores a set of documents for relevance to a query. Cross-encoder
// models and LLM-based scorers implement this interface.
type Reranker interface {
	// Rerank re-scores the given documents for the query and returns them
	// in descending order of relevance with updated Score fields.
	Rerank(ctx context.Context, query string, docs []schema.Document) ([]schema.Document, error)
}

// RerankRetriever wraps an inner Retriever and applies a Reranker to the
// results before returning them. This is the standard two-stage
// retrieve-then-rerank pattern.
type RerankRetriever struct {
	inner    Retriever
	reranker Reranker
	topN     int
	hooks    Hooks
}

// RerankOption configures a RerankRetriever.
type RerankOption func(*RerankRetriever)

// WithRerankTopN sets the number of documents to return after re-ranking.
// Defaults to 0 (return all re-ranked documents).
func WithRerankTopN(n int) RerankOption {
	return func(r *RerankRetriever) {
		r.topN = n
	}
}

// WithRerankHooks sets hooks on the RerankRetriever.
func WithRerankHooks(h Hooks) RerankOption {
	return func(r *RerankRetriever) {
		r.hooks = h
	}
}

// NewRerankRetriever creates a retriever that retrieves from inner and then
// re-ranks the results using the given reranker.
func NewRerankRetriever(inner Retriever, reranker Reranker, opts ...RerankOption) *RerankRetriever {
	r := &RerankRetriever{
		inner:    inner,
		reranker: reranker,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Retrieve fetches documents from the inner retriever, applies the reranker,
// and returns the top results.
func (r *RerankRetriever) Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
	if r.hooks.BeforeRetrieve != nil {
		if err := r.hooks.BeforeRetrieve(ctx, query); err != nil {
			return nil, err
		}
	}

	docs, err := r.inner.Retrieve(ctx, query, opts...)
	if err != nil {
		return nil, fmt.Errorf("retriever: rerank inner retrieve: %w", err)
	}

	if len(docs) == 0 {
		if r.hooks.AfterRetrieve != nil {
			r.hooks.AfterRetrieve(ctx, docs, nil)
		}
		return docs, nil
	}

	reranked, err := r.reranker.Rerank(ctx, query, docs)
	if err != nil {
		return nil, fmt.Errorf("retriever: rerank: %w", err)
	}

	if r.hooks.OnRerank != nil {
		r.hooks.OnRerank(ctx, query, docs, reranked)
	}

	if r.topN > 0 && len(reranked) > r.topN {
		reranked = reranked[:r.topN]
	}

	if r.hooks.AfterRetrieve != nil {
		r.hooks.AfterRetrieve(ctx, reranked, nil)
	}

	return reranked, nil
}
