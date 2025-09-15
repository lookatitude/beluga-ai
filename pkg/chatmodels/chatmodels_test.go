package chatmodels

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func TestNewChatModel(t *testing.T) {
	tests := []struct {
		name        string
		model       string
		config      *Config
		opts        []iface.Option
		expectError bool
		errorCode   string
	}{
		{
			name:  "valid openai model",
			model: "gpt-4",
			config: &Config{
				DefaultProvider:         "openai",
				DefaultTemperature:      0.7,
				DefaultMaxTokens:        1000,
				DefaultTimeout:          time.Second,
				DefaultMaxRetries:       3,
				DefaultRetryDelay:       time.Second,
				MaxConcurrentRequests:   100,
				StreamBufferSize:        100,
				StreamTimeout:           time.Minute,
				Providers:               make(map[string]interface{}),
			},
			opts:        []iface.Option{},
			expectError: false,
		},
		{
			name:  "valid mock model",
			model: "mock-gpt-4",
			config: &Config{
				DefaultProvider:         "mock",
				DefaultTemperature:      0.7,
				DefaultMaxTokens:        1000,
				DefaultTimeout:          time.Second,
				DefaultMaxRetries:       3,
				DefaultRetryDelay:       time.Second,
				MaxConcurrentRequests:   100,
				StreamBufferSize:        100,
				StreamTimeout:           time.Minute,
				Providers:               make(map[string]interface{}),
			},
			opts:        []iface.Option{},
			expectError: false,
		},
		{
			name:  "unsupported provider",
			model: "gpt-4",
			config: &Config{
				DefaultProvider:         "unsupported",
				DefaultTemperature:      0.7,
				DefaultMaxTokens:        1000,
				DefaultTimeout:          time.Second,
				DefaultMaxRetries:       3,
				DefaultRetryDelay:       time.Second,
				MaxConcurrentRequests:   100,
				StreamBufferSize:        100,
				StreamTimeout:           time.Minute,
				Providers:               make(map[string]interface{}),
			},
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
			config: &Config{
				DefaultProvider:         "mock",
				DefaultTemperature:      0.5,
				DefaultMaxTokens:        1000,
				DefaultTimeout:          time.Second,
				DefaultMaxRetries:       3,
				DefaultRetryDelay:       time.Second,
				MaxConcurrentRequests:   100,
				StreamBufferSize:        100,
				StreamTimeout:           time.Minute,
				Providers:               make(map[string]interface{}),
			},
			opts:        []iface.Option{WithTemperature(0.8)},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := NewChatModel(tt.model, tt.config, tt.opts...)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorCode != "" {
					var chatErr *ChatModelError
					if errors.As(err, &chatErr) {
						if chatErr.Code != tt.errorCode {
							t.Errorf("expected error code %s, got %s", tt.errorCode, chatErr.Code)
						}
					} else {
						t.Errorf("expected ChatModelError, got %T", err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if model == nil {
					t.Errorf("expected model to be created, got nil")
				}
			}
		})
	}
}

func TestNewOpenAIChatModel(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		apiKey    string
		opts      []iface.Option
		wantError bool
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := NewOpenAIChatModel(tt.model, tt.apiKey, tt.opts...)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if model == nil {
					t.Errorf("expected model to be created, got nil")
				}
			}
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := NewMockChatModel(tt.model, tt.opts...)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if model == nil {
					t.Errorf("expected model to be created, got nil")
				}
			}
		})
	}
}

func TestChatModel_GenerateMessages(t *testing.T) {
	model, err := NewMockChatModel("test-model")
	if err != nil {
		t.Fatalf("Failed to create mock model: %v", err)
	}

	ctx := context.Background()
	messages := []schema.Message{
		schema.NewHumanMessage("Hello, how are you?"),
	}

	response, err := model.GenerateMessages(ctx, messages)
	if err != nil {
		t.Fatalf("GenerateMessages failed: %v", err)
	}

	if len(response) == 0 {
		t.Fatal("Expected at least one response message")
	}

	// Check that response contains expected content
	found := false
	for _, msg := range response {
		if strings.Contains(msg.GetContent(), "mock response") ||
		   strings.Contains(msg.GetContent(), "simulated") ||
		   strings.Contains(msg.GetContent(), "test response") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected mock response content not found")
	}
}

func TestChatModel_StreamMessages(t *testing.T) {
	model, err := NewMockChatModel("test-model")
	if err != nil {
		t.Fatalf("Failed to create mock model: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	messages := []schema.Message{
		schema.NewHumanMessage("Hello, streaming test"),
	}

	stream, err := model.StreamMessages(ctx, messages)
	if err != nil {
		t.Fatalf("StreamMessages failed: %v", err)
	}

	var receivedMessages []schema.Message
	for msg := range stream {
		receivedMessages = append(receivedMessages, msg)
	}

	if len(receivedMessages) == 0 {
		t.Fatal("Expected at least one streamed message")
	}

	// Combine all message content
	var fullContent strings.Builder
	for _, msg := range receivedMessages {
		fullContent.WriteString(msg.GetContent())
	}

	if fullContent.Len() == 0 {
		t.Error("Expected non-empty streamed content")
	}
}

func TestChatModel_GetModelInfo(t *testing.T) {
	model, err := NewMockChatModel("test-model")
	if err != nil {
		t.Fatalf("Failed to create mock model: %v", err)
	}

	info := model.GetModelInfo()

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
}

func TestChatModel_CheckHealth(t *testing.T) {
	model, err := NewMockChatModel("test-model")
	if err != nil {
		t.Fatalf("Failed to create mock model: %v", err)
	}

	health := model.CheckHealth()

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
		provider        string
		expectedModels  []string
	}{
		{
			provider: "openai",
			expectedModels: []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo", "gpt-4o", "gpt-4o-mini"},
		},
		{
			provider: "mock",
			expectedModels: []string{"mock-gpt-4", "mock-claude", "mock-general"},
		},
		{
			provider: "unknown",
			expectedModels: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			models := GetSupportedModels(tt.provider)

			if len(models) != len(tt.expectedModels) {
				t.Errorf("Expected %d models for provider %s, got %d",
					len(tt.expectedModels), tt.provider, len(models))
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
		DefaultProvider:         "unsupported",
		DefaultTemperature:      0.7,
		DefaultMaxTokens:        1000,
		DefaultTimeout:          time.Second,
		DefaultMaxRetries:       3,
		DefaultRetryDelay:       time.Second,
		MaxConcurrentRequests:   100,
		StreamBufferSize:        100,
		StreamTimeout:           time.Minute,
		Providers:               make(map[string]interface{}),
	})

	if err == nil {
		t.Fatal("Expected error for unsupported provider")
	}

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
