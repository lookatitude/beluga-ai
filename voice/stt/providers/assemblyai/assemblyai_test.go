package assemblyai

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

	t.Run("custom urls", func(t *testing.T) {
		e, err := New(stt.Config{
			Extra: map[string]any{
				"api_key":  "test-key",
				"base_url": "https://custom.assemblyai.com/v2",
				"ws_url":   "wss://custom.assemblyai.com/v2/realtime/ws",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "https://custom.assemblyai.com/v2", e.baseURL)
		assert.Equal(t, "wss://custom.assemblyai.com/v2/realtime/ws", e.wsURL)
	})
}

func TestTranscribe(t *testing.T) {
	t.Run("successful transcription", func(t *testing.T) {
		callCount := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "test-key", r.Header.Get("Authorization"))

			switch {
			case r.URL.Path == "/upload":
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(uploadResponse{
					UploadURL: "https://cdn.assemblyai.com/upload/test123",
				})

			case r.URL.Path == "/transcript" && r.Method == http.MethodPost:
				var req transcriptRequest
				json.NewDecoder(r.Body).Decode(&req)
				assert.Equal(t, "https://cdn.assemblyai.com/upload/test123", req.AudioURL)

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(transcriptResponse{
					ID:     "tx_123",
					Status: "queued",
				})

			case r.URL.Path == "/transcript/tx_123" && r.Method == http.MethodGet:
				callCount++
				status := "processing"
				text := ""
				if callCount >= 2 {
					status = "completed"
					text = "hello world"
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(transcriptResponse{
					ID:     "tx_123",
					Status: status,
					Text:   text,
				})
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

	t.Run("transcription error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/upload":
				json.NewEncoder(w).Encode(uploadResponse{UploadURL: "https://cdn.test/x"})
			case r.URL.Path == "/transcript" && r.Method == http.MethodPost:
				json.NewEncoder(w).Encode(transcriptResponse{ID: "tx_err", Status: "error", Error: "bad audio"})
			case r.URL.Path == "/transcript/tx_err":
				json.NewEncoder(w).Encode(transcriptResponse{ID: "tx_err", Status: "error", Error: "bad audio"})
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

		_, err = e.Transcribe(context.Background(), []byte("bad-audio"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bad audio")
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

	t.Run("with options", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/upload":
				json.NewEncoder(w).Encode(uploadResponse{UploadURL: "https://cdn.test/x"})
			case r.URL.Path == "/transcript" && r.Method == http.MethodPost:
				var req transcriptRequest
				json.NewDecoder(r.Body).Decode(&req)
				assert.Equal(t, "en", req.LanguageCode)
				assert.NotNil(t, req.Punctuate)
				assert.True(t, *req.Punctuate)
				assert.NotNil(t, req.SpeakerLabels)
				assert.True(t, *req.SpeakerLabels)
				json.NewEncoder(w).Encode(transcriptResponse{ID: "tx_opt", Status: "completed", Text: "test"})
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

		text, err := e.Transcribe(context.Background(), []byte("audio"),
			stt.WithLanguage("en"),
			stt.WithPunctuation(true),
			stt.WithDiarization(true),
		)
		require.NoError(t, err)
		assert.Equal(t, "test", text)
	})
}

func TestRegistry(t *testing.T) {
	t.Run("registered as assemblyai", func(t *testing.T) {
		names := stt.List()
		found := false
		for _, name := range names {
			if name == "assemblyai" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected 'assemblyai' in registered providers: %v", names)
	})
}
