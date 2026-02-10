package bifrost

import (
	"fmt"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/llm"
)

func init() {
	llm.Register("bifrost", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new Bifrost ChatModel.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("bifrost: base_url is required")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("bifrost: model is required")
	}
	return openaicompat.New(cfg)
}
