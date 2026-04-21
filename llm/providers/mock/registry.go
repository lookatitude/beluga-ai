package mock

import (
	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/llm"
)

func init() {
	llm.Register("mock", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}
