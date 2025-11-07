package viper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/config/iface"
)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestNewViperProvider(t *testing.T) {
	tests := []struct {
		name        string
		configName  string
		configPaths []string
		envPrefix   string
		format      string
		expectError bool
	}{
		{
			name:        "valid provider with config",
			configName:  "test",
			configPaths: []string{"./testdata"},
			envPrefix:   "TEST",
			format:      "yaml",
			expectError: false,
		},
		{
			name:        "valid provider without config",
			configName:  "",
			configPaths: nil,
			envPrefix:   "TEST",
			format:      "",
			expectError: false,
		},
		{
			name:        "valid provider with auto format",
			configName:  "test",
			configPaths: []string{"./testdata"},
			envPrefix:   "TEST",
			format:      "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewViperProvider(tt.configName, tt.configPaths, tt.envPrefix, tt.format)
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
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func TestViperProvider_Load(t *testing.T) {
	// Create a temporary YAML config file
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

	provider, err := NewViperProvider("test", []string{tempDir}, "", "yaml")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	var cfg iface.Config
	err = provider.Load(&cfg)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.LLMProviders) != 1 {
		t.Errorf("expected 1 LLM provider, got %d", len(cfg.LLMProviders))
	}
	if cfg.LLMProviders[0].Name != "test-llm" {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		t.Errorf("expected LLM provider name 'test-llm', got %s", cfg.LLMProviders[0].Name)
	}
}

func TestViperProvider_Getters(t *testing.T) {
	provider, err := NewViperProvider("", nil, "", "")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Set some test values
	provider.v.Set("string_key", "test_value")
	provider.v.Set("int_key", 42)
	provider.v.Set("bool_key", true)
	provider.v.Set("float_key", 3.14)
	provider.v.Set("map_key", map[string]string{"key1": "value1", "key2": "value2"})

	tests := []struct {
		name     string
		key      string
		expected interface{}
		getter   func(string) interface{}
	}{
		{"string getter", "string_key", "test_value", func(k string) interface{} { return provider.GetString(k) }},
		{"int getter", "int_key", 42, func(k string) interface{} { return provider.GetInt(k) }},
		{"bool getter", "bool_key", true, func(k string) interface{} { return provider.GetBool(k) }},
		{"float getter", "float_key", 3.14, func(k string) interface{} { return provider.GetFloat64(k) }},
		{"map getter", "map_key", map[string]string{"key1": "value1", "key2": "value2"}, func(k string) interface{} { return provider.GetStringMapString(k) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.getter(tt.key)

			// Handle map comparison separately since maps can't be compared directly
			if tt.name == "map getter" {
				resultMap, ok := result.(map[string]string)
				if !ok {
					t.Errorf("expected map[string]string, got %T", result)
					return
				}
				expectedMap, ok := tt.expected.(map[string]string)
				if !ok {
					t.Errorf("expected map[string]string for expected value")
					return
				}
				if len(resultMap) != len(expectedMap) {
					t.Errorf("expected map length %d, got %d", len(expectedMap), len(resultMap))
					return
				}
				for k, v := range expectedMap {
					if resultMap[k] != v {
						t.Errorf("expected map[%s]=%s, got %s", k, v, resultMap[k])
					}
				}
			} else if result != tt.expected {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestViperProvider_IsSet(t *testing.T) {
	provider, err := NewViperProvider("", nil, "", "")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	provider.v.Set("existing_key", "value")

	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{"existing key", "existing_key", true},
		{"non-existing key", "non_existing_key", false},
		{"empty key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			result := provider.IsSet(tt.key)
			if result != tt.expected {
				t.Errorf("expected %v for key %s, got %v", tt.expected, tt.key, result)
			}
		})
	}
}

func TestViperProvider_GetLLMProvidersConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")
	configContent := `
llm_providers:
  - name: "openai-gpt4"
    provider: "openai"
    api_key: "sk-test"
    model_name: "gpt-4"
  - name: "anthropic-claude"
    provider: "anthropic"
    api_key: "sk-ant-test"
    model_name: "claude-3"
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	provider, err := NewViperProvider("test", []string{tempDir}, "", "yaml")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	configs, err := provider.GetLLMProvidersConfig()
	if err != nil {
		t.Fatalf("failed to get LLM providers config: %v", err)
	}

	if len(configs) != 2 {
		t.Errorf("expected 2 LLM providers, got %d", len(configs))
	}

	// Check first provider
	if configs[0].Name != "openai-gpt4" {
		t.Errorf("expected first provider name 'openai-gpt4', got %s", configs[0].Name)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	if configs[0].Provider != "openai" {
		t.Errorf("expected first provider type 'openai', got %s", configs[0].Provider)
	}

	// Check second provider
	if configs[1].Name != "anthropic-claude" {
		t.Errorf("expected second provider name 'anthropic-claude', got %s", configs[1].Name)
	}
}

func TestViperProvider_GetLLMProviderConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")
	configContent := `
llm_providers:
  - name: "openai-gpt4"
    provider: "openai"
    api_key: "sk-test"
    model_name: "gpt-4"
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	provider, err := NewViperProvider("test", []string{tempDir}, "", "yaml")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Test existing provider
	config, err := provider.GetLLMProviderConfig("openai-gpt4")
	if err != nil {
		t.Fatalf("failed to get existing LLM provider config: %v", err)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	}
	if config.Name != "openai-gpt4" {
		t.Errorf("expected provider name 'openai-gpt4', got %s", config.Name)
	}

	// Test non-existing provider
	_, err = provider.GetLLMProviderConfig("non-existing")
	if err == nil {
		t.Error("expected error for non-existing provider, got nil")
	}
}

func TestViperProvider_GetEmbeddingProvidersConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")
	configContent := `
embedding_providers:
  - name: "openai-embeddings"
    provider: "openai"
    api_key: "sk-test"
    model_name: "text-embedding-ada-002"
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	provider, err := NewViperProvider("test", []string{tempDir}, "", "yaml")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	configs, err := provider.GetEmbeddingProvidersConfig()
	if err != nil {
		t.Fatalf("failed to get embedding providers config: %v", err)
	}

	if len(configs) != 1 {
		t.Errorf("expected 1 embedding provider, got %d", len(configs))
	}
	if configs[0].Name != "openai-embeddings" {
		t.Errorf("expected provider name 'openai-embeddings', got %s", configs[0].Name)
	}
}

func TestViperProvider_GetVectorStoresConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")
	configContent := `
vector_stores:
  - name: "chroma-db"
    provider: "chroma"
    host: "localhost"
    port: 8000
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	provider, err := NewViperProvider("test", []string{tempDir}, "", "yaml")
	if err != nil {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		t.Fatalf("failed to create provider: %v", err)
	}

	configs, err := provider.GetVectorStoresConfig()
	if err != nil {
		t.Fatalf("failed to get vector stores config: %v", err)
	}

	if len(configs) != 1 {
		t.Errorf("expected 1 vector store, got %d", len(configs))
	}
	if configs[0].Name != "chroma-db" {
		t.Errorf("expected vector store name 'chroma-db', got %s", configs[0].Name)
	}
}

func TestViperProvider_GetAgentsConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")
	configContent := `
agents:
  - name: "assistant"
    description: "General purpose AI assistant"
    llm_provider_name: "openai-gpt4"
    max_iterations: 10
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	provider, err := NewViperProvider("test", []string{tempDir}, "", "yaml")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	configs, err := provider.GetAgentsConfig()
	if err != nil {
		t.Fatalf("failed to get agents config: %v", err)
	}

	if len(configs) != 1 {
		t.Errorf("expected 1 agent, got %d", len(configs))
	}
	if configs[0].Name != "assistant" {
		t.Errorf("expected agent name 'assistant', got %s", configs[0].Name)
	}
}

func TestViperProvider_GetAgentConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")
	configContent := `
agents:
  - name: "assistant"
    description: "General purpose AI assistant"
    llm_provider_name: "openai-gpt4"
    max_iterations: 10
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	provider, err := NewViperProvider("test", []string{tempDir}, "", "yaml")
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Test existing agent
	config, err := provider.GetAgentConfig("assistant")
	if err != nil {
		t.Fatalf("failed to get existing agent config: %v", err)
	}
	if config.Name != "assistant" {
		t.Errorf("expected agent name 'assistant', got %s", config.Name)
	}

	// Test non-existing agent
	_, err = provider.GetAgentConfig("non-existing")
	if err == nil {
		t.Error("expected error for non-existing agent, got nil")
	}
}

func TestViperProvider_GetToolsConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")
	configContent := `
tools:
  - name: "calculator"
    provider: "calculator"
    description: "Performs mathematical calculations"
    enabled: true
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	provider, err := NewViperProvider("test", []string{tempDir}, "", "yaml")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	configs, err := provider.GetToolsConfig()
	if err != nil {
		t.Fatalf("failed to get tools config: %v", err)
	}

	if len(configs) != 1 {
		t.Errorf("expected 1 tool, got %d", len(configs))
	}
	if configs[0].Name != "calculator" {
		t.Errorf("expected tool name 'calculator', got %s", configs[0].Name)
	}
}

func TestViperProvider_GetToolConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")
	configContent := `
tools:
  - name: "calculator"
    provider: "calculator"
    description: "Performs mathematical calculations"
    enabled: true
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		t.Fatalf("failed to create test config file: %v", err)
	}

	provider, err := NewViperProvider("test", []string{tempDir}, "", "yaml")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Test existing tool
	config, err := provider.GetToolConfig("calculator")
	if err != nil {
		t.Fatalf("failed to get existing tool config: %v", err)
	}
	if config.Name != "calculator" {
		t.Errorf("expected tool name 'calculator', got %s", config.Name)
	}

	// Test non-existing tool
	_, err = provider.GetToolConfig("non-existing")
	if err == nil {
		t.Error("expected error for non-existing tool, got nil")
	}
}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

func TestViperProvider_UnmarshalKey(t *testing.T) {
	provider, err := NewViperProvider("", nil, "", "")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Set test data
	testData := map[string]interface{}{
		"nested": map[string]interface{}{
			"key": "value",
		},
	}
	provider.v.Set("test_key", testData)

	var result map[string]interface{}
	err = provider.UnmarshalKey("test_key", &result)
	if err != nil {
		t.Fatalf("failed to unmarshal key: %v", err)
	}

	if result["nested"].(map[string]interface{})["key"] != "value" {
		t.Errorf("expected unmarshalled value 'value', got %v", result["nested"])
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func TestViperProvider_Validate(t *testing.T) {
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

	provider, err := NewViperProvider("test", []string{tempDir}, "", "yaml")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	err = provider.Validate()
	if err != nil {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		t.Errorf("expected validation to succeed, got error: %v", err)
	}
}

func TestViperProvider_SetDefaults(t *testing.T) {
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

	provider, err := NewViperProvider("test", []string{tempDir}, "", "yaml")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	err = provider.SetDefaults()
	if err != nil {
		t.Errorf("expected SetDefaults to succeed, got error: %v", err)
	}
}

func TestViperProvider_EnvironmentVariables(t *testing.T) {
	// Set up environment variables
	testEnvVars := map[string]string{
		"TEST_LLM_PROVIDERS_0_NAME":       "env-llm",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		"TEST_LLM_PROVIDERS_0_PROVIDER":   "openai",
		"TEST_LLM_PROVIDERS_0_API_KEY":    "env-key",
		"TEST_LLM_PROVIDERS_0_MODEL_NAME": "gpt-4",
	}

	// Set environment variables
	for key, value := range testEnvVars {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	provider, err := NewViperProvider("", nil, "TEST", "")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	llmConfigs, err := provider.GetLLMProvidersConfig()
	if err != nil {
		t.Fatalf("failed to get LLM providers config: %v", err)
	}

	if len(llmConfigs) != 1 {
		t.Errorf("expected 1 LLM provider from env vars, got %d", len(llmConfigs))
	}
	if llmConfigs[0].Name != "env-llm" {
		t.Errorf("expected LLM provider name 'env-llm', got %s", llmConfigs[0].Name)
	}
	if llmConfigs[0].APIKey != "env-key" {
		t.Errorf("expected API key 'env-key', got %s", llmConfigs[0].APIKey)
	}
}

func TestViperProvider_JSONSupport(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.json")
	configContent := `{
  "llm_providers": [
    {
      "name": "json-llm",
      "provider": "openai",
      "api_key": "json-key",
      "model_name": "gpt-4"
    }
  ]
}`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	provider, err := NewViperProvider("test", []string{tempDir}, "", "json")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	llmConfigs, err := provider.GetLLMProvidersConfig()
	if err != nil {
		t.Fatalf("failed to get LLM providers config: %v", err)
	}

	if len(llmConfigs) != 1 {
		t.Errorf("expected 1 LLM provider, got %d", len(llmConfigs))
	}
	if llmConfigs[0].Name != "json-llm" {
		t.Errorf("expected LLM provider name 'json-llm', got %s", llmConfigs[0].Name)
	}
}
