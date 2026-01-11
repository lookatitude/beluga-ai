package weaviate

import "github.com/lookatitude/beluga-ai/pkg/vectorstores"

func init() {
	// Register Weaviate provider with the global registry
	vectorstores.GetRegistry().Register("weaviate", NewWeaviateStoreFromConfig)
}
