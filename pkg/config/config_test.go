package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/config/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/require"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		config  *iface.Config
		name    string
		wantErr bool
	}{
		{
			name: "valid config",
			config: &iface.Config{
				LLMProviders: []schema.LLMProviderConfig{
					{
						Name:      "test-openai",
						Provider:  "openai",
						ModelName: "gpt-4",
						APIKey:    "sk-test",
					},
				},
				EmbeddingProviders: []schema.EmbeddingProviderConfig{
					{
						Name:      "test-embeddings",
						Provider:  "openai",
						ModelName: "text-embedding-ada-002",
						APIKey:    "sk-test",
					},
				},
				VectorStores: []schema.VectorStoreConfig{
					{
						Name:     "test-vectorstore",
						Provider: "inmemory",
					},
				},
				Tools: []iface.ToolConfig{
					{
						Name:     "test-tool",
						Provider: "echo",
					},
				},
				Agents: []schema.AgentConfig{
					{
						Name:            "test-agent",
						LLMProviderName: "test-openai",
						MaxIterations:   10,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid LLM provider - missing name",
			config: &iface.Config{
				LLMProviders: []schema.LLMProviderConfig{
					{
						Provider:  "openai",
						ModelName: "gpt-4",
						APIKey:    "sk-test",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid LLM provider - missing model name",
			config: &iface.Config{
				LLMProviders: []schema.LLMProviderConfig{
					{
						Name:     "test-openai",
						Provider: "openai",
						APIKey:   "sk-test",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid embedding provider - missing API key",
			config: &iface.Config{
				EmbeddingProviders: []schema.EmbeddingProviderConfig{
					{
						Name:      "test-embeddings",
						Provider:  "openai",
						ModelName: "text-embedding-ada-002",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid vector store - missing provider",
			config: &iface.Config{
				VectorStores: []schema.VectorStoreConfig{
					{
						Name: "test-vectorstore",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid tool - missing provider",
			config: &iface.Config{
				Tools: []iface.ToolConfig{
					{
						Name: "test-tool",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid agent - missing LLM provider name",
			config: &iface.Config{
				Agents: []schema.AgentConfig{
					{
						Name:          "test-agent",
						MaxIterations: 10,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid agent - invalid max iterations",
			config: &iface.Config{
				Agents: []schema.AgentConfig{
					{
						Name:            "test-agent",
						LLMProviderName: "test-openai",
						MaxIterations:   0,
					},
				},
			},
			wantErr: true,
		},
		{
			name:    "empty config",
			config:  &iface.Config{},
			wantErr: false, // Empty config is valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := iface.ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSetDefaults(t *testing.T) {
	config := &iface.Config{
		LLMProviders: []schema.LLMProviderConfig{
			{
				Name:      "test-openai",
				Provider:  "openai",
				ModelName: "gpt-4",
				APIKey:    "sk-test",
			},
		},
		EmbeddingProviders: []schema.EmbeddingProviderConfig{
			{
				Name:      "test-embeddings",
				Provider:  "openai",
				ModelName: "text-embedding-ada-002",
				APIKey:    "sk-test",
			},
		},
	}

	iface.SetDefaults(config)

	// Check that defaults were set for LLM provider
	if config.LLMProviders[0].DefaultCallOptions == nil {
		t.Error("Expected DefaultCallOptions to be initialized")
	}

	// Check temperature default
	if temp, ok := config.LLMProviders[0].DefaultCallOptions["temperature"]; !ok || temp != 0.7 {
		t.Errorf("Expected temperature to be 0.7, got %v", temp)
	}

	// Check max_tokens default
	if maxTokens, ok := config.LLMProviders[0].DefaultCallOptions["max_tokens"]; !ok || maxTokens != 1000 {
		t.Errorf("Expected max_tokens to be 1000, got %v", maxTokens)
	}
}

func TestValidationError_Error(t *testing.T) {
	err := iface.ValidationError{
		Field:   "test_field",
		Message: "test message",
	}

	expected := "validation error for field 'test_field': test message"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %q, want %q", err.Error(), expected)
	}
}

func TestValidationErrors_Error(t *testing.T) {
	errs := iface.ValidationErrors{
		iface.ValidationError{Field: "field1", Message: "error1"},
		iface.ValidationError{Field: "field2", Message: "error2"},
	}

	result := errs.Error()
	expected := "configuration validation failed: validation error for field 'field1': error1; validation error for field 'field2': error2"

	if result != expected {
		t.Errorf("ValidationErrors.Error() = %q, want %q", result, expected)
	}
}

func TestIsRequiredField(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected bool
	}{
		{
			name:     "required field",
			tag:      `validate:"required"`,
			expected: true,
		},
		{
			name:     "required with other validations",
			tag:      `validate:"required,min=1"`,
			expected: true,
		},
		{
			name:     "not required",
			tag:      `validate:"min=1"`,
			expected: false,
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.StructField{
				Tag: reflect.StructTag(tt.tag),
			}
			result := iface.IsRequiredField(field)
			if result != tt.expected {
				t.Errorf("IsRequiredField() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetFieldName(t *testing.T) {
	tests := []struct {
		name     string
		jsonTag  string
		yamlTag  string
		expected string
	}{
		{
			name:     "json tag",
			jsonTag:  `json:"field_name"`,
			expected: "field_name",
		},
		{
			name:     "json tag with options",
			jsonTag:  `json:"field_name,omitempty"`,
			expected: "field_name",
		},
		{
			name:     "yaml tag",
			yamlTag:  `yaml:"field_name"`,
			expected: "field_name",
		},
		{
			name:     "yaml tag with options",
			yamlTag:  `yaml:"field_name,omitempty"`,
			expected: "field_name",
		},
		{
			name:     "no tags",
			expected: "TestField",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.StructField{
				Name: "TestField",
			}
			if tt.jsonTag != "" {
				field.Tag = reflect.StructTag(tt.jsonTag)
			}
			if tt.yamlTag != "" {
				field.Tag = reflect.StructTag(tt.yamlTag)
			}

			result := iface.GetFieldName(field)
			if result != tt.expected {
				t.Errorf("GetFieldName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	configContent := `
llm_providers:
  - name: "test-openai"
    provider: "openai"
    api_key: "sk-test"
    model_name: "gpt-4"
`
	err := os.WriteFile(configFile, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	// Change to temp directory to test default config loading
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	_ = os.Chdir(tempDir)
	defer func() { _ = os.Chdir(oldWd) }()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig() returned nil config")
	}

	if len(cfg.LLMProviders) != 1 {
		t.Errorf("expected 1 LLM provider, got %d", len(cfg.LLMProviders))
	}

	if cfg.LLMProviders[0].Name != "test-openai" {
		t.Errorf("expected LLM provider name 'test-openai', got %s", cfg.LLMProviders[0].Name)
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Set up test environment variables
	testEnvVars := map[string]string{
		"TEST_LLM_PROVIDERS_0_NAME":       "env-openai",
		"TEST_LLM_PROVIDERS_0_PROVIDER":   "openai",
		"TEST_LLM_PROVIDERS_0_API_KEY":    "sk-env-test",
		"TEST_LLM_PROVIDERS_0_MODEL_NAME": "gpt-4",
	}

	// Set environment variables
	for key, value := range testEnvVars {
		_ = os.Setenv(key, value)
		defer func(k string) { _ = os.Unsetenv(k) }(key)
	}

	cfg, err := LoadFromEnv("TEST")
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("LoadFromEnv() returned nil config")
	}

	if len(cfg.LLMProviders) != 1 {
		t.Fatalf("expected 1 LLM provider from env vars, got %d", len(cfg.LLMProviders))
	}

	if cfg.LLMProviders[0].Name != "env-openai" {
		t.Errorf("expected LLM provider name 'env-openai', got %s", cfg.LLMProviders[0].Name)
	}

	if cfg.LLMProviders[0].APIKey != "sk-env-test" {
		t.Errorf("expected API key 'sk-env-test', got %s", cfg.LLMProviders[0].APIKey)
	}
}

func TestLoadFromFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		fileName    string
		content     string
		expectError bool
	}{
		{
			name:     "load from YAML file",
			fileName: "test.yaml",
			content: `
llm_providers:
  - name: "yaml-openai"
    provider: "openai"
    api_key: "sk-yaml-test"
    model_name: "gpt-4"
`,
			expectError: false,
		},
		{
			name:     "load from JSON file",
			fileName: "test.json",
			content: `{
  "llm_providers": [
    {
      "name": "json-openai",
      "provider": "openai",
      "api_key": "sk-json-test",
      "model_name": "gpt-4"
    }
  ]
}`,
			expectError: false,
		},
		{
			name:        "load from non-existent file",
			fileName:    "nonexistent.yaml",
			content:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.fileName)

			if tt.content != "" {
				err := os.WriteFile(filePath, []byte(tt.content), 0o600)
				if err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
			}

			cfg, err := LoadFromFile(filePath)
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
			if !tt.expectError && cfg != nil && len(cfg.LLMProviders) > 0 {
				expectedName := "yaml-openai"
				if tt.fileName == "test.json" {
					expectedName = "json-openai"
				}
				if cfg.LLMProviders[0].Name != expectedName {
					t.Errorf("expected LLM provider name %s, got %s", expectedName, cfg.LLMProviders[0].Name)
				}
			}
		})
	}
}

func TestMustLoadConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	configContent := `
llm_providers:
  - name: "must-load-openai"
    provider: "openai"
    api_key: "sk-must-test"
    model_name: "gpt-4"
`
	err := os.WriteFile(configFile, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	_ = os.Chdir(tempDir)
	defer func() { _ = os.Chdir(oldWd) }()

	cfg := MustLoadConfig()
	if cfg == nil {
		t.Fatal("MustLoadConfig() returned nil config")
	}

	if len(cfg.LLMProviders) != 1 {
		t.Errorf("expected 1 LLM provider, got %d", len(cfg.LLMProviders))
	}
}

func TestDefaultLoaderOptions(t *testing.T) {
	options := DefaultLoaderOptions()

	if options.ConfigName != "config" {
		t.Errorf("expected ConfigName 'config', got %s", options.ConfigName)
	}

	if len(options.ConfigPaths) != 2 {
		t.Errorf("expected 2 config paths, got %d", len(options.ConfigPaths))
	}

	if options.ConfigPaths[0] != "./config" {
		t.Errorf("expected first config path './config', got %s", options.ConfigPaths[0])
	}

	if options.ConfigPaths[1] != "." {
		t.Errorf("expected second config path '.', got %s", options.ConfigPaths[1])
	}

	if options.EnvPrefix != "BELUGA" {
		t.Errorf("expected EnvPrefix 'BELUGA', got %s", options.EnvPrefix)
	}

	if !options.Validate {
		t.Error("expected Validate to be true")
	}

	if !options.SetDefaults {
		t.Error("expected SetDefaults to be true")
	}
}

func TestNewLoader(t *testing.T) {
	options := DefaultLoaderOptions()
	loader, err := NewLoader(options)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	if loader == nil {
		t.Fatal("NewLoader() returned nil")
	}
}

func TestNewProvider_FactoryFunctions(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		factoryFunc func(string, []string, string) (iface.Provider, error)
		name        string
		expectError bool
	}{
		{func(name string, paths []string, prefix string) (iface.Provider, error) {
			return NewYAMLProvider(name, paths, prefix)
		}, "NewYAMLProvider", true}, // Providers return errors when config files don't exist
		{func(name string, paths []string, prefix string) (iface.Provider, error) {
			return NewJSONProvider(name, paths, prefix)
		}, "NewJSONProvider", true}, // Providers return errors when config files don't exist
		{func(name string, paths []string, prefix string) (iface.Provider, error) {
			return NewTOMLProvider(name, paths, prefix)
		}, "NewTOMLProvider", true}, // Providers return errors when config files don't exist
		{func(name string, paths []string, prefix string) (iface.Provider, error) {
			return NewAutoDetectProvider(name, paths, prefix)
		}, "NewAutoDetectProvider", true}, // Providers return errors when config files don't exist
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := tt.factoryFunc("test", []string{tempDir}, "TEST")
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
			if !tt.expectError && provider == nil {
				t.Error("expected provider but got nil")
			}
		})
	}
}

func TestNewCompositeProvider(t *testing.T) {
	tempDir := t.TempDir()
	provider1, err := NewYAMLProvider("test1", []string{tempDir}, "TEST1")
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	provider2, err := NewJSONProvider("test2", []string{tempDir}, "TEST2")
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	composite := NewCompositeProvider(provider1, provider2)
	if composite == nil {
		t.Fatal("NewCompositeProvider() returned nil")
	}
}

func TestValidateConfig_Integration(t *testing.T) {
	// Test ValidateConfig function from main package
	validConfig := &iface.Config{
		LLMProviders: []schema.LLMProviderConfig{
			{
				Name:      "test-llm",
				Provider:  "openai",
				ModelName: "gpt-4",
				APIKey:    "sk-test",
			},
		},
	}

	err := ValidateConfig(validConfig)
	if err != nil {
		t.Errorf("ValidateConfig() expected no error for valid config, got: %v", err)
	}

	invalidConfig := &iface.Config{
		LLMProviders: []schema.LLMProviderConfig{
			{
				Name:     "", // Missing required name
				Provider: "openai",
				APIKey:   "sk-test",
			},
		},
	}

	err = ValidateConfig(invalidConfig)
	if err == nil {
		t.Error("ValidateConfig() expected error for invalid config, got none")
	}
}

func TestSetDefaults_Integration(t *testing.T) {
	// Test SetDefaults function from main package
	config := &iface.Config{
		LLMProviders: []schema.LLMProviderConfig{
			{
				Name:      "test-llm",
				Provider:  "openai",
				ModelName: "gpt-4",
				APIKey:    "sk-test",
			},
		},
	}

	SetDefaults(config)

	if config.LLMProviders[0].DefaultCallOptions == nil {
		t.Error("expected DefaultCallOptions to be initialized")
	}

	// Check temperature default
	if temp, ok := config.LLMProviders[0].DefaultCallOptions["temperature"]; !ok || temp != 0.7 {
		t.Errorf("expected temperature 0.7, got %v", temp)
	}

	// Check max_tokens default
	if maxTokens, ok := config.LLMProviders[0].DefaultCallOptions["max_tokens"]; !ok {
		t.Errorf("expected max_tokens to be set, got missing")
	} else {
		// Convert to float64 for comparison (could be int or float64)
		var maxTokensFloat float64
		switch v := maxTokens.(type) {
		case float64:
			maxTokensFloat = v
		case int:
			maxTokensFloat = float64(v)
		case int64:
			maxTokensFloat = float64(v)
		default:
			t.Errorf("expected max_tokens to be numeric, got %T", maxTokens)
			return
		}
		if maxTokensFloat != 1000.0 {
			t.Errorf("expected max_tokens 1000, got %v", maxTokensFloat)
		}
	}
}

func TestGetEnvConfigMap(t *testing.T) {
	// Set up test environment variables
	testEnvVars := map[string]string{
		"APP_VAR1":  "value1",
		"APP_VAR2":  "value2",
		"OTHER_VAR": "other_value",
	}

	// Set environment variables
	for key, value := range testEnvVars {
		_ = os.Setenv(key, value)
		defer func(k string) { _ = os.Unsetenv(k) }(key)
	}

	envMap := GetEnvConfigMap("APP")

	if len(envMap) != 2 {
		t.Errorf("expected 2 env vars with APP prefix, got %d", len(envMap))
	}

	if envMap["var1"] != "value1" {
		t.Errorf("expected var1=value1, got %s", envMap["var1"])
	}

	if envMap["var2"] != "value2" {
		t.Errorf("expected var2=value2, got %s", envMap["var2"])
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
			name:     "simple conversion",
			prefix:   "APP",
			key:      "database.host",
			expected: "APP_DATABASE_HOST",
		},
		{
			name:     "empty prefix",
			prefix:   "",
			key:      "simple.key",
			expected: "_SIMPLE_KEY",
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
			name:     "simple conversion",
			prefix:   "APP",
			envVar:   "APP_DATABASE_HOST",
			expected: "database.host",
		},
		{
			name:     "empty prefix",
			prefix:   "",
			envVar:   "_SIMPLE_KEY",
			expected: "simple.key",
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

func TestOption_Functions(t *testing.T) {
	// Test the option functions (they're no-ops for now but test they don't panic)
	config := &iface.Config{}

	WithConfigName("test-config")(config)
	WithConfigPaths("./config", "/etc/app")(config)
	WithEnvPrefix("MYAPP")(config)

	// Since they're no-ops, we just verify they don't panic
}

func TestLoadConfig_WithInvalidConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	// Use a config with missing required fields that won't be filtered out
	configContent := `
llm_providers:
  - name: "test-llm"
    # Missing required fields: provider and model_name
    api_key: "sk-test"
`
	err := os.WriteFile(configFile, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	_ = os.Chdir(tempDir)
	defer func() { _ = os.Chdir(oldWd) }()

	// This should fail during validation
	_, err = LoadConfig()
	if err == nil {
		t.Error("expected error for invalid config, got none")
	}
}

func TestConfig_String(t *testing.T) {
	// Test that Config.String() returns a redacted message
	config := &iface.Config{
		LLMProviders: []schema.LLMProviderConfig{
			{
				Name:      "test-openai",
				Provider:  "openai",
				ModelName: "gpt-4",
				APIKey:    "sk-test-secret-key", // This should be redacted
			},
		},
		EmbeddingProviders: []schema.EmbeddingProviderConfig{
			{
				Name:      "test-embeddings",
				Provider:  "openai",
				ModelName: "text-embedding-ada-002",
				APIKey:    "sk-another-secret", // This should be redacted
			},
		},
	}

	result := config.String()

	// Should not contain the actual API keys
	if result == "" {
		t.Error("String() returned empty string")
	}

	expected := "<redacted configuration - sensitive fields not displayed for security>"
	if result != expected {
		t.Errorf("Config.String() = %q, want %q", result, expected)
	}
}
