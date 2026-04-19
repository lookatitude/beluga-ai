package bifrost

import (
	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/internal/openaicompat"
	"github.com/lookatitude/beluga-ai/v2/llm"
)

func init() {
	llm.Register("bifrost", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// New creates a new Bifrost ChatModel.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	if cfg.BaseURL == "" {
		return nil, core.Errorf(core.ErrInvalidInput, "bifrost: base_url is required")
	}
	if cfg.Model == "" {
		return nil, core.Errorf(core.ErrInvalidInput, "bifrost: model is required")
	}
	return openaicompat.New(cfg)
}
