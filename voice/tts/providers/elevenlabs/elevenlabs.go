// Package elevenlabs provides the ElevenLabs TTS provider for the Beluga AI
// voice pipeline. It uses the ElevenLabs Text-to-Speech API.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/elevenlabs"
//
//	engine, err := tts.New("elevenlabs", tts.Config{
//	    Voice: "rachel",
//	    Extra: map[string]any{"api_key": "xi-..."},
//	})
//	audio, err := engine.Synthesize(ctx, "Hello, world!")
package elevenlabs

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
	defaultBaseURL = "https://api.elevenlabs.io/v1"
	defaultVoice   = "21m00Tcm4TlvDq8ikWAM" // Rachel
	defaultModel   = "eleven_monolingual_v1"
)

func init() {
	tts.Register("elevenlabs", func(cfg tts.Config) (tts.TTS, error) {
		return New(cfg)
	})
}

// Engine implements tts.TTS using the ElevenLabs API.
type Engine struct {
	apiKey  string
	baseURL string
	cfg     tts.Config
}

// New creates a new ElevenLabs TTS engine.
func New(cfg tts.Config) (*Engine, error) {
	apiKey, _ := cfg.Extra["api_key"].(string)
	if apiKey == "" {
		return nil, fmt.Errorf("elevenlabs tts: api_key is required in Extra")
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

// synthesizeRequest is the JSON body for the ElevenLabs TTS API.
type synthesizeRequest struct {
	Text    string         `json:"text"`
	ModelID string         `json:"model_id"`
	Voice   *voiceSettings `json:"voice_settings,omitempty"`
}

type voiceSettings struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"`
}

// Synthesize converts text to audio using the ElevenLabs TTS API.
func (e *Engine) Synthesize(ctx context.Context, text string, opts ...tts.Option) ([]byte, error) {
	cfg := e.cfg
	for _, opt := range opts {
		opt(&cfg)
	}

	reqBody := synthesizeRequest{
		Text:    text,
		ModelID: cfg.Model,
		Voice: &voiceSettings{
			Stability:       0.5,
			SimilarityBoost: 0.75,
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("elevenlabs tts: marshal request: %w", err)
	}

	u := fmt.Sprintf("%s/text-to-speech/%s", e.baseURL, cfg.Voice)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("elevenlabs tts: create request: %w", err)
	}
	req.Header.Set("xi-api-key", e.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("elevenlabs tts: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("elevenlabs tts: API error (status %d): %s", resp.StatusCode, string(body))
	}

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("elevenlabs tts: read response: %w", err)
	}

	return audio, nil
}

// SynthesizeStream converts a streaming text source to a stream of audio chunks
// using the ElevenLabs streaming TTS API.
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
