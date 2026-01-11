package mock

import "github.com/lookatitude/beluga-ai/pkg/llms"

func init() {
	// Register Mock provider with the global registry
	llms.GetRegistry().Register("mock", NewMockProviderFactory())
}
