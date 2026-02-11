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

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/voice/s2s"
)

func TestNew(t *testing.T) {
	t.Run("missing api key", func(t *testing.T) {
		_, err := New(s2s.Config{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "api_key is required")
	})

	t.Run("valid config", func(t *testing.T) {
		e, err := New(s2s.Config{
			Extra: map[string]any{"api_key": "sk-test"},
		})
		require.NoError(t, err)
		assert.NotNil(t, e)
		assert.Equal(t, defaultModel, e.cfg.Model)
		assert.Equal(t, "alloy", e.cfg.Voice)
	})

	t.Run("custom config", func(t *testing.T) {
		e, err := New(s2s.Config{
			Model: "gpt-4o-mini-realtime",
			Voice: "shimmer",
			Extra: map[string]any{"api_key": "sk-test"},
		})
		require.NoError(t, err)
		assert.Equal(t, "gpt-4o-mini-realtime", e.cfg.Model)
		assert.Equal(t, "shimmer", e.cfg.Voice)
	})

	t.Run("custom base url", func(t *testing.T) {
		e, err := New(s2s.Config{
			Extra: map[string]any{
				"api_key":  "sk-test",
				"base_url": "wss://custom.openai.com/v1/realtime",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "wss://custom.openai.com/v1/realtime", e.baseURL)
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

			// Read session.update.
			_, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			var msg map[string]any
			json.Unmarshal(data, &msg)
			assert.Equal(t, "session.update", msg["type"])

			// Send audio delta.
			event := map[string]any{
				"type":  "response.audio.delta",
				"delta": encodedAudio,
			}
			eventData, _ := json.Marshal(event)
			conn.Write(ctx, websocket.MessageText, eventData)

			// Send turn end.
			endEvent := map[string]any{
				"type": "response.done",
			}
			endData, _ := json.Marshal(endEvent)
			conn.Write(ctx, websocket.MessageText, endData)

			// Keep connection alive briefly.
			time.Sleep(100 * time.Millisecond)
			conn.Close(websocket.StatusNormalClosure, "")
		})
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, err := New(s2s.Config{
			Extra: map[string]any{
				"api_key":  "sk-test",
				"base_url": wsURL,
			},
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

		// First should be audio.
		assert.Equal(t, s2s.EventAudioOutput, events[0].Type)
		assert.Equal(t, audioData, events[0].Audio)

		// Last should be turn end.
		assert.Equal(t, s2s.EventTurnEnd, events[len(events)-1].Type)
	})

	t.Run("receive text and transcript", func(t *testing.T) {
		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Read session.update.
			conn.Read(ctx)

			// Send transcript.
			event := map[string]any{
				"type":       "conversation.item.input_audio_transcription.completed",
				"transcript": "hello user",
			}
			data, _ := json.Marshal(event)
			conn.Write(ctx, websocket.MessageText, data)

			// Send text delta.
			textEvent := map[string]any{
				"type":  "response.audio_transcript.delta",
				"delta": "hello",
			}
			textData, _ := json.Marshal(textEvent)
			conn.Write(ctx, websocket.MessageText, textData)

			time.Sleep(100 * time.Millisecond)
			conn.Close(websocket.StatusNormalClosure, "")
		})
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, err := New(s2s.Config{
			Extra: map[string]any{
				"api_key":  "sk-test",
				"base_url": wsURL,
			},
		})
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		session, err := e.Start(ctx)
		require.NoError(t, err)
		defer session.Close()

		var events []s2s.SessionEvent
		timeout := time.After(2 * time.Second)
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
		require.Equal(t, 2, len(events))
		assert.Equal(t, s2s.EventTranscript, events[0].Type)
		assert.Equal(t, "hello user", events[0].Text)
		assert.Equal(t, s2s.EventTextOutput, events[1].Type)
		assert.Equal(t, "hello", events[1].Text)
	})

	t.Run("receive tool call", func(t *testing.T) {
		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn.Read(ctx)

			event := map[string]any{
				"type":      "response.function_call_arguments.done",
				"call_id":   "call_123",
				"name":      "get_weather",
				"arguments": `{"city":"NYC"}`,
			}
			data, _ := json.Marshal(event)
			conn.Write(ctx, websocket.MessageText, data)

			time.Sleep(100 * time.Millisecond)
			conn.Close(websocket.StatusNormalClosure, "")
		})
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, err := New(s2s.Config{
			Extra: map[string]any{
				"api_key":  "sk-test",
				"base_url": wsURL,
			},
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
			assert.Equal(t, "call_123", event.ToolCall.ID)
			assert.Equal(t, "get_weather", event.ToolCall.Name)
			assert.Equal(t, `{"city":"NYC"}`, event.ToolCall.Arguments)
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for tool call event")
		}
	})
}

func TestSendAudio(t *testing.T) {
	srv := newWSServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Read session.update.
		conn.Read(ctx)

		// Read audio append.
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var msg map[string]any
		json.Unmarshal(data, &msg)
		assert.Equal(t, "input_audio_buffer.append", msg["type"])
		assert.NotEmpty(t, msg["audio"])

		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, err := New(s2s.Config{
		Extra: map[string]any{
			"api_key":  "sk-test",
			"base_url": wsURL,
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := e.Start(ctx)
	require.NoError(t, err)
	defer session.Close()

	err = session.SendAudio(ctx, []byte("audio-pcm-data"))
	require.NoError(t, err)
}

func TestSendText(t *testing.T) {
	srv := newWSServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn.Read(ctx)

		// Read conversation.item.create.
		_, data, _ := conn.Read(ctx)
		var msg map[string]any
		json.Unmarshal(data, &msg)
		assert.Equal(t, "conversation.item.create", msg["type"])

		// Read response.create.
		_, data, _ = conn.Read(ctx)
		json.Unmarshal(data, &msg)
		assert.Equal(t, "response.create", msg["type"])

		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, err := New(s2s.Config{
		Extra: map[string]any{
			"api_key":  "sk-test",
			"base_url": wsURL,
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := e.Start(ctx)
	require.NoError(t, err)
	defer session.Close()

	err = session.SendText(ctx, "hello")
	require.NoError(t, err)
}

func TestSendToolResult(t *testing.T) {
	srv := newWSServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn.Read(ctx)

		// Read tool result.
		_, data, _ := conn.Read(ctx)
		var msg map[string]any
		json.Unmarshal(data, &msg)
		assert.Equal(t, "conversation.item.create", msg["type"])
		item := msg["item"].(map[string]any)
		assert.Equal(t, "function_call_output", item["type"])
		assert.Equal(t, "call_123", item["call_id"])
		assert.Equal(t, "result text", item["output"])

		// Read response.create.
		conn.Read(ctx)

		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, err := New(s2s.Config{
		Extra: map[string]any{
			"api_key":  "sk-test",
			"base_url": wsURL,
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := e.Start(ctx)
	require.NoError(t, err)
	defer session.Close()

	err = session.SendToolResult(ctx, schema.ToolResult{
		CallID: "call_123",
		Content: []schema.ContentPart{
			schema.TextPart{Text: "result text"},
		},
	})
	require.NoError(t, err)
}

func TestInterrupt(t *testing.T) {
	srv := newWSServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn.Read(ctx)

		// Read response.cancel.
		_, data, _ := conn.Read(ctx)
		var msg map[string]any
		json.Unmarshal(data, &msg)
		assert.Equal(t, "response.cancel", msg["type"])

		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, err := New(s2s.Config{
		Extra: map[string]any{
			"api_key":  "sk-test",
			"base_url": wsURL,
		},
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

func TestStartWithOptions(t *testing.T) {
	t.Run("with instructions and tools", func(t *testing.T) {
		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Read session.update and verify it contains instructions and tools.
			_, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			var msg map[string]any
			json.Unmarshal(data, &msg)
			assert.Equal(t, "session.update", msg["type"])

			session := msg["session"].(map[string]any)
			assert.Equal(t, "You are a helpful assistant", session["instructions"])
			assert.Equal(t, "nova", session["voice"])

			tools := session["tools"].([]any)
			require.Len(t, tools, 1)
			tool := tools[0].(map[string]any)
			assert.Equal(t, "function", tool["type"])
			assert.Equal(t, "get_weather", tool["name"])

			time.Sleep(100 * time.Millisecond)
			conn.Close(websocket.StatusNormalClosure, "")
		})
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, err := New(s2s.Config{
			Extra: map[string]any{
				"api_key":  "sk-test",
				"base_url": wsURL,
			},
		})
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		session, err := e.Start(ctx,
			s2s.WithVoice("nova"),
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
	t.Run("error event with message", func(t *testing.T) {
		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn.Read(ctx) // session.update

			event := map[string]any{
				"type": "error",
				"error": map[string]any{
					"type":    "invalid_request_error",
					"message": "rate limit exceeded",
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
			Extra: map[string]any{
				"api_key":  "sk-test",
				"base_url": wsURL,
			},
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

			conn.Read(ctx) // session.update

			event := map[string]any{
				"type": "error",
			}
			data, _ := json.Marshal(event)
			conn.Write(ctx, websocket.MessageText, data)

			time.Sleep(100 * time.Millisecond)
			conn.Close(websocket.StatusNormalClosure, "")
		})
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, err := New(s2s.Config{
			Extra: map[string]any{
				"api_key":  "sk-test",
				"base_url": wsURL,
			},
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

	t.Run("connection error emits error event", func(t *testing.T) {
		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn.Read(ctx) // session.update

			// Close abruptly.
			conn.Close(websocket.StatusInternalError, "server crash")
		})
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, err := New(s2s.Config{
			Extra: map[string]any{
				"api_key":  "sk-test",
				"base_url": wsURL,
			},
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
			assert.Contains(t, event.Error.Error(), "openai realtime: read:")
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for connection error event")
		}
	})
}

func TestSendTextWriteError(t *testing.T) {
	srv := newWSServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn.Read(ctx) // session.update

		time.Sleep(200 * time.Millisecond)
		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, err := New(s2s.Config{
		Extra: map[string]any{
			"api_key":  "sk-test",
			"base_url": wsURL,
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := e.Start(ctx)
	require.NoError(t, err)

	// Close the session, then try to send text — should fail.
	session.Close()
	time.Sleep(50 * time.Millisecond)

	err = session.SendText(ctx, "hello after close")
	assert.Error(t, err)
}

func TestInterruptWriteError(t *testing.T) {
	srv := newWSServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn.Read(ctx) // session.update

		time.Sleep(200 * time.Millisecond)
		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, err := New(s2s.Config{
		Extra: map[string]any{
			"api_key":  "sk-test",
			"base_url": wsURL,
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := e.Start(ctx)
	require.NoError(t, err)

	// Close, then interrupt — should fail.
	session.Close()
	time.Sleep(50 * time.Millisecond)

	err = session.Interrupt(ctx)
	assert.Error(t, err)
}

func TestStartDialError(t *testing.T) {
	e, err := New(s2s.Config{
		Extra: map[string]any{
			"api_key":  "sk-test",
			"base_url": "ws://127.0.0.1:1",
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = e.Start(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "openai realtime: websocket dial:")
}

func TestRegistry(t *testing.T) {
	t.Run("registered as openai_realtime", func(t *testing.T) {
		names := s2s.List()
		found := false
		for _, name := range names {
			if name == "openai_realtime" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected 'openai_realtime' in registered providers: %v", names)
	})

	t.Run("duplicate registration panics", func(t *testing.T) {
		assert.Panics(t, func() {
			s2s.Register("openai_realtime", func(cfg s2s.Config) (s2s.S2S, error) {
				return nil, nil
			})
		})
	})

	t.Run("factory creates engine", func(t *testing.T) {
		engine, err := s2s.New("openai_realtime", s2s.Config{
			Extra: map[string]any{"api_key": "sk-test"},
		})
		require.NoError(t, err)
		assert.NotNil(t, engine)
	})
}
