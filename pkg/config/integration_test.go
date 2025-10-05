package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/config/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func TestIntegration_LoadConfigWithFileAndEnvOverrides(t *testing.T) {
	tempDir := t.TempDir()

	// Create a base config file
	configFile := filepath.Join(tempDir, "config.yaml")
	configContent := `
llm_providers:
  - name: "openai-gpt4"
    provider: "openai"
    api_key: "file-api-key"
    model_name: "gpt-4"
    default_call_options:
      temperature: 0.5

embedding_providers:
  - name: "openai-embeddings"
    provider: "openai"
    api_key: "file-embed-key"
    model_name: "text-embedding-ada-002"

vector_stores:
  - name: "chroma-db"
    provider: "chroma"
    host: "localhost"
    port: 8000

	agents:
  - name: "assistant"
    llm_provider_name: "openai-gpt4"
    max_iterations: 5

tools:
  - name: "calculator"
    provider: "calculator"
    description: "Performs mathematical calculations"
    enabled: false
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// Set environment variable overrides
	envVars := map[string]string{
		"BELUGA_LLM_PROVIDERS_0_API_KEY":                          "env-api-key",
		"BELUGA_LLM_PROVIDERS_0_DEFAULT_CALL_OPTIONS_TEMPERATURE": "0.8",
		"BELUGA_EMBEDDING_PROVIDERS_0_API_KEY":                    "env-embed-key",
		"BELUGA_AGENTS_0_MAX_ITERATIONS":                          "10",
		"BELUGA_TOOLS_0_ENABLED":                                  "true",
	}

	for key, value := range envVars {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	// Change to temp directory and load config
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Verify file-based config was loaded
	if len(cfg.LLMProviders) != 1 {
		t.Errorf("expected 1 LLM provider, got %d", len(cfg.LLMProviders))
	}

	if len(cfg.EmbeddingProviders) != 1 {
		t.Errorf("expected 1 embedding provider, got %d", len(cfg.EmbeddingProviders))
	}

	if len(cfg.VectorStores) != 1 {
		t.Errorf("expected 1 vector store, got %d", len(cfg.VectorStores))
	}

	if len(cfg.Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(cfg.Agents))
	}

	if len(cfg.Tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(cfg.Tools))
	}

	// Verify environment variable overrides
	llmProvider := cfg.LLMProviders[0]
	if llmProvider.APIKey != "env-api-key" {
		t.Errorf("expected API key from env 'env-api-key', got %s", llmProvider.APIKey)
	}

	if llmProvider.DefaultCallOptions == nil {
		t.Fatal("expected DefaultCallOptions to be set")
	}

	if temp, ok := llmProvider.DefaultCallOptions["temperature"]; !ok || temp != 0.8 {
		t.Errorf("expected temperature from env 0.8, got %v", temp)
	}

	embedProvider := cfg.EmbeddingProviders[0]
	if embedProvider.APIKey != "env-embed-key" {
		t.Errorf("expected embedding API key from env 'env-embed-key', got %s", embedProvider.APIKey)
	}

	agent := cfg.Agents[0]
	if agent.MaxIterations != 10 {
		t.Errorf("expected max iterations from env 10, got %d", agent.MaxIterations)
	}

	tool := cfg.Tools[0]
	if !tool.Enabled {
		t.Errorf("expected tool to be enabled from env, got %v", tool.Enabled)
	}
}

func TestIntegration_CompositeProviderWithMultipleSources(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file provider config
	fileConfig := filepath.Join(tempDir, "file.yaml")
	fileContent := `
llm_providers:
  - name: "file-provider"
    provider: "openai"
    api_key: "file-key"
    model_name: "gpt-4"
`
	err := os.WriteFile(fileConfig, []byte(fileContent), 0644)
	if err != nil {
		t.Fatalf("failed to create file config: %v", err)
	}

	// Create env provider with different provider
	envVars := map[string]string{
		"ENV_LLM_PROVIDERS_0_NAME":     "env-provider",
		"ENV_LLM_PROVIDERS_0_PROVIDER": "anthropic",
		"ENV_LLM_PROVIDERS_0_API_KEY":  "env-key",
	}

	for key, value := range envVars {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	// Create providers
	fileProvider, err := NewYAMLProvider("file", []string{tempDir}, "")
	if err != nil {
		t.Fatalf("failed to create file provider: %v", err)
	}

	envProvider, err := NewYAMLProvider("", nil, "ENV")
	if err != nil {
		t.Fatalf("failed to create env provider: %v", err)
	}

	// Create composite provider
	compositeProvider := NewCompositeProvider(fileProvider, envProvider)

	// Load config
	var cfg iface.Config
	err = compositeProvider.Load(&cfg)
	if err != nil {
		t.Fatalf("composite provider load failed: %v", err)
	}

	// Should have providers from both sources
	if len(cfg.LLMProviders) != 2 {
		t.Errorf("expected 2 LLM providers from composite, got %d", len(cfg.LLMProviders))
	}

	// Find providers by name
	var fileProviderCfg, envProviderCfg schema.LLMProviderConfig
	for _, provider := range cfg.LLMProviders {
		if provider.Name == "file-provider" {
			fileProviderCfg = provider
		} else if provider.Name == "env-provider" {
			envProviderCfg = provider
		}
	}

	if fileProviderCfg.Name != "file-provider" {
		t.Errorf("expected file provider name 'file-provider', got %s", fileProviderCfg.Name)
	}

	if fileProviderCfg.Provider != "openai" {
		t.Errorf("expected file provider type 'openai', got %s", fileProviderCfg.Provider)
	}

	if envProviderCfg.Name != "env-provider" {
		t.Errorf("expected env provider name 'env-provider', got %s", envProviderCfg.Name)
	}

	if envProviderCfg.Provider != "anthropic" {
		t.Errorf("expected env provider type 'anthropic', got %s", envProviderCfg.Provider)
	}
}

func TestIntegration_LoadFromFileWithValidationAndDefaults(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "complete.yaml")
	configContent := `
llm_providers:
  - name: "test-llm"
    provider: "openai"
    api_key: "sk-test"
    model_name: "gpt-4"

embedding_providers:
  - name: "test-embeddings"
    provider: "openai"
    api_key: "sk-embed-test"
    model_name: "text-embedding-ada-002"

vector_stores:
  - name: "test-vectorstore"
    provider: "inmemory"

agents:
  - name: "test-agent"
    llm_provider_name: "test-llm"
    max_iterations: 5

tools:
  - name: "test-tool"
    provider: "echo"
    description: "Test tool"
    enabled: true
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// Load from file (this should validate and set defaults)
	cfg, err := LoadFromFile(configFile)
	if err != nil {
		t.Fatalf("LoadFromFile() failed: %v", err)
	}

	// Verify validation passed
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}

	// Verify defaults were set
	if cfg.LLMProviders[0].DefaultCallOptions == nil {
		t.Error("expected DefaultCallOptions to be set by SetDefaults")
	}

	// Verify cross-references are valid
	agent := cfg.Agents[0]
	if agent.LLMProviderName != "test-llm" {
		t.Errorf("expected agent to reference 'test-llm', got %s", agent.LLMProviderName)
	}

	// Verify the referenced LLM provider exists
	found := false
	for _, llm := range cfg.LLMProviders {
		if llm.Name == agent.LLMProviderName {
			found = true
			break
		}
	}
	if !found {
		t.Error("agent references non-existent LLM provider")
	}
}

func TestIntegration_ConfigRoundTrip(t *testing.T) {
	// Create a config, save it, then load it back
	originalConfig := &iface.Config{
		LLMProviders: []schema.LLMProviderConfig{
			{
				Name:      "roundtrip-llm",
				Provider:  "openai",
				APIKey:    "sk-roundtrip",
				ModelName: "gpt-4",
			},
		},
		EmbeddingProviders: []schema.EmbeddingProviderConfig{
			{
				Name:      "roundtrip-embed",
				Provider:  "openai",
				APIKey:    "sk-embed-roundtrip",
				ModelName: "text-embedding-ada-002",
			},
		},
		Agents: []schema.AgentConfig{
			{
				Name:            "roundtrip-agent",
				LLMProviderName: "roundtrip-llm",
				MaxIterations:   3,
			},
		},
	}

	tempDir := t.TempDir()

	// Create a provider and load the config
	provider, err := NewYAMLProvider("roundtrip", []string{tempDir}, "")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Load the original config into the provider
	err = provider.Load(originalConfig)
	if err != nil {
		t.Fatalf("failed to load original config: %v", err)
	}

	// Now load it back
	var loadedConfig iface.Config
	err = provider.Load(&loadedConfig)
	if err != nil {
		t.Fatalf("failed to load config back: %v", err)
	}

	// Compare key fields
	if len(loadedConfig.LLMProviders) != len(originalConfig.LLMProviders) {
		t.Errorf("LLM providers count mismatch: got %d, want %d",
			len(loadedConfig.LLMProviders), len(originalConfig.LLMProviders))
	}

	if len(loadedConfig.EmbeddingProviders) != len(originalConfig.EmbeddingProviders) {
		t.Errorf("Embedding providers count mismatch: got %d, want %d",
			len(loadedConfig.EmbeddingProviders), len(originalConfig.EmbeddingProviders))
	}

	if len(loadedConfig.Agents) != len(originalConfig.Agents) {
		t.Errorf("Agents count mismatch: got %d, want %d",
			len(loadedConfig.Agents), len(originalConfig.Agents))
	}

	// Check specific values
	if loadedConfig.LLMProviders[0].Name != originalConfig.LLMProviders[0].Name {
		t.Errorf("LLM provider name mismatch: got %s, want %s",
			loadedConfig.LLMProviders[0].Name, originalConfig.LLMProviders[0].Name)
	}

	if loadedConfig.Agents[0].LLMProviderName != originalConfig.Agents[0].LLMProviderName {
		t.Errorf("Agent LLM provider reference mismatch: got %s, want %s",
			loadedConfig.Agents[0].LLMProviderName, originalConfig.Agents[0].LLMProviderName)
	}
}

func TestIntegration_ErrorHandlingAndRecovery(t *testing.T) {
	tempDir := t.TempDir()

	// Test 1: Invalid config file
	invalidConfigFile := filepath.Join(tempDir, "invalid.yaml")
	invalidContent := `
llm_providers:
  - name: ""  # Invalid: empty name
    provider: "openai"
`
	err := os.WriteFile(invalidConfigFile, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("failed to create invalid config file: %v", err)
	}

	_, err = LoadFromFile(invalidConfigFile)
	if err == nil {
		t.Error("expected error for invalid config file")
	}

	// Test 2: Non-existent file
	_, err = LoadFromFile(filepath.Join(tempDir, "nonexistent.yaml"))
	if err == nil {
		t.Error("expected error for non-existent file")
	}

	// Test 3: Recovery with valid file after invalid attempts
	validConfigFile := filepath.Join(tempDir, "valid.yaml")
	validContent := `
llm_providers:
  - name: "recovery-test"
    provider: "openai"
    api_key: "sk-recovery"
    model_name: "gpt-4"
`
	err = os.WriteFile(validConfigFile, []byte(validContent), 0644)
	if err != nil {
		t.Fatalf("failed to create valid config file: %v", err)
	}

	cfg, err := LoadFromFile(validConfigFile)
	if err != nil {
		t.Fatalf("failed to load valid config after invalid attempts: %v", err)
	}

	if cfg == nil || len(cfg.LLMProviders) != 1 {
		t.Error("expected valid config to load successfully")
	}
}

func TestIntegration_ComplexConfigWithMultipleProviders(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "complex.yaml")
	configContent := `
llm_providers:
  - name: "openai-gpt4"
    provider: "openai"
    api_key: "sk-gpt4"
    model_name: "gpt-4"
    default_call_options:
      temperature: 0.7
      max_tokens: 1000
  - name: "anthropic-claude"
    provider: "anthropic"
    api_key: "sk-claude"
    model_name: "claude-3-sonnet"
    default_call_options:
      temperature: 0.8

embedding_providers:
  - name: "openai-embeddings"
    provider: "openai"
    api_key: "sk-embed"
    model_name: "text-embedding-ada-002"

vector_stores:
  - name: "chroma-local"
    provider: "chroma"
    host: "localhost"
    port: 8000
  - name: "pinecone-cloud"
    provider: "pinecone"
    api_key: "pc-cloud-key"
    index_name: "beluga-index"

agents:
  - name: "code-assistant"
    llm_provider_name: "openai-gpt4"
    max_iterations: 10
    tool_names: ["calculator"]
  - name: "creative-writer"
    llm_provider_name: "anthropic-claude"
    max_iterations: 15

tools:
  - name: "calculator"
    provider: "calculator"
    description: "Advanced calculator"
    enabled: true
    config:
      precision: 4
  - name: "web-search"
    provider: "web"
    description: "Web search tool"
    enabled: false
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create complex config file: %v", err)
	}

	cfg, err := LoadFromFile(configFile)
	if err != nil {
		t.Fatalf("failed to load complex config: %v", err)
	}

	// Verify all components loaded correctly
	if len(cfg.LLMProviders) != 2 {
		t.Errorf("expected 2 LLM providers, got %d", len(cfg.LLMProviders))
	}

	if len(cfg.EmbeddingProviders) != 1 {
		t.Errorf("expected 1 embedding provider, got %d", len(cfg.EmbeddingProviders))
	}

	if len(cfg.VectorStores) != 2 {
		t.Errorf("expected 2 vector stores, got %d", len(cfg.VectorStores))
	}

	if len(cfg.Agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(cfg.Agents))
	}

	if len(cfg.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(cfg.Tools))
	}

	// Verify agent references are valid
	for _, agent := range cfg.Agents {
		found := false
		for _, llm := range cfg.LLMProviders {
			if llm.Name == agent.LLMProviderName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("agent %s references non-existent LLM provider %s", agent.Name, agent.LLMProviderName)
		}
	}

	// Verify tool configurations
	for _, tool := range cfg.Tools {
		if tool.Name == "calculator" {
			if !tool.Enabled {
				t.Error("calculator tool should be enabled")
			}
		}
	}
}

func TestIntegration_Performance_LoadLargeConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "large.yaml")

	// Create a moderately large config with multiple providers
	var configContent string
	configContent += "llm_providers:\n"
	for i := 0; i < 10; i++ {
		configContent += `
  - name: "llm-` + string(rune(i+'0')) + `"
    provider: "openai"
    api_key: "sk-test-` + string(rune(i+'0')) + `"
    model_name: "gpt-4"
`
	}

	configContent += "embedding_providers:\n"
	for i := 0; i < 5; i++ {
		configContent += `
  - name: "embed-` + string(rune(i+'0')) + `"
    provider: "openai"
    api_key: "sk-embed-` + string(rune(i+'0')) + `"
    model_name: "text-embedding-ada-002"
`
	}

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create large config file: %v", err)
	}

	cfg, err := LoadFromFile(configFile)
	if err != nil {
		t.Fatalf("failed to load large config: %v", err)
	}

	if len(cfg.LLMProviders) != 10 {
		t.Errorf("expected 10 LLM providers, got %d", len(cfg.LLMProviders))
	}

	if len(cfg.EmbeddingProviders) != 5 {
		t.Errorf("expected 5 embedding providers, got %d", len(cfg.EmbeddingProviders))
	}
}
