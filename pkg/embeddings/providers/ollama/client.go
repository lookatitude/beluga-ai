//go:build experimental

package ollama

import (
	"context"

	"github.com/ollama/ollama/api"
)

// Client defines the interface for Ollama API client operations.
type Client interface {
	Embeddings(ctx context.Context, req *api.EmbeddingRequest) (*api.EmbeddingResponse, error)
}
