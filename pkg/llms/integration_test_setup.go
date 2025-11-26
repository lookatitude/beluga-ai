// Package llms provides integration test setup utilities for testing with real LLM providers.
// This file contains utilities to help set up and run integration tests safely.
//
// IMPORTANT: These tests require real API keys and will make actual API calls.
// They should only be run in CI/CD environments or with explicit permission.
//
// Usage:
//
//	go test -tags=integration ./pkg/llms/...
package llms

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// IntegrationTestConfig holds configuration for integration tests.
type IntegrationTestConfig struct {
	// Provider configurations
	AnthropicAPIKey string
	OpenAIAPIKey    string
	BedrockRegion   string
	OllamaBaseURL   string

	// Test settings
	Timeout       time.Duration
	MaxRetries    int
	SkipExpensive bool // Skip tests that cost money or take long time
	Verbose       bool

	// Safety limits
	MaxTokensPerTest int
	MaxRequests      int
	RequestDelay     time.Duration
}

// DefaultIntegrationTestConfig returns a default configuration for integration tests.
func DefaultIntegrationTestConfig() *IntegrationTestConfig {
	return &IntegrationTestConfig{
		Timeout:          30 * time.Second,
		MaxRetries:       2,
		SkipExpensive:    true,
		Verbose:          false,
		MaxTokensPerTest: 1000,
		MaxRequests:      10,
		RequestDelay:     100 * time.Millisecond,
	}
}

// LoadIntegrationTestConfig loads configuration from environment variables.
func LoadIntegrationTestConfig() *IntegrationTestConfig {
	config := DefaultIntegrationTestConfig()

	// Load API keys from environment
	config.AnthropicAPIKey = os.Getenv("ANTHROPIC_API_KEY")
	config.OpenAIAPIKey = os.Getenv("OPENAI_API_KEY")
	config.BedrockRegion = os.Getenv("AWS_REGION")
	if config.BedrockRegion == "" {
		config.BedrockRegion = "us-east-1"
	}
	config.OllamaBaseURL = os.Getenv("OLLAMA_BASE_URL")
	if config.OllamaBaseURL == "" {
		config.OllamaBaseURL = "http://localhost:11434"
	}

	// Load test settings
	if timeout := os.Getenv("INTEGRATION_TEST_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			config.Timeout = d
		}
	}

	if skip := os.Getenv("SKIP_EXPENSIVE_TESTS"); skip == "false" {
		config.SkipExpensive = false
	}

	if verbose := os.Getenv("INTEGRATION_TEST_VERBOSE"); verbose == "true" {
		config.Verbose = true
	}

	return config
}

// IntegrationTestHelper provides utilities for integration testing.
type IntegrationTestHelper struct {
	lastRequest  time.Time
	config       *IntegrationTestConfig
	factory      *Factory
	providers    map[string]iface.ChatModel
	requestCount int
}

// NewIntegrationTestHelper creates a new integration test helper.
func NewIntegrationTestHelper() *IntegrationTestHelper {
	config := LoadIntegrationTestConfig()
	factory := NewFactory()

	return &IntegrationTestHelper{
		config:       config,
		factory:      factory,
		providers:    make(map[string]iface.ChatModel),
		requestCount: 0,
	}
}

// SetupProvider sets up a provider for integration testing.
func (h *IntegrationTestHelper) SetupProvider(providerName string, config *Config) (iface.ChatModel, error) {
	if cached, exists := h.providers[providerName]; exists {
		return cached, nil
	}

	provider, err := h.factory.CreateProvider(providerName, config)
	if err != nil {
		return nil, err
	}

	h.providers[providerName] = provider
	return provider, nil
}

// SetupAnthropicProvider sets up an Anthropic provider.
func (h *IntegrationTestHelper) SetupAnthropicProvider(modelName string) (iface.ChatModel, error) {
	if h.config.AnthropicAPIKey == "" {
		return nil, errors.New("ANTHROPIC_API_KEY not set")
	}

	config := NewConfig(
		WithProvider("anthropic"),
		WithModelName(modelName),
		WithAPIKey(h.config.AnthropicAPIKey),
		WithTimeout(h.config.Timeout),
		WithMaxTokensConfig(h.config.MaxTokensPerTest),
		WithRetryConfig(h.config.MaxRetries, time.Second, 2.0),
	)

	return h.SetupProvider("anthropic", config)
}

// SetupOpenAIProvider sets up an OpenAI provider.
func (h *IntegrationTestHelper) SetupOpenAIProvider(modelName string) (iface.ChatModel, error) {
	if h.config.OpenAIAPIKey == "" {
		return nil, errors.New("OPENAI_API_KEY not set")
	}

	config := NewConfig(
		WithProvider("openai"),
		WithModelName(modelName),
		WithAPIKey(h.config.OpenAIAPIKey),
		WithTimeout(h.config.Timeout),
		WithMaxTokensConfig(h.config.MaxTokensPerTest),
		WithRetryConfig(h.config.MaxRetries, time.Second, 2.0),
	)

	return h.SetupProvider("openai", config)
}

// SetupBedrockProvider sets up an AWS Bedrock provider.
func (h *IntegrationTestHelper) SetupBedrockProvider(modelName string) (iface.ChatModel, error) {
	config := NewConfig(
		WithProvider("bedrock"),
		WithModelName(modelName),
		WithProviderSpecific("region", h.config.BedrockRegion),
		WithTimeout(h.config.Timeout),
		WithMaxTokensConfig(h.config.MaxTokensPerTest),
		WithRetryConfig(h.config.MaxRetries, time.Second, 2.0),
	)

	return h.SetupProvider("bedrock", config)
}

// SetupOllamaProvider sets up an Ollama provider.
func (h *IntegrationTestHelper) SetupOllamaProvider(modelName string) (iface.ChatModel, error) {
	config := NewConfig(
		WithProvider("ollama"),
		WithModelName(modelName),
		WithProviderSpecific("base_url", h.config.OllamaBaseURL),
		WithTimeout(h.config.Timeout),
		WithMaxTokensConfig(h.config.MaxTokensPerTest),
		WithRetryConfig(h.config.MaxRetries, time.Second, 2.0),
	)

	return h.SetupProvider("ollama", config)
}

// RateLimit enforces rate limiting between requests.
func (h *IntegrationTestHelper) RateLimit() {
	h.requestCount++

	if h.config.RequestDelay > 0 {
		elapsed := time.Since(h.lastRequest)
		if elapsed < h.config.RequestDelay {
			time.Sleep(h.config.RequestDelay - elapsed)
		}
	}

	h.lastRequest = time.Now()

	if h.requestCount >= h.config.MaxRequests {
		panic(fmt.Sprintf("Integration test exceeded maximum requests (%d)", h.config.MaxRequests))
	}
}

// SetupMockProvider sets up a mock provider for integration testing.
func (h *IntegrationTestHelper) SetupMockProvider(providerName, modelName string, opts ...any) iface.ChatModel {
	// Create a mock provider directly using the test utilities
	mockOpts := make([]MockOption, 0, len(opts))
	for _, opt := range opts {
		if mockOpt, ok := opt.(MockOption); ok {
			mockOpts = append(mockOpts, mockOpt)
		}
	}

	mockProvider := NewAdvancedMockChatModel(modelName, mockOpts...)

	// Register it with the factory for consistency
	h.factory.RegisterProvider(providerName, mockProvider)

	return mockProvider
}

// GetFactory returns the factory.
func (h *IntegrationTestHelper) GetFactory() *Factory {
	return h.factory
}

// GetConfig returns the test configuration as *Config for compatibility.
func (h *IntegrationTestHelper) GetConfig() *Config {
	// Convert IntegrationTestConfig to Config for factory compatibility
	return NewConfig(
		WithProvider("mock"),
		WithModelName("test-model"),
		WithAPIKey("test-key"),
		WithTimeout(h.config.Timeout),
		WithMaxTokensConfig(h.config.MaxTokensPerTest),
		WithRetryConfig(h.config.MaxRetries, time.Second, 2.0),
	)
}

// GetMetrics returns a mock metrics recorder.
func (h *IntegrationTestHelper) GetMetrics() *MockMetricsRecorder {
	return NewMockMetricsRecorder()
}

// GetTracing returns a mock tracing helper.
func (h *IntegrationTestHelper) GetTracing() *MockTracingHelper {
	return NewMockTracingHelper()
}

// TestProviderIntegration tests a provider with basic integration tests.
func (h *IntegrationTestHelper) TestProviderIntegration(t *testing.T, provider iface.ChatModel, providerName string) {
	if h.config.Verbose {
		t.Logf("Testing provider: %s", providerName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.config.Timeout)
	defer cancel()

	// Test basic generation
	t.Run("basic_generation", func(t *testing.T) {
		h.RateLimit()

		messages := []schema.Message{
			schema.NewSystemMessage("You are a helpful assistant. Keep responses brief."),
			schema.NewHumanMessage("Say 'Hello, World!' and nothing else."),
		}

		response, err := provider.Generate(ctx, messages)
		require.NoError(t, err, "Generate should not error")
		assert.NotNil(t, response, "Response should not be nil")
		assert.NotEmpty(t, response.GetContent(), "Response should have content")

		if h.config.Verbose {
			t.Logf("Response: %s", response.GetContent())
		}

		// Verify response contains expected content
		content := strings.ToLower(response.GetContent())
		assert.Contains(t, content, "hello", "Response should contain greeting")
		assert.Contains(t, content, "world", "Response should contain world")
	})

	// Test streaming (only if not skipping expensive tests)
	if !h.config.SkipExpensive {
		t.Run("streaming", func(t *testing.T) {
			h.RateLimit()

			messages := []schema.Message{
				schema.NewHumanMessage("Count from 1 to 3 slowly."),
			}

			streamChan, err := provider.StreamChat(ctx, messages)
			require.NoError(t, err, "StreamChat should not error")

			var collectedContent strings.Builder
			chunkCount := 0

			for chunk := range streamChan {
				chunkCount++
				if chunk.Err != nil {
					t.Fatalf("Stream error: %v", chunk.Err)
				}
				_, _ = collectedContent.WriteString(chunk.Content)
			}

			assert.Positive(t, chunkCount, "Should receive at least one chunk")
			assert.NotEmpty(t, collectedContent.String(), "Should collect some content")

			if h.config.Verbose {
				t.Logf("Streaming response: %s", collectedContent.String())
			}
		})
	}

	// Test batch processing (only if not skipping expensive tests)
	if !h.config.SkipExpensive {
		t.Run("batch_processing", func(t *testing.T) {
			h.RateLimit()

			inputs := []any{
				[]schema.Message{schema.NewHumanMessage("What is 2+2?")},
				[]schema.Message{schema.NewHumanMessage("What is the color of the sky?")},
				[]schema.Message{schema.NewHumanMessage("Name one programming language.")},
			}

			results, err := provider.Batch(ctx, inputs)
			require.NoError(t, err, "Batch should not error")
			assert.Len(t, results, len(inputs), "Should return correct number of results")

			for i, result := range results {
				assert.NotNil(t, result, "Result %d should not be nil", i)
				if msg, ok := result.(schema.Message); ok {
					assert.NotEmpty(t, msg.GetContent(), "Result %d should have content", i)
				}
			}
		})
	}

	// Test health check
	t.Run("health_check", func(t *testing.T) {
		health := provider.CheckHealth()
		assert.NotNil(t, health, "Health check should return data")
		assert.Contains(t, health, "state", "Health should contain state")
	})

	// Test Runnable interface
	t.Run("runnable_interface", func(t *testing.T) {
		h.RateLimit()

		result, err := provider.Invoke(ctx, "Say 'test' and nothing else.")
		require.NoError(t, err, "Invoke should not error")
		assert.NotNil(t, result, "Invoke should return result")
	})
}

// TestCrossProviderComparison compares responses from multiple providers.
func (h *IntegrationTestHelper) TestCrossProviderComparison(t *testing.T, providers map[string]iface.ChatModel, testPrompt string) {
	if h.config.SkipExpensive {
		t.Skip("Skipping expensive cross-provider comparison test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.config.Timeout)
	defer cancel()

	messages := []schema.Message{
		schema.NewSystemMessage("You are a helpful assistant. Keep responses brief and factual."),
		schema.NewHumanMessage(testPrompt),
	}

	results := make(map[string]string)

	for name, provider := range providers {
		t.Run("provider_"+name, func(t *testing.T) {
			h.RateLimit()

			response, err := provider.Generate(ctx, messages)
			require.NoError(t, err, "Provider %s should not error", name)

			content := response.GetContent()
			results[name] = content

			assert.NotEmpty(t, content, "Provider %s should return content", name)

			if h.config.Verbose {
				t.Logf("Provider %s response: %s", name, content)
			}
		})
	}

	// Verify that responses are different (providers should give different answers)
	if len(results) > 1 {
		responses := make([]string, 0, len(results))
		for _, resp := range results {
			responses = append(responses, resp)
		}

		// Check if all responses are identical (they shouldn't be)
		allSame := true
		for i := 1; i < len(responses); i++ {
			if responses[i] != responses[0] {
				allSame = false
				break
			}
		}

		assert.False(t, allSame, "Providers should give different responses")
	}
}

// TestProviderErrorHandling tests error handling scenarios.
func (h *IntegrationTestHelper) TestProviderErrorHandling(t *testing.T, provider iface.ChatModel, providerName string) {
	if h.config.SkipExpensive {
		t.Skip("Skipping expensive error handling test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.config.Timeout)
	defer cancel()

	// Test with invalid input that should cause an error
	t.Run("invalid_input_handling", func(t *testing.T) {
		h.RateLimit()

		// Try with empty messages
		_, err := provider.Generate(ctx, []schema.Message{})
		// Note: Some providers might handle empty messages gracefully,
		// so we don't assert on the error here
		if err != nil && h.config.Verbose {
			t.Logf("Provider %s error with empty messages: %v", providerName, err)
		}
	})

	// Test timeout handling
	t.Run("timeout_handling", func(t *testing.T) {
		shortCtx, shortCancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer shortCancel()

		h.RateLimit()

		messages := []schema.Message{
			schema.NewHumanMessage("This should timeout due to very short context"),
		}

		_, err := provider.Generate(shortCtx, messages)
		// Should error due to timeout
		assert.Error(t, err, "Should error due to timeout")
		if h.config.Verbose {
			t.Logf("Provider %s timeout error: %v", providerName, err)
		}
	})
}

// IntegrationTestSuite represents a complete integration test suite.
type IntegrationTestSuite struct {
	SetupFunc   func(t *testing.T) *IntegrationTestHelper
	TestFunc    func(t *testing.T, helper *IntegrationTestHelper)
	Name        string
	Description string
}

// RunIntegrationTestSuite runs a complete integration test suite.
func RunIntegrationTestSuite(t *testing.T, suite IntegrationTestSuite) {
	t.Run(suite.Name, func(t *testing.T) {
		if suite.Description != "" {
			t.Logf("Running integration test suite: %s", suite.Description)
		}

		helper := suite.SetupFunc(t)
		require.NotNil(t, helper, "Setup should return valid helper")

		suite.TestFunc(t, helper)
	})
}

// Example integration test suites

// AnthropicIntegrationTestSuite returns a test suite for Anthropic.
func AnthropicIntegrationTestSuite() IntegrationTestSuite {
	return IntegrationTestSuite{
		Name:        "anthropic_integration",
		Description: "Integration tests for Anthropic Claude models",
		SetupFunc: func(t *testing.T) *IntegrationTestHelper {
			helper := NewIntegrationTestHelper()

			provider, err := helper.SetupAnthropicProvider("claude-3-haiku-20240307")
			require.NoError(t, err, "Should set up Anthropic provider")

			helper.providers["anthropic"] = provider
			return helper
		},
		TestFunc: func(t *testing.T, helper *IntegrationTestHelper) {
			provider := helper.providers["anthropic"]
			helper.TestProviderIntegration(t, provider, "anthropic")
		},
	}
}

// OpenAIIntegrationTestSuite returns a test suite for OpenAI.
func OpenAIIntegrationTestSuite() IntegrationTestSuite {
	return IntegrationTestSuite{
		Name:        "openai_integration",
		Description: "Integration tests for OpenAI GPT models",
		SetupFunc: func(t *testing.T) *IntegrationTestHelper {
			helper := NewIntegrationTestHelper()

			provider, err := helper.SetupOpenAIProvider("gpt-3.5-turbo")
			require.NoError(t, err, "Should set up OpenAI provider")

			helper.providers["openai"] = provider
			return helper
		},
		TestFunc: func(t *testing.T, helper *IntegrationTestHelper) {
			provider := helper.providers["openai"]
			helper.TestProviderIntegration(t, provider, "openai")
		},
	}
}

// MultiProviderIntegrationTestSuite returns a test suite comparing multiple providers.
func MultiProviderIntegrationTestSuite() IntegrationTestSuite {
	return IntegrationTestSuite{
		Name:        "multi_provider_comparison",
		Description: "Compare responses across multiple LLM providers",
		SetupFunc: func(t *testing.T) *IntegrationTestHelper {
			helper := NewIntegrationTestHelper()

			providers := make(map[string]iface.ChatModel)

			// Try to set up available providers
			if helper.config.AnthropicAPIKey != "" {
				if provider, err := helper.SetupAnthropicProvider("claude-3-haiku-20240307"); err == nil {
					providers["anthropic"] = provider
				}
			}

			if helper.config.OpenAIAPIKey != "" {
				if provider, err := helper.SetupOpenAIProvider("gpt-3.5-turbo"); err == nil {
					providers["openai"] = provider
				}
			}

			require.NotEmpty(t, providers, "At least one provider should be available")

			helper.providers = providers
			return helper
		},
		TestFunc: func(t *testing.T, helper *IntegrationTestHelper) {
			// Test each provider individually
			for name, provider := range helper.providers {
				t.Run("individual_"+name, func(t *testing.T) {
					helper.TestProviderIntegration(t, provider, name)
				})
			}

			// Test cross-provider comparison
			if len(helper.providers) > 1 {
				t.Run("cross_provider_comparison", func(t *testing.T) {
					helper.TestCrossProviderComparison(
						t,
						helper.providers,
						"Explain quantum computing in one sentence.",
					)
				})
			}
		},
	}
}

// OllamaIntegrationTestSuite returns a test suite for Ollama.
func OllamaIntegrationTestSuite() IntegrationTestSuite {
	return IntegrationTestSuite{
		Name:        "ollama_integration",
		Description: "Integration tests for local Ollama models",
		SetupFunc: func(t *testing.T) *IntegrationTestHelper {
			helper := NewIntegrationTestHelper()

			provider, err := helper.SetupOllamaProvider("llama2")
			require.NoError(t, err, "Should set up Ollama provider")

			helper.providers["ollama"] = provider
			return helper
		},
		TestFunc: func(t *testing.T, helper *IntegrationTestHelper) {
			provider := helper.providers["ollama"]
			helper.TestProviderIntegration(t, provider, "ollama")
		},
	}
}

// RunAllIntegrationTests runs all available integration tests.
func RunAllIntegrationTests(t *testing.T) {
	suites := []IntegrationTestSuite{}

	config := LoadIntegrationTestConfig()

	// Add suites based on available API keys
	if config.AnthropicAPIKey != "" {
		suites = append(suites, AnthropicIntegrationTestSuite())
	}

	if config.OpenAIAPIKey != "" {
		suites = append(suites, OpenAIIntegrationTestSuite())
	}

	// Only add multi-provider if we have multiple providers
	if (config.AnthropicAPIKey != "" && config.OpenAIAPIKey != "") ||
		(config.AnthropicAPIKey != "" && config.OllamaBaseURL != "") ||
		(config.OpenAIAPIKey != "" && config.OllamaBaseURL != "") {
		suites = append(suites, MultiProviderIntegrationTestSuite())
	}

	if config.OllamaBaseURL != "" {
		suites = append(suites, OllamaIntegrationTestSuite())
	}

	if len(suites) == 0 {
		t.Skip("No API keys configured for integration tests")
	}

	for _, suite := range suites {
		t.Run(suite.Name, func(t *testing.T) {
			RunIntegrationTestSuite(t, suite)
		})
	}
}

// Example usage:
//
// func TestIntegration(t *testing.T) {
//     if testing.Short() {
//         t.Skip("Skipping integration tests in short mode")
//     }
//
//     // Run all available integration tests
//     RunAllIntegrationTests(t)
// }
//
// func TestAnthropicIntegration(t *testing.T) {
//     if testing.Short() {
//         t.Skip("Skipping integration tests in short mode")
//     }
//
//     suite := AnthropicIntegrationTestSuite()
//     RunIntegrationTestSuite(t, suite)
// }
//
// To run integration tests:
//   ANTHROPIC_API_KEY=your_key OPENAI_API_KEY=your_key go test -tags=integration -v ./pkg/llms/...
