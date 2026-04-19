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

	"github.com/lookatitude/beluga-ai/v2/voice/stt"
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

func TestTranscribe_SuccessfulTranscription(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-key", r.Header.Get("Authorization"))
		switch {
		case r.URL.Path == "/upload":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(uploadResponse{UploadURL: "https://cdn.assemblyai.com/upload/test123"}) //nolint:errcheck
		case r.URL.Path == "/transcript" && r.Method == http.MethodPost:
			var req transcriptRequest
			json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
			assert.Equal(t, "https://cdn.assemblyai.com/upload/test123", req.AudioURL)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(transcriptResponse{ID: "tx_123", Status: "queued"}) //nolint:errcheck
		case r.URL.Path == "/transcript/tx_123" && r.Method == http.MethodGet:
			callCount++
			status, text := "processing", ""
			if callCount >= 2 {
				status, text = "completed", "hello world"
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(transcriptResponse{ID: "tx_123", Status: status, Text: text}) //nolint:errcheck
		}
	}))
	defer srv.Close()

	e, err := New(stt.Config{Extra: map[string]any{"api_key": "test-key", "base_url": srv.URL}})
	require.NoError(t, err)

	text, err := e.Transcribe(context.Background(), []byte("fake-audio"))
	require.NoError(t, err)
	assert.Equal(t, "hello world", text)
}

func TestTranscribe_TranscriptionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/upload":
			json.NewEncoder(w).Encode(uploadResponse{UploadURL: "https://cdn.test/x"}) //nolint:errcheck
		case r.URL.Path == "/transcript" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(transcriptResponse{ID: "tx_err", Status: "error", Error: "bad audio"}) //nolint:errcheck
		case r.URL.Path == "/transcript/tx_err":
			json.NewEncoder(w).Encode(transcriptResponse{ID: "tx_err", Status: "error", Error: "bad audio"}) //nolint:errcheck
		}
	}))
	defer srv.Close()

	e, err := New(stt.Config{Extra: map[string]any{"api_key": "test-key", "base_url": srv.URL}})
	require.NoError(t, err)

	_, err = e.Transcribe(context.Background(), []byte("bad-audio"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad audio")
}

func TestTranscribe_UploadError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`)) //nolint:errcheck
	}))
	defer srv.Close()

	e, err := New(stt.Config{Extra: map[string]any{"api_key": "bad-key", "base_url": srv.URL}})
	require.NoError(t, err)

	_, err = e.Transcribe(context.Background(), []byte("audio"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestTranscribe_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	e, err := New(stt.Config{Extra: map[string]any{"api_key": "test-key", "base_url": srv.URL}})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = e.Transcribe(ctx, []byte("audio"))
	require.Error(t, err)
}

func TestTranscribe_WithOptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/upload":
			json.NewEncoder(w).Encode(uploadResponse{UploadURL: "https://cdn.test/x"}) //nolint:errcheck
		case r.URL.Path == "/transcript" && r.Method == http.MethodPost:
			var req transcriptRequest
			json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
			assert.Equal(t, "en", req.LanguageCode)
			assert.NotNil(t, req.Punctuate)
			assert.True(t, *req.Punctuate)
			assert.NotNil(t, req.SpeakerLabels)
			assert.True(t, *req.SpeakerLabels)
			json.NewEncoder(w).Encode(transcriptResponse{ID: "tx_opt", Status: "completed", Text: "test"}) //nolint:errcheck
		}
	}))
	defer srv.Close()

	e, err := New(stt.Config{Extra: map[string]any{"api_key": "test-key", "base_url": srv.URL}})
	require.NoError(t, err)

	text, err := e.Transcribe(context.Background(), []byte("audio"),
		stt.WithLanguage("en"),
		stt.WithPunctuation(true),
		stt.WithDiarization(true),
	)
	require.NoError(t, err)
	assert.Equal(t, "test", text)
}

func TestTranscribe_UploadDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json")) //nolint:errcheck
	}))
	defer srv.Close()

	e, _ := New(stt.Config{Extra: map[string]any{"api_key": "k", "base_url": srv.URL}})

	_, err := e.Transcribe(context.Background(), []byte("audio"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode upload")
}

func TestTranscribe_CreateTranscriptHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/upload":
			json.NewEncoder(w).Encode(uploadResponse{UploadURL: "https://cdn.test/x"}) //nolint:errcheck
		case r.URL.Path == "/transcript" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"server error"}`)) //nolint:errcheck
		}
	}))
	defer srv.Close()

	e, _ := New(stt.Config{Extra: map[string]any{"api_key": "k", "base_url": srv.URL}})

	_, err := e.Transcribe(context.Background(), []byte("audio"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestTranscribe_CreateTranscriptDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/upload":
			json.NewEncoder(w).Encode(uploadResponse{UploadURL: "https://cdn.test/x"}) //nolint:errcheck
		case r.URL.Path == "/transcript":
			w.Write([]byte("not json")) //nolint:errcheck
		}
	}))
	defer srv.Close()

	e, _ := New(stt.Config{Extra: map[string]any{"api_key": "k", "base_url": srv.URL}})

	_, err := e.Transcribe(context.Background(), []byte("audio"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode transcript")
}

func TestTranscribe_PollDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/upload":
			json.NewEncoder(w).Encode(uploadResponse{UploadURL: "https://cdn.test/x"}) //nolint:errcheck
		case r.URL.Path == "/transcript" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(transcriptResponse{ID: "tx_dec", Status: "queued"}) //nolint:errcheck
		case r.URL.Path == "/transcript/tx_dec":
			w.Write([]byte("not json")) //nolint:errcheck
		}
	}))
	defer srv.Close()

	e, _ := New(stt.Config{Extra: map[string]any{"api_key": "k", "base_url": srv.URL}})

	_, err := e.Transcribe(context.Background(), []byte("audio"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode poll")
}

func TestTranscribe_UploadHTTPFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		c, _, _ := hj.Hijack()
		c.Close()
	}))
	defer srv.Close()

	e, _ := New(stt.Config{Extra: map[string]any{"api_key": "k", "base_url": srv.URL}})

	_, err := e.Transcribe(context.Background(), []byte("audio"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upload failed")
}

func TestTranscribe_CreateTranscriptHTTPFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/upload":
			json.NewEncoder(w).Encode(uploadResponse{UploadURL: "https://cdn.test/x"}) //nolint:errcheck
		case r.URL.Path == "/transcript":
			hj, ok := w.(http.Hijacker)
			if !ok {
				return
			}
			c, _, _ := hj.Hijack()
			c.Close()
		}
	}))
	defer srv.Close()

	e, _ := New(stt.Config{Extra: map[string]any{"api_key": "k", "base_url": srv.URL}})

	_, err := e.Transcribe(context.Background(), []byte("audio"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create transcript failed")
}

func TestTranscribe_PollHTTPFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/upload":
			json.NewEncoder(w).Encode(uploadResponse{UploadURL: "https://cdn.test/x"}) //nolint:errcheck
		case r.URL.Path == "/transcript" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(transcriptResponse{ID: "tx_pf", Status: "queued"}) //nolint:errcheck
		case r.URL.Path == "/transcript/tx_pf":
			hj, ok := w.(http.Hijacker)
			if !ok {
				return
			}
			c, _, _ := hj.Hijack()
			c.Close()
		}
	}))
	defer srv.Close()

	e, _ := New(stt.Config{Extra: map[string]any{"api_key": "k", "base_url": srv.URL}})

	_, err := e.Transcribe(context.Background(), []byte("audio"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "poll failed")
}

func TestTranscribe_ContextCancelledDuringPoll(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/upload":
			json.NewEncoder(w).Encode(uploadResponse{UploadURL: "https://cdn.test/x"}) //nolint:errcheck
		case r.URL.Path == "/transcript" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(transcriptResponse{ID: "tx_cp", Status: "queued"}) //nolint:errcheck
		case strings.HasPrefix(r.URL.Path, "/transcript/tx_cp"):
			json.NewEncoder(w).Encode(transcriptResponse{ID: "tx_cp", Status: "processing"}) //nolint:errcheck
		}
	}))
	defer srv.Close()

	e, _ := New(stt.Config{Extra: map[string]any{"api_key": "k", "base_url": srv.URL}})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := e.Transcribe(ctx, []byte("audio"))
	require.Error(t, err)
}

func TestTranscribeStream_SuccessfulStreaming(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		ctx := r.Context()
		conn.Read(ctx) //nolint:errcheck
		wsjson.Write(ctx, conn, realtimeMessage{ //nolint:errcheck
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
		conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, err := New(stt.Config{
		Extra: map[string]any{"api_key": "test-key", "ws_url": wsURL},
	})
	require.NoError(t, err)

	audioStream := func(yield func([]byte, error) bool) { yield([]byte{0x01, 0x02}, nil) }

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
}

func TestTranscribeStream_PartialTranscript(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		ctx := r.Context()
		conn.Read(ctx) //nolint:errcheck
		wsjson.Write(ctx, conn, realtimeMessage{ //nolint:errcheck
			MessageType: "PartialTranscript",
			Text:        "hello",
			Confidence:  0.7,
		})
		conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, _ := New(stt.Config{Extra: map[string]any{"api_key": "k", "ws_url": wsURL}})

	audioStream := func(yield func([]byte, error) bool) { yield([]byte{0x01}, nil) }

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
}

func TestTranscribeStream_EmptyTextSkipped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		ctx := r.Context()
		conn.Read(ctx) //nolint:errcheck
		wsjson.Write(ctx, conn, realtimeMessage{MessageType: "PartialTranscript", Text: ""})          //nolint:errcheck
		wsjson.Write(ctx, conn, realtimeMessage{MessageType: "FinalTranscript", Text: "actual", Confidence: 0.9}) //nolint:errcheck
		conn.Close(websocket.StatusNormalClosure, "")                                                  //nolint:errcheck
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, _ := New(stt.Config{Extra: map[string]any{"api_key": "k", "ws_url": wsURL}})

	audioStream := func(yield func([]byte, error) bool) { yield([]byte{0x01}, nil) }

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
}

func TestTranscribeStream_NonJSONMessageSkipped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		ctx := r.Context()
		conn.Read(ctx)                                                                                      //nolint:errcheck
		conn.Write(ctx, websocket.MessageText, []byte("not json"))                                         //nolint:errcheck
		wsjson.Write(ctx, conn, realtimeMessage{MessageType: "FinalTranscript", Text: "valid", Confidence: 0.9}) //nolint:errcheck
		conn.Close(websocket.StatusNormalClosure, "")                                                      //nolint:errcheck
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, _ := New(stt.Config{Extra: map[string]any{"api_key": "k", "ws_url": wsURL}})

	audioStream := func(yield func([]byte, error) bool) { yield([]byte{0x01}, nil) }

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
}

func TestTranscribeStream_AudioStreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck
		for {
			if _, _, err := conn.Read(r.Context()); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, _ := New(stt.Config{Extra: map[string]any{"api_key": "k", "ws_url": wsURL}})

	audioStream := func(yield func([]byte, error) bool) { yield(nil, assert.AnError) }

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
}

func TestTranscribeStream_WebsocketDialError(t *testing.T) {
	e, _ := New(stt.Config{
		Extra: map[string]any{"api_key": "k", "ws_url": "ws://localhost:1/bad"},
	})

	audioStream := func(yield func([]byte, error) bool) {
		// no-op: test exercises connection error path
	}

	var gotErr error
	for _, err := range e.TranscribeStream(context.Background(), audioStream) {
		if err != nil {
			gotErr = err
			break
		}
	}
	require.Error(t, gotErr)
	assert.Contains(t, gotErr.Error(), "websocket dial")
}

func TestTranscribeStream_ContextCancelledDuringStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck
		for {
			if _, _, err := conn.Read(r.Context()); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, _ := New(stt.Config{Extra: map[string]any{"api_key": "k", "ws_url": wsURL}})

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
}

func TestTranscribeStream_CustomSampleRate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "sample_rate=44100")
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, _ := New(stt.Config{
		SampleRate: 44100,
		Extra:      map[string]any{"api_key": "k", "ws_url": wsURL},
	})

	audioStream := func(yield func([]byte, error) bool) {
		// no-op: test verifies server closes connection cleanly
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, err := range e.TranscribeStream(ctx, audioStream) {
		if err != nil {
			break
		}
	}
}

func TestTranscribeStream_WriteErrorOnClosedConnection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, _ := New(stt.Config{Extra: map[string]any{"api_key": "k", "ws_url": wsURL}})

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
}

func TestTranscribeStream_YieldReturnsFalse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck
		ctx := r.Context()
		for i := 0; i < 5; i++ {
			wsjson.Write(ctx, conn, realtimeMessage{ //nolint:errcheck
				MessageType: "FinalTranscript",
				Text:        "text",
				Confidence:  0.9,
			})
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	e, _ := New(stt.Config{Extra: map[string]any{"api_key": "k", "ws_url": wsURL}})

	audioStream := func(yield func([]byte, error) bool) { yield([]byte{0x01}, nil) }

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
