// Package session provides advanced test utilities and comprehensive mocks for testing Session implementations.
package session

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockSession provides a comprehensive mock implementation for testing.
type AdvancedMockSession struct {
	errorToReturn error
	mock.Mock
	sessionID            string
	callCount            int
	processingDelay      time.Duration
	mu                   sync.RWMutex
	active               bool
	started              bool
	stopped              bool
	shouldError          bool
	simulateNetworkDelay bool
}

// NewAdvancedMockSession creates a new advanced mock with configurable behavior.
func NewAdvancedMockSession(sessionID string, opts ...MockOption) *AdvancedMockSession {
	m := &AdvancedMockSession{
		sessionID:       sessionID,
		processingDelay: 10 * time.Millisecond,
		active:          false,
		started:         false,
		stopped:         false,
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// MockOption configures the behavior of AdvancedMockSession.
type MockOption func(*AdvancedMockSession)

// WithMockSessionID sets the session ID for the mock.
func WithMockSessionID(id string) MockOption {
	return func(m *AdvancedMockSession) {
		m.sessionID = id
	}
}

// WithActive sets the active state.
func WithActive(active bool) MockOption {
	return func(m *AdvancedMockSession) {
		m.active = active
	}
}

// WithStarted sets the started state.
func WithStarted(started bool) MockOption {
	return func(m *AdvancedMockSession) {
		m.started = started
	}
}

// WithError configures the mock to return an error.
func WithError(err error) MockOption {
	return func(m *AdvancedMockSession) {
		m.shouldError = true
		m.errorToReturn = err
	}
}

// WithProcessingDelay sets the delay for processing.
func WithProcessingDelay(delay time.Duration) MockOption {
	return func(m *AdvancedMockSession) {
		m.processingDelay = delay
	}
}

// WithNetworkDelay enables network delay simulation.
func WithNetworkDelay(enabled bool) MockOption {
	return func(m *AdvancedMockSession) {
		m.simulateNetworkDelay = enabled
	}
}

// Start implements the Session interface.
func (m *AdvancedMockSession) Start(ctx context.Context) error {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	// Check if mock expectations are set up
	if m.ExpectedCalls != nil && len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx)
		return args.Error(0)
	}

	// Check if session is already active
	m.mu.RLock()
	active := m.active
	m.mu.RUnlock()

	if active {
		return &SessionError{
			Op:   "Start",
			Code: ErrCodeSessionAlreadyActive,
			Err:  nil,
		}
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return m.errorToReturn
		}
		return &SessionError{
			Op:   "Start",
			Code: ErrCodeInternalError,
			Err:  nil,
		}
	}

	// Simulate processing delay
	if m.processingDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.processingDelay):
		}
	}

	m.mu.Lock()
	m.started = true
	m.active = true
	m.mu.Unlock()

	return nil
}

// Stop implements the Session interface.
func (m *AdvancedMockSession) Stop(ctx context.Context) error {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	// Check if mock expectations are set up
	if m.ExpectedCalls != nil && len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx)
		return args.Error(0)
	}

	// Check if session is active
	m.mu.RLock()
	active := m.active
	m.mu.RUnlock()

	if !active {
		return &SessionError{
			Op:   "Stop",
			Code: ErrCodeSessionNotActive,
			Err:  nil,
		}
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return m.errorToReturn
		}
		return &SessionError{
			Op:   "Stop",
			Code: ErrCodeInternalError,
			Err:  nil,
		}
	}

	m.mu.Lock()
	m.stopped = true
	m.active = false
	m.mu.Unlock()

	return nil
}

// IsActive implements the Session interface.
func (m *AdvancedMockSession) IsActive() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active
}

// GetSessionID implements the VoiceSession interface.
func (m *AdvancedMockSession) GetSessionID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessionID
}

// GetState implements the VoiceSession interface.
func (m *AdvancedMockSession) GetState() iface.SessionState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.stopped {
		return iface.SessionStateEnded
	}
	if m.active {
		return iface.SessionStateListening
	}
	return iface.SessionStateInitial
}

// Say implements the VoiceSession interface.
func (m *AdvancedMockSession) Say(ctx context.Context, text string) (iface.SayHandle, error) {
	// Placeholder implementation
	return nil, nil
}

// SayWithOptions implements the VoiceSession interface.
func (m *AdvancedMockSession) SayWithOptions(ctx context.Context, text string, options iface.SayOptions) (iface.SayHandle, error) {
	// Placeholder implementation
	return nil, nil
}

// ProcessAudio implements the VoiceSession interface.
func (m *AdvancedMockSession) ProcessAudio(ctx context.Context, audio []byte) error {
	// Placeholder implementation
	return nil
}

// OnStateChanged implements the VoiceSession interface.
func (m *AdvancedMockSession) OnStateChanged(callback func(iface.SessionState)) {
	// Placeholder implementation
}

// GetCallCount returns the number of times methods have been called.
func (m *AdvancedMockSession) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// AssertSessionInterface ensures that a type implements the VoiceSession interface.
func AssertSessionInterface(t *testing.T, s iface.VoiceSession) {
	t.Helper()
	assert.NotNil(t, s, "VoiceSession should not be nil")

	// Test GetSessionID method
	id := s.GetSessionID()
	assert.NotEmpty(t, id, "Session ID should not be empty")

	// Test GetState method
	_ = s.GetState()

	// Test Start method
	ctx := context.Background()
	err := s.Start(ctx)
	// We don't care about the result, just that the method exists and can be called
	_ = err
}
