package groq

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
	defaultBaseURL = "https://api.groq.com/openai/v1"
	defaultModel   = "whisper-large-v3"
)

var _ stt.STT = (*Engine)(nil) // compile-time interface check

func init() {
	stt.Register("groq", func(cfg stt.Config) (stt.STT, error) {
		return New(cfg)
	})
}

// Engine implements stt.STT using the Groq Whisper API.
type Engine struct {
	apiKey  string
	baseURL string
	cfg     stt.Config
}

// New creates a new Groq STT engine.
func New(cfg stt.Config) (*Engine, error) {
	apiKey, _ := cfg.Extra["api_key"].(string)
	if apiKey == "" {
		return nil, fmt.Errorf("groq stt: api_key is required in Extra")
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

// whisperResponse is the JSON response from the Groq Whisper API.
type whisperResponse struct {
	Text string `json:"text"`
}

// Transcribe converts audio to text using the Groq Whisper API.
func (e *Engine) Transcribe(ctx context.Context, audio []byte, opts ...stt.Option) (string, error) {
	cfg := e.cfg
	for _, opt := range opts {
		opt(&cfg)
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// Write the audio file part.
	part, err := w.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", fmt.Errorf("groq stt: create form file: %w", err)
	}
	if _, err := part.Write(audio); err != nil {
		return "", fmt.Errorf("groq stt: write audio: %w", err)
	}

	// Write model field.
	if err := w.WriteField("model", cfg.Model); err != nil {
		return "", fmt.Errorf("groq stt: write model field: %w", err)
	}

	if cfg.Language != "" {
		if err := w.WriteField("language", cfg.Language); err != nil {
			return "", fmt.Errorf("groq stt: write language field: %w", err)
		}
	}

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("groq stt: close multipart: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		e.baseURL+"/audio/transcriptions", &buf)
	if err != nil {
		return "", fmt.Errorf("groq stt: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("groq stt: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("groq stt: API error (status %d): %s", resp.StatusCode, body)
	}

	var result whisperResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("groq stt: decode response: %w", err)
	}

	return result.Text, nil
}

// TranscribeStream converts streaming audio to transcript events.
// Groq Whisper does not support native streaming, so this buffers
// audio chunks and transcribes them as they arrive.
func (e *Engine) TranscribeStream(ctx context.Context, audioStream iter.Seq2[[]byte, error], opts ...stt.Option) iter.Seq2[stt.TranscriptEvent, error] {
	return func(yield func(stt.TranscriptEvent, error) bool) {
		var allAudio []byte
		for chunk, err := range audioStream {
			if err != nil {
				yield(stt.TranscriptEvent{}, err)
				return
			}
			if ctx.Err() != nil {
				yield(stt.TranscriptEvent{}, ctx.Err())
				return
			}
			allAudio = append(allAudio, chunk...)
		}

		if len(allAudio) == 0 {
			return
		}

		text, err := e.Transcribe(ctx, allAudio, opts...)
		if err != nil {
			yield(stt.TranscriptEvent{}, err)
			return
		}

		if text != "" {
			yield(stt.TranscriptEvent{
				Text:    text,
				IsFinal: true,
			}, nil)
		}
	}
}
