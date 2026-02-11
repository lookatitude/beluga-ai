package lmnt

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
	})

	t.Run("custom voice", func(t *testing.T) {
		e, err := New(tts.Config{
			Voice: "mila",
			Extra: map[string]any{"api_key": "test-key"},
		})
		require.NoError(t, err)
		assert.Equal(t, "mila", e.cfg.Voice)
	})

	t.Run("custom base url", func(t *testing.T) {
		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": "https://custom.lmnt.com/v1",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "https://custom.lmnt.com/v1", e.baseURL)
	})
}

func TestSynthesize(t *testing.T) {
	t.Run("successful synthesis", func(t *testing.T) {
		expectedAudio := []byte("fake-audio")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/ai/speech", r.URL.Path)
			assert.Equal(t, "test-key", r.Header.Get("X-API-Key"))

			body, _ := io.ReadAll(r.Body)
			var req synthesizeRequest
			json.Unmarshal(body, &req)
			assert.Equal(t, "Hello!", req.Text)
			assert.Equal(t, defaultVoice, req.Voice)

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
			assert.Equal(t, 1.5, req.Speed)
			assert.Equal(t, "override-voice", req.Voice)
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

		_, err = e.Synthesize(context.Background(), "test",
			tts.WithVoice("override-voice"),
			tts.WithSpeed(1.5))
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

	t.Run("with format option", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req synthesizeRequest
			json.Unmarshal(body, &req)
			assert.Equal(t, "wav", req.Format)
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

		_, err = e.Synthesize(context.Background(), "test", tts.WithFormat(tts.FormatWAV))
		require.NoError(t, err)
	})

	t.Run("connection error", func(t *testing.T) {
		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": "http://localhost:1",
			},
		})
		require.NoError(t, err)

		_, err = e.Synthesize(context.Background(), "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "request failed")
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

	t.Run("text stream error", func(t *testing.T) {
		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": "http://localhost:1",
			},
		})
		require.NoError(t, err)

		textStream := func(yield func(string, error) bool) {
			yield("", fmt.Errorf("stream error"))
		}

		for _, err := range e.SynthesizeStream(context.Background(), textStream) {
			require.Error(t, err)
			assert.Contains(t, err.Error(), "stream error")
			break
		}
	})

	t.Run("context cancelled", func(t *testing.T) {
		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": "http://localhost:1",
			},
		})
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		textStream := func(yield func(string, error) bool) {
			yield("hello", nil)
		}

		for _, err := range e.SynthesizeStream(ctx, textStream) {
			require.Error(t, err)
			break
		}
	})

	t.Run("synthesis error propagated", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server error"))
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
			yield("hello", nil)
		}

		for _, err := range e.SynthesizeStream(context.Background(), textStream) {
			require.Error(t, err)
			assert.Contains(t, err.Error(), "500")
			break
		}
	})

	t.Run("consumer stops early", func(t *testing.T) {
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
			if !yield("first", nil) {
				return
			}
			yield("second", nil)
		}

		var count int
		for chunk, err := range e.SynthesizeStream(context.Background(), textStream) {
			require.NoError(t, err)
			assert.NotEmpty(t, chunk)
			count++
			break
		}
		assert.Equal(t, 1, count)
	})
}

func TestRegistry(t *testing.T) {
	t.Run("registered as lmnt", func(t *testing.T) {
		names := tts.List()
		found := false
		for _, name := range names {
			if name == "lmnt" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected 'lmnt' in registered providers: %v", names)
	})

	t.Run("create via registry", func(t *testing.T) {
		e, err := tts.New("lmnt", tts.Config{
			Extra: map[string]any{"api_key": "registry-key"},
		})
		require.NoError(t, err)
		require.NotNil(t, e)
	})
}
