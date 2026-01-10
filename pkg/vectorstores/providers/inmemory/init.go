package inmemory

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

func init() {
	// Register inmemory provider with the global registry
	vectorstores.GetRegistry().Register("inmemory", func(ctx context.Context, config vectorstoresiface.Config) (vectorstores.VectorStore, error) {
		// Convert vectorstoresiface.Config to local Config
		localConfig := Config{
			Embedder:       config.Embedder,
			SearchK:        config.SearchK,
			ScoreThreshold: config.ScoreThreshold,
		}
		return NewInMemoryVectorStoreFromConfig(ctx, localConfig)
	})
}
