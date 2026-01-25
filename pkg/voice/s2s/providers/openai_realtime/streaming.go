package openai_realtime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
)

// OpenAIRealtimeStreamingSession implements StreamingSession for OpenAI Realtime API.
type OpenAIRealtimeStreamingSession struct {
	ctx       context.Context
	conn      WebSocketConn
	config    *OpenAIRealtimeConfig
	provider  *OpenAIRealtimeProvider
	audioCh   chan iface.AudioOutputChunk
	sessionID string
	mu        sync.RWMutex
	closed    bool
}

// RealtimeEvent represents an event in the OpenAI Realtime API protocol.
type RealtimeEvent struct {
	Type    string          `json:"type"`
	EventID string          `json:"event_id,omitempty"`
	Body    json.RawMessage `json:"body,omitempty"`
}

// RealtimeAudioEvent represents an audio event.
type RealtimeAudioEvent struct {
	Audio string `json:"audio"` // base64 encoded audio
}

// RealtimeResponseEvent represents a response event.
type RealtimeResponseEvent struct {
	Type       string `json:"type"`
	Audio      string `json:"audio,omitempty"` // base64 encoded audio
	Transcript string `json:"transcript,omitempty"`
	Delta      string `json:"delta,omitempty"`
}

// NewOpenAIRealtimeStreamingSession creates a new streaming session.
func NewOpenAIRealtimeStreamingSession(ctx context.Context, config *OpenAIRealtimeConfig, provider *OpenAIRealtimeProvider) (*OpenAIRealtimeStreamingSession, error) {
	session := &OpenAIRealtimeStreamingSession{
		ctx:      ctx,
		config:   config,
		provider: provider,
		audioCh:  make(chan iface.AudioOutputChunk, 10),
	}

	// Build WebSocket URL
	wsURL := fmt.Sprintf("wss://api.openai.com/v1/realtime?model=%s&voice=%s", config.Model, config.VoiceID)

	// Set headers
	headers := make(map[string][]string)
	headers["Authorization"] = []string{"Bearer " + config.APIKey}
	headers["OpenAI-Beta"] = []string{"realtime=v1"}

	// Use injected dialer or create default one
	var dialer WebSocketDialer
	if provider.wsDialer != nil {
		dialer = provider.wsDialer
	} else {
		// Create default WebSocket dialer
		wsDialer := &websocket.Dialer{
			HandshakeTimeout: config.Timeout,
		}
		dialer = &defaultWebSocketDialer{dialer: wsDialer}
	}

	// Connect to WebSocket
	conn, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OpenAI Realtime API: %w", err)
	}

	session.conn = conn

	// Start goroutine to receive messages
	go session.receiveMessages()

	// Send session configuration
	if err := session.sendSessionConfig(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to send session config: %w", err)
	}

	return session, nil
}

// sendSessionConfig sends the session configuration to OpenAI Realtime API.
func (s *OpenAIRealtimeStreamingSession) sendSessionConfig() error {
	configEvent := RealtimeEvent{
		Type: "session.update",
		Body: json.RawMessage(fmt.Sprintf(`{
			"modalities": ["audio"],
			"instructions": "",
			"voice": "%s",
			"temperature": %f,
			"input_audio_format": "%s",
			"output_audio_format": "%s",
			"input_audio_transcription": {
				"model": "whisper-1"
			},
			"turn_detection": {
				"type": "server_vad",
				"threshold": 0.5,
				"prefix_padding_ms": 300,
				"silence_duration_ms": 500
			},
			"tools": [],
			"tool_choice": "auto"
		}`, s.config.VoiceID, s.config.Temperature, s.config.AudioFormat, s.config.AudioFormat)),
	}

	return s.conn.WriteJSON(configEvent)
}

// receiveMessages receives messages from the WebSocket connection.
func (s *OpenAIRealtimeStreamingSession) receiveMessages() {
	defer close(s.audioCh)

	for {
		s.mu.RLock()
		closed := s.closed
		s.mu.RUnlock()

		if closed {
			return
		}

		var event RealtimeEvent
		if err := s.conn.ReadJSON(&event); err != nil {
			if !s.isClosed() {
				s.audioCh <- iface.AudioOutputChunk{
					Error: fmt.Errorf("failed to read message: %w", err),
				}
			}
			return
		}

		// Process different event types
		switch event.Type {
		case "response.audio.delta":
			chunk, err := s.processAudioDelta(event.Body)
			if err != nil {
				s.audioCh <- iface.AudioOutputChunk{Error: err}
				continue
			}
			if chunk != nil {
				s.audioCh <- *chunk
			}
		case "response.audio_transcript.delta":
			// Handle transcript delta (optional)
		case "conversation.item.input_audio_transcription.completed":
			// Handle transcription completion (optional)
		case "error":
			s.audioCh <- iface.AudioOutputChunk{
				Error: fmt.Errorf("OpenAI Realtime API error: %s", string(event.Body)),
			}
		}
	}
}

// processAudioDelta processes an audio delta event.
func (s *OpenAIRealtimeStreamingSession) processAudioDelta(body json.RawMessage) (*iface.AudioOutputChunk, error) {
	var deltaEvent RealtimeResponseEvent
	if err := json.Unmarshal(body, &deltaEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal audio delta: %w", err)
	}

	if deltaEvent.Audio == "" {
		return nil, nil
	}

	// Decode base64 audio
	audioData, err := base64.StdEncoding.DecodeString(deltaEvent.Audio)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio: %w", err)
	}

	return &iface.AudioOutputChunk{
		Audio: audioData,
	}, nil
}

// isClosed checks if the session is closed.
func (s *OpenAIRealtimeStreamingSession) isClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

// SendAudio implements the StreamingSession interface.
func (s *OpenAIRealtimeStreamingSession) SendAudio(ctx context.Context, audio []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return s2s.NewS2SError("SendAudio", s2s.ErrCodeStreamClosed,
			errors.New("streaming session is closed"))
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return s2s.NewS2SError("SendAudio", s2s.ErrCodeContextCanceled,
			fmt.Errorf("context canceled: %w", ctx.Err()))
	case <-s.ctx.Done():
		return s2s.NewS2SError("SendAudio", s2s.ErrCodeContextCanceled,
			fmt.Errorf("session context canceled: %w", s.ctx.Err()))
	default:
	}

	// Encode audio as base64
	audioBase64 := base64.StdEncoding.EncodeToString(audio)

	// Send input_audio_buffer.append event
	event := RealtimeEvent{
		Type: "input_audio_buffer.append",
		Body: json.RawMessage(fmt.Sprintf(`{"audio": "%s"}`, audioBase64)),
	}

	if err := s.conn.WriteJSON(event); err != nil {
		return s2s.NewS2SError("SendAudio", s2s.ErrCodeStreamError,
			fmt.Errorf("failed to send audio: %w", err))
	}

	// Send input_audio_buffer.commit event to process the audio
	commitEvent := RealtimeEvent{
		Type: "input_audio_buffer.commit",
	}

	if err := s.conn.WriteJSON(commitEvent); err != nil {
		return s2s.NewS2SError("SendAudio", s2s.ErrCodeStreamError,
			fmt.Errorf("failed to commit audio: %w", err))
	}

	return nil
}

// ReceiveAudio implements the StreamingSession interface.
func (s *OpenAIRealtimeStreamingSession) ReceiveAudio() <-chan iface.AudioOutputChunk {
	return s.audioCh
}

// Close implements the StreamingSession interface.
func (s *OpenAIRealtimeStreamingSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	close(s.audioCh)

	// Send close event
	if s.conn != nil {
		closeEvent := RealtimeEvent{
			Type: "session.update",
			Body: json.RawMessage(`{"session": {"status": "closed"}}`),
		}
		_ = s.conn.WriteJSON(closeEvent)

		// Close WebSocket connection
		_ = s.conn.Close()
	}

	return nil
}
