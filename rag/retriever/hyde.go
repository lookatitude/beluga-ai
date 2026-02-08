package retriever

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
)

// HyDERetriever implements Hypothetical Document Embeddings. It uses an LLM
// to generate a hypothetical answer to the query, embeds that answer, and
// searches for real documents similar to the hypothetical embedding. This
// bridges the semantic gap between short queries and long documents.
type HyDERetriever struct {
	llm      llm.ChatModel
	embedder embedding.Embedder
	store    vectorstore.VectorStore
	prompt   string
	hooks    Hooks
}

// HyDEOption configures a HyDERetriever.
type HyDEOption func(*HyDERetriever)

// WithHyDEPrompt sets a custom prompt template for generating hypothetical
// documents. The template should include %s where the query will be inserted.
func WithHyDEPrompt(prompt string) HyDEOption {
	return func(r *HyDERetriever) {
		r.prompt = prompt
	}
}

// WithHyDEHooks sets hooks on the HyDERetriever.
func WithHyDEHooks(h Hooks) HyDEOption {
	return func(r *HyDERetriever) {
		r.hooks = h
	}
}

const defaultHyDEPrompt = "Write a detailed passage that would answer the following question. " +
	"Do not include any preamble, just write the passage.\n\nQuestion: %s"

// NewHyDERetriever creates a HyDE retriever that generates hypothetical
// documents using the LLM, embeds them, and searches the vector store.
func NewHyDERetriever(model llm.ChatModel, embedder embedding.Embedder, store vectorstore.VectorStore, opts ...HyDEOption) *HyDERetriever {
	r := &HyDERetriever{
		llm:      model,
		embedder: embedder,
		store:    store,
		prompt:   defaultHyDEPrompt,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Retrieve generates a hypothetical document, embeds it, and searches for
// similar real documents.
func (r *HyDERetriever) Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
	cfg := ApplyOptions(opts...)

	if r.hooks.BeforeRetrieve != nil {
		if err := r.hooks.BeforeRetrieve(ctx, query); err != nil {
			return nil, err
		}
	}

	// Generate hypothetical document.
	prompt := fmt.Sprintf(r.prompt, query)
	msgs := []schema.Message{
		schema.NewHumanMessage(prompt),
	}

	resp, err := r.llm.Generate(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("retriever: hyde generate: %w", err)
	}

	hypoDoc := resp.Text()

	// Embed the hypothetical document.
	vec, err := r.embedder.EmbedSingle(ctx, hypoDoc)
	if err != nil {
		return nil, fmt.Errorf("retriever: hyde embed: %w", err)
	}

	// Search for similar real documents.
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
