package openai_realtime

import (
	"time"

	"github.com/gorilla/websocket"
)

// defaultWebSocketDialer adapts gorilla/websocket.Dialer to WebSocketDialer interface.
type defaultWebSocketDialer struct {
	dialer *websocket.Dialer
}

// Dial implements WebSocketDialer interface.
func (d *defaultWebSocketDialer) Dial(url string, headers map[string][]string) (WebSocketConn, error) {
	conn, _, err := d.dialer.Dial(url, headers)
	if err != nil {
		return nil, err
	}
	return &websocketConnAdapter{conn: conn}, nil
}

// websocketConnAdapter adapts gorilla/websocket.Conn to WebSocketConn interface.
type websocketConnAdapter struct {
	conn *websocket.Conn
}

// ReadJSON implements WebSocketConn interface.
func (w *websocketConnAdapter) ReadJSON(v any) error {
	return w.conn.ReadJSON(v)
}

// WriteJSON implements WebSocketConn interface.
func (w *websocketConnAdapter) WriteJSON(v any) error {
	return w.conn.WriteJSON(v)
}

// Close implements WebSocketConn interface.
func (w *websocketConnAdapter) Close() error {
	return w.conn.Close()
}

// SetReadDeadline sets the read deadline (optional, for compatibility).
func (w *websocketConnAdapter) SetReadDeadline(t time.Time) error {
	return w.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline (optional, for compatibility).
func (w *websocketConnAdapter) SetWriteDeadline(t time.Time) error {
	return w.conn.SetWriteDeadline(t)
}
