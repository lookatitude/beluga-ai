package twilio

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/messaging"
	"github.com/lookatitude/beluga-ai/pkg/messaging/iface"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockTwilioProvider provides a comprehensive mock implementation for testing Twilio provider.
type AdvancedMockTwilioProvider struct {
	mock.Mock
	mu                sync.RWMutex
	callCount         int
	shouldError       bool
	errorToReturn     error
	conversations     map[string]*iface.Conversation
	messages          map[string][]*iface.Message
	participants      map[string][]*iface.Participant
	simulateDelay     time.Duration
	simulateRateLimit bool
	rateLimitCount    int
	healthState       iface.HealthStatus
	config            *TwilioConfig
}

// NewAdvancedMockTwilioProvider creates a new advanced mock with configurable behavior.
func NewAdvancedMockTwilioProvider(config *TwilioConfig) *AdvancedMockTwilioProvider {
	if config == nil {
		config = &TwilioConfig{}
	}

	mock := &AdvancedMockTwilioProvider{
		config:        config,
		conversations: make(map[string]*iface.Conversation),
		messages:      make(map[string][]*iface.Message),
		participants:  make(map[string][]*iface.Participant),
		healthState:   iface.HealthStatusHealthy,
	}
	return mock
}

// Start implements the ConversationalBackend interface.
func (m *AdvancedMockTwilioProvider) Start(ctx context.Context) error {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if shouldError {
		if errorToReturn != nil {
			return errorToReturn
		}
		return messaging.NewMessagingError("Start", messaging.ErrCodeAuthentication, errors.New("mock start error"))
	}

	return nil
}

// Stop implements the ConversationalBackend interface.
func (m *AdvancedMockTwilioProvider) Stop(ctx context.Context) error {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if shouldError {
		if errorToReturn != nil {
			return errorToReturn
		}
		return messaging.NewMessagingError("Stop", messaging.ErrCodeInternalError, errors.New("mock stop error"))
	}

	return nil
}

// CreateConversation implements the ConversationalBackend interface.
func (m *AdvancedMockTwilioProvider) CreateConversation(ctx context.Context, config *iface.ConversationConfig) (*iface.Conversation, error) {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if shouldError {
		if errorToReturn != nil {
			return nil, errorToReturn
		}
		return nil, messaging.NewMessagingError("CreateConversation", messaging.ErrCodeInternalError, errors.New("mock create conversation error"))
	}

	m.mu.Lock()
	conv := &iface.Conversation{
		ConversationSID: "CH" + time.Now().Format("20060102150405"),
		FriendlyName:    config.FriendlyName,
		State:           iface.ConversationStateActive,
		DateCreated:     time.Now(),
		DateUpdated:     time.Now(),
	}
	m.conversations[conv.ConversationSID] = conv
	m.mu.Unlock()

	return conv, nil
}

// GetConversation implements the ConversationalBackend interface.
func (m *AdvancedMockTwilioProvider) GetConversation(ctx context.Context, conversationID string) (*iface.Conversation, error) {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if shouldError {
		if errorToReturn != nil {
			return nil, errorToReturn
		}
		return nil, messaging.NewMessagingError("GetConversation", messaging.ErrCodeNotFound, errors.New("conversation not found"))
	}

	m.mu.RLock()
	conv, exists := m.conversations[conversationID]
	m.mu.RUnlock()

	if !exists {
		return nil, messaging.NewMessagingError("GetConversation", messaging.ErrCodeNotFound, errors.New("conversation not found"))
	}

	return conv, nil
}

// ListConversations implements the ConversationalBackend interface.
func (m *AdvancedMockTwilioProvider) ListConversations(ctx context.Context) ([]*iface.Conversation, error) {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if shouldError {
		if errorToReturn != nil {
			return nil, errorToReturn
		}
		return nil, messaging.NewMessagingError("ListConversations", messaging.ErrCodeInternalError, errors.New("mock list conversations error"))
	}

	m.mu.RLock()
	conversations := make([]*iface.Conversation, 0, len(m.conversations))
	for _, conv := range m.conversations {
		conversations = append(conversations, conv)
	}
	m.mu.RUnlock()

	return conversations, nil
}

// CloseConversation implements the ConversationalBackend interface.
func (m *AdvancedMockTwilioProvider) CloseConversation(ctx context.Context, conversationID string) error {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if shouldError {
		if errorToReturn != nil {
			return errorToReturn
		}
		return messaging.NewMessagingError("CloseConversation", messaging.ErrCodeInternalError, errors.New("mock close conversation error"))
	}

	m.mu.Lock()
	conv, exists := m.conversations[conversationID]
	if !exists {
		m.mu.Unlock()
		return messaging.NewMessagingError("CloseConversation", messaging.ErrCodeNotFound, errors.New("conversation not found"))
	}
	conv.State = iface.ConversationStateClosed
	conv.DateUpdated = time.Now()
	m.mu.Unlock()

	return nil
}

// SendMessage implements the ConversationalBackend interface.
func (m *AdvancedMockTwilioProvider) SendMessage(ctx context.Context, conversationID string, message *iface.Message) error {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if shouldError {
		if errorToReturn != nil {
			return errorToReturn
		}
		return messaging.NewMessagingError("SendMessage", messaging.ErrCodeInternalError, errors.New("mock send message error"))
	}

	m.mu.Lock()
	if _, exists := m.conversations[conversationID]; !exists {
		m.mu.Unlock()
		return messaging.NewMessagingError("SendMessage", messaging.ErrCodeNotFound, errors.New("conversation not found"))
	}
	m.messages[conversationID] = append(m.messages[conversationID], message)
	m.mu.Unlock()

	return nil
}

// ReceiveMessages implements the ConversationalBackend interface.
func (m *AdvancedMockTwilioProvider) ReceiveMessages(ctx context.Context, conversationID string) (<-chan *iface.Message, error) {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.mu.Unlock()

	if shouldError {
		if errorToReturn != nil {
			return nil, errorToReturn
		}
		return nil, messaging.NewMessagingError("ReceiveMessages", messaging.ErrCodeInternalError, errors.New("mock receive messages error"))
	}

	ch := make(chan *iface.Message, 10)
	return ch, nil
}

// AddParticipant implements the ConversationalBackend interface.
func (m *AdvancedMockTwilioProvider) AddParticipant(ctx context.Context, conversationID string, participant *iface.Participant) error {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if shouldError {
		if errorToReturn != nil {
			return errorToReturn
		}
		return messaging.NewMessagingError("AddParticipant", messaging.ErrCodeInternalError, errors.New("mock add participant error"))
	}

	m.mu.Lock()
	if _, exists := m.conversations[conversationID]; !exists {
		m.mu.Unlock()
		return messaging.NewMessagingError("AddParticipant", messaging.ErrCodeNotFound, errors.New("conversation not found"))
	}
	m.participants[conversationID] = append(m.participants[conversationID], participant)
	m.mu.Unlock()

	return nil
}

// RemoveParticipant implements the ConversationalBackend interface.
func (m *AdvancedMockTwilioProvider) RemoveParticipant(ctx context.Context, conversationID string, participantID string) error {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if shouldError {
		if errorToReturn != nil {
			return errorToReturn
		}
		return messaging.NewMessagingError("RemoveParticipant", messaging.ErrCodeInternalError, errors.New("mock remove participant error"))
	}

	return nil
}

// HandleWebhook implements the ConversationalBackend interface.
func (m *AdvancedMockTwilioProvider) HandleWebhook(ctx context.Context, event *iface.WebhookEvent) error {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if shouldError {
		if errorToReturn != nil {
			return errorToReturn
		}
		return messaging.NewMessagingError("HandleWebhook", messaging.ErrCodeInternalError, errors.New("mock handle webhook error"))
	}

	return nil
}

// HealthCheck implements the ConversationalBackend interface.
func (m *AdvancedMockTwilioProvider) HealthCheck(ctx context.Context) (*iface.HealthStatus, error) {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	healthState := m.healthState
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if shouldError {
		if errorToReturn != nil {
			return nil, errorToReturn
		}
		return nil, messaging.NewMessagingError("HealthCheck", messaging.ErrCodeInternalError, errors.New("mock health check error"))
	}

	return &healthState, nil
}

// GetConfig implements the ConversationalBackend interface.
func (m *AdvancedMockTwilioProvider) GetConfig() interface{} {
	return m.config
}

// SetError configures the mock to return an error.
func (m *AdvancedMockTwilioProvider) SetError(shouldError bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = shouldError
	m.errorToReturn = err
}

// SetDelay configures the mock to simulate delay.
func (m *AdvancedMockTwilioProvider) SetDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateDelay = delay
}

// SetRateLimit configures the mock to simulate rate limiting.
func (m *AdvancedMockTwilioProvider) SetRateLimit(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateRateLimit = enabled
}

// SetHealthState configures the mock health state.
func (m *AdvancedMockTwilioProvider) SetHealthState(state iface.HealthStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthState = state
}

// GetCallCount returns the number of times methods have been called.
func (m *AdvancedMockTwilioProvider) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// Reset resets the mock state.
func (m *AdvancedMockTwilioProvider) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = 0
	m.shouldError = false
	m.errorToReturn = nil
	m.conversations = make(map[string]*iface.Conversation)
	m.messages = make(map[string][]*iface.Message)
	m.participants = make(map[string][]*iface.Participant)
	m.rateLimitCount = 0
	m.simulateRateLimit = false
	m.simulateDelay = 0
	m.healthState = iface.HealthStatusHealthy
}

// Ensure AdvancedMockTwilioProvider implements the interface.
var (
	_ iface.ConversationalBackend = (*AdvancedMockTwilioProvider)(nil)
)
