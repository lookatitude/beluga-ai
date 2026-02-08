// Package openrouter provides the OpenRouter LLM provider for the Beluga AI framework.
// OpenRouter exposes an OpenAI-compatible API that routes to many different model
// providers, so this is a thin wrapper around the shared openaicompat package.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/openrouter"
//
//	model, err := llm.New("openrouter", config.ProviderConfig{
//	    Model:  "anthropic/claude-sonnet-4-5-20250929",
//	    APIKey: "sk-or-...",
//	})
package openrouter

import (
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/llm"
)

const defaultBaseURL = "https://openrouter.ai/api/v1"

func init() {
	llm.Register("openrouter", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new OpenRouter ChatModel.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	return openaicompat.New(cfg)
}
