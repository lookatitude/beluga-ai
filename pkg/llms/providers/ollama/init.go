package ollama

import "github.com/lookatitude/beluga-ai/pkg/llms"

func init() {
	// Register Ollama provider with the global registry
	llms.GetRegistry().Register("ollama", NewOllamaProviderFactory())
}
