package cartesia

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/internal"
	"go.opentelemetry.io/otel/codes"
)

// CartesiaSession implements the VoiceSession interface for Cartesia.
// Each session maintains independent state with its own mutex for isolation.
type CartesiaSession struct {
	config               *CartesiaConfig
	sessionConfig        *vbiface.SessionConfig
	pipelineOrchestrator *internal.PipelineOrchestrator
	httpClient           *http.Client
	metadata             map[string]any
	audioOutput          chan []byte
	id                   string
	state                vbiface.PipelineState
	persistenceStatus    vbiface.PersistenceStatus
	mu                   sync.RWMutex
	active               bool
}

// NewCartesiaSession creates a new Cartesia session.
func NewCartesiaSession(config *CartesiaConfig, sessionConfig *vbiface.SessionConfig, httpClient *http.Client) (*CartesiaSession, error) {
	sessionID := uuid.New().String()

	orchestrator := internal.NewPipelineOrchestrator(config.Config)

	return &CartesiaSession{
		id:                   sessionID,
		config:               config,
		sessionConfig:        sessionConfig,
		pipelineOrchestrator: orchestrator,
		httpClient:           httpClient,
		state:                vbiface.PipelineStateIdle,
		persistenceStatus:    vbiface.PersistenceStatusActive,
		metadata:             make(map[string]any),
		audioOutput:          make(chan []byte, 100),
		active:               false,
	}, nil
}

// GetID returns the session ID.
func (s *CartesiaSession) GetID() string {
	return s.id
}

// Start starts the voice session, connecting to Cartesia streaming session.
func (s *CartesiaSession) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		return backend.NewBackendError("Start", "session_already_active", nil)
	}

	s.active = true
	s.state = vbiface.PipelineStateListening

	// TODO: In a full implementation, this would:
	// 1. Connect to Cartesia streaming session via WebSocket
	// 2. Subscribe to user's audio stream
	// 3. Publish agent's audio stream
	// 4. Set up audio processing pipeline

	return nil
}

// Stop stops the voice session, disconnecting from Cartesia streaming session.
func (s *CartesiaSession) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return backend.NewBackendError("Stop", "session_not_active", nil)
	}

	s.active = false
	s.state = vbiface.PipelineStateIdle
	s.persistenceStatus = vbiface.PersistenceStatusCompleted

	// Close audio output channel
	close(s.audioOutput)

	// TODO: In a full implementation, this would:
	// 1. Unsubscribe from user audio stream
	// 2. Unpublish agent audio stream
	// 3. Close WebSocket connection
	// 4. End Cartesia streaming session via API

	return nil
}

// ProcessAudio processes incoming audio data through the pipeline orchestrator.
func (s *CartesiaSession) ProcessAudio(ctx context.Context, audio []byte) error {
	ctx, span := backend.StartSpan(ctx, "CartesiaSession.ProcessAudio", "cartesia")
	defer span.End()

	backend.AddSpanAttributes(span, map[string]any{
		"session_id": s.id,
		"audio_size": len(audio),
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
	s.state = vbiface.PipelineStateProcessing
	s.mu.Unlock()

	backend.AddSpanAttributes(span, map[string]any{
		"pipeline_state": string(s.state),
	})

	// Process through pipeline orchestrator with error recovery
	outputAudio, err := s.pipelineOrchestrator.ProcessAudio(ctx, audio, agentCallback, agentInstance)
	if err != nil {
		// Check if error is retryable
		if backend.IsRetryableError(err) {
			backend.LogWithOTELContext(ctx, slog.LevelWarn, "Retryable error in audio processing, attempting recovery", "error", err)

			// Retry once with a short delay
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				outputAudio, err = s.pipelineOrchestrator.ProcessAudio(ctx, audio, agentCallback, agentInstance)
				if err != nil {
					s.mu.Lock()
					s.state = vbiface.PipelineStateError
					s.mu.Unlock()
					backend.RecordSpanError(span, err)
					backend.LogWithOTELContext(ctx, slog.LevelError, "Failed to process audio after retry", "error", err)
					return err
				}
				backend.LogWithOTELContext(ctx, slog.LevelInfo, "Audio processing recovered from transient error")
			}
		} else {
			// Non-retryable error
			s.mu.Lock()
			s.state = vbiface.PipelineStateError
			s.mu.Unlock()
			backend.RecordSpanError(span, err)
			backend.LogWithOTELContext(ctx, slog.LevelError, "Failed to process audio", "error", err)
			return err
		}
	}

	backend.AddSpanAttributes(span, map[string]any{
		"output_audio_size": len(outputAudio),
	})

	// Send output audio with buffer overflow protection
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
	s.state = vbiface.PipelineStateListening
	s.mu.Unlock()

	span.SetStatus(codes.Ok, "audio processed successfully")
	backend.LogWithOTELContext(ctx, slog.LevelInfo, "Audio processed successfully",
		"input_size", len(audio), "output_size", len(outputAudio))

	return nil
}

// SendAudio sends audio data to the user via Cartesia WebSocket.
func (s *CartesiaSession) SendAudio(ctx context.Context, audio []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.active {
		return backend.NewBackendError("SendAudio", "session_not_active", nil)
	}

	// TODO: Implement WebSocket send to Cartesia streaming session
	// For now, send to audio output channel
	select {
	case s.audioOutput <- audio:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return backend.NewBackendError("SendAudio", backend.ErrCodeTimeout, nil)
	}
}

// ReceiveAudio receives audio data from the user via Cartesia WebSocket.
func (s *CartesiaSession) ReceiveAudio() <-chan []byte {
	return s.audioOutput
}

// SetAgentCallback sets the agent callback function.
func (s *CartesiaSession) SetAgentCallback(callback func(context.Context, string) (string, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionConfig.AgentCallback = callback
	return nil
}

// SetAgentInstance sets the agent instance.
func (s *CartesiaSession) SetAgentInstance(agent agentsiface.Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionConfig.AgentInstance = agent
	return nil
}

// GetState returns the current pipeline state.
func (s *CartesiaSession) GetState() vbiface.PipelineState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// GetPersistenceStatus returns the persistence status.
func (s *CartesiaSession) GetPersistenceStatus() vbiface.PersistenceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.persistenceStatus
}

// UpdateMetadata updates the session metadata.
func (s *CartesiaSession) UpdateMetadata(metadata map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.metadata == nil {
		s.metadata = make(map[string]any)
	}

	for k, v := range metadata {
		s.metadata[k] = v
	}

	return nil
}
