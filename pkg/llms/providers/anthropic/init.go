package anthropic

import "github.com/lookatitude/beluga-ai/pkg/llms"

func init() {
	// Register Anthropic provider with the global registry
	llms.GetRegistry().Register("anthropic", NewAnthropicProviderFactory())
}
