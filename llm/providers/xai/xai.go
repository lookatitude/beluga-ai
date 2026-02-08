// Package xai provides the xAI Grok LLM provider for the Beluga AI framework.
// xAI exposes an OpenAI-compatible API, so this provider is a thin wrapper
// around the shared openaicompat package with xAI's base URL.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/xai"
//
//	model, err := llm.New("xai", config.ProviderConfig{
//	    Model:  "grok-3",
//	    APIKey: "xai-...",
//	})
package xai

import (
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/llm"
)

const defaultBaseURL = "https://api.x.ai/v1"

func init() {
	llm.Register("xai", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new xAI Grok ChatModel.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.Model == "" {
		cfg.Model = "grok-3"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	return openaicompat.New(cfg)
}
