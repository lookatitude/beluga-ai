package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type wsTestMsg struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

func newWSTestServer(t *testing.T, handler func(*websocket.Conn)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Logf("websocket accept error: %v", err)
			return
		}
		handler(conn)
	}))
}

func TestDialWS(t *testing.T) {
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, err := DialWS(context.Background(), wsURL, nil)
	require.NoError(t, err)
	require.NotNil(t, ws)
	ws.Close()
}

func TestWSReadWriteJSON(t *testing.T) {
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		// Echo: read a message and send it back.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		conn.Write(ctx, websocket.MessageText, data)
		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, err := DialWS(context.Background(), wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	ctx := context.Background()

	// Write a JSON message.
	sent := wsTestMsg{Type: "greeting", Payload: "hello"}
	require.NoError(t, ws.WriteJSON(ctx, sent))

	// Read it back (echoed by server).
	var received wsTestMsg
	require.NoError(t, ws.ReadJSON(ctx, &received))

	assert.Equal(t, sent.Type, received.Type)
	assert.Equal(t, sent.Payload, received.Payload)
}

func TestWSClose(t *testing.T) {
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		// Wait for close from client, then close our end.
		conn.Read(context.Background())
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, err := DialWS(context.Background(), wsURL, nil)
	require.NoError(t, err)

	err = ws.Close()
	require.NoError(t, err)
}

func TestDialWS_WithHeaders(t *testing.T) {
	var receivedHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Custom-Header")
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		conn.Close(websocket.StatusNormalClosure, "")
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	headers := http.Header{}
	headers.Set("X-Custom-Header", "test-value")

	ws, err := DialWS(context.Background(), wsURL, headers)
	require.NoError(t, err)
	ws.Close()

	assert.Equal(t, "test-value", receivedHeader)
}
