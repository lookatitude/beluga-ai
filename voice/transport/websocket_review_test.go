package transport

import (
	"context"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FI-37: a network failure must be surfaced by Recv, while a normal close
// ends the stream cleanly.
func TestWebSocketTransport_RecvSurfacesTerminalError(t *testing.T) {
	tests := []struct {
		name    string
		finish  func(*websocket.Conn)
		wantErr bool
	}{
		{name: "abrupt close", finish: func(c *websocket.Conn) { _ = c.CloseNow() }, wantErr: true},
		{name: "protocol error close", finish: func(c *websocket.Conn) { _ = c.Close(websocket.StatusInternalError, "boom") }, wantErr: true},
		{name: "normal close", finish: func(c *websocket.Conn) { _ = c.Close(websocket.StatusNormalClosure, "") }},
		{name: "going away", finish: func(c *websocket.Conn) { _ = c.Close(websocket.StatusGoingAway, "") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newWSTestServer(t, func(conn *websocket.Conn) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = conn.Write(ctx, websocket.MessageBinary, []byte{1, 2})
				tt.finish(conn)
			})
			defer srv.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			tr, err := NewWebSocketTransport(ctx, wsURL(srv))
			require.NoError(t, err)
			defer func() { assert.NoError(t, tr.Close()) }()

			frames, err := drainFrames(tr.Recv(ctx))
			assert.Len(t, frames, 1)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// FI-37: Close after the peer already tore the connection down must succeed.
func TestWebSocketTransport_CloseAfterPeerClose(t *testing.T) {
	for i := 0; i < 20; i++ {
		srv := newWSTestServer(t, func(conn *websocket.Conn) { _ = conn.CloseNow() })
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		tr, err := NewWebSocketTransport(ctx, wsURL(srv))
		require.NoError(t, err)
		_, _ = drainFrames(tr.Recv(ctx))
		require.NoError(t, tr.Close())
		require.NoError(t, tr.Close())
		cancel()
		srv.Close()
	}
}
