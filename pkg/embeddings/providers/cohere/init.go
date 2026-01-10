package cohere

import (
	"context"
	"errors"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"go.opentelemetry.io/otel"
)

func init() {
	// Register Cohere provider with the global registry
	embeddings.GetRegistry().Register("cohere", func(ctx context.Context, config embeddings.Config) (embeddingsiface.Embedder, error) {
		if config.Cohere == nil || !config.Cohere.Enabled {
			return nil, errors.New("cohere provider is not configured or disabled")
		}

		if err := config.Cohere.Validate(); err != nil {
			return nil, fmt.Errorf("invalid Cohere configuration: %w", err)
		}

		cohereConfig := &Config{
			APIKey:     config.Cohere.APIKey,
			Model:      config.Cohere.Model,
			BaseURL:    config.Cohere.BaseURL,
			Timeout:    config.Cohere.Timeout,
			MaxRetries: config.Cohere.MaxRetries,
			Enabled:    config.Cohere.Enabled,
		}

		tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/embeddings")
		return NewCohereEmbedder(cohereConfig, tracer)
	})
}
