package qdrant

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

func init() {
	// Register Qdrant provider with the global registry
	vectorstores.GetRegistry().Register("qdrant", func(ctx context.Context, config vectorstoresiface.Config) (vectorstores.VectorStore, error) {
		return NewQdrantStoreFromConfig(ctx, config)
	})
}
