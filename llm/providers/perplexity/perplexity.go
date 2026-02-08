// Package perplexity provides the Perplexity LLM provider for the Beluga AI framework.
// Perplexity exposes an OpenAI-compatible API, so this provider is a thin wrapper
// around the shared openaicompat package with Perplexity's base URL.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/perplexity"
//
//	model, err := llm.New("perplexity", config.ProviderConfig{
//	    Model:  "sonar-pro",
//	    APIKey: "pplx-...",
//	})
package perplexity

import (
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/llm"
)

const defaultBaseURL = "https://api.perplexity.ai"

func init() {
	llm.Register("perplexity", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new Perplexity ChatModel.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	return openaicompat.New(cfg)
}
