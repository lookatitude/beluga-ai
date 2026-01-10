package openai

import (
	"context"
	"errors"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"go.opentelemetry.io/otel"
)

func init() {
	// Register OpenAI provider with the global registry
	embeddings.GetRegistry().Register("openai", func(ctx context.Context, config embeddings.Config) (embeddingsiface.Embedder, error) {
		if config.OpenAI == nil || !config.OpenAI.Enabled {
			return nil, errors.New("OpenAI provider is not configured or disabled")
		}

		if err := config.OpenAI.Validate(); err != nil {
			return nil, fmt.Errorf("invalid OpenAI configuration: %w", err)
		}

		openaiConfig := &Config{
			APIKey:     config.OpenAI.APIKey,
			Model:      config.OpenAI.Model,
			BaseURL:    config.OpenAI.BaseURL,
			APIVersion: config.OpenAI.APIVersion,
			Timeout:    config.OpenAI.Timeout,
			MaxRetries: config.OpenAI.MaxRetries,
			Enabled:    config.OpenAI.Enabled,
		}

		tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/embeddings")
		return NewOpenAIEmbedder(openaiConfig, tracer)
	})
}
