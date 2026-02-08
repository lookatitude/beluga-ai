package memory

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
)

// ArchivalConfig configures the archival memory tier.
type ArchivalConfig struct {
	// VectorStore is the backend for storing and searching document embeddings.
	VectorStore vectorstore.VectorStore

	// Embedder produces vector embeddings for text content.
	Embedder embedding.Embedder
}

// Archival implements the MemGPT archival memory tier. It provides long-term
// storage backed by vector embeddings, enabling semantic search over historical
// content. Messages are embedded and stored as documents for later retrieval.
type Archival struct {
	vs  vectorstore.VectorStore
	emb embedding.Embedder
	seq atomic.Int64 // monotonically increasing ID sequence
}

// NewArchival creates a new Archival memory with the given configuration.
// Both VectorStore and Embedder must be non-nil.
func NewArchival(cfg ArchivalConfig) (*Archival, error) {
	if cfg.VectorStore == nil {
		return nil, fmt.Errorf("memory/archival: VectorStore is required")
	}
	if cfg.Embedder == nil {
		return nil, fmt.Errorf("memory/archival: Embedder is required")
	}
	return &Archival{
		vs:  cfg.VectorStore,
		emb: cfg.Embedder,
	}, nil
}

// Save implements Memory. Embeds both the input and output messages and stores
// them as documents in the vector store.
func (a *Archival) Save(ctx context.Context, input, output schema.Message) error {
	msgs := []schema.Message{input, output}
	var texts []string
	var docs []schema.Document
	for _, msg := range msgs {
		text := extractMessageText(msg)
		if text == "" {
			continue
		}
		id := a.seq.Add(1)
		docs = append(docs, schema.Document{
			ID:      fmt.Sprintf("archival-%d", id),
			Content: text,
			Metadata: map[string]any{
				"role": string(msg.GetRole()),
			},
		})
		texts = append(texts, text)
	}
	if len(texts) == 0 {
		return nil
	}
	embeddings, err := a.emb.Embed(ctx, texts)
	if err != nil {
		return fmt.Errorf("memory/archival: embed: %w", err)
	}
	return a.vs.Add(ctx, docs, embeddings)
}

// Load implements Memory. Archival memory does not support message retrieval
// directly; use Search for document-based retrieval.
func (a *Archival) Load(ctx context.Context, query string) ([]schema.Message, error) {
	return nil, nil
}

// Search implements Memory. Embeds the query and performs similarity search
// over the vector store, returning at most k results.
func (a *Archival) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	if k <= 0 {
		k = 10
	}
	vec, err := a.emb.EmbedSingle(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("memory/archival: embed query: %w", err)
	}
	return a.vs.Search(ctx, vec, k)
}

// Clear implements Memory. Archival memory clear is a no-op because vector
// stores do not universally support bulk deletion. Use the vector store's
// Delete method directly if needed.
func (a *Archival) Clear(ctx context.Context) error {
	return nil
}

func init() {
	Register("archival", func(cfg config.ProviderConfig) (Memory, error) {
		// Archival requires VectorStore and Embedder to be provided via
		// composite memory options rather than the generic registry.
		return nil, fmt.Errorf("memory/archival: use NewArchival directly with ArchivalConfig; " +
			"the registry factory requires VectorStore and Embedder")
	})
}
