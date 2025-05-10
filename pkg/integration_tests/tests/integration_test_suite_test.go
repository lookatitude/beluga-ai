package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools/providers"
	"github.com/lookatitude/beluga-ai/pkg/config"
	embeddingsFactory "github.com/lookatitude/beluga-ai/pkg/embeddings/factory"
	"github.com/lookatitude/beluga-ai/pkg/llms/mock"
	// "github.com/lookatitude/beluga-ai/pkg/memory" // Not used in the current version of the test
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConfigProvider is a mock implementation of config.Provider for testing.
// It currently only implements methods used by the factories in this test.
type mockConfigProvider struct {
	cfg *config.Config
}

// Ensure mockConfigProvider implements config.Provider
var _ config.Provider = (*mockConfigProvider)(nil)

func (m *mockConfigProvider) Load(configStruct interface{}) error { return nil } // Mock implementation

func (m *mockConfigProvider) GetString(key string) string                                { return "" }
func (m *mockConfigProvider) GetInt(key string) int                                   { return 0 }
func (m *mockConfigProvider) GetBool(key string) bool                                 { return false }
func (m *mockConfigProvider) GetFloat64(key string) float64                             { return 0.0 }
func (m *mockConfigProvider) GetStringSlice(key string) []string                        { return nil } // Added to satisfy potential interface changes
func (m *mockConfigProvider) GetStringMapString(key string) map[string]string            { return nil } // Added to satisfy potential interface changes
func (m *mockConfigProvider) IsSet(key string) bool                                   { return false }
func (m *mockConfigProvider) UnmarshalKey(key string, rawVal interface{}) error         { return nil }

func (m *mockConfigProvider) GetLLMProviderConfig(name string) (schema.LLMProviderConfig, error) {
	for _, llmCfg := range m.cfg.LLMProviders {
		if llmCfg.Name == name {
			return llmCfg, nil
		}
	}
	return schema.LLMProviderConfig{}, fmt.Errorf("LLM provider config %s not found", name)
}

func (m *mockConfigProvider) GetLLMProvidersConfig() ([]schema.LLMProviderConfig, error) {
	return m.cfg.LLMProviders, nil
}
func (m *mockConfigProvider) GetEmbeddingProvidersConfig() ([]schema.EmbeddingProviderConfig, error) {
	return m.cfg.EmbeddingProviders, nil
}
func (m *mockConfigProvider) GetVectorStoresConfig() ([]schema.VectorStoreConfig, error) {
	return m.cfg.VectorStores, nil
}
func (m *mockConfigProvider) GetToolsConfig() ([]config.ToolConfig, error) {
	return m.cfg.Tools, nil
}

func (m *mockConfigProvider) GetToolConfig(name string) (config.ToolConfig, error) {
    for _, toolCfg := range m.cfg.Tools {
        if toolCfg.Name == name {
            return toolCfg, nil
        }
    }
    return config.ToolConfig{}, fmt.Errorf("tool config %s not found", name)
}

func (m *mockConfigProvider) GetAgentsConfig() ([]schema.AgentConfig, error) {
	return m.cfg.Agents, nil
}
func (m *mockConfigProvider) GetAgentConfig(name string) (schema.AgentConfig, error) {
	for _, agentCfg := range m.cfg.Agents {
		if agentCfg.Name == name {
			return agentCfg, nil
		}
	}
	return schema.AgentConfig{}, fmt.Errorf("agent config %s not found", name)
}

// TestAgentWithVectorStoreMemoryAndOpenAIEmbedder tests an agent using VectorStoreMemory with an OpenAI embedder.
func TestAgentWithVectorStoreMemoryAndOpenAIEmbedder(t *testing.T) {
	ctx := context.Background()

	// 1. Setup Configuration
	// Note: Using schema types directly for slices as config.Config struct defines them this way.
	cfg := &config.Config{
		LLMProviders: []schema.LLMProviderConfig{
			{Provider: "mockllm", Name: "mock-chat", ModelName: "mock-model", APIKey: "test-key"},
		},
		EmbeddingProviders: []schema.EmbeddingProviderConfig{
			{Provider: "openai", Name: "openai-embed", ModelName: "text-embedding-ada-002", APIKey: "sk-testkey"},
		},
		VectorStores: []schema.VectorStoreConfig{
			{Provider: "inmemory", Name: "test-inmemory-vs"},
		},
		Tools: []config.ToolConfig{
			{Provider: "EchoTool", Name: "echo", Description: "Echoes input"},
		},
	}

	// 2. Initialize Components based on config (simplified for this test)
	// Mock LLM
	mockLLMInstance, err := mock.NewMockLLM(config.MockLLMConfig{ModelName: "mock-model", Responses: []string{"Mock response to: Hello, world!"}}) // Use config.MockLLMConfig
	require.NoError(t, err, "Failed to create MockLLM")
	require.NotNil(t, mockLLMInstance, "MockLLM instance is nil")

	// Embedder (using factory)
	mockCfgProvider := &mockConfigProvider{cfg: cfg}
	embedderF, err := embeddingsFactory.NewEmbedderProviderFactory(mockCfgProvider)
	require.NoError(t, err, "Failed to create embedder factory")
	openaiEmbedder, err := embedderF.GetProvider("openai-embed") // Use GetProvider
	require.NoError(t, err, "Failed to get OpenAI embedder from factory")
	require.NotNil(t, openaiEmbedder, "OpenAI embedder is nil")

	// Tool
	var echoToolConfig config.ToolConfig
	foundTool := false
	for _, toolCfg := range cfg.Tools {
		if toolCfg.Name == "echo" {
			echoToolConfig = toolCfg
			foundTool = true
			break
		}
	}
	require.True(t, foundTool, "Echo tool config not found")

	echoToolInstance, err := providers.NewEchoTool(echoToolConfig) // Pass the config.ToolConfig
	require.NoError(t, err, "Failed to create EchoTool")
	require.NotNil(t, echoToolInstance, "EchoTool instance is nil")

	// --- Test Scenario: Basic interaction with Embedder and Mock LLM ---
	t.Run("EmbedderAndLLMInteraction", func(t *testing.T) {
		userInput := "Hello, world!"
		embeddings, err := openaiEmbedder.EmbedQuery(ctx, userInput)
		require.NoError(t, err, "EmbedQuery failed")
		assert.NotEmpty(t, embeddings, "Embeddings should not be empty")
		dim, _ := openaiEmbedder.GetDimension(ctx)
		assert.Len(t, embeddings, dim, fmt.Sprintf("Embeddings should have dimension %d", dim))

		messages := []schema.Message{schema.NewHumanMessage(userInput)}
		aiResponse, err := mockLLMInstance.Chat(ctx, messages)
		require.NoError(t, err, "mockLLMInstance.Chat failed")
		require.NotNil(t, aiResponse, "mockLLMInstance.Chat returned nil response")
		assert.Contains(t, aiResponse.GetContent(), "Mock response to: Hello, world!", "Mock LLM response mismatch")
	})

	t.Log("TestAgentWithVectorStoreMemoryAndOpenAIEmbedder (partially implemented) finished.")
}

