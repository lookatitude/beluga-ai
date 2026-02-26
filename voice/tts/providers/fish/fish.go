package fish

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
	defaultBaseURL = "https://api.fish.audio/v1"
	defaultVoice   = "default"
)

var _ tts.TTS = (*Engine)(nil) // compile-time interface check

func init() {
	tts.Register("fish", func(cfg tts.Config) (tts.TTS, error) {
		return New(cfg)
	})
}

// Engine implements tts.TTS using the Fish Audio API.
type Engine struct {
	apiKey  string
	baseURL string
	cfg     tts.Config
}

// New creates a new Fish Audio TTS engine.
func New(cfg tts.Config) (*Engine, error) {
	apiKey, _ := cfg.Extra["api_key"].(string)
	if apiKey == "" {
		return nil, fmt.Errorf("fish tts: api_key is required in Extra")
	}

	baseURL, _ := cfg.Extra["base_url"].(string)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	if cfg.Voice == "" {
		cfg.Voice = defaultVoice
	}

	return &Engine{
		apiKey:  apiKey,
		baseURL: baseURL,
		cfg:     cfg,
	}, nil
}

// synthesizeRequest is the JSON body for the Fish Audio TTS API.
type synthesizeRequest struct {
	Text        string `json:"text"`
	ReferenceID string `json:"reference_id,omitempty"`
	Format      string `json:"format,omitempty"`
	Latency     string `json:"latency,omitempty"`
}

// Synthesize converts text to audio using the Fish Audio TTS API.
func (e *Engine) Synthesize(ctx context.Context, text string, opts ...tts.Option) ([]byte, error) {
	cfg := e.cfg
	for _, opt := range opts {
		opt(&cfg)
	}

	reqBody := synthesizeRequest{
		Text:        text,
		ReferenceID: cfg.Voice,
	}
	if cfg.Format != "" {
		reqBody.Format = string(cfg.Format)
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("fish tts: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		e.baseURL+"/tts", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("fish tts: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fish tts: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fish tts: API error (status %d): %s", resp.StatusCode, body)
	}

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fish tts: read response: %w", err)
	}

	return audio, nil
}

// synthesizeChunk synthesizes a single text chunk and returns the audio or an error.
func (e *Engine) synthesizeChunk(ctx context.Context, text string, opts ...tts.Option) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return e.Synthesize(ctx, text, opts...)
}

// SynthesizeStream converts streaming text to audio chunks.
func (e *Engine) SynthesizeStream(ctx context.Context, textStream iter.Seq2[string, error], opts ...tts.Option) iter.Seq2[[]byte, error] {
	return func(yield func([]byte, error) bool) {
		for text, err := range textStream {
			if err != nil {
				yield(nil, err)
				return
			}
			if text == "" {
				continue
			}

			audio, synthErr := e.synthesizeChunk(ctx, text, opts...)
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
