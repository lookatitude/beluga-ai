// Package cerebras provides the Cerebras LLM provider for the Beluga AI framework.
// Cerebras exposes an OpenAI-compatible API, so this provider is a thin wrapper
// around the shared openaicompat package with Cerebras' base URL.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/cerebras"
//
//	model, err := llm.New("cerebras", config.ProviderConfig{
//	    Model:  "llama-3.3-70b",
//	    APIKey: "csk-...",
//	})
package cerebras

import (
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/llm"
)

const defaultBaseURL = "https://api.cerebras.ai/v1"

func init() {
	llm.Register("cerebras", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new Cerebras ChatModel.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	return openaicompat.New(cfg)
}
