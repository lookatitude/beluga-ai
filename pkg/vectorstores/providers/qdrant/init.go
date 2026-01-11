package qdrant

import "github.com/lookatitude/beluga-ai/pkg/vectorstores"

func init() {
	// Register Qdrant provider with the global registry
	vectorstores.GetRegistry().Register("qdrant", NewQdrantStoreFromConfig)
}
