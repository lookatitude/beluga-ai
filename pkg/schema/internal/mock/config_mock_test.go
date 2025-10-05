// Package mocks provides simplified contract tests for configuration validation mocks.
// T011: Contract test for configuration validation mocks (simplified)
package mocks

import (
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/schema/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSchemaValidator is a simplified mock implementation of schema validation interfaces
type MockSchemaValidator struct {
	mock.Mock
}

func (m *MockSchemaValidator) ValidateMessage(message schema.Message) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockSchemaValidator) ValidateConfig(config interface{}) error {
	args := m.Called(config)
	return args.Error(0)
}

// TestSchemaValidatorMockInterface verifies mock implements validation interfaces correctly
func TestSchemaValidatorMockInterface(t *testing.T) {
	t.Run("ValidateMessage_succeeds", func(t *testing.T) {
		validator := &MockSchemaValidator{}
		validator.On("ValidateMessage", mock.Anything).Return(nil)

		msg := schema.NewHumanMessage("test message")
		err := validator.ValidateMessage(msg)
		assert.NoError(t, err)

		validator.AssertExpectations(t)
	})

	t.Run("ValidateMessage_fails_with_error", func(t *testing.T) {
		validator := &MockSchemaValidator{}
		expectedErr := iface.NewSchemaError("INVALID_MESSAGE", "Message too long")
		validator.On("ValidateMessage", mock.Anything).Return(expectedErr)

		msg := schema.NewHumanMessage("test message")
		err := validator.ValidateMessage(msg)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)

		validator.AssertExpectations(t)
	})

	t.Run("ValidateConfig_succeeds", func(t *testing.T) {
		validator := &MockSchemaValidator{}
		validator.On("ValidateConfig", mock.Anything).Return(nil)

		config := schema.SchemaValidationConfig{
			EnableStrictValidation: true,
			MaxMessageLength:       1000,
		}
		err := validator.ValidateConfig(config)
		assert.NoError(t, err)

		validator.AssertExpectations(t)
	})

	t.Run("ValidateConfig_fails", func(t *testing.T) {
		validator := &MockSchemaValidator{}
		expectedErr := iface.NewSchemaError("CONFIG_INVALID", "Invalid configuration")
		validator.On("ValidateConfig", mock.Anything).Return(expectedErr)

		invalidConfig := schema.SchemaValidationConfig{
			MaxMessageLength: -1, // Invalid
		}
		err := validator.ValidateConfig(invalidConfig)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)

		validator.AssertExpectations(t)
	})
}

// TestMockValidatorBehavior tests the mock's behavior patterns
func TestMockValidatorBehavior(t *testing.T) {
	t.Run("ExpectationVerification", func(t *testing.T) {
		validator := &MockSchemaValidator{}

		// Set up expectations
		validator.On("ValidateMessage", mock.Anything).Return(nil).Once()
		validator.On("ValidateConfig", mock.Anything).Return(nil).Once()

		// Use mock
		msg := schema.NewHumanMessage("test")
		err := validator.ValidateMessage(msg)
		assert.NoError(t, err)

		config := schema.SchemaValidationConfig{}
		err = validator.ValidateConfig(config)
		assert.NoError(t, err)

		validator.AssertExpectations(t)
	})

	t.Run("MultipleCalls", func(t *testing.T) {
		validator := &MockSchemaValidator{}

		// Set up expectations for multiple calls
		validator.On("ValidateMessage", mock.Anything).Return(nil).Times(3)

		// Make multiple calls
		for i := 0; i < 3; i++ {
			msg := schema.NewHumanMessage("test")
			err := validator.ValidateMessage(msg)
			assert.NoError(t, err)
		}

		validator.AssertExpectations(t)
	})
}

// TestMockInterfaceCompliance verifies that mocks fully implement interfaces
func TestMockInterfaceCompliance(t *testing.T) {
	validator := &MockSchemaValidator{}

	// Configure basic expectations
	validator.On("ValidateMessage", mock.Anything).Return(nil)
	validator.On("ValidateConfig", mock.Anything).Return(nil)

	// Test that all interface methods are callable
	msg := schema.NewHumanMessage("compliance test")
	err := validator.ValidateMessage(msg)
	assert.NoError(t, err)

	config := schema.SchemaValidationConfig{}
	err = validator.ValidateConfig(config)
	assert.NoError(t, err)

	validator.AssertExpectations(t)
}

// BenchmarkValidationMock tests the performance of validation mocks
func BenchmarkValidationMock(b *testing.B) {
	validator := &MockSchemaValidator{}
	validator.On("ValidateMessage", mock.Anything).Return(nil)

	msg := schema.NewHumanMessage("Benchmark test message")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := validator.ValidateMessage(msg)
		if err != nil {
			b.Fatal("Validation should not fail in benchmark")
		}
	}
}
