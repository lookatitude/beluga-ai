// Package session provides advanced test utilities and comprehensive mocks for testing Session implementations.
package session

import (
	"context"
	"errors"
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

// Agent Instance Test Utilities
// These utilities support the AgentInstance integration that will be defined in Phase 3.4.

// AgentState represents the state of an agent in a voice session.
// This type matches the contract definition for agent state.
type AgentState string

const (
	AgentStateIdle        AgentState = "idle"
	AgentStateListening   AgentState = "listening"
	AgentStateProcessing  AgentState = "processing"
	AgentStateStreaming   AgentState = "streaming"
	AgentStateExecuting   AgentState = "executing_tool"
	AgentStateSpeaking    AgentState = "speaking"
	AgentStateInterrupted AgentState = "interrupted"
)

// AgentStreamChunk represents a chunk of agent execution output.
// This matches the definition from pkg/agents for consistency.
type AgentStreamChunk struct {
	Action    any
	Finish    any
	Err       error
	Metadata  map[string]any
	Content   string
	ToolCalls []any
}

// AgentContext represents the agent-specific context within a voice session.
// This type matches the contract definition from data-model.md.
type AgentContext struct {
	LastInterruption    time.Time
	ConversationHistory []any
	ToolResults         []any
	CurrentPlan         []any
	StreamingActive     bool
}

// StreamingState represents the current streaming state.
type StreamingState struct {
	LastChunkTime time.Time
	CurrentStream <-chan AgentStreamChunk
	Buffer        []AgentStreamChunk
	Active        bool
	Interrupted   bool
}

// AdvancedMockAgentInstance provides a comprehensive mock implementation for agent instances.
type AdvancedMockAgentInstance struct {
	agent             any
	config            any
	context           *AgentContext
	state             AgentState
	streamChunks      []AgentStreamChunk
	interruptionCount int
	mu                sync.RWMutex
	streamingActive   bool
}

// MockAgentInstanceOption defines functional options for configuring agent instance mocks.
type MockAgentInstanceOption func(*AdvancedMockAgentInstance)

// WithAgentState sets the initial agent state.
func WithAgentState(state AgentState) MockAgentInstanceOption {
	return func(a *AdvancedMockAgentInstance) {
		a.state = state
	}
}

// WithAgentContext sets the agent context.
func WithAgentContext(ctx *AgentContext) MockAgentInstanceOption {
	return func(a *AdvancedMockAgentInstance) {
		a.context = ctx
	}
}

// WithStreamingChunks sets predefined chunks to stream.
func WithStreamingChunks(chunks []AgentStreamChunk) MockAgentInstanceOption {
	return func(a *AdvancedMockAgentInstance) {
		a.streamChunks = chunks
	}
}

// NewAdvancedMockAgentInstance creates a new advanced agent instance mock.
func NewAdvancedMockAgentInstance(agent, config any, options ...MockAgentInstanceOption) *AdvancedMockAgentInstance {
	mock := &AdvancedMockAgentInstance{
		agent:  agent,
		config: config,
		state:  AgentStateIdle,
		context: &AgentContext{
			ConversationHistory: make([]any, 0),
			ToolResults:         make([]any, 0),
			CurrentPlan:         make([]any, 0),
			StreamingActive:     false,
		},
		streamChunks: make([]AgentStreamChunk, 0),
	}

	// Apply options
	for _, opt := range options {
		opt(mock)
	}

	return mock
}

// GetState returns the current agent state.
func (a *AdvancedMockAgentInstance) GetState() AgentState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state
}

// SetState sets the agent state.
func (a *AdvancedMockAgentInstance) SetState(state AgentState) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state = state
}

// GetContext returns the agent context.
func (a *AdvancedMockAgentInstance) GetContext() *AgentContext {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.context
}

// IsStreamingActive returns whether streaming is currently active.
func (a *AdvancedMockAgentInstance) IsStreamingActive() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.streamingActive
}

// SetStreamingActive sets the streaming active flag.
func (a *AdvancedMockAgentInstance) SetStreamingActive(active bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.streamingActive = active
}

// IncrementInterruptionCount increments the interruption counter.
func (a *AdvancedMockAgentInstance) IncrementInterruptionCount() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.interruptionCount++
}

// GetInterruptionCount returns the number of interruptions.
func (a *AdvancedMockAgentInstance) GetInterruptionCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.interruptionCount
}

// MockStreamingAgentIntegration provides a mock for testing agent integration in voice sessions.
type MockStreamingAgentIntegration struct {
	agentInstance *AdvancedMockAgentInstance
	mu            sync.RWMutex
	started       bool
	stopped       bool
}

// NewMockStreamingAgentIntegration creates a new mock streaming agent integration.
func NewMockStreamingAgentIntegration(agentInstance *AdvancedMockAgentInstance) *MockStreamingAgentIntegration {
	return &MockStreamingAgentIntegration{
		agentInstance: agentInstance,
		started:       false,
		stopped:       false,
	}
}

// Start initializes the agent integration.
func (m *MockStreamingAgentIntegration) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		return errors.New("agent integration already started")
	}
	m.started = true
	if m.agentInstance != nil {
		m.agentInstance.SetState(AgentStateIdle)
	}
	return nil
}

// Stop stops the agent integration.
func (m *MockStreamingAgentIntegration) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started {
		return errors.New("agent integration not started")
	}
	m.stopped = true
	if m.agentInstance != nil {
		m.agentInstance.SetState(AgentStateIdle)
		m.agentInstance.SetStreamingActive(false)
	}
	return nil
}

// IsStarted returns whether the integration is started.
func (m *MockStreamingAgentIntegration) IsStarted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.started && !m.stopped
}

// GetAgentInstance returns the agent instance.
func (m *MockStreamingAgentIntegration) GetAgentInstance() *AdvancedMockAgentInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.agentInstance
}

// Voice-Agent Integration Test Helpers

// CreateTestAgentContext creates a test agent context.
func CreateTestAgentContext() *AgentContext {
	return &AgentContext{
		ConversationHistory: make([]any, 0),
		ToolResults:         make([]any, 0),
		CurrentPlan:         make([]any, 0),
		StreamingActive:     false,
		LastInterruption:    time.Time{},
	}
}

// CreateTestStreamingState creates a test streaming state.
func CreateTestStreamingState() *StreamingState {
	return &StreamingState{
		Active:        false,
		CurrentStream: nil,
		Buffer:        make([]AgentStreamChunk, 0),
		LastChunkTime: time.Time{},
		Interrupted:   false,
	}
}

// CreateTestSessionWithAgent creates a test session setup helper for agent integration.
func CreateTestSessionWithAgent(sessionID string, agentInstance *AdvancedMockAgentInstance) (*AdvancedMockSession, *MockStreamingAgentIntegration) {
	session := NewAdvancedMockSession(sessionID)
	integration := NewMockStreamingAgentIntegration(agentInstance)
	return session, integration
}

// AssertAgentState validates agent state transitions.
func AssertAgentState(t *testing.T, instance *AdvancedMockAgentInstance, expectedState AgentState) {
	t.Helper()
	assert.Equal(t, expectedState, instance.GetState(), "Agent state should match expected state")
}

// AssertStreamingActive validates streaming active state.
func AssertStreamingActive(t *testing.T, instance *AdvancedMockAgentInstance, expectedActive bool) {
	t.Helper()
	assert.Equal(t, expectedActive, instance.IsStreamingActive(), "Streaming active state should match expected value")
}
