package schema

import (
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/schema/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema/internal"
)

func TestNewHumanMessage(t *testing.T) {
	content := "Hello, world!"
	msg := NewHumanMessage(content)

	if msg.GetType() != iface.RoleHuman {
		t.Errorf("Expected message type to be RoleHuman, got %v", msg.GetType())
	}

	if msg.GetContent() != content {
		t.Errorf("Expected content to be %q, got %q", content, msg.GetContent())
	}
}

func TestNewAIMessage(t *testing.T) {
	content := "Hello from AI!"
	msg := NewAIMessage(content)

	if msg.GetType() != iface.RoleAssistant {
		t.Errorf("Expected message type to be RoleAssistant, got %v", msg.GetType())
	}

	if msg.GetContent() != content {
		t.Errorf("Expected content to be %q, got %q", content, msg.GetContent())
	}
}

func TestNewSystemMessage(t *testing.T) {
	content := "You are a helpful assistant."
	msg := NewSystemMessage(content)

	if msg.GetType() != iface.RoleSystem {
		t.Errorf("Expected message type to be RoleSystem, got %v", msg.GetType())
	}

	if msg.GetContent() != content {
		t.Errorf("Expected content to be %q, got %q", content, msg.GetContent())
	}
}

func TestNewToolMessage(t *testing.T) {
	content := "Tool execution result"
	toolCallID := "call_123"
	msg := NewToolMessage(content, toolCallID)

	if msg.GetType() != iface.RoleTool {
		t.Errorf("Expected message type to be RoleTool, got %v", msg.GetType())
	}

	if msg.GetContent() != content {
		t.Errorf("Expected content to be %q, got %q", content, msg.GetContent())
	}

	// Note: We can't test the ToolCallID field directly through the Message interface
	// since it's an implementation detail. The interface provides the public API.
}

func TestAIMessage_AdditionalArgs(t *testing.T) {
	msg := NewAIMessage("Hello")

	args := msg.AdditionalArgs()
	if args == nil {
		t.Error("Expected non-nil additional args")
		return
	}

	// AIMessage should have additional args initialized
	if len(args) != 0 {
		// This is expected for a basic AIMessage without tool calls
	}
}

func TestToolMessage_AdditionalArgs(t *testing.T) {
	toolCallID := "call_456"
	msg := NewToolMessage("Result", toolCallID)

	args := msg.AdditionalArgs()
	if args == nil {
		t.Error("Expected non-nil additional args")
		return
	}

	// ToolMessage doesn't store tool_call_id in additional args
	// It's stored in the ToolCallID field
	// Note: We can't do type assertion here since the Message interface
	// doesn't guarantee the underlying type. This test focuses on the interface behavior.
}

func TestBaseMessage_GetContent(t *testing.T) {
	content := "Test message content"
	baseMsg := internal.BaseMessage{
		Content: content,
	}

	if baseMsg.GetContent() != content {
		t.Errorf("Expected content to be %q, got %q", content, baseMsg.GetContent())
	}
}

func TestBaseMessage_AdditionalArgs(t *testing.T) {
	baseMsg := internal.BaseMessage{
		Content: "Test",
	}

	args := baseMsg.AdditionalArgs()
	if args != nil {
		t.Error("Expected nil additional args for BaseMessage")
	}
}

func TestDocument_GetType(t *testing.T) {
	doc := NewDocument("test content", map[string]string{"key": "value"})

	if doc.GetType() != iface.RoleSystem {
		t.Errorf("Expected document type to be RoleSystem, got %v", doc.GetType())
	}
}

func TestDocument_GetContent(t *testing.T) {
	content := "Test document content"
	doc := NewDocument(content, map[string]string{"key": "value"})

	if doc.GetContent() != content {
		t.Errorf("Expected content to be %q, got %q", content, doc.GetContent())
	}
}

// Config validation tests

func TestAgentConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *AgentConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &AgentConfig{
				Name:            "test-agent",
				LLMProviderName: "openai-gpt4",
				MaxIterations:   5,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: &AgentConfig{
				LLMProviderName: "openai-gpt4",
				MaxIterations:   5,
			},
			wantErr: true,
		},
		{
			name: "missing llm provider",
			config: &AgentConfig{
				Name:          "test-agent",
				MaxIterations: 5,
			},
			wantErr: true,
		},
		{
			name: "invalid max iterations",
			config: &AgentConfig{
				Name:            "test-agent",
				LLMProviderName: "openai-gpt4",
				MaxIterations:   0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("AgentConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLLMProviderConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *LLMProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &LLMProviderConfig{
				Name:      "openai-gpt4",
				Provider:  "openai",
				ModelName: "gpt-4-turbo",
				APIKey:    "sk-test",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: &LLMProviderConfig{
				Provider:  "openai",
				ModelName: "gpt-4-turbo",
				APIKey:    "sk-test",
			},
			wantErr: true,
		},
		{
			name: "missing provider",
			config: &LLMProviderConfig{
				Name:      "openai-gpt4",
				ModelName: "gpt-4-turbo",
				APIKey:    "sk-test",
			},
			wantErr: true,
		},
		{
			name: "missing model name",
			config: &LLMProviderConfig{
				Name:     "openai-gpt4",
				Provider: "openai",
				APIKey:   "sk-test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("LLMProviderConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmbeddingProviderConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *EmbeddingProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &EmbeddingProviderConfig{
				Name:      "openai-embeddings",
				Provider:  "openai",
				ModelName: "text-embedding-ada-002",
				APIKey:    "sk-test",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: &EmbeddingProviderConfig{
				Provider:  "openai",
				ModelName: "text-embedding-ada-002",
				APIKey:    "sk-test",
			},
			wantErr: true,
		},
		{
			name: "missing api key",
			config: &EmbeddingProviderConfig{
				Name:      "openai-embeddings",
				Provider:  "openai",
				ModelName: "text-embedding-ada-002",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("EmbeddingProviderConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVectorStoreConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *VectorStoreConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &VectorStoreConfig{
				Name:     "pgvector-store",
				Provider: "pgvector",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: &VectorStoreConfig{
				Provider: "pgvector",
			},
			wantErr: true,
		},
		{
			name: "missing provider",
			config: &VectorStoreConfig{
				Name: "pgvector-store",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("VectorStoreConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewAgentConfig(t *testing.T) {
	tests := []struct {
		name            string
		agentName       string
		llmProviderName string
		opts            []AgentOption
		wantErr         bool
	}{
		{
			name:            "valid config",
			agentName:       "test-agent",
			llmProviderName: "openai-gpt4",
			opts:            []AgentOption{WithMaxIterations(5)},
			wantErr:         false,
		},
		{
			name:            "empty name",
			agentName:       "",
			llmProviderName: "openai-gpt4",
			opts:            []AgentOption{},
			wantErr:         true,
		},
		{
			name:            "empty llm provider",
			agentName:       "test-agent",
			llmProviderName: "",
			opts:            []AgentOption{},
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewAgentConfig(tt.agentName, tt.llmProviderName, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAgentConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && config == nil {
				t.Error("NewAgentConfig() returned nil config without error")
			}
			if !tt.wantErr && config.Name != tt.agentName {
				t.Errorf("NewAgentConfig() config.Name = %q, want %q", config.Name, tt.agentName)
			}
		})
	}
}

func TestNewLLMProviderConfig(t *testing.T) {
	tests := []struct {
		name       string
		configName string
		provider   string
		modelName  string
		opts       []LLMProviderOption
		wantErr    bool
	}{
		{
			name:       "valid config",
			configName: "openai-gpt4",
			provider:   "openai",
			modelName:  "gpt-4-turbo",
			opts:       []LLMProviderOption{WithAPIKey("sk-test")},
			wantErr:    false,
		},
		{
			name:       "empty name",
			configName: "",
			provider:   "openai",
			modelName:  "gpt-4-turbo",
			opts:       []LLMProviderOption{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewLLMProviderConfig(tt.configName, tt.provider, tt.modelName, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLLMProviderConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && config == nil {
				t.Error("NewLLMProviderConfig() returned nil config without error")
			}
		})
	}
}

func TestFunctionalOptions(t *testing.T) {
	// Test AgentOption
	agentConfig, err := NewAgentConfig("test-agent", "openai-gpt4",
		WithToolNames([]string{"tool1", "tool2"}),
		WithMaxIterations(20),
		WithPromptTemplate("You are a helpful assistant"),
		WithAgentType("react"),
	)
	if err != nil {
		t.Fatalf("NewAgentConfig() error = %v", err)
	}

	if len(agentConfig.ToolNames) != 2 {
		t.Errorf("ToolNames length = %d, want 2", len(agentConfig.ToolNames))
	}
	if agentConfig.MaxIterations != 20 {
		t.Errorf("MaxIterations = %d, want 20", agentConfig.MaxIterations)
	}
	if agentConfig.PromptTemplate != "You are a helpful assistant" {
		t.Errorf("PromptTemplate = %q, want %q", agentConfig.PromptTemplate, "You are a helpful assistant")
	}
	if agentConfig.AgentType != "react" {
		t.Errorf("AgentType = %q, want %q", agentConfig.AgentType, "react")
	}

	// Test LLMProviderOption
	llmConfig, err := NewLLMProviderConfig("openai-gpt4", "openai", "gpt-4-turbo",
		WithAPIKey("sk-test"),
		WithBaseURL("https://api.openai.com"),
		WithDefaultCallOptions(map[string]interface{}{"temperature": 0.7}),
	)
	if err != nil {
		t.Fatalf("NewLLMProviderConfig() error = %v", err)
	}

	if llmConfig.APIKey != "sk-test" {
		t.Errorf("APIKey = %q, want %q", llmConfig.APIKey, "sk-test")
	}
	if llmConfig.BaseURL != "https://api.openai.com" {
		t.Errorf("BaseURL = %q, want %q", llmConfig.BaseURL, "https://api.openai.com")
	}
	if llmConfig.DefaultCallOptions["temperature"] != 0.7 {
		t.Errorf("DefaultCallOptions[temperature] = %v, want 0.7", llmConfig.DefaultCallOptions["temperature"])
	}
}
