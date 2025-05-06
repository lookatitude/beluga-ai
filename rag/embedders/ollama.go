// Package embedders provides implementations of the rag.Embedder interface.
package embedders

import (
	"context"
	"errors"
	"fmt"

	"github.com/lookatitude/beluga-ai/rag"
	"github.com/ollama/ollama/api"
)

// OllamaEmbedder implements the rag.Embedder interface using a local Ollama instance.
type OllamaEmbedder struct {
	client *api.Client
	Model  string // Name of the embedding model served by Ollama
	// TODO: Add options for keep_alive, etc.
}

// NewOllamaEmbedder creates a new OllamaEmbedder.
// It attempts to connect to a local Ollama instance at the default address.
// If model is empty, the user must ensure Ollama serves a default embedding model or specify one later.
func NewOllamaEmbedder(model string, ollamaHost ...string) (*OllamaEmbedder, error) {
	var client *api.Client
	var err error

	if len(ollamaHost) > 0 && ollamaHost[0] != "" {
		// TODO: Need a way to create client with custom host, the library doesn't expose this easily.
		// For now, we rely on the default client creation which uses env vars or defaults.
		// client, err = api.ClientFromEnvironment() // Or similar if available
		fmt.Printf("Warning: Custom Ollama host (	%s	) requested but not yet supported by NewOllamaEmbedder. Using default.\n", ollamaHost[0])
		client, err = api.ClientFromEnvironment()
	} else {
		client, err = api.ClientFromEnvironment()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}
	if client == nil {
		// Double check as ClientFromEnvironment might return nil error but nil client?
		return nil, errors.New("failed to create Ollama client (nil client returned)")
	}

	return &OllamaEmbedder{
		client: client,
		Model:  model,
	}, nil
}

// EmbedDocuments creates embeddings for a batch of document texts.
// Ollama API currently processes one document at a time for embeddings.
func (e *OllamaEmbedder) EmbedDocuments(ctx context.Context, documents []string) ([][]float32, error) {
	if e.Model == "" {
		return nil, errors.New("Ollama model name must be set for embedding")
	}
	if len(documents) == 0 {
		return [][]float32{}, nil
	}

	embeddings := make([][]float32, len(documents))
	var firstErr error

	for i, doc := range documents {
		req := &api.EmbeddingRequest{
			Model:  e.Model,
			Prompt: doc,
			// TODO: Add Options map if needed
		}
		resp, err := e.client.Embeddings(ctx, req)
		if err != nil {
			// Capture the first error and continue trying others?
			// Or fail fast?
			if firstErr == nil {
				firstErr = fmt.Errorf("Ollama Embeddings failed for document %d: %w", i, err)
			}
			// Add a placeholder or skip? Let's add nil for now.
			embeddings[i] = nil
			continue // Try next document
		}
		// Convert []float64 to []float32
		embeddingF32 := make([]float32, len(resp.Embedding))
		for j, val := range resp.Embedding {
			embeddingF32[j] = float32(val)
		}
		embeddings[i] = embeddingF32
	}

	// If any errors occurred, return the first one encountered.
	if firstErr != nil {
		return embeddings, firstErr // Return partial results along with the error
	}

	return embeddings, nil
}

// EmbedQuery creates an embedding for a single query string.
func (e *OllamaEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	if e.Model == "" {
		return nil, errors.New("Ollama model name must be set for embedding")
	}
	if query == "" {
		return nil, errors.New("query cannot be empty")
	}

	req := &api.EmbeddingRequest{
		Model:  e.Model,
		Prompt: query,
	}

	resp, err := e.client.Embeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("Ollama Embeddings for query failed: %w", err)
	}

	// Convert []float64 to []float32
	embeddingF32 := make([]float32, len(resp.Embedding))
	for j, val := range resp.Embedding {
		embeddingF32[j] = float32(val)
	}

	return embeddingF32, nil
}

// Ensure OllamaEmbedder implements the interface.
var _ rag.Embedder = (*OllamaEmbedder)(nil)
