package groq

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/voice/tts"
)

func TestNew(t *testing.T) {
	t.Run("missing api key", func(t *testing.T) {
		_, err := New(tts.Config{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "api_key is required")
	})

	t.Run("valid config", func(t *testing.T) {
		e, err := New(tts.Config{
			Extra: map[string]any{"api_key": "gsk-test"},
		})
		require.NoError(t, err)
		assert.Equal(t, defaultVoice, e.cfg.Voice)
		assert.Equal(t, defaultModel, e.cfg.Model)
	})

	t.Run("custom voice and model", func(t *testing.T) {
		e, err := New(tts.Config{
			Voice: "custom-voice",
			Model: "custom-model",
			Extra: map[string]any{"api_key": "gsk-test"},
		})
		require.NoError(t, err)
		assert.Equal(t, "custom-voice", e.cfg.Voice)
		assert.Equal(t, "custom-model", e.cfg.Model)
	})

	t.Run("custom base url", func(t *testing.T) {
		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "gsk-test",
				"base_url": "https://custom.groq.com/openai/v1",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "https://custom.groq.com/openai/v1", e.baseURL)
	})
}

func TestSynthesize(t *testing.T) {
	t.Run("successful synthesis", func(t *testing.T) {
		expectedAudio := []byte("fake-audio-data")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/audio/speech", r.URL.Path)
			assert.Equal(t, "Bearer gsk-test", r.Header.Get("Authorization"))

			body, _ := io.ReadAll(r.Body)
			var req synthesizeRequest
			json.Unmarshal(body, &req)
			assert.Equal(t, "Hello!", req.Input)
			assert.Equal(t, defaultModel, req.Model)
			assert.Equal(t, defaultVoice, req.Voice)

			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write(expectedAudio)
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "gsk-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		audio, err := e.Synthesize(context.Background(), "Hello!")
		require.NoError(t, err)
		assert.Equal(t, expectedAudio, audio)
	})

	t.Run("with options", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req synthesizeRequest
			json.Unmarshal(body, &req)
			assert.Equal(t, "override-voice", req.Voice)
			assert.Equal(t, 1.5, req.Speed)
			assert.Equal(t, "wav", req.ResponseFormat)
			w.Write([]byte("audio"))
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "gsk-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		_, err = e.Synthesize(context.Background(), "test",
			tts.WithVoice("override-voice"),
			tts.WithSpeed(1.5),
			tts.WithFormat(tts.FormatWAV),
		)
		require.NoError(t, err)
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "bad-key",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		_, err = e.Synthesize(context.Background(), "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})

	t.Run("context cancelled", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "gsk-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err = e.Synthesize(ctx, "test")
		require.Error(t, err)
	})
}

func TestSynthesizeStream(t *testing.T) {
	t.Run("stream chunks", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("audio-chunk"))
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "gsk-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		textStream := func(yield func(string, error) bool) {
			if !yield("Hello", nil) {
				return
			}
			yield("World", nil)
		}

		var chunks [][]byte
		for chunk, err := range e.SynthesizeStream(context.Background(), textStream) {
			require.NoError(t, err)
			chunks = append(chunks, chunk)
		}
		assert.Len(t, chunks, 2)
	})
}

func TestRegistry(t *testing.T) {
	t.Run("registered as groq", func(t *testing.T) {
		names := tts.List()
		found := false
		for _, name := range names {
			if name == "groq" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected 'groq' in registered providers: %v", names)
	})
}
