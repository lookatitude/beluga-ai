package config

import (
	"reflect"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/config/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *iface.Config
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
			name: "invalid LLM provider - missing API key",
			config: &iface.Config{
				LLMProviders: []schema.LLMProviderConfig{
					{
						Name:      "test-openai",
						Provider:  "openai",
						ModelName: "gpt-4",
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
						Name: "test-agent",
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
		{Field: "field1", Message: "error1"},
		{Field: "field2", Message: "error2"},
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
