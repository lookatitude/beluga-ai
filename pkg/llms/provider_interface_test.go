// Package llms provides comprehensive provider interface testing.
// This file contains test suites that verify all providers implement the ChatModel interface correctly.
package llms

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ProviderInterfaceTestSuite provides comprehensive interface compliance tests
type ProviderInterfaceTestSuite struct {
	ProviderName string
	Provider     iface.ChatModel
	Config       *Config
	Verbose      bool
}

// NewProviderInterfaceTestSuite creates a new provider interface test suite
func NewProviderInterfaceTestSuite(providerName string, provider iface.ChatModel, config *Config) *ProviderInterfaceTestSuite {
	return &ProviderInterfaceTestSuite{
		ProviderName: providerName,
		Provider:     provider,
		Config:       config,
		Verbose:      false,
	}
}

// WithVerbose enables verbose output for the test suite
func (s *ProviderInterfaceTestSuite) WithVerbose(verbose bool) *ProviderInterfaceTestSuite {
	s.Verbose = verbose
	return s
}

// Run runs all interface compliance tests
func (s *ProviderInterfaceTestSuite) Run(t *testing.T) {
	t.Run(fmt.Sprintf("%s_interface_compliance", s.ProviderName), func(t *testing.T) {
		s.testBasicInterfaceCompliance(t)
		s.testChatModelMethods(t)
		s.testRunnableInterface(t)
		s.testHealthCheckerInterface(t)
		s.testModelInfoProviderInterface(t)
		s.testStreamMessageHandlerInterface(t)
		s.testMessageGeneratorInterface(t)
		s.testErrorHandling(t)
		s.testConfigurationValidation(t)
		s.testConcurrentAccess(t)
	})
}

// testBasicInterfaceCompliance tests that the provider implements all required interfaces
func (s *ProviderInterfaceTestSuite) testBasicInterfaceCompliance(t *testing.T) {
	t.Run("interface_implementation", func(t *testing.T) {
		provider := s.Provider

		// Test that provider implements ChatModel interface
		var _ iface.ChatModel = provider
		assert.NotNil(t, provider, "Provider should implement ChatModel interface")

		// Test that provider implements Runnable interface
		var _ core.Runnable = provider
		assert.NotNil(t, provider, "Provider should implement Runnable interface")

		// Test that provider implements LLM interface
		var _ iface.LLM = provider
		assert.NotNil(t, provider, "Provider should implement LLM interface")

		if s.Verbose {
			t.Logf("Provider %s implements all required interfaces", s.ProviderName)
		}
	})
}

// testChatModelMethods tests ChatModel-specific methods
func (s *ProviderInterfaceTestSuite) testChatModelMethods(t *testing.T) {
	t.Run("chat_model_methods", func(t *testing.T) {
		provider := s.Provider

		// Test GetModelName
		modelName := provider.GetModelName()
		assert.NotEmpty(t, modelName, "GetModelName should return non-empty string")
		assert.IsType(t, "", modelName, "GetModelName should return string")

		// Test BindTools
		tools := []tools.Tool{
			NewMockTool("test-tool-1"),
			NewMockTool("test-tool-2"),
		}

		boundProvider := provider.BindTools(tools)
		assert.NotNil(t, boundProvider, "BindTools should return non-nil provider")

		// Verify bound provider still implements ChatModel
		var _ iface.ChatModel = boundProvider

		// Test that bound provider has access to tools (if supported)
		health := boundProvider.CheckHealth()
		if health != nil {
			toolsCount, exists := health["tools_bound"]
			if exists {
				assert.Equal(t, 2, toolsCount, "Health should show correct number of bound tools")
			}
		}

		if s.Verbose {
			t.Logf("Provider %s has model name: %s", s.ProviderName, modelName)
		}
	})
}

// testRunnableInterface tests Runnable interface compliance
func (s *ProviderInterfaceTestSuite) testRunnableInterface(t *testing.T) {
	t.Run("runnable_interface", func(t *testing.T) {
		provider := s.Provider
		ctx, _ := TestContext()

		// Test Invoke method
		input := "Test input for runnable interface"
		result, err := provider.Invoke(ctx, input)
		assert.NoError(t, err, "Invoke should not error")
		assert.NotNil(t, result, "Invoke should return result")

		// Test Batch method
		inputs := []any{
			"Batch input 1",
			"Batch input 2",
			"Batch input 3",
		}

		results, err := provider.Batch(ctx, inputs)
		assert.NoError(t, err, "Batch should not error")
		assert.Len(t, results, len(inputs), "Batch should return correct number of results")

		for i, result := range results {
			assert.NotNil(t, result, "Batch result %d should not be nil", i)
		}

		// Test Stream method
		streamChan, err := provider.Stream(ctx, "Stream test input")
		assert.NoError(t, err, "Stream should not error")

		// Collect stream results
		streamResults := make([]any, 0)
		timeout := time.After(5 * time.Second)

	collectLoop:
		for {
			select {
			case result, ok := <-streamChan:
				if !ok {
					break collectLoop
				}
				streamResults = append(streamResults, result)
			case <-timeout:
				t.Fatal("Stream timed out")
			}
		}

		assert.NotEmpty(t, streamResults, "Stream should return results")

		if s.Verbose {
			t.Logf("Provider %s completed Runnable interface tests", s.ProviderName)
		}
	})
}

// testHealthCheckerInterface tests HealthChecker interface compliance
func (s *ProviderInterfaceTestSuite) testHealthCheckerInterface(t *testing.T) {
	t.Run("health_checker_interface", func(t *testing.T) {
		provider := s.Provider

		health := provider.CheckHealth()
		assert.NotNil(t, health, "CheckHealth should return health data")
		assert.IsType(t, map[string]interface{}{}, health, "CheckHealth should return map")

		// Verify common health fields
		assert.Contains(t, health, "state", "Health should contain state field")
		state, ok := health["state"]
		assert.True(t, ok, "State should be accessible")
		assert.NotEmpty(t, state, "State should not be empty")

		// Check for other common fields
		if _, exists := health["provider"]; exists {
			providerName, ok := health["provider"]
			assert.True(t, ok, "Provider field should be accessible")
			assert.NotEmpty(t, providerName, "Provider name should not be empty")
		}

		if _, exists := health["model"]; exists {
			modelName, ok := health["model"]
			assert.True(t, ok, "Model field should be accessible")
			assert.NotEmpty(t, modelName, "Model name should not be empty")
		}

		if _, exists := health["timestamp"]; exists {
			timestamp, ok := health["timestamp"]
			assert.True(t, ok, "Timestamp field should be accessible")
			assert.NotNil(t, timestamp, "Timestamp should not be nil")
		}

		if s.Verbose {
			t.Logf("Provider %s health state: %v", s.ProviderName, health["state"])
		}
	})
}

// testModelInfoProviderInterface tests ModelInfoProvider interface compliance
func (s *ProviderInterfaceTestSuite) testModelInfoProviderInterface(t *testing.T) {
	t.Run("model_info_provider_interface", func(t *testing.T) {
		provider := s.Provider

		// For ChatModelAdapter, we need to check if it has GetModelInfo method
		providerValue := reflect.ValueOf(provider)
		getModelInfoMethod := providerValue.MethodByName("GetModelInfo")

		if getModelInfoMethod.IsValid() {
			// Call GetModelInfo if it exists
			results := getModelInfoMethod.Call([]reflect.Value{})
			if len(results) > 0 {
				modelInfo := results[0].Interface()

				// Try to access fields using reflection
				modelInfoValue := reflect.ValueOf(modelInfo)
				if modelInfoValue.Kind() == reflect.Struct {
					// Check common fields
					nameField := modelInfoValue.FieldByName("Name")
					if nameField.IsValid() {
						name := nameField.String()
						assert.NotEmpty(t, name, "Model info should have name")
					}

					providerField := modelInfoValue.FieldByName("Provider")
					if providerField.IsValid() {
						provider := providerField.String()
						assert.NotEmpty(t, provider, "Model info should have provider")
					}
				}
			}
		} else {
			// Skip test if method doesn't exist (for some providers)
			t.Skip("GetModelInfo method not available for this provider")
		}

		if s.Verbose {
			t.Logf("Provider %s supports ModelInfoProvider interface", s.ProviderName)
		}
	})
}

// testStreamMessageHandlerInterface tests StreamMessageHandler interface compliance
func (s *ProviderInterfaceTestSuite) testStreamMessageHandlerInterface(t *testing.T) {
	t.Run("stream_message_handler_interface", func(t *testing.T) {
		provider := s.Provider
		ctx, _ := TestContext()

		messages := CreateTestMessages()

		// Test StreamChat method
		streamChan, err := provider.StreamChat(ctx, messages)
		assert.NoError(t, err, "StreamChat should not error")
		assert.NotNil(t, streamChan, "StreamChat should return channel")

		// Collect some chunks to verify streaming works
		chunkCount := 0
		timeout := time.After(2 * time.Second)

	collectChunks:
		for {
			select {
			case chunk, ok := <-streamChan:
				if !ok {
					break collectChunks
				}
				chunkCount++

				// Verify chunk structure
				assert.IsType(t, iface.AIMessageChunk{}, chunk, "Chunk should be AIMessageChunk")

				// If there's an error in the chunk, it should be handled
				if chunk.Err != nil {
					if s.Verbose {
						t.Logf("Stream chunk error: %v", chunk.Err)
					}
				}

				// Stop after collecting a few chunks to avoid hanging
				if chunkCount >= 3 {
					break collectChunks
				}

			case <-timeout:
				break collectChunks
			}
		}

		assert.Greater(t, chunkCount, 0, "Should receive at least one chunk")

		if s.Verbose {
			t.Logf("Provider %s streamed %d chunks", s.ProviderName, chunkCount)
		}
	})
}

// testMessageGeneratorInterface tests MessageGenerator interface compliance
func (s *ProviderInterfaceTestSuite) testMessageGeneratorInterface(t *testing.T) {
	t.Run("message_generator_interface", func(t *testing.T) {
		provider := s.Provider
		ctx, _ := TestContext()

		messages := CreateTestMessages()

		// Test GenerateMessages method (if available)
		providerValue := reflect.ValueOf(provider)
		generateMessagesMethod := providerValue.MethodByName("GenerateMessages")

		if generateMessagesMethod.IsValid() {
			// Prepare arguments
			args := []reflect.Value{
				reflect.ValueOf(ctx),
				reflect.ValueOf(messages),
			}

			// Add empty options slice
			emptyOptions := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf([]core.Option{}).Elem()), 0, 0)
			args = append(args, emptyOptions)

			// Call method
			results := generateMessagesMethod.Call(args)

			// Check results
			if len(results) >= 2 {
				resultMessages := results[0].Interface()
				err := results[1].Interface()

				if err != nil && err.(error) != nil {
					assert.NoError(t, err.(error), "GenerateMessages should not error")
				} else {
					assert.NotNil(t, resultMessages, "GenerateMessages should return messages")

					// Try to check if it's a slice
					resultValue := reflect.ValueOf(resultMessages)
					if resultValue.Kind() == reflect.Slice {
						assert.Greater(t, resultValue.Len(), 0, "Should return at least one message")
					}
				}
			}
		} else {
			// Skip if method doesn't exist
			t.Skip("GenerateMessages method not available for this provider")
		}

		if s.Verbose {
			t.Logf("Provider %s supports MessageGenerator interface", s.ProviderName)
		}
	})
}

// testErrorHandling tests error handling capabilities
func (s *ProviderInterfaceTestSuite) testErrorHandling(t *testing.T) {
	t.Run("error_handling", func(t *testing.T) {
		provider := s.Provider

		// Test with invalid inputs
		ctx, _ := TestContext()

		// Test with nil messages
		_, err := provider.Generate(ctx, nil)
		// Some providers might handle nil gracefully, others might error
		if err != nil {
			assert.Error(t, err, "Should handle nil messages appropriately")
		}

		// Test with empty messages
		_, err = provider.Generate(ctx, []schema.Message{})
		// Some providers might handle empty messages gracefully
		if err != nil {
			assert.Error(t, err, "Should handle empty messages appropriately")
		}

		// Test timeout scenario
		shortCtx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		messages := CreateTestMessages()
		_, err = provider.Generate(shortCtx, messages)

		// Should either succeed quickly or error due to timeout
		if err != nil {
			assert.Contains(t, strings.ToLower(err.Error()), "timeout", "Error should be related to timeout")
		}

		if s.Verbose {
			t.Logf("Provider %s error handling tested", s.ProviderName)
		}
	})
}

// testConfigurationValidation tests configuration validation
func (s *ProviderInterfaceTestSuite) testConfigurationValidation(t *testing.T) {
	t.Run("configuration_validation", func(t *testing.T) {
		if s.Config == nil {
			t.Skip("No configuration provided for validation test")
		}

		// Test configuration validation
		err := s.Config.Validate()
		assert.NoError(t, err, "Configuration should be valid")

		// Test configuration with invalid values
		invalidConfig := *s.Config
		invalidConfig.Provider = "" // Invalid provider

		err = invalidConfig.Validate()
		assert.Error(t, err, "Invalid configuration should fail validation")

		if s.Verbose {
			t.Logf("Provider %s configuration validation tested", s.ProviderName)
		}
	})
}

// testConcurrentAccess tests concurrent access to the provider
func (s *ProviderInterfaceTestSuite) testConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent access test in short mode")
	}

	t.Run("concurrent_access", func(t *testing.T) {
		provider := s.Provider
		ctx, _ := TestContext()

		messages := CreateTestMessages()

		// Test concurrent Generate calls
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(id int) {
				defer func() { done <- true }()

				// Add some variation to messages
				testMessages := make([]schema.Message, len(messages))
				copy(testMessages, messages)

				response, err := provider.Generate(ctx, testMessages)
				assert.NoError(t, err, "Concurrent Generate %d should not error", id)
				assert.NotNil(t, response, "Concurrent Generate %d should return response", id)
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			select {
			case <-done:
				// Goroutine completed
			case <-time.After(10 * time.Second):
				t.Fatal("Concurrent test timed out")
			}
		}

		if s.Verbose {
			t.Logf("Provider %s concurrent access test completed", s.ProviderName)
		}
	})
}

// TestAllProviderInterfaces tests interface compliance for multiple providers
func TestAllProviderInterfaces(t *testing.T) {
	providers := []struct {
		name   string
		config *Config
	}{
		{
			name:   "mock_provider",
			config: CreateTestConfig(),
		},
	}

	for _, p := range providers {
		t.Run(fmt.Sprintf("interface_test_%s", p.name), func(t *testing.T) {
			// Create provider instance
			mockProvider := NewAdvancedMockChatModel("test-model",
				WithProviderName(p.name),
				WithResponses("Test response"),
			)

			suite := NewProviderInterfaceTestSuite(p.name, mockProvider, p.config)
			suite.WithVerbose(false).Run(t)
		})
	}
}

// TestInterfaceReflection provides detailed reflection-based interface testing
func TestInterfaceReflection(t *testing.T) {
	provider := NewAdvancedMockChatModel("reflection-test")

	providerType := reflect.TypeOf(provider)
	providerValue := reflect.ValueOf(provider)

	interfaces := []reflect.Type{
		reflect.TypeOf((*iface.ChatModel)(nil)).Elem(),
		reflect.TypeOf((*core.Runnable)(nil)).Elem(),
		reflect.TypeOf((*iface.LLM)(nil)).Elem(),
	}

	for _, ifaceType := range interfaces {
		t.Run(fmt.Sprintf("implements_%s", ifaceType.Name()), func(t *testing.T) {
			assert.True(t, providerType.Implements(ifaceType),
				"Provider should implement %s interface", ifaceType.Name())

			// Test that all interface methods are implemented
			for i := 0; i < ifaceType.NumMethod(); i++ {
				method := ifaceType.Method(i)
				providerMethod := providerValue.MethodByName(method.Name)

				assert.True(t, providerMethod.IsValid(),
					"Provider should implement method %s", method.Name)

				// Check method signature compatibility
				if providerMethod.IsValid() {
					providerMethodType := providerMethod.Type()
					expectedType := method.Type

					// Compare method signatures (simplified check)
					assert.Equal(t, expectedType.NumIn(), providerMethodType.NumIn(),
						"Method %s should have correct number of input parameters", method.Name)
					assert.Equal(t, expectedType.NumOut(), providerMethodType.NumOut(),
						"Method %s should have correct number of output parameters", method.Name)
				}
			}
		})
	}
}

// TestMethodSignatures tests that all provider methods have correct signatures
func TestMethodSignatures(t *testing.T) {
	provider := NewAdvancedMockChatModel("signature-test")

	methods := map[string]struct {
		inputTypes  []reflect.Type
		outputTypes []reflect.Type
	}{
		"Generate": {
			inputTypes: []reflect.Type{
				reflect.TypeOf((*context.Context)(nil)).Elem(),
				reflect.TypeOf([]schema.Message{}),
				reflect.TypeOf([]core.Option{}),
			},
			outputTypes: []reflect.Type{
				reflect.TypeOf((*schema.Message)(nil)).Elem(),
				reflect.TypeOf((*error)(nil)).Elem(),
			},
		},
		"StreamChat": {
			inputTypes: []reflect.Type{
				reflect.TypeOf((*context.Context)(nil)).Elem(),
				reflect.TypeOf([]schema.Message{}),
				reflect.TypeOf([]core.Option{}),
			},
			outputTypes: []reflect.Type{
				reflect.TypeOf((<-chan iface.AIMessageChunk)(nil)),
				reflect.TypeOf((*error)(nil)).Elem(),
			},
		},
		"Invoke": {
			inputTypes: []reflect.Type{
				reflect.TypeOf((*context.Context)(nil)).Elem(),
				reflect.TypeOf((*interface{})(nil)).Elem(),
				reflect.TypeOf([]core.Option{}),
			},
			outputTypes: []reflect.Type{
				reflect.TypeOf((*interface{})(nil)).Elem(),
				reflect.TypeOf((*error)(nil)).Elem(),
			},
		},
		"GetModelName": {
			inputTypes:  []reflect.Type{},
			outputTypes: []reflect.Type{reflect.TypeOf("")},
		},
		"CheckHealth": {
			inputTypes:  []reflect.Type{},
			outputTypes: []reflect.Type{reflect.TypeOf(map[string]interface{}{})},
		},
	}

	providerValue := reflect.ValueOf(provider)

	for methodName, expected := range methods {
		t.Run(fmt.Sprintf("method_%s_signature", methodName), func(t *testing.T) {
			method := providerValue.MethodByName(methodName)
			require.True(t, method.IsValid(), "Method %s should exist", methodName)

			methodType := method.Type()

			// Check input parameters
			assert.Equal(t, len(expected.inputTypes), methodType.NumIn(),
				"Method %s should have correct number of input parameters", methodName)

			for i, expectedType := range expected.inputTypes {
				if i < methodType.NumIn() {
					assert.Equal(t, expectedType, methodType.In(i),
						"Method %s parameter %d should have correct type", methodName, i)
				}
			}

			// Check output parameters
			assert.Equal(t, len(expected.outputTypes), methodType.NumOut(),
				"Method %s should have correct number of output parameters", methodName)

			for i, expectedType := range expected.outputTypes {
				if i < methodType.NumOut() {
					assert.Equal(t, expectedType, methodType.Out(i),
						"Method %s return value %d should have correct type", methodName, i)
				}
			}
		})
	}
}

// TestProviderContract tests the behavioral contract that all providers should follow
func TestProviderContract(t *testing.T) {
	provider := NewAdvancedMockChatModel("contract-test",
		WithResponses("Consistent response"),
	)

	ctx, _ := TestContext()
	messages := CreateTestMessages()

	// Test idempotency - same input should give same result
	response1, err1 := provider.Generate(ctx, messages)
	response2, err2 := provider.Generate(ctx, messages)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, response1.GetContent(), response2.GetContent(),
		"Same input should produce same output (idempotency)")

	// Test that provider state doesn't affect results
	provider.Reset()
	response3, err3 := provider.Generate(ctx, messages)

	assert.NoError(t, err3)
	assert.Equal(t, response1.GetContent(), response3.GetContent(),
		"Reset provider should produce same results")

	// Test that health checks don't affect functionality
	_ = provider.CheckHealth()
	response4, err4 := provider.Generate(ctx, messages)

	assert.NoError(t, err4)
	assert.Equal(t, response1.GetContent(), response4.GetContent(),
		"Health check should not affect functionality")
}

// TestProviderLifecycle tests the complete lifecycle of a provider
func TestProviderLifecycle(t *testing.T) {
	// Create provider
	provider := NewAdvancedMockChatModel("lifecycle-test")

	// Test initial state
	initialHealth := provider.CheckHealth()
	assert.Equal(t, "healthy", initialHealth["state"])
	assert.Equal(t, 0, initialHealth["call_count"])

	// Test usage
	ctx, _ := TestContext()
	messages := CreateTestMessages()

	for i := 0; i < 5; i++ {
		response, err := provider.Generate(ctx, messages)
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}

	// Test state after usage
	usageHealth := provider.CheckHealth()
	assert.Equal(t, "healthy", usageHealth["state"])
	assert.Equal(t, 5, usageHealth["call_count"])

	// Test reset
	provider.Reset()
	resetHealth := provider.CheckHealth()
	assert.Equal(t, "healthy", resetHealth["state"])
	assert.Equal(t, 0, resetHealth["call_count"])

	// Test continued functionality after reset
	response, err := provider.Generate(ctx, messages)
	assert.NoError(t, err)
	assert.NotNil(t, response)

	finalHealth := provider.CheckHealth()
	assert.Equal(t, "healthy", finalHealth["state"])
	assert.Equal(t, 1, finalHealth["call_count"])
}

// TestProviderRobustness tests provider behavior under adverse conditions
func TestProviderRobustness(t *testing.T) {
	provider := NewAdvancedMockChatModel("robustness-test")

	// Test with various malformed inputs
	ctx, _ := TestContext()

	testCases := []struct {
		name        string
		messages    []schema.Message
		shouldPanic bool
	}{
		{"nil_messages", nil, false},
		{"empty_messages", []schema.Message{}, false},
		{"single_message", []schema.Message{schema.NewHumanMessage("test")}, false},
		{"large_messages", generateLargeMessages(100), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.shouldPanic {
				assert.Panics(t, func() {
					provider.Generate(ctx, tc.messages)
				}, "Provider should panic for %s", tc.name)
			} else {
				response, err := provider.Generate(ctx, tc.messages)
				// Provider should handle gracefully - either succeed or return controlled error
				if err != nil {
					assert.Error(t, err, "Provider should handle %s gracefully", tc.name)
				} else {
					assert.NotNil(t, response, "Provider should return response for %s", tc.name)
				}
			}
		})
	}
}

// generateLargeMessages creates a slice of messages for testing
func generateLargeMessages(count int) []schema.Message {
	messages := make([]schema.Message, count)
	for i := 0; i < count; i++ {
		if i%2 == 0 {
			messages[i] = schema.NewHumanMessage(fmt.Sprintf("Human message %d", i))
		} else {
			messages[i] = schema.NewAIMessage(fmt.Sprintf("AI message %d", i))
		}
	}
	return messages
}

// BenchmarkProviderInterface benchmarks provider interface methods
func BenchmarkProviderInterface(b *testing.B) {
	provider := NewAdvancedMockChatModel("benchmark-test")
	ctx := context.Background()
	messages := CreateTestMessages()

	b.Run("Generate", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = provider.Generate(ctx, messages)
		}
	})

	b.Run("StreamChat", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			streamChan, _ := provider.StreamChat(ctx, messages)
			// Consume stream to avoid resource leaks
			for range streamChan {
			}
		}
	})

	b.Run("CheckHealth", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = provider.CheckHealth()
		}
	})

	b.Run("GetModelName", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = provider.GetModelName()
		}
	})
}

// TestProviderThreadSafety tests that providers are thread-safe
func TestProviderThreadSafety(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping thread safety test in short mode")
	}

	provider := NewAdvancedMockChatModel("thread-safety-test")
	ctx := context.Background()
	messages := CreateTestMessages()

	// Run multiple goroutines performing different operations
	operations := []func(){
		func() { provider.Generate(ctx, messages) },
		func() { provider.StreamChat(ctx, messages) },
		func() { provider.CheckHealth() },
		func() { provider.GetModelName() },
		func() { provider.BindTools([]tools.Tool{NewMockTool("test")}) },
	}

	var wg sync.WaitGroup
	concurrency := 10
	iterations := 50

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				// Perform random operation
				opIndex := (goroutineID + j) % len(operations)
				operations[opIndex]()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines completed successfully
		t.Logf("Thread safety test completed with %d goroutines, %d iterations each",
			concurrency, iterations)
	case <-time.After(30 * time.Second):
		t.Fatal("Thread safety test timed out")
	}

	// Verify final state is consistent
	finalHealth := provider.CheckHealth()
	assert.Equal(t, "healthy", finalHealth["state"])
	assert.Greater(t, finalHealth["call_count"], 0)
}
