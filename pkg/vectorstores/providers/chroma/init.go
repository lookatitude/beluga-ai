package chroma

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

func init() {
	// Register Chroma provider with the global registry
	vectorstores.GetRegistry().Register("chroma", func(ctx context.Context, config vectorstoresiface.Config) (vectorstores.VectorStore, error) {
		return NewChromaStoreFromConfig(ctx, config)
	})
}
