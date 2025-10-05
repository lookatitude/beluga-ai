// Package mocks provides simple mock utilities for schema testing.
// T018: Create custom mock utilities for complex test scenarios (simplified)
package mocks

import (
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/schema/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMessage is a simplified mock implementation for Message interface
type MockMessage struct {
	mock.Mock
}

func (m *MockMessage) GetType() iface.MessageType {
	args := m.Called()
	return args.Get(0).(iface.MessageType)
}

func (m *MockMessage) GetContent() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockMessage) ToolCalls() []iface.ToolCall {
	args := m.Called()
	return args.Get(0).([]iface.ToolCall)
}

func (m *MockMessage) AdditionalArgs() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

// MockChatHistory is a simplified mock implementation for ChatHistory interface
type MockChatHistory struct {
	mock.Mock
}

func (m *MockChatHistory) AddMessage(message iface.Message) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockChatHistory) AddUserMessage(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockChatHistory) AddAIMessage(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockChatHistory) Messages() ([]iface.Message, error) {
	args := m.Called()
	return args.Get(0).([]iface.Message), args.Error(1)
}

func (m *MockChatHistory) GetMessages() []iface.Message {
	args := m.Called()
	return args.Get(0).([]iface.Message)
}

func (m *MockChatHistory) GetLast() iface.Message {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(iface.Message)
}

func (m *MockChatHistory) Size() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockChatHistory) Clear() error {
	args := m.Called()
	return args.Error(0)
}

// CreateStandardMockMessage creates a mock message with standard test configuration
func CreateStandardMockMessage(messageType iface.MessageType, content string) *MockMessage {
	mockMsg := &MockMessage{}
	mockMsg.On("GetType").Return(messageType)
	mockMsg.On("GetContent").Return(content)
	mockMsg.On("ToolCalls").Return([]iface.ToolCall{})
	mockMsg.On("AdditionalArgs").Return(map[string]interface{}{})
	return mockMsg
}

// CreateStandardMockChatHistory creates a mock chat history with standard configuration
func CreateStandardMockChatHistory() *MockChatHistory {
	mockHistory := &MockChatHistory{}
	mockHistory.On("AddMessage", mock.Anything).Return(nil)
	mockHistory.On("AddUserMessage", mock.Anything).Return(nil)
	mockHistory.On("AddAIMessage", mock.Anything).Return(nil)
	mockHistory.On("Messages").Return([]iface.Message{}, nil)
	mockHistory.On("GetMessages").Return([]iface.Message{})
	mockHistory.On("GetLast").Return(nil)
	mockHistory.On("Clear").Return(nil)
	mockHistory.On("Size").Return(0)
	return mockHistory
}

// CreateMockConversation creates a complete mock conversation for testing
func CreateMockConversation(exchanges []struct{ Human, AI string }) (*MockChatHistory, []iface.Message) {
	messages := make([]iface.Message, 0, len(exchanges)*2)
	mockHistory := &MockChatHistory{}

	for _, exchange := range exchanges {
		// Human message
		humanMock := CreateStandardMockMessage(iface.RoleHuman, exchange.Human)
		messages = append(messages, humanMock)

		// AI message
		aiMock := CreateStandardMockMessage(iface.RoleAssistant, exchange.AI)
		messages = append(messages, aiMock)
	}

	// Configure mock history
	mockHistory.On("Messages").Return(messages, nil)
	mockHistory.On("GetMessages").Return(messages)
	mockHistory.On("Size").Return(len(messages))

	if len(messages) > 0 {
		mockHistory.On("GetLast").Return(messages[len(messages)-1])
	} else {
		mockHistory.On("GetLast").Return(nil)
	}

	for _, msg := range messages {
		mockHistory.On("AddMessage", msg).Return(nil)
	}

	mockHistory.On("Clear").Return(nil)

	return mockHistory, messages
}

// TestBasicMockUtilities provides basic tests for the mock utilities
func TestBasicMockUtilities(t *testing.T) {
	t.Run("StandardMockMessage", func(t *testing.T) {
		mock := CreateStandardMockMessage(iface.RoleHuman, "test content")
		assert.Equal(t, iface.RoleHuman, mock.GetType())
		assert.Equal(t, "test content", mock.GetContent())
		assert.Empty(t, mock.ToolCalls())
		mock.AssertExpectations(t)
	})

	t.Run("StandardMockChatHistory", func(t *testing.T) {
		mock := CreateStandardMockChatHistory()

		// Test basic operations
		err := mock.AddMessage(CreateStandardMockMessage(iface.RoleHuman, "test"))
		assert.NoError(t, err)

		messages, err := mock.Messages()
		assert.NoError(t, err)
		assert.NotNil(t, messages)

		size := mock.Size()
		assert.Equal(t, 0, size)

		mock.AssertExpectations(t)
	})

	t.Run("MockConversation", func(t *testing.T) {
		exchanges := []struct{ Human, AI string }{
			{"Hello", "Hi there!"},
			{"How are you?", "I'm doing well!"},
		}

		mockHistory, messages := CreateMockConversation(exchanges)
		assert.Len(t, messages, 4) // 2 exchanges = 4 messages

		retrievedMessages, err := mockHistory.Messages()
		assert.NoError(t, err)
		assert.Equal(t, messages, retrievedMessages)

		size := mockHistory.Size()
		assert.Equal(t, 4, size)

		mockHistory.AssertExpectations(t)
	})
}
