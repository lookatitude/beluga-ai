// Package together provides the Together AI LLM provider for the Beluga AI framework.
// Together AI exposes an OpenAI-compatible API, so this provider is a thin wrapper
// around the shared openaicompat package with Together's base URL.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/together"
//
//	model, err := llm.New("together", config.ProviderConfig{
//	    Model:  "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo",
//	    APIKey: "...",
//	})
package together

import (
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/llm"
)

const defaultBaseURL = "https://api.together.xyz/v1"

func init() {
	llm.Register("together", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new Together AI ChatModel.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.Model == "" {
		cfg.Model = "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	return openaicompat.New(cfg)
}
