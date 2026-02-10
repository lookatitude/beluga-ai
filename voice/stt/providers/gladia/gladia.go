package gladia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/lookatitude/beluga-ai/voice/stt"
)

const (
	defaultBaseURL = "https://api.gladia.io/v2"
)

var _ stt.STT = (*Engine)(nil) // compile-time interface check

func init() {
	stt.Register("gladia", func(cfg stt.Config) (stt.STT, error) {
		return New(cfg)
	})
}

// Engine implements stt.STT using the Gladia API.
type Engine struct {
	apiKey  string
	baseURL string
	cfg     stt.Config
}

// New creates a new Gladia STT engine.
func New(cfg stt.Config) (*Engine, error) {
	apiKey, _ := cfg.Extra["api_key"].(string)
	if apiKey == "" {
		return nil, fmt.Errorf("gladia: api_key is required in Extra")
	}

	baseURL, _ := cfg.Extra["base_url"].(string)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &Engine{
		apiKey:  apiKey,
		baseURL: baseURL,
		cfg:     cfg,
	}, nil
}

// uploadResponse is the response from Gladia's upload endpoint.
type uploadResponse struct {
	AudioURL string `json:"audio_url"`
}

// transcriptionRequest is the request body for Gladia transcription.
type transcriptionRequest struct {
	AudioURL string `json:"audio_url"`
	Language string `json:"language,omitempty"`
}

// transcriptionResponse is the response from the Gladia transcription endpoint.
type transcriptionResponse struct {
	ID         string `json:"id"`
	ResultURL  string `json:"result_url"`
	Status     string `json:"status"`
}

// transcriptionResult holds the final transcription result.
type transcriptionResult struct {
	Status string `json:"status"`
	Result struct {
		Transcription struct {
			FullTranscript string `json:"full_transcript"`
		} `json:"transcription"`
	} `json:"result"`
}

// Transcribe converts audio to text using Gladia's REST API.
func (e *Engine) Transcribe(ctx context.Context, audio []byte, opts ...stt.Option) (string, error) {
	cfg := e.cfg
	for _, opt := range opts {
		opt(&cfg)
	}

	// Upload audio via multipart.
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("audio", "audio.wav")
	if err != nil {
		return "", fmt.Errorf("gladia: create form file: %w", err)
	}
	if _, err := part.Write(audio); err != nil {
		return "", fmt.Errorf("gladia: write audio: %w", err)
	}
	if cfg.Language != "" {
		w.WriteField("language", cfg.Language)
	}
	w.Close()

	uploadReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		e.baseURL+"/upload", &buf)
	if err != nil {
		return "", fmt.Errorf("gladia: create upload request: %w", err)
	}
	uploadReq.Header.Set("x-gladia-key", e.apiKey)
	uploadReq.Header.Set("Content-Type", w.FormDataContentType())

	uploadResp, err := http.DefaultClient.Do(uploadReq)
	if err != nil {
		return "", fmt.Errorf("gladia: upload failed: %w", err)
	}
	defer uploadResp.Body.Close()

	if uploadResp.StatusCode != http.StatusOK && uploadResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(uploadResp.Body)
		return "", fmt.Errorf("gladia: upload error (status %d): %s", uploadResp.StatusCode, body)
	}

	var upload uploadResponse
	if err := json.NewDecoder(uploadResp.Body).Decode(&upload); err != nil {
		return "", fmt.Errorf("gladia: decode upload: %w", err)
	}

	// Create transcription.
	txReq := transcriptionRequest{
		AudioURL: upload.AudioURL,
		Language: cfg.Language,
	}
	txData, _ := json.Marshal(txReq)

	createReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		e.baseURL+"/transcription", bytes.NewReader(txData))
	if err != nil {
		return "", fmt.Errorf("gladia: create transcription request: %w", err)
	}
	createReq.Header.Set("x-gladia-key", e.apiKey)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		return "", fmt.Errorf("gladia: create transcription failed: %w", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusOK && createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		return "", fmt.Errorf("gladia: transcription error (status %d): %s", createResp.StatusCode, body)
	}

	var txResp transcriptionResponse
	if err := json.NewDecoder(createResp.Body).Decode(&txResp); err != nil {
		return "", fmt.Errorf("gladia: decode transcription: %w", err)
	}

	// Poll for result.
	resultURL := txResp.ResultURL
	if resultURL == "" {
		resultURL = e.baseURL + "/transcription/" + txResp.ID
	}

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}

		pollReq, err := http.NewRequestWithContext(ctx, http.MethodGet, resultURL, nil)
		if err != nil {
			return "", fmt.Errorf("gladia: create poll request: %w", err)
		}
		pollReq.Header.Set("x-gladia-key", e.apiKey)

		pollResp, err := http.DefaultClient.Do(pollReq)
		if err != nil {
			return "", fmt.Errorf("gladia: poll failed: %w", err)
		}

		var result transcriptionResult
		if err := json.NewDecoder(pollResp.Body).Decode(&result); err != nil {
			pollResp.Body.Close()
			return "", fmt.Errorf("gladia: decode result: %w", err)
		}
		pollResp.Body.Close()

		if result.Status == "done" {
			return result.Result.Transcription.FullTranscript, nil
		}
		if result.Status == "error" {
			return "", fmt.Errorf("gladia: transcription failed")
		}
	}
}

// gladiaStreamMsg is a message from the Gladia real-time WebSocket.
type gladiaStreamMsg struct {
	Type       string  `json:"type"`
	Transcript string  `json:"transcription,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
	IsFinal    bool    `json:"is_final,omitempty"`
	Duration   float64 `json:"duration,omitempty"`
}

// TranscribeStream converts streaming audio to transcript events using Gladia's
// WebSocket API.
func (e *Engine) TranscribeStream(ctx context.Context, audioStream iter.Seq2[[]byte, error], opts ...stt.Option) iter.Seq2[stt.TranscriptEvent, error] {
	return func(yield func(stt.TranscriptEvent, error) bool) {
		cfg := e.cfg
		for _, opt := range opts {
			opt(&cfg)
		}

		// First initiate a live session via HTTP.
		initReq := map[string]any{
			"encoding":    "wav",
			"sample_rate": 16000,
		}
		if cfg.SampleRate > 0 {
			initReq["sample_rate"] = cfg.SampleRate
		}
		if cfg.Language != "" {
			initReq["language"] = cfg.Language
		}

		initData, _ := json.Marshal(initReq)
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
			e.baseURL+"/live", bytes.NewReader(initData))
		if err != nil {
			yield(stt.TranscriptEvent{}, fmt.Errorf("gladia: create live request: %w", err))
			return
		}
		httpReq.Header.Set("x-gladia-key", e.apiKey)
		httpReq.Header.Set("Content-Type", "application/json")

		httpResp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			yield(stt.TranscriptEvent{}, fmt.Errorf("gladia: live request failed: %w", err))
			return
		}
		defer httpResp.Body.Close()

		var liveResp struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(httpResp.Body).Decode(&liveResp); err != nil {
			yield(stt.TranscriptEvent{}, fmt.Errorf("gladia: decode live response: %w", err))
			return
		}

		if liveResp.URL == "" {
			yield(stt.TranscriptEvent{}, fmt.Errorf("gladia: no websocket URL returned"))
			return
		}

		conn, _, err := websocket.Dial(ctx, liveResp.URL, nil)
		if err != nil {
			yield(stt.TranscriptEvent{}, fmt.Errorf("gladia: websocket dial: %w", err))
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		events := make(chan stt.TranscriptEvent, 16)
		errs := make(chan error, 1)

		go func() {
			defer close(events)
			for {
				_, data, readErr := conn.Read(ctx)
				if readErr != nil {
					if ctx.Err() == nil {
						errs <- fmt.Errorf("gladia: websocket read: %w", readErr)
					}
					return
				}

				var msg gladiaStreamMsg
				if jsonErr := json.Unmarshal(data, &msg); jsonErr != nil {
					continue
				}

				if msg.Transcript == "" {
					continue
				}

				events <- stt.TranscriptEvent{
					Text:       msg.Transcript,
					IsFinal:    msg.IsFinal,
					Confidence: msg.Confidence,
					Timestamp:  time.Duration(msg.Duration * float64(time.Second)),
					Language:   cfg.Language,
				}
			}
		}()

		go func() {
			for chunk, streamErr := range audioStream {
				if streamErr != nil {
					errs <- streamErr
					return
				}
				if writeErr := conn.Write(ctx, websocket.MessageBinary, chunk); writeErr != nil {
					if ctx.Err() == nil {
						errs <- fmt.Errorf("gladia: websocket write: %w", writeErr)
					}
					return
				}
			}
		}()

		for {
			select {
			case event, ok := <-events:
				if !ok {
					return
				}
				if !yield(event, nil) {
					return
				}
			case err := <-errs:
				yield(stt.TranscriptEvent{}, err)
				return
			case <-ctx.Done():
				yield(stt.TranscriptEvent{}, ctx.Err())
				return
			}
		}
	}
}
