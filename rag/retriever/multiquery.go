package retriever

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// MultiQueryRetriever generates multiple query variations using an LLM and
// retrieves documents for each, deduplicating and merging results. This
// improves recall by capturing different phrasings of the same intent.
type MultiQueryRetriever struct {
	inner      Retriever
	llm        llm.ChatModel
	numQueries int
	hooks      Hooks
}

// MultiQueryOption configures a MultiQueryRetriever.
type MultiQueryOption func(*MultiQueryRetriever)

// WithMultiQueryCount sets the number of query variations to generate.
// Defaults to 3.
func WithMultiQueryCount(n int) MultiQueryOption {
	return func(r *MultiQueryRetriever) {
		if n > 0 {
			r.numQueries = n
		}
	}
}

// WithMultiQueryHooks sets hooks on the MultiQueryRetriever.
func WithMultiQueryHooks(h Hooks) MultiQueryOption {
	return func(r *MultiQueryRetriever) {
		r.hooks = h
	}
}

// NewMultiQueryRetriever creates a retriever that generates query variations
// using the provided LLM and retrieves from the inner retriever for each.
func NewMultiQueryRetriever(inner Retriever, model llm.ChatModel, opts ...MultiQueryOption) *MultiQueryRetriever {
	r := &MultiQueryRetriever{
		inner:      inner,
		llm:        model,
		numQueries: 3,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Retrieve generates query variations, retrieves for each, and returns
// deduplicated results.
func (r *MultiQueryRetriever) Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
	if r.hooks.BeforeRetrieve != nil {
		if err := r.hooks.BeforeRetrieve(ctx, query); err != nil {
			return nil, err
		}
	}

	queries, err := r.generateQueries(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("retriever: generate queries: %w", err)
	}

	// Always include the original query.
	allQueries := append([]string{query}, queries...)

	seen := make(map[string]struct{})
	var results []schema.Document

	for _, q := range allQueries {
		docs, err := r.inner.Retrieve(ctx, q, opts...)
		if err != nil {
			return nil, fmt.Errorf("retriever: multiquery retrieve %q: %w", q, err)
		}
		for _, doc := range docs {
			if _, ok := seen[doc.ID]; !ok {
				seen[doc.ID] = struct{}{}
				results = append(results, doc)
			}
		}
	}

	if r.hooks.AfterRetrieve != nil {
		r.hooks.AfterRetrieve(ctx, results, nil)
	}

	return results, nil
}

// generateQueries uses the LLM to produce query variations.
func (r *MultiQueryRetriever) generateQueries(ctx context.Context, query string) ([]string, error) {
	prompt := fmt.Sprintf(
		"Generate %d alternative search queries for the following query. "+
			"Return each query on a separate line, without numbering or bullet points.\n\nQuery: %s",
		r.numQueries, query,
	)

	msgs := []schema.Message{
		schema.NewHumanMessage(prompt),
	}

	resp, err := r.llm.Generate(ctx, msgs)
	if err != nil {
		return nil, err
	}

	text := resp.Text()
	lines := strings.Split(strings.TrimSpace(text), "\n")

	var queries []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			queries = append(queries, line)
		}
	}

	return queries, nil
}
