// Package elevenlabs provides the ElevenLabs Scribe STT provider for the
// Beluga AI voice pipeline. It uses the ElevenLabs Speech-to-Text API.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/elevenlabs"
//
//	engine, err := stt.New("elevenlabs", stt.Config{
//	    Extra: map[string]any{"api_key": "xi-..."},
//	})
//	text, err := engine.Transcribe(ctx, audioBytes)
package elevenlabs

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

	"github.com/lookatitude/beluga-ai/voice/stt"
)

const (
	defaultBaseURL = "https://api.elevenlabs.io/v1"
	defaultModel   = "scribe_v1"
)

var _ stt.STT = (*Engine)(nil) // compile-time interface check

func init() {
	stt.Register("elevenlabs", func(cfg stt.Config) (stt.STT, error) {
		return New(cfg)
	})
}

// Engine implements stt.STT using the ElevenLabs Scribe API.
type Engine struct {
	apiKey  string
	baseURL string
	cfg     stt.Config
}

// New creates a new ElevenLabs STT engine.
func New(cfg stt.Config) (*Engine, error) {
	apiKey, _ := cfg.Extra["api_key"].(string)
	if apiKey == "" {
		return nil, fmt.Errorf("elevenlabs stt: api_key is required in Extra")
	}

	baseURL, _ := cfg.Extra["base_url"].(string)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	if cfg.Model == "" {
		cfg.Model = defaultModel
	}

	return &Engine{
		apiKey:  apiKey,
		baseURL: baseURL,
		cfg:     cfg,
	}, nil
}

// scribeResponse is the JSON response from the ElevenLabs Scribe API.
type scribeResponse struct {
	Text  string       `json:"text"`
	Words []scribeWord `json:"words"`
}

type scribeWord struct {
	Text  string  `json:"text"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Type  string  `json:"type"`
}

// Transcribe converts a complete audio buffer to text using the ElevenLabs
// Scribe API via multipart form upload.
func (e *Engine) Transcribe(ctx context.Context, audio []byte, opts ...stt.Option) (string, error) {
	cfg := e.cfg
	for _, opt := range opts {
		opt(&cfg)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", fmt.Errorf("elevenlabs stt: create form file: %w", err)
	}
	if _, err := part.Write(audio); err != nil {
		return "", fmt.Errorf("elevenlabs stt: write audio: %w", err)
	}

	if err := writer.WriteField("model_id", cfg.Model); err != nil {
		return "", fmt.Errorf("elevenlabs stt: write model field: %w", err)
	}
	if cfg.Language != "" {
		if err := writer.WriteField("language_code", cfg.Language); err != nil {
			return "", fmt.Errorf("elevenlabs stt: write language field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("elevenlabs stt: close multipart writer: %w", err)
	}

	u := e.baseURL + "/speech-to-text"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, &body)
	if err != nil {
		return "", fmt.Errorf("elevenlabs stt: create request: %w", err)
	}
	req.Header.Set("xi-api-key", e.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("elevenlabs stt: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("elevenlabs stt: API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result scribeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("elevenlabs stt: decode response: %w", err)
	}

	return result.Text, nil
}

// TranscribeStream implements streaming transcription by transcribing each
// audio chunk independently. ElevenLabs Scribe does not support native
// streaming, so each chunk is sent as a batch request.
func (e *Engine) TranscribeStream(ctx context.Context, audioStream iter.Seq2[[]byte, error], opts ...stt.Option) iter.Seq2[stt.TranscriptEvent, error] {
	return func(yield func(stt.TranscriptEvent, error) bool) {
		for chunk, err := range audioStream {
			if err != nil {
				yield(stt.TranscriptEvent{}, err)
				return
			}
			if ctx.Err() != nil {
				yield(stt.TranscriptEvent{}, ctx.Err())
				return
			}

			text, transcribeErr := e.Transcribe(ctx, chunk, opts...)
			if transcribeErr != nil {
				yield(stt.TranscriptEvent{}, transcribeErr)
				return
			}

			if text != "" {
				if !yield(stt.TranscriptEvent{
					Text:      text,
					IsFinal:   true,
					Timestamp: time.Duration(0),
				}, nil) {
					return
				}
			}
		}
	}
}
