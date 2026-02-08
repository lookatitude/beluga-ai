// Package sambanova provides the SambaNova LLM provider for the Beluga AI framework.
// SambaNova exposes an OpenAI-compatible API, so this provider is a thin wrapper
// around the shared openaicompat package with SambaNova's base URL.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/sambanova"
//
//	model, err := llm.New("sambanova", config.ProviderConfig{
//	    Model:  "Meta-Llama-3.3-70B-Instruct",
//	    APIKey: "sn-...",
//	})
package sambanova

import (
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/llm"
)

const defaultBaseURL = "https://api.sambanova.ai/v1"

func init() {
	llm.Register("sambanova", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new SambaNova ChatModel.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	return openaicompat.New(cfg)
}
