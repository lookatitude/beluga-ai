// Package fireworks provides the Fireworks AI LLM provider for the Beluga AI framework.
// Fireworks AI exposes an OpenAI-compatible API, so this provider is a thin wrapper
// around the shared openaicompat package with Fireworks' base URL.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/fireworks"
//
//	model, err := llm.New("fireworks", config.ProviderConfig{
//	    Model:  "accounts/fireworks/models/llama-v3p1-70b-instruct",
//	    APIKey: "fw_...",
//	})
package fireworks

import (
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/llm"
)

const defaultBaseURL = "https://api.fireworks.ai/inference/v1"

func init() {
	llm.Register("fireworks", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new Fireworks AI ChatModel.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.Model == "" {
		cfg.Model = "accounts/fireworks/models/llama-v3p1-70b-instruct"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	return openaicompat.New(cfg)
}
