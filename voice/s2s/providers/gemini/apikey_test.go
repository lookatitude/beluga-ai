package gemini

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

	"github.com/lookatitude/beluga-ai/v2/voice/s2s"
)

// FI-05: the API key must be sent as a header, not in the URL.
func TestStart_APIKeyInHeaderNotURL(t *testing.T) {
	var gotHeader, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("x-goog-api-key")
		gotQuery = r.URL.RawQuery
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		_, _, _ = conn.Read(r.Context())
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}))
	defer srv.Close()

	e, err := New(s2s.Config{Extra: map[string]any{
		"api_key": "SECRET-KEY-123", "base_url": "ws" + strings.TrimPrefix(srv.URL, "http"),
	}})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sess, err := e.Start(ctx)
	require.NoError(t, err)
	_ = sess.Close()

	assert.Equal(t, "SECRET-KEY-123", gotHeader)
	assert.NotContains(t, gotQuery, "SECRET-KEY-123")
}

func TestStart_DialErrorDoesNotLeakKey(t *testing.T) {
	e, err := New(s2s.Config{Extra: map[string]any{
		"api_key": "SECRET-KEY-123", "base_url": "ws://127.0.0.1:1/ws",
	}})
	require.NoError(t, err)
	_, err = e.Start(context.Background())
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "SECRET-KEY-123")
}
