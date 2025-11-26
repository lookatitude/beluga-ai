package messagebus

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockMessageBus is a mock implementation of MessageBus for testing.
type MockMessageBus struct {
	mock.Mock
}

func (m *MockMessageBus) Publish(ctx context.Context, topic string, payload any, metadata map[string]any) error {
	args := m.Called(ctx, topic, payload, metadata)
	return args.Error(0)
}

func (m *MockMessageBus) Subscribe(ctx context.Context, topic string, handler HandlerFunc) (string, error) {
	args := m.Called(ctx, topic, handler)
	return args.String(0), args.Error(1)
}

func (m *MockMessageBus) Unsubscribe(ctx context.Context, topic, subscriberID string) error {
	args := m.Called(ctx, topic, subscriberID)
	return args.Error(0)
}

func (m *MockMessageBus) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMessageBus) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMessageBus) GetName() string {
	args := m.Called()
	return args.String(0)
}

func TestNewMessagingSystem(t *testing.T) {
	mockBus := &MockMessageBus{}
	system := NewMessagingSystem(mockBus)

	assert.NotNil(t, system)
	assert.Equal(t, mockBus, system.messageBus)
}

func TestMessagingSystem_SendMessage_Success(t *testing.T) {
	mockBus := &MockMessageBus{}
	system := NewMessagingSystem(mockBus)

	msg := Message{
		ID:       "test-id",
		Topic:    "test.topic",
		Payload:  "test payload",
		Metadata: map[string]any{"key": "value"},
	}

	mockBus.On("Publish", mock.Anything, "test.topic", "test payload", msg.Metadata).Return(nil)

	err := system.SendMessage(msg)

	require.NoError(t, err)
	mockBus.AssertExpectations(t)
}

func TestMessagingSystem_SendMessage_Error(t *testing.T) {
	mockBus := &MockMessageBus{}
	system := NewMessagingSystem(mockBus)

	msg := Message{
		ID:       "test-id",
		Topic:    "test.topic",
		Payload:  "test payload",
		Metadata: nil,
	}

	mockBus.On("Publish", mock.Anything, "test.topic", "test payload", mock.AnythingOfType("map[string]interface {}")).Return(assert.AnError)

	err := system.SendMessage(msg)

	require.Error(t, err)
	assert.Equal(t, assert.AnError, err)
	mockBus.AssertExpectations(t)
}

func TestMessagingSystem_SendMessageWithRetry_Success(t *testing.T) {
	mockBus := &MockMessageBus{}
	system := NewMessagingSystem(mockBus)

	msg := Message{
		ID:    "test-id",
		Topic: "test.topic",
	}

	// Mock successful publish on first attempt
	mockBus.On("Publish", mock.Anything, "test.topic", mock.Anything, mock.Anything).Return(nil)

	err := system.SendMessageWithRetry(msg, 3, 10*time.Millisecond)

	require.NoError(t, err)
	mockBus.AssertExpectations(t)
}

func TestMessagingSystem_SendMessageWithRetry_EventualSuccess(t *testing.T) {
	mockBus := &MockMessageBus{}
	system := NewMessagingSystem(mockBus)

	msg := Message{
		ID:    "test-id",
		Topic: "test.topic",
	}

	// Mock failures followed by success
	mockBus.On("Publish", mock.Anything, "test.topic", mock.Anything, mock.Anything).Return(assert.AnError).Times(2)
	mockBus.On("Publish", mock.Anything, "test.topic", mock.Anything, mock.Anything).Return(nil).Once()

	// Capture log output
	var logOutput bytes.Buffer
	log.SetOutput(&logOutput)

	err := system.SendMessageWithRetry(msg, 3, 1*time.Millisecond)

	require.NoError(t, err)

	// Verify retry logs
	output := logOutput.String()
	assert.Contains(t, output, "Failed to send message (attempt 1)")
	assert.Contains(t, output, "Failed to send message (attempt 2)")

	mockBus.AssertExpectations(t)
}

func TestMessagingSystem_SendMessageWithRetry_ExhaustRetries(t *testing.T) {
	mockBus := &MockMessageBus{}
	system := NewMessagingSystem(mockBus)

	msg := Message{
		ID:    "test-id",
		Topic: "test.topic",
	}

	// Mock all attempts fail
	mockBus.On("Publish", mock.Anything, "test.topic", mock.Anything, mock.Anything).Return(assert.AnError).Times(4) // 3 retries + 1 initial

	// Capture log output
	var logOutput bytes.Buffer
	log.SetOutput(&logOutput)

	err := system.SendMessageWithRetry(msg, 3, 1*time.Millisecond)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send message after 3 retries")

	// Verify retry logs
	output := logOutput.String()
	assert.Contains(t, output, "Failed to send message (attempt 1)")
	assert.Contains(t, output, "Failed to send message (attempt 2)")
	assert.Contains(t, output, "Failed to send message (attempt 3)")

	mockBus.AssertExpectations(t)
}

func TestMessagingSystem_SendMessageWithRetry_ExponentialBackoff(t *testing.T) {
	mockBus := &MockMessageBus{}
	system := NewMessagingSystem(mockBus)

	msg := Message{
		ID:    "test-id",
		Topic: "test.topic",
	}

	// Mock all attempts fail to test timing
	// With retries=3, we get 4 attempts total: initial + 3 retries
	// Backoff: 10ms (after attempt 1), 20ms (after attempt 2), 40ms (after attempt 3)
	// Total: 10ms + 20ms + 40ms = 70ms
	mockBus.On("Publish", mock.Anything, "test.topic", mock.Anything, mock.Anything).Return(assert.AnError).Times(4)

	start := time.Now()
	err := system.SendMessageWithRetry(msg, 3, 10*time.Millisecond)
	duration := time.Since(start)

	require.Error(t, err)
	// Should take at least 10ms + 20ms + 40ms = 70ms due to exponential backoff
	assert.GreaterOrEqual(t, duration, 70*time.Millisecond, "Duration should be at least 70ms, got %v", duration)

	mockBus.AssertExpectations(t)
}

func TestMessagingSystem_ReceiveMessage(t *testing.T) {
	mockBus := &MockMessageBus{}
	system := NewMessagingSystem(mockBus)

	msg, err := system.ReceiveMessage()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ReceiveMessage not implemented")
	assert.Equal(t, Message{}, msg)
}

func TestValidateMessage_Valid(t *testing.T) {
	msg := Message{
		ID:    "test-id",
		Topic: "test.topic",
	}

	err := ValidateMessage(msg)

	require.NoError(t, err)
}

func TestValidateMessage_MissingID(t *testing.T) {
	msg := Message{
		Topic: "test.topic",
	}

	err := ValidateMessage(msg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required fields")
}

func TestValidateMessage_MissingTopic(t *testing.T) {
	msg := Message{
		ID: "test-id",
	}

	err := ValidateMessage(msg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required fields")
}

func TestValidateMessage_MissingBoth(t *testing.T) {
	msg := Message{}

	err := ValidateMessage(msg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required fields")
}

func TestValidateMessage_WithPayloadAndMetadata(t *testing.T) {
	msg := Message{
		ID:       "test-id",
		Topic:    "test.topic",
		Payload:  "test payload",
		Metadata: map[string]any{"key": "value"},
	}

	err := ValidateMessage(msg)

	require.NoError(t, err)
}

func TestSerializeMessage_Success(t *testing.T) {
	msg := Message{
		ID:       "test-id",
		Topic:    "test.topic",
		Payload:  "test payload",
		Metadata: map[string]any{"key": "value", "number": 42},
	}

	serialized, err := SerializeMessage(msg)

	require.NoError(t, err)
	assert.NotEmpty(t, serialized)

	// Verify it's valid JSON
	var deserialized map[string]any
	err = json.Unmarshal([]byte(serialized), &deserialized)
	require.NoError(t, err)

	// Verify content
	assert.Equal(t, "test-id", deserialized["ID"])
	assert.Equal(t, "test.topic", deserialized["Topic"])
	assert.Equal(t, "test payload", deserialized["Payload"])
}

func TestSerializeMessage_ComplexPayload(t *testing.T) {
	complexPayload := map[string]any{
		"nested": map[string]any{
			"array":  []string{"item1", "item2"},
			"number": 123,
		},
		"simple": "value",
	}

	msg := Message{
		ID:      "complex-id",
		Topic:   "complex.topic",
		Payload: complexPayload,
	}

	serialized, err := SerializeMessage(msg)

	require.NoError(t, err)
	assert.NotEmpty(t, serialized)

	// Verify deserialization works
	deserialized, err := DeserializeMessage(serialized)
	require.NoError(t, err)
	assert.Equal(t, msg.ID, deserialized.ID)
	assert.Equal(t, msg.Topic, deserialized.Topic)
	// JSON unmarshaling converts numbers to float64, so we need to compare carefully
	deserializedPayload, ok := deserialized.Payload.(map[string]any)
	assert.True(t, ok, "Payload should be a map")
	nested, ok := deserializedPayload["nested"].(map[string]any)
	assert.True(t, ok, "Nested should be a map")
	// Compare number as float64 (JSON unmarshaling converts int to float64)
	assert.Equal(t, float64(123), nested["number"])
	assert.Equal(t, "value", deserializedPayload["simple"])
}

func TestSerializeMessage_InvalidPayload(t *testing.T) {
	// Create a payload that can't be JSON serialized
	type NonSerializable struct {
		Func func() // Functions can't be JSON serialized
	}

	msg := Message{
		ID:      "invalid-id",
		Topic:   "invalid.topic",
		Payload: NonSerializable{Func: func() {}},
	}

	serialized, err := SerializeMessage(msg)

	require.Error(t, err)
	assert.Empty(t, serialized)
	assert.Contains(t, err.Error(), "failed to serialize message")
}

func TestDeserializeMessage_Success(t *testing.T) {
	jsonStr := `{
		"ID": "test-id",
		"Topic": "test.topic",
		"Payload": "test payload",
		"Metadata": {"key": "value"}
	}`

	msg, err := DeserializeMessage(jsonStr)

	require.NoError(t, err)
	assert.Equal(t, "test-id", msg.ID)
	assert.Equal(t, "test.topic", msg.Topic)
	assert.Equal(t, "test payload", msg.Payload)

	if msg.Metadata != nil {
		assert.Equal(t, "value", msg.Metadata["key"])
	}
}

func TestDeserializeMessage_InvalidJSON(t *testing.T) {
	invalidJSON := `{"ID": "test", "invalid": json}`

	msg, err := DeserializeMessage(invalidJSON)

	require.Error(t, err)
	assert.Equal(t, Message{}, msg)
	assert.Contains(t, err.Error(), "failed to deserialize message")
}

func TestDeserializeMessage_MissingFields(t *testing.T) {
	jsonStr := `{"Payload": "test"}`

	msg, err := DeserializeMessage(jsonStr)

	require.NoError(t, err)
	assert.Empty(t, msg.ID)
	assert.Empty(t, msg.Topic)
	assert.Equal(t, "test", msg.Payload)
}

func TestMessagingSystem_Integration_SerializeDeserialize(t *testing.T) {
	originalMsg := Message{
		ID:       "integration-id",
		Topic:    "integration.topic",
		Payload:  map[string]any{"data": "integration test"},
		Metadata: map[string]any{"source": "test", "version": "1.0"},
	}

	// Serialize
	serialized, err := SerializeMessage(originalMsg)
	require.NoError(t, err)

	// Deserialize
	deserialized, err := DeserializeMessage(serialized)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, originalMsg.ID, deserialized.ID)
	assert.Equal(t, originalMsg.Topic, deserialized.Topic)
	assert.Equal(t, originalMsg.Payload, deserialized.Payload)
	assert.Equal(t, originalMsg.Metadata, deserialized.Metadata)
}

func TestMessagingSystem_EndToEnd(t *testing.T) {
	mockBus := &MockMessageBus{}
	system := NewMessagingSystem(mockBus)

	// Create and validate message
	msg := Message{
		ID:       "e2e-id",
		Topic:    "e2e.topic",
		Payload:  "end to end payload",
		Metadata: map[string]any{"test": "e2e"},
	}

	err := ValidateMessage(msg)
	require.NoError(t, err)

	// Serialize message
	serialized, err := SerializeMessage(msg)
	require.NoError(t, err)

	// Deserialize message
	deserialized, err := DeserializeMessage(serialized)
	require.NoError(t, err)

	// Mock successful publish
	mockBus.On("Publish", mock.Anything, "e2e.topic", "end to end payload", msg.Metadata).Return(nil)

	// Send the deserialized message
	err = system.SendMessage(deserialized)

	require.NoError(t, err)
	mockBus.AssertExpectations(t)
}

func TestMessagingSystem_RetryBackoffCalculation(t *testing.T) {
	mockBus := &MockMessageBus{}
	system := NewMessagingSystem(mockBus)

	msg := Message{ID: "retry-test", Topic: "retry.topic"}

	// Mock all attempts fail
	mockBus.On("Publish", mock.Anything, "retry.topic", mock.Anything, mock.Anything).Return(assert.AnError).Times(4)

	start := time.Now()
	err := system.SendMessageWithRetry(msg, 3, 5*time.Millisecond)
	duration := time.Since(start)

	require.Error(t, err)
	// Should take at least 5ms + 10ms + 20ms = 35ms due to exponential backoff
	assert.GreaterOrEqual(t, duration, 35*time.Millisecond, "Duration should be at least 35ms, got %v", duration)
}

func TestMessagingSystem_ZeroRetries(t *testing.T) {
	mockBus := &MockMessageBus{}
	system := NewMessagingSystem(mockBus)

	msg := Message{ID: "zero-retry", Topic: "zero.topic"}

	// Mock failure on first attempt
	mockBus.On("Publish", mock.Anything, "zero.topic", mock.Anything, mock.Anything).Return(assert.AnError).Once()

	err := system.SendMessageWithRetry(msg, 0, 1*time.Millisecond)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send message after 0 retries")
	mockBus.AssertExpectations(t)
}

func TestMessagingSystem_NegativeRetries(t *testing.T) {
	mockBus := &MockMessageBus{}
	system := NewMessagingSystem(mockBus)

	msg := Message{ID: "negative-retry", Topic: "negative.topic"}

	// Should still attempt at least once
	mockBus.On("Publish", mock.Anything, "negative.topic", mock.Anything, mock.Anything).Return(assert.AnError).Once()

	err := system.SendMessageWithRetry(msg, -1, 1*time.Millisecond)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send message after -1 retries")
	mockBus.AssertExpectations(t)
}

func TestValidateMessage_EdgeCases(t *testing.T) {
	testCases := []struct {
		message Message
		name    string
		valid   bool
	}{
		{
			name: "minimal valid message",
			message: Message{
				ID:    "id",
				Topic: "topic",
			},
			valid: true,
		},
		{
			name: "empty strings",
			message: Message{
				ID:    "",
				Topic: "",
			},
			valid: false,
		},
		{
			name: "whitespace only",
			message: Message{
				ID:    "   ",
				Topic: "   ",
			},
			valid: false,
		},
		{
			name: "very long strings",
			message: Message{
				ID:    strings.Repeat("a", 10000),
				Topic: strings.Repeat("b", 10000),
			},
			valid: true,
		},
		{
			name: "special characters",
			message: Message{
				ID:    "id:with:colons",
				Topic: "topic/with/slashes",
			},
			valid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateMessage(tc.message)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
