package gemini

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
			Extra: map[string]any{"api_key": "test-key"},
		})
		require.NoError(t, err)
		assert.NotNil(t, e)
		assert.Equal(t, defaultModel, e.cfg.Model)
	})

	t.Run("custom model", func(t *testing.T) {
		e, err := New(s2s.Config{
			Model: "gemini-2.0-pro",
			Extra: map[string]any{"api_key": "test-key"},
		})
		require.NoError(t, err)
		assert.Equal(t, "gemini-2.0-pro", e.cfg.Model)
	})

	t.Run("custom base url", func(t *testing.T) {
		e, err := New(s2s.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": "wss://custom.google.com/ws",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "wss://custom.google.com/ws", e.baseURL)
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
			assert.NotNil(t, msg["setup"])

			// Send setupComplete.
			setupComplete := map[string]any{
				"setupComplete": map[string]any{},
			}
			scData, _ := json.Marshal(setupComplete)
			conn.Write(ctx, websocket.MessageText, scData)

			// Send audio content.
			audioEvent := geminiServerMsg{
				ServerContent: &geminiContent{
					ModelTurn: &geminiTurn{
						Parts: []geminiPart{
							{
								InlineData: &geminiBlob{
									MimeType: "audio/pcm",
									Data:     encodedAudio,
								},
							},
						},
					},
				},
			}
			aeData, _ := json.Marshal(audioEvent)
			conn.Write(ctx, websocket.MessageText, aeData)

			// Send turn complete.
			turnEnd := geminiServerMsg{
				ServerContent: &geminiContent{
					TurnComplete: true,
				},
			}
			teData, _ := json.Marshal(turnEnd)
			conn.Write(ctx, websocket.MessageText, teData)

			time.Sleep(100 * time.Millisecond)
			conn.Close(websocket.StatusNormalClosure, "")
		})
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, err := New(s2s.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
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
		assert.Equal(t, s2s.EventAudioOutput, events[0].Type)
		assert.Equal(t, audioData, events[0].Audio)
		assert.Equal(t, s2s.EventTurnEnd, events[len(events)-1].Type)
	})

	t.Run("receive text output", func(t *testing.T) {
		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn.Read(ctx) // setup

			textEvent := geminiServerMsg{
				ServerContent: &geminiContent{
					ModelTurn: &geminiTurn{
						Parts: []geminiPart{
							{Text: "Hello from Gemini"},
						},
					},
				},
			}
			data, _ := json.Marshal(textEvent)
			conn.Write(ctx, websocket.MessageText, data)

			time.Sleep(100 * time.Millisecond)
			conn.Close(websocket.StatusNormalClosure, "")
		})
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, err := New(s2s.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
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
			assert.Equal(t, s2s.EventTextOutput, event.Type)
			assert.Equal(t, "Hello from Gemini", event.Text)
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for text event")
		}
	})

	t.Run("receive tool call", func(t *testing.T) {
		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn.Read(ctx) // setup

			toolEvent := map[string]any{
				"toolCall": map[string]any{
					"functionCalls": []map[string]any{
						{
							"id":   "call_456",
							"name": "search",
							"args": map[string]any{"query": "weather"},
						},
					},
				},
			}
			data, _ := json.Marshal(toolEvent)
			conn.Write(ctx, websocket.MessageText, data)

			time.Sleep(100 * time.Millisecond)
			conn.Close(websocket.StatusNormalClosure, "")
		})
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, err := New(s2s.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
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
			assert.Equal(t, "call_456", event.ToolCall.ID)
			assert.Equal(t, "search", event.ToolCall.Name)
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for tool call event")
		}
	})
}

func TestSendAudio(t *testing.T) {
	srv := newWSServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn.Read(ctx) // setup

		// Read audio send.
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var msg map[string]any
		json.Unmarshal(data, &msg)
		rtInput := msg["realtimeInput"].(map[string]any)
		chunks := rtInput["mediaChunks"].([]any)
		assert.Len(t, chunks, 1)

		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, err := New(s2s.Config{
		Extra: map[string]any{
			"api_key":  "test-key",
			"base_url": wsURL,
		},
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

func TestSendText(t *testing.T) {
	srv := newWSServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn.Read(ctx) // setup

		_, data, _ := conn.Read(ctx)
		var msg map[string]any
		json.Unmarshal(data, &msg)
		assert.NotNil(t, msg["clientContent"])

		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, err := New(s2s.Config{
		Extra: map[string]any{
			"api_key":  "test-key",
			"base_url": wsURL,
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := e.Start(ctx)
	require.NoError(t, err)
	defer session.Close()

	err = session.SendText(ctx, "hello gemini")
	require.NoError(t, err)
}

func TestSendToolResult(t *testing.T) {
	srv := newWSServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn.Read(ctx) // setup

		_, data, _ := conn.Read(ctx)
		var msg map[string]any
		json.Unmarshal(data, &msg)
		toolResp := msg["toolResponse"].(map[string]any)
		funcResps := toolResp["functionResponses"].([]any)
		assert.Len(t, funcResps, 1)

		fr := funcResps[0].(map[string]any)
		assert.Equal(t, "call_789", fr["id"])

		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, err := New(s2s.Config{
		Extra: map[string]any{
			"api_key":  "test-key",
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
		CallID: "call_789",
		Content: []schema.ContentPart{
			schema.TextPart{Text: "result data"},
		},
	})
	require.NoError(t, err)
}

func TestInterrupt(t *testing.T) {
	srv := newWSServer(t, func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn.Read(ctx) // setup

		// Interrupt is a no-op for Gemini (server VAD), so just close.
		time.Sleep(100 * time.Millisecond)
		conn.Close(websocket.StatusNormalClosure, "")
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, err := New(s2s.Config{
		Extra: map[string]any{
			"api_key":  "test-key",
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

			// Read setup and verify it contains instructions and tools.
			_, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			var msg map[string]any
			json.Unmarshal(data, &msg)

			setup := msg["setup"].(map[string]any)

			// Verify system instruction.
			sysInst := setup["systemInstruction"].(map[string]any)
			parts := sysInst["parts"].([]any)
			require.Len(t, parts, 1)
			first := parts[0].(map[string]any)
			assert.Equal(t, "You are a helpful assistant", first["text"])

			// Verify tools.
			tools := setup["tools"].([]any)
			require.Len(t, tools, 1)
			toolObj := tools[0].(map[string]any)
			funcDecls := toolObj["functionDeclarations"].([]any)
			require.Len(t, funcDecls, 1)
			funcDecl := funcDecls[0].(map[string]any)
			assert.Equal(t, "get_weather", funcDecl["name"])

			time.Sleep(100 * time.Millisecond)
			conn.Close(websocket.StatusNormalClosure, "")
		})
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, err := New(s2s.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": wsURL,
			},
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
	t.Run("connection error emits error event", func(t *testing.T) {
		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn.Read(ctx) // setup

			// Close abruptly.
			conn.Close(websocket.StatusInternalError, "server crash")
		})
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, err := New(s2s.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
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
			assert.Contains(t, event.Error.Error(), "gemini live: read:")
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for connection error event")
		}
	})

	t.Run("text and audio in same model turn", func(t *testing.T) {
		audioData := []byte("pcm-audio-data")
		encodedAudio := base64.StdEncoding.EncodeToString(audioData)

		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn.Read(ctx) // setup

			// Send a model turn with both text and audio.
			event := geminiServerMsg{
				ServerContent: &geminiContent{
					ModelTurn: &geminiTurn{
						Parts: []geminiPart{
							{Text: "caption text"},
							{
								InlineData: &geminiBlob{
									MimeType: "audio/pcm",
									Data:     encodedAudio,
								},
							},
						},
					},
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
				"api_key":  "test-key",
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
				if len(events) >= 2 {
					goto done
				}
			case <-timeout:
				goto done
			}
		}
	done:
		require.Len(t, events, 2)
		assert.Equal(t, s2s.EventTextOutput, events[0].Type)
		assert.Equal(t, "caption text", events[0].Text)
		assert.Equal(t, s2s.EventAudioOutput, events[1].Type)
		assert.Equal(t, audioData, events[1].Audio)
	})

	t.Run("multiple tool calls", func(t *testing.T) {
		srv := newWSServer(t, func(conn *websocket.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn.Read(ctx) // setup

			toolEvent := map[string]any{
				"toolCall": map[string]any{
					"functionCalls": []map[string]any{
						{
							"id":   "call_1",
							"name": "search",
							"args": map[string]any{"query": "weather"},
						},
						{
							"id":   "call_2",
							"name": "calculate",
							"args": map[string]any{"expr": "2+2"},
						},
					},
				},
			}
			data, _ := json.Marshal(toolEvent)
			conn.Write(ctx, websocket.MessageText, data)

			time.Sleep(100 * time.Millisecond)
			conn.Close(websocket.StatusNormalClosure, "")
		})
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, err := New(s2s.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
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
					goto done2
				}
				events = append(events, event)
				if len(events) >= 2 {
					goto done2
				}
			case <-timeout:
				goto done2
			}
		}
	done2:
		require.Len(t, events, 2)
		assert.Equal(t, s2s.EventToolCall, events[0].Type)
		assert.Equal(t, "call_1", events[0].ToolCall.ID)
		assert.Equal(t, s2s.EventToolCall, events[1].Type)
		assert.Equal(t, "call_2", events[1].ToolCall.ID)
	})
}

func TestStartDialError(t *testing.T) {
	e, err := New(s2s.Config{
		Extra: map[string]any{
			"api_key":  "test-key",
			"base_url": "ws://127.0.0.1:1",
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = e.Start(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gemini live: websocket dial:")
}

func TestRegistry(t *testing.T) {
	t.Run("registered as gemini_live", func(t *testing.T) {
		names := s2s.List()
		found := false
		for _, name := range names {
			if name == "gemini_live" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected 'gemini_live' in registered providers: %v", names)
	})

	t.Run("duplicate registration panics", func(t *testing.T) {
		assert.Panics(t, func() {
			s2s.Register("gemini_live", func(cfg s2s.Config) (s2s.S2S, error) {
				return nil, nil
			})
		})
	})

	t.Run("factory creates engine", func(t *testing.T) {
		engine, err := s2s.New("gemini_live", s2s.Config{
			Extra: map[string]any{"api_key": "test-key"},
		})
		require.NoError(t, err)
		assert.NotNil(t, engine)
	})
}
