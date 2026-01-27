package mock

import (
	"github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
)

func init() {
	// Register Mock provider with the global registry
	// Use iface package directly to avoid import cycles
	iface.GetRegistry().Register("mock", func(model string, config any, options *iface.Options) (iface.ChatModel, error) {
		// NewMockChatModel accepts any for config, so we can pass it through directly
		// The actual provider implementation will handle config validation if needed
		return NewMockChatModel(model, config, options)
	})
}
