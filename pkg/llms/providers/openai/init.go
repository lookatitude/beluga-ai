package openai

import "github.com/lookatitude/beluga-ai/pkg/llms"

func init() {
	// Register OpenAI provider with the global registry
	llms.GetRegistry().Register("openai", NewOpenAIProviderFactory())
}
