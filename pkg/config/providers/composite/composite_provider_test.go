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
		errorCode   string
	}{
		{
			name:        "no providers",
			providers:   []iface.Provider{},
			expectError: true,
			errorCode:   iface.ErrCodeInvalidParameters,
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
			errorCode:   iface.ErrCodeAllProvidersFailed,
		},
		{
			name: "nil provider in list",
			providers: []iface.Provider{
				nil,
				&mockProvider{success: true},
			},
			expectError: true,
		},
		{
			name: "provider with specific error",
			providers: []iface.Provider{
				&mockProvider{success: false, errorCode: iface.ErrCodeFileNotFound},
			},
			expectError: true,
			errorCode:   iface.ErrCodeAllProvidersFailed,
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
			if tt.expectError && err != nil && tt.errorCode != "" {
				var configErr *iface.ConfigError
				if iface.AsConfigError(err, &configErr) {
					if configErr.Code != tt.errorCode {
						t.Errorf("expected error code %s, got %s", tt.errorCode, configErr.Code)
					}
				} else {
					t.Error("expected ConfigError but got different error type")
				}
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
	errorCode  string
}

func (m *mockProvider) Load(configStruct interface{}) error {
	if !m.success {
		code := iface.ErrCodeLoadFailed
		if m.errorCode != "" {
			code = m.errorCode
		}
		return iface.NewConfigError(code, "mock load failure")
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
	for _, cfg := range m.llmConfigs {
		if cfg.Name == name {
			return cfg, nil
		}
	}
	return schema.LLMProviderConfig{}, iface.NewConfigError(iface.ErrCodeConfigNotFound, "LLM provider config %s not found", name)
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
	if !m.success {
		return iface.NewConfigError(m.errorCode, "validation failed")
	}
	return nil
}

func (m *mockProvider) SetDefaults() error {
	return nil
}

func TestCompositeProvider_UnmarshalKey(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		providers   []iface.Provider
		expectError bool
	}{
		{
			name: "key found in first provider",
			key:  "test_key",
			providers: []iface.Provider{
				&mockProvider{success: true, setKeys: map[string]bool{"test_key": true}},
			},
			expectError: false,
		},
		{
			name: "key found in second provider",
			key:  "test_key",
			providers: []iface.Provider{
				&mockProvider{success: true, setKeys: map[string]bool{"other_key": true}},
				&mockProvider{success: true, setKeys: map[string]bool{"test_key": true}},
			},
			expectError: false,
		},
		{
			name: "key not found in any provider",
			key:  "missing_key",
			providers: []iface.Provider{
				&mockProvider{success: true, setKeys: map[string]bool{"other_key": true}},
				&mockProvider{success: true, setKeys: map[string]bool{"another_key": true}},
			},
			expectError: true,
		},
		{
			name:        "no providers",
			key:         "test_key",
			providers:   []iface.Provider{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := NewCompositeProvider(tt.providers...)
			var result map[string]interface{}
			err := cp.UnmarshalKey(tt.key, &result)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestCompositeProvider_GetterMethods(t *testing.T) {
	provider1 := &mockProvider{
		values:  map[string]string{"key1": "value1", "shared": "value1"},
		setKeys: map[string]bool{"key1": true, "shared": true},
	}
	provider2 := &mockProvider{
		values:  map[string]string{"key2": "value2", "shared": "value2"},
		setKeys: map[string]bool{"key2": true, "shared": true},
	}

	cp := NewCompositeProvider(provider1, provider2)

	tests := []struct {
		name     string
		key      string
		expected string
		getter   func(string) string
	}{
		{"GetString from first provider", "key1", "value1", cp.GetString},
		{"GetString from second provider", "key2", "value2", cp.GetString},
		{"GetString shared key", "shared", "value1", cp.GetString}, // First provider wins
		{"GetString missing key", "missing", "", cp.GetString},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.getter(tt.key)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCompositeProvider_GetSpecificConfigs(t *testing.T) {
	llmConfig1 := schema.LLMProviderConfig{Name: "llm1", Provider: "openai"}
	llmConfig2 := schema.LLMProviderConfig{Name: "llm2", Provider: "anthropic"}

	provider1 := &mockProvider{
		llmConfigs: []schema.LLMProviderConfig{llmConfig1},
	}
	provider2 := &mockProvider{
		llmConfigs: []schema.LLMProviderConfig{llmConfig2},
	}

	cp := NewCompositeProvider(provider1, provider2)

	// Test GetLLMProviderConfig
	config, err := cp.GetLLMProviderConfig("llm1")
	if err != nil {
		t.Errorf("expected no error getting llm1, got: %v", err)
	}
	if config.Name != "llm1" {
		t.Errorf("expected config name 'llm1', got %s", config.Name)
	}

	_, err = cp.GetLLMProviderConfig("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent LLM provider")
	}

	// Test GetLLMProvidersConfig
	configs, err := cp.GetLLMProvidersConfig()
	if err != nil {
		t.Errorf("expected no error getting LLM providers, got: %v", err)
	}
	if len(configs) != 2 {
		t.Errorf("expected 2 LLM providers, got %d", len(configs))
	}
}

func TestCompositeProvider_Validate(t *testing.T) {
	tests := []struct {
		name        string
		providers   []iface.Provider
		expectError bool
	}{
		{
			name: "all providers validate successfully",
			providers: []iface.Provider{
				&mockProvider{success: true},
				&mockProvider{success: true},
			},
			expectError: false,
		},
		{
			name: "first provider validation fails",
			providers: []iface.Provider{
				&mockProvider{success: false},
				&mockProvider{success: true},
			},
			expectError: true,
		},
		{
			name: "all providers validation fails",
			providers: []iface.Provider{
				&mockProvider{success: false},
				&mockProvider{success: false},
			},
			expectError: true,
		},
		{
			name:        "no providers",
			providers:   []iface.Provider{},
			expectError: false, // Validate on empty composite should succeed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := NewCompositeProvider(tt.providers...)
			err := cp.Validate()

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestCompositeProvider_SetDefaults(t *testing.T) {
	provider1 := &mockProvider{success: true}
	provider2 := &mockProvider{success: true}

	cp := NewCompositeProvider(provider1, provider2)

	err := cp.SetDefaults()
	if err != nil {
		t.Errorf("expected no error from SetDefaults, got: %v", err)
	}
}

func TestCompositeProvider_EdgeCases(t *testing.T) {
	t.Run("nil composite provider", func(t *testing.T) {
		var cp *CompositeProvider

		// These should not panic
		if cp.GetString("test") != "" {
			t.Error("expected empty string from nil provider")
		}
		if cp.IsSet("test") {
			t.Error("expected false from nil provider IsSet")
		}
	})

	t.Run("empty providers list", func(t *testing.T) {
		cp := NewCompositeProvider()

		// These should not panic and return appropriate defaults
		if cp.GetString("test") != "" {
			t.Error("expected empty string from empty provider list")
		}
		if cp.GetInt("test") != 0 {
			t.Error("expected 0 from empty provider list")
		}
		if cp.GetBool("test") {
			t.Error("expected false from empty provider list")
		}
		if cp.GetFloat64("test") != 0.0 {
			t.Error("expected 0.0 from empty provider list")
		}
		if cp.IsSet("test") {
			t.Error("expected false from empty provider list")
		}
	})

	t.Run("provider returns error in getter", func(t *testing.T) {
		failingProvider := &mockProvider{
			success:   false,
			errorCode: iface.ErrCodeConfigNotFound,
		}
		cp := NewCompositeProvider(failingProvider)

		// These should not panic and return defaults
		if cp.GetString("test") != "" {
			t.Error("expected empty string when provider fails")
		}
	})
}
