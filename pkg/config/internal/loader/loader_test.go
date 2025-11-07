package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/config/iface"
)

func TestNewLoader(t *testing.T) {
	tests := []struct {
		name    string
		options iface.LoaderOptions
	}{
		{
			name: "default options",
			options: iface.LoaderOptions{
				ConfigName:  "test",
				ConfigPaths: []string{"./testdata"},
				EnvPrefix:   "TEST",
				Validate:    true,
				SetDefaults: true,
			},
		},
		{
			name:    "empty options",
			options: iface.LoaderOptions{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader, err := NewLoader(tt.options)
			if err != nil {
				t.Fatalf("NewLoader() error = %v", err)
			}
			if loader == nil {
				t.Fatal("NewLoader() returned nil loader")
			}
			if loader.options.ConfigName != tt.options.ConfigName {
				t.Errorf("NewLoader() ConfigName = %v, want %v", loader.options.ConfigName, tt.options.ConfigName)
			}
		})
	}
}

func TestLoader_LoadConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")
	configContent := `
llm_providers:
  - name: "test-llm"
    provider: "openai"
    api_key: "test-key"
    model_name: "gpt-4"
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	tests := []struct {
		name        string
		options     iface.LoaderOptions
		expectError bool
	}{
		{
			name: "successful load with defaults and validation",
			options: iface.LoaderOptions{
				ConfigName:  "test",
				ConfigPaths: []string{tempDir},
				Validate:    true,
				SetDefaults: true,
			},
			expectError: false,
		},
		{
			name: "successful load without validation",
			options: iface.LoaderOptions{
				ConfigName:  "test",
				ConfigPaths: []string{tempDir},
				Validate:    false,
				SetDefaults: false,
			},
			expectError: false,
		},
		{
			name: "load with non-existent config file",
			options: iface.LoaderOptions{
				ConfigName:  "nonexistent",
				ConfigPaths: []string{tempDir},
				Validate:    false,
				SetDefaults: false,
			},
			expectError: false, // Viper handles missing files gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader, err := NewLoader(tt.options)
			if err != nil {
				t.Fatalf("failed to create loader: %v", err)
			}

			cfg, err := loader.LoadConfig()
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
			if !tt.expectError && cfg == nil {
				t.Error("expected config but got nil")
			}
		})
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Set up test environment variables
	testEnvVars := map[string]string{
		"TEST_LLM_PROVIDERS_0_NAME":       "env-llm",
		"TEST_LLM_PROVIDERS_0_PROVIDER":   "openai",
		"TEST_LLM_PROVIDERS_0_API_KEY":    "env-key",
		"TEST_LLM_PROVIDERS_0_MODEL_NAME": "gpt-4",
	}

	// Set environment variables
	for key, value := range testEnvVars {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	tests := []struct {
		name        string
		prefix      string
		expectError bool
	}{
		{
			name:        "successful load from env",
			prefix:      "TEST",
			expectError: false,
		},
		{
			name:        "load from env with empty prefix",
			prefix:      "",
			expectError: false,
		},
		{
			name:        "load from env with non-matching prefix",
			prefix:      "OTHER",
			expectError: false, // Should succeed but with empty config
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadFromEnv(tt.prefix)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
			if !tt.expectError && cfg == nil {
				t.Error("expected config but got nil")
			}

			// Check if we got the expected config for the TEST prefix
			if tt.prefix == "TEST" && !tt.expectError {
				if len(cfg.LLMProviders) != 1 {
					t.Errorf("expected 1 LLM provider from env vars, got %d", len(cfg.LLMProviders))
				}
				if len(cfg.LLMProviders) > 0 && cfg.LLMProviders[0].Name != "env-llm" {
					t.Errorf("expected LLM provider name 'env-llm', got %s", cfg.LLMProviders[0].Name)
				}
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		filePath    string
		createFile  bool
		content     string
		expectError bool
	}{
		{
			name:       "successful load from yaml file",
			filePath:   filepath.Join(tempDir, "test.yaml"),
			createFile: true,
			content: `
llm_providers:
  - name: "file-llm"
    provider: "openai"
    api_key: "file-key"
    model_name: "gpt-4"
`,
			expectError: false,
		},
		{
			name:        "load from non-existent file",
			filePath:    filepath.Join(tempDir, "nonexistent.yaml"),
			createFile:  false,
			expectError: true,
		},
		{
			name:       "load from json file",
			filePath:   filepath.Join(tempDir, "test.json"),
			createFile: true,
			content: `{
  "llm_providers": [
    {
      "name": "json-llm",
      "provider": "openai",
      "api_key": "json-key",
      "model_name": "gpt-4"
    }
  ]
}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.createFile {
				err := os.WriteFile(tt.filePath, []byte(tt.content), 0644)
				if err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
			}

			cfg, err := LoadFromFile(tt.filePath)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
			if !tt.expectError && cfg == nil {
				t.Error("expected config but got nil")
			}

			// Verify config content for successful cases
			if !tt.expectError && cfg != nil {
				if len(cfg.LLMProviders) != 1 {
					t.Errorf("expected 1 LLM provider, got %d", len(cfg.LLMProviders))
				}
			}
		})
	}
}

func TestLoader_MustLoadConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")
	configContent := `
llm_providers:
  - name: "test-llm"
    provider: "openai"
    api_key: "test-key"
    model_name: "gpt-4"
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	loader, err := NewLoader(iface.LoaderOptions{
		ConfigName:  "test",
		ConfigPaths: []string{tempDir},
		Validate:    false,
		SetDefaults: false,
	})
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}

	// Test successful case
	cfg := loader.MustLoadConfig()
	if cfg == nil {
		t.Error("MustLoadConfig() returned nil config")
	}
	if len(cfg.LLMProviders) != 1 {
		t.Errorf("expected 1 LLM provider, got %d", len(cfg.LLMProviders))
	}
}

func TestLoader_MustLoadConfig_Panic(t *testing.T) {
	// Test panic case - create a loader with invalid YAML to force a load error
	tempDir := t.TempDir()
	invalidFile := filepath.Join(tempDir, "test.yaml")
	os.WriteFile(invalidFile, []byte("invalid: yaml: [unclosed"), 0644)

	loader, err := NewLoader(iface.LoaderOptions{
		ConfigName:  "test",
		ConfigPaths: []string{tempDir},
		Validate:    false,
		SetDefaults: false,
	})
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustLoadConfig() should have panicked")
		}
	}()

	loader.MustLoadConfig()
}

func TestLoader_FluentAPI(t *testing.T) {
	loader, err := NewLoader(iface.LoaderOptions{})
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}

	// Test fluent API methods
	result := loader.
		WithConfigName("test-config").
		WithConfigPaths("./config", "/etc/app").
		WithEnvPrefix("MYAPP").
		WithValidation(true).
		WithDefaults(true)

	if result != loader {
		t.Error("fluent API methods should return the same loader instance")
	}

	if loader.options.ConfigName != "test-config" {
		t.Errorf("ConfigName = %s, want test-config", loader.options.ConfigName)
	}

	if len(loader.options.ConfigPaths) != 2 {
		t.Errorf("expected 2 config paths, got %d", len(loader.options.ConfigPaths))
	}

	if loader.options.EnvPrefix != "MYAPP" {
		t.Errorf("EnvPrefix = %s, want MYAPP", loader.options.EnvPrefix)
	}

	if !loader.options.Validate {
		t.Error("Validate should be true")
	}

	if !loader.options.SetDefaults {
		t.Error("SetDefaults should be true")
	}
}

func TestGetEnvConfigMap(t *testing.T) {
	// Set up test environment variables
	testEnvVars := map[string]string{
		"TEST_VAR1": "value1",
		"TEST_VAR2": "value2",
		"OTHER_VAR": "other_value",
	}

	// Set environment variables
	for key, value := range testEnvVars {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	tests := []struct {
		name     string
		prefix   string
		expected map[string]string
	}{
		{
			name:   "get TEST prefixed vars",
			prefix: "TEST",
			expected: map[string]string{
				"var1": "value1",
				"var2": "value2",
			},
		},
		{
			name:   "get OTHER prefixed vars",
			prefix: "OTHER",
			expected: map[string]string{
				"var": "other_value",
			},
		},
		{
			name:     "get empty prefix vars",
			prefix:   "",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetEnvConfigMap(tt.prefix)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d env vars, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists || actualValue != expectedValue {
					t.Errorf("expected %s=%s, got %s=%s", key, expectedValue, key, actualValue)
				}
			}
		})
	}
}

func TestEnvVarName(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		key      string
		expected string
	}{
		{
			name:     "simple key",
			prefix:   "APP",
			key:      "database.host",
			expected: "APP_DATABASE_HOST",
		},
		{
			name:     "key with multiple dots",
			prefix:   "MYAPP",
			key:      "llm.providers.openai.api_key",
			expected: "MYAPP_LLM_PROVIDERS_OPENAI_API_KEY",
		},
		{
			name:     "empty prefix",
			prefix:   "",
			key:      "simple.key",
			expected: "_SIMPLE_KEY",
		},
		{
			name:     "empty key",
			prefix:   "APP",
			key:      "",
			expected: "APP_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EnvVarName(tt.prefix, tt.key)
			if result != tt.expected {
				t.Errorf("EnvVarName(%s, %s) = %s, want %s", tt.prefix, tt.key, result, tt.expected)
			}
		})
	}
}

func TestConfigKey(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		envVar   string
		expected string
	}{
		{
			name:     "simple env var",
			prefix:   "APP",
			envVar:   "APP_DATABASE_HOST",
			expected: "database.host",
		},
		{
			name:     "complex env var",
			prefix:   "MYAPP",
			envVar:   "MYAPP_LLM_PROVIDERS_OPENAI_API_KEY",
			expected: "llm.providers.openai.api.key",
		},
		{
			name:     "empty prefix",
			prefix:   "",
			envVar:   "_SIMPLE_KEY",
			expected: "simple.key",
		},
		{
			name:     "no prefix match",
			prefix:   "APP",
			envVar:   "OTHER_VAR",
			expected: "other.var",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConfigKey(tt.prefix, tt.envVar)
			if result != tt.expected {
				t.Errorf("ConfigKey(%s, %s) = %s, want %s", tt.prefix, tt.envVar, result, tt.expected)
			}
		})
	}
}

func TestLoader_LoadConfig_WithValidationError(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "invalid.yaml")
	// Create invalid config (missing required fields)
	configContent := `
llm_providers:
  - name: ""
    provider: "openai"
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	loader, err := NewLoader(iface.LoaderOptions{
		ConfigName:  "invalid",
		ConfigPaths: []string{tempDir},
		Validate:    true,
		SetDefaults: false,
	})
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}

	_, err = loader.LoadConfig()
	if err == nil {
		t.Error("expected validation error but got none")
	}
}

func TestLoader_LoadConfig_WithDefaults(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")
	configContent := `
llm_providers:
  - name: "test-llm"
    provider: "openai"
    api_key: "test-key"
    model_name: "gpt-4"
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	loader, err := NewLoader(iface.LoaderOptions{
		ConfigName:  "test",
		ConfigPaths: []string{tempDir},
		Validate:    false,
		SetDefaults: true,
	})
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}

	cfg, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected config but got nil")
	}

	// Check that defaults were set
	if len(cfg.LLMProviders) > 0 {
		provider := cfg.LLMProviders[0]
		if provider.DefaultCallOptions == nil {
			t.Error("expected DefaultCallOptions to be initialized")
		}
	}
}
