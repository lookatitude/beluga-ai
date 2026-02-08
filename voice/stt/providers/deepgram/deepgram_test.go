package deepgram

import (
	"context"
	"encoding/json"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
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
			Model: "nova-2",
			Extra: map[string]any{"api_key": "test-key"},
		})
		require.NoError(t, err)
		assert.NotNil(t, e)
		assert.Equal(t, "nova-2", e.cfg.Model)
	})

	t.Run("default model", func(t *testing.T) {
		e, err := New(stt.Config{
			Extra: map[string]any{"api_key": "test-key"},
		})
		require.NoError(t, err)
		assert.Equal(t, defaultModel, e.cfg.Model)
	})

	t.Run("custom base url", func(t *testing.T) {
		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": "https://custom.deepgram.com/v1",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "https://custom.deepgram.com/v1", e.baseURL)
	})
}

func TestTranscribe(t *testing.T) {
	t.Run("successful transcription", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Contains(t, r.URL.Path, "/listen")
			assert.Equal(t, "Token test-key", r.Header.Get("Authorization"))
			assert.Equal(t, "audio/wav", r.Header.Get("Content-Type"))

			// Check query params.
			assert.Equal(t, "nova-2", r.URL.Query().Get("model"))
			assert.Equal(t, "en", r.URL.Query().Get("language"))

			resp := deepgramResponse{}
			resp.Results.Channels = []struct {
				Alternatives []struct {
					Transcript string         `json:"transcript"`
					Confidence float64        `json:"confidence"`
					Words      []deepgramWord `json:"words"`
				} `json:"alternatives"`
			}{
				{
					Alternatives: []struct {
						Transcript string         `json:"transcript"`
						Confidence float64        `json:"confidence"`
						Words      []deepgramWord `json:"words"`
					}{
						{
							Transcript: "hello world",
							Confidence: 0.98,
							Words: []deepgramWord{
								{Word: "hello", Start: 0.1, End: 0.5, Confidence: 0.99},
								{Word: "world", Start: 0.6, End: 1.0, Confidence: 0.97},
							},
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Language: "en",
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

	t.Run("empty response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := deepgramResponse{}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
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
		assert.Equal(t, "", text)
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
	})

	t.Run("with options", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "true", r.URL.Query().Get("punctuate"))
			assert.Equal(t, "true", r.URL.Query().Get("diarize"))
			assert.Equal(t, "linear16", r.URL.Query().Get("encoding"))
			assert.Equal(t, "16000", r.URL.Query().Get("sample_rate"))

			resp := deepgramResponse{}
			resp.Results.Channels = []struct {
				Alternatives []struct {
					Transcript string         `json:"transcript"`
					Confidence float64        `json:"confidence"`
					Words      []deepgramWord `json:"words"`
				} `json:"alternatives"`
			}{
				{
					Alternatives: []struct {
						Transcript string         `json:"transcript"`
						Confidence float64        `json:"confidence"`
						Words      []deepgramWord `json:"words"`
					}{
						{Transcript: "test"},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": srv.URL,
			},
		})
		require.NoError(t, err)

		text, err := e.Transcribe(context.Background(), []byte("audio"),
			stt.WithPunctuation(true),
			stt.WithDiarization(true),
			stt.WithEncoding("linear16"),
			stt.WithSampleRate(16000),
		)
		require.NoError(t, err)
		assert.Equal(t, "test", text)
	})

	t.Run("context cancelled", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
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

func TestTranscribeStream(t *testing.T) {
	t.Run("successful streaming", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Read first audio chunk.
			_, _, err = conn.Read(ctx)
			if err != nil {
				conn.Close(websocket.StatusNormalClosure, "")
				return
			}

			// Send a partial result.
			partial := deepgramStreamResponse{
				Type:    "Results",
				IsFinal: false,
				Start:   0.0,
			}
			partial.Channel.Alternatives = []struct {
				Transcript string         `json:"transcript"`
				Confidence float64        `json:"confidence"`
				Words      []deepgramWord `json:"words"`
			}{
				{Transcript: "hel", Confidence: 0.9},
			}
			data, _ := json.Marshal(partial)
			conn.Write(ctx, websocket.MessageText, data)

			// Send a final result.
			final := deepgramStreamResponse{
				Type:    "Results",
				IsFinal: true,
				Start:   0.0,
			}
			final.Channel.Alternatives = []struct {
				Transcript string         `json:"transcript"`
				Confidence float64        `json:"confidence"`
				Words      []deepgramWord `json:"words"`
			}{
				{
					Transcript: "hello world",
					Confidence: 0.98,
					Words: []deepgramWord{
						{Word: "hello", Start: 0.1, End: 0.5, Confidence: 0.99},
					},
				},
			}
			data, _ = json.Marshal(final)
			conn.Write(ctx, websocket.MessageText, data)

			// Read close message from client before closing.
			conn.Read(ctx)
			conn.Close(websocket.StatusNormalClosure, "done")
		}))
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key": "test-key",
				"ws_url":  wsURL,
			},
		})
		require.NoError(t, err)

		audioStream := func(yield func([]byte, error) bool) {
			yield([]byte("chunk1"), nil)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var events []stt.TranscriptEvent
		for event, err := range e.TranscribeStream(ctx, audioStream) {
			if err != nil {
				// WebSocket close is expected after all events received.
				break
			}
			events = append(events, event)
		}

		require.GreaterOrEqual(t, len(events), 1)

		// Check at least the final event is present.
		var hasFinal bool
		for _, ev := range events {
			if ev.IsFinal {
				hasFinal = true
				assert.Equal(t, "hello world", ev.Text)
			}
		}
		assert.True(t, hasFinal, "expected a final transcript event")
	})
}

func TestRegistry(t *testing.T) {
	t.Run("registered as deepgram", func(t *testing.T) {
		names := stt.List()
		found := false
		for _, name := range names {
			if name == "deepgram" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected 'deepgram' in registered providers: %v", names)
	})
}

func TestBuildQueryParams(t *testing.T) {
	e := &Engine{cfg: stt.Config{Model: "nova-2"}}

	t.Run("basic params", func(t *testing.T) {
		params := e.buildQueryParams(stt.Config{
			Model:    "nova-2",
			Language: "en",
		})
		assert.Equal(t, "nova-2", params.Get("model"))
		assert.Equal(t, "en", params.Get("language"))
	})

	t.Run("with options", func(t *testing.T) {
		params := e.buildQueryParams(stt.Config{
			Model:       "nova-2",
			Punctuation: true,
			Diarization: true,
			Encoding:    "opus",
			SampleRate:  48000,
		})
		assert.Equal(t, "true", params.Get("punctuate"))
		assert.Equal(t, "true", params.Get("diarize"))
		assert.Equal(t, "opus", params.Get("encoding"))
		assert.Equal(t, "48000", params.Get("sample_rate"))
	})
}

// audioStreamFromChunks creates an iter.Seq2 from byte slices.
func audioStreamFromChunks(chunks ...[]byte) iter.Seq2[[]byte, error] {
	return func(yield func([]byte, error) bool) {
		for _, chunk := range chunks {
			if !yield(chunk, nil) {
				return
			}
		}
	}
}
