package websocket

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/transport"
)

func init() {
	// Register WebSocket provider with the global registry
	transport.GetRegistry().Register("websocket", NewWebSocketTransport)
}
