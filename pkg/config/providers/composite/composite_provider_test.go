package composite

import (
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/config/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func TestCompositeProvider_Load(t *testing.T) {
	tests := []struct {
		name        string
		providers   []iface.Provider
		expectError bool
	}{
		{
			name:        "no providers",
			providers:   []iface.Provider{},
			expectError: true,
		},
		{
			name: "single provider success",
			providers: []iface.Provider{
				&mockProvider{success: true},
			},
			expectError: false,
		},
		{
			name: "multiple providers - first succeeds",
			providers: []iface.Provider{
				&mockProvider{success: true},
				&mockProvider{success: false},
			},
			expectError: false,
		},
		{
			name: "multiple providers - first fails, second succeeds",
			providers: []iface.Provider{
				&mockProvider{success: false},
				&mockProvider{success: true},
			},
			expectError: false,
		},
		{
			name: "all providers fail",
			providers: []iface.Provider{
				&mockProvider{success: false},
				&mockProvider{success: false},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := NewCompositeProvider(tt.providers...)
			var cfg iface.Config
			err := cp.Load(&cfg)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestCompositeProvider_GetString(t *testing.T) {
	cp := NewCompositeProvider(
		&mockProvider{
			values:  map[string]string{"key1": "value1"},
			setKeys: map[string]bool{"key1": true},
		},
		&mockProvider{
			values:  map[string]string{"key2": "value2"},
			setKeys: map[string]bool{"key2": true},
		},
	)

	tests := []struct {
		key      string
		expected string
	}{
		{"key1", "value1"},
		{"key2", "value2"},
		{"key3", ""},
	}

	for _, tt := range tests {
		result := cp.GetString(tt.key)
		if result != tt.expected {
			t.Errorf("GetString(%s) = %s, want %s", tt.key, result, tt.expected)
		}
	}
}

func TestCompositeProvider_IsSet(t *testing.T) {
	cp := NewCompositeProvider(
		&mockProvider{setKeys: map[string]bool{"key1": true}},
		&mockProvider{setKeys: map[string]bool{"key2": true}},
	)

	tests := []struct {
		key      string
		expected bool
	}{
		{"key1", true},
		{"key2", true},
		{"key3", false},
	}

	for _, tt := range tests {
		result := cp.IsSet(tt.key)
		if result != tt.expected {
			t.Errorf("IsSet(%s) = %v, want %v", tt.key, result, tt.expected)
		}
	}
}

func TestCompositeProvider_GetLLMProvidersConfig(t *testing.T) {
	llmConfig1 := schema.LLMProviderConfig{Name: "provider1", Provider: "openai"}
	llmConfig2 := schema.LLMProviderConfig{Name: "provider2", Provider: "anthropic"}

	cp := NewCompositeProvider(
		&mockProvider{llmConfigs: []schema.LLMProviderConfig{llmConfig1}},
		&mockProvider{llmConfigs: []schema.LLMProviderConfig{llmConfig2}},
	)

	configs, err := cp.GetLLMProvidersConfig()
	if err != nil {
		t.Fatalf("GetLLMProvidersConfig() error = %v", err)
	}

	if len(configs) != 2 {
		t.Errorf("expected 2 configs, got %d", len(configs))
	}

	// Check that both configs are present (order may vary due to map)
	configNames := make(map[string]bool)
	for _, config := range configs {
		configNames[config.Name] = true
	}

	if !configNames["provider1"] || !configNames["provider2"] {
		t.Errorf("expected both provider1 and provider2, got %v", configNames)
	}
}

// mockProvider is a mock implementation of iface.Provider for testing
type mockProvider struct {
	success    bool
	values     map[string]string
	setKeys    map[string]bool
	llmConfigs []schema.LLMProviderConfig
}

func (m *mockProvider) Load(configStruct interface{}) error {
	if !m.success {
		return iface.NewConfigError(iface.ErrCodeLoadFailed, "mock load failure")
	}
	return nil
}

func (m *mockProvider) UnmarshalKey(key string, rawVal interface{}) error {
	return nil
}

func (m *mockProvider) GetString(key string) string {
	if m.values != nil {
		if val, ok := m.values[key]; ok {
			return val
		}
	}
	return ""
}

func (m *mockProvider) GetInt(key string) int {
	return 0
}

func (m *mockProvider) GetBool(key string) bool {
	return false
}

func (m *mockProvider) GetFloat64(key string) float64 {
	return 0.0
}

func (m *mockProvider) GetStringMapString(key string) map[string]string {
	return nil
}

func (m *mockProvider) IsSet(key string) bool {
	if m.setKeys != nil {
		return m.setKeys[key]
	}
	return false
}

func (m *mockProvider) GetLLMProviderConfig(name string) (schema.LLMProviderConfig, error) {
	return schema.LLMProviderConfig{}, nil
}

func (m *mockProvider) GetLLMProvidersConfig() ([]schema.LLMProviderConfig, error) {
	return m.llmConfigs, nil
}

func (m *mockProvider) GetEmbeddingProvidersConfig() ([]schema.EmbeddingProviderConfig, error) {
	return nil, nil
}

func (m *mockProvider) GetVectorStoresConfig() ([]schema.VectorStoreConfig, error) {
	return nil, nil
}

func (m *mockProvider) GetAgentConfig(name string) (schema.AgentConfig, error) {
	return schema.AgentConfig{}, nil
}

func (m *mockProvider) GetAgentsConfig() ([]schema.AgentConfig, error) {
	return nil, nil
}

func (m *mockProvider) GetToolConfig(name string) (iface.ToolConfig, error) {
	return iface.ToolConfig{}, nil
}

func (m *mockProvider) GetToolsConfig() ([]iface.ToolConfig, error) {
	return nil, nil
}

func (m *mockProvider) Validate() error {
	return nil
}

func (m *mockProvider) SetDefaults() error {
	return nil
}
