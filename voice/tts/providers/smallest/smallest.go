package smallest

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
	defaultBaseURL = "https://api.smallest.ai/v1"
	defaultVoice   = "emily"
	defaultModel   = "lightning"
)

var _ tts.TTS = (*Engine)(nil) // compile-time interface check

func init() {
	tts.Register("smallest", func(cfg tts.Config) (tts.TTS, error) {
		return New(cfg)
	})
}

// Engine implements tts.TTS using the Smallest.ai API.
type Engine struct {
	apiKey  string
	baseURL string
	cfg     tts.Config
}

// New creates a new Smallest.ai TTS engine.
func New(cfg tts.Config) (*Engine, error) {
	apiKey, _ := cfg.Extra["api_key"].(string)
	if apiKey == "" {
		return nil, fmt.Errorf("smallest: api_key is required in Extra")
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

// synthesizeRequest is the JSON body for the Smallest.ai TTS API.
type synthesizeRequest struct {
	Text    string  `json:"text"`
	Voice   string  `json:"voice_id"`
	Model   string  `json:"model,omitempty"`
	Speed   float64 `json:"speed,omitempty"`
	Format  string  `json:"sample_rate,omitempty"`
}

// Synthesize converts text to audio using the Smallest.ai TTS API.
func (e *Engine) Synthesize(ctx context.Context, text string, opts ...tts.Option) ([]byte, error) {
	cfg := e.cfg
	for _, opt := range opts {
		opt(&cfg)
	}

	reqBody := synthesizeRequest{
		Text:  text,
		Voice: cfg.Voice,
		Model: cfg.Model,
	}
	if cfg.Speed > 0 {
		reqBody.Speed = cfg.Speed
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("smallest: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		e.baseURL+"/synthesize", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("smallest: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("smallest: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("smallest: API error (status %d): %s", resp.StatusCode, body)
	}

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("smallest: read response: %w", err)
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
