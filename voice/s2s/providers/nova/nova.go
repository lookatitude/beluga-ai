package nova

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
	defaultModel  = "amazon.nova-sonic-v1:0"
	defaultRegion = "us-east-1"
	// Nova S2S uses a WebSocket endpoint for bidirectional streaming.
	defaultBaseURL = "wss://bedrock-runtime.%s.amazonaws.com/model/%s/converse-stream"
)

var _ s2s.S2S = (*Engine)(nil) // compile-time interface check

func init() {
	s2s.Register("nova", func(cfg s2s.Config) (s2s.S2S, error) {
		return New(cfg)
	})
}

// Engine implements s2s.S2S using Amazon Nova via Bedrock.
type Engine struct {
	region  string
	baseURL string
	cfg     s2s.Config
}

// New creates a new Amazon Nova S2S engine.
func New(cfg s2s.Config) (*Engine, error) {
	region, _ := cfg.Extra["region"].(string)
	if region == "" {
		region = defaultRegion
	}

	if cfg.Model == "" {
		cfg.Model = defaultModel
	}

	baseURL, _ := cfg.Extra["base_url"].(string)
	if baseURL == "" {
		baseURL = fmt.Sprintf(defaultBaseURL, region, cfg.Model)
	}

	return &Engine{
		region:  region,
		baseURL: baseURL,
		cfg:     cfg,
	}, nil
}

// Start initiates a new Nova S2S session.
func (e *Engine) Start(ctx context.Context, opts ...s2s.Option) (s2s.Session, error) {
	cfg := e.cfg
	for _, opt := range opts {
		opt(&cfg)
	}

	conn, _, err := websocket.Dial(ctx, e.baseURL, &websocket.DialOptions{
		HTTPHeader: http.Header{},
	})
	if err != nil {
		return nil, fmt.Errorf("nova: websocket dial: %w", err)
	}

	sess := &novaSession{
		conn:   conn,
		events: make(chan s2s.SessionEvent, 64),
		done:   make(chan struct{}),
		cfg:    cfg,
	}

	// Send session configuration.
	if err := sess.sendSetup(ctx); err != nil {
		conn.Close(websocket.StatusNormalClosure, "")
		return nil, fmt.Errorf("nova: setup: %w", err)
	}

	go sess.readLoop(ctx)

	return sess, nil
}

// novaSession implements s2s.Session for Amazon Nova.
type novaSession struct {
	conn   *websocket.Conn
	events chan s2s.SessionEvent
	done   chan struct{}
	once   sync.Once
	cfg    s2s.Config
}

// novaServerEvent represents a server event from Nova.
type novaServerEvent struct {
	Type       string          `json:"type"`
	AudioChunk string          `json:"audioChunk,omitempty"` // base64
	Text       string          `json:"text,omitempty"`
	Transcript string          `json:"transcript,omitempty"`
	ToolUse    *novaToolUse    `json:"toolUse,omitempty"`
	Error      *novaError      `json:"error,omitempty"`
}

type novaToolUse struct {
	ToolUseID string          `json:"toolUseId"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
}

type novaError struct {
	Message string `json:"message"`
}

func (s *novaSession) sendSetup(ctx context.Context) error {
	setup := map[string]any{
		"type": "sessionStart",
		"inferenceConfiguration": map[string]any{
			"maxTokens": 1024,
		},
	}

	if s.cfg.Instructions != "" {
		setup["system"] = []map[string]any{
			{"text": s.cfg.Instructions},
		}
	}

	if len(s.cfg.Tools) > 0 {
		tools := make([]map[string]any, len(s.cfg.Tools))
		for i, t := range s.cfg.Tools {
			tools[i] = map[string]any{
				"toolSpec": map[string]any{
					"name":        t.Name,
					"description": t.Description,
					"inputSchema": map[string]any{"json": t.InputSchema},
				},
			}
		}
		setup["toolConfig"] = map[string]any{"tools": tools}
	}

	data, err := json.Marshal(setup)
	if err != nil {
		return err
	}
	return s.conn.Write(ctx, websocket.MessageText, data)
}

func (s *novaSession) readLoop(ctx context.Context) {
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
						Error: fmt.Errorf("nova: read: %w", err),
					}
				}
				return
			}
		}

		var event novaServerEvent
		if jsonErr := json.Unmarshal(data, &event); jsonErr != nil {
			continue
		}

		switch event.Type {
		case "contentBlockDelta":
			if event.AudioChunk != "" {
				audioData, decErr := base64.StdEncoding.DecodeString(event.AudioChunk)
				if decErr == nil && len(audioData) > 0 {
					s.events <- s2s.SessionEvent{
						Type:  s2s.EventAudioOutput,
						Audio: audioData,
					}
				}
			}
			if event.Text != "" {
				s.events <- s2s.SessionEvent{
					Type: s2s.EventTextOutput,
					Text: event.Text,
				}
			}

		case "contentBlockStop", "messageStop":
			s.events <- s2s.SessionEvent{
				Type: s2s.EventTurnEnd,
			}

		case "toolUse":
			if event.ToolUse != nil {
				args, _ := json.Marshal(event.ToolUse.Input)
				s.events <- s2s.SessionEvent{
					Type: s2s.EventToolCall,
					ToolCall: &schema.ToolCall{
						ID:        event.ToolUse.ToolUseID,
						Name:      event.ToolUse.Name,
						Arguments: string(args),
					},
				}
			}

		case "inputTranscript":
			if event.Transcript != "" {
				s.events <- s2s.SessionEvent{
					Type: s2s.EventTranscript,
					Text: event.Transcript,
				}
			}

		case "error":
			msg := "unknown error"
			if event.Error != nil {
				msg = event.Error.Message
			}
			s.events <- s2s.SessionEvent{
				Type:  s2s.EventError,
				Error: fmt.Errorf("nova: %s", msg),
			}
		}
	}
}

// SendAudio sends an audio chunk to Nova.
func (s *novaSession) SendAudio(ctx context.Context, audio []byte) error {
	msg := map[string]any{
		"type":       "inputAudio",
		"audioChunk": base64.StdEncoding.EncodeToString(audio),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.conn.Write(ctx, websocket.MessageText, data)
}

// SendText sends a text message to Nova.
func (s *novaSession) SendText(ctx context.Context, text string) error {
	msg := map[string]any{
		"type": "inputText",
		"content": []map[string]any{
			{"text": text},
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.conn.Write(ctx, websocket.MessageText, data)
}

// SendToolResult sends a tool execution result back to Nova.
func (s *novaSession) SendToolResult(ctx context.Context, result schema.ToolResult) error {
	output := ""
	for _, part := range result.Content {
		if tp, ok := part.(schema.TextPart); ok && tp.Text != "" {
			output += tp.Text
		}
	}

	msg := map[string]any{
		"type": "toolResult",
		"toolResult": map[string]any{
			"toolUseId": result.CallID,
			"content":   []map[string]any{{"text": output}},
			"status":    "success",
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.conn.Write(ctx, websocket.MessageText, data)
}

// Recv returns the channel of session events.
func (s *novaSession) Recv() <-chan s2s.SessionEvent {
	return s.events
}

// Interrupt signals user interruption.
func (s *novaSession) Interrupt(ctx context.Context) error {
	msg := map[string]any{
		"type": "inputAudioInterrupt",
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.conn.Write(ctx, websocket.MessageText, data)
}

// Close terminates the session.
func (s *novaSession) Close() error {
	s.once.Do(func() {
		close(s.done)
	})
	return s.conn.Close(websocket.StatusNormalClosure, "")
}
