package whisper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
			Extra: map[string]any{"api_key": "sk-test"},
		})
		require.NoError(t, err)
		assert.NotNil(t, e)
		assert.Equal(t, defaultModel, e.cfg.Model)
	})

	t.Run("custom model", func(t *testing.T) {
		e, err := New(stt.Config{
			Model: "whisper-large",
			Extra: map[string]any{"api_key": "sk-test"},
		})
		require.NoError(t, err)
		assert.Equal(t, "whisper-large", e.cfg.Model)
	})

	t.Run("custom base url", func(t *testing.T) {
		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "sk-test",
				"base_url": "https://custom.openai.com/v1",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "https://custom.openai.com/v1", e.baseURL)
	})
}

func TestTranscribe(t *testing.T) {
	t.Run("successful transcription", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/audio/transcriptions", r.URL.Path)
			assert.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
			assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")

			// Parse multipart form to verify fields.
			err := r.ParseMultipartForm(10 << 20)
			require.NoError(t, err)

			assert.Equal(t, "whisper-1", r.FormValue("model"))

			file, _, err := r.FormFile("file")
			require.NoError(t, err)
			defer file.Close()

			data, err := io.ReadAll(file)
			require.NoError(t, err)
			assert.Equal(t, []byte("fake-audio"), data)

			resp := whisperResponse{Text: "hello world"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "sk-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		text, err := e.Transcribe(context.Background(), []byte("fake-audio"))
		require.NoError(t, err)
		assert.Equal(t, "hello world", text)
	})

	t.Run("with language option", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.ParseMultipartForm(10 << 20)
			assert.Equal(t, "es", r.FormValue("language"))

			resp := whisperResponse{Text: "hola mundo"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "sk-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		text, err := e.Transcribe(context.Background(), []byte("audio"),
			stt.WithLanguage("es"),
		)
		require.NoError(t, err)
		assert.Equal(t, "hola mundo", text)
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
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
				"api_key":  "sk-test",
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
				"api_key":  "sk-test",
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
	t.Run("stream multiple chunks", func(t *testing.T) {
		callCount := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			resp := whisperResponse{Text: "chunk " + r.FormValue("model")}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "sk-test",
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

		assert.Equal(t, 2, len(events))
		for _, ev := range events {
			assert.True(t, ev.IsFinal)
		}
	})

	t.Run("stream error propagation", func(t *testing.T) {
		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "sk-test",
				"base_url": "http://localhost:0",
			},
		})
		require.NoError(t, err)

		audioStream := func(yield func([]byte, error) bool) {
			yield([]byte("chunk"), nil)
		}

		var gotErr bool
		for _, err := range e.TranscribeStream(context.Background(), audioStream) {
			if err != nil {
				gotErr = true
				break
			}
		}
		assert.True(t, gotErr)
	})

	t.Run("audio stream error", func(t *testing.T) {
		e, err := New(stt.Config{
			Extra: map[string]any{"api_key": "sk-test"},
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

	t.Run("context cancelled", func(t *testing.T) {
		e, err := New(stt.Config{
			Extra: map[string]any{"api_key": "sk-test"},
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

	t.Run("empty text skipped", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(whisperResponse{Text: ""})
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "sk-test",
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

	t.Run("early consumer break", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(whisperResponse{Text: "text"})
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "sk-test",
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

		count := 0
		for _, err := range e.TranscribeStream(context.Background(), audioStream) {
			require.NoError(t, err)
			count++
			break
		}
		assert.Equal(t, 1, count)
	})
}

func TestRegistry(t *testing.T) {
	t.Run("registered as whisper", func(t *testing.T) {
		names := stt.List()
		found := false
		for _, name := range names {
			if name == "whisper" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected 'whisper' in registered providers: %v", names)
	})

	t.Run("new via registry", func(t *testing.T) {
		engine, err := stt.New("whisper", stt.Config{
			Extra: map[string]any{"api_key": "sk-test"},
		})
		require.NoError(t, err)
		assert.NotNil(t, engine)
	})

	t.Run("new via registry error", func(t *testing.T) {
		_, err := stt.New("whisper", stt.Config{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "api_key is required")
	})
}
