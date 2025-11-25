// Package memory provides comprehensive integration tests for memory implementations.
// These tests simulate real-world usage scenarios and test complete workflows.
package memory

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMemoryIntegration_ChatApplication simulates a chat application using memory
func TestMemoryIntegration_ChatApplication(t *testing.T) {

	t.Run("BufferMemory_ChatSession", func(t *testing.T) {
	ctx := context.Background()
		// Create buffer memory for a chat session
		memory, err := NewMemory(MemoryTypeBuffer,
			WithMemoryKey("chat_history"),
			WithReturnMessages(false),
			WithHumanPrefix("User"),
			WithAIPrefix("Assistant"),
		)
		require.NoError(t, err)

		// Simulate a conversation
		conversation := []struct {
			userInput string
			aiOutput  string
		}{
			{"Hello!", "Hi there! How can I help you today?"},
			{"What's the weather like?", "I don't have access to real-time weather data, but I can help you with other questions!"},
			{"Tell me about Go programming", "Go is a statically typed, compiled programming language designed at Google. It's known for its simplicity, efficiency, and strong support for concurrent programming."},
			{"How do I create a variable in Go?", "You can create a variable in Go using the `var` keyword or short variable declaration. For example:\nvar x int = 42\nor\ny := 42"},
		}

		// Save conversation to memory
		for i, msg := range conversation {
			inputs := map[string]any{"input": msg.userInput}
			outputs := map[string]any{"output": msg.aiOutput}

			err := memory.SaveContext(ctx, inputs, outputs)
			assert.NoError(t, err, "Failed to save context for message %d", i)
		}

		// Load memory and verify conversation history
		vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
		assert.NoError(t, err)

		history, ok := vars["chat_history"].(string)
		assert.True(t, ok, "Expected chat_history to be a string")

		// Verify all conversation turns are present
		for _, msg := range conversation {
			assert.Contains(t, history, msg.userInput, "User input should be in history")
			assert.Contains(t, history, msg.aiOutput, "AI output should be in history")
		}

		// Verify format
		assert.Contains(t, history, "User: Hello!")
		assert.Contains(t, history, "Assistant: Hi there!")
		assert.Contains(t, history, "User: What's the weather like?")
		assert.Contains(t, history, "Assistant: I don't have access to real-time weather data")

		// Test memory persistence across operations
		newVars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
		assert.NoError(t, err)

		newHistory, ok := newVars["chat_history"].(string)
		assert.True(t, ok)
		assert.Equal(t, history, newHistory, "Memory should persist across operations")
	})

	t.Run("WindowMemory_ChatSession", func(t *testing.T) {
		ctx := context.Background()
		// Create window memory with size 3
		memory, err := NewMemory(MemoryTypeBufferWindow,
			WithMemoryKey("recent_history"),
			WithWindowSize(3),
			WithReturnMessages(false),
		)
		require.NoError(t, err)

		// Simulate a long conversation
		for i := 1; i <= 6; i++ {
			inputs := map[string]any{"input": "Question " + string(rune(i+'0'))}
			outputs := map[string]any{"output": "Answer " + string(rune(i+'0'))}

			err := memory.SaveContext(ctx, inputs, outputs)
			assert.NoError(t, err, "Failed to save context for turn %d", i)
		}

		// Load memory and verify only recent 3 messages are kept (window size is 3 messages, not interactions)
		vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
		assert.NoError(t, err)

		history, ok := vars["recent_history"].(string)
		assert.True(t, ok)

		// With 6 interactions (12 messages total) and window of 3, should contain last 3 messages
		// Messages: Q1, A1, Q2, A2, Q3, A3, Q4, A4, Q5, A5, Q6, A6
		// Last 3: A5, Q6, A6
		assert.Contains(t, history, "Answer 5")
		assert.Contains(t, history, "Question 6")
		assert.Contains(t, history, "Answer 6")

		// Should NOT contain the first 3 interactions
		assert.NotContains(t, history, "Question 1")
		assert.NotContains(t, history, "Answer 1")
		assert.NotContains(t, history, "Question 2")
		assert.NotContains(t, history, "Answer 2")
		assert.NotContains(t, history, "Question 3")
		assert.NotContains(t, history, "Answer 3")
	})

	t.Run("MemoryFactory_ChatApplication", func(t *testing.T) {
		factory := NewFactory()

		// Configuration for different chat scenarios
		scenarios := []struct {
			name   string
			config Config
		}{
			{
				name: "GeneralChat",
				config: Config{
					Type:           MemoryTypeBuffer,
					MemoryKey:      "chat_history",
					ReturnMessages: false,
					HumanPrefix:    "Human",
					AIPrefix:       "AI",
					Enabled:        true,
				},
			},
			{
				name: "ShortTermMemory",
				config: Config{
					Type:           MemoryTypeBufferWindow,
					MemoryKey:      "recent_context",
					ReturnMessages: true,
					WindowSize:     5,
					Enabled:        true,
				},
			},
		}

		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				ctx := context.Background()
				memory, err := factory.CreateMemory(ctx, scenario.config)
				require.NoError(t, err)
				assert.NotNil(t, memory)

				// Test basic functionality
				vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
				assert.NoError(t, err)
				assert.Contains(t, vars, scenario.config.MemoryKey)
			})
		}
	})
}

// TestMemoryIntegration_ErrorScenarios tests error handling in integration scenarios
func TestMemoryIntegration_ErrorScenarios(t *testing.T) {

	t.Run("MemoryDisabled", func(t *testing.T) {
	ctx := context.Background()
		factory := NewFactory()
		config := Config{
			Type:    MemoryTypeBuffer,
			Enabled: false,
		}

		memory, err := factory.CreateMemory(ctx, config)
		assert.NoError(t, err)
		assert.IsType(t, &NoOpMemory{}, memory)

		// Test that NoOpMemory works correctly
		vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{}, vars)

		err = memory.SaveContext(ctx, map[string]any{}, map[string]any{})
		assert.NoError(t, err)

		err = memory.Clear(ctx)
		assert.NoError(t, err)
	})

	t.Run("InvalidConfiguration", func(t *testing.T) {
		ctx := context.Background()
		factory := NewFactory()

		invalidConfigs := []Config{
			{Type: "", Enabled: true},
			{Type: "invalid_memory_type", Enabled: true},
			{Type: "nonexistent", Enabled: true},
		}

		for _, config := range invalidConfigs {
			_, err := factory.CreateMemory(ctx, config)
			assert.Error(t, err, "Expected error for config: %+v", config)
		}
	})
}

// TestMemoryIntegration_PerformanceScenarios tests memory performance under load
func TestMemoryIntegration_PerformanceScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}


	t.Run("HighFrequencyOperations", func(t *testing.T) {
		ctx := context.Background()
		memory, err := NewMemory(MemoryTypeBuffer)
		require.NoError(t, err)

		// Simulate high-frequency chat interactions
		numInteractions := 100

		start := time.Now()
		for i := 0; i < numInteractions; i++ {
			inputs := map[string]any{"input": "Quick question " + string(rune(i%10+'0'))}
			outputs := map[string]any{"output": "Quick answer " + string(rune(i%10+'0'))}

			err := memory.SaveContext(ctx, inputs, outputs)
			assert.NoError(t, err)
		}
		saveDuration := time.Since(start)

		// Load memory and verify all interactions are present
		start = time.Now()
		vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
		assert.NoError(t, err)
		loadDuration := time.Since(start)

		history := vars["history"].(string)

		// Verify performance is reasonable (adjust thresholds as needed)
		assert.Less(t, saveDuration, 1*time.Second, "Save operations should complete quickly")
		assert.Less(t, loadDuration, 100*time.Millisecond, "Load operation should complete quickly")

		// Verify data integrity
		assert.Contains(t, history, "Quick question")
		assert.Contains(t, history, "Quick answer")
	})

	t.Run("LargeConversationHistory", func(t *testing.T) {
		ctx := context.Background()
		memory, err := NewMemory(MemoryTypeBuffer)
		require.NoError(t, err)

		// Simulate a large conversation
		numMessages := 1000
		longMessage := strings.Repeat("This is a long message to test memory capacity. ", 10)

		start := time.Now()
		for i := 0; i < numMessages; i++ {
			inputs := map[string]any{"input": longMessage + string(rune(i%10+'0'))}
			outputs := map[string]any{"output": "Response to: " + string(rune(i%10+'0'))}

			err := memory.SaveContext(ctx, inputs, outputs)
			assert.NoError(t, err)
		}
		saveDuration := time.Since(start)

		// Load memory
		start = time.Now()
		vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
		assert.NoError(t, err)
		loadDuration := time.Since(start)

		history := vars["history"].(string)

		// Verify performance is acceptable for large conversations
		assert.Less(t, saveDuration, 5*time.Second, "Large conversation save should complete reasonably")
		assert.Less(t, loadDuration, 500*time.Millisecond, "Large conversation load should complete reasonably")

		// Verify data integrity
		assert.Contains(t, history, "This is a long message")
		assert.Contains(t, history, "Response to:")
		assert.Greater(t, len(history), 10000, "History should contain substantial content")
	})
}

// TestMemoryIntegration_ComplexWorkflows tests complex memory usage patterns
func TestMemoryIntegration_ComplexWorkflows(t *testing.T) {

	t.Run("MemorySwitching", func(t *testing.T) {
		ctx := context.Background()
		factory := NewFactory()

		// Start with buffer memory
		bufferConfig := Config{
			Type:      MemoryTypeBuffer,
			MemoryKey: "conversation",
			Enabled:   true,
		}

		bufferMemory, err := factory.CreateMemory(ctx, bufferConfig)
		require.NoError(t, err)

		// Add some conversation
		inputs := map[string]any{"input": "Hello"}
		outputs := map[string]any{"output": "Hi!"}
		err = bufferMemory.SaveContext(ctx, inputs, outputs)
		assert.NoError(t, err)

		// Switch to window memory with different configuration
		windowConfig := Config{
			Type:       MemoryTypeBufferWindow,
			MemoryKey:  "recent_conversation",
			WindowSize: 2,
			Enabled:    true,
		}

		windowMemory, err := factory.CreateMemory(ctx, windowConfig)
		require.NoError(t, err)

		// Add more conversation to window memory
		for i := 0; i < 4; i++ {
			inputs := map[string]any{"input": "Question " + string(rune(i+'0'))}
			outputs := map[string]any{"output": "Answer " + string(rune(i+'0'))}
			err = windowMemory.SaveContext(ctx, inputs, outputs)
			assert.NoError(t, err)
		}

		// Verify window memory only keeps recent messages
		vars, err := windowMemory.LoadMemoryVariables(ctx, map[string]any{})
		assert.NoError(t, err)

		history := vars["recent_conversation"].(string)
		// With 4 interactions (8 messages: Q0, A0, Q1, A1, Q2, A2, Q3, A3) and window of 2
		// Should contain the last 2 messages: Q3, A3
		assert.Contains(t, history, "Question 3")
		assert.Contains(t, history, "Answer 3")
		assert.NotContains(t, history, "Question 0")
		assert.NotContains(t, history, "Question 1")
	})

	t.Run("MemoryPersistenceSimulation", func(t *testing.T) {
		ctx := context.Background()
		// Simulate memory persistence across "sessions"
		memory, err := NewMemory(MemoryTypeBuffer, WithMemoryKey("persistent_history"))
		require.NoError(t, err)

		// Session 1
		session1 := []struct{ input, output string }{
			{"What's your name?", "I'm an AI assistant."},
			{"What's the time?", "I don't have access to current time."},
		}

		for _, msg := range session1 {
			err := memory.SaveContext(ctx, map[string]any{"input": msg.input}, map[string]any{"output": msg.output})
			assert.NoError(t, err)
		}

		// Simulate session break (memory persists)
		vars1, err := memory.LoadMemoryVariables(ctx, map[string]any{})
		assert.NoError(t, err)

		// Session 2
		session2 := []struct{ input, output string }{
			{"Tell me a joke", "Why did the computer go to therapy? It had too many bytes of emotional baggage!"},
			{"That's funny!", "I'm glad you liked it!"},
		}

		for _, msg := range session2 {
			err := memory.SaveContext(ctx, map[string]any{"input": msg.input}, map[string]any{"output": msg.output})
			assert.NoError(t, err)
		}

		// Verify all messages are present
		vars2, err := memory.LoadMemoryVariables(ctx, map[string]any{})
		assert.NoError(t, err)

		history1 := vars1["persistent_history"].(string)
		history2 := vars2["persistent_history"].(string)

		// Session 2 should contain all messages from both sessions
		assert.Contains(t, history2, session1[0].input)
		assert.Contains(t, history2, session1[0].output)
		assert.Contains(t, history2, session1[1].input)
		assert.Contains(t, history2, session1[1].output)
		assert.Contains(t, history2, session2[0].input)
		assert.Contains(t, history2, session2[0].output)
		assert.Contains(t, history2, session2[1].input)
		assert.Contains(t, history2, session2[1].output)

		// Session 1 history should be subset of session 2
		assert.Contains(t, history2, history1)
		assert.Greater(t, len(history2), len(history1))
	})

	t.Run("MemoryWithCustomKeys", func(t *testing.T) {
		ctx := context.Background()
		memory, err := NewMemory(MemoryTypeBuffer,
			WithInputKey("user_message"),
			WithOutputKey("assistant_response"),
			WithMemoryKey("conversation_log"),
		)
		require.NoError(t, err)

		// Test with custom keys
		inputs := map[string]any{"user_message": "Custom input"}
		outputs := map[string]any{"assistant_response": "Custom output"}

		err = memory.SaveContext(ctx, inputs, outputs)
		assert.NoError(t, err)

		vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
		assert.NoError(t, err)

		history := vars["conversation_log"].(string)
		assert.Contains(t, history, "Custom input")
		assert.Contains(t, history, "Custom output")
	})
}

// TestMemoryIntegration_MultiMemorySetup tests using multiple memory instances
func TestMemoryIntegration_MultiMemorySetup(t *testing.T) {
	ctx := context.Background()
	factory := NewFactory()

	// Create multiple memory instances for different purposes
	memories := make(map[string]Memory)

	configs := map[string]Config{
		"short_term": {
			Type:       MemoryTypeBufferWindow,
			MemoryKey:  "recent_messages",
			WindowSize: 3,
			Enabled:    true,
		},
		"long_term": {
			Type:      MemoryTypeBuffer,
			MemoryKey: "full_history",
			Enabled:   true,
		},
		"summary": {
			Type:      MemoryTypeBuffer,
			MemoryKey: "summary",
			Enabled:   true,
		},
	}

	// Initialize all memories
	for name, config := range configs {
		memory, err := factory.CreateMemory(ctx, config)
		require.NoError(t, err)
		memories[name] = memory
	}

	// Add conversation to all memories
	conversation := []struct{ input, output string }{
		{"Hello", "Hi there!"},
		{"How are you?", "I'm doing well."},
		{"What's new?", "I've been learning about memory systems."},
		{"That's interesting", "Yes, memory is crucial for maintaining context."},
		{"Tell me more", "Memory allows AI systems to remember previous interactions."},
	}

	for _, msg := range conversation {
		inputs := map[string]any{"input": msg.input}
		outputs := map[string]any{"output": msg.output}

		for name, memory := range memories {
			err := memory.SaveContext(ctx, inputs, outputs)
			assert.NoError(t, err, "Failed to save to %s memory", name)
		}
	}

	// Test each memory type behaves as expected
	t.Run("ShortTermMemory", func(t *testing.T) {
		vars, err := memories["short_term"].LoadMemoryVariables(ctx, map[string]any{})
		assert.NoError(t, err)

		history := vars["recent_messages"].(string)
		// With window of 3 messages and 5 interactions (10 messages total)
		// Should contain the last 3 messages: A4, Q5, A5 (or similar)
		// The window keeps messages, not interactions
		assert.Contains(t, history, "Tell me more")
		assert.Contains(t, history, "Memory allows")
		// Should not contain the first interactions
		assert.NotContains(t, history, "Hello")
		assert.NotContains(t, history, "How are you?")
	})

	t.Run("LongTermMemory", func(t *testing.T) {
		vars, err := memories["long_term"].LoadMemoryVariables(ctx, map[string]any{})
		assert.NoError(t, err)

		history := vars["full_history"].(string)
		// Should contain all interactions
		for _, msg := range conversation {
			assert.Contains(t, history, msg.input)
			assert.Contains(t, history, msg.output)
		}
	})

	t.Run("SummaryMemory", func(t *testing.T) {
		vars, err := memories["summary"].LoadMemoryVariables(ctx, map[string]any{})
		assert.NoError(t, err)

		history := vars["summary"].(string)
		// Should contain all interactions (same as long-term for buffer memory)
		for _, msg := range conversation {
			assert.Contains(t, history, msg.input)
			assert.Contains(t, history, msg.output)
		}
	})
}

// TestMemoryIntegration_Cleanup tests memory cleanup scenarios
func TestMemoryIntegration_Cleanup(t *testing.T) {

	t.Run("MemoryClearing", func(t *testing.T) {
		ctx := context.Background()
		memory, err := NewMemory(MemoryTypeBuffer)
		require.NoError(t, err)

		// Add some content
		inputs := map[string]any{"input": "Test message"}
		outputs := map[string]any{"output": "Test response"}
		err = memory.SaveContext(ctx, inputs, outputs)
		assert.NoError(t, err)

		// Verify content exists
		vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
		assert.NoError(t, err)
		history := vars["history"].(string)
		assert.Contains(t, history, "Test message")

		// Clear memory
		err = memory.Clear(ctx)
		assert.NoError(t, err)

		// Verify content is cleared
		vars, err = memory.LoadMemoryVariables(ctx, map[string]any{})
		assert.NoError(t, err)
		clearedHistory := vars["history"].(string)
		assert.Equal(t, "", clearedHistory)
		assert.NotEqual(t, history, clearedHistory)
	})

	t.Run("MultipleClearOperations", func(t *testing.T) {
		ctx := context.Background()
		memory, err := NewMemory(MemoryTypeBuffer)
		require.NoError(t, err)

		// Multiple clear operations should be safe
		for i := 0; i < 3; i++ {
			err = memory.Clear(ctx)
			assert.NoError(t, err, "Clear operation %d should succeed", i+1)
		}

		// Memory should still work after multiple clears
		inputs := map[string]any{"input": "After clear"}
		outputs := map[string]any{"output": "Still working"}
		err = memory.SaveContext(ctx, inputs, outputs)
		assert.NoError(t, err)

		vars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
		assert.NoError(t, err)
		history := vars["history"].(string)
		assert.Contains(t, history, "After clear")
	})
}

// BenchmarkIntegration benchmarks complete memory workflows
func BenchmarkIntegration(b *testing.B) {
	ctx := context.Background()

	b.Run("CompleteChatWorkflow", func(b *testing.B) {
		memory, _ := NewMemory(MemoryTypeBuffer)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate a complete chat turn
			inputs := map[string]any{"input": "Hello"}
			outputs := map[string]any{"output": "Hi there!"}

			memory.SaveContext(ctx, inputs, outputs)
			memory.LoadMemoryVariables(ctx, map[string]any{})
		}
	})

	b.Run("FactoryWorkflow", func(b *testing.B) {
		factory := NewFactory()
		config := Config{
			Type:      MemoryTypeBuffer,
			Enabled:   true,
			MemoryKey: "benchmark_history",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			memory, _ := factory.CreateMemory(ctx, config)
			_ = memory.MemoryVariables()
		}
	})
}
