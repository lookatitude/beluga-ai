// Package integration provides integration tests for schema package mock consistency.
// T012: Integration test for generated mock consistency
package integration

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/schema/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema/internal/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockInterfaceConsistency verifies that all mocks properly implement their interfaces
func TestMockInterfaceConsistency(t *testing.T) {
	tests := []struct {
		name            string
		mockInstance    interface{}
		targetInterface reflect.Type
		description     string
	}{
		{
			name:            "MockMessage_implements_Message",
			mockInstance:    &mock.MockMessage{},
			targetInterface: reflect.TypeOf((*iface.Message)(nil)).Elem(),
			description:     "MockMessage must implement iface.Message interface",
		},
		{
			name:            "MockChatHistory_implements_ChatHistory",
			mockInstance:    &mock.MockChatHistory{},
			targetInterface: reflect.TypeOf((*iface.ChatHistory)(nil)).Elem(),
			description:     "MockChatHistory must implement iface.ChatHistory interface",
		},
		{
			name:            "MockSchemaValidator_implements_validation",
			mockInstance:    &mock.MockSchemaValidator{},
			targetInterface: reflect.TypeOf((*mock.MockSchemaValidator)(nil)).Elem(), // Self-reference for validation interface
			description:     "MockSchemaValidator must implement validation interface methods",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockType := reflect.TypeOf(tt.mockInstance)

			// Check if mock type implements the target interface
			implements := mockType.Implements(tt.targetInterface)
			assert.True(t, implements, "%s: %s", tt.description, tt.name)

			// Additional verification: check that all interface methods are present
			for i := 0; i < tt.targetInterface.NumMethod(); i++ {
				method := tt.targetInterface.Method(i)
				mockMethod, found := mockType.MethodByName(method.Name)

				assert.True(t, found, "Mock missing method: %s", method.Name)

				if found {
					// Verify method signature matches
					assert.Equal(t, method.Type.NumIn(), mockMethod.Type.NumIn(),
						"Method %s has wrong number of input parameters", method.Name)
					assert.Equal(t, method.Type.NumOut(), mockMethod.Type.NumOut(),
						"Method %s has wrong number of output parameters", method.Name)
				}
			}
		})
	}
}

// TestMockGenerationConsistency verifies mock generation produces consistent results
func TestMockGenerationConsistency(t *testing.T) {
	t.Run("MockMessage_consistency", func(t *testing.T) {
		// Create multiple mock instances
		mock1 := &mock.MockMessage{}
		mock2 := &mock.MockMessage{}

		// Configure same behavior
		mock1.On("GetContent").Return("test content")
		mock1.On("GetType").Return(iface.RoleHuman)

		mock2.On("GetContent").Return("test content")
		mock2.On("GetType").Return(iface.RoleHuman)

		// Both should behave identically
		assert.Equal(t, mock1.GetContent(), mock2.GetContent())
		assert.Equal(t, mock1.GetType(), mock2.GetType())

		mock1.AssertExpectations(t)
		mock2.AssertExpectations(t)
	})

	t.Run("MockChatHistory_consistency", func(t *testing.T) {
		// Test that multiple ChatHistory mocks behave consistently
		history1 := &mock.MockChatHistory{}
		history2 := &mock.MockChatHistory{}

		// Configure identical behavior
		history1.On("Size").Return(5)
		history2.On("Size").Return(5)

		// Both should return the same size
		assert.Equal(t, history1.Size(), history2.Size())

		history1.AssertExpectations(t)
		history2.AssertExpectations(t)
	})
}

// TestMockIntegrationWithRealTypes tests mocks work properly with real schema types
func TestMockIntegrationWithRealTypes(t *testing.T) {
	t.Run("MockMessage_with_real_ChatHistory", func(t *testing.T) {
		// Create real chat history and mock message
		history := schema.NewChatHistory()
		mockMsg := &mock.MockMessage{}

		// Configure mock message
		mockMsg.On("GetType").Return(iface.RoleHuman)
		mockMsg.On("GetContent").Return("Mock message in real history")
		mockMsg.On("ToolCalls").Return([]iface.ToolCall{})
		mockMsg.On("AdditionalArgs").Return(map[string]interface{}{})

		// Add mock message to real history
		err := history.AddMessage(mockMsg)
		require.NoError(t, err)

		// Verify integration
		messages := history.GetMessages()
		assert.Len(t, messages, 1)
		assert.Equal(t, "Mock message in real history", messages[0].GetContent())
		assert.Equal(t, iface.RoleHuman, messages[0].GetType())

		mockMsg.AssertExpectations(t)
	})

	t.Run("RealMessage_with_MockChatHistory", func(t *testing.T) {
		// Create real message and mock chat history
		realMsg := schema.NewAIMessage("Real message with mock history")
		mockHistory := &mock.MockChatHistory{}

		// Configure mock history to accept real message
		mockHistory.On("AddMessage", realMsg).Return(nil)
		mockHistory.On("GetMessages").Return([]iface.Message{realMsg})
		mockHistory.On("Size").Return(1)

		// Use real message with mock history
		err := mockHistory.AddMessage(realMsg)
		require.NoError(t, err)

		messages := mockHistory.GetMessages()
		assert.Len(t, messages, 1)

		size := mockHistory.Size()
		assert.Equal(t, 1, size)

		mockHistory.AssertExpectations(t)
	})
}

// TestMockErrorBehavior tests error handling in mocks
func TestMockErrorBehavior(t *testing.T) {
	t.Run("MockValidator_error_scenarios", func(t *testing.T) {
		validator := &mock.MockSchemaValidator{}

		// Configure different error scenarios
		validationErr := schema.NewValidationError("MOCK_VALIDATION_ERROR", "Mock validation failed")
		validator.On("ValidateMessage", mock.MatchedBy(func(msg schema.Message) bool {
			return msg.GetContent() == "invalid"
		}), mock.Anything).Return(validationErr)

		validator.On("ValidateMessage", mock.MatchedBy(func(msg schema.Message) bool {
			return msg.GetContent() != "invalid"
		}), mock.Anything).Return(nil)

		// Test valid message
		validMsg := schema.NewHumanMessage("valid message")
		err := validator.ValidateMessage(validMsg, schema.ValidationConfig{})
		assert.NoError(t, err)

		// Test invalid message
		invalidMsg := schema.NewHumanMessage("invalid")
		err = validator.ValidateMessage(invalidMsg, schema.ValidationConfig{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MOCK_VALIDATION_ERROR")

		validator.AssertExpectations(t)
	})
}

// TestCrossPackageIntegration tests schema mocks work with other framework packages
func TestCrossPackageIntegration(t *testing.T) {
	t.Run("schema_mocks_in_workflow", func(t *testing.T) {
		// This test simulates how schema mocks would be used in other packages

		// Create a mock message chain
		messages := []iface.Message{}

		for i := 0; i < 3; i++ {
			mockMsg := &mock.MockMessage{}
			mockMsg.On("GetContent").Return(fmt.Sprintf("Message %d", i+1))
			mockMsg.On("GetType").Return(iface.RoleHuman)
			mockMsg.On("ToolCalls").Return([]iface.ToolCall{})
			mockMsg.On("AdditionalArgs").Return(map[string]interface{}{})

			messages = append(messages, mockMsg)
		}

		// Simulate processing messages (like an LLM package might do)
		totalLength := 0
		for _, msg := range messages {
			content := msg.GetContent()
			totalLength += len(content)
			assert.NotEmpty(t, content)
			assert.NotEqual(t, iface.MessageType(""), msg.GetType())
		}

		assert.Greater(t, totalLength, 0)

		// Verify all mock expectations
		for _, msg := range messages {
			if mockMsg, ok := msg.(*mock.MockMessage); ok {
				mockMsg.AssertExpectations(t)
			}
		}
	})
}

// BenchmarkMockIntegrationOverhead measures performance overhead of using mocks
func BenchmarkMockIntegrationOverhead(b *testing.B) {
	// Compare real vs mock message performance
	b.Run("RealMessage", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			msg := schema.NewHumanMessage("benchmark message")
			_ = msg.GetContent()
			_ = msg.GetType()
		}
	})

	b.Run("MockMessage", func(b *testing.B) {
		mock := &mock.MockMessage{}
		mock.On("GetContent").Return("benchmark message")
		mock.On("GetType").Return(iface.RoleHuman)

		var msg iface.Message = mock

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = msg.GetContent()
			_ = msg.GetType()
		}
	})
}
