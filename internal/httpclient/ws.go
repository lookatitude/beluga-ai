package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coder/websocket"
)

// WSConn wraps a WebSocket connection with typed JSON helpers.
type WSConn struct {
	conn *websocket.Conn
}

// DialWS opens a WebSocket connection.
func DialWS(ctx context.Context, url string, headers http.Header) (*WSConn, error) {
	opts := &websocket.DialOptions{}
	if headers != nil {
		opts.HTTPHeader = headers
	}
	conn, _, err := websocket.Dial(ctx, url, opts)
	if err != nil {
		return nil, fmt.Errorf("httpclient: websocket dial: %w", err)
	}
	return &WSConn{conn: conn}, nil
}

// ReadJSON reads and decodes a JSON message from the WebSocket.
func (ws *WSConn) ReadJSON(ctx context.Context, v any) error {
	_, data, err := ws.conn.Read(ctx)
	if err != nil {
		return fmt.Errorf("httpclient: websocket read: %w", err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("httpclient: websocket unmarshal: %w", err)
	}
	return nil
}

// WriteJSON encodes and sends a JSON message over the WebSocket.
func (ws *WSConn) WriteJSON(ctx context.Context, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("httpclient: websocket marshal: %w", err)
	}
	if err := ws.conn.Write(ctx, websocket.MessageText, data); err != nil {
		return fmt.Errorf("httpclient: websocket write: %w", err)
	}
	return nil
}

// Close gracefully closes the WebSocket connection.
func (ws *WSConn) Close() error {
	return ws.conn.Close(websocket.StatusNormalClosure, "")
}
