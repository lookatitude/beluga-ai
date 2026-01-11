package pinecone

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

func init() {
	// Register Pinecone provider with the global registry
	vectorstores.GetRegistry().Register("pinecone", func(ctx context.Context, config vectorstoresiface.Config) (vectorstores.VectorStore, error) {
		return NewPineconeStoreFromConfig(ctx, config)
	})
}
