// Package playht provides the PlayHT TTS provider for the Beluga AI voice
// pipeline. It uses the PlayHT Text-to-Speech API.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/playht"
//
//	engine, err := tts.New("playht", tts.Config{
//	    Voice: "s3://voice-cloning-zero-shot/...",
//	    Extra: map[string]any{"api_key": "...", "user_id": "..."},
//	})
//	audio, err := engine.Synthesize(ctx, "Hello!")
package playht

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
	defaultBaseURL = "https://api.play.ht/api/v2"
	defaultVoice   = "s3://voice-cloning-zero-shot/775ae416-49bb-4fb6-bd45-740f205d3571/jennifersaad/manifest.json"
)

func init() {
	tts.Register("playht", func(cfg tts.Config) (tts.TTS, error) {
		return New(cfg)
	})
}

// Engine implements tts.TTS using the PlayHT API.
type Engine struct {
	apiKey  string
	userID  string
	baseURL string
	cfg     tts.Config
}

// New creates a new PlayHT TTS engine.
func New(cfg tts.Config) (*Engine, error) {
	apiKey, _ := cfg.Extra["api_key"].(string)
	if apiKey == "" {
		return nil, fmt.Errorf("playht: api_key is required in Extra")
	}

	userID, _ := cfg.Extra["user_id"].(string)
	if userID == "" {
		return nil, fmt.Errorf("playht: user_id is required in Extra")
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
		userID:  userID,
		baseURL: baseURL,
		cfg:     cfg,
	}, nil
}

// synthesizeRequest is the JSON body for the PlayHT TTS API.
type synthesizeRequest struct {
	Text         string  `json:"text"`
	Voice        string  `json:"voice"`
	OutputFormat string  `json:"output_format,omitempty"`
	Speed        float64 `json:"speed,omitempty"`
	Quality      string  `json:"quality,omitempty"`
}

// Synthesize converts text to audio using the PlayHT TTS API.
func (e *Engine) Synthesize(ctx context.Context, text string, opts ...tts.Option) ([]byte, error) {
	cfg := e.cfg
	for _, opt := range opts {
		opt(&cfg)
	}

	format := "mp3"
	if cfg.Format != "" {
		format = string(cfg.Format)
	}

	reqBody := synthesizeRequest{
		Text:         text,
		Voice:        cfg.Voice,
		OutputFormat: format,
	}
	if cfg.Speed > 0 {
		reqBody.Speed = cfg.Speed
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("playht: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		e.baseURL+"/tts", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("playht: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("X-USER-ID", e.userID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("playht: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("playht: API error (status %d): %s", resp.StatusCode, body)
	}

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("playht: read response: %w", err)
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
