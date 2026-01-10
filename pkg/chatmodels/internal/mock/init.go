package mock

import (
	"github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
	"github.com/lookatitude/beluga-ai/pkg/chatmodels/registry"
)

func init() {
	// Register Mock provider with the global registry
	// Use registry package directly to avoid import cycles in tests
	// Note: We don't import chatmodels here to avoid cycles - the factory accepts any for config
	registry.GetRegistry().Register("mock", func(model string, config any, options *iface.Options) (iface.ChatModel, error) {
		// NewMockChatModel accepts any for config, so we can pass it through directly
		// The actual provider implementation will handle config validation if needed
		return NewMockChatModel(model, config, options)
	})
}
