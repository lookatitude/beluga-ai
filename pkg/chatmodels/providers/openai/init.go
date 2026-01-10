package openai

import (
	"github.com/lookatitude/beluga-ai/pkg/chatmodels"
	"github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
)

func init() {
	// Register OpenAI provider with the global registry
	chatmodels.GetRegistry().Register("openai", func(model string, config *chatmodels.Config, options *iface.Options) (iface.ChatModel, error) {
		return NewOpenAIChatModel(model, config, options)
	})
}
