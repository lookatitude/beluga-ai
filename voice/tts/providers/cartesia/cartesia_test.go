package cartesia

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
			Extra: map[string]any{"api_key": "sk-test"},
		})
		require.NoError(t, err)
		assert.NotNil(t, e)
		assert.Equal(t, defaultModel, e.cfg.Model)
	})

	t.Run("custom model", func(t *testing.T) {
		e, err := New(tts.Config{
			Model: "sonic-1",
			Extra: map[string]any{"api_key": "sk-test"},
		})
		require.NoError(t, err)
		assert.Equal(t, "sonic-1", e.cfg.Model)
	})

	t.Run("custom base url", func(t *testing.T) {
		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "sk-test",
				"base_url": "https://custom.cartesia.ai",
			},
		})
		require.NoError(t, err)
		// The base URL is set on the internal httpclient, verify engine was created.
		assert.NotNil(t, e.client)
	})
}

func TestSynthesize(t *testing.T) {
	t.Run("successful synthesis", func(t *testing.T) {
		expectedAudio := []byte("raw-pcm-audio-data")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/tts/bytes", r.URL.Path)
			assert.NotEmpty(t, r.Header.Get("X-API-Key"))
			assert.Equal(t, apiVersion, r.Header.Get("Cartesia-Version"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var req cartesiaRequest
			require.NoError(t, json.Unmarshal(body, &req))
			assert.Equal(t, "Hello, world!", req.Transcript)
			assert.Equal(t, defaultModel, req.ModelID)
			assert.Equal(t, "id", req.Voice.Mode)
			assert.Equal(t, "test-voice-id", req.Voice.ID)
			assert.Equal(t, "raw", req.OutputFormat.Container)
			assert.Equal(t, "pcm_s16le", req.OutputFormat.Encoding)
			assert.Equal(t, 24000, req.OutputFormat.SampleRate)

			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(expectedAudio)
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Voice: "test-voice-id",
			Extra: map[string]any{
				"api_key":  "sk-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		audio, err := e.Synthesize(context.Background(), "Hello, world!")
		require.NoError(t, err)
		assert.Equal(t, expectedAudio, audio)
	})

	t.Run("custom sample rate", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req cartesiaRequest
			json.Unmarshal(body, &req)
			assert.Equal(t, 44100, req.OutputFormat.SampleRate)
			w.Write([]byte("audio"))
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Voice: "voice-id",
			Extra: map[string]any{
				"api_key":  "sk-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		audio, err := e.Synthesize(context.Background(), "test", tts.WithSampleRate(44100))
		require.NoError(t, err)
		assert.NotEmpty(t, audio)
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"message": "invalid voice"})
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Voice: "invalid",
			Extra: map[string]any{
				"api_key":  "sk-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		_, err = e.Synthesize(context.Background(), "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid voice")
	})

	t.Run("context cancelled", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Voice: "voice-id",
			Extra: map[string]any{
				"api_key":  "sk-test",
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
	t.Run("stream text chunks", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("audio-data"))
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Voice: "voice-id",
			Extra: map[string]any{
				"api_key":  "sk-test",
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

		assert.Equal(t, 2, len(chunks))
	})

	t.Run("skip empty text", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("audio"))
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Voice: "voice-id",
			Extra: map[string]any{
				"api_key":  "sk-test",
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

		var chunks [][]byte
		for chunk, err := range e.SynthesizeStream(context.Background(), textStream) {
			require.NoError(t, err)
			chunks = append(chunks, chunk)
		}

		assert.Equal(t, 1, len(chunks))
	})

	t.Run("text stream error", func(t *testing.T) {
		e, err := New(tts.Config{
			Voice: "voice-id",
			Extra: map[string]any{
				"api_key":  "sk-test",
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
			Voice: "voice-id",
			Extra: map[string]any{
				"api_key":  "sk-test",
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
			w.Write([]byte(`{"message":"server error"}`))
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Voice: "voice-id",
			Extra: map[string]any{
				"api_key":  "sk-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		textStream := func(yield func(string, error) bool) {
			yield("hello", nil)
		}

		for _, err := range e.SynthesizeStream(context.Background(), textStream) {
			require.Error(t, err)
			break
		}
	})

	t.Run("consumer stops early", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("audio"))
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Voice: "voice-id",
			Extra: map[string]any{
				"api_key":  "sk-test",
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
	t.Run("registered as cartesia", func(t *testing.T) {
		names := tts.List()
		found := false
		for _, name := range names {
			if name == "cartesia" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected 'cartesia' in registered providers: %v", names)
	})
}
