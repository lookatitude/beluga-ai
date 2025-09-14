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
