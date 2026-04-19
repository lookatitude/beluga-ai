package groq

import (
	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/v2/llm"
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
