package pgvector

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

func init() {
	// Register pgvector provider with the global registry
	vectorstores.GetRegistry().Register("pgvector", func(ctx context.Context, config vectorstoresiface.Config) (vectorstores.VectorStore, error) {
		return NewPgVectorStoreFromConfig(ctx, config)
	})
}
