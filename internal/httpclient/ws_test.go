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

func TestWSReadJSON_MalformedJSON(t *testing.T) {
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// Send malformed JSON.
		conn.Write(ctx, websocket.MessageText, []byte("{not valid json"))
		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, err := DialWS(context.Background(), wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	var msg wsTestMsg
	err = ws.ReadJSON(context.Background(), &msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestWSReadJSON_ConnectionClosed(t *testing.T) {
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		// Close immediately without sending data.
		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, err := DialWS(context.Background(), wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	var msg wsTestMsg
	err = ws.ReadJSON(context.Background(), &msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "websocket read")
}

func TestWSWriteJSON_MarshalError(t *testing.T) {
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, err := DialWS(context.Background(), wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Try to marshal something that will fail (channels can't be marshaled).
	type invalidMsg struct {
		Channel chan int
	}
	err = ws.WriteJSON(context.Background(), invalidMsg{Channel: make(chan int)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal")
}

func TestWSWriteJSON_ConnectionClosed(t *testing.T) {
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, err := DialWS(context.Background(), wsURL, nil)
	require.NoError(t, err)

	// Close the connection.
	ws.Close()

	// Try to write after close.
	err = ws.WriteJSON(context.Background(), wsTestMsg{Type: "test", Payload: "data"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "websocket write")
}

func TestDialWS_InvalidURL(t *testing.T) {
	_, err := DialWS(context.Background(), "invalid://url", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "websocket dial")
}

func TestDialWS_ContextCanceled(t *testing.T) {
	srv := newWSTestServer(t, func(conn *websocket.Conn) {
		time.Sleep(1 * time.Second)
		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	_, err := DialWS(ctx, wsURL, nil)
	require.Error(t, err)
}
