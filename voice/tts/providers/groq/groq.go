package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"

	"github.com/lookatitude/beluga-ai/voice/tts"
)

const (
	defaultBaseURL = "https://api.groq.com/openai/v1"
	defaultVoice   = "aura-asteria-en"
	defaultModel   = "playai-tts"
)

var _ tts.TTS = (*Engine)(nil) // compile-time interface check

func init() {
	tts.Register("groq", func(cfg tts.Config) (tts.TTS, error) {
		return New(cfg)
	})
}

// Engine implements tts.TTS using the Groq TTS API.
type Engine struct {
	apiKey  string
	baseURL string
	cfg     tts.Config
}

// New creates a new Groq TTS engine.
func New(cfg tts.Config) (*Engine, error) {
	apiKey, _ := cfg.Extra["api_key"].(string)
	if apiKey == "" {
		return nil, fmt.Errorf("groq tts: api_key is required in Extra")
	}

	baseURL, _ := cfg.Extra["base_url"].(string)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	if cfg.Voice == "" {
		cfg.Voice = defaultVoice
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

// synthesizeRequest is the JSON body for the OpenAI-compatible TTS endpoint.
type synthesizeRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Speed          float64 `json:"speed,omitempty"`
}

// Synthesize converts text to audio using the Groq TTS API.
func (e *Engine) Synthesize(ctx context.Context, text string, opts ...tts.Option) ([]byte, error) {
	cfg := e.cfg
	for _, opt := range opts {
		opt(&cfg)
	}

	reqBody := synthesizeRequest{
		Model: cfg.Model,
		Input: text,
		Voice: cfg.Voice,
	}
	if cfg.Format != "" {
		reqBody.ResponseFormat = string(cfg.Format)
	}
	if cfg.Speed > 0 {
		reqBody.Speed = cfg.Speed
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("groq tts: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		e.baseURL+"/audio/speech", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("groq tts: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("groq tts: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("groq tts: API error (status %d): %s", resp.StatusCode, body)
	}

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("groq tts: read response: %w", err)
	}

	return audio, nil
}

// SynthesizeStream converts streaming text to audio chunks.
func (e *Engine) SynthesizeStream(ctx context.Context, textStream iter.Seq2[string, error], opts ...tts.Option) iter.Seq2[[]byte, error] {
	return func(yield func([]byte, error) bool) {
		for text, err := range textStream {
			if err != nil {
				yield(nil, err)
				return
			}
			if ctx.Err() != nil {
				yield(nil, ctx.Err())
				return
			}
			if text == "" {
				continue
			}

			audio, synthErr := e.Synthesize(ctx, text, opts...)
			if synthErr != nil {
				yield(nil, synthErr)
				return
			}

			if !yield(audio, nil) {
				return
			}
		}
	}
}
