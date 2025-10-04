package openai

import (
	"context"

	"github.com/sashabaranov/go-openai"
)

// Client defines the interface for OpenAI API client operations
type Client interface {
	CreateEmbeddings(ctx context.Context, req openai.EmbeddingRequestConverter) (openai.EmbeddingResponse, error)
}
