package groq

import "github.com/lookatitude/beluga-ai/pkg/llms"

func init() {
	// Register Groq provider with the global registry
	llms.GetRegistry().Register("groq", NewGroqProviderFactory())
}
