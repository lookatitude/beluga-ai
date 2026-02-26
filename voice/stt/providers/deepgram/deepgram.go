package deepgram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
	"github.com/lookatitude/beluga-ai/voice/stt"
)

const (
	defaultBaseURL = "https://api.deepgram.com/v1"
	defaultWSURL   = "wss://api.deepgram.com/v1"
	defaultModel   = "nova-2"
)

var _ stt.STT = (*Engine)(nil) // compile-time interface check

func init() {
	stt.Register("deepgram", func(cfg stt.Config) (stt.STT, error) {
		return New(cfg)
	})
}

// Engine implements stt.STT using the Deepgram API.
type Engine struct {
	client  *httpclient.Client
	apiKey  string
	baseURL string
	wsURL   string
	cfg     stt.Config
}

// New creates a new Deepgram STT engine.
func New(cfg stt.Config) (*Engine, error) {
	apiKey, _ := cfg.Extra["api_key"].(string)
	if apiKey == "" {
		return nil, fmt.Errorf("deepgram: api_key is required in Extra")
	}

	baseURL, _ := cfg.Extra["base_url"].(string)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	wsURL, _ := cfg.Extra["ws_url"].(string)
	if wsURL == "" {
		wsURL = defaultWSURL
	}

	if cfg.Model == "" {
		cfg.Model = defaultModel
	}

	client := httpclient.New(
		httpclient.WithBaseURL(baseURL),
		httpclient.WithBearerToken("Token "+apiKey),
		httpclient.WithRetries(2),
	)

	return &Engine{
		client:  client,
		apiKey:  apiKey,
		baseURL: baseURL,
		wsURL:   wsURL,
		cfg:     cfg,
	}, nil
}

// deepgramResponse is the JSON response from Deepgram's transcription API.
type deepgramResponse struct {
	Results struct {
		Channels []struct {
			Alternatives []struct {
				Transcript string          `json:"transcript"`
				Confidence float64         `json:"confidence"`
				Words      []deepgramWord  `json:"words"`
			} `json:"alternatives"`
		} `json:"channels"`
	} `json:"results"`
}

type deepgramWord struct {
	Word       string  `json:"word"`
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Confidence float64 `json:"confidence"`
}

// Transcribe converts a complete audio buffer to text using Deepgram's REST API.
func (e *Engine) Transcribe(ctx context.Context, audio []byte, opts ...stt.Option) (string, error) {
	cfg := e.cfg
	for _, opt := range opts {
		opt(&cfg)
	}

	params := e.buildQueryParams(cfg)
	path := "/listen?" + params.Encode()

	resp, err := e.client.Do(ctx, http.MethodPost, path, nil, map[string]string{
		"Content-Type":  "audio/wav",
		"Authorization": "Token " + e.apiKey,
	})
	if err != nil {
		return "", fmt.Errorf("deepgram: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", &httpclient.APIError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}

	// We need to send raw audio, not JSON, so we handle the request manually.
	// Re-do with raw body.
	return e.transcribeRaw(ctx, audio, cfg)
}

func (e *Engine) transcribeRaw(ctx context.Context, audio []byte, cfg stt.Config) (string, error) {
	params := e.buildQueryParams(cfg)
	u := e.baseURL + "/listen?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(string(audio)))
	if err != nil {
		return "", fmt.Errorf("deepgram: create request: %w", err)
	}
	req.Header.Set("Content-Type", "audio/wav")
	req.Header.Set("Authorization", "Token "+e.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("deepgram: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", &httpclient.APIError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}

	var result deepgramResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("deepgram: decode response: %w", err)
	}

	if len(result.Results.Channels) > 0 && len(result.Results.Channels[0].Alternatives) > 0 {
		return result.Results.Channels[0].Alternatives[0].Transcript, nil
	}
	return "", nil
}

// deepgramStreamResponse is a message from the Deepgram WebSocket stream.
type deepgramStreamResponse struct {
	Type    string `json:"type"`
	Channel struct {
		Alternatives []struct {
			Transcript string         `json:"transcript"`
			Confidence float64        `json:"confidence"`
			Words      []deepgramWord `json:"words"`
		} `json:"alternatives"`
	} `json:"channel"`
	IsFinal  bool    `json:"is_final"`
	Start    float64 `json:"start"`
	Duration float64 `json:"duration"`
}

// dialStream opens a WebSocket connection to Deepgram's streaming endpoint.
func (e *Engine) dialStream(ctx context.Context, cfg stt.Config) (*websocket.Conn, error) {
	params := e.buildQueryParams(cfg)
	wsEndpoint := e.wsURL + "/listen?" + params.Encode()

	headers := http.Header{}
	headers.Set("Authorization", "Token "+e.apiKey)

	conn, _, err := websocket.Dial(ctx, wsEndpoint, &websocket.DialOptions{
		HTTPHeader: headers,
	})
	if err != nil {
		return nil, fmt.Errorf("deepgram: websocket dial: %w", err)
	}
	return conn, nil
}

// parseStreamMessage parses a Deepgram WebSocket message into a TranscriptEvent.
// Returns the event and whether it should be emitted.
func (e *Engine) parseStreamMessage(data []byte, language string) (stt.TranscriptEvent, bool) {
	var msg deepgramStreamResponse
	if jsonErr := json.Unmarshal(data, &msg); jsonErr != nil {
		return stt.TranscriptEvent{}, false
	}

	if msg.Type != "Results" || len(msg.Channel.Alternatives) == 0 {
		return stt.TranscriptEvent{}, false
	}

	alt := msg.Channel.Alternatives[0]
	if alt.Transcript == "" {
		return stt.TranscriptEvent{}, false
	}

	var words []stt.Word
	for _, w := range alt.Words {
		words = append(words, stt.Word{
			Text:       w.Word,
			Start:      time.Duration(w.Start * float64(time.Second)),
			End:        time.Duration(w.End * float64(time.Second)),
			Confidence: w.Confidence,
		})
	}

	return stt.TranscriptEvent{
		Text:       alt.Transcript,
		IsFinal:    msg.IsFinal,
		Confidence: alt.Confidence,
		Timestamp:  time.Duration(msg.Start * float64(time.Second)),
		Language:   language,
		Words:      words,
	}, true
}

// readStreamMessages reads WebSocket messages and sends parsed events to the channel.
func (e *Engine) readStreamMessages(ctx context.Context, conn *websocket.Conn, language string, events chan<- stt.TranscriptEvent, errs chan<- error) {
	defer close(events)
	for {
		_, data, readErr := conn.Read(ctx)
		if readErr != nil {
			if ctx.Err() == nil {
				errs <- fmt.Errorf("deepgram: websocket read: %w", readErr)
			}
			return
		}
		if event, ok := e.parseStreamMessage(data, language); ok {
			events <- event
		}
	}
}

// sendAudioStream sends audio chunks over the WebSocket and signals end of stream.
func (e *Engine) sendAudioStream(ctx context.Context, conn *websocket.Conn, audioStream iter.Seq2[[]byte, error], errs chan<- error) {
	for chunk, streamErr := range audioStream {
		if streamErr != nil {
			errs <- streamErr
			return
		}
		if writeErr := conn.Write(ctx, websocket.MessageBinary, chunk); writeErr != nil {
			if ctx.Err() == nil {
				errs <- fmt.Errorf("deepgram: websocket write: %w", writeErr)
			}
			return
		}
	}
	// Signal end of stream with close message.
	conn.Write(ctx, websocket.MessageText, []byte(`{"type": "CloseStream"}`))
}

// TranscribeStream converts a streaming audio source to transcript events
// using Deepgram's WebSocket API.
func (e *Engine) TranscribeStream(ctx context.Context, audioStream iter.Seq2[[]byte, error], opts ...stt.Option) iter.Seq2[stt.TranscriptEvent, error] {
	return func(yield func(stt.TranscriptEvent, error) bool) {
		cfg := e.cfg
		for _, opt := range opts {
			opt(&cfg)
		}

		conn, err := e.dialStream(ctx, cfg)
		if err != nil {
			yield(stt.TranscriptEvent{}, err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		events := make(chan stt.TranscriptEvent, 16)
		errs := make(chan error, 1)

		go e.readStreamMessages(ctx, conn, cfg.Language, events, errs)
		go e.sendAudioStream(ctx, conn, audioStream, errs)

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

func (e *Engine) buildQueryParams(cfg stt.Config) url.Values {
	params := url.Values{}
	params.Set("model", cfg.Model)
	if cfg.Language != "" {
		params.Set("language", cfg.Language)
	}
	if cfg.Punctuation {
		params.Set("punctuate", "true")
	}
	if cfg.Diarization {
		params.Set("diarize", "true")
	}
	if cfg.Encoding != "" {
		params.Set("encoding", cfg.Encoding)
	}
	if cfg.SampleRate > 0 {
		params.Set("sample_rate", fmt.Sprintf("%d", cfg.SampleRate))
	}
	return params
}
