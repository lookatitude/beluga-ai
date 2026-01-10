package mock

import (
	"context"
	"errors"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"go.opentelemetry.io/otel"
)

func init() {
	// Register Mock provider with the global registry
	embeddings.GetRegistry().Register("mock", func(ctx context.Context, config embeddings.Config) (embeddingsiface.Embedder, error) {
		if config.Mock == nil || !config.Mock.Enabled {
			return nil, errors.New("mock provider is not configured or disabled")
		}

		if err := config.Mock.Validate(); err != nil {
			return nil, fmt.Errorf("invalid Mock configuration: %w", err)
		}

		mockConfig := &Config{
			Dimension:    config.Mock.Dimension,
			Seed:         config.Mock.Seed,
			RandomizeNil: config.Mock.RandomizeNil,
			Enabled:      config.Mock.Enabled,
		}

		tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/embeddings")
		return NewMockEmbedder(mockConfig, tracer)
	})
}
