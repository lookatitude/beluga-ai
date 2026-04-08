package retriever

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// QueryRewriter wraps an inner Retriever with LLM-driven query rewriting.
// After the initial retrieval, it evaluates the relevance of returned documents.
// If the average relevance score falls below a configurable threshold, the LLM
// reformulates the query and retries retrieval, up to a maximum number of
// rewrites.
type QueryRewriter struct {
	inner              Retriever
	llm                llm.ChatModel
	rewriteModel       llm.ChatModel
	maxRewrites        int
	relevanceThreshold float64
	hooks              Hooks
}

// Compile-time interface check.
var _ Retriever = (*QueryRewriter)(nil)

// RewriteOption configures a QueryRewriter.
type RewriteOption func(*QueryRewriter)

// WithRewriteModel sets a separate LLM for query rewriting. If not set, the
// main LLM is used for both relevance evaluation and rewriting.
func WithRewriteModel(m llm.ChatModel) RewriteOption {
	return func(r *QueryRewriter) {
		r.rewriteModel = m
	}
}

// WithMaxRewrites sets the maximum number of query rewrite attempts.
// Defaults to 3.
func WithMaxRewrites(n int) RewriteOption {
	return func(r *QueryRewriter) {
		if n > 0 {
			r.maxRewrites = n
		}
	}
}

// WithRelevanceThreshold sets the minimum average relevance score (0 to 1)
// for the retrieval to be considered successful. If the average score of
// returned documents falls below this threshold, the query is rewritten.
// Defaults to 0.7.
func WithRelevanceThreshold(t float64) RewriteOption {
	return func(r *QueryRewriter) {
		r.relevanceThreshold = t
	}
}

// WithRewriteHooks sets hooks on the QueryRewriter.
func WithRewriteHooks(h Hooks) RewriteOption {
	return func(r *QueryRewriter) {
		r.hooks = h
	}
}

// NewQueryRewriter creates a QueryRewriter that wraps the inner retriever.
// The model is used for both relevance evaluation and query rewriting unless
// a separate rewrite model is specified via WithRewriteModel.
func NewQueryRewriter(inner Retriever, model llm.ChatModel, opts ...RewriteOption) *QueryRewriter {
	r := &QueryRewriter{
		inner:              inner,
		llm:                model,
		maxRewrites:        3,
		relevanceThreshold: 0.7,
	}
	for _, o := range opts {
		o(r)
	}
	if r.rewriteModel == nil {
		r.rewriteModel = model
	}
	return r
}

// Retrieve fetches documents using the inner retriever. If the average
// relevance score is below the threshold, the query is rewritten using the
// LLM and retrieval is retried, up to maxRewrites times.
func (r *QueryRewriter) Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
	if r.hooks.BeforeRetrieve != nil {
		if err := r.hooks.BeforeRetrieve(ctx, query); err != nil {
			return nil, err
		}
	}

	currentQuery := query
	for attempt := 0; attempt <= r.maxRewrites; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		docs, err := r.inner.Retrieve(ctx, currentQuery, opts...)
		if err != nil {
			return nil, fmt.Errorf("retriever: rewrite retrieve (attempt %d): %w", attempt, err)
		}

		if len(docs) == 0 && attempt < r.maxRewrites {
			rewritten, err := r.rewriteQuery(ctx, query, currentQuery, attempt)
			if err != nil {
				return nil, fmt.Errorf("retriever: rewrite query: %w", err)
			}
			currentQuery = rewritten
			continue
		}

		if len(docs) == 0 {
			if r.hooks.AfterRetrieve != nil {
				r.hooks.AfterRetrieve(ctx, nil, nil)
			}
			return nil, nil
		}

		avgScore := r.averageRelevance(docs)
		if avgScore >= r.relevanceThreshold || attempt >= r.maxRewrites {
			if r.hooks.AfterRetrieve != nil {
				r.hooks.AfterRetrieve(ctx, docs, nil)
			}
			return docs, nil
		}

		// Relevance too low — rewrite the query.
		rewritten, err := r.rewriteQuery(ctx, query, currentQuery, attempt)
		if err != nil {
			return nil, fmt.Errorf("retriever: rewrite query: %w", err)
		}
		currentQuery = rewritten
	}

	// Should not be reached, but return empty for safety.
	if r.hooks.AfterRetrieve != nil {
		r.hooks.AfterRetrieve(ctx, nil, nil)
	}
	return nil, nil
}

// averageRelevance computes the mean Score across docs.
func (r *QueryRewriter) averageRelevance(docs []schema.Document) float64 {
	if len(docs) == 0 {
		return 0
	}
	var sum float64
	for _, d := range docs {
		sum += d.Score
	}
	return sum / float64(len(docs))
}

// rewriteQuery asks the LLM to reformulate the query for better retrieval.
func (r *QueryRewriter) rewriteQuery(ctx context.Context, originalQuery, currentQuery string, attempt int) (string, error) {
	prompt := fmt.Sprintf(
		"The following search query did not return sufficiently relevant results.\n"+
			"Original query: %s\n"+
			"Current query (attempt %d): %s\n\n"+
			"Please reformulate this query to improve search results. "+
			"Return ONLY the rewritten query, nothing else.",
		originalQuery, attempt+1, currentQuery,
	)

	msgs := []schema.Message{
		schema.NewHumanMessage(prompt),
	}

	resp, err := r.rewriteModel.Generate(ctx, msgs)
	if err != nil {
		return "", fmt.Errorf("rewrite generate: %w", err)
	}

	rewritten := strings.TrimSpace(resp.Text())
	if rewritten == "" {
		return currentQuery, nil
	}
	return rewritten, nil
}
