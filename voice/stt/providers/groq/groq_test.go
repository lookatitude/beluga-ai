package groq

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
			Extra: map[string]any{"api_key": "gsk-test"},
		})
		require.NoError(t, err)
		assert.NotNil(t, e)
		assert.Equal(t, defaultModel, e.cfg.Model)
	})

	t.Run("custom model", func(t *testing.T) {
		e, err := New(stt.Config{
			Model: "whisper-large-v3-turbo",
			Extra: map[string]any{"api_key": "gsk-test"},
		})
		require.NoError(t, err)
		assert.Equal(t, "whisper-large-v3-turbo", e.cfg.Model)
	})

	t.Run("custom base url", func(t *testing.T) {
		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "gsk-test",
				"base_url": "https://custom.groq.com/openai/v1",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "https://custom.groq.com/openai/v1", e.baseURL)
	})
}

func TestTranscribe(t *testing.T) {
	t.Run("successful transcription", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/audio/transcriptions", r.URL.Path)
			assert.Equal(t, "Bearer gsk-test", r.Header.Get("Authorization"))
			assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")

			err := r.ParseMultipartForm(10 << 20)
			require.NoError(t, err)
			assert.Equal(t, defaultModel, r.FormValue("model"))

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(whisperResponse{Text: "hello world"})
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "gsk-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		text, err := e.Transcribe(context.Background(), []byte("fake-audio"))
		require.NoError(t, err)
		assert.Equal(t, "hello world", text)
	})

	t.Run("with language", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.ParseMultipartForm(10 << 20)
			assert.Equal(t, "es", r.FormValue("language"))

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(whisperResponse{Text: "hola mundo"})
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "gsk-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		text, err := e.Transcribe(context.Background(), []byte("audio"),
			stt.WithLanguage("es"))
		require.NoError(t, err)
		assert.Equal(t, "hola mundo", text)
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid api key"}`))
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

	t.Run("invalid json response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{invalid json`))
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "gsk-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		_, err = e.Transcribe(context.Background(), []byte("audio"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decode response")
	})

	t.Run("context cancelled", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "gsk-test",
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

func TestTranscribeStream(t *testing.T) {
	t.Run("buffers and transcribes", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(whisperResponse{Text: "streamed text"})
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "gsk-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		audioStream := func(yield func([]byte, error) bool) {
			if !yield([]byte("chunk1"), nil) {
				return
			}
			yield([]byte("chunk2"), nil)
		}

		var events []stt.TranscriptEvent
		for event, err := range e.TranscribeStream(context.Background(), audioStream) {
			require.NoError(t, err)
			events = append(events, event)
		}

		require.Len(t, events, 1)
		assert.Equal(t, "streamed text", events[0].Text)
		assert.True(t, events[0].IsFinal)
	})

	t.Run("empty stream", func(t *testing.T) {
		e, err := New(stt.Config{
			Extra: map[string]any{"api_key": "gsk-test"},
		})
		require.NoError(t, err)

		audioStream := func(yield func([]byte, error) bool) {}

		var count int
		for range e.TranscribeStream(context.Background(), audioStream) {
			count++
		}
		assert.Equal(t, 0, count)
	})

	t.Run("audio stream error", func(t *testing.T) {
		e, err := New(stt.Config{
			Extra: map[string]any{"api_key": "gsk-test"},
		})
		require.NoError(t, err)

		audioStream := func(yield func([]byte, error) bool) {
			yield(nil, fmt.Errorf("audio read error"))
		}

		var gotErr error
		for _, err := range e.TranscribeStream(context.Background(), audioStream) {
			if err != nil {
				gotErr = err
				break
			}
		}
		require.Error(t, gotErr)
		assert.Contains(t, gotErr.Error(), "audio read error")
	})

	t.Run("context cancelled during stream", func(t *testing.T) {
		e, err := New(stt.Config{
			Extra: map[string]any{"api_key": "gsk-test"},
		})
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		audioStream := func(yield func([]byte, error) bool) {
			yield([]byte("chunk"), nil)
		}

		var gotErr error
		for _, err := range e.TranscribeStream(ctx, audioStream) {
			if err != nil {
				gotErr = err
				break
			}
		}
		require.Error(t, gotErr)
		assert.ErrorIs(t, gotErr, context.Canceled)
	})

	t.Run("transcribe error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server error"))
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "gsk-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		audioStream := func(yield func([]byte, error) bool) {
			yield([]byte("chunk"), nil)
		}

		var gotErr error
		for _, err := range e.TranscribeStream(context.Background(), audioStream) {
			if err != nil {
				gotErr = err
				break
			}
		}
		require.Error(t, gotErr)
	})

	t.Run("empty text result", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(whisperResponse{Text: ""})
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "gsk-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		audioStream := func(yield func([]byte, error) bool) {
			yield([]byte("chunk"), nil)
		}

		var count int
		for range e.TranscribeStream(context.Background(), audioStream) {
			count++
		}
		assert.Equal(t, 0, count)
	})
}

func TestRegistry(t *testing.T) {
	t.Run("registered as groq", func(t *testing.T) {
		names := stt.List()
		found := false
		for _, name := range names {
			if name == "groq" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected 'groq' in registered providers: %v", names)
	})

	t.Run("new via registry", func(t *testing.T) {
		engine, err := stt.New("groq", stt.Config{
			Extra: map[string]any{"api_key": "gsk-test"},
		})
		require.NoError(t, err)
		assert.NotNil(t, engine)
	})

	t.Run("new via registry error", func(t *testing.T) {
		_, err := stt.New("groq", stt.Config{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "api_key is required")
	})
}
