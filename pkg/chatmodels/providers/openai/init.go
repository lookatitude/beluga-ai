package openai

import (
	"github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
)

func init() {
	// Register OpenAI provider with the global registry
	// Use iface package directly to avoid import cycles
	iface.GetRegistry().Register("openai", func(model string, config any, options *iface.Options) (iface.ChatModel, error) {
		// NewOpenAIChatModel accepts any for config, so we can pass it through directly
		// The actual provider implementation will handle config validation if needed
		return NewOpenAIChatModel(model, config, options)
	})
}
