// Package deepseek provides the DeepSeek LLM provider for the Beluga AI framework.
// DeepSeek exposes an OpenAI-compatible API, so this provider is a thin wrapper
// around the shared openaicompat package with DeepSeek's base URL.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/deepseek"
//
//	model, err := llm.New("deepseek", config.ProviderConfig{
//	    Model:  "deepseek-chat",
//	    APIKey: "sk-...",
//	})
package deepseek

import (
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/llm"
)

const defaultBaseURL = "https://api.deepseek.com/v1"

func init() {
	llm.Register("deepseek", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new DeepSeek ChatModel.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.Model == "" {
		cfg.Model = "deepseek-chat"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	return openaicompat.New(cfg)
}
