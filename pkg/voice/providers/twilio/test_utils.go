// Package twilio provides advanced test utilities and comprehensive mocks for testing Twilio voice implementations.
package twilio

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockTwilioVoice provides a comprehensive mock implementation for testing.
type AdvancedMockTwilioVoice struct {
	mock.Mock

	// Configuration
	name      string
	callCount int
	mu        sync.RWMutex

	// Configurable behavior
	shouldError   bool
	errorToReturn error
	simulateDelay time.Duration

	// Health check data
	healthState     string
	lastHealthCheck time.Time

	// Session data
	sessions map[string]vbiface.VoiceSession
}

// MockTwilioVoiceOption configures the behavior of AdvancedMockTwilioVoice.
type MockTwilioVoiceOption func(*AdvancedMockTwilioVoice)

// WithMockError configures the mock to return an error.
func WithMockError(shouldError bool, err error) MockTwilioVoiceOption {
	return func(m *AdvancedMockTwilioVoice) {
		m.shouldError = shouldError
		m.errorToReturn = err
	}
}

// WithMockDelay sets the delay for operations.
func WithMockDelay(delay time.Duration) MockTwilioVoiceOption {
	return func(m *AdvancedMockTwilioVoice) {
		m.simulateDelay = delay
	}
}

// WithHealthState sets the health check state.
func WithHealthState(state string) MockTwilioVoiceOption {
	return func(m *AdvancedMockTwilioVoice) {
		m.healthState = state
	}
}

// NewAdvancedMockTwilioVoice creates a new advanced mock with configurable behavior.
func NewAdvancedMockTwilioVoice(opts ...MockTwilioVoiceOption) *AdvancedMockTwilioVoice {
	m := &AdvancedMockTwilioVoice{
		name:            "advanced-mock-twilio",
		healthState:     "healthy",
		lastHealthCheck: time.Now(),
		simulateDelay:   10 * time.Millisecond,
		sessions:        make(map[string]vbiface.VoiceSession),
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Start implements the VoiceBackend interface.
func (m *AdvancedMockTwilioVoice) Start(ctx context.Context) error {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return m.errorToReturn
		}
		return errors.New("mock start error")
	}

	return nil
}

// Stop implements the VoiceBackend interface.
func (m *AdvancedMockTwilioVoice) Stop(ctx context.Context) error {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return m.errorToReturn
		}
		return errors.New("mock stop error")
	}

	return nil
}

// CreateSession implements the VoiceBackend interface.
func (m *AdvancedMockTwilioVoice) CreateSession(ctx context.Context, config *vbiface.SessionConfig) (vbiface.VoiceSession, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, errors.New("mock create session error")
	}

	// Create and return a mock session
	sessionID := fmt.Sprintf("mock-session-%d", m.callCount)
	mockSession := NewMockTwilioVoiceSession(sessionID, config)

	// Store session for GetSession to retrieve
	m.sessions[sessionID] = mockSession

	return mockSession, nil
}

// GetSession implements the VoiceBackend interface.
func (m *AdvancedMockTwilioVoice) GetSession(ctx context.Context, sessionID string) (vbiface.VoiceSession, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, errors.New("mock get session error")
	}

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, errors.New("session not found")
	}

	return session, nil
}

// ListSessions implements the VoiceBackend interface.
func (m *AdvancedMockTwilioVoice) ListSessions(ctx context.Context) ([]vbiface.VoiceSession, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, errors.New("mock list sessions error")
	}

	sessions := make([]vbiface.VoiceSession, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// CloseSession implements the VoiceBackend interface.
func (m *AdvancedMockTwilioVoice) CloseSession(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return m.errorToReturn
		}
		return errors.New("mock close session error")
	}

	delete(m.sessions, sessionID)
	return nil
}

// HealthCheck implements the VoiceBackend interface.
func (m *AdvancedMockTwilioVoice) HealthCheck(ctx context.Context) (*vbiface.HealthStatus, error) {
	m.mu.Lock()
	m.callCount++
	m.lastHealthCheck = time.Now()
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, errors.New("mock health check error")
	}

	return &vbiface.HealthStatus{
		Status:    m.healthState,
		LastCheck: m.lastHealthCheck,
		Details:   make(map[string]any),
	}, nil
}

// GetConnectionState implements the VoiceBackend interface.
func (m *AdvancedMockTwilioVoice) GetConnectionState() vbiface.ConnectionState {
	return vbiface.ConnectionStateConnected
}

// GetActiveSessionCount implements the VoiceBackend interface.
func (m *AdvancedMockTwilioVoice) GetActiveSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// GetConfig implements the VoiceBackend interface.
func (m *AdvancedMockTwilioVoice) GetConfig() *vbiface.Config {
	return &vbiface.Config{
		Provider: "twilio",
	}
}

// UpdateConfig implements the VoiceBackend interface.
func (m *AdvancedMockTwilioVoice) UpdateConfig(ctx context.Context, config *vbiface.Config) error {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.shouldError {
		if m.errorToReturn != nil {
			return m.errorToReturn
		}
		return errors.New("mock update config error")
	}

	return nil
}

// GetCallCount returns the number of method calls made.
func (m *AdvancedMockTwilioVoice) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// ConcurrentTestRunner provides utilities for concurrent testing.
type ConcurrentTestRunner struct {
	NumGoroutines int
	TestDuration  time.Duration
	TestFunc      func() error
}

// NewConcurrentTestRunner creates a new concurrent test runner.
func NewConcurrentTestRunner(numGoroutines int, duration time.Duration, testFunc func() error) *ConcurrentTestRunner {
	return &ConcurrentTestRunner{
		NumGoroutines: numGoroutines,
		TestDuration:  duration,
		TestFunc:      testFunc,
	}
}

// Run executes the concurrent test.
func (r *ConcurrentTestRunner) Run() error {
	var wg sync.WaitGroup
	errCh := make(chan error, r.NumGoroutines)
	done := make(chan struct{})

	// Start goroutines
	for i := 0; i < r.NumGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					if err := r.TestFunc(); err != nil {
						errCh <- err
					}
				}
			}
		}()
	}

	// Run for specified duration
	time.Sleep(r.TestDuration)
	close(done)
	wg.Wait()

	// Check for errors
	close(errCh)
	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

// MockTwilioVoiceSession provides a simple mock implementation of VoiceSession for testing.
type MockTwilioVoiceSession struct {
	id                string
	sessionConfig     *vbiface.SessionConfig
	state             vbiface.PipelineState
	persistenceStatus vbiface.PersistenceStatus
	metadata          map[string]any
	audioInput        chan []byte
	audioOutput       chan []byte
	mu                sync.RWMutex
	active            bool
}

// NewMockTwilioVoiceSession creates a new mock Twilio voice session.
func NewMockTwilioVoiceSession(sessionID string, config *vbiface.SessionConfig) *MockTwilioVoiceSession {
	return &MockTwilioVoiceSession{
		id:                sessionID,
		sessionConfig:     config,
		state:             vbiface.PipelineStateIdle,
		persistenceStatus: vbiface.PersistenceStatusActive,
		metadata:          make(map[string]any),
		audioInput:        make(chan []byte, 100),
		audioOutput:       make(chan []byte, 100),
		active:            false,
	}
}

// Start implements the VoiceSession interface.
func (s *MockTwilioVoiceSession) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active = true
	s.state = vbiface.PipelineStateListening
	return nil
}

// Stop implements the VoiceSession interface.
func (s *MockTwilioVoiceSession) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active = false
	s.state = vbiface.PipelineStateIdle
	close(s.audioInput)
	close(s.audioOutput)
	return nil
}

// ProcessAudio implements the VoiceSession interface.
func (s *MockTwilioVoiceSession) ProcessAudio(ctx context.Context, audio []byte) error {
	s.mu.RLock()
	active := s.active
	s.mu.RUnlock()
	if !active {
		return errors.New("session not active")
	}
	return nil
}

// SendAudio implements the VoiceSession interface.
func (s *MockTwilioVoiceSession) SendAudio(ctx context.Context, audio []byte) error {
	s.mu.RLock()
	active := s.active
	s.mu.RUnlock()
	if !active {
		return errors.New("session not active")
	}
	select {
	case s.audioOutput <- audio:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ReceiveAudio implements the VoiceSession interface.
func (s *MockTwilioVoiceSession) ReceiveAudio() <-chan []byte {
	return s.audioInput
}

// SetAgentCallback implements the VoiceSession interface.
func (s *MockTwilioVoiceSession) SetAgentCallback(callback func(context.Context, string) (string, error)) error {
	return nil
}

// SetAgentInstance implements the VoiceSession interface.
func (s *MockTwilioVoiceSession) SetAgentInstance(agent agentsiface.Agent) error {
	return nil
}

// GetState implements the VoiceSession interface.
func (s *MockTwilioVoiceSession) GetState() vbiface.PipelineState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// GetPersistenceStatus implements the VoiceSession interface.
func (s *MockTwilioVoiceSession) GetPersistenceStatus() vbiface.PersistenceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.persistenceStatus
}

// UpdateMetadata implements the VoiceSession interface.
func (s *MockTwilioVoiceSession) UpdateMetadata(metadata map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range metadata {
		s.metadata[k] = v
	}
	return nil
}

// GetID implements the VoiceSession interface.
func (s *MockTwilioVoiceSession) GetID() string {
	return s.id
}
