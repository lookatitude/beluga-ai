package ollama

import (
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/llm"
)

const defaultBaseURL = "http://localhost:11434/v1"

func init() {
	llm.Register("ollama", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new Ollama ChatModel using the OpenAI-compatible endpoint.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.APIKey == "" {
		cfg.APIKey = "ollama"
	}
	return openaicompat.New(cfg)
}
