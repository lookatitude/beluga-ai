// Package embedders provides implementations of the rag.Embedder interface.
package embedders

import (
	"context"
	"errors" // Added missing import
	"fmt"

	"github.com/lookatitude/beluga-ai/rag"
	"github.com/sashabaranov/go-openai"
)

// OpenAIEmbedder implements the rag.Embedder interface using the OpenAI API.
type OpenAIEmbedder struct {
	client *openai.Client
	Model  openai.EmbeddingModel // Model to use, e.g., text-embedding-ada-002
	// TODO: Add options for batch size, retries, etc.
}

// NewOpenAIEmbedder creates a new OpenAIEmbedder.
// Requires an OpenAI API key (usually set via environment variable OPENAI_API_KEY).
// If model is empty, it defaults to text-embedding-ada-002.
func NewOpenAIEmbedder(apiKey string, model openai.EmbeddingModel) (*OpenAIEmbedder, error) {
	client := openai.NewClient(apiKey) // Assumes API key is passed or set in env
	if client == nil {
		// The go-openai client constructor doesn't return an error, but let's keep a check
		// It might panic or misbehave later if the key is truly invalid/missing.
		// A better approach might be a health check call.
		fmt.Println("Warning: OpenAI client created, but API key validity not checked.")
	}

	modelName := model
	if modelName == "" {
		modelName = openai.AdaEmbeddingV2 // Default model
	}

	return &OpenAIEmbedder{
		client: client,
		Model:  modelName,
	}, nil
}

// EmbedDocuments creates embeddings for a batch of document texts.
func (e *OpenAIEmbedder) EmbedDocuments(ctx context.Context, documents []string) ([][]float32, error) {
	if len(documents) == 0 {
		return [][]float32{}, nil
	}

	req := openai.EmbeddingRequest{
		Input: documents,
		Model: e.Model,
		// TODO: Add User field if needed
	}

	resp, err := e.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI CreateEmbeddings failed: %w", err)
	}

	if len(resp.Data) != len(documents) {
		return nil, fmt.Errorf("OpenAI returned %d embeddings for %d documents", len(resp.Data), len(documents))
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		// Ensure correct index mapping, although OpenAI usually preserves order
		if data.Index != i {
			// This indicates a potential issue with OpenAI's response or our assumption
			return nil, fmt.Errorf("OpenAI embedding index mismatch: expected %d, got %d", i, data.Index)
		}
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

// EmbedQuery creates an embedding for a single query string.
func (e *OpenAIEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	if query == "" {
		return nil, errors.New("query cannot be empty")
	}

	req := openai.EmbeddingRequest{
		Input: []string{query},
		Model: e.Model,
	}

	resp, err := e.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI CreateEmbeddings for query failed: %w", err)
	}

	if len(resp.Data) != 1 {
		return nil, fmt.Errorf("OpenAI returned %d embeddings for 1 query", len(resp.Data))
	}

	return resp.Data[0].Embedding, nil
}

// Ensure OpenAIEmbedder implements the interface.
var _ rag.Embedder = (*OpenAIEmbedder)(nil)
