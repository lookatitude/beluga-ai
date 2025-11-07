package chatmodels

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Test helper functions and utilities
func createValidConfig(provider string) *Config {
	return &Config{
		DefaultProvider:       provider,
		DefaultTemperature:    0.7,
		DefaultMaxTokens:      1000,
		DefaultTimeout:        30 * time.Second,
		DefaultMaxRetries:     3,
		DefaultRetryDelay:     2 * time.Second,
		MaxConcurrentRequests: 100,
		StreamBufferSize:      100,
		StreamTimeout:         5 * time.Minute,
		EnableMetrics:         true,
		EnableTracing:         true,
		Providers:             make(map[string]interface{}),
	}
}

func createTestMessages(count int) []schema.Message {
	messages := make([]schema.Message, count)
	for i := 0; i < count; i++ {
		if i%2 == 0 {
			messages[i] = schema.NewHumanMessage("Test message " + string(rune('A'+i)))
		} else {
			messages[i] = schema.NewAIMessage("AI response " + string(rune('A'+i)))
		}
	}
	return messages
}

func assertErrorType(t *testing.T, err error, expectedCode string, operation string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error but got none")
	}

	var chatErr *ChatModelError
	if !errors.As(err, &chatErr) {
		t.Fatalf("expected ChatModelError, got %T", err)
	}

	if chatErr.Code != expectedCode {
		t.Errorf("expected error code %s, got %s", expectedCode, chatErr.Code)
	}

	if chatErr.Op != operation {
		t.Errorf("expected operation %s, got %s", operation, chatErr.Op)
	}
}

func assertModelCreated(t *testing.T, model iface.ChatModel, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error creating model: %v", err)
	}
	if model == nil {
		t.Fatal("expected model to be created, got nil")
	}
}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestNewChatModel(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		config        *Config
		opts          []iface.Option
		expectError   bool
		errorCode     string
		validateModel func(t *testing.T, model iface.ChatModel)
	}{
		{
			name:        "valid openai model",
			model:       "gpt-4",
			config:      createValidConfig("openai"),
			opts:        []iface.Option{},
			expectError: false,
		},
		{
			name:        "valid mock model",
			model:       "mock-gpt-4",
			config:      createValidConfig("mock"),
			opts:        []iface.Option{},
			expectError: false,
		},
		{
			name:        "unsupported provider",
			model:       "gpt-4",
			config:      createValidConfig("unsupported"),
			opts:        []iface.Option{},
			expectError: true,
			errorCode:   ErrCodeProviderNotSupported,
		},
		{
			name:        "nil config uses default",
			model:       "mock-gpt-4",
			config:      nil,
			opts:        []iface.Option{},
			expectError: false,
		},
		{
			name:  "with temperature option",
			model: "mock-gpt-4",
			config: func() *Config {
				c := createValidConfig("mock")
				c.DefaultTemperature = 0.5
				return c
			}(),
			opts:        []iface.Option{WithTemperature(0.8)},
			expectError: false,
		},
		{
			name:   "with multiple options",
			model:  "mock-gpt-4",
			config: createValidConfig("mock"),
			opts: []iface.Option{
				WithTemperature(0.9),
				WithMaxTokens(2000),
				WithTopP(0.95),
				WithSystemPrompt("You are a helpful assistant"),
				WithTimeout(60 * time.Second),
			},
			expectError: false,
		},
		{
			name:        "empty model name",
			model:       "",
			config:      createValidConfig("mock"),
			opts:        []iface.Option{},
			expectError: false, // Should work with empty model name
		},
		{
			name:  "invalid config temperature too low",
			model: "mock-gpt-4",
			config: func() *Config {
				c := createValidConfig("mock")
				c.DefaultTemperature = -0.5
				return c
			}(),
			opts:        []iface.Option{},
			expectError: true,
			errorCode:   ErrCodeConfigInvalid,
		},
		{
			name:  "invalid config temperature too high",
			model: "mock-gpt-4",
			config: func() *Config {
				c := createValidConfig("mock")
				c.DefaultTemperature = 2.5
				return c
			}(),
			opts:        []iface.Option{},
			expectError: true,
			errorCode:   ErrCodeConfigInvalid,
		},
		{
			name:  "invalid config zero max tokens",
			model: "mock-gpt-4",
			config: func() *Config {
				c := createValidConfig("mock")
				c.DefaultMaxTokens = 0
				return c
			}(),
			opts:        []iface.Option{},
			expectError: true,
			errorCode:   ErrCodeConfigInvalid,
		},
		{
			name:  "invalid config negative timeout",
			model: "mock-gpt-4",
			config: func() *Config {
				c := createValidConfig("mock")
				c.DefaultTimeout = -1 * time.Second
				return c
			}(),
			opts:        []iface.Option{},
			expectError: true,
			errorCode:   ErrCodeConfigInvalid,
		},
		{
			name:  "invalid config negative max retries",
			model: "mock-gpt-4",
			config: func() *Config {
				c := createValidConfig("mock")
				c.DefaultMaxRetries = -1
				return c
			}(),
			opts:        []iface.Option{},
			expectError: true,
			errorCode:   ErrCodeConfigInvalid,
		},
		{
			name:   "with function calling enabled",
			model:  "mock-gpt-4",
			config: createValidConfig("mock"),
			opts: []iface.Option{
				WithFunctionCalling(true),
			},
			expectError: false,
		},
		{
			name:   "with stop sequences",
			model:  "mock-gpt-4",
			config: createValidConfig("mock"),
			opts: []iface.Option{
				WithStopSequences([]string{"\n", "END", "STOP"}),
			},
			expectError: false,
		},
		{
			name:  "with metrics and tracing disabled",
			model: "mock-gpt-4",
			config: func() *Config {
				c := createValidConfig("mock")
				c.EnableMetrics = false
				c.EnableTracing = false
				return c
			}(),
			opts:        []iface.Option{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := NewChatModel(tt.model, tt.config, tt.opts...)

			if tt.expectError {
				assertErrorType(t, err, tt.errorCode, "creation")
			} else {
				assertModelCreated(t, model, err)

				// Additional validation if provided
				if tt.validateModel != nil {
					tt.validateModel(t, model)
				}

				// Basic interface compliance check
				if model != nil {
					// Test that all required interfaces are implemented
					var _ iface.ChatModel = model
					var _ iface.MessageGenerator = model
					var _ iface.StreamMessageHandler = model
					var _ iface.ModelInfoProvider = model
					var _ iface.HealthChecker = model
				}
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func TestNewOpenAIChatModel(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		apiKey    string
		opts      []iface.Option
		wantError bool
		errorCode string
	}{
		{
			name:      "valid openai model",
			model:     "gpt-4",
			apiKey:    "test-api-key",
			opts:      []iface.Option{},
			wantError: false,
		},
		{
			name:      "with options",
			model:     "gpt-3.5-turbo",
			apiKey:    "test-api-key",
			opts:      []iface.Option{WithTemperature(0.7), WithMaxTokens(1000)},
			wantError: false,
		},
		{
			name:      "empty api key",
			model:     "gpt-4",
			apiKey:    "",
			opts:      []iface.Option{},
			wantError: false, // Currently doesn't validate API key at creation
		},
		{
			name:      "all openai models",
			model:     "gpt-4o",
			apiKey:    "test-api-key",
			opts:      []iface.Option{},
			wantError: false,
		},
		{
			name:   "with comprehensive options",
			model:  "gpt-4-turbo",
			apiKey: "test-api-key",
			opts: []iface.Option{
				WithTemperature(0.8),
				WithMaxTokens(2000),
				WithTopP(0.9),
				WithSystemPrompt("You are a helpful AI assistant"),
				WithFunctionCalling(true),
				WithTimeout(45 * time.Second),
				WithMaxRetries(5),
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := NewOpenAIChatModel(tt.model, tt.apiKey, tt.opts...)

			if tt.wantError {
				assertErrorType(t, err, tt.errorCode, "creation")
			} else {
				assertModelCreated(t, model, err)
			}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		})
	}
}

func TestNewMockChatModel(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		opts      []iface.Option
		wantError bool
	}{
		{
			name:      "valid mock model",
			model:     "mock-gpt-4",
			opts:      []iface.Option{},
			wantError: false,
		},
		{
			name:      "with options",
			model:     "mock-claude",
			opts:      []iface.Option{WithTemperature(0.5), WithMaxTokens(500)},
			wantError: false,
		},
		{
			name:      "all supported mock models",
			model:     "mock-general",
			opts:      []iface.Option{},
			wantError: false,
		},
		{
			name:      "empty model name",
			model:     "",
			opts:      []iface.Option{},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := NewMockChatModel(tt.model, tt.opts...)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				assertModelCreated(t, model, err)
			}
		})
	}
}

func TestChatModel_GenerateMessages(t *testing.T) {
	tests := []struct {
		name             string
		messages         []schema.Message
		opts             []iface.Option
		expectError      bool
		validateResponse func(t *testing.T, response []schema.Message)
	}{
		{
			name: "single message",
			messages: []schema.Message{
				schema.NewHumanMessage("Hello, how are you?"),
			},
			opts:        []iface.Option{},
			expectError: false,
			validateResponse: func(t *testing.T, response []schema.Message) {
				if len(response) == 0 {
					t.Fatal("Expected at least one response message")
				}
				if len(response) != 1 {
					t.Errorf("Expected 1 response message, got %d", len(response))
				}
			},
		},
		{
			name:        "multiple messages",
			messages:    createTestMessages(3),
			opts:        []iface.Option{},
			expectError: false,
			validateResponse: func(t *testing.T, response []schema.Message) {
				if len(response) == 0 {
					t.Fatal("Expected at least one response message")
				}
			},
		},
		{
			name:        "empty messages",
			messages:    []schema.Message{},
			opts:        []iface.Option{},
			expectError: false,
			validateResponse: func(t *testing.T, response []schema.Message) {
				if len(response) == 0 {
					t.Fatal("Expected at least one response message")
				}
			},
		},
		{
			name: "with temperature option",
			messages: []schema.Message{
				schema.NewHumanMessage("Test with temperature"),
			},
			opts:        []iface.Option{WithTemperature(0.5)},
			expectError: false,
			validateResponse: func(t *testing.T, response []schema.Message) {
				if len(response) == 0 {
					t.Fatal("Expected at least one response message")
				}
			},
		},
		{
			name: "with max tokens option",
			messages: []schema.Message{
				schema.NewHumanMessage("Test with max tokens"),
			},
			opts:        []iface.Option{WithMaxTokens(50)},
			expectError: false,
			validateResponse: func(t *testing.T, response []schema.Message) {
				if len(response) == 0 {
					t.Fatal("Expected at least one response message")
				}
			},
		},
		{
			name: "with system prompt",
			messages: []schema.Message{
				schema.NewHumanMessage("Test with system prompt"),
			},
			opts:        []iface.Option{WithSystemPrompt("You are a helpful assistant")},
			expectError: false,
			validateResponse: func(t *testing.T, response []schema.Message) {
				if len(response) == 0 {
					t.Fatal("Expected at least one response message")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := NewMockChatModel("test-model")
			if err != nil {
				t.Fatalf("Failed to create mock model: %v", err)
			}

			ctx := context.Background()
			// Convert iface.Option to core.Option
			coreOpts := make([]core.Option, len(tt.opts))
			for i, opt := range tt.opts {
				coreOpts[i] = opt
			}
			response, err := model.GenerateMessages(ctx, tt.messages, coreOpts...)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("GenerateMessages failed: %v", err)
				}
				if tt.validateResponse != nil {
					tt.validateResponse(t, response)
				}
			}
		})
	}
}

func TestChatModel_StreamMessages(t *testing.T) {
	tests := []struct {
		name           string
		messages       []schema.Message
		opts           []iface.Option
		timeout        time.Duration
		expectError    bool
		validateStream func(t *testing.T, messages []schema.Message)
	}{
		{
			name: "basic streaming",
			messages: []schema.Message{
				schema.NewHumanMessage("Hello, streaming test"),
			},
			opts:        []iface.Option{},
			timeout:     5 * time.Second,
			expectError: false,
			validateStream: func(t *testing.T, messages []schema.Message) {
				if len(messages) == 0 {
					t.Fatal("Expected at least one streamed message")
				}
				// Check that we have some content
				var fullContent strings.Builder
				for _, msg := range messages {
					fullContent.WriteString(msg.GetContent())
				}
				if fullContent.Len() == 0 {
					t.Error("Expected non-empty streamed content")
				}
			},
		},
		{
			name:        "streaming with multiple messages",
			messages:    createTestMessages(2),
			opts:        []iface.Option{},
			timeout:     3 * time.Second,
			expectError: false,
			validateStream: func(t *testing.T, messages []schema.Message) {
				if len(messages) == 0 {
					t.Fatal("Expected at least one streamed message")
				}
			},
		},
		{
			name: "streaming with temperature",
			messages: []schema.Message{
				schema.NewHumanMessage("Test streaming with temperature"),
			},
			opts:        []iface.Option{WithTemperature(0.8)},
			timeout:     3 * time.Second,
			expectError: false,
			validateStream: func(t *testing.T, messages []schema.Message) {
				if len(messages) == 0 {
					t.Fatal("Expected at least one streamed message")
				}
			},
		},
		{
			name: "streaming with short timeout",
			messages: []schema.Message{
				schema.NewHumanMessage("Test with short timeout"),
			},
			opts:        []iface.Option{},
			timeout:     100 * time.Millisecond, // Reasonable short timeout
			expectError: false,                  // Should still work
			validateStream: func(t *testing.T, messages []schema.Message) {
				if len(messages) == 0 {
					t.Fatal("Expected at least one streamed message")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := NewMockChatModel("test-model")
			if err != nil {
				t.Fatalf("Failed to create mock model: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			// Convert iface.Option to core.Option
			coreOpts := make([]core.Option, len(tt.opts))
			for i, opt := range tt.opts {
				coreOpts[i] = opt
			}
			stream, err := model.StreamMessages(ctx, tt.messages, coreOpts...)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("StreamMessages failed: %v", err)
				}

				var receivedMessages []schema.Message
				for msg := range stream {
					receivedMessages = append(receivedMessages, msg)
				}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				if tt.validateStream != nil {
					tt.validateStream(t, receivedMessages)
				}
			}
		})
	}
}

func TestChatModel_GetModelInfo(t *testing.T) {
	tests := []struct {
		name         string
		modelName    string
		validateInfo func(t *testing.T, info iface.ModelInfo)
	}{
		{
			name:      "mock model info",
			modelName: "test-model",
			validateInfo: func(t *testing.T, info iface.ModelInfo) {
				if info.Name != "test-model" {
					t.Errorf("Expected model name 'test-model', got '%s'", info.Name)
				}
				if info.Provider != "mock" {
					t.Errorf("Expected provider 'mock', got '%s'", info.Provider)
				}
				if info.MaxTokens <= 0 {
					t.Errorf("Expected positive MaxTokens, got %d", info.MaxTokens)
				}
				if len(info.Capabilities) == 0 {
					t.Error("Expected at least one capability")
				}
				if info.Version == "" {
					t.Error("Expected non-empty version")
				}
			},
		},
		{
			name:      "empty model name",
			modelName: "",
			validateInfo: func(t *testing.T, info iface.ModelInfo) {
				if info.Provider != "mock" {
					t.Errorf("Expected provider 'mock', got '%s'", info.Provider)
				}
				if len(info.Capabilities) == 0 {
					t.Error("Expected at least one capability")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := NewMockChatModel(tt.modelName)
			if err != nil {
				t.Fatalf("Failed to create mock model: %v", err)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			}

			info := model.GetModelInfo()
			if tt.validateInfo != nil {
				tt.validateInfo(t, info)
			}
		})
	}
}

func TestChatModel_CheckHealth(t *testing.T) {
	tests := []struct {
		name           string
		modelName      string
		validateHealth func(t *testing.T, health map[string]interface{})
	}{
		{
			name:      "mock model health",
			modelName: "test-model",
			validateHealth: func(t *testing.T, health map[string]interface{}) {
				if health == nil {
					t.Fatal("Expected health map to be non-nil")
				}

				status, ok := health["status"]
				if !ok {
					t.Error("Expected 'status' key in health map")
				}

				if status != "healthy" {
					t.Errorf("Expected status 'healthy', got '%v'", status)
				}

				if _, ok := health["model"]; !ok {
					t.Error("Expected 'model' key in health map")
				}

				if _, ok := health["provider"]; !ok {
					t.Error("Expected 'provider' key in health map")
				}

				if _, ok := health["last_check"]; !ok {
					t.Error("Expected 'last_check' key in health map")
				}
			},
		},
		{
			name:      "different model health",
			modelName: "another-model",
			validateHealth: func(t *testing.T, health map[string]interface{}) {
				if health == nil {
					t.Fatal("Expected health map to be non-nil")
				}

				status, ok := health["status"]
				if !ok {
					t.Error("Expected 'status' key in health map")
				}

				if status != "healthy" {
					t.Errorf("Expected status 'healthy', got '%v'", status)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := NewMockChatModel(tt.modelName)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			if err != nil {
				t.Fatalf("Failed to create mock model: %v", err)
			}

			health := model.CheckHealth()
			if tt.validateHealth != nil {
				tt.validateHealth(t, health)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
	}{
		{
			name:      "valid default config",
			config:    DefaultConfig(),
			wantError: false,
		},
		{
			name: "invalid temperature too low",
			config: &Config{
				DefaultTemperature: -0.1,
				DefaultMaxTokens:   1000,
				DefaultTimeout:     time.Second,
				DefaultMaxRetries:  3,
			},
			wantError: true,
		},
		{
			name: "invalid temperature too high",
			config: &Config{
				DefaultTemperature: 2.1,
				DefaultMaxTokens:   1000,
				DefaultTimeout:     time.Second,
				DefaultMaxRetries:  3,
			},
			wantError: true,
		},
		{
			name: "invalid max tokens zero",
			config: &Config{
				DefaultTemperature: 0.7,
				DefaultMaxTokens:   0,
				DefaultTimeout:     time.Second,
				DefaultMaxRetries:  3,
			},
			wantError: true,
		},
		{
			name: "invalid timeout zero",
			config: &Config{
				DefaultTemperature: 0.7,
				DefaultMaxTokens:   1000,
				DefaultTimeout:     0,
				DefaultMaxRetries:  3,
			},
			wantError: true,
		},
		{
			name: "invalid max retries negative",
			config: &Config{
				DefaultTemperature: 0.7,
				DefaultMaxTokens:   1000,
				DefaultTimeout:     time.Second,
				DefaultMaxRetries:  -1,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			err := tt.config.Validate()
			if tt.wantError {
				if err == nil {
					t.Error("expected validation error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected validation error: %v", err)
				}
			}
		})
	}
}

func TestGetSupportedProviders(t *testing.T) {
	providers := GetSupportedProviders()

	expectedProviders := []string{"openai", "mock"}
	if len(providers) != len(expectedProviders) {
		t.Errorf("Expected %d providers, got %d", len(expectedProviders), len(providers))
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	}

	for _, expected := range expectedProviders {
		found := false
		for _, provider := range providers {
			if provider == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected provider '%s' not found in supported providers", expected)
		}
	}
}

func TestGetSupportedModels(t *testing.T) {
	tests := []struct {
		provider       string
		expectedModels []string
	}{
		{
			provider:       "openai",
			expectedModels: []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo", "gpt-4o", "gpt-4o-mini"},
		},
		{
			provider:       "mock",
			expectedModels: []string{"mock-gpt-4", "mock-claude", "mock-general"},
		},
		{
			provider:       "unknown",
			expectedModels: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			models := GetSupportedModels(tt.provider)

			if len(models) != len(tt.expectedModels) {
				t.Errorf("Expected %d models for provider %s, got %d",
					len(tt.expectedModels), tt.provider, len(models))
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			}

			for _, expected := range tt.expectedModels {
				found := false
				for _, model := range models {
					if model == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected model '%s' not found for provider %s", expected, tt.provider)
				}
			}
		})
	}
}

func TestFunctionalOptions(t *testing.T) {
	// Test that functional options work correctly
	config := DefaultConfig()
	config.DefaultProvider = "mock"

	model, err := NewChatModel("test-model", config,
		WithTemperature(0.8),
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		WithMaxTokens(2000),
		WithTopP(0.9),
		WithStopSequences([]string{"\n", "END"}),
		WithSystemPrompt("You are a helpful assistant"),
		WithFunctionCalling(true),
		WithTimeout(30*time.Second),
		WithMaxRetries(5),
		WithMetrics(true),
		WithTracing(true),
	)

	if err != nil {
		t.Fatalf("Failed to create model with options: %v", err)
	}

	if model == nil {
		t.Fatal("Expected model to be created")
	}
}

func TestErrorHandling(t *testing.T) {
	// Test unsupported provider error with valid config
	_, err := NewChatModel("gpt-4", &Config{
		DefaultProvider:       "unsupported",
		DefaultTemperature:    0.7,
		DefaultMaxTokens:      1000,
		DefaultTimeout:        time.Second,
		DefaultMaxRetries:     3,
		DefaultRetryDelay:     time.Second,
		MaxConcurrentRequests: 100,
		StreamBufferSize:      100,
		StreamTimeout:         time.Minute,
		Providers:             make(map[string]interface{}),
	})

	if err == nil {
		t.Fatal("Expected error for unsupported provider")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	var chatErr *ChatModelError
	if !errors.As(err, &chatErr) {
		t.Fatalf("Expected ChatModelError, got %T", err)
	}

	if chatErr.Code != ErrCodeProviderNotSupported {
		t.Errorf("Expected error code %s, got %s", ErrCodeProviderNotSupported, chatErr.Code)
	}

	if chatErr.Op != "creation" {
		t.Errorf("Expected operation 'creation', got '%s'", chatErr.Op)
	}

	if chatErr.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", chatErr.Model)
	}

	if chatErr.Provider != "unsupported" {
		t.Errorf("Expected provider 'unsupported', got '%s'", chatErr.Provider)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "rate limit error",
			err:      NewChatModelError("test", "model", "provider", ErrCodeRateLimit, errors.New("rate limited")),
			expected: true,
		},
		{
			name:     "network error",
			err:      NewChatModelError("test", "model", "provider", ErrCodeNetworkError, errors.New("network failed")),
			expected: true,
		},
		{
			name:     "timeout error",
			err:      NewChatModelError("test", "model", "provider", ErrCodeTimeout, errors.New("timed out")),
			expected: true,
		},
		{
			name:     "authentication error",
			err:      NewChatModelError("test", "model", "provider", ErrCodeAuthentication, errors.New("auth failed")),
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			expected: false,
		},
		{
			name:     "standard timeout error",
			err:      ErrTimeout,
			expected: true,
		},
		{
			name:     "standard rate limit error",
			err:      ErrRateLimitExceeded,
			expected: true,
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryable(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsValidationError(t *testing.T) {
	validErr := NewValidationError("field", "message")
	invalidErr := errors.New("regular error")
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	if !IsValidationError(validErr) {
		t.Error("Expected validation error to be recognized")
	}

	if IsValidationError(invalidErr) {
		t.Error("Expected regular error to not be recognized as validation error")
	}
}

func TestIsAuthenticationError(t *testing.T) {
	authErr := NewChatModelError("test", "model", "provider", ErrCodeAuthentication, errors.New("auth failed"))
	regularErr := NewChatModelError("test", "model", "provider", ErrCodeNetworkError, errors.New("network failed"))
	standardAuthErr := ErrAuthenticationFailed

	if !IsAuthenticationError(authErr) {
		t.Error("Expected authentication error to be recognized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	if IsAuthenticationError(regularErr) {
		t.Error("Expected non-auth error to not be recognized as auth error")
	}

	if !IsAuthenticationError(standardAuthErr) {
		t.Error("Expected standard auth error to be recognized")
	}
}

func TestHealthCheck(t *testing.T) {
	model, err := NewMockChatModel("test-model")
	if err != nil {
		t.Fatalf("Failed to create mock model: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	health := HealthCheck(model)
	if health == nil {
		t.Fatal("Expected health map to be non-nil")
	}

	status, exists := health["status"]
	if !exists {
		t.Error("Expected 'status' key in health map")
	}

	if status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%v'", status)
	}
}

func TestGetModelInfo(t *testing.T) {
	model, err := NewMockChatModel("test-model")
	if err != nil {
		t.Fatalf("Failed to create mock model: %v", err)
	}

	info := GetModelInfo(model)

	if info.Name != "test-model" {
		t.Errorf("Expected model name 'test-model', got '%s'", info.Name)
	}

	if info.Provider != "mock" {
		t.Errorf("Expected provider 'mock', got '%s'", info.Provider)
	}
}

func TestGenerateMessages(t *testing.T) {
	model, err := NewMockChatModel("test-model")
	if err != nil {
		t.Fatalf("Failed to create mock model: %v", err)
	}

	ctx := context.Background()
	messages := []schema.Message{
		schema.NewHumanMessage("Test message"),
	}

	response, err := GenerateMessages(ctx, model, messages)
	if err != nil {
		t.Fatalf("GenerateMessages failed: %v", err)
	}

	if len(response) == 0 {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		t.Fatal("Expected at least one response message")
	}
}

func TestStreamMessages(t *testing.T) {
	model, err := NewMockChatModel("test-model")
	if err != nil {
		t.Fatalf("Failed to create mock model: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	messages := []schema.Message{
		schema.NewHumanMessage("Test streaming"),
	}

	stream, err := StreamMessages(ctx, model, messages)
	if err != nil {
		t.Fatalf("StreamMessages failed: %v", err)
	}

	messageCount := 0
	for range stream {
		messageCount++
	}

	if messageCount == 0 {
		t.Error("Expected at least one streamed message")
	}
}

// New comprehensive test functions for integration test readiness

func TestConcurrentUsage(t *testing.T) {
	model, err := NewMockChatModel("concurrent-test-model")
	if err != nil {
		t.Fatalf("Failed to create mock model: %v", err)
	}

	const numGoroutines = 10
	const numRequestsPerGoroutine = 5

	var wg sync.WaitGroup
	errorChan := make(chan error, numGoroutines*numRequestsPerGoroutine)

	// Launch multiple goroutines making concurrent requests
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numRequestsPerGoroutine; j++ {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				ctx := context.Background()
				messages := []schema.Message{
					schema.NewHumanMessage("Concurrent test message " + string(rune('A'+j))),
				}

				// Test both generate and stream concurrently
				if j%2 == 0 {
					_, err := model.GenerateMessages(ctx, messages)
					if err != nil {
						errorChan <- err
						return
					}
				} else {
					stream, err := model.StreamMessages(ctx, messages)
					if err != nil {
						errorChan <- err
						return
					}
					// Consume the stream
					for range stream {
						// Just consume, don't need to do anything
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	// Check for any errors
	for err := range errorChan {
		t.Errorf("Concurrent request failed: %v", err)
	}
}

func TestContextCancellation(t *testing.T) {
	tests := []struct {
		name       string
		cancelFunc func(ctx context.Context, cancel context.CancelFunc)
	}{
		{
			name: "immediate cancellation",
			cancelFunc: func(ctx context.Context, cancel context.CancelFunc) {
				cancel() // Cancel immediately
			},
		},
		{
			name: "cancellation during streaming",
			cancelFunc: func(ctx context.Context, cancel context.CancelFunc) {
				time.Sleep(10 * time.Millisecond) // Let streaming start
				cancel()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := NewMockChatModel("cancel-test-model")
			if err != nil {
				t.Fatalf("Failed to create mock model: %v", err)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			}

			ctx, cancel := context.WithCancel(context.Background())

			// Start cancellation in a goroutine
			go tt.cancelFunc(ctx, cancel)

			messages := []schema.Message{
				schema.NewHumanMessage("Test cancellation"),
			}

			// Test GenerateMessages with cancellation
			_, err = model.GenerateMessages(ctx, messages)
			if err != nil && !errors.Is(err, context.Canceled) {
				// Error is expected due to cancellation, but check it's the right error
				if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "cancel") {
					t.Logf("Got unexpected error (might be expected): %v", err)
				}
			}

			// Test StreamMessages with fresh cancelled context
			ctx2, cancel2 := context.WithCancel(context.Background())
			cancel2() // Cancel immediately

			stream, err := model.StreamMessages(ctx2, messages)
			if err != nil {
				// Streaming might fail with cancelled context, which is expected
				t.Logf("StreamMessages with cancelled context failed as expected: %v", err)
			} else {
				// If it didn't fail, consume the stream
				for range stream {
					// Just consume
				}
			}
		})
	}
}

func TestErrorScenarios(t *testing.T) {
	tests := []struct {
		name          string
		setupModel    func() (iface.ChatModel, error)
		expectError   bool
		errorContains string
	}{
		{
			name: "normal operation",
			setupModel: func() (iface.ChatModel, error) {
				return NewMockChatModel("normal-model")
			},
			expectError: false,
		},
		{
			name: "invalid provider",
			setupModel: func() (iface.ChatModel, error) {
				return NewChatModel("test", &Config{
					DefaultProvider:       "nonexistent",
					DefaultTemperature:    0.7,
					DefaultMaxTokens:      1000,
					DefaultTimeout:        time.Second,
					DefaultMaxRetries:     3,
					MaxConcurrentRequests: 100,
					StreamBufferSize:      100,
					StreamTimeout:         time.Minute,
					EnableMetrics:         true,
					EnableTracing:         true,
					Providers:             make(map[string]interface{}),
				})
			},
			expectError:   true,
			errorContains: "unsupported provider",
		},
		{
			name: "invalid config",
			setupModel: func() (iface.ChatModel, error) {
				return NewChatModel("test", &Config{
					DefaultProvider:    "mock",
					DefaultTemperature: -1, // Invalid temperature
					DefaultMaxTokens:   1000,
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
					DefaultTimeout:     time.Second,
					DefaultMaxRetries:  3,
				})
			},
			expectError:   true,
			errorContains: "validation error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := tt.setupModel()

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if model == nil {
					t.Fatal("Expected model to be created")
				}

				// Test basic functionality
				ctx := context.Background()
				messages := []schema.Message{schema.NewHumanMessage("test")}
				_, err := model.GenerateMessages(ctx, messages)
				if err != nil {
					t.Errorf("Basic GenerateMessages failed: %v", err)
				}
			}
		})
	}
}

func TestOptionValidation(t *testing.T) {
	model, err := NewMockChatModel("option-test-model")
	if err != nil {
		t.Fatalf("Failed to create mock model: %v", err)
	}

	tests := []struct {
		name      string
		opts      []iface.Option
		expectErr bool
	}{
		{
			name: "valid options",
			opts: []iface.Option{
				WithTemperature(0.7),
				WithMaxTokens(1000),
				WithTopP(0.9),
			},
			expectErr: false,
		},
		{
			name: "extreme but valid temperature",
			opts: []iface.Option{
				WithTemperature(0.0),
				WithTemperature(2.0),
			},
			expectErr: false,
		},
		{
			name: "large max tokens",
			opts: []iface.Option{
				WithMaxTokens(100000),
			},
			expectErr: false,
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		},
		{
			name: "empty stop sequences",
			opts: []iface.Option{
				WithStopSequences([]string{}),
			},
			expectErr: false,
		},
		{
			name: "multiple stop sequences",
			opts: []iface.Option{
				WithStopSequences([]string{"\n", "END", "STOP", "FINISH"}),
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			messages := []schema.Message{schema.NewHumanMessage("test")}

			// Convert options
			coreOpts := make([]core.Option, len(tt.opts))
			for i, opt := range tt.opts {
				coreOpts[i] = opt
			}

			_, err := model.GenerateMessages(ctx, messages, coreOpts...)
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestMetricsAndTracing(t *testing.T) {
	// Test that models can be created with metrics and tracing enabled/disabled
	tests := []struct {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		name          string
		enableMetrics bool
		enableTracing bool
	}{
		{"metrics enabled, tracing enabled", true, true},
		{"metrics enabled, tracing disabled", true, false},
		{"metrics disabled, tracing enabled", false, true},
		{"metrics disabled, tracing disabled", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createValidConfig("mock")
			config.EnableMetrics = tt.enableMetrics
			config.EnableTracing = tt.enableTracing

			model, err := NewChatModel("metrics-test-model", config)
			if err != nil {
				t.Fatalf("Failed to create model: %v", err)
			}

			// Test basic functionality still works
			ctx := context.Background()
			messages := []schema.Message{schema.NewHumanMessage("test")}
			_, err = model.GenerateMessages(ctx, messages)
			if err != nil {
				t.Errorf("GenerateMessages failed: %v", err)
			}

			// Test health check
			health := model.CheckHealth()
			if health == nil {
				t.Error("Expected health check to return data")
			}

			// Test model info
			info := model.GetModelInfo()
			if info.Name == "" {
				t.Error("Expected model info to have name")
			}
		})
	}
}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestRunnableInterface(t *testing.T) {
	model, err := NewMockChatModel("runnable-test-model")
	if err != nil {
		t.Fatalf("Failed to create mock model: %v", err)
	}

	// Test Invoke method
	ctx := context.Background()
	messages := []schema.Message{schema.NewHumanMessage("test invoke")}
	result, err := model.Invoke(ctx, messages)
	if err != nil {
		t.Errorf("Invoke failed: %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result from Invoke")
	}

	// Test Batch method
	batchInputs := []any{
		[]schema.Message{schema.NewHumanMessage("batch test 1")},
		[]schema.Message{schema.NewHumanMessage("batch test 2")},
	}
	batchResults, err := model.Batch(ctx, batchInputs)
	if err != nil {
		t.Errorf("Batch failed: %v", err)
	}
	if len(batchResults) != len(batchInputs) {
		t.Errorf("Expected %d batch results, got %d", len(batchInputs), len(batchResults))
	}

	// Test Stream method
	stream, err := model.Stream(ctx, messages)
	if err != nil {
		t.Errorf("Stream failed: %v", err)
	} else {
		// Consume the stream
		messageCount := 0
		for range stream {
			messageCount++
		}
		if messageCount == 0 {
			t.Error("Expected at least one message from Stream")
		}
	}
}

func TestInterfaceCompliance(t *testing.T) {
	// Test that all implementations properly satisfy their interfaces
	model, err := NewMockChatModel("interface-test-model")
	if err != nil {
		t.Fatalf("Failed to create mock model: %v", err)
	}

	// Compile-time checks (these will fail at compile time if interfaces aren't satisfied)
	var _ iface.ChatModel = model
	var _ iface.MessageGenerator = model
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	var _ iface.StreamMessageHandler = model
	var _ iface.ModelInfoProvider = model
	var _ iface.HealthChecker = model
	var _ core.Runnable = model

	// Runtime checks
	ctx := context.Background()
	messages := []schema.Message{schema.NewHumanMessage("interface test")}

	// Test MessageGenerator interface
	_, err = model.GenerateMessages(ctx, messages)
	if err != nil {
		t.Errorf("MessageGenerator.GenerateMessages failed: %v", err)
	}

	// Test StreamMessageHandler interface
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	stream, err := model.StreamMessages(ctx, messages)
	if err != nil {
		t.Errorf("StreamMessageHandler.StreamMessages failed: %v", err)
	} else {
		// Consume stream
		for range stream {
		}
	}

	// Test ModelInfoProvider interface
	info := model.GetModelInfo()
	if info.Name == "" {
		t.Error("ModelInfoProvider.GetModelInfo returned empty name")
	}

	// Test HealthChecker interface
	health := model.CheckHealth()
	if health == nil {
		t.Error("HealthChecker.CheckHealth returned nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	// Test core.Runnable interface
	result, err := model.Invoke(ctx, messages)
	if err != nil {
		t.Errorf("Runnable.Invoke failed: %v", err)
	}
	if result == nil {
		t.Error("Runnable.Invoke returned nil result")
	}
}

// Benchmarks for performance testing
func BenchmarkGenerateMessages(b *testing.B) {
	model, err := NewMockChatModel("benchmark-model")
	if err != nil {
		b.Fatalf("Failed to create mock model: %v", err)
	}

	ctx := context.Background()
	messages := []schema.Message{schema.NewHumanMessage("benchmark test message")}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := model.GenerateMessages(ctx, messages)
		if err != nil {
			b.Errorf("GenerateMessages failed: %v", err)
		}
	}
}

func BenchmarkStreamMessages(b *testing.B) {
	model, err := NewMockChatModel("benchmark-stream-model")
	if err != nil {
		b.Fatalf("Failed to create mock model: %v", err)
	}

	ctx := context.Background()
	messages := []schema.Message{schema.NewHumanMessage("benchmark stream test")}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream, err := model.StreamMessages(ctx, messages)
		if err != nil {
			b.Errorf("StreamMessages failed: %v", err)
			continue
		}
		// Consume the stream
		for range stream {
		}
	}
}

func BenchmarkConcurrentRequests(b *testing.B) {
	model, err := NewMockChatModel("concurrent-benchmark-model")
	if err != nil {
		b.Fatalf("Failed to create mock model: %v", err)
	}

	ctx := context.Background()
	messages := []schema.Message{schema.NewHumanMessage("concurrent benchmark test")}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := model.GenerateMessages(ctx, messages)
			if err != nil {
				b.Errorf("Concurrent GenerateMessages failed: %v", err)
			}
		}
	})
}
