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

func TestSendText(t *testing.T) {
	srv := newWSServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Read setup.
		conn.Read(ctx)

		// Read text message.
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var msg map[string]any
		json.Unmarshal(data, &msg)
		assert.Equal(t, "inputText", msg["type"])
		content := msg["content"].([]any)
		require.Len(t, content, 1)
		first := content[0].(map[string]any)
		assert.Equal(t, "hello nova", first["text"])

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

	err = session.SendText(ctx, "hello nova")
	require.NoError(t, err)
}

func TestStartWithOptions(t *testing.T) {
	t.Run("with instructions and tools", func(t *testing.T) {
		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Read setup and verify it contains instructions and tools.
			_, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			var msg map[string]any
			json.Unmarshal(data, &msg)
			assert.Equal(t, "sessionStart", msg["type"])

			// Verify instructions.
			system := msg["system"].([]any)
			require.Len(t, system, 1)
			first := system[0].(map[string]any)
			assert.Equal(t, "You are a helpful assistant", first["text"])

			// Verify tool config.
			toolConfig := msg["toolConfig"].(map[string]any)
			tools := toolConfig["tools"].([]any)
			require.Len(t, tools, 1)
			toolSpec := tools[0].(map[string]any)["toolSpec"].(map[string]any)
			assert.Equal(t, "get_weather", toolSpec["name"])

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

		session, err := e.Start(ctx,
			s2s.WithInstructions("You are a helpful assistant"),
			s2s.WithTools([]schema.ToolDefinition{
				{
					Name:        "get_weather",
					Description: "Get weather for a city",
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"city": map[string]any{"type": "string"},
						},
					},
				},
			}),
		)
		require.NoError(t, err)
		defer session.Close()
	})
}

func TestReadLoopEvents(t *testing.T) {
	t.Run("text output in contentBlockDelta", func(t *testing.T) {
		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn.Read(ctx) // setup

			// Send text delta.
			event := novaServerEvent{
				Type: "contentBlockDelta",
				Text: "Hello from Nova",
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
			assert.Equal(t, s2s.EventTextOutput, event.Type)
			assert.Equal(t, "Hello from Nova", event.Text)
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for text event")
		}
	})

	t.Run("input transcript", func(t *testing.T) {
		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn.Read(ctx) // setup

			event := novaServerEvent{
				Type:       "inputTranscript",
				Transcript: "user said hello",
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
			assert.Equal(t, s2s.EventTranscript, event.Type)
			assert.Equal(t, "user said hello", event.Text)
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for transcript event")
		}
	})

	t.Run("error event", func(t *testing.T) {
		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn.Read(ctx) // setup

			event := novaServerEvent{
				Type:  "error",
				Error: &novaError{Message: "rate limit exceeded"},
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
			assert.Equal(t, s2s.EventError, event.Type)
			assert.Contains(t, event.Error.Error(), "rate limit exceeded")
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for error event")
		}
	})

	t.Run("error event with nil error field", func(t *testing.T) {
		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn.Read(ctx) // setup

			event := novaServerEvent{
				Type: "error",
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
			assert.Equal(t, s2s.EventError, event.Type)
			assert.Contains(t, event.Error.Error(), "unknown error")
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for error event")
		}
	})

	t.Run("messageStop event", func(t *testing.T) {
		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn.Read(ctx) // setup

			event := novaServerEvent{Type: "messageStop"}
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
			assert.Equal(t, s2s.EventTurnEnd, event.Type)
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for messageStop event")
		}
	})

	t.Run("connection error emits error event", func(t *testing.T) {
		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn.Read(ctx) // setup

			// Close abruptly without normal closure.
			conn.Close(websocket.StatusInternalError, "server crash")
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
			assert.Equal(t, s2s.EventError, event.Type)
			assert.Contains(t, event.Error.Error(), "nova: read:")
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for connection error event")
		}
	})

	t.Run("audio and text in same contentBlockDelta", func(t *testing.T) {
		audioData := []byte("pcm-data")
		encodedAudio := base64.StdEncoding.EncodeToString(audioData)

		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn.Read(ctx) // setup

			// Send event with both audio and text.
			event := novaServerEvent{
				Type:       "contentBlockDelta",
				AudioChunk: encodedAudio,
				Text:       "caption text",
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

		var events []s2s.SessionEvent
		timeout := time.After(3 * time.Second)
		for {
			select {
			case event, ok := <-session.Recv():
				if !ok {
					goto done
				}
				events = append(events, event)
				if len(events) >= 2 {
					goto done
				}
			case <-timeout:
				goto done
			}
		}
	done:
		require.Len(t, events, 2)
		assert.Equal(t, s2s.EventAudioOutput, events[0].Type)
		assert.Equal(t, audioData, events[0].Audio)
		assert.Equal(t, s2s.EventTextOutput, events[1].Type)
		assert.Equal(t, "caption text", events[1].Text)
	})
}

func TestStartDialError(t *testing.T) {
	e, err := New(s2s.Config{
		Extra: map[string]any{"base_url": "ws://127.0.0.1:1"},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = e.Start(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nova: websocket dial:")
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

	t.Run("duplicate registration panics", func(t *testing.T) {
		assert.Panics(t, func() {
			s2s.Register("nova", func(cfg s2s.Config) (s2s.S2S, error) {
				return nil, nil
			})
		})
	})

	t.Run("factory creates engine", func(t *testing.T) {
		engine, err := s2s.New("nova", s2s.Config{})
		require.NoError(t, err)
		assert.NotNil(t, engine)
	})
}
