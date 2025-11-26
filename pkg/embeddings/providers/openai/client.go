package openai

import (
	"context"

	openaiClient "github.com/sashabaranov/go-openai"
)

// Client defines the interface for OpenAI API client operations.
type Client interface {
	CreateEmbeddings(ctx context.Context, req openaiClient.EmbeddingRequestConverter) (openaiClient.EmbeddingResponse, error)
}
