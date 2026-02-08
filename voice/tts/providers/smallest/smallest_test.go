package smallest

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
			Extra: map[string]any{"api_key": "test-key"},
		})
		require.NoError(t, err)
		assert.Equal(t, defaultVoice, e.cfg.Voice)
		assert.Equal(t, defaultModel, e.cfg.Model)
	})

	t.Run("custom voice", func(t *testing.T) {
		e, err := New(tts.Config{
			Voice: "custom",
			Extra: map[string]any{"api_key": "test-key"},
		})
		require.NoError(t, err)
		assert.Equal(t, "custom", e.cfg.Voice)
	})

	t.Run("custom base url", func(t *testing.T) {
		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": "https://custom.smallest.ai/v1",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "https://custom.smallest.ai/v1", e.baseURL)
	})
}

func TestSynthesize(t *testing.T) {
	t.Run("successful synthesis", func(t *testing.T) {
		expectedAudio := []byte("fake-audio")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/synthesize", r.URL.Path)
			assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

			body, _ := io.ReadAll(r.Body)
			var req synthesizeRequest
			json.Unmarshal(body, &req)
			assert.Equal(t, "Hello!", req.Text)
			assert.Equal(t, defaultVoice, req.Voice)
			assert.Equal(t, defaultModel, req.Model)

			w.Write(expectedAudio)
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		audio, err := e.Synthesize(context.Background(), "Hello!")
		require.NoError(t, err)
		assert.Equal(t, expectedAudio, audio)
	})

	t.Run("with speed option", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req synthesizeRequest
			json.Unmarshal(body, &req)
			assert.Equal(t, 2.0, req.Speed)
			w.Write([]byte("audio"))
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		_, err = e.Synthesize(context.Background(), "test", tts.WithSpeed(2.0))
		require.NoError(t, err)
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"forbidden"}`))
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
		assert.Contains(t, err.Error(), "403")
	})

	t.Run("context cancelled", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
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
				"api_key":  "test-key",
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

	t.Run("skip empty text", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("audio"))
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		textStream := func(yield func(string, error) bool) {
			if !yield("", nil) {
				return
			}
			yield("text", nil)
		}

		var count int
		for _, err := range e.SynthesizeStream(context.Background(), textStream) {
			require.NoError(t, err)
			count++
		}
		assert.Equal(t, 1, count)
	})
}

func TestRegistry(t *testing.T) {
	t.Run("registered as smallest", func(t *testing.T) {
		names := tts.List()
		found := false
		for _, name := range names {
			if name == "smallest" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected 'smallest' in registered providers: %v", names)
	})
}
