// Package litellm provides a ChatModel backed by a LiteLLM gateway.
// LiteLLM (https://litellm.ai) is a proxy that exposes an OpenAI-compatible
// API in front of 100+ LLM providers. This provider is a thin wrapper around
// the shared openaicompat package with the LiteLLM base URL.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/litellm"
//
//	model, err := llm.New("litellm", config.ProviderConfig{
//	    Model:   "gpt-4o",
//	    APIKey:  "sk-...",
//	    BaseURL: "http://localhost:4000/v1",
//	})
package litellm

import (
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/llm"
)

const defaultBaseURL = "http://localhost:4000/v1"

func init() {
	llm.Register("litellm", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new LiteLLM ChatModel.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.Model == "" {
		cfg.Model = "gpt-4o"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	return openaicompat.New(cfg)
}
