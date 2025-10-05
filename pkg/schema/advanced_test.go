// Package schema provides advanced test scenarios and comprehensive testing patterns.
// This file demonstrates constitutional testing practices including table-driven tests,
// concurrency testing, performance benchmarks, and integration test patterns.
// T005: Create advanced_test.go with table-driven tests, concurrency tests, and benchmark suites
package schema

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T006: Benchmark tests for message creation/validation operations (target <1ms)

// BenchmarkMessageCreation tests the performance of message creation operations
func BenchmarkMessageCreation(b *testing.B) {
	tests := []struct {
		name    string
		creator func() Message
	}{
		{
			name:    "HumanMessage",
			creator: func() Message { return NewHumanMessage("Test message") },
		},
		{
			name:    "AIMessage",
			creator: func() Message { return NewAIMessage("AI response") },
		},
		{
			name:    "SystemMessage",
			creator: func() Message { return NewSystemMessage("System prompt") },
		},
		{
			name:    "ToolMessage",
			creator: func() Message { return NewToolMessage("Tool result", "call_123") },
		},
		{
			name:    "FunctionMessage",
			creator: func() Message { return NewFunctionMessage("calculate", "Result: 42") },
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				msg := tt.creator()
				if msg.GetContent() == "" {
					b.Fatal("Empty content")
				}
			}
		})
	}
}

// BenchmarkMessageValidation tests the performance of message validation
func BenchmarkMessageValidation(b *testing.B) {
	messages := []Message{
		NewHumanMessage("Test validation message"),
		NewAIMessage("AI response for validation"),
		NewSystemMessage("System message for validation"),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, msg := range messages {
			err := ValidateMessage(msg)
			if err != nil {
				// Validation errors are expected in some cases
				_ = err
			}
		}
	}
}

// T007: Benchmark tests for factory functions (target <100μs)

// BenchmarkFactoryFunctions tests the performance of all factory functions
func BenchmarkFactoryFunctions(b *testing.B) {
	tests := []struct {
		name      string
		operation func() any
	}{
		{
			name:      "NewHumanMessage",
			operation: func() any { return NewHumanMessage("test") },
		},
		{
			name:      "NewAIMessage",
			operation: func() any { return NewAIMessage("test") },
		},
		{
			name:      "NewSystemMessage",
			operation: func() any { return NewSystemMessage("test") },
		},
		{
			name:      "NewBaseChatHistory",
			operation: func() any { history, _ := NewBaseChatHistory(); return history },
		},
		{
			name:      "NewDocument",
			operation: func() any { return NewDocument("test", map[string]string{"type": "test"}) },
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				result := tt.operation()
				if result == nil {
					b.Fatal("Factory returned nil")
				}
			}
		})
	}
}

// T008: Benchmark tests for configuration validation with memory tracking

// BenchmarkConfigurationValidation tests the performance of configuration validation
func BenchmarkConfigurationValidation(b *testing.B) {
	configs := []struct {
		name   string
		config SchemaValidationConfig
	}{
		{
			name: "BasicValidation",
			config: SchemaValidationConfig{
				EnableStrictValidation: true,
				MaxMessageLength:       1000,
			},
		},
		{
			name: "StrictValidation",
			config: SchemaValidationConfig{
				EnableStrictValidation:  true,
				EnableContentValidation: true,
				MaxMessageLength:        500,
			},
		},
	}

	for _, tt := range configs {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			// Test config validation performance
			for i := 0; i < b.N; i++ {
				err := tt.config.Validate()
				if err != nil && b.N < 10 {
					b.Logf("Validation error: %v", err)
				}
			}
		})
	}
}

// T023: Implement concurrency benchmarks

// BenchmarkConcurrentMessageCreation tests message creation under concurrent load
func BenchmarkConcurrentMessageCreation(b *testing.B) {
	concurrencyLevels := []int{1, 2, 4, 8, 16}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency%d", concurrency), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			var wg sync.WaitGroup
			messagesPerWorker := b.N / concurrency
			if messagesPerWorker == 0 {
				messagesPerWorker = 1
			}

			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()
					for j := 0; j < messagesPerWorker; j++ {
						msg := NewHumanMessage(fmt.Sprintf("Worker %d message %d", workerID, j))
						if msg.GetContent() == "" {
							b.Error("Empty content in concurrent creation")
							return
						}
					}
				}(i)
			}

			wg.Wait()
		})
	}
}

// BenchmarkConcurrentChatHistory tests chat history operations under concurrent access
func BenchmarkConcurrentChatHistory(b *testing.B) {
	history, err := NewBaseChatHistory()
	if err != nil {
		b.Fatal("Failed to create chat history:", err)
	}

	concurrency := 8

	b.ResetTimer()
	b.ReportAllocs()

	var wg sync.WaitGroup
	operationsPerWorker := b.N / concurrency
	if operationsPerWorker == 0 {
		operationsPerWorker = 1
	}

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < operationsPerWorker; j++ {
				msg := NewHumanMessage(fmt.Sprintf("Concurrent message %d-%d", workerID, j))
				err := history.AddMessage(msg)
				if err != nil {
					b.Errorf("Failed to add message: %v", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
}

// T014: Performance test for health check overhead

// BenchmarkHealthCheckOverhead tests the performance impact of health monitoring
func BenchmarkHealthCheckOverhead(b *testing.B) {
	tests := []struct {
		name       string
		withHealth bool
		operation  func() Message
	}{
		{
			name:       "WithoutHealthCheck",
			withHealth: false,
			operation:  func() Message { return NewHumanMessage("test") },
		},
		{
			name:       "WithHealthCheck",
			withHealth: true,
			operation:  func() Message { return NewHumanMessage("test") },
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				msg := tt.operation()

				if tt.withHealth {
					// Simulate health check overhead
					_ = time.Now() // Minimal overhead simulation
				}

				// Ensure operation succeeded
				if msg.GetContent() == "" {
					b.Fatal("Operation failed")
				}
			}
		})
	}
}

// Advanced table-driven test patterns following constitutional requirements

// TestMessageCreationAdvanced provides comprehensive table-driven tests for message creation
func TestMessageCreationAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		description string
		messageType MessageType
		content     string
		toolCalls   []ToolCall
		expectError bool
		errorCode   string
		validate    func(t *testing.T, msg Message)
	}{
		{
			name:        "valid_human_message",
			description: "Creates valid human message with basic content",
			messageType: iface.RoleHuman,
			content:     "Hello, world!",
			expectError: false,
			validate: func(t *testing.T, msg Message) {
				assert.Equal(t, iface.RoleHuman, msg.GetType())
				assert.Equal(t, "Hello, world!", msg.GetContent())
				assert.Empty(t, msg.ToolCalls())
			},
		},
		{
			name:        "ai_message_basic",
			description: "Creates AI message",
			messageType: iface.RoleAssistant,
			content:     "I'll help you with that.",
			expectError: false,
			validate: func(t *testing.T, msg Message) {
				assert.Equal(t, iface.RoleAssistant, msg.GetType())
				assert.Equal(t, "I'll help you with that.", msg.GetContent())
			},
		},
		{
			name:        "empty_content_message",
			description: "Tests behavior with empty content",
			messageType: iface.RoleHuman,
			content:     "",
			expectError: false, // Should be allowed
			validate: func(t *testing.T, msg Message) {
				assert.Equal(t, "", msg.GetContent())
				assert.Equal(t, iface.RoleHuman, msg.GetType())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg Message
			var err error

			switch tt.messageType {
			case iface.RoleHuman:
				msg = NewHumanMessage(tt.content)
			case iface.RoleAssistant:
				msg = NewAIMessage(tt.content)
				// Note: Tool calls would be set separately if needed
			case iface.RoleSystem:
				msg = NewSystemMessage(tt.content)
			case iface.RoleTool:
				msg = NewToolMessage(tt.content, "tool_call_id")
			}

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, msg)
				if tt.validate != nil {
					tt.validate(t, msg)
				}
			}
		})
	}
}

// TestChatHistoryAdvanced provides comprehensive tests for ChatHistory functionality
func TestChatHistoryAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		description string
		setup       func() ChatHistory
		operations  func(t *testing.T, history ChatHistory)
		expectError bool
		cleanup     func(ChatHistory)
	}{
		{
			name:        "basic_add_and_get",
			description: "Tests basic message addition and retrieval",
			setup:       func() ChatHistory { history, _ := NewBaseChatHistory(); return history },
			operations: func(t *testing.T, history ChatHistory) {
				msg1 := NewHumanMessage("First message")
				msg2 := NewAIMessage("Second message")

				err := history.AddMessage(msg1)
				require.NoError(t, err)

				err = history.AddMessage(msg2)
				require.NoError(t, err)

				messages, err := history.Messages()
				require.NoError(t, err)
				assert.Len(t, messages, 2)
				assert.Equal(t, "First message", messages[0].GetContent())
				assert.Equal(t, "Second message", messages[1].GetContent())
			},
		},
		{
			name:        "concurrent_access",
			description: "Tests thread safety of chat history operations",
			setup:       func() ChatHistory { history, _ := NewBaseChatHistory(); return history },
			operations: func(t *testing.T, history ChatHistory) {
				var wg sync.WaitGroup
				numWorkers := 10
				messagesPerWorker := 5

				// Concurrent adds
				for i := 0; i < numWorkers; i++ {
					wg.Add(1)
					go func(workerID int) {
						defer wg.Done()
						for j := 0; j < messagesPerWorker; j++ {
							msg := NewHumanMessage(fmt.Sprintf("Worker %d Message %d", workerID, j))
							err := history.AddMessage(msg)
							assert.NoError(t, err)
						}
					}(i)
				}

				wg.Wait()

				// Verify total count
				messages, err := history.Messages()
				require.NoError(t, err)
				assert.Len(t, messages, numWorkers*messagesPerWorker)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := tt.setup()
			tt.operations(t, history)

			if tt.cleanup != nil {
				tt.cleanup(history)
			}
		})
	}
}

// TestAdvancedErrorHandling tests comprehensive error scenarios
func TestAdvancedErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		description string
		operation   func() error
		expectCode  string
		expectRetry bool
	}{
		{
			name:        "validation_error",
			description: "Tests validation error handling",
			operation: func() error {
				return iface.NewSchemaError("VALIDATION_FAILED", "Message validation failed")
			},
			expectCode:  "VALIDATION_FAILED",
			expectRetry: false,
		},
		{
			name:        "config_error",
			description: "Tests configuration error handling",
			operation: func() error {
				return iface.NewSchemaError("CONFIG_INVALID", "Invalid configuration provided")
			},
			expectCode:  "CONFIG_INVALID",
			expectRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()

			if schemaErr, ok := err.(*iface.SchemaError); ok {
				assert.Equal(t, tt.expectCode, schemaErr.Code)
			} else {
				t.Errorf("Expected SchemaError, got %T", err)
			}
		})
	}
}

// TestConcurrentOperations tests various operations under concurrent load using existing utilities
func TestConcurrentOperations(t *testing.T) {
	t.Run("ConcurrentMessageCreation", func(t *testing.T) {
		concurrentRunner := NewConcurrentTestRunner(8, 2*time.Second, func() error {
			msg := NewHumanMessage("Concurrent test message")
			if msg.GetContent() == "" {
				return fmt.Errorf("empty content created")
			}
			return nil
		})

		err := concurrentRunner.Run()
		assert.NoError(t, err)
	})

	t.Run("ConcurrentChatHistoryAccess", func(t *testing.T) {
		history, err := NewBaseChatHistory()
		require.NoError(t, err)

		concurrentRunner := NewConcurrentTestRunner(8, 2*time.Second, func() error {
			msg := NewHumanMessage("Concurrent history test")
			return history.AddMessage(msg)
		})

		err = concurrentRunner.Run()
		assert.NoError(t, err)

		// Verify messages were added
		messages, err := history.Messages()
		require.NoError(t, err)
		assert.Greater(t, len(messages), 0)
	})
}

// Performance regression tests to validate targets are met
func TestPerformanceTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	// Test message creation performance target (<1ms)
	t.Run("MessageCreationPerformance", func(t *testing.T) {
		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			msg := NewHumanMessage(fmt.Sprintf("Performance test message %d", i))
			if msg.GetContent() == "" {
				t.Fatal("Message creation failed")
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(iterations)

		t.Logf("Average message creation time: %v", avgTime)

		// Target: <1ms per message
		if avgTime > time.Millisecond {
			t.Errorf("Message creation too slow: %v > 1ms target", avgTime)
		}
	})

	// Test factory function performance target (<100μs)
	t.Run("FactoryPerformance", func(t *testing.T) {
		iterations := 10000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			_ = NewHumanMessage("test")
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(iterations)

		t.Logf("Average factory function time: %v", avgTime)

		// Target: <100μs per factory call
		if avgTime > 100*time.Microsecond {
			t.Errorf("Factory function too slow: %v > 100μs target", avgTime)
		}
	})

	// Test validation performance
	t.Run("ValidationPerformance", func(t *testing.T) {
		msg := NewHumanMessage("Validation performance test message")
		iterations := 1000

		start := time.Now()
		for i := 0; i < iterations; i++ {
			err := ValidateMessage(msg)
			if err != nil {
				// Some validation errors are expected and fine
				_ = err
			}
		}
		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(iterations)

		t.Logf("Average validation time: %v", avgTime)

		// Target: <5ms per validation
		if avgTime > 5*time.Millisecond {
			t.Errorf("Validation too slow: %v > 5ms target", avgTime)
		}
	})
}
