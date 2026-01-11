package google

import "github.com/lookatitude/beluga-ai/pkg/llms"

func init() {
	// Register Google provider with the global registry
	llms.GetRegistry().Register("google", NewGoogleProviderFactory())
}
