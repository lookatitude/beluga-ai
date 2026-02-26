package gemini

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
	defaultBaseURL = "wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent"
	defaultModel   = "gemini-2.0-flash-exp"
)

var _ s2s.S2S = (*Engine)(nil) // compile-time interface check

func init() {
	s2s.Register("gemini_live", func(cfg s2s.Config) (s2s.S2S, error) {
		return New(cfg)
	})
}

// Engine implements s2s.S2S using the Gemini Live API.
type Engine struct {
	apiKey  string
	baseURL string
	cfg     s2s.Config
}

// New creates a new Gemini Live S2S engine.
func New(cfg s2s.Config) (*Engine, error) {
	apiKey, _ := cfg.Extra["api_key"].(string)
	if apiKey == "" {
		return nil, fmt.Errorf("gemini live: api_key is required in Extra")
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

// Start initiates a new Gemini Live session.
func (e *Engine) Start(ctx context.Context, opts ...s2s.Option) (s2s.Session, error) {
	cfg := e.cfg
	for _, opt := range opts {
		opt(&cfg)
	}

	wsURL := fmt.Sprintf("%s?key=%s", e.baseURL, e.apiKey)
	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{},
	})
	if err != nil {
		return nil, fmt.Errorf("gemini live: websocket dial: %w", err)
	}

	sess := &geminiSession{
		conn:   conn,
		events: make(chan s2s.SessionEvent, 64),
		done:   make(chan struct{}),
		cfg:    cfg,
	}

	// Send setup message.
	if err := sess.sendSetup(ctx); err != nil {
		conn.Close(websocket.StatusNormalClosure, "")
		return nil, fmt.Errorf("gemini live: setup: %w", err)
	}

	go sess.readLoop(ctx)

	return sess, nil
}

// geminiSession implements s2s.Session for Gemini Live.
type geminiSession struct {
	conn   *websocket.Conn
	events chan s2s.SessionEvent
	done   chan struct{}
	once   sync.Once
	cfg    s2s.Config
}

// geminiServerMsg represents a server message from the Gemini Live API.
type geminiServerMsg struct {
	SetupComplete  json.RawMessage  `json:"setupComplete,omitempty"`
	ServerContent  *geminiContent   `json:"serverContent,omitempty"`
	ToolCall       *geminiToolCall  `json:"toolCall,omitempty"`
	ToolCallCancel json.RawMessage  `json:"toolCallCancellation,omitempty"`
}

type geminiContent struct {
	ModelTurn  *geminiTurn `json:"modelTurn,omitempty"`
	TurnComplete bool     `json:"turnComplete,omitempty"`
}

type geminiTurn struct {
	Parts []geminiPart `json:"parts,omitempty"`
}

type geminiPart struct {
	Text       string          `json:"text,omitempty"`
	InlineData *geminiBlob     `json:"inlineData,omitempty"`
}

type geminiBlob struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // base64
}

type geminiToolCall struct {
	FunctionCalls []geminiFuncCall `json:"functionCalls,omitempty"`
}

type geminiFuncCall struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

func (s *geminiSession) sendSetup(ctx context.Context) error {
	setup := map[string]any{
		"setup": map[string]any{
			"model": "models/" + s.cfg.Model,
			"generationConfig": map[string]any{
				"responseModalities": []string{"AUDIO"},
				"speechConfig": map[string]any{
					"voiceConfig": map[string]any{
						"prebuiltVoiceConfig": map[string]any{
							"voiceName": s.cfg.Voice,
						},
					},
				},
			},
		},
	}

	if s.cfg.Instructions != "" {
		setupInner := setup["setup"].(map[string]any)
		setupInner["systemInstruction"] = map[string]any{
			"parts": []map[string]any{
				{"text": s.cfg.Instructions},
			},
		}
	}

	if len(s.cfg.Tools) > 0 {
		funcs := make([]map[string]any, len(s.cfg.Tools))
		for i, t := range s.cfg.Tools {
			funcs[i] = map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.InputSchema,
			}
		}
		setupInner := setup["setup"].(map[string]any)
		setupInner["tools"] = []map[string]any{
			{"functionDeclarations": funcs},
		}
	}

	data, err := json.Marshal(setup)
	if err != nil {
		return err
	}
	return s.conn.Write(ctx, websocket.MessageText, data)
}

// handleServerContent processes server content messages and emits session events.
func (s *geminiSession) handleServerContent(content *geminiContent) {
	if content.ModelTurn != nil {
		for _, part := range content.ModelTurn.Parts {
			s.handleContentPart(part)
		}
	}
	if content.TurnComplete {
		s.events <- s2s.SessionEvent{
			Type: s2s.EventTurnEnd,
		}
	}
}

// handleContentPart processes a single content part and emits the appropriate event.
func (s *geminiSession) handleContentPart(part geminiPart) {
	if part.Text != "" {
		s.events <- s2s.SessionEvent{
			Type: s2s.EventTextOutput,
			Text: part.Text,
		}
	}
	if part.InlineData != nil {
		audioData, decErr := base64.StdEncoding.DecodeString(part.InlineData.Data)
		if decErr == nil && len(audioData) > 0 {
			s.events <- s2s.SessionEvent{
				Type:  s2s.EventAudioOutput,
				Audio: audioData,
			}
		}
	}
}

// handleToolCall processes tool call messages and emits session events.
func (s *geminiSession) handleToolCall(tc *geminiToolCall) {
	for _, fc := range tc.FunctionCalls {
		args, _ := json.Marshal(fc.Args)
		s.events <- s2s.SessionEvent{
			Type: s2s.EventToolCall,
			ToolCall: &schema.ToolCall{
				ID:        fc.ID,
				Name:      fc.Name,
				Arguments: string(args),
			},
		}
	}
}

// handleServerMessage dispatches a parsed server message to the appropriate handler.
func (s *geminiSession) handleServerMessage(msg geminiServerMsg) {
	if msg.ServerContent != nil {
		s.handleServerContent(msg.ServerContent)
	}
	if msg.ToolCall != nil {
		s.handleToolCall(msg.ToolCall)
	}
}

func (s *geminiSession) readLoop(ctx context.Context) {
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
						Error: fmt.Errorf("gemini live: read: %w", err),
					}
				}
				return
			}
		}

		var msg geminiServerMsg
		if jsonErr := json.Unmarshal(data, &msg); jsonErr != nil {
			continue
		}

		s.handleServerMessage(msg)
	}
}

// SendAudio sends an audio chunk to the Gemini Live session.
func (s *geminiSession) SendAudio(ctx context.Context, audio []byte) error {
	msg := map[string]any{
		"realtimeInput": map[string]any{
			"mediaChunks": []map[string]any{
				{
					"mimeType": "audio/pcm;rate=16000",
					"data":     base64.StdEncoding.EncodeToString(audio),
				},
			},
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.conn.Write(ctx, websocket.MessageText, data)
}

// SendText sends a text message to the Gemini Live session.
func (s *geminiSession) SendText(ctx context.Context, text string) error {
	msg := map[string]any{
		"clientContent": map[string]any{
			"turns": []map[string]any{
				{
					"role": "user",
					"parts": []map[string]any{
						{"text": text},
					},
				},
			},
			"turnComplete": true,
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.conn.Write(ctx, websocket.MessageText, data)
}

// SendToolResult sends a tool execution result back to the model.
func (s *geminiSession) SendToolResult(ctx context.Context, result schema.ToolResult) error {
	output := make(map[string]any)
	for _, part := range result.Content {
		if tp, ok := part.(schema.TextPart); ok && tp.Text != "" {
			output["result"] = tp.Text
		}
	}

	msg := map[string]any{
		"toolResponse": map[string]any{
			"functionResponses": []map[string]any{
				{
					"id":       result.CallID,
					"response": output,
				},
			},
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.conn.Write(ctx, websocket.MessageText, data)
}

// Recv returns the channel of session events.
func (s *geminiSession) Recv() <-chan s2s.SessionEvent {
	return s.events
}

// Interrupt signals user interruption to the Gemini Live session.
func (s *geminiSession) Interrupt(ctx context.Context) error {
	// Gemini Live handles interruptions via VAD on the server side.
	// Send an empty audio chunk to signal activity.
	return nil
}

// Close terminates the session and releases resources.
func (s *geminiSession) Close() error {
	s.once.Do(func() {
		close(s.done)
	})
	return s.conn.Close(websocket.StatusNormalClosure, "")
}
