package assemblyai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
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

	t.Run("upload decode error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not json"))
		}))
		defer srv.Close()

		e, _ := New(stt.Config{
			Extra: map[string]any{"api_key": "k", "base_url": srv.URL},
		})

		_, err := e.Transcribe(context.Background(), []byte("audio"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decode upload")
	})

	t.Run("create transcript HTTP error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/upload":
				json.NewEncoder(w).Encode(uploadResponse{UploadURL: "https://cdn.test/x"})
			case r.URL.Path == "/transcript" && r.Method == http.MethodPost:
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"server error"}`))
			}
		}))
		defer srv.Close()

		e, _ := New(stt.Config{
			Extra: map[string]any{"api_key": "k", "base_url": srv.URL},
		})

		_, err := e.Transcribe(context.Background(), []byte("audio"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})

	t.Run("create transcript decode error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/upload":
				json.NewEncoder(w).Encode(uploadResponse{UploadURL: "https://cdn.test/x"})
			case r.URL.Path == "/transcript":
				w.Write([]byte("not json"))
			}
		}))
		defer srv.Close()

		e, _ := New(stt.Config{
			Extra: map[string]any{"api_key": "k", "base_url": srv.URL},
		})

		_, err := e.Transcribe(context.Background(), []byte("audio"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decode transcript")
	})

	t.Run("poll decode error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/upload":
				json.NewEncoder(w).Encode(uploadResponse{UploadURL: "https://cdn.test/x"})
			case r.URL.Path == "/transcript" && r.Method == http.MethodPost:
				json.NewEncoder(w).Encode(transcriptResponse{ID: "tx_dec", Status: "queued"})
			case r.URL.Path == "/transcript/tx_dec":
				w.Write([]byte("not json"))
			}
		}))
		defer srv.Close()

		e, _ := New(stt.Config{
			Extra: map[string]any{"api_key": "k", "base_url": srv.URL},
		})

		_, err := e.Transcribe(context.Background(), []byte("audio"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decode poll")
	})

	t.Run("upload HTTP failed", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hj, ok := w.(http.Hijacker)
			if ok {
				c, _, _ := hj.Hijack()
				c.Close()
				return
			}
		}))
		defer srv.Close()

		e, _ := New(stt.Config{
			Extra: map[string]any{"api_key": "k", "base_url": srv.URL},
		})

		_, err := e.Transcribe(context.Background(), []byte("audio"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upload failed")
	})

	t.Run("create transcript HTTP failed", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/upload":
				json.NewEncoder(w).Encode(uploadResponse{UploadURL: "https://cdn.test/x"})
			case r.URL.Path == "/transcript":
				hj, ok := w.(http.Hijacker)
				if ok {
					c, _, _ := hj.Hijack()
					c.Close()
					return
				}
			}
		}))
		defer srv.Close()

		e, _ := New(stt.Config{
			Extra: map[string]any{"api_key": "k", "base_url": srv.URL},
		})

		_, err := e.Transcribe(context.Background(), []byte("audio"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create transcript failed")
	})

	t.Run("poll HTTP failed", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/upload":
				json.NewEncoder(w).Encode(uploadResponse{UploadURL: "https://cdn.test/x"})
			case r.URL.Path == "/transcript" && r.Method == http.MethodPost:
				json.NewEncoder(w).Encode(transcriptResponse{ID: "tx_pf", Status: "queued"})
			case r.URL.Path == "/transcript/tx_pf":
				hj, ok := w.(http.Hijacker)
				if ok {
					c, _, _ := hj.Hijack()
					c.Close()
					return
				}
			}
		}))
		defer srv.Close()

		e, _ := New(stt.Config{
			Extra: map[string]any{"api_key": "k", "base_url": srv.URL},
		})

		_, err := e.Transcribe(context.Background(), []byte("audio"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "poll failed")
	})

	t.Run("context cancelled during poll", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/upload":
				json.NewEncoder(w).Encode(uploadResponse{UploadURL: "https://cdn.test/x"})
			case r.URL.Path == "/transcript" && r.Method == http.MethodPost:
				json.NewEncoder(w).Encode(transcriptResponse{ID: "tx_cp", Status: "queued"})
			case strings.HasPrefix(r.URL.Path, "/transcript/tx_cp"):
				json.NewEncoder(w).Encode(transcriptResponse{ID: "tx_cp", Status: "processing"})
			}
		}))
		defer srv.Close()

		e, _ := New(stt.Config{
			Extra: map[string]any{"api_key": "k", "base_url": srv.URL},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		_, err := e.Transcribe(ctx, []byte("audio"))
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
			ctx := r.Context()

			// Read audio data from client
			conn.Read(ctx)

			// Send transcript with words
			wsjson.Write(ctx, conn, realtimeMessage{
				MessageType: "FinalTranscript",
				Text:        "hello world",
				Confidence:  0.98,
				AudioStart:  0,
				AudioEnd:    1500,
				Words: []struct {
					Text       string  `json:"text"`
					Start      int     `json:"start"`
					End        int     `json:"end"`
					Confidence float64 `json:"confidence"`
				}{
					{Text: "hello", Start: 0, End: 500, Confidence: 0.99},
					{Text: "world", Start: 600, End: 1500, Confidence: 0.97},
				},
			})

			conn.Close(websocket.StatusNormalClosure, "")
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
			yield([]byte{0x01, 0x02}, nil)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var events []stt.TranscriptEvent
		for event, err := range e.TranscribeStream(ctx, audioStream) {
			if err != nil {
				break
			}
			events = append(events, event)
		}
		require.GreaterOrEqual(t, len(events), 1)
		assert.Equal(t, "hello world", events[0].Text)
		assert.True(t, events[0].IsFinal)
		assert.Equal(t, 0.98, events[0].Confidence)
		require.Len(t, events[0].Words, 2)
		assert.Equal(t, "hello", events[0].Words[0].Text)
	})

	t.Run("partial transcript", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			ctx := r.Context()

			conn.Read(ctx)

			wsjson.Write(ctx, conn, realtimeMessage{
				MessageType: "PartialTranscript",
				Text:        "hello",
				Confidence:  0.7,
			})

			conn.Close(websocket.StatusNormalClosure, "")
		}))
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, _ := New(stt.Config{
			Extra: map[string]any{"api_key": "k", "ws_url": wsURL},
		})

		audioStream := func(yield func([]byte, error) bool) {
			yield([]byte{0x01}, nil)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var events []stt.TranscriptEvent
		for event, err := range e.TranscribeStream(ctx, audioStream) {
			if err != nil {
				break
			}
			events = append(events, event)
		}
		require.Len(t, events, 1)
		assert.False(t, events[0].IsFinal)
	})

	t.Run("empty text skipped", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			ctx := r.Context()

			conn.Read(ctx)

			// Send empty text (should be skipped)
			wsjson.Write(ctx, conn, realtimeMessage{
				MessageType: "PartialTranscript",
				Text:        "",
			})
			// Send non-empty
			wsjson.Write(ctx, conn, realtimeMessage{
				MessageType: "FinalTranscript",
				Text:        "actual",
				Confidence:  0.9,
			})

			conn.Close(websocket.StatusNormalClosure, "")
		}))
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, _ := New(stt.Config{
			Extra: map[string]any{"api_key": "k", "ws_url": wsURL},
		})

		audioStream := func(yield func([]byte, error) bool) {
			yield([]byte{0x01}, nil)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var events []stt.TranscriptEvent
		for event, err := range e.TranscribeStream(ctx, audioStream) {
			if err != nil {
				break
			}
			events = append(events, event)
		}
		require.Len(t, events, 1)
		assert.Equal(t, "actual", events[0].Text)
	})

	t.Run("non-JSON message skipped", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			ctx := r.Context()

			conn.Read(ctx)

			// Send non-JSON (should be skipped)
			conn.Write(ctx, websocket.MessageText, []byte("not json"))
			// Send valid
			wsjson.Write(ctx, conn, realtimeMessage{
				MessageType: "FinalTranscript",
				Text:        "valid",
				Confidence:  0.9,
			})

			conn.Close(websocket.StatusNormalClosure, "")
		}))
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, _ := New(stt.Config{
			Extra: map[string]any{"api_key": "k", "ws_url": wsURL},
		})

		audioStream := func(yield func([]byte, error) bool) {
			yield([]byte{0x01}, nil)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var events []stt.TranscriptEvent
		for event, err := range e.TranscribeStream(ctx, audioStream) {
			if err != nil {
				break
			}
			events = append(events, event)
		}
		require.Len(t, events, 1)
		assert.Equal(t, "valid", events[0].Text)
	})

	t.Run("audio stream error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close(websocket.StatusNormalClosure, "")
			for {
				_, _, err := conn.Read(r.Context())
				if err != nil {
					return
				}
			}
		}))
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, _ := New(stt.Config{
			Extra: map[string]any{"api_key": "k", "ws_url": wsURL},
		})

		audioStream := func(yield func([]byte, error) bool) {
			yield(nil, assert.AnError)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var gotErr error
		for _, err := range e.TranscribeStream(ctx, audioStream) {
			if err != nil {
				gotErr = err
				break
			}
		}
		require.Error(t, gotErr)
	})

	t.Run("websocket dial error", func(t *testing.T) {
		e, _ := New(stt.Config{
			Extra: map[string]any{
				"api_key": "k",
				"ws_url":  "ws://localhost:1/bad",
			},
		})

		audioStream := func(yield func([]byte, error) bool) {}

		var gotErr error
		for _, err := range e.TranscribeStream(context.Background(), audioStream) {
			if err != nil {
				gotErr = err
				break
			}
		}
		require.Error(t, gotErr)
		assert.Contains(t, gotErr.Error(), "websocket dial")
	})

	t.Run("context cancelled during stream", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close(websocket.StatusNormalClosure, "")
			for {
				_, _, err := conn.Read(r.Context())
				if err != nil {
					return
				}
			}
		}))
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, _ := New(stt.Config{
			Extra: map[string]any{"api_key": "k", "ws_url": wsURL},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		audioStream := func(yield func([]byte, error) bool) {
			for {
				time.Sleep(50 * time.Millisecond)
				if !yield([]byte{0x01}, nil) {
					return
				}
			}
		}

		var gotErr error
		for _, err := range e.TranscribeStream(ctx, audioStream) {
			if err != nil {
				gotErr = err
				break
			}
		}
		require.Error(t, gotErr)
	})

	t.Run("with custom sample rate", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify sample_rate in query
			assert.Contains(t, r.URL.RawQuery, "sample_rate=44100")
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			conn.Close(websocket.StatusNormalClosure, "")
		}))
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, _ := New(stt.Config{
			SampleRate: 44100,
			Extra:      map[string]any{"api_key": "k", "ws_url": wsURL},
		})

		audioStream := func(yield func([]byte, error) bool) {}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		for _, err := range e.TranscribeStream(ctx, audioStream) {
			if err != nil {
				break
			}
		}
	})

	t.Run("write error on closed connection", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			// Close immediately so client write fails
			conn.Close(websocket.StatusNormalClosure, "")
		}))
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, _ := New(stt.Config{
			Extra: map[string]any{"api_key": "k", "ws_url": wsURL},
		})

		audioStream := func(yield func([]byte, error) bool) {
			time.Sleep(50 * time.Millisecond)
			yield([]byte{0x01}, nil)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var gotErr error
		for _, err := range e.TranscribeStream(ctx, audioStream) {
			if err != nil {
				gotErr = err
				break
			}
		}
		require.Error(t, gotErr)
	})

	t.Run("yield returns false", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close(websocket.StatusNormalClosure, "")
			ctx := r.Context()
			for i := 0; i < 5; i++ {
				wsjson.Write(ctx, conn, realtimeMessage{
					MessageType: "FinalTranscript",
					Text:        "text",
					Confidence:  0.9,
				})
			}
		}))
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		e, _ := New(stt.Config{
			Extra: map[string]any{"api_key": "k", "ws_url": wsURL},
		})

		audioStream := func(yield func([]byte, error) bool) {
			yield([]byte{0x01}, nil)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		count := 0
		for _, err := range e.TranscribeStream(ctx, audioStream) {
			if err != nil {
				break
			}
			count++
			if count >= 1 {
				break
			}
		}
		assert.Equal(t, 1, count)
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
