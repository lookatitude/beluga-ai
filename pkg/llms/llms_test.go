package llms

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing

// MockChatModel is a mock implementation of ChatModel for testing
type MockChatModel struct {
	mock.Mock
	modelName string
}

func NewMockChatModel(modelName string) *MockChatModel {
	return &MockChatModel{modelName: modelName}
}

func (m *MockChatModel) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	args := m.Called(ctx, messages, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(schema.Message), args.Error(1)
}

func (m *MockChatModel) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan iface.AIMessageChunk, error) {
	args := m.Called(ctx, messages, options)
	return args.Get(0).(<-chan iface.AIMessageChunk), args.Error(1)
}

func (m *MockChatModel) BindTools(toolsToBind []tools.Tool) iface.ChatModel {
	args := m.Called(toolsToBind)
	return args.Get(0).(iface.ChatModel)
}

func (m *MockChatModel) GetModelName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockChatModel) GetProviderName() string {
	return "mock"
}

func (m *MockChatModel) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	args := m.Called(ctx, input, options)
	return args.Get(0), args.Error(1)
}

func (m *MockChatModel) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	args := m.Called(ctx, inputs, options)
	return args.Get(0).([]any), args.Error(1)
}

func (m *MockChatModel) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	args := m.Called(ctx, input, options)
	return args.Get(0).(<-chan any), args.Error(1)
}

func (m *MockChatModel) CheckHealth() map[string]interface{} {
	return map[string]interface{}{
		"state":         "healthy",
		"provider":      "mock",
		"model":         m.modelName,
		"timestamp":     int64(1234567890),
		"call_count":    0,
		"tools_count":   0,
		"should_error":  false,
		"responses_len": 1,
	}
}

// Test cases

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestEnsureMessages(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected []schema.Message
		wantErr  bool
	}{
		{
			name:     "string input",
			input:    "hello world",
			expected: []schema.Message{schema.NewHumanMessage("hello world")},
			wantErr:  false,
		},
		{
			name:     "message input",
			input:    schema.NewHumanMessage("test"),
			expected: []schema.Message{schema.NewHumanMessage("test")},
			wantErr:  false,
		},
		{
			name:     "message slice input",
			input:    []schema.Message{schema.NewHumanMessage("test1"), schema.NewHumanMessage("test2")},
			expected: []schema.Message{schema.NewHumanMessage("test1"), schema.NewHumanMessage("test2")},
			wantErr:  false,
		},
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid input type",
			input:    123,
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EnsureMessages(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFactory(t *testing.T) {
	factory := NewFactory()
	mockModel := NewMockChatModel("test-model")

	// Test registering and getting provider
	factory.RegisterProvider("test", mockModel)

	retrieved, err := factory.GetProvider("test")
	assert.NoError(t, err)
	assert.Equal(t, mockModel, retrieved)

	// Test getting non-existent provider
	_, err = factory.GetProvider("nonexistent")
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "not registered"))

	// Test listing providers
	providers := factory.ListProviders()
	assert.Contains(t, providers, "test")
}

func TestConfiguration(t *testing.T) {
	tests := []struct {
		name      string
		configFn  func() *Config
		shouldErr bool
	}{
		{
			name: "valid config",
			configFn: func() *Config {
				return NewConfig(
					WithProvider("anthropic"),
					WithModelName("claude-3-sonnet"),
					WithAPIKey("test-key"),
				)
			},
			shouldErr: false,
		},
		{
			name: "missing provider",
			configFn: func() *Config {
				return NewConfig(
					WithModelName("claude-3-sonnet"),
					WithAPIKey("test-key"),
				)
			},
			shouldErr: true,
		},
		{
			name: "missing model name",
			configFn: func() *Config {
				return NewConfig(
					WithProvider("anthropic"),
					WithAPIKey("test-key"),
				)
			},
			shouldErr: true,
		},
		{
			name: "mock provider without API key",
			configFn: func() *Config {
				return NewConfig(
					WithProvider("mock"),
					WithModelName("mock-model"),
				)
			},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.configFn()
			err := config.Validate()
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		isLLMError  bool
		isRetryable bool
		errorCode   string
	}{
		{
			name:        "LLM error",
			err:         NewLLMError("test", ErrCodeInvalidRequest, errors.New("test error")),
			isLLMError:  true,
			isRetryable: false,
			errorCode:   ErrCodeInvalidRequest,
		},
		{
			name:        "rate limit error",
			err:         NewLLMError("test", ErrCodeRateLimit, errors.New("rate limited")),
			isLLMError:  true,
			isRetryable: true,
			errorCode:   ErrCodeRateLimit,
		},
		{
			name:        "network error",
			err:         NewLLMError("test", ErrCodeNetworkError, errors.New("network failed")),
			isLLMError:  true,
			isRetryable: true,
			errorCode:   ErrCodeNetworkError,
		},
		{
			name:        "regular error",
			err:         errors.New("regular error"),
			isLLMError:  false,
			isRetryable: true, // Default for unknown errors
			errorCode:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isLLMError, IsLLMError(tt.err))
			assert.Equal(t, tt.isRetryable, IsRetryableError(tt.err))
			assert.Equal(t, tt.errorCode, GetLLMErrorCode(tt.err))
		})
	}
}

func TestGenerateText(t *testing.T) {
	mockModel := NewMockChatModel("test-model")
	expectedResponse := "Test response"

	mockModel.On("Generate", mock.Anything, mock.Anything, mock.Anything).
		Return(schema.NewAIMessage(expectedResponse), nil)

	result, err := GenerateText(context.Background(), mockModel, "Test prompt")

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
	mockModel.AssertExpectations(t)
}

func TestBatchGenerate(t *testing.T) {
	mockModel := NewMockChatModel("test-model")
	prompts := []string{"Prompt 1", "Prompt 2"}
	expectedResponses := []string{"Response 1", "Response 2"}

	// Mock batch processing
	mockModel.On("Batch", mock.Anything, mock.Anything, mock.Anything).
		Return([]any{
			schema.NewAIMessage(expectedResponses[0]),
			schema.NewAIMessage(expectedResponses[1]),
		}, nil)

	results, err := BatchGenerate(context.Background(), mockModel, prompts)

	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, expectedResponses[0], results[0])
	assert.Equal(t, expectedResponses[1], results[1])
	mockModel.AssertExpectations(t)
}

func TestValidateModelName(t *testing.T) {
	tests := []struct {
		name      string
		provider  string
		modelName string
		wantErr   bool
	}{
		{
			name:      "valid OpenAI model",
			provider:  "openai",
			modelName: "gpt-4",
			wantErr:   false,
		},
		{
			name:      "invalid OpenAI model",
			provider:  "openai",
			modelName: "invalid-model",
			wantErr:   true,
		},
		{
			name:      "valid Anthropic model",
			provider:  "anthropic",
			modelName: "claude-3-sonnet",
			wantErr:   false,
		},
		{
			name:      "invalid Anthropic model",
			provider:  "anthropic",
			modelName: "invalid-model",
			wantErr:   true,
		},
		{
			name:      "empty model name",
			provider:  "openai",
			modelName: "",
			wantErr:   true,
		},
		{
			name:      "unknown provider",
			provider:  "unknown",
			modelName: "some-model",
			wantErr:   false, // Should not error for unknown providers
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModelName(tt.provider, tt.modelName)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStreaming(t *testing.T) {
	mockModel := NewMockChatModel("test-model")

	// Create a mock streaming channel
	streamChan := make(chan iface.AIMessageChunk, 2)
	streamChan <- iface.AIMessageChunk{Content: "Hello"}
	streamChan <- iface.AIMessageChunk{Content: " World"}
	close(streamChan)

	mockModel.On("StreamChat", mock.Anything, mock.Anything, mock.Anything).
		Return((<-chan iface.AIMessageChunk)(streamChan), nil)

	resultChan, err := mockModel.StreamChat(context.Background(), []schema.Message{schema.NewHumanMessage("test")})

	assert.NoError(t, err)

	var collectedContent strings.Builder
	for chunk := range resultChan {
		collectedContent.WriteString(chunk.Content)
	}

	assert.Equal(t, "Hello World", collectedContent.String())
	mockModel.AssertExpectations(t)
}

func TestToolBinding(t *testing.T) {
	mockModel := NewMockChatModel("test-model")
	mockTool := &MockTool{name: "calculator"}

	mockModel.On("BindTools", mock.AnythingOfType("[]tools.Tool")).
		Return(mockModel) // Return self for simplicity

	result := mockModel.BindTools([]tools.Tool{mockTool})

	assert.Equal(t, mockModel, result)
	mockModel.AssertExpectations(t)
}

// Note: MockTool is defined in test_utils.go

// MockLLM for testing ChatModelAdapter
type MockLLM struct {
	modelName string
}

func (m *MockLLM) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	return "Mock LLM response", nil
}

func (m *MockLLM) GetModelName() string {
	return m.modelName
}

func (m *MockLLM) GetProviderName() string {
	return "mock"
}

// Add test for unification
func TestChatModelUnification(t *testing.T) {
	mockModel := NewMockChatModel("unification-test")

	mockModel.On("GetModelName").Return("unification-test")
	mockModel.On("GetProviderName").Return("mock")

	// Check if ChatModel can be used as LLM
	var llm iface.LLM = mockModel
	assert.NotNil(t, llm)
	assert.Equal(t, "unification-test", llm.GetModelName())
	assert.Equal(t, "mock", llm.GetProviderName())
}

// Add test for LLM factories
func TestLLMFactories(t *testing.T) {
	tests := []struct {
		name string
		fn   func(opts ...ConfigOption) (iface.LLM, error)
	}{
		{"Anthropic", NewAnthropicLLM},
		{"OpenAI", NewOpenAILLM},
		{"Bedrock", NewBedrockLLM},
		{"Ollama", NewOllamaLLM},
		{"Mock", NewMockLLM},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm, err := tt.fn()
			assert.Error(t, err) // Expected error from dummy implementation
			if llm != nil {
				assert.NotEmpty(t, llm.GetModelName())
				assert.NotEmpty(t, llm.GetProviderName())
			}
		})
	}
}

// Benchmark tests

func BenchmarkEnsureMessages(b *testing.B) {
	input := "benchmark test message"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EnsureMessages(input)
	}
}

func BenchmarkGenerateText(b *testing.B) {
	mockModel := NewMockChatModel("benchmark-model")
	mockModel.On("Generate", mock.Anything, mock.Anything, mock.Anything).
		Return(schema.NewAIMessage("benchmark response"), nil)

	ctx := context.Background()
	prompt := "benchmark prompt"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GenerateText(ctx, mockModel, prompt)
	}
}

// Integration test example
func TestFactoryIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	factory := NewFactory()

	// Test that factory starts empty
	assert.Empty(t, factory.ListProviders())

	// Test provider registration and retrieval
	mockModel := NewMockChatModel("integration-test-model")

	mockModel.On("GetModelName").Return("integration-test-model")

	factory.RegisterProvider("integration-test", mockModel)

	// Get provider and check health
	retrieved, err := factory.GetProvider("integration-test")
	assert.NoError(t, err)

	health := retrieved.CheckHealth()
	assert.NotNil(t, health)
	assert.Equal(t, "healthy", health["state"])
}

// Test configuration with timeout
func TestConfigurationTimeout(t *testing.T) {
	config := NewConfig(
		WithProvider("anthropic"),
		WithModelName("claude-3-sonnet"),
		WithAPIKey("test-key"),
		WithTimeout(5*time.Second),
	)

	assert.Equal(t, 5*time.Second, config.Timeout)

	err := ValidateProviderConfig(context.Background(), config)
	assert.NoError(t, err)
}

// Test error wrapping
func TestErrorWrapping(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := WrapError("test_operation", originalErr)

	assert.Error(t, wrappedErr)
	assert.Contains(t, wrappedErr.Error(), "test_operation")
	assert.Contains(t, wrappedErr.Error(), "original error")

	// Test that it's still accessible as LLMError
	var llmErr *LLMError
	assert.True(t, errors.As(wrappedErr, &llmErr))
	assert.Equal(t, "test_operation", llmErr.Op)
}

// Test CheckHealth functionality
func TestCheckHealth(t *testing.T) {
	mockModel := NewMockChatModel("test-model")

	health := mockModel.CheckHealth()

	assert.NotNil(t, health)
	assert.Equal(t, "healthy", health["state"])
	assert.Equal(t, "mock", health["provider"])
	assert.Equal(t, "test-model", health["model"])
	assert.Contains(t, health, "timestamp")
	assert.Contains(t, health, "call_count")
	assert.Contains(t, health, "tools_count")
	assert.Contains(t, health, "should_error")
	assert.Contains(t, health, "responses_len")
}

// Test factory provider registration and health checks
func TestFactoryProviderHealth(t *testing.T) {
	factory := NewFactory()

	// Register a mock provider
	mockModel := NewMockChatModel("factory-test-model")
	factory.RegisterProvider("factory-test", mockModel)

	// Get provider and check health
	retrieved, err := factory.GetProvider("factory-test")
	assert.NoError(t, err)

	health := retrieved.CheckHealth()
	assert.NotNil(t, health)
	assert.Equal(t, "healthy", health["state"])
	assert.Equal(t, "mock", health["provider"])
}

// Test convenience factory functions
func TestConvenienceFactoryFunctions(t *testing.T) {
	// Test NewAnthropicChat (should return error due to missing provider)
	_, err := NewAnthropicChat(WithAPIKey("test-key"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "anthropic provider not available")

	// Test NewOpenAIChat (should return error due to missing provider)
	_, err = NewOpenAIChat(WithAPIKey("test-key"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "openai provider not available")

	// Test NewOllamaChat (should return error due to missing provider)
	_, err = NewOllamaChat(WithModelName("llama2"), WithAPIKey("dummy"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ollama provider not available")
}

// Test ChatModelAdapter functionality
func TestChatModelAdapter(t *testing.T) {
	mockLLM := &MockLLM{modelName: "adapter-test"}
	adapter := NewChatModelAdapter(mockLLM, &ChatOptions{})

	// Test GetModelInfo
	info := adapter.GetModelInfo()
	assert.Equal(t, "adapter-test", info.Name)
	assert.Equal(t, "mock", info.Provider) // Updated from "unknown"

	// Test CheckHealth
	health := adapter.CheckHealth()
	assert.NotNil(t, health)
	assert.Equal(t, "healthy", health["state"])
	assert.Contains(t, health, "timestamp")
}

// Test configuration with all options
func TestConfigurationComprehensive(t *testing.T) {
	config := NewConfig(
		WithProvider("anthropic"),
		WithModelName("claude-3-sonnet"),
		WithAPIKey("test-key"),
		WithTemperatureConfig(0.7),
		WithTopPConfig(0.9),
		WithMaxTokensConfig(1000),
		WithStopSequences([]string{"STOP", "END"}),
		WithTimeout(30*time.Second),
		WithMaxConcurrentBatches(10),
		WithRetryConfig(5, time.Second, 2.0),
		WithObservability(true, true, true),
		WithToolCalling(true),
		WithProviderSpecific("api_version", "2023-06-01"),
	)

	assert.Equal(t, "anthropic", config.Provider)
	assert.Equal(t, "claude-3-sonnet", config.ModelName)
	assert.Equal(t, "test-key", config.APIKey)
	assert.Equal(t, float32(0.7), *config.Temperature)
	assert.Equal(t, float32(0.9), *config.TopP)
	assert.Equal(t, 1000, *config.MaxTokens)
	assert.Equal(t, []string{"STOP", "END"}, config.StopSequences)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 10, config.MaxConcurrentBatches)
	assert.Equal(t, 5, config.MaxRetries)
	assert.Equal(t, time.Second, config.RetryDelay)
	assert.Equal(t, 2.0, config.RetryBackoff)
	assert.True(t, config.EnableTracing)
	assert.True(t, config.EnableMetrics)
	assert.True(t, config.EnableStructuredLogging)
	assert.True(t, config.EnableToolCalling)
	assert.Equal(t, "2023-06-01", config.ProviderSpecific["api_version"])
}
