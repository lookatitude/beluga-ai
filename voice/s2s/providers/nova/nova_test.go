package nova

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

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/voice/s2s"
)

func TestNew(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		e, err := New(s2s.Config{})
		require.NoError(t, err)
		assert.Equal(t, defaultModel, e.cfg.Model)
		assert.Equal(t, defaultRegion, e.region)
	})

	t.Run("custom config", func(t *testing.T) {
		e, err := New(s2s.Config{
			Model: "custom-model",
			Extra: map[string]any{"region": "eu-west-1"},
		})
		require.NoError(t, err)
		assert.Equal(t, "custom-model", e.cfg.Model)
		assert.Equal(t, "eu-west-1", e.region)
	})

	t.Run("custom base url", func(t *testing.T) {
		e, err := New(s2s.Config{
			Extra: map[string]any{
				"base_url": "wss://custom.bedrock.aws/model/test",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "wss://custom.bedrock.aws/model/test", e.baseURL)
	})
}

func newWSServer(t *testing.T, handler func(*websocket.Conn)) *httptest.Server {
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

func TestStartAndRecv(t *testing.T) {
	t.Run("receive audio output", func(t *testing.T) {
		audioData := []byte("test-pcm-audio")
		encodedAudio := base64.StdEncoding.EncodeToString(audioData)

		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Read setup.
			_, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			var msg map[string]any
			json.Unmarshal(data, &msg)
			assert.Equal(t, "sessionStart", msg["type"])

			// Send audio delta.
			event := novaServerEvent{
				Type:       "contentBlockDelta",
				AudioChunk: encodedAudio,
			}
			eventData, _ := json.Marshal(event)
			conn.Write(ctx, websocket.MessageText, eventData)

			// Send turn end.
			endEvent := novaServerEvent{Type: "contentBlockStop"}
			endData, _ := json.Marshal(endEvent)
			conn.Write(ctx, websocket.MessageText, endData)

			time.Sleep(100 * time.Millisecond)
			conn.Close(websocket.StatusNormalClosure, "")
		})
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, err := New(s2s.Config{
			Extra: map[string]any{"base_url": wsURL},
		})
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		session, err := e.Start(ctx)
		require.NoError(t, err)
		defer session.Close()

		var events []s2s.SessionEvent
		timeout := time.After(3 * time.Second)
		for {
			select {
			case event, ok := <-session.Recv():
				if !ok {
					goto done
				}
				events = append(events, event)
				if event.Type == s2s.EventTurnEnd {
					goto done
				}
			case <-timeout:
				goto done
			}
		}
	done:
		require.GreaterOrEqual(t, len(events), 2)
		assert.Equal(t, s2s.EventAudioOutput, events[0].Type)
		assert.Equal(t, audioData, events[0].Audio)
		assert.Equal(t, s2s.EventTurnEnd, events[len(events)-1].Type)
	})

	t.Run("receive tool call", func(t *testing.T) {
		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn.Read(ctx)

			event := novaServerEvent{
				Type: "toolUse",
				ToolUse: &novaToolUse{
					ToolUseID: "tu_123",
					Name:      "get_weather",
					Input:     json.RawMessage(`{"city":"NYC"}`),
				},
			}
			data, _ := json.Marshal(event)
			conn.Write(ctx, websocket.MessageText, data)

			time.Sleep(100 * time.Millisecond)
			conn.Close(websocket.StatusNormalClosure, "")
		})
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, err := New(s2s.Config{
			Extra: map[string]any{"base_url": wsURL},
		})
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		session, err := e.Start(ctx)
		require.NoError(t, err)
		defer session.Close()

		select {
		case event := <-session.Recv():
			assert.Equal(t, s2s.EventToolCall, event.Type)
			require.NotNil(t, event.ToolCall)
			assert.Equal(t, "tu_123", event.ToolCall.ID)
			assert.Equal(t, "get_weather", event.ToolCall.Name)
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for tool call event")
		}
	})
}

func TestSendAudio(t *testing.T) {
	srv := newWSServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn.Read(ctx)

		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var msg map[string]any
		json.Unmarshal(data, &msg)
		assert.Equal(t, "inputAudio", msg["type"])
		assert.NotEmpty(t, msg["audioChunk"])

		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, err := New(s2s.Config{
		Extra: map[string]any{"base_url": wsURL},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := e.Start(ctx)
	require.NoError(t, err)
	defer session.Close()

	err = session.SendAudio(ctx, []byte("audio-data"))
	require.NoError(t, err)
}

func TestSendToolResult(t *testing.T) {
	srv := newWSServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn.Read(ctx)

		_, data, _ := conn.Read(ctx)
		var msg map[string]any
		json.Unmarshal(data, &msg)
		assert.Equal(t, "toolResult", msg["type"])
		tr := msg["toolResult"].(map[string]any)
		assert.Equal(t, "tu_123", tr["toolUseId"])

		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, err := New(s2s.Config{
		Extra: map[string]any{"base_url": wsURL},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := e.Start(ctx)
	require.NoError(t, err)
	defer session.Close()

	err = session.SendToolResult(ctx, schema.ToolResult{
		CallID: "tu_123",
		Content: []schema.ContentPart{
			schema.TextPart{Text: "72F sunny"},
		},
	})
	require.NoError(t, err)
}

func TestInterrupt(t *testing.T) {
	srv := newWSServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn.Read(ctx)

		_, data, _ := conn.Read(ctx)
		var msg map[string]any
		json.Unmarshal(data, &msg)
		assert.Equal(t, "inputAudioInterrupt", msg["type"])

		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, err := New(s2s.Config{
		Extra: map[string]any{"base_url": wsURL},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := e.Start(ctx)
	require.NoError(t, err)
	defer session.Close()

	err = session.Interrupt(ctx)
	require.NoError(t, err)
}

func TestRegistry(t *testing.T) {
	t.Run("registered as nova", func(t *testing.T) {
		names := s2s.List()
		found := false
		for _, name := range names {
			if name == "nova" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected 'nova' in registered providers: %v", names)
	})
}
