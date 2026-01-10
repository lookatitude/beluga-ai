package ollama

import (
	"context"
	"errors"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"go.opentelemetry.io/otel"
)

func init() {
	// Register Ollama provider with the global registry
	embeddings.GetRegistry().Register("ollama", func(ctx context.Context, config embeddings.Config) (embeddingsiface.Embedder, error) {
		if config.Ollama == nil || !config.Ollama.Enabled {
			return nil, errors.New("ollama provider is not configured or disabled")
		}

		if err := config.Ollama.Validate(); err != nil {
			return nil, fmt.Errorf("invalid Ollama configuration: %w", err)
		}

		ollamaConfig := &Config{
			ServerURL:  config.Ollama.ServerURL,
			Model:      config.Ollama.Model,
			Timeout:    config.Ollama.Timeout,
			MaxRetries: config.Ollama.MaxRetries,
			KeepAlive:  config.Ollama.KeepAlive,
			Enabled:    config.Ollama.Enabled,
		}

		tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/embeddings")
		return NewOllamaEmbedder(ollamaConfig, tracer)
	})
}
