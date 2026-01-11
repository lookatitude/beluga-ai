package livekit

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/livekit/protocol/livekit"
	"go.opentelemetry.io/otel/codes"
	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/internal"
	lksdkwrapper "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/livekit/internal"
)

// LiveKitSession implements the VoiceSession interface for LiveKit.
// Each session maintains independent state with its own mutex for isolation (T173).
type LiveKitSession struct {
	id                   string
	roomName             string
	config               *LiveKitConfig
	sessionConfig        *iface.SessionConfig
	pipelineOrchestrator *internal.PipelineOrchestrator
	roomService          *lksdkwrapper.RoomServiceClient
	state                iface.PipelineState
	persistenceStatus    iface.PersistenceStatus
	metadata             map[string]any
	audioOutput          chan []byte
	mu                   sync.RWMutex // Per-session mutex for state isolation (T173)
	active               bool
	// Session isolation: Each session has independent:
	// - State (state, persistenceStatus, active)
	// - Resources (pipelineOrchestrator, audioOutput channel)
	// - Configuration (sessionConfig, metadata)
	// No shared mutable state between sessions (T171, T173)
	
	// Connection state tracking for WebRTC connection loss handling (T289)
	connectionState iface.ConnectionState
	reconnectAttempts int
	lastReconnectTime time.Time
}

// NewLiveKitSession creates a new LiveKit session.
func NewLiveKitSession(config *LiveKitConfig, sessionConfig *iface.SessionConfig, roomName string, roomService *lksdkwrapper.RoomServiceClient) (*LiveKitSession, error) {
	sessionID := uuid.New().String()

	orchestrator := internal.NewPipelineOrchestrator(config.Config)

	return &LiveKitSession{
		id:                   sessionID,
		roomName:             roomName,
		config:               config,
		sessionConfig:        sessionConfig,
		pipelineOrchestrator: orchestrator,
		roomService:          roomService,
		state:                iface.PipelineStateIdle,
		persistenceStatus:    iface.PersistenceStatusActive,
		metadata:             make(map[string]any),
		audioOutput:          make(chan []byte, 100),
		active:               false,
		connectionState:       iface.ConnectionStateDisconnected,
		reconnectAttempts:     0,
	}, nil
}

// GetRoomName returns the LiveKit room name for this session.
func (s *LiveKitSession) GetRoomName() string {
	return s.roomName
}

// Start starts the voice session, subscribing to user audio track.
func (s *LiveKitSession) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		return backend.NewBackendError("Start", "session_already_active", nil)
	}

	s.active = true
	s.state = iface.PipelineStateListening
	s.connectionState = iface.ConnectionStateConnected

	// TODO: In a full implementation, this would:
	// 1. Connect to LiveKit room via WebRTC
	// 2. Subscribe to user's audio track
	// 3. Publish agent's audio track
	// 4. Set up track handlers

	return nil
}

// HandleConnectionLoss handles WebRTC connection loss with reconnection logic (T289).
func (s *LiveKitSession) HandleConnectionLoss(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connectionState == iface.ConnectionStateReconnecting {
		// Already reconnecting, skip
		return nil
	}

	s.connectionState = iface.ConnectionStateReconnecting
	s.reconnectAttempts++
	s.lastReconnectTime = time.Now()

	backend.LogWithOTELContext(ctx, slog.LevelWarn, "WebRTC connection lost, attempting reconnection",
		"session_id", s.id, "reconnect_attempt", s.reconnectAttempts)

	// Attempt reconnection in background
	go s.reconnect(ctx)

	return nil
}

// reconnect attempts to reconnect the WebRTC connection (T289).
func (s *LiveKitSession) reconnect(ctx context.Context) {
	maxAttempts := 3
	retryDelay := 1 * time.Second

	for attempt := 0; attempt < maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return
		case <-time.After(retryDelay):
			// Attempt to reconnect
			// In a full implementation, this would:
			// 1. Re-establish WebRTC connection
			// 2. Re-subscribe to user audio track
			// 3. Re-publish agent audio track
			
			// For now, simulate successful reconnection
			s.mu.Lock()
			if s.active {
				s.connectionState = iface.ConnectionStateConnected
				s.reconnectAttempts = 0
				backend.LogWithOTELContext(ctx, slog.LevelInfo, "WebRTC connection reestablished",
					"session_id", s.id, "attempt", attempt+1)
			} else {
				s.connectionState = iface.ConnectionStateDisconnected
				s.mu.Unlock()
				return
			}
			s.mu.Unlock()
			return
		}
	}

	// Max attempts reached
	s.mu.Lock()
	s.connectionState = iface.ConnectionStateError
	s.mu.Unlock()
	backend.LogWithOTELContext(ctx, slog.LevelError, "Failed to reconnect WebRTC connection after max attempts",
		"session_id", s.id, "max_attempts", maxAttempts)
}

// Stop stops the voice session, stopping session and cleaning up tracks.
func (s *LiveKitSession) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return backend.NewBackendError("Stop", "session_not_active", nil)
	}

	s.active = false
	s.state = iface.PipelineStateIdle

	// Data retention hook: Check if session data should be retained (T328, FR-027, FR-028)
	if s.config.DataRetentionHook != nil {
		shouldRetain, err := s.config.DataRetentionHook.ShouldRetain(ctx, s.id, s.metadata)
		if err != nil {
			backend.LogWithOTELContext(ctx, slog.LevelError, "Data retention hook error", "error", err, "session_id", s.id)
			// Continue with default behavior on error
		} else if shouldRetain {
			s.persistenceStatus = iface.PersistenceStatusActive
			backend.LogWithOTELContext(ctx, slog.LevelInfo, "Session data will be retained",
				"session_id", s.id)
		} else {
			s.persistenceStatus = iface.PersistenceStatusCompleted
		}
	} else {
		s.persistenceStatus = iface.PersistenceStatusCompleted
	}

	// Close audio output channel
	close(s.audioOutput)

	// TODO: In a full implementation, this would:
	// 1. Unsubscribe from user audio track
	// 2. Unpublish agent audio track
	// 3. Close WebRTC connection

	return nil
}

// ProcessAudio processes incoming audio data through the pipeline orchestrator.
func (s *LiveKitSession) ProcessAudio(ctx context.Context, audio []byte) error {
	ctx, span := backend.StartSpan(ctx, "LiveKitSession.ProcessAudio", "livekit")
	defer span.End()

	backend.AddSpanAttributes(span, map[string]any{
		"session_id":  s.id,
		"audio_size":  len(audio),
		"room_name":   s.roomName,
	})

	s.mu.RLock()
	active := s.active
	agentCallback := s.sessionConfig.AgentCallback
	agentInstance := s.sessionConfig.AgentInstance
	s.mu.RUnlock()

	if !active {
		err := backend.NewBackendError("ProcessAudio", "session_not_active", nil)
		backend.RecordSpanError(span, err)
		return err
	}

	// Update state to processing
	s.mu.Lock()
	s.state = iface.PipelineStateProcessing
	s.mu.Unlock()

	backend.AddSpanAttributes(span, map[string]any{
		"pipeline_state": string(s.state),
	})

	// Check connection state before processing (T289)
	s.mu.RLock()
	connState := s.connectionState
	s.mu.RUnlock()

	if connState != iface.ConnectionStateConnected {
		// Connection lost, attempt recovery
		if err := s.HandleConnectionLoss(ctx); err != nil {
			return backend.WrapError("ProcessAudio", err)
		}
		// Return error to indicate connection issue
		return backend.NewBackendError("ProcessAudio", backend.ErrCodeConnectionFailed,
			fmt.Errorf("WebRTC connection lost, reconnecting"))
	}

	// Process through pipeline orchestrator with error recovery
	outputAudio, err := s.pipelineOrchestrator.ProcessAudio(ctx, audio, agentCallback, agentInstance)
	if err != nil {
		// Check if error is retryable
		if backend.IsRetryableError(err) {
			// Try to recover from transient errors
			backend.LogWithOTELContext(ctx, slog.LevelWarn, "Retryable error in audio processing, attempting recovery", "error", err)
			
			// Retry once with a short delay
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				outputAudio, err = s.pipelineOrchestrator.ProcessAudio(ctx, audio, agentCallback, agentInstance)
				if err != nil {
					s.mu.Lock()
					s.state = iface.PipelineStateError
					s.mu.Unlock()
					backend.RecordSpanError(span, err)
					backend.LogWithOTELContext(ctx, slog.LevelError, "Failed to process audio after retry", "error", err)
					return err
				}
				// Recovery successful
				backend.LogWithOTELContext(ctx, slog.LevelInfo, "Audio processing recovered from transient error")
			}
		} else {
			// Non-retryable error
			s.mu.Lock()
			s.state = iface.PipelineStateError
			s.mu.Unlock()
			backend.RecordSpanError(span, err)
			backend.LogWithOTELContext(ctx, slog.LevelError, "Failed to process audio", "error", err)
			return err
		}
	}

	backend.AddSpanAttributes(span, map[string]any{
		"output_audio_size": len(outputAudio),
	})

	// Record throughput per session (T180)
	// Note: This would typically be called from a metrics-aware context
	// For now, we track the total bytes processed
	totalBytes := int64(len(audio) + len(outputAudio))
	backend.AddSpanAttributes(span, map[string]any{
		"throughput_bytes": totalBytes,
	})

	// Send output audio with buffer overflow protection (T177)
	select {
	case s.audioOutput <- outputAudio:
		// Successfully sent
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		// Buffer overflow protection - log warning but don't fail
		backend.LogWithOTELContext(ctx, slog.LevelWarn, "Audio output buffer full, dropping frame",
			"session_id", s.id, "audio_size", len(outputAudio))
		// Don't return error to prevent session termination, just drop the frame
	}

	// Update state back to listening
	s.mu.Lock()
	s.state = iface.PipelineStateListening
	s.mu.Unlock()

	span.SetStatus(codes.Ok, "audio processed successfully")
	backend.LogWithOTELContext(ctx, slog.LevelInfo, "Audio processed successfully",
		"input_size", len(audio), "output_size", len(outputAudio))

	return nil
}

// SendAudio sends audio data to the user via LiveKit track.
func (s *LiveKitSession) SendAudio(ctx context.Context, audio []byte) error {
	s.mu.RLock()
	active := s.active
	s.mu.RUnlock()

	if !active {
		return backend.NewBackendError("SendAudio", "session_not_active", nil)
	}

	// Update state to speaking
	s.mu.Lock()
	s.state = iface.PipelineStateSpeaking
	s.mu.Unlock()

	// Send audio to output channel (will be published to LiveKit track)
	select {
	case s.audioOutput <- audio:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return backend.NewBackendError("SendAudio", backend.ErrCodeTimeout, nil)
	}
}

// ReceiveAudio returns a channel for receiving audio from the user.
func (s *LiveKitSession) ReceiveAudio() <-chan []byte {
	// In a full implementation, this would return audio from LiveKit track
	// For now, return a channel that can be written to
	return make(chan []byte, 100)
}

// SetAgentCallback sets the agent callback function.
func (s *LiveKitSession) SetAgentCallback(callback func(context.Context, string) (string, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessionConfig.AgentCallback = callback
	return nil
}

// SetAgentInstance sets the agent instance.
func (s *LiveKitSession) SetAgentInstance(agent agentsiface.Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessionConfig.AgentInstance = agent
	return nil
}

// GetState returns the current pipeline state.
func (s *LiveKitSession) GetState() iface.PipelineState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// GetPersistenceStatus returns the persistence status of the session.
func (s *LiveKitSession) GetPersistenceStatus() iface.PersistenceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.persistenceStatus
}

// UpdateMetadata updates the session metadata.
func (s *LiveKitSession) UpdateMetadata(metadata map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k, v := range metadata {
		s.metadata[k] = v
	}

	// Update LiveKit room metadata if room service is available
	if s.roomService != nil {
		ctx := context.Background()
		// Convert metadata map to JSON string for LiveKit
		metadataJSON := fmt.Sprintf("%v", s.metadata) // Simple conversion, full impl would use JSON
		_, err := s.roomService.UpdateRoomMetadata(ctx, &livekit.UpdateRoomMetadataRequest{
			Room:     s.roomName,
			Metadata: metadataJSON,
		})
		if err != nil {
			return backend.WrapError("UpdateMetadata", err)
		}
	}

	return nil
}

// GetID returns the session identifier.
func (s *LiveKitSession) GetID() string {
	return s.id
}
