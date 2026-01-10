package qdrant

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

func init() {
	// Register Qdrant provider with the global registry
	// TODO: Implement NewQdrantStoreFromConfig
	vectorstores.GetRegistry().Register("qdrant", func(ctx context.Context, config vectorstoresiface.Config) (vectorstores.VectorStore, error) {
		// Qdrant provider is not yet implemented
		return nil, fmt.Errorf("qdrant provider is not yet implemented")
	})
}
