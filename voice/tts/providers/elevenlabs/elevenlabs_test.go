package elevenlabs

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
			Extra: map[string]any{"api_key": "xi-test"},
		})
		require.NoError(t, err)
		assert.NotNil(t, e)
		assert.Equal(t, defaultVoice, e.cfg.Voice)
		assert.Equal(t, defaultModel, e.cfg.Model)
	})

	t.Run("custom voice", func(t *testing.T) {
		e, err := New(tts.Config{
			Voice: "custom-voice-id",
			Extra: map[string]any{"api_key": "xi-test"},
		})
		require.NoError(t, err)
		assert.Equal(t, "custom-voice-id", e.cfg.Voice)
	})

	t.Run("custom base url", func(t *testing.T) {
		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "xi-test",
				"base_url": "https://custom.elevenlabs.io/v1",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "https://custom.elevenlabs.io/v1", e.baseURL)
	})
}

func TestSynthesize(t *testing.T) {
	t.Run("successful synthesis", func(t *testing.T) {
		expectedAudio := []byte("fake-mp3-audio-data")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Contains(t, r.URL.Path, "/text-to-speech/")
			assert.Equal(t, "xi-test", r.Header.Get("xi-api-key"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var req synthesizeRequest
			require.NoError(t, json.Unmarshal(body, &req))
			assert.Equal(t, "Hello, world!", req.Text)
			assert.Equal(t, defaultModel, req.ModelID)

			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write(expectedAudio)
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "xi-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		audio, err := e.Synthesize(context.Background(), "Hello, world!")
		require.NoError(t, err)
		assert.Equal(t, expectedAudio, audio)
	})

	t.Run("with custom voice", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "/text-to-speech/custom-id")
			w.Write([]byte("audio"))
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Voice: "custom-id",
			Extra: map[string]any{
				"api_key":  "xi-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		audio, err := e.Synthesize(context.Background(), "test")
		require.NoError(t, err)
		assert.NotEmpty(t, audio)
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"detail":"unauthorized"}`))
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
				"api_key":  "xi-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err = e.Synthesize(ctx, "test")
		require.Error(t, err)
	})

	t.Run("with voice option override", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "/text-to-speech/override-voice")
			w.Write([]byte("audio"))
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Voice: "original",
			Extra: map[string]any{
				"api_key":  "xi-test",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		audio, err := e.Synthesize(context.Background(), "test", tts.WithVoice("override-voice"))
		require.NoError(t, err)
		assert.NotEmpty(t, audio)
	})
}

func TestSynthesizeStream(t *testing.T) {
	t.Run("stream multiple text chunks", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write([]byte("audio-chunk"))
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "xi-test",
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
		for _, c := range chunks {
			assert.Equal(t, []byte("audio-chunk"), c)
		}
	})

	t.Run("skip empty text", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("audio"))
		}))
		defer srv.Close()

		e, err := New(tts.Config{
			Extra: map[string]any{
				"api_key":  "xi-test",
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
			Extra: map[string]any{
				"api_key":  "xi-test",
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
				"api_key":  "xi-test",
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
				"api_key":  "xi-test",
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
				"api_key":  "xi-test",
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
	t.Run("registered as elevenlabs", func(t *testing.T) {
		names := tts.List()
		found := false
		for _, name := range names {
			if name == "elevenlabs" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected 'elevenlabs' in registered providers: %v", names)
	})
}
