package gladia

import (
	"context"
	"encoding/json"
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
						Result: struct {
							Transcription struct {
								FullTranscript string `json:"full_transcript"`
							} `json:"transcription"`
						}{
							Transcription: struct {
								FullTranscript string `json:"full_transcript"`
							}{
								FullTranscript: "hello world",
							},
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
					Result: struct {
						Transcription struct {
							FullTranscript string `json:"full_transcript"`
						} `json:"transcription"`
					}{
						Transcription: struct {
							FullTranscript string `json:"full_transcript"`
						}{
							FullTranscript: "bonjour",
						},
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
