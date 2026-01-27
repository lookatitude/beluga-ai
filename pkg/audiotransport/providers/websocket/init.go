package websocket

import (
	"github.com/lookatitude/beluga-ai/pkg/audiotransport"
)

func init() {
	// Register WebSocket provider with the global registry
	audiotransport.GetRegistry().Register("websocket", NewWebSocketTransport)
}
