package gemini

import "github.com/lookatitude/beluga-ai/pkg/llms"

func init() {
	// Register Gemini provider with the global registry
	llms.GetRegistry().Register("gemini", NewGeminiProviderFactory())
}
