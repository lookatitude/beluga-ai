// Package ollama provides the Ollama LLM provider for the Beluga AI framework.
// Ollama exposes an OpenAI-compatible API, so this provider uses the shared
// openaicompat package with Ollama's local base URL.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/ollama"
//
//	model, err := llm.New("ollama", config.ProviderConfig{
//	    Model:   "llama3.2",
//	    BaseURL: "http://localhost:11434/v1",
//	})
package ollama

import (
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/llm"
)

const defaultBaseURL = "http://localhost:11434/v1"

func init() {
	llm.Register("ollama", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new Ollama ChatModel using the OpenAI-compatible endpoint.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.APIKey == "" {
		cfg.APIKey = "ollama"
	}
	return openaicompat.New(cfg)
}
