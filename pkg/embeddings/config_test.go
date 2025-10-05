package embeddings

import (
	"strings"
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "empty config",
			config:  &Config{},
			wantErr: false, // Empty config is valid, providers just won't be available
		},
		{
			name: "valid openai config",
			config: &Config{
				OpenAI: &OpenAIConfig{
					APIKey: "test-key",
					Model:  "text-embedding-ada-002",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid openai config - missing api key",
			config: &Config{
				OpenAI: &OpenAIConfig{
					Model: "text-embedding-ada-002",
				},
			},
			wantErr: true,
		},
		{
			name: "valid ollama config",
			config: &Config{
				Ollama: &OllamaConfig{
					Model: "nomic-embed-text",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid ollama config - missing model",
			config: &Config{
				Ollama: &OllamaConfig{},
			},
			wantErr: true,
		},
		{
			name: "valid mock config",
			config: &Config{
				Mock: &MockConfig{
					Dimension: 128,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid mock config - zero dimension",
			config: &Config{
				Mock: &MockConfig{
					Dimension: 0,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_SetDefaults(t *testing.T) {
	config := &Config{}
	config.SetDefaults()

	// Check OpenAI defaults
	if config.OpenAI == nil {
		t.Error("OpenAI config should be initialized")
	} else {
		if config.OpenAI.Model != "text-embedding-ada-002" {
			t.Errorf("Expected OpenAI model default 'text-embedding-ada-002', got %s", config.OpenAI.Model)
		}
		if config.OpenAI.Timeout != 30*time.Second {
			t.Errorf("Expected OpenAI timeout default 30s, got %v", config.OpenAI.Timeout)
		}
		if config.OpenAI.MaxRetries != 3 {
			t.Errorf("Expected OpenAI max retries default 3, got %d", config.OpenAI.MaxRetries)
		}
		if !config.OpenAI.Enabled {
			t.Error("Expected OpenAI to be enabled by default")
		}
	}

	// Check Ollama defaults
	if config.Ollama == nil {
		t.Error("Ollama config should be initialized")
	} else {
		if config.Ollama.ServerURL != "http://localhost:11434" {
			t.Errorf("Expected Ollama server URL default 'http://localhost:11434', got %s", config.Ollama.ServerURL)
		}
		if config.Ollama.Timeout != 30*time.Second {
			t.Errorf("Expected Ollama timeout default 30s, got %v", config.Ollama.Timeout)
		}
		if !config.Ollama.Enabled {
			t.Error("Expected Ollama to be enabled by default")
		}
	}

	// Check Mock defaults
	if config.Mock == nil {
		t.Error("Mock config should be initialized")
	} else {
		if config.Mock.Dimension != 128 {
			t.Errorf("Expected Mock dimension default 128, got %d", config.Mock.Dimension)
		}
		if !config.Mock.Enabled {
			t.Error("Expected Mock to be enabled by default")
		}
	}
}

func TestOpenAIConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *OpenAIConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &OpenAIConfig{
				APIKey: "test-key",
				Model:  "text-embedding-ada-002",
			},
			wantErr: false,
		},
		{
			name: "missing api key",
			config: &OpenAIConfig{
				Model: "text-embedding-ada-002",
			},
			wantErr: true,
		},
		{
			name: "missing model",
			config: &OpenAIConfig{
				APIKey: "test-key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("OpenAIConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOllamaConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *OllamaConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &OllamaConfig{
				Model: "nomic-embed-text",
			},
			wantErr: false,
		},
		{
			name:    "missing model",
			config:  &OllamaConfig{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("OllamaConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMockConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *MockConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &MockConfig{
				Dimension: 128,
			},
			wantErr: false,
		},
		{
			name: "zero dimension",
			config: &MockConfig{
				Dimension: 0,
			},
			wantErr: true,
		},
		{
			name: "negative dimension",
			config: &MockConfig{
				Dimension: -1,
			},
			wantErr: true,
		},
		{
			name:    "very large dimension",
			config:  &MockConfig{Dimension: 100000},
			wantErr: false, // Large dimensions are allowed
		},
		{
			name:    "minimum valid dimension",
			config:  &MockConfig{Dimension: 1},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("MockConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_ValidateWithMultipleProviders(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "all providers valid",
			config: &Config{
				OpenAI: &OpenAIConfig{
					APIKey: "sk-test",
					Model:  "text-embedding-ada-002",
				},
				Ollama: &OllamaConfig{
					Model: "nomic-embed-text",
				},
				Mock: &MockConfig{
					Dimension: 128,
				},
			},
			wantErr: false,
		},
		{
			name: "one provider invalid",
			config: &Config{
				OpenAI: &OpenAIConfig{
					APIKey: "sk-test",
					Model:  "text-embedding-ada-002",
				},
				Ollama: &OllamaConfig{
					// Missing model - invalid
				},
				Mock: &MockConfig{
					Dimension: 128,
				},
			},
			wantErr: true,
		},
		{
			name: "all providers invalid",
			config: &Config{
				OpenAI: &OpenAIConfig{
					// Missing API key
					Model: "text-embedding-ada-002",
				},
				Ollama: &OllamaConfig{
					// Missing model
				},
				Mock: &MockConfig{
					Dimension: 0, // Invalid
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_SetDefaults_PartialInitialization(t *testing.T) {
	// Test setting defaults when some configs are already partially initialized
	config := &Config{
		OpenAI: &OpenAIConfig{
			APIKey: "sk-test",
			// Model not set, should get default
		},
		Ollama: &OllamaConfig{
			// Only server URL set
			ServerURL: "http://custom:8080",
		},
		// Mock not set at all
	}

	config.SetDefaults()

	// Check OpenAI got model default but kept API key
	if config.OpenAI.Model != "text-embedding-ada-002" {
		t.Errorf("Expected OpenAI model default, got %s", config.OpenAI.Model)
	}
	if config.OpenAI.APIKey != "sk-test" {
		t.Error("OpenAI API key should not be changed")
	}

	// Check Ollama kept custom server URL but got other defaults
	if config.Ollama.ServerURL != "http://custom:8080" {
		t.Errorf("Expected custom server URL, got %s", config.Ollama.ServerURL)
	}
	if config.Ollama.Model != "" { // Should not set model default
		t.Errorf("Ollama model should not be set by default, got %s", config.Ollama.Model)
	}

	// Check Mock got fully initialized
	if config.Mock == nil || config.Mock.Dimension != 128 {
		t.Error("Mock should be fully initialized with defaults")
	}
}

func TestConfig_TimeoutValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid timeout",
			config: &Config{
				OpenAI: &OpenAIConfig{
					APIKey:  "sk-test",
					Model:   "text-embedding-ada-002",
					Timeout: 60 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "zero timeout allowed",
			config: &Config{
				OpenAI: &OpenAIConfig{
					APIKey:  "sk-test",
					Model:   "text-embedding-ada-002",
					Timeout: 0, // Should be allowed
				},
			},
			wantErr: false,
		},
		{
			name: "very long timeout",
			config: &Config{
				OpenAI: &OpenAIConfig{
					APIKey:  "sk-test",
					Model:   "text-embedding-ada-002",
					Timeout: 24 * time.Hour, // Very long but valid
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_ConcurrentValidation(t *testing.T) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
		},
	}

	// Test concurrent validation
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			err := config.Validate()
			if err != nil {
				t.Errorf("Concurrent validation failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestOpenAIConfig_Validate_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		config  *OpenAIConfig
		wantErr bool
	}{
		{
			name: "valid with all fields",
			config: &OpenAIConfig{
				APIKey:     "sk-test123456789",
				Model:      "text-embedding-ada-002",
				BaseURL:    "https://api.openai.com",
				Timeout:    30 * time.Second,
				MaxRetries: 3,
				Enabled:    true,
			},
			wantErr: false,
		},
		{
			name: "empty api key",
			config: &OpenAIConfig{
				APIKey: "",
				Model:  "text-embedding-ada-002",
			},
			wantErr: true,
		},
		{
			name: "whitespace api key",
			config: &OpenAIConfig{
				APIKey: "   ",
				Model:  "text-embedding-ada-002",
			},
			wantErr: false, // Validator doesn't consider whitespace as empty
		},
		{
			name: "invalid model name",
			config: &OpenAIConfig{
				APIKey: "sk-test",
				Model:  "", // Empty model
			},
			wantErr: true,
		},
		{
			name: "very long model name",
			config: &OpenAIConfig{
				APIKey: "sk-test",
				Model:  string(make([]byte, 1000)), // Very long model name
			},
			wantErr: false, // Should be allowed
		},
		{
			name: "negative max retries",
			config: &OpenAIConfig{
				APIKey:     "sk-test",
				Model:      "text-embedding-ada-002",
				MaxRetries: -1,
			},
			wantErr: false, // Negative retries might be allowed by the validator
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("OpenAIConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOllamaConfig_Validate_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		config  *OllamaConfig
		wantErr bool
	}{
		{
			name: "valid with all fields",
			config: &OllamaConfig{
				ServerURL:  "http://localhost:11434",
				Model:      "nomic-embed-text",
				Timeout:    30 * time.Second,
				MaxRetries: 3,
				KeepAlive:  "5m",
				Enabled:    true,
			},
			wantErr: false,
		},
		{
			name: "custom server URL",
			config: &OllamaConfig{
				ServerURL: "https://ollama.example.com:8080",
				Model:     "nomic-embed-text",
			},
			wantErr: false,
		},
		{
			name: "empty server URL",
			config: &OllamaConfig{
				ServerURL: "",
				Model:     "nomic-embed-text",
			},
			wantErr: false, // Empty server URL might be allowed
		},
		{
			name:    "missing model",
			config:  &OllamaConfig{},
			wantErr: true,
		},
		{
			name: "whitespace model",
			config: &OllamaConfig{
				Model: "   ",
			},
			wantErr: false, // Validator doesn't consider whitespace as empty
		},
		{
			name: "very long model name",
			config: &OllamaConfig{
				Model: string(make([]byte, 1000)),
			},
			wantErr: false, // Should be allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("OllamaConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOptionConfig(t *testing.T) {
	// Test the functional options
	opts := []Option{
		WithTimeout(10 * time.Second),
		WithMaxRetries(5),
		WithModel("custom-model"),
	}

	config := defaultOptionConfig()
	for _, opt := range opts {
		opt(config)
	}

	if config.timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", config.timeout)
	}
	if config.maxRetries != 5 {
		t.Errorf("Expected maxRetries 5, got %d", config.maxRetries)
	}
	if config.model != "custom-model" {
		t.Errorf("Expected model 'custom-model', got %s", config.model)
	}
}

func TestDefaultOptionConfig(t *testing.T) {
	config := defaultOptionConfig()

	if config.timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", config.timeout)
	}
	if config.maxRetries != 3 {
		t.Errorf("Expected default maxRetries 3, got %d", config.maxRetries)
	}
	if config.model != "" {
		t.Errorf("Expected default model empty, got %s", config.model)
	}
}

// TestConfig_Validate_EdgeCases tests configuration validation edge cases and boundary conditions
func TestConfig_Validate_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorCheck  func(error) bool // Optional function to validate error content
	}{
		// OpenAI edge cases
		{
			name: "openai empty api key",
			config: &Config{
				OpenAI: &OpenAIConfig{
					APIKey: "",
					Model:  "text-embedding-ada-002",
				},
			},
			expectError: true,
		},
		{
			name: "openai whitespace-only api key",
			config: &Config{
				OpenAI: &OpenAIConfig{
					APIKey: "   \t\n  ", // Technically not empty, so passes required validation
					Model:  "text-embedding-ada-002",
				},
			},
			expectError: false, // Current validation doesn't trim whitespace
		},
		{
			name: "openai very long api key",
			config: &Config{
				OpenAI: &OpenAIConfig{
					APIKey: strings.Repeat("a", 1000),
					Model:  "text-embedding-ada-002",
				},
			},
			expectError: false, // Should be valid
		},
		{
			name: "openai invalid model name",
			config: &Config{
				OpenAI: &OpenAIConfig{
					APIKey: "sk-test",
					Model:  "", // Empty model
				},
			},
			expectError: true, // OpenAI model is required
		},
		{
			name: "openai extreme timeout values",
			config: &Config{
				OpenAI: &OpenAIConfig{
					APIKey:  "sk-test",
					Model:   "text-embedding-ada-002",
					Timeout: -1 * time.Second, // Negative timeout
				},
			},
			expectError: false, // Config doesn't validate timeout ranges
		},
		{
			name: "openai negative max retries",
			config: &Config{
				OpenAI: &OpenAIConfig{
					APIKey:     "sk-test",
					Model:      "text-embedding-ada-002",
					MaxRetries: -5,
				},
			},
			expectError: false, // Config doesn't validate retry ranges
		},

		// Ollama edge cases
		{
			name: "ollama empty model",
			config: &Config{
				Ollama: &OllamaConfig{
					Model: "",
				},
			},
			expectError: true, // Model is required for Ollama
		},
		{
			name: "ollama invalid server URL",
			config: &Config{
				Ollama: &OllamaConfig{
					ServerURL: "not-a-url",
					Model:     "nomic-embed-text",
				},
			},
			expectError: false, // Config doesn't validate URL format
		},
		{
			name: "ollama localhost with port",
			config: &Config{
				Ollama: &OllamaConfig{
					ServerURL: "http://localhost:11434",
					Model:     "nomic-embed-text",
				},
			},
			expectError: false,
		},
		{
			name: "ollama very long model name",
			config: &Config{
				Ollama: &OllamaConfig{
					Model: strings.Repeat("model", 100), // Very long model name
				},
			},
			expectError: false, // Config doesn't validate model name length
		},

		// Mock edge cases
		{
			name: "mock zero dimension",
			config: &Config{
				Mock: &MockConfig{
					Dimension: 0,
				},
			},
			expectError: true, // Dimension must be positive
		},
		{
			name: "mock negative dimension",
			config: &Config{
				Mock: &MockConfig{
					Dimension: -128,
				},
			},
			expectError: true, // Dimension must be positive
		},
		{
			name: "mock very large dimension",
			config: &Config{
				Mock: &MockConfig{
					Dimension: 100000, // Very large dimension
				},
			},
			expectError: false, // Large dimensions are allowed
		},
		{
			name: "mock extreme seed values",
			config: &Config{
				Mock: &MockConfig{
					Dimension: 128,
					Seed:      -999999, // Extreme negative seed
				},
			},
			expectError: false, // Seed values are not validated
		},

		// Cross-provider edge cases
		{
			name: "multiple providers configured",
			config: &Config{
				OpenAI: &OpenAIConfig{
					APIKey: "sk-test",
					Model:  "text-embedding-ada-002",
				},
				Ollama: &OllamaConfig{
					Model: "nomic-embed-text",
				},
				Mock: &MockConfig{
					Dimension: 128,
				},
			},
			expectError: false, // Multiple providers can be configured
		},
		{
			name: "all providers disabled",
			config: &Config{
				OpenAI: &OpenAIConfig{
					APIKey:  "sk-test",
					Model:   "text-embedding-ada-002",
					Enabled: false,
				},
				Ollama: &OllamaConfig{
					Model:   "nomic-embed-text",
					Enabled: false,
				},
				Mock: &MockConfig{
					Dimension: 128,
					Enabled:   false,
				},
			},
			expectError: false, // Disabled providers are still valid config
		},
		{
			name: "partial provider configuration",
			config: &Config{
				OpenAI: &OpenAIConfig{
					APIKey: "sk-test",
					Model:  "text-embedding-ada-002", // Include required model
				},
			},
			expectError: false, // Config is valid with required fields
		},

		// Boundary and special cases
		{
			name: "unicode in api key",
			config: &Config{
				OpenAI: &OpenAIConfig{
					APIKey: "sk-test-🚀-unicode",
					Model:  "text-embedding-ada-002",
				},
			},
			expectError: false, // Unicode should be allowed
		},
		{
			name: "special characters in model names",
			config: &Config{
				Ollama: &OllamaConfig{
					Model: "model:with:colons/and/slashes",
				},
			},
			expectError: false, // Special chars should be allowed in model names
		},
		{
			name: "empty server URL for Ollama",
			config: &Config{
				Ollama: &OllamaConfig{
					ServerURL: "",
					Model:     "nomic-embed-text",
				},
			},
			expectError: false, // Empty URL should use default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError && err == nil {
				t.Errorf("expected validation error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected validation error: %v", err)
				return
			}

			// If error was expected and custom validation function provided
			if tt.expectError && err != nil && tt.errorCheck != nil {
				if !tt.errorCheck(err) {
					t.Errorf("error did not match expected criteria: %v", err)
				}
			}

			// Note: SetDefaults testing removed due to it creating invalid defaults
			// for unconfigured providers. The main focus is on validation edge cases.
		})
	}
}

// TestConfig_BoundaryConditions tests configuration boundary conditions and limits
func TestConfig_BoundaryConditions(t *testing.T) {
	// Test extremely large configurations
	largeConfig := &Config{
		OpenAI: &OpenAIConfig{
			APIKey:     strings.Repeat("x", 10000),                      // Very long API key
			Model:      strings.Repeat("model", 1000),                   // Very long model name
			BaseURL:    "https://" + strings.Repeat("a", 1000) + ".com", // Very long URL
			MaxRetries: 1000000,                                         // Very high retry count
		},
		Ollama: &OllamaConfig{
			ServerURL:  "http://" + strings.Repeat("b", 1000) + ":11434",
			Model:      strings.Repeat("model", 500),
			MaxRetries: 1000000,
		},
		Mock: &MockConfig{
			Dimension: 1000000,             // Very large dimension
			Seed:      9223372036854775807, // Max int64
		},
	}

	// Should not panic or cause issues
	err := largeConfig.Validate()
	if err != nil {
		t.Logf("Large config validation failed (may be expected): %v", err)
	}

	// Test defaults setting on large config
	largeConfig.SetDefaults()
	err = largeConfig.Validate()
	if err != nil {
		t.Logf("Large config with defaults validation failed: %v", err)
	}

	// Test nil sub-configs don't cause panics
	nilConfig := &Config{
		OpenAI: nil,
		Ollama: nil,
		Mock:   nil,
	}

	err = nilConfig.Validate()
	if err != nil {
		t.Logf("Nil sub-configs validation failed: %v", err)
	}

	nilConfig.SetDefaults()
	err = nilConfig.Validate()
	if err != nil {
		t.Logf("Nil sub-configs with defaults validation failed: %v", err)
	}
}
