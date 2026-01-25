package mock

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/internal"
)

// MockSession implements the VoiceSession interface for testing.
type MockSession struct {
	config               *MockConfig
	sessionConfig        *vbiface.SessionConfig
	pipelineOrchestrator *internal.PipelineOrchestrator
	metadata             map[string]any
	audioOutput          chan []byte
	id                   string
	state                vbiface.PipelineState
	persistenceStatus    vbiface.PersistenceStatus
	mu                   sync.RWMutex
	active               bool
}

// NewMockSession creates a new mock session.
func NewMockSession(config *MockConfig, sessionConfig *vbiface.SessionConfig) (*MockSession, error) {
	sessionID := uuid.New().String()

	orchestrator := internal.NewPipelineOrchestrator(config.Config)

	return &MockSession{
		id:                   sessionID,
		config:               config,
		sessionConfig:        sessionConfig,
		pipelineOrchestrator: orchestrator,
		state:                vbiface.PipelineStateIdle,
		persistenceStatus:    vbiface.PersistenceStatusActive,
		metadata:             make(map[string]any),
		audioOutput:          make(chan []byte, 100),
		active:               false,
	}, nil
}

// Start starts the voice session.
func (s *MockSession) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		return backend.NewBackendError("Start", "session_already_active",
			nil)
	}

	s.active = true
	s.state = vbiface.PipelineStateListening
	return nil
}

// Stop stops the voice session.
func (s *MockSession) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return backend.NewBackendError("Stop", "session_not_active", nil)
	}

	s.active = false
	s.state = vbiface.PipelineStateIdle
	s.persistenceStatus = vbiface.PersistenceStatusCompleted
	close(s.audioOutput)
	return nil
}

// ProcessAudio processes incoming audio data through the pipeline.
func (s *MockSession) ProcessAudio(ctx context.Context, audio []byte) error {
	s.mu.RLock()
	active := s.active
	agentCallback := s.sessionConfig.AgentCallback
	agentInstance := s.sessionConfig.AgentInstance
	s.mu.RUnlock()

	if !active {
		return backend.NewBackendError("ProcessAudio", "session_not_active", nil)
	}

	// Update state to processing
	s.mu.Lock()
	s.state = vbiface.PipelineStateProcessing
	s.mu.Unlock()

	// Process through pipeline orchestrator
	outputAudio, err := s.pipelineOrchestrator.ProcessAudio(ctx, audio, agentCallback, agentInstance)
	if err != nil {
		s.mu.Lock()
		s.state = vbiface.PipelineStateError
		s.mu.Unlock()
		return err
	}

	// Send output audio
	select {
	case s.audioOutput <- outputAudio:
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return backend.NewBackendError("ProcessAudio", backend.ErrCodeTimeout,
			nil)
	}

	// Update state back to listening
	s.mu.Lock()
	s.state = vbiface.PipelineStateListening
	s.mu.Unlock()

	return nil
}

// SendAudio sends audio data to the user.
func (s *MockSession) SendAudio(ctx context.Context, audio []byte) error {
	s.mu.RLock()
	active := s.active
	s.mu.RUnlock()

	if !active {
		return backend.NewBackendError("SendAudio", "session_not_active", nil)
	}

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
func (s *MockSession) ReceiveAudio() <-chan []byte {
	// Mock implementation: return a channel that can be written to
	// In a real implementation, this would receive audio from the transport
	return make(chan []byte, 100)
}

// SetAgentCallback sets the agent callback function.
func (s *MockSession) SetAgentCallback(callback func(context.Context, string) (string, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessionConfig.AgentCallback = callback
	return nil
}

// SetAgentInstance sets the agent instance.
func (s *MockSession) SetAgentInstance(agent agentsiface.Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessionConfig.AgentInstance = agent
	return nil
}

// GetState returns the current pipeline state.
func (s *MockSession) GetState() vbiface.PipelineState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// GetPersistenceStatus returns the persistence status of the session.
func (s *MockSession) GetPersistenceStatus() vbiface.PersistenceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.persistenceStatus
}

// UpdateMetadata updates the session metadata.
func (s *MockSession) UpdateMetadata(metadata map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k, v := range metadata {
		s.metadata[k] = v
	}
	return nil
}

// GetID returns the session identifier.
func (s *MockSession) GetID() string {
	return s.id
}
