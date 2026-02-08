package retriever

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// QueryComplexity classifies how complex a query is, determining which
// retrieval strategy to use.
type QueryComplexity string

const (
	// NoRetrieval indicates the query can be answered from the LLM's
	// parametric knowledge without retrieval.
	NoRetrieval QueryComplexity = "no_retrieval"
	// SimpleRetrieval indicates a single-step retrieval is sufficient.
	SimpleRetrieval QueryComplexity = "simple"
	// ComplexRetrieval indicates multi-step or advanced retrieval is needed.
	ComplexRetrieval QueryComplexity = "complex"
)

// AdaptiveRetriever classifies query complexity using an LLM and routes to
// the appropriate retriever. Simple queries use a basic retriever, complex
// queries use a more sophisticated one, and trivial queries skip retrieval.
type AdaptiveRetriever struct {
	llm              llm.ChatModel
	simpleRetriever  Retriever
	complexRetriever Retriever
	hooks            Hooks
}

// AdaptiveOption configures an AdaptiveRetriever.
type AdaptiveOption func(*AdaptiveRetriever)

// WithAdaptiveHooks sets hooks on the AdaptiveRetriever.
func WithAdaptiveHooks(h Hooks) AdaptiveOption {
	return func(r *AdaptiveRetriever) {
		r.hooks = h
	}
}

// NewAdaptiveRetriever creates a retriever that classifies queries by
// complexity and routes to the appropriate retriever. If complexRetriever is
// nil, simpleRetriever is used for all retrievals.
func NewAdaptiveRetriever(model llm.ChatModel, simpleRetriever, complexRetriever Retriever, opts ...AdaptiveOption) *AdaptiveRetriever {
	if complexRetriever == nil {
		complexRetriever = simpleRetriever
	}
	r := &AdaptiveRetriever{
		llm:              model,
		simpleRetriever:  simpleRetriever,
		complexRetriever: complexRetriever,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Retrieve classifies the query complexity, then either returns no documents
// or delegates to the appropriate retriever.
func (r *AdaptiveRetriever) Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
	if r.hooks.BeforeRetrieve != nil {
		if err := r.hooks.BeforeRetrieve(ctx, query); err != nil {
			return nil, err
		}
	}

	complexity, err := r.classifyQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("retriever: adaptive classify: %w", err)
	}

	var docs []schema.Document
	switch complexity {
	case NoRetrieval:
		docs = nil
	case SimpleRetrieval:
		docs, err = r.simpleRetriever.Retrieve(ctx, query, opts...)
		if err != nil {
			return nil, fmt.Errorf("retriever: adaptive simple: %w", err)
		}
	case ComplexRetrieval:
		docs, err = r.complexRetriever.Retrieve(ctx, query, opts...)
		if err != nil {
			return nil, fmt.Errorf("retriever: adaptive complex: %w", err)
		}
	default:
		// Default to simple retrieval on unrecognised classification.
		docs, err = r.simpleRetriever.Retrieve(ctx, query, opts...)
		if err != nil {
			return nil, fmt.Errorf("retriever: adaptive default: %w", err)
		}
	}

	if r.hooks.AfterRetrieve != nil {
		r.hooks.AfterRetrieve(ctx, docs, nil)
	}

	return docs, nil
}

// classifyQuery uses the LLM to determine the query complexity.
func (r *AdaptiveRetriever) classifyQuery(ctx context.Context, query string) (QueryComplexity, error) {
	prompt := fmt.Sprintf(
		"Classify the following query into one of three categories:\n"+
			"- no_retrieval: The query can be answered from general knowledge without looking up documents.\n"+
			"- simple: The query needs a single straightforward document lookup.\n"+
			"- complex: The query requires multi-step reasoning or multiple document lookups.\n\n"+
			"Respond with ONLY one of: no_retrieval, simple, complex\n\n"+
			"Query: %s",
		query,
	)

	msgs := []schema.Message{
		schema.NewHumanMessage(prompt),
	}

	resp, err := r.llm.Generate(ctx, msgs)
	if err != nil {
		return SimpleRetrieval, fmt.Errorf("classify query: %w", err)
	}

	text := strings.TrimSpace(strings.ToLower(resp.Text()))

	switch {
	case strings.Contains(text, "no_retrieval"):
		return NoRetrieval, nil
	case strings.Contains(text, "complex"):
		return ComplexRetrieval, nil
	default:
		return SimpleRetrieval, nil
	}
}
