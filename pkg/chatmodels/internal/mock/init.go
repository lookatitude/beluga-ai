package mock

import (
	"github.com/lookatitude/beluga-ai/pkg/chatmodels"
	"github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
)

func init() {
	// Register Mock provider with the global registry
	chatmodels.GetRegistry().Register("mock", func(model string, config *chatmodels.Config, options *iface.Options) (iface.ChatModel, error) {
		return NewMockChatModel(model, config, options)
	})
}
