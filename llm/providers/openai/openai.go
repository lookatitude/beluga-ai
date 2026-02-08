// Package openai provides the OpenAI LLM provider for the Beluga AI framework.
// It implements the llm.ChatModel interface using the openai-go SDK via the
// shared openaicompat package.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
//
//	model, err := llm.New("openai", config.ProviderConfig{
//	    Model:  "gpt-4o",
//	    APIKey: "sk-...",
//	})
package openai

import (
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/llm"
)

const defaultBaseURL = "https://api.openai.com/v1"

func init() {
	llm.Register("openai", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new OpenAI ChatModel.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	return openaicompat.New(cfg)
}
