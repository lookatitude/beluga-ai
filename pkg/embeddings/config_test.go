package embeddings

import (
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
