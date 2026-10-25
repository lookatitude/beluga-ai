package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
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

// FI-32: sessions target the GA Realtime API and understand GA event names.
func TestStart_GARealtimeProtocol(t *testing.T) {
	var betaHeader, model string
	audio := []byte("pcm")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		betaHeader = r.Header.Get("OpenAI-Beta")
		model = r.URL.Query().Get("model")
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var msg map[string]any
		_ = json.Unmarshal(data, &msg)
		sess, _ := msg["session"].(map[string]any)
		if sess["type"] != "realtime" {
			_ = conn.Close(websocket.StatusPolicyViolation, "not GA")
			return
		}
		for _, ev := range []map[string]any{
			{"type": "response.output_audio.delta", "delta": base64.StdEncoding.EncodeToString(audio)},
			{"type": "response.output_audio_transcript.delta", "delta": "hi"},
			{"type": "response.done"},
		} {
			b, _ := json.Marshal(ev)
			_ = conn.Write(ctx, websocket.MessageText, b)
		}
		time.Sleep(100 * time.Millisecond)
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}))
	defer srv.Close()

	e, err := New(s2s.Config{Extra: map[string]any{"api_key": "sk", "base_url": "ws" + strings.TrimPrefix(srv.URL, "http")}})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sess, err := e.Start(ctx)
	require.NoError(t, err)
	defer sess.Close()

	var got []s2s.SessionEvent
	for ev, err := range sess.Recv(ctx) {
		require.NoError(t, err)
		got = append(got, ev)
		if len(got) == 3 {
			break
		}
	}
	require.Len(t, got, 3)
	assert.Equal(t, s2s.EventAudioOutput, got[0].Type)
	assert.Equal(t, audio, got[0].Audio)
	assert.Equal(t, s2s.SessionEvent{Type: s2s.EventTextOutput, Text: "hi"}, got[1])
	assert.Equal(t, s2s.EventTurnEnd, got[2].Type)
	assert.Empty(t, betaHeader)
	assert.Equal(t, "gpt-realtime-2.1", model)
}
