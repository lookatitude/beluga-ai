// Package cartesia provides the Cartesia TTS provider for the Beluga AI voice
// pipeline. It uses the Cartesia Text-to-Speech API via direct HTTP.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/cartesia"
//
//	engine, err := tts.New("cartesia", tts.Config{
//	    Voice: "a0e99841-438c-4a64-b679-ae501e7d6091",
//	    Extra: map[string]any{"api_key": "sk-..."},
//	})
//	audio, err := engine.Synthesize(ctx, "Hello, world!")
package cartesia

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"

	"github.com/lookatitude/beluga-ai/internal/httpclient"
	"github.com/lookatitude/beluga-ai/voice/tts"
)

const (
	defaultBaseURL = "https://api.cartesia.ai"
	defaultModel   = "sonic-2"
	apiVersion     = "2024-06-10"
)

func init() {
	tts.Register("cartesia", func(cfg tts.Config) (tts.TTS, error) {
		return New(cfg)
	})
}

// Engine implements tts.TTS using the Cartesia API.
type Engine struct {
	client *httpclient.Client
	cfg    tts.Config
}

// New creates a new Cartesia TTS engine.
func New(cfg tts.Config) (*Engine, error) {
	apiKey, _ := cfg.Extra["api_key"].(string)
	if apiKey == "" {
		return nil, fmt.Errorf("cartesia: api_key is required in Extra")
	}

	baseURL, _ := cfg.Extra["base_url"].(string)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	if cfg.Model == "" {
		cfg.Model = defaultModel
	}

	client := httpclient.New(
		httpclient.WithBaseURL(baseURL),
		httpclient.WithHeader("X-API-Key", apiKey),
		httpclient.WithHeader("Cartesia-Version", apiVersion),
		httpclient.WithRetries(2),
	)

	return &Engine{
		client: client,
		cfg:    cfg,
	}, nil
}

// cartesiaRequest is the JSON body for the Cartesia TTS API.
type cartesiaRequest struct {
	ModelID      string            `json:"model_id"`
	Transcript   string            `json:"transcript"`
	Voice        cartesiaVoice     `json:"voice"`
	OutputFormat cartesiaOutFormat `json:"output_format"`
}

type cartesiaVoice struct {
	Mode string `json:"mode"`
	ID   string `json:"id"`
}

type cartesiaOutFormat struct {
	Container  string `json:"container"`
	Encoding   string `json:"encoding"`
	SampleRate int    `json:"sample_rate"`
}

// Synthesize converts text to audio using the Cartesia TTS API.
func (e *Engine) Synthesize(ctx context.Context, text string, opts ...tts.Option) ([]byte, error) {
	cfg := e.cfg
	for _, opt := range opts {
		opt(&cfg)
	}

	sampleRate := cfg.SampleRate
	if sampleRate == 0 {
		sampleRate = 24000
	}

	reqBody := cartesiaRequest{
		ModelID:    cfg.Model,
		Transcript: text,
		Voice: cartesiaVoice{
			Mode: "id",
			ID:   cfg.Voice,
		},
		OutputFormat: cartesiaOutFormat{
			Container:  "raw",
			Encoding:   "pcm_s16le",
			SampleRate: sampleRate,
		},
	}

	resp, err := e.client.Do(ctx, http.MethodPost, "/tts/bytes", reqBody, nil)
	if err != nil {
		return nil, fmt.Errorf("cartesia: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		msg := string(body)
		var errResp struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Message != "" {
			msg = errResp.Message
		}

		return nil, &httpclient.APIError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
			Message:    msg,
		}
	}

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cartesia: read response: %w", err)
	}

	return audio, nil
}

// SynthesizeStream converts a streaming text source to a stream of audio chunks.
// Each text chunk is synthesized independently via the Cartesia API.
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
