// Package integration provides integration tests for schema package health monitoring.
// T013: Integration test for health monitoring
package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
)

// TestSchemaHealthChecking tests health monitoring integration across schema components
func TestSchemaHealthChecking(t *testing.T) {
	ctx := context.Background()

	t.Run("ValidationHealthCheck", func(t *testing.T) {
		// This test will be enhanced once health check interfaces are implemented

		// Create validation config
		config := schema.ValidationConfig{
			EnableValidation: true,
			MaxMessageLength: 1000,
			AllowedTypes: []schema.MessageType{
				schema.RoleHuman,
				schema.RoleAssistant,
				schema.RoleSystem,
			},
		}

		// Validate that validation config itself is healthy
		err := schema.ValidateConfig(config)
		assert.NoError(t, err, "Valid configuration should pass health check")

		// Test with various message types to ensure validation system is healthy
		testMessages := []schema.Message{
			schema.NewHumanMessage("Test human message"),
			schema.NewAIMessage("Test AI message"),
			schema.NewSystemMessage("Test system message"),
		}

		for _, msg := range testMessages {
			err := schema.ValidateMessage(msg, config)
			assert.NoError(t, err, "Valid messages should pass validation health check")
		}
	})

	t.Run("FactoryHealthCheck", func(t *testing.T) {
		// Test that all factory functions are working properly
		factoryTests := []struct {
			name      string
			operation func() interface{}
		}{
			{
				name:      "NewHumanMessage_health",
				operation: func() interface{} { return schema.NewHumanMessage("health test") },
			},
			{
				name:      "NewAIMessage_health",
				operation: func() interface{} { return schema.NewAIMessage("health test") },
			},
			{
				name:      "NewSystemMessage_health",
				operation: func() interface{} { return schema.NewSystemMessage("health test") },
			},
			{
				name:      "NewChatHistory_health",
				operation: func() interface{} { return schema.NewChatHistory() },
			},
			{
				name:      "NewDocument_health",
				operation: func() interface{} { return schema.NewDocument("test", "content") },
			},
		}

		for _, tt := range factoryTests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.operation()
				assert.NotNil(t, result, "Factory function should create non-nil result")

				// Verify the created object is functional
				switch v := result.(type) {
				case schema.Message:
					assert.NotEmpty(t, v.GetContent(), "Message should have content")
					assert.NotEqual(t, schema.MessageType(""), v.GetType(), "Message should have valid type")
				case schema.ChatHistory:
					assert.Equal(t, 0, v.Size(), "New chat history should be empty")
				case schema.Document:
					assert.NotEmpty(t, v.GetContent(), "Document should have content")
				}
			})
		}
	})

	t.Run("MemoryHealthCheck", func(t *testing.T) {
		// Test memory usage patterns to ensure no leaks
		initialAllocs := testing.AllocsPerRun(100, func() {
			msg := schema.NewHumanMessage("memory test")
			_ = msg.GetContent()
		})

		// Run the same operation many times
		subsequentAllocs := testing.AllocsPerRun(100, func() {
			msg := schema.NewHumanMessage("memory test")
			_ = msg.GetContent()
		})

		// Memory allocations should be consistent (no leaks)
		assert.InDelta(t, initialAllocs, subsequentAllocs, 0.5,
			"Memory allocation pattern should be consistent")

		t.Logf("Memory allocations per operation: initial=%.2f, subsequent=%.2f",
			initialAllocs, subsequentAllocs)
	})

	t.Run("ConcurrencyHealthCheck", func(t *testing.T) {
		// Test that concurrent operations don't cause health issues
		history := schema.NewChatHistory()

		// Run concurrent operations
		numWorkers := 10
		operationsPerWorker := 20

		errChan := make(chan error, numWorkers*operationsPerWorker)
		done := make(chan bool, numWorkers)

		for i := 0; i < numWorkers; i++ {
			go func(workerID int) {
				defer func() { done <- true }()

				for j := 0; j < operationsPerWorker; j++ {
					msg := schema.NewHumanMessage(fmt.Sprintf("Worker %d Message %d", workerID, j))
					err := history.AddMessage(msg)
					if err != nil {
						errChan <- err
					}
				}
			}(i)
		}

		// Wait for all workers to complete
		for i := 0; i < numWorkers; i++ {
			<-done
		}
		close(errChan)

		// Check for any errors
		var errors []error
		for err := range errChan {
			errors = append(errors, err)
		}

		assert.Empty(t, errors, "Concurrent operations should not produce errors")

		// Verify final state is healthy
		assert.Equal(t, numWorkers*operationsPerWorker, history.Size(),
			"All messages should be added successfully")
	})

	t.Run("PerformanceHealthCheck", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping performance health check in short mode")
		}

		// Performance thresholds for health checking
		const (
			maxMessageCreationTime = 1 * time.Millisecond
			maxFactoryTime         = 100 * time.Microsecond
			maxValidationTime      = 5 * time.Millisecond
		)

		// Test message creation performance health
		start := time.Now()
		msg := schema.NewHumanMessage("Performance health test message")
		creationTime := time.Since(start)

		assert.Less(t, creationTime, maxMessageCreationTime,
			"Message creation should be under performance threshold")

		// Test factory performance health
		start = time.Now()
		for i := 0; i < 100; i++ {
			_ = schema.NewHumanMessage("factory test")
		}
		avgFactoryTime := time.Since(start) / 100

		assert.Less(t, avgFactoryTime, maxFactoryTime,
			"Factory functions should be under performance threshold")

		// Test validation performance health
		config := schema.ValidationConfig{
			EnableValidation: true,
			MaxMessageLength: 1000,
		}

		start = time.Now()
		err := schema.ValidateMessage(msg, config)
		validationTime := time.Since(start)

		assert.NoError(t, err)
		assert.Less(t, validationTime, maxValidationTime,
			"Validation should be under performance threshold")

		t.Logf("Health check timings: creation=%v, factory=%v, validation=%v",
			creationTime, avgFactoryTime, validationTime)
	})
}

// TestHealthCheckIntegrationWithObservability tests health monitoring with metrics
func TestHealthCheckIntegrationWithObservability(t *testing.T) {
	t.Run("HealthMetricsCollection", func(t *testing.T) {
		// This test will be enhanced once health metrics are implemented

		// Simulate health check operations
		operations := []func(){
			func() { schema.NewHumanMessage("health metric test 1") },
			func() { schema.NewAIMessage("health metric test 2") },
			func() { schema.NewSystemMessage("health metric test 3") },
		}

		// Execute operations (metrics would be collected here)
		for i, op := range operations {
			start := time.Now()
			op()
			duration := time.Since(start)

			t.Logf("Operation %d duration: %v", i+1, duration)
			assert.Less(t, duration, time.Millisecond, "Operations should be fast")
		}
	})

	t.Run("HealthCheckWithErrors", func(t *testing.T) {
		// Test health checking behavior when errors occur

		// Create invalid configuration to trigger error
		invalidConfig := schema.ValidationConfig{
			MaxMessageLength: -1, // Invalid negative value
		}

		// Attempt validation (should fail)
		msg := schema.NewHumanMessage("test")
		err := schema.ValidateConfig(invalidConfig)

		// Error should be properly structured and not crash the system
		assert.Error(t, err, "Invalid configuration should be detected")

		// System should still be healthy after error
		validMsg := schema.NewHumanMessage("system still healthy")
		assert.NotNil(t, validMsg)
		assert.Equal(t, "system still healthy", validMsg.GetContent())
	})
}

// TestHealthCheckRecovery tests system recovery from health issues
func TestHealthCheckRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping health recovery test in short mode")
	}

	t.Run("RecoveryFromMemoryPressure", func(t *testing.T) {
		// Create many messages to simulate memory pressure
		messages := make([]schema.Message, 1000)

		start := time.Now()
		for i := 0; i < 1000; i++ {
			messages[i] = schema.NewHumanMessage(fmt.Sprintf("Memory pressure test %d", i))
		}
		creationTime := time.Since(start)

		// Verify system is still responsive after memory pressure
		testMsg := schema.NewHumanMessage("post-pressure test")
		assert.Equal(t, "post-pressure test", testMsg.GetContent())

		t.Logf("Created 1000 messages in %v, system remains healthy", creationTime)

		// Clean up by clearing references (simulate GC)
		messages = nil

		// Verify system works normally after cleanup
		normalMsg := schema.NewHumanMessage("normal operation")
		assert.Equal(t, "normal operation", normalMsg.GetContent())
	})

	t.Run("RecoveryFromConcurrentStress", func(t *testing.T) {
		// Create stress scenario with high concurrency
		history := schema.NewChatHistory()

		// High stress concurrent operations
		numWorkers := 50
		operationsPerWorker := 100

		errChan := make(chan error, numWorkers*operationsPerWorker)
		done := make(chan bool, numWorkers)

		start := time.Now()
		for i := 0; i < numWorkers; i++ {
			go func(workerID int) {
				defer func() { done <- true }()

				for j := 0; j < operationsPerWorker; j++ {
					msg := schema.NewHumanMessage(fmt.Sprintf("Stress W%d-Op%d", workerID, j))
					err := history.AddMessage(msg)
					if err != nil {
						errChan <- err
					}
				}
			}(i)
		}

		// Wait for completion
		for i := 0; i < numWorkers; i++ {
			<-done
		}
		stressTime := time.Since(start)
		close(errChan)

		// Verify no errors occurred during stress
		var errors []error
		for err := range errChan {
			errors = append(errors, err)
		}
		assert.Empty(t, errors, "System should handle stress without errors")

		// Verify system recovered and is still functional
		finalSize := history.Size()
		assert.Equal(t, numWorkers*operationsPerWorker, finalSize,
			"All operations should complete successfully")

		t.Logf("Completed %d operations in %v, system healthy", finalSize, stressTime)

		// Test normal operation after stress
		normalMsg := schema.NewHumanMessage("post-stress normal message")
		err := history.AddMessage(normalMsg)
		assert.NoError(t, err, "System should work normally after stress test")

		assert.Equal(t, finalSize+1, history.Size(), "System should continue working normally")
	})
}
