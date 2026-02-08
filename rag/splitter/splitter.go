// Package splitter provides text splitting capabilities for the RAG pipeline.
// It defines the TextSplitter interface for dividing text content into smaller
// chunks suitable for embedding and retrieval.
//
// Built-in splitters:
//   - "recursive"  — Recursive character splitter with configurable separators
//   - "markdown"   — Markdown-aware splitter that respects heading hierarchy
//   - "token"      — Token-based splitter using an llm.Tokenizer
//
// Usage:
//
//	s, err := splitter.New("recursive", config.ProviderConfig{
//	    Options: map[string]any{"chunk_size": 1000, "chunk_overlap": 200},
//	})
//	if err != nil { ... }
//	chunks, err := s.Split(ctx, longText)
package splitter

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/schema"
)

// TextSplitter splits text content into smaller chunks. Implementations
// determine chunk boundaries based on different strategies (characters,
// tokens, document structure).
type TextSplitter interface {
	// Split divides text into a slice of chunks.
	Split(ctx context.Context, text string) ([]string, error)

	// SplitDocuments splits each document's content and returns new documents
	// for each chunk, preserving and augmenting the original metadata.
	SplitDocuments(ctx context.Context, docs []schema.Document) ([]schema.Document, error)
}

// Factory creates a TextSplitter from a ProviderConfig.
type Factory func(cfg config.ProviderConfig) (TextSplitter, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds a splitter factory to the global registry. It is intended to
// be called from provider init() functions. Duplicate registrations for the
// same name silently overwrite the previous factory.
func Register(name string, f Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = f
}

// New creates a TextSplitter by looking up the provider name in the registry
// and calling its factory with the given configuration.
func New(name string, cfg config.ProviderConfig) (TextSplitter, error) {
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("splitter: unknown provider %q (registered: %v)", name, List())
	}
	return f(cfg)
}

// List returns the names of all registered splitters, sorted alphabetically.
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// splitDocumentsHelper is a shared implementation for SplitDocuments used by
// concrete splitters.
func splitDocumentsHelper(ctx context.Context, s TextSplitter, docs []schema.Document) ([]schema.Document, error) {
	var result []schema.Document
	for _, doc := range docs {
		chunks, err := s.Split(ctx, doc.Content)
		if err != nil {
			return nil, fmt.Errorf("splitter: splitting document %q: %w", doc.ID, err)
		}
		for i, chunk := range chunks {
			meta := make(map[string]any, len(doc.Metadata)+2)
			for k, v := range doc.Metadata {
				meta[k] = v
			}
			meta["chunk_index"] = i
			meta["chunk_total"] = len(chunks)
			meta["parent_id"] = doc.ID

			result = append(result, schema.Document{
				ID:       fmt.Sprintf("%s#chunk%d", doc.ID, i),
				Content:  chunk,
				Metadata: meta,
			})
		}
	}
	return result, nil
}
