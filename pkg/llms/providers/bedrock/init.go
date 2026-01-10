package bedrock

import "github.com/lookatitude/beluga-ai/pkg/llms"

func init() {
	// Register Bedrock provider with the global registry
	llms.GetRegistry().Register("bedrock", NewBedrockProviderFactory())
}
