package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/coder/websocket"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/voice/s2s"
)

const (
	defaultBaseURL = "wss://api.openai.com/v1/realtime"
	defaultModel   = "gpt-4o-realtime-preview"
)

var _ s2s.S2S = (*Engine)(nil) // compile-time interface check

func init() {
	s2s.Register("openai_realtime", func(cfg s2s.Config) (s2s.S2S, error) {
		return New(cfg)
	})
}

// Engine implements s2s.S2S using the OpenAI Realtime API.
type Engine struct {
	apiKey  string
	baseURL string
	cfg     s2s.Config
}

// New creates a new OpenAI Realtime S2S engine.
func New(cfg s2s.Config) (*Engine, error) {
	apiKey, _ := cfg.Extra["api_key"].(string)
	if apiKey == "" {
		return nil, fmt.Errorf("openai realtime: api_key is required in Extra")
	}

	baseURL, _ := cfg.Extra["base_url"].(string)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	if cfg.Model == "" {
		cfg.Model = defaultModel
	}
	if cfg.Voice == "" {
		cfg.Voice = "alloy"
	}

	return &Engine{
		apiKey:  apiKey,
		baseURL: baseURL,
		cfg:     cfg,
	}, nil
}

// Start initiates a new Realtime session with the OpenAI API.
func (e *Engine) Start(ctx context.Context, opts ...s2s.Option) (s2s.Session, error) {
	cfg := e.cfg
	for _, opt := range opts {
		opt(&cfg)
	}

	wsURL := fmt.Sprintf("%s?model=%s", e.baseURL, cfg.Model)
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+e.apiKey)
	headers.Set("OpenAI-Beta", "realtime=v1")

	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: headers,
	})
	if err != nil {
		return nil, fmt.Errorf("openai realtime: websocket dial: %w", err)
	}

	sess := &realtimeSession{
		conn:   conn,
		events: make(chan s2s.SessionEvent, 64),
		done:   make(chan struct{}),
		cfg:    cfg,
	}

	// Send session configuration.
	if err := sess.sendSessionUpdate(ctx); err != nil {
		conn.Close(websocket.StatusNormalClosure, "")
		return nil, fmt.Errorf("openai realtime: session update: %w", err)
	}

	// Start reading events.
	go sess.readLoop(ctx)

	return sess, nil
}

// realtimeSession implements s2s.Session for OpenAI Realtime.
type realtimeSession struct {
	conn   *websocket.Conn
	events chan s2s.SessionEvent
	done   chan struct{}
	once   sync.Once
	cfg    s2s.Config
}

// serverEvent represents a server-sent event from the Realtime API.
type serverEvent struct {
	Type     string          `json:"type"`
	Delta    string          `json:"delta,omitempty"`
	Audio    string          `json:"audio,omitempty"`
	Text     string          `json:"text,omitempty"`
	Transcript string        `json:"transcript,omitempty"`
	Name     string          `json:"name,omitempty"`
	CallID   string          `json:"call_id,omitempty"`
	Arguments string         `json:"arguments,omitempty"`
	Error    *serverError    `json:"error,omitempty"`
	Item     json.RawMessage `json:"item,omitempty"`
	Response json.RawMessage `json:"response,omitempty"`
}

type serverError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (s *realtimeSession) sendSessionUpdate(ctx context.Context) error {
	msg := map[string]any{
		"type": "session.update",
		"session": map[string]any{
			"modalities":            []string{"audio", "text"},
			"voice":                 s.cfg.Voice,
			"input_audio_format":    "pcm16",
			"output_audio_format":   "pcm16",
			"turn_detection":        map[string]any{"type": "server_vad"},
		},
	}

	if s.cfg.Instructions != "" {
		sessionMap := msg["session"].(map[string]any)
		sessionMap["instructions"] = s.cfg.Instructions
	}

	if len(s.cfg.Tools) > 0 {
		tools := make([]map[string]any, len(s.cfg.Tools))
		for i, t := range s.cfg.Tools {
			tools[i] = map[string]any{
				"type":        "function",
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.InputSchema,
			}
		}
		sessionMap := msg["session"].(map[string]any)
		sessionMap["tools"] = tools
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.conn.Write(ctx, websocket.MessageText, data)
}

func (s *realtimeSession) readLoop(ctx context.Context) {
	defer close(s.events)
	for {
		_, data, err := s.conn.Read(ctx)
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				if ctx.Err() == nil {
					s.events <- s2s.SessionEvent{
						Type:  s2s.EventError,
						Error: fmt.Errorf("openai realtime: read: %w", err),
					}
				}
				return
			}
		}

		var event serverEvent
		if jsonErr := json.Unmarshal(data, &event); jsonErr != nil {
			continue
		}

		switch event.Type {
		case "response.audio.delta":
			audioData, decErr := base64.StdEncoding.DecodeString(event.Delta)
			if decErr == nil && len(audioData) > 0 {
				s.events <- s2s.SessionEvent{
					Type:  s2s.EventAudioOutput,
					Audio: audioData,
				}
			}

		case "response.audio_transcript.delta":
			if event.Delta != "" {
				s.events <- s2s.SessionEvent{
					Type: s2s.EventTextOutput,
					Text: event.Delta,
				}
			}

		case "conversation.item.input_audio_transcription.completed":
			if event.Transcript != "" {
				s.events <- s2s.SessionEvent{
					Type: s2s.EventTranscript,
					Text: event.Transcript,
				}
			}

		case "response.function_call_arguments.done":
			s.events <- s2s.SessionEvent{
				Type: s2s.EventToolCall,
				ToolCall: &schema.ToolCall{
					ID:        event.CallID,
					Name:      event.Name,
					Arguments: event.Arguments,
				},
			}

		case "response.done":
			s.events <- s2s.SessionEvent{
				Type: s2s.EventTurnEnd,
			}

		case "error":
			msg := "unknown error"
			if event.Error != nil {
				msg = event.Error.Message
			}
			s.events <- s2s.SessionEvent{
				Type:  s2s.EventError,
				Error: fmt.Errorf("openai realtime: %s", msg),
			}
		}
	}
}

// SendAudio sends an audio chunk to the Realtime API.
func (s *realtimeSession) SendAudio(ctx context.Context, audio []byte) error {
	encoded := base64.StdEncoding.EncodeToString(audio)
	msg := map[string]any{
		"type":  "input_audio_buffer.append",
		"audio": encoded,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.conn.Write(ctx, websocket.MessageText, data)
}

// SendText sends a text message as a conversation item.
func (s *realtimeSession) SendText(ctx context.Context, text string) error {
	msg := map[string]any{
		"type": "conversation.item.create",
		"item": map[string]any{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": text,
				},
			},
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if err := s.conn.Write(ctx, websocket.MessageText, data); err != nil {
		return err
	}
	// Trigger a response.
	return s.sendResponseCreate(ctx)
}

// SendToolResult sends a tool execution result back to the model.
func (s *realtimeSession) SendToolResult(ctx context.Context, result schema.ToolResult) error {
	output := ""
	for _, part := range result.Content {
		if tp, ok := part.(schema.TextPart); ok && tp.Text != "" {
			output += tp.Text
		}
	}

	msg := map[string]any{
		"type": "conversation.item.create",
		"item": map[string]any{
			"type":    "function_call_output",
			"call_id": result.CallID,
			"output":  output,
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if err := s.conn.Write(ctx, websocket.MessageText, data); err != nil {
		return err
	}
	return s.sendResponseCreate(ctx)
}

func (s *realtimeSession) sendResponseCreate(ctx context.Context) error {
	msg := map[string]any{
		"type": "response.create",
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.conn.Write(ctx, websocket.MessageText, data)
}

// Recv returns the channel of session events.
func (s *realtimeSession) Recv() <-chan s2s.SessionEvent {
	return s.events
}

// Interrupt signals that the user has interrupted the model's output.
func (s *realtimeSession) Interrupt(ctx context.Context) error {
	msg := map[string]any{
		"type": "response.cancel",
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.conn.Write(ctx, websocket.MessageText, data)
}

// Close terminates the session and releases resources.
func (s *realtimeSession) Close() error {
	s.once.Do(func() {
		close(s.done)
	})
	return s.conn.Close(websocket.StatusNormalClosure, "")
}
