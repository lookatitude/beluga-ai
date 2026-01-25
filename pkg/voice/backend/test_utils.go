package backend

import (
	"context"
	"fmt"
	"sync"
	"time"

	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// MockOption is a functional option for configuring mock behavior (T249).
type MockOption func(*MockConfig)

// MockConfig holds configuration for advanced mock behavior.
type MockConfig struct {
	CallCounts        map[string]int
	ErrorCode         string
	AudioData         []byte
	AudioResponses    [][]byte
	ErrorDelay        time.Duration
	OperationDelay    time.Duration
	ProcessingDelay   time.Duration
	MaxSessions       int
	SessionDelay      time.Duration
	CallCountsMu      sync.RWMutex
	ShouldError       bool
	AutoStartSessions bool
}

// WithMockError configures the mock to return an error (T249).
func WithMockError(errorCode string) MockOption {
	return func(c *MockConfig) {
		c.ShouldError = true
		c.ErrorCode = errorCode
	}
}

// WithMockDelay configures the mock to add a delay to operations (T249).
func WithMockDelay(delay time.Duration) MockOption {
	return func(c *MockConfig) {
		c.OperationDelay = delay
	}
}

// WithAudioData configures the mock with audio data responses (T249).
func WithAudioData(audio []byte) MockOption {
	return func(c *MockConfig) {
		c.AudioData = audio
	}
}

// WithAudioResponses configures the mock with multiple audio responses (T249).
func WithAudioResponses(responses [][]byte) MockOption {
	return func(c *MockConfig) {
		c.AudioResponses = responses
	}
}

// WithProcessingDelay configures the mock with processing delay (T249).
func WithProcessingDelay(delay time.Duration) MockOption {
	return func(c *MockConfig) {
		c.ProcessingDelay = delay
	}
}

// WithMaxSessions configures the mock with maximum session limit (T249).
func WithMaxSessions(max int) MockOption {
	return func(c *MockConfig) {
		c.MaxSessions = max
	}
}

// WithAutoStartSessions configures the mock to auto-start sessions (T249).
func WithAutoStartSessions(autoStart bool) MockOption {
	return func(c *MockConfig) {
		c.AutoStartSessions = autoStart
	}
}

// AdvancedMockVoiceBackend provides an advanced mock implementation of VoiceBackend (T247, T248).
// It supports configurable behavior for testing various scenarios.
type AdvancedMockVoiceBackend struct {
	config          *MockConfig
	sessions        map[string]iface.VoiceSession
	healthStatus    *iface.HealthStatus
	connectionState iface.ConnectionState
	mu              sync.RWMutex
}

// NewAdvancedMockVoiceBackend creates a new advanced mock voice backend (T248).
func NewAdvancedMockVoiceBackend(opts ...MockOption) *AdvancedMockVoiceBackend {
	mockConfig := &MockConfig{
		CallCounts: make(map[string]int),
	}

	for _, opt := range opts {
		opt(mockConfig)
	}

	baseConfig := iface.Config{
		Provider:              "mock",
		MaxConcurrentSessions: mockConfig.MaxSessions,
	}
	if baseConfig.MaxConcurrentSessions == 0 {
		baseConfig.MaxConcurrentSessions = 100 // Default
	}

	return &AdvancedMockVoiceBackend{
		config:          mockConfig,
		sessions:        make(map[string]iface.VoiceSession),
		connectionState: iface.ConnectionStateDisconnected,
		healthStatus: &iface.HealthStatus{
			Status:    "healthy",
			Details:   make(map[string]any),
			LastCheck: time.Now(),
		},
	}
}

// incrementCallCount increments the call count for an operation.
func (b *AdvancedMockVoiceBackend) incrementCallCount(operation string) {
	b.config.CallCountsMu.Lock()
	defer b.config.CallCountsMu.Unlock()
	b.config.CallCounts[operation]++
}

// GetCallCount returns the call count for an operation.
func (b *AdvancedMockVoiceBackend) GetCallCount(operation string) int {
	b.config.CallCountsMu.RLock()
	defer b.config.CallCountsMu.RUnlock()
	return b.config.CallCounts[operation]
}

// Start starts the mock backend.
func (b *AdvancedMockVoiceBackend) Start(ctx context.Context) error {
	b.incrementCallCount("Start")

	if b.config.OperationDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(b.config.OperationDelay):
		}
	}

	if b.config.ShouldError && b.config.ErrorCode == "start_error" {
		return NewBackendError("Start", ErrCodeConnectionFailed, nil)
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.connectionState = iface.ConnectionStateConnected
	return nil
}

// Stop stops the mock backend.
func (b *AdvancedMockVoiceBackend) Stop(ctx context.Context) error {
	b.incrementCallCount("Stop")

	if b.config.OperationDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(b.config.OperationDelay):
		}
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.connectionState = iface.ConnectionStateDisconnected
	return nil
}

// CreateSession creates a new mock session.
func (b *AdvancedMockVoiceBackend) CreateSession(ctx context.Context, config *iface.SessionConfig) (iface.VoiceSession, error) {
	b.incrementCallCount("CreateSession")

	if b.config.OperationDelay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(b.config.OperationDelay):
		}
	}

	if b.config.ShouldError && b.config.ErrorCode == "create_session_error" {
		return nil, NewBackendError("CreateSession", b.config.ErrorCode, nil)
	}

	// Check session limit
	b.mu.RLock()
	currentCount := len(b.sessions)
	b.mu.RUnlock()
	if b.config.MaxSessions > 0 && currentCount >= b.config.MaxSessions {
		return nil, NewBackendError("CreateSession", ErrCodeSessionLimitExceeded, nil)
	}

	session := NewAdvancedMockVoiceSession(b.config, config)

	b.mu.Lock()
	b.sessions[session.GetID()] = session
	b.mu.Unlock()

	if b.config.AutoStartSessions {
		_ = session.Start(ctx)
	}

	return session, nil
}

// GetSession retrieves a session by ID.
func (b *AdvancedMockVoiceBackend) GetSession(ctx context.Context, sessionID string) (iface.VoiceSession, error) {
	b.incrementCallCount("GetSession")
	b.mu.RLock()
	defer b.mu.RUnlock()
	session, ok := b.sessions[sessionID]
	if !ok {
		return nil, NewBackendError("GetSession", ErrCodeSessionNotFound, nil)
	}
	return session, nil
}

// ListSessions returns all active sessions.
func (b *AdvancedMockVoiceBackend) ListSessions(ctx context.Context) ([]iface.VoiceSession, error) {
	b.incrementCallCount("ListSessions")
	b.mu.RLock()
	defer b.mu.RUnlock()
	sessions := make([]iface.VoiceSession, 0, len(b.sessions))
	for _, session := range b.sessions {
		sessions = append(sessions, session)
	}
	return sessions, nil
}

// CloseSession closes a session.
func (b *AdvancedMockVoiceBackend) CloseSession(ctx context.Context, sessionID string) error {
	b.incrementCallCount("CloseSession")
	b.mu.Lock()
	defer b.mu.Unlock()
	session, ok := b.sessions[sessionID]
	if !ok {
		return NewBackendError("CloseSession", ErrCodeSessionNotFound, nil)
	}
	_ = session.Stop(ctx)
	delete(b.sessions, sessionID)
	return nil
}

// HealthCheck returns the health status.
func (b *AdvancedMockVoiceBackend) HealthCheck(ctx context.Context) (*iface.HealthStatus, error) {
	b.incrementCallCount("HealthCheck")
	b.mu.Lock()
	b.healthStatus.LastCheck = time.Now()
	// Return a copy to avoid returning pointer to protected data
	status := *b.healthStatus
	b.mu.Unlock()
	return &status, nil
}

// GetConnectionState returns the connection state.
func (b *AdvancedMockVoiceBackend) GetConnectionState() iface.ConnectionState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connectionState
}

// GetActiveSessionCount returns the active session count.
func (b *AdvancedMockVoiceBackend) GetActiveSessionCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	count := 0
	for _, session := range b.sessions {
		if session.GetState() != iface.PipelineStateIdle {
			count++
		}
	}
	return count
}

// GetConfig returns the backend configuration.
func (b *AdvancedMockVoiceBackend) GetConfig() *iface.Config {
	return &iface.Config{
		Provider: "mock",
	}
}

// UpdateConfig updates the backend configuration.
func (b *AdvancedMockVoiceBackend) UpdateConfig(ctx context.Context, config *iface.Config) error {
	b.incrementCallCount("UpdateConfig")
	return nil
}

// AdvancedMockVoiceSession provides an advanced mock implementation of VoiceSession (T250).
type AdvancedMockVoiceSession struct {
	config            *MockConfig
	sessionConfig     *iface.SessionConfig
	metadata          map[string]any
	audioOutput       chan []byte
	callCounts        map[string]int
	id                string
	state             iface.PipelineState
	persistenceStatus iface.PersistenceStatus
	mu                sync.RWMutex
	callCountsMu      sync.RWMutex
	active            bool
}

// NewAdvancedMockVoiceSession creates a new advanced mock session (T250).
func NewAdvancedMockVoiceSession(mockConfig *MockConfig, sessionConfig *iface.SessionConfig) *AdvancedMockVoiceSession {
	return &AdvancedMockVoiceSession{
		id:                fmt.Sprintf("mock-session-%d", time.Now().UnixNano()),
		config:            mockConfig,
		sessionConfig:     sessionConfig,
		state:             iface.PipelineStateIdle,
		persistenceStatus: iface.PersistenceStatusActive,
		metadata:          make(map[string]any),
		audioOutput:       make(chan []byte, 100),
		active:            false,
		callCounts:        make(map[string]int),
	}
}

// incrementCallCount increments the call count for an operation.
func (s *AdvancedMockVoiceSession) incrementCallCount(operation string) {
	s.callCountsMu.Lock()
	defer s.callCountsMu.Unlock()
	s.callCounts[operation]++
}

// GetCallCount returns the call count for an operation.
func (s *AdvancedMockVoiceSession) GetCallCount(operation string) int {
	s.callCountsMu.RLock()
	defer s.callCountsMu.RUnlock()
	return s.callCounts[operation]
}

// Start starts the session.
func (s *AdvancedMockVoiceSession) Start(ctx context.Context) error {
	s.incrementCallCount("Start")

	if s.config.OperationDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.config.OperationDelay):
		}
	}

	if s.config.ShouldError && s.config.ErrorCode == "start_error" {
		return NewBackendError("Start", s.config.ErrorCode, nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.active = true
	s.state = iface.PipelineStateListening
	return nil
}

// Stop stops the session.
func (s *AdvancedMockVoiceSession) Stop(ctx context.Context) error {
	s.incrementCallCount("Stop")

	if s.config.OperationDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.config.OperationDelay):
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.active = false
	s.state = iface.PipelineStateIdle
	s.persistenceStatus = iface.PersistenceStatusCompleted
	close(s.audioOutput)
	return nil
}

// ProcessAudio processes audio data.
func (s *AdvancedMockVoiceSession) ProcessAudio(ctx context.Context, audio []byte) error {
	s.incrementCallCount("ProcessAudio")

	if s.config.ProcessingDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.config.ProcessingDelay):
		}
	}

	if s.config.ShouldError && s.config.ErrorCode == "process_audio_error" {
		return NewBackendError("ProcessAudio", s.config.ErrorCode, nil)
	}

	s.mu.RLock()
	active := s.active
	s.mu.RUnlock()

	if !active {
		return NewBackendError("ProcessAudio", "session_not_active", nil)
	}

	// Use configured audio data or generate default
	outputAudio := s.config.AudioData
	if len(s.config.AudioResponses) > 0 {
		callCount := s.GetCallCount("ProcessAudio")
		if callCount-1 < len(s.config.AudioResponses) {
			outputAudio = s.config.AudioResponses[callCount-1]
		}
	}
	if len(outputAudio) == 0 {
		outputAudio = audio // Echo input if no configured response
	}

	select {
	case s.audioOutput <- outputAudio:
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return NewBackendError("ProcessAudio", ErrCodeTimeout, nil)
	}

	return nil
}

// SendAudio sends audio data.
func (s *AdvancedMockVoiceSession) SendAudio(ctx context.Context, audio []byte) error {
	s.incrementCallCount("SendAudio")

	if s.config.OperationDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.config.OperationDelay):
		}
	}

	s.mu.RLock()
	active := s.active
	s.mu.RUnlock()

	if !active {
		return NewBackendError("SendAudio", "session_not_active", nil)
	}

	select {
	case s.audioOutput <- audio:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return NewBackendError("SendAudio", ErrCodeTimeout, nil)
	}
}

// ReceiveAudio returns the audio output channel.
func (s *AdvancedMockVoiceSession) ReceiveAudio() <-chan []byte {
	return s.audioOutput
}

// SetAgentCallback sets the agent callback.
func (s *AdvancedMockVoiceSession) SetAgentCallback(callback func(context.Context, string) (string, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionConfig.AgentCallback = callback
	return nil
}

// SetAgentInstance sets the agent instance.
func (s *AdvancedMockVoiceSession) SetAgentInstance(agent agentsiface.Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionConfig.AgentInstance = agent
	return nil
}

// GetID returns the session ID.
func (s *AdvancedMockVoiceSession) GetID() string {
	return s.id
}

// GetState returns the current state.
func (s *AdvancedMockVoiceSession) GetState() iface.PipelineState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// GetPersistenceStatus returns the persistence status.
func (s *AdvancedMockVoiceSession) GetPersistenceStatus() iface.PersistenceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.persistenceStatus
}

// GetMetadata returns the session metadata.
func (s *AdvancedMockVoiceSession) GetMetadata() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copied := make(map[string]any, len(s.metadata))
	for k, v := range s.metadata {
		copied[k] = v
	}
	return copied
}

// UpdateMetadata updates the session metadata.
func (s *AdvancedMockVoiceSession) UpdateMetadata(metadata map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range metadata {
		s.metadata[k] = v
	}
	return nil
}

// ConcurrentTestRunner provides utilities for concurrent testing (T251).
type ConcurrentTestRunner struct {
	workers    int
	iterations int
	timeout    time.Duration
}

// NewConcurrentTestRunner creates a new concurrent test runner (T251).
func NewConcurrentTestRunner(workers, iterations int, timeout time.Duration) *ConcurrentTestRunner {
	return &ConcurrentTestRunner{
		workers:    workers,
		iterations: iterations,
		timeout:    timeout,
	}
}

// Run executes a function concurrently across multiple workers.
func (r *ConcurrentTestRunner) Run(ctx context.Context, fn func(ctx context.Context, workerID, iteration int) error) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	var wg sync.WaitGroup
	errChan := make(chan error, r.workers*r.iterations)

	for w := 0; w < r.workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < r.iterations; i++ {
				if err := fn(ctx, workerID, i); err != nil {
					select {
					case errChan <- err:
					default:
					}
				}
			}
		}(w)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return errors[0] // Return first error for simplicity
	}

	return nil
}

// RunLoadTest runs a load test with configurable parameters (T252).
func RunLoadTest(ctx context.Context, backend iface.VoiceBackend, numSessions, operationsPerSession int, timeout time.Duration) error {
	runner := NewConcurrentTestRunner(numSessions, operationsPerSession, timeout)
	return runner.Run(ctx, func(ctx context.Context, workerID, iteration int) error {
		sessionConfig := &iface.SessionConfig{
			UserID:       "load-test-user",
			Transport:    "websocket",
			PipelineType: iface.PipelineTypeSTTTTS,
		}

		session, err := backend.CreateSession(ctx, sessionConfig)
		if err != nil {
			return err
		}

		if err := session.Start(ctx); err != nil {
			return err
		}

		// Simulate audio processing
		audio := []byte{1, 2, 3, 4, 5}
		if err := session.ProcessAudio(ctx, audio); err != nil {
			return err
		}

		return session.Stop(ctx)
	})
}
