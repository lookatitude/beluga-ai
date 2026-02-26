package gladia

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/voice/stt"
)

func TestNew(t *testing.T) {
	t.Run("missing api key", func(t *testing.T) {
		_, err := New(stt.Config{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "api_key is required")
	})

	t.Run("valid config", func(t *testing.T) {
		e, err := New(stt.Config{
			Extra: map[string]any{"api_key": "test-key"},
		})
		require.NoError(t, err)
		assert.NotNil(t, e)
		assert.Equal(t, defaultBaseURL, e.baseURL)
	})

	t.Run("custom base url", func(t *testing.T) {
		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": "https://custom.gladia.io/v2",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "https://custom.gladia.io/v2", e.baseURL)
	})
}

func TestTranscribe(t *testing.T) {
	t.Run("successful transcription", func(t *testing.T) {
		pollCount := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "test-key", r.Header.Get("x-gladia-key"))

			switch {
			case r.URL.Path == "/upload":
				json.NewEncoder(w).Encode(uploadResponse{
					AudioURL: "https://storage.gladia.io/test123",
				})

			case r.URL.Path == "/transcription" && r.Method == http.MethodPost:
				var req transcriptionRequest
				json.NewDecoder(r.Body).Decode(&req)
				assert.Equal(t, "https://storage.gladia.io/test123", req.AudioURL)

				json.NewEncoder(w).Encode(transcriptionResponse{
					ID:        "tx_456",
					ResultURL: "",
					Status:    "queued",
				})

			case r.URL.Path == "/transcription/tx_456" && r.Method == http.MethodGet:
				pollCount++
				if pollCount >= 2 {
					json.NewEncoder(w).Encode(transcriptionResult{
						Status: "done",
						Result: transcriptionResultData{
							Transcription: transcriptionTranscription{FullTranscript: "hello world"},
						},
					})
				} else {
					json.NewEncoder(w).Encode(transcriptionResult{Status: "processing"})
				}
			}
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		text, err := e.Transcribe(context.Background(), []byte("fake-audio"))
		require.NoError(t, err)
		assert.Equal(t, "hello world", text)
	})

	t.Run("upload error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "bad-key",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		_, err = e.Transcribe(context.Background(), []byte("audio"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})

	t.Run("with language", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/upload":
				json.NewEncoder(w).Encode(uploadResponse{AudioURL: "https://storage.gladia.io/x"})
			case r.URL.Path == "/transcription" && r.Method == http.MethodPost:
				var req transcriptionRequest
				json.NewDecoder(r.Body).Decode(&req)
				assert.Equal(t, "fr", req.Language)
				json.NewEncoder(w).Encode(transcriptionResponse{ID: "tx_fr", ResultURL: ""})
			case r.URL.Path == "/transcription/tx_fr":
				json.NewEncoder(w).Encode(transcriptionResult{
					Status: "done",
					Result: transcriptionResultData{
							Transcription: transcriptionTranscription{FullTranscript: "bonjour"},
						},
						})
			}
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Language: "fr",
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		text, err := e.Transcribe(context.Background(), []byte("audio"))
		require.NoError(t, err)
		assert.Equal(t, "bonjour", text)
	})

	t.Run("context cancelled", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err = e.Transcribe(ctx, []byte("audio"))
		require.Error(t, err)
	})
}

func TestTranscribe_CreateError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/upload":
			json.NewEncoder(w).Encode(uploadResponse{AudioURL: "https://storage.gladia.io/x"})
		case r.URL.Path == "/transcription" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"bad request"}`))
		}
	}))
	defer srv.Close()

	e, err := New(stt.Config{
		Extra: map[string]any{
			"api_key":  "test-key",
			"base_url": srv.URL,
		},
	})
	require.NoError(t, err)

	_, err = e.Transcribe(context.Background(), []byte("audio"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestTranscribe_PollError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/upload":
			json.NewEncoder(w).Encode(uploadResponse{AudioURL: "https://storage.gladia.io/x"})
		case r.URL.Path == "/transcription" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(transcriptionResponse{ID: "tx_err"})
		case r.URL.Path == "/transcription/tx_err":
			json.NewEncoder(w).Encode(transcriptionResult{Status: "error"})
		}
	}))
	defer srv.Close()

	e, err := New(stt.Config{
		Extra: map[string]any{
			"api_key":  "test-key",
			"base_url": srv.URL,
		},
	})
	require.NoError(t, err)

	_, err = e.Transcribe(context.Background(), []byte("audio"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transcription failed")
}

func TestTranscribe_WithResultURL(t *testing.T) {
	var srvURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/upload":
			json.NewEncoder(w).Encode(uploadResponse{AudioURL: "https://storage.gladia.io/x"})
		case r.URL.Path == "/transcription" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(transcriptionResponse{
				ID:        "tx_url",
				ResultURL: srvURL + "/custom-result",
			})
		case r.URL.Path == "/custom-result":
			json.NewEncoder(w).Encode(transcriptionResult{
				Status: "done",
				Result: transcriptionResultData{
						Transcription: transcriptionTranscription{FullTranscript: "result via url"},
					},
					})
		}
	}))
	srvURL = srv.URL
	defer srv.Close()

	e, err := New(stt.Config{
		Extra: map[string]any{
			"api_key":  "test-key",
			"base_url": srv.URL,
		},
	})
	require.NoError(t, err)

	text, err := e.Transcribe(context.Background(), []byte("audio"))
	require.NoError(t, err)
	assert.Equal(t, "result via url", text)
}

func TestTranscribe_ContextCancelledDuringPoll(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/upload":
			json.NewEncoder(w).Encode(uploadResponse{AudioURL: "https://storage.gladia.io/x"})
		case r.URL.Path == "/transcription" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(transcriptionResponse{ID: "tx_cancel"})
		case r.URL.Path == "/transcription/tx_cancel":
			// Always return processing to force polling
			json.NewEncoder(w).Encode(transcriptionResult{Status: "processing"})
		}
	}))
	defer srv.Close()

	e, err := New(stt.Config{
		Extra: map[string]any{
			"api_key":  "test-key",
			"base_url": srv.URL,
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err = e.Transcribe(ctx, []byte("audio"))
	require.Error(t, err)
}

func TestTranscribe_WithOption(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/upload":
			json.NewEncoder(w).Encode(uploadResponse{AudioURL: "https://storage.gladia.io/x"})
		case r.URL.Path == "/transcription" && r.Method == http.MethodPost:
			var req transcriptionRequest
			json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "de", req.Language)
			json.NewEncoder(w).Encode(transcriptionResponse{ID: "tx_de"})
		case r.URL.Path == "/transcription/tx_de":
			json.NewEncoder(w).Encode(transcriptionResult{
				Status: "done",
				Result: transcriptionResultData{
						Transcription: transcriptionTranscription{FullTranscript: "hallo"},
					},
					})
		}
	}))
	defer srv.Close()

	e, err := New(stt.Config{
		Extra: map[string]any{
			"api_key":  "test-key",
			"base_url": srv.URL,
		},
	})
	require.NoError(t, err)

	text, err := e.Transcribe(context.Background(), []byte("audio"), stt.WithLanguage("de"))
	require.NoError(t, err)
	assert.Equal(t, "hallo", text)
}

func TestTranscribeStream(t *testing.T) {
	t.Run("successful streaming", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/live" {
				json.NewEncoder(w).Encode(map[string]string{
					"url": "ws://" + r.Host + "/ws",
				})
				return
			}
			if r.URL.Path == "/ws" {
				conn, err := websocket.Accept(w, r, nil)
				if err != nil {
					return
				}

				ctx := r.Context()

				// Read one audio chunk from client
				conn.Read(ctx)

				// Send transcript events
				wsjson.Write(ctx, conn, gladiaStreamMsg{
					Type:       "transcript",
					Transcript: "hello",
					Confidence: 0.85,
					IsFinal:    false,
					Duration:   0.5,
				})
				wsjson.Write(ctx, conn, gladiaStreamMsg{
					Type:       "transcript",
					Transcript: "hello world",
					Confidence: 0.95,
					IsFinal:    true,
					Duration:   1.5,
				})

				// Close the connection
				conn.Close(websocket.StatusNormalClosure, "")
				return
			}
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		audioStream := func(yield func([]byte, error) bool) {
			yield([]byte{0x01, 0x02}, nil)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var events []stt.TranscriptEvent
		for event, err := range e.TranscribeStream(ctx, audioStream) {
			if err != nil {
				// WebSocket close is expected at end
				break
			}
			events = append(events, event)
		}
		require.GreaterOrEqual(t, len(events), 1)
		assert.NotEmpty(t, events[0].Text)
	})

	t.Run("empty URL returned", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]string{"url": ""})
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		audioStream := func(yield func([]byte, error) bool) {}

		var gotErr error
		for _, err := range e.TranscribeStream(context.Background(), audioStream) {
			if err != nil {
				gotErr = err
				break
			}
		}
		require.Error(t, gotErr)
		assert.Contains(t, gotErr.Error(), "no websocket URL")
	})

	t.Run("live request HTTP error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not json"))
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		audioStream := func(yield func([]byte, error) bool) {}

		var gotErr error
		for _, err := range e.TranscribeStream(context.Background(), audioStream) {
			if err != nil {
				gotErr = err
				break
			}
		}
		// Either decode error or empty URL error
		require.Error(t, gotErr)
	})

	t.Run("context cancelled", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Second)
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		audioStream := func(yield func([]byte, error) bool) {}

		var gotErr error
		for _, err := range e.TranscribeStream(ctx, audioStream) {
			if err != nil {
				gotErr = err
				break
			}
		}
		require.Error(t, gotErr)
	})

	t.Run("with language and sample rate", func(t *testing.T) {
		var receivedInit map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/live" {
				json.NewDecoder(r.Body).Decode(&receivedInit)
				json.NewEncoder(w).Encode(map[string]string{
					"url": "ws://" + r.Host + "/ws",
				})
				return
			}
			if r.URL.Path == "/ws" {
				conn, err := websocket.Accept(w, r, nil)
				if err != nil {
					return
				}
				defer conn.Close(websocket.StatusNormalClosure, "")
				wsjson.Write(r.Context(), conn, gladiaStreamMsg{
					Type:       "transcript",
					Transcript: "test",
					IsFinal:    true,
					Confidence: 0.9,
				})
				return
			}
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Language:   "es",
			SampleRate: 44100,
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		audioStream := func(yield func([]byte, error) bool) {
			yield([]byte{0x01}, nil)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		for _, err := range e.TranscribeStream(ctx, audioStream) {
			if err != nil {
				break
			}
		}
		// Verify init request included language and sample rate
		assert.Equal(t, "es", receivedInit["language"])
		assert.Equal(t, float64(44100), receivedInit["sample_rate"])
	})

	t.Run("audio stream error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/live" {
				json.NewEncoder(w).Encode(map[string]string{
					"url": "ws://" + r.Host + "/ws",
				})
				return
			}
			if r.URL.Path == "/ws" {
				conn, err := websocket.Accept(w, r, nil)
				if err != nil {
					return
				}
				defer conn.Close(websocket.StatusNormalClosure, "")
				// Keep reading to allow audio send goroutine to work
				for {
					_, _, err := conn.Read(r.Context())
					if err != nil {
						return
					}
				}
			}
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		audioStream := func(yield func([]byte, error) bool) {
			yield(nil, assert.AnError)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var gotErr error
		for _, err := range e.TranscribeStream(ctx, audioStream) {
			if err != nil {
				gotErr = err
				break
			}
		}
		require.Error(t, gotErr)
	})

	t.Run("websocket message with empty transcript skipped", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/live" {
				json.NewEncoder(w).Encode(map[string]string{
					"url": "ws://" + r.Host + "/ws",
				})
				return
			}
			if r.URL.Path == "/ws" {
				conn, err := websocket.Accept(w, r, nil)
				if err != nil {
					return
				}
				defer conn.Close(websocket.StatusNormalClosure, "")
				ctx := r.Context()
				// Send empty transcript (should be skipped)
				wsjson.Write(ctx, conn, gladiaStreamMsg{
					Type:       "transcript",
					Transcript: "",
				})
				// Send non-empty transcript
				wsjson.Write(ctx, conn, gladiaStreamMsg{
					Type:       "transcript",
					Transcript: "actual",
					IsFinal:    true,
					Confidence: 0.9,
				})
				return
			}
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		audioStream := func(yield func([]byte, error) bool) {
			yield([]byte{0x01}, nil)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var events []stt.TranscriptEvent
		for event, err := range e.TranscribeStream(ctx, audioStream) {
			if err != nil {
				break
			}
			events = append(events, event)
		}
		// Only non-empty transcript should be received
		require.Len(t, events, 1)
		assert.Equal(t, "actual", events[0].Text)
	})

	t.Run("websocket dial failure", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Return a URL that won't accept WebSocket
			json.NewEncoder(w).Encode(map[string]string{
				"url": "ws://localhost:1/bad",
			})
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		audioStream := func(yield func([]byte, error) bool) {}

		var gotErr error
		for _, err := range e.TranscribeStream(context.Background(), audioStream) {
			if err != nil {
				gotErr = err
				break
			}
		}
		require.Error(t, gotErr)
		assert.Contains(t, gotErr.Error(), "websocket dial")
	})
}

func TestTranscribe_UploadDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	e, err := New(stt.Config{
		Extra: map[string]any{"api_key": "k", "base_url": srv.URL},
	})
	require.NoError(t, err)

	_, err = e.Transcribe(context.Background(), []byte("audio"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode upload")
}

func TestTranscribe_TranscriptionDecodeError(t *testing.T) {
	reqCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount++
		switch {
		case r.URL.Path == "/upload":
			json.NewEncoder(w).Encode(uploadResponse{AudioURL: "https://x"})
		case r.URL.Path == "/transcription" && r.Method == http.MethodPost:
			w.Write([]byte("not json"))
		}
	}))
	defer srv.Close()

	e, err := New(stt.Config{
		Extra: map[string]any{"api_key": "k", "base_url": srv.URL},
	})
	require.NoError(t, err)

	_, err = e.Transcribe(context.Background(), []byte("audio"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode transcription")
}

func TestTranscribe_PollDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/upload":
			json.NewEncoder(w).Encode(uploadResponse{AudioURL: "https://x"})
		case r.URL.Path == "/transcription" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(transcriptionResponse{ID: "tx_dec"})
		case r.URL.Path == "/transcription/tx_dec":
			w.Write([]byte("not json"))
		}
	}))
	defer srv.Close()

	e, err := New(stt.Config{
		Extra: map[string]any{"api_key": "k", "base_url": srv.URL},
	})
	require.NoError(t, err)

	_, err = e.Transcribe(context.Background(), []byte("audio"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode result")
}

func TestTranscribe_PollHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/upload":
			json.NewEncoder(w).Encode(uploadResponse{AudioURL: "https://x"})
		case r.URL.Path == "/transcription" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(transcriptionResponse{ID: "tx_poll"})
		}
		// No handler for /transcription/tx_poll — will 404 but still return valid JSON-parseable body
	}))
	defer srv.Close()

	e, err := New(stt.Config{
		Extra: map[string]any{"api_key": "k", "base_url": "http://localhost:1"},
	})
	require.NoError(t, err)

	// Use a server that returns valid upload/transcription but then becomes unreachable for polling
	e.baseURL = srv.URL

	// Override to point polling at an unreachable server
	var pollSrv *httptest.Server
	pollCount := 0
	pollSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/upload":
			json.NewEncoder(w).Encode(uploadResponse{AudioURL: "https://x"})
		case r.URL.Path == "/transcription" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(transcriptionResponse{ID: "tx_pfail"})
		case r.URL.Path == "/transcription/tx_pfail":
			pollCount++
			if pollCount == 1 {
				// Close connection to trigger HTTP error
				hj, ok := w.(http.Hijacker)
				if ok {
					c, _, _ := hj.Hijack()
					c.Close()
					return
				}
			}
		}
	}))
	defer pollSrv.Close()

	e2, _ := New(stt.Config{
		Extra: map[string]any{"api_key": "k", "base_url": pollSrv.URL},
	})

	_, err = e2.Transcribe(context.Background(), []byte("audio"))
	require.Error(t, err)
	// Connection closed by server, so error could be poll HTTP or decode
	assert.True(t, strings.Contains(err.Error(), "poll failed") || strings.Contains(err.Error(), "decode result"), "unexpected error: %s", err.Error())
}

func TestTranscribe_CreateTranscriptionHTTPFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/upload":
			json.NewEncoder(w).Encode(uploadResponse{AudioURL: "https://x"})
		case r.URL.Path == "/transcription":
			// Close connection to trigger HTTP error
			hj, ok := w.(http.Hijacker)
			if ok {
				c, _, _ := hj.Hijack()
				c.Close()
				return
			}
		}
	}))
	defer srv.Close()

	e, _ := New(stt.Config{
		Extra: map[string]any{"api_key": "k", "base_url": srv.URL},
	})

	_, err := e.Transcribe(context.Background(), []byte("audio"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create transcription failed")
}

func TestTranscribeStream_NonJSONWSMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/live" {
			json.NewEncoder(w).Encode(map[string]string{
				"url": "ws://" + r.Host + "/ws",
			})
			return
		}
		if r.URL.Path == "/ws" {
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			ctx := r.Context()
			// Send non-JSON message (should be skipped)
			conn.Write(ctx, websocket.MessageText, []byte("not json"))
			// Send valid message
			wsjson.Write(ctx, conn, gladiaStreamMsg{
				Type:       "transcript",
				Transcript: "after noise",
				IsFinal:    true,
				Confidence: 0.9,
			})
			conn.Close(websocket.StatusNormalClosure, "")
			return
		}
	}))
	defer srv.Close()

	e, _ := New(stt.Config{
		Extra: map[string]any{"api_key": "k", "base_url": srv.URL},
	})

	audioStream := func(yield func([]byte, error) bool) {
		yield([]byte{0x01}, nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var events []stt.TranscriptEvent
	for event, err := range e.TranscribeStream(ctx, audioStream) {
		if err != nil {
			break
		}
		events = append(events, event)
	}
	require.Len(t, events, 1)
	assert.Equal(t, "after noise", events[0].Text)
}

func TestTranscribeStream_WriteError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/live" {
			json.NewEncoder(w).Encode(map[string]string{
				"url": "ws://" + r.Host + "/ws",
			})
			return
		}
		if r.URL.Path == "/ws" {
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			// Close immediately so client write fails
			conn.Close(websocket.StatusNormalClosure, "")
			return
		}
	}))
	defer srv.Close()

	e, _ := New(stt.Config{
		Extra: map[string]any{"api_key": "k", "base_url": srv.URL},
	})

	audioStream := func(yield func([]byte, error) bool) {
		// Give server time to close
		time.Sleep(50 * time.Millisecond)
		yield([]byte{0x01}, nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var gotErr error
	for _, err := range e.TranscribeStream(ctx, audioStream) {
		if err != nil {
			gotErr = err
			break
		}
	}
	require.Error(t, gotErr)
}

func TestTranscribe_UploadHTTPFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if ok {
			c, _, _ := hj.Hijack()
			c.Close()
			return
		}
	}))
	defer srv.Close()

	e, _ := New(stt.Config{
		Extra: map[string]any{"api_key": "k", "base_url": srv.URL},
	})

	_, err := e.Transcribe(context.Background(), []byte("audio"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upload failed")
}

func TestTranscribeStream_ContextCancelledDuringStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/live" {
			json.NewEncoder(w).Encode(map[string]string{
				"url": "ws://" + r.Host + "/ws",
			})
			return
		}
		if r.URL.Path == "/ws" {
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close(websocket.StatusNormalClosure, "")
			// Keep connection alive, wait for client to cancel
			for {
				_, _, err := conn.Read(r.Context())
				if err != nil {
					return
				}
			}
		}
	}))
	defer srv.Close()

	e, _ := New(stt.Config{
		Extra: map[string]any{"api_key": "k", "base_url": srv.URL},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	audioStream := func(yield func([]byte, error) bool) {
		// Send audio slowly so context times out
		for {
			time.Sleep(50 * time.Millisecond)
			if !yield([]byte{0x01}, nil) {
				return
			}
		}
	}

	var gotErr error
	for _, err := range e.TranscribeStream(ctx, audioStream) {
		if err != nil {
			gotErr = err
			break
		}
	}
	require.Error(t, gotErr)
}

func TestTranscribeStream_YieldReturnsFalse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/live" {
			json.NewEncoder(w).Encode(map[string]string{
				"url": "ws://" + r.Host + "/ws",
			})
			return
		}
		if r.URL.Path == "/ws" {
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close(websocket.StatusNormalClosure, "")
			ctx := r.Context()
			// Send multiple messages
			for i := 0; i < 5; i++ {
				wsjson.Write(ctx, conn, gladiaStreamMsg{
					Type:       "transcript",
					Transcript: "text",
					IsFinal:    true,
					Confidence: 0.9,
				})
			}
			return
		}
	}))
	defer srv.Close()

	e, _ := New(stt.Config{
		Extra: map[string]any{"api_key": "k", "base_url": srv.URL},
	})

	audioStream := func(yield func([]byte, error) bool) {
		yield([]byte{0x01}, nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Break after first event — tests the yield returning false path
	count := 0
	for _, err := range e.TranscribeStream(ctx, audioStream) {
		if err != nil {
			break
		}
		count++
		if count >= 1 {
			break
		}
	}
	assert.Equal(t, 1, count)
}

func TestTranscribeStream_LiveHTTPFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if ok {
			c, _, _ := hj.Hijack()
			c.Close()
			return
		}
	}))
	defer srv.Close()

	e, _ := New(stt.Config{
		Extra: map[string]any{"api_key": "k", "base_url": srv.URL},
	})

	audioStream := func(yield func([]byte, error) bool) {}

	var gotErr error
	for _, err := range e.TranscribeStream(context.Background(), audioStream) {
		if err != nil {
			gotErr = err
			break
		}
	}
	require.Error(t, gotErr)
	assert.Contains(t, gotErr.Error(), "live request failed")
}

func TestRegistry(t *testing.T) {
	t.Run("registered as gladia", func(t *testing.T) {
		names := stt.List()
		found := false
		for _, name := range names {
			if name == "gladia" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected 'gladia' in registered providers: %v", names)
	})
}

// Ensure websocket and wsjson imports are used.
var _ = strings.TrimPrefix
var _ = (*websocket.Conn)(nil)
