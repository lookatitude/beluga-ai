package fish

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

	"github.com/lookatitude/beluga-ai/v2/voice/tts"
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
			Voice: "custom-ref",
			Extra: map[string]any{"api_key": "test-key"},
		})
		require.NoError(t, err)
		assert.Equal(t, "custom-ref", e.cfg.Voice)
	})

	t.Run("custom base url", func(t *testing.T) {
		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": "https://custom.fish.audio/v1",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "https://custom.fish.audio/v1", e.baseURL)
	})
}

func TestSynthesize(t *testing.T) {
	t.Run("successful synthesis", func(t *testing.T) {
		expectedAudio := []byte("fake-audio")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/tts", r.URL.Path)
			assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

			body, _ := io.ReadAll(r.Body)
			var req synthesizeRequest
			json.Unmarshal(body, &req)
			assert.Equal(t, "Hello!", req.Text)
			assert.Equal(t, defaultVoice, req.ReferenceID)

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

	t.Run("with custom voice", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req synthesizeRequest
			json.Unmarshal(body, &req)
			assert.Equal(t, "custom-ref", req.ReferenceID)
			w.Write([]byte("audio"))
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Voice: "custom-ref",
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		_, err = e.Synthesize(context.Background(), "test")
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
}

func twoWordStream(yield func(string, error) bool) {
	if !yield("Hi", nil) {
		return
	}
	yield("there", nil)
}

func emptyFirstStream(yield func(string, error) bool) {
	if !yield("", nil) {
		return
	}
	yield("text", nil)
}

func singleWordStream(yield func(string, error) bool) {
	yield("hello", nil)
}

func errorStream(yield func(string, error) bool) {
	yield("", fmt.Errorf("stream error"))
}

func twoWordStopEarlyStream(yield func(string, error) bool) {
	if !yield("first", nil) {
		return
	}
	yield("second", nil)
}

func collectChunks(t *testing.T, iter func(func([]byte, error) bool)) [][]byte {
	t.Helper()
	var chunks [][]byte
	for chunk, err := range iter {
		require.NoError(t, err)
		chunks = append(chunks, chunk)
	}
	return chunks
}

func requireFirstError(t *testing.T, iter func(func([]byte, error) bool)) error {
	t.Helper()
	for _, err := range iter {
		require.Error(t, err)
		return err
	}
	t.Fatal("expected at least one error from stream")
	return nil
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

		chunks := collectChunks(t, e.SynthesizeStream(context.Background(), twoWordStream))
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

		chunks := collectChunks(t, e.SynthesizeStream(context.Background(), emptyFirstStream))
		assert.Equal(t, 1, len(chunks))
	})

	t.Run("text stream error", func(t *testing.T) {
		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": "http://localhost:1",
			},
		})
		require.NoError(t, err)

		streamErr := requireFirstError(t, e.SynthesizeStream(context.Background(), errorStream))
		assert.Contains(t, streamErr.Error(), "stream error")
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

		requireFirstError(t, e.SynthesizeStream(ctx, singleWordStream))
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

		streamErr := requireFirstError(t, e.SynthesizeStream(context.Background(), singleWordStream))
		assert.Contains(t, streamErr.Error(), "500")
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

		var count int
		for chunk, err := range e.SynthesizeStream(context.Background(), twoWordStopEarlyStream) {
			require.NoError(t, err)
			assert.NotEmpty(t, chunk)
			count++
			break
		}
		assert.Equal(t, 1, count)
	})
}

func TestSynthesize_ConnectionError(t *testing.T) {
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
}

func TestRegistry(t *testing.T) {
	t.Run("registered as fish", func(t *testing.T) {
		names := tts.List()
		found := false
		for _, name := range names {
			if name == "fish" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected 'fish' in registered providers: %v", names)
	})

	t.Run("create via registry", func(t *testing.T) {
		e, err := tts.New("fish", tts.Config{
			Extra: map[string]any{"api_key": "registry-key"},
		})
		require.NoError(t, err)
		require.NotNil(t, e)
	})
}
