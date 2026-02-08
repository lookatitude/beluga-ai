// Package groq provides the Groq LLM provider for the Beluga AI framework.
// Groq exposes an OpenAI-compatible API, so this provider is a thin wrapper
// around the shared openaicompat package with Groq's base URL.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/groq"
//
//	model, err := llm.New("groq", config.ProviderConfig{
//	    Model:  "llama-3.3-70b-versatile",
//	    APIKey: "gsk_...",
//	})
package groq

import (
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/llm"
)

const defaultBaseURL = "https://api.groq.com/openai/v1"

func init() {
	llm.Register("groq", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new Groq ChatModel.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	return openaicompat.New(cfg)
}
