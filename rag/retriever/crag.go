package retriever

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// WebSearcher performs web search queries. Implementations may wrap external
// search APIs (Google, Bing, Tavily, etc.).
type WebSearcher interface {
	// Search returns documents from the web matching the query.
	Search(ctx context.Context, query string, k int) ([]schema.Document, error)
}

// CRAGRetriever implements Corrective RAG. After retrieving documents, it
// evaluates their relevance to the query using an LLM. If documents score
// below the threshold, it optionally rewrites the query and retries before
// falling back to a web search for more relevant content.
type CRAGRetriever struct {
	inner       Retriever
	llm         llm.ChatModel
	web         WebSearcher
	threshold   float64
	maxAttempts int
	hooks       Hooks
}

// CRAGOption configures a CRAGRetriever.
type CRAGOption func(*CRAGRetriever)

// WithCRAGThreshold sets the minimum relevance score (-1 to 1) for documents.
// Documents scoring below this are considered irrelevant. Defaults to 0.
func WithCRAGThreshold(t float64) CRAGOption {
	return func(r *CRAGRetriever) {
		r.threshold = t
	}
}

// WithCRAGMaxAttempts sets the maximum number of retrieval attempts with
// query rewriting before falling back to web search. When set to a value
// greater than 1, the retriever rewrites the query using the LLM on
// relevance failure and retries with the inner retriever. Defaults to 1
// (no retries).
func WithCRAGMaxAttempts(n int) CRAGOption {
	return func(r *CRAGRetriever) {
		if n > 0 {
			r.maxAttempts = n
		}
	}
}

// WithCRAGHooks sets hooks on the CRAGRetriever.
func WithCRAGHooks(h Hooks) CRAGOption {
	return func(r *CRAGRetriever) {
		r.hooks = h
	}
}

// NewCRAGRetriever creates a Corrective RAG retriever. It retrieves documents
// from the inner retriever, evaluates them with the LLM, and falls back to
// web search if relevance is low. If web is nil, no fallback is performed.
func NewCRAGRetriever(inner Retriever, model llm.ChatModel, web WebSearcher, opts ...CRAGOption) *CRAGRetriever {
	r := &CRAGRetriever{
		inner:       inner,
		llm:         model,
		web:         web,
		threshold:   0,
		maxAttempts: 1,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Retrieve fetches documents, evaluates relevance, optionally rewrites the
// query and retries, and falls back to web search if relevance remains low.
func (r *CRAGRetriever) Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
	cfg := ApplyOptions(opts...)

	if r.hooks.BeforeRetrieve != nil {
		if err := r.hooks.BeforeRetrieve(ctx, query); err != nil {
			return nil, err
		}
	}

	currentQuery := query
	for attempt := 0; attempt < r.maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		docs, err := r.inner.Retrieve(ctx, currentQuery, opts...)
		if err != nil {
			return nil, core.Errorf(core.ErrProviderDown, "retriever: crag inner retrieve: %w", err)
		}

		if len(docs) == 0 {
			// No docs — try rewriting before web fallback.
			if attempt < r.maxAttempts-1 {
				rewritten, err := r.rewriteQuery(ctx, query, currentQuery)
				if err != nil {
					return nil, core.Errorf(core.ErrProviderDown, "retriever: crag rewrite: %w", err)
				}
				currentQuery = rewritten
				continue
			}
			return r.fallbackSearch(ctx, currentQuery, cfg)
		}

		// Evaluate relevance of retrieved documents.
		relevant, err := r.evaluateRelevance(ctx, currentQuery, docs)
		if err != nil {
			return nil, core.Errorf(core.ErrProviderDown, "retriever: crag evaluate: %w", err)
		}

		// If enough relevant documents, return them.
		if len(relevant) > 0 {
			if cfg.TopK > 0 && len(relevant) > cfg.TopK {
				relevant = relevant[:cfg.TopK]
			}
			if r.hooks.AfterRetrieve != nil {
				r.hooks.AfterRetrieve(ctx, relevant, nil)
			}
			return relevant, nil
		}

		// Relevance too low — rewrite and retry if attempts remain.
		if attempt < r.maxAttempts-1 {
			rewritten, err := r.rewriteQuery(ctx, query, currentQuery)
			if err != nil {
				return nil, core.Errorf(core.ErrProviderDown, "retriever: crag rewrite: %w", err)
			}
			currentQuery = rewritten
			continue
		}

		// Final attempt — fall back to web search.
		return r.fallbackSearch(ctx, currentQuery, cfg)
	}

	// Unreachable under normal operation: every iteration of the loop
	// either returns or continues. Left as a defensive safety net.
	return r.fallbackSearch(ctx, currentQuery, cfg)
}

// evaluateRelevance uses the LLM to score each document's relevance to the
// query on a scale from -1 to 1.
func (r *CRAGRetriever) evaluateRelevance(ctx context.Context, query string, docs []schema.Document) ([]schema.Document, error) {
	var relevant []schema.Document

	for _, doc := range docs {
		score, err := r.scoreDocument(ctx, query, doc)
		if err != nil {
			return nil, err
		}

		if score >= r.threshold {
			doc.Score = score
			relevant = append(relevant, doc)
		}
	}

	return relevant, nil
}

// scoreDocument asks the LLM to rate relevance of a single document.
func (r *CRAGRetriever) scoreDocument(ctx context.Context, query string, doc schema.Document) (float64, error) {
	prompt := fmt.Sprintf(
		"Rate the relevance of the following document to the query on a scale from -1 to 1, "+
			"where -1 is completely irrelevant and 1 is highly relevant. "+
			"Respond with ONLY a single number.\n\n"+
			"Query: %s\n\nDocument: %s",
		query, doc.Content,
	)

	msgs := []schema.Message{
		schema.NewHumanMessage(prompt),
	}

	resp, err := r.llm.Generate(ctx, msgs)
	if err != nil {
		return 0, core.Errorf(core.ErrProviderDown, "crag score: %w", err)
	}

	text := strings.TrimSpace(resp.Text())
	score, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, core.Errorf(core.ErrInvalidInput, "crag score parse %q: %w", text, err)
	}

	// Clamp to [-1, 1].
	if score < -1 {
		score = -1
	}
	if score > 1 {
		score = 1
	}

	return score, nil
}

// rewriteQuery asks the LLM to reformulate the query for better retrieval.
func (r *CRAGRetriever) rewriteQuery(ctx context.Context, originalQuery, currentQuery string) (string, error) {
	prompt := fmt.Sprintf(
		"The following search query did not return relevant documents.\n"+
			"Original query: %s\n"+
			"Current query: %s\n\n"+
			"Reformulate this query to improve search results. "+
			"Return ONLY the rewritten query, nothing else.",
		originalQuery, currentQuery,
	)

	msgs := []schema.Message{
		schema.NewHumanMessage(prompt),
	}

	resp, err := r.llm.Generate(ctx, msgs)
	if err != nil {
		return "", core.Errorf(core.ErrProviderDown, "crag rewrite generate: %w", err)
	}

	rewritten := strings.TrimSpace(resp.Text())
	if rewritten == "" {
		return currentQuery, nil
	}
	return rewritten, nil
}

// fallbackSearch performs web search when document relevance is low.
func (r *CRAGRetriever) fallbackSearch(ctx context.Context, query string, cfg Config) ([]schema.Document, error) {
	if r.web == nil {
		if r.hooks.AfterRetrieve != nil {
			r.hooks.AfterRetrieve(ctx, nil, nil)
		}
		return nil, nil
	}

	k := cfg.TopK
	if k <= 0 {
		k = 10
	}

	docs, err := r.web.Search(ctx, query, k)
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "retriever: crag web search: %w", err)
	}

	if r.hooks.AfterRetrieve != nil {
		r.hooks.AfterRetrieve(ctx, docs, nil)
	}

	return docs, nil
}
