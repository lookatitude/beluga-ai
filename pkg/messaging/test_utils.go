// Package messaging provides advanced test utilities and comprehensive mocks for testing messaging implementations.
// This file contains utilities designed to support both unit tests and integration tests.
//
// Test Coverage Exclusions:
//
// The following code paths are intentionally excluded from 100% coverage requirements:
//
// 1. Panic Recovery Paths:
//    - Panic handlers in concurrent test runners (ConcurrentTestRunner)
//    - These paths are difficult to test without causing actual panics in test code
//
// 2. Context Cancellation Edge Cases:
//    - Some context cancellation paths in ReceiveMessages are difficult to reliably test
//    - Race conditions between context cancellation and channel operations
//
// 3. Error Paths Requiring System Conditions:
//    - Network errors that require actual network failures (provider implementations)
//    - File system errors that require specific OS conditions
//    - Memory exhaustion scenarios
//
// 4. Provider-Specific Untestable Paths:
//    - Provider implementations in pkg/messaging/providers/* require external service failures
//    - These are tested through integration tests rather than unit tests
//    - Provider registry initialization code (init() functions)
//
// 5. Test Utility Functions:
//    - Helper functions in test_utils.go that are used by tests but not directly tested
//    - These are validated through their usage in actual test cases
//
// 6. Initialization Code:
//    - Package init() functions and global variable initialization
//    - Registry registration code that executes automatically
//
// 7. OTEL Context Logging:
//    - logWithOTELContext function has paths that require valid OTEL context
//    - Some edge cases in trace/span ID extraction are difficult to test in isolation
//
// All exclusions are documented here to maintain transparency about coverage goals.
// The target is 100% coverage of testable code paths, excluding the above categories.
package messaging

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/messaging/iface"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockMessaging provides a comprehensive mock implementation for testing.
type AdvancedMockMessaging struct {
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
	conversations map[string]*iface.Conversation
	messages      map[string][]*iface.Message
	participants  map[string][]*iface.Participant
}

// MockMessagingOption configures the behavior of AdvancedMockMessaging.
type MockMessagingOption func(*AdvancedMockMessaging)

// WithMockError configures the mock to return an error.
func WithMockError(shouldError bool, err error) MockMessagingOption {
	return func(m *AdvancedMockMessaging) {
		m.shouldError = shouldError
		m.errorToReturn = err
	}
}

// WithMockDelay sets the delay for operations.
func WithMockDelay(delay time.Duration) MockMessagingOption {
	return func(m *AdvancedMockMessaging) {
		m.simulateDelay = delay
	}
}

// WithHealthState sets the health check state.
func WithHealthState(state string) MockMessagingOption {
	return func(m *AdvancedMockMessaging) {
		m.healthState = state
	}
}

// WithErrorCode configures the mock to return a MessagingError with a specific error code.
// This is a convenience function for creating common error scenarios.
func WithErrorCode(op, code string) MockMessagingOption {
	return func(m *AdvancedMockMessaging) {
		m.shouldError = true
		m.errorToReturn = NewMessagingError(op, code, errors.New("mock error"))
	}
}

// WithRateLimitError configures the mock to return a rate limit error.
func WithRateLimitError(op string) MockMessagingOption {
	return WithErrorCode(op, ErrCodeRateLimit)
}

// WithTimeoutError configures the mock to return a timeout error.
func WithTimeoutError(op string) MockMessagingOption {
	return WithErrorCode(op, ErrCodeTimeout)
}

// WithNetworkError configures the mock to return a network error.
func WithNetworkError(op string) MockMessagingOption {
	return WithErrorCode(op, ErrCodeNetworkError)
}

// WithInvalidConfigError configures the mock to return an invalid config error.
func WithInvalidConfigError(op string) MockMessagingOption {
	return WithErrorCode(op, ErrCodeInvalidConfig)
}

// WithNotFoundError configures the mock to return a not found error.
func WithNotFoundError(op string) MockMessagingOption {
	return WithErrorCode(op, ErrCodeNotFound)
}

// WithConversationNotFoundError configures the mock to return a conversation not found error.
func WithConversationNotFoundError(op string) MockMessagingOption {
	return WithErrorCode(op, ErrCodeConversationNotFound)
}

// WithInternalError configures the mock to return an internal error.
func WithInternalError(op string) MockMessagingOption {
	return WithErrorCode(op, ErrCodeInternalError)
}

// NewAdvancedMockMessaging creates a new advanced mock with configurable behavior.
func NewAdvancedMockMessaging(opts ...MockMessagingOption) *AdvancedMockMessaging {
	m := &AdvancedMockMessaging{
		name:            "advanced-mock",
		healthState:     "healthy",
		lastHealthCheck: time.Now(),
		simulateDelay:   10 * time.Millisecond,
		conversations:   make(map[string]*iface.Conversation),
		messages:        make(map[string][]*iface.Message),
		participants:    make(map[string][]*iface.Participant),
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Start implements the ConversationalBackend interface.
func (m *AdvancedMockMessaging) Start(ctx context.Context) error {
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

// Stop implements the ConversationalBackend interface.
func (m *AdvancedMockMessaging) Stop(ctx context.Context) error {
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

// CreateConversation implements the ConversationalBackend interface.
func (m *AdvancedMockMessaging) CreateConversation(ctx context.Context, config *iface.ConversationConfig) (*iface.Conversation, error) {
	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, errors.New("mock create conversation error")
	}

	conv := &iface.Conversation{
		ConversationSID: "CH" + time.Now().Format("20060102150405"),
		FriendlyName:    config.FriendlyName,
		State:           iface.ConversationStateActive,
		DateCreated:     time.Now(),
		DateUpdated:     time.Now(),
	}

	m.mu.Lock()
	m.callCount++
	m.conversations[conv.ConversationSID] = conv
	m.mu.Unlock()
	return conv, nil
}

// GetConversation implements the ConversationalBackend interface.
func (m *AdvancedMockMessaging) GetConversation(ctx context.Context, conversationID string) (*iface.Conversation, error) {
	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, errors.New("mock get conversation error")
	}

	m.mu.Lock()
	m.callCount++
	conv, exists := m.conversations[conversationID]
	m.mu.Unlock()

	if !exists {
		return nil, errors.New("conversation not found")
	}

	return conv, nil
}

// ListConversations implements the ConversationalBackend interface.
func (m *AdvancedMockMessaging) ListConversations(ctx context.Context) ([]*iface.Conversation, error) {
	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, errors.New("mock list conversations error")
	}

	m.mu.Lock()
	m.callCount++
	conversations := make([]*iface.Conversation, 0, len(m.conversations))
	for _, conv := range m.conversations {
		conversations = append(conversations, conv)
	}
	m.mu.Unlock()

	return conversations, nil
}

// CloseConversation implements the ConversationalBackend interface.
func (m *AdvancedMockMessaging) CloseConversation(ctx context.Context, conversationID string) error {
	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return m.errorToReturn
		}
		return errors.New("mock close conversation error")
	}

	m.mu.Lock()
	m.callCount++
	conv, exists := m.conversations[conversationID]
	if !exists {
		m.mu.Unlock()
		return errors.New("conversation not found")
	}

	conv.State = iface.ConversationStateClosed
	conv.DateUpdated = time.Now()
	m.mu.Unlock()
	return nil
}

// SendMessage implements the ConversationalBackend interface.
func (m *AdvancedMockMessaging) SendMessage(ctx context.Context, conversationID string, message *iface.Message) error {
	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return m.errorToReturn
		}
		return errors.New("mock send message error")
	}

	m.mu.Lock()
	m.callCount++
	if _, exists := m.conversations[conversationID]; !exists {
		m.mu.Unlock()
		return errors.New("conversation not found")
	}

	m.messages[conversationID] = append(m.messages[conversationID], message)
	m.mu.Unlock()
	return nil
}

// ReceiveMessages implements the ConversationalBackend interface.
func (m *AdvancedMockMessaging) ReceiveMessages(ctx context.Context, conversationID string) (<-chan *iface.Message, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, errors.New("mock receive messages error")
	}

	ch := make(chan *iface.Message, 10)
	return ch, nil
}

// AddParticipant implements the ConversationalBackend interface.
func (m *AdvancedMockMessaging) AddParticipant(ctx context.Context, conversationID string, participant *iface.Participant) error {
	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return m.errorToReturn
		}
		return errors.New("mock add participant error")
	}

	m.mu.Lock()
	m.callCount++
	if _, exists := m.conversations[conversationID]; !exists {
		m.mu.Unlock()
		return errors.New("conversation not found")
	}

	m.participants[conversationID] = append(m.participants[conversationID], participant)
	m.mu.Unlock()
	return nil
}

// RemoveParticipant implements the ConversationalBackend interface.
func (m *AdvancedMockMessaging) RemoveParticipant(ctx context.Context, conversationID string, participantID string) error {
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
		return errors.New("mock remove participant error")
	}

	return nil
}

// HandleWebhook implements the ConversationalBackend interface.
func (m *AdvancedMockMessaging) HandleWebhook(ctx context.Context, event *iface.WebhookEvent) error {
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
		return errors.New("mock handle webhook error")
	}

	return nil
}

// HealthCheck implements the ConversationalBackend interface.
func (m *AdvancedMockMessaging) HealthCheck(ctx context.Context) (*iface.HealthStatus, error) {
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

	status := iface.HealthStatus(m.healthState)
	return &status, nil
}

// GetConfig implements the ConversationalBackend interface.
func (m *AdvancedMockMessaging) GetConfig() interface{} {
	return DefaultConfig()
}

// GetCallCount returns the number of method calls made.
func (m *AdvancedMockMessaging) GetCallCount() int {
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
