// Package whisper provides the OpenAI Whisper STT provider for the Beluga AI
// voice pipeline. It uses the OpenAI Audio Transcriptions API.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/whisper"
//
//	engine, err := stt.New("whisper", stt.Config{
//	    Model: "whisper-1",
//	    Extra: map[string]any{"api_key": "sk-..."},
//	})
//	text, err := engine.Transcribe(ctx, audioBytes)
package whisper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"mime/multipart"
	"net/http"

	"github.com/lookatitude/beluga-ai/voice/stt"
)

const (
	defaultBaseURL = "https://api.openai.com/v1"
	defaultModel   = "whisper-1"
)

func init() {
	stt.Register("whisper", func(cfg stt.Config) (stt.STT, error) {
		return New(cfg)
	})
}

// Engine implements stt.STT using the OpenAI Whisper API.
type Engine struct {
	apiKey  string
	baseURL string
	cfg     stt.Config
}

// New creates a new Whisper STT engine.
func New(cfg stt.Config) (*Engine, error) {
	apiKey, _ := cfg.Extra["api_key"].(string)
	if apiKey == "" {
		return nil, fmt.Errorf("whisper: api_key is required in Extra")
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

// whisperResponse is the JSON response from the OpenAI transcriptions API.
type whisperResponse struct {
	Text string `json:"text"`
}

// Transcribe converts a complete audio buffer to text using OpenAI's
// Audio Transcriptions API via multipart form upload.
func (e *Engine) Transcribe(ctx context.Context, audio []byte, opts ...stt.Option) (string, error) {
	cfg := e.cfg
	for _, opt := range opts {
		opt(&cfg)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", fmt.Errorf("whisper: create form file: %w", err)
	}
	if _, err := part.Write(audio); err != nil {
		return "", fmt.Errorf("whisper: write audio: %w", err)
	}

	if err := writer.WriteField("model", cfg.Model); err != nil {
		return "", fmt.Errorf("whisper: write model field: %w", err)
	}
	if cfg.Language != "" {
		if err := writer.WriteField("language", cfg.Language); err != nil {
			return "", fmt.Errorf("whisper: write language field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("whisper: close multipart writer: %w", err)
	}

	u := e.baseURL + "/audio/transcriptions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, &body)
	if err != nil {
		return "", fmt.Errorf("whisper: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("whisper: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("whisper: API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result whisperResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("whisper: decode response: %w", err)
	}

	return result.Text, nil
}

// TranscribeStream implements streaming transcription by collecting audio chunks
// and performing batch transcription. Whisper does not support native streaming,
// so each audio chunk in the stream is transcribed independently.
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
					Text:    text,
					IsFinal: true,
				}, nil) {
					return
				}
			}
		}
	}
}
