package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools/providers"
	"github.com/lookatitude/beluga-ai/pkg/config"

	// TODO: Re-enable when factory packages are implemented
	// embeddingsfactory "github.com/lookatitude/beluga-ai/pkg/embeddings/factory"
	// _ "github.com/lookatitude/beluga-ai/pkg/embeddings/internal/providers/openai"
	// llmsfactory "github.com/lookatitude/beluga-ai/pkg/llms/factory"
	// _ "github.com/lookatitude/beluga-ai/pkg/llms/providers/mock"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/schema"

	// vectorstorefactory "github.com/lookatitude/beluga-ai/pkg/vectorstores/internal/factory"
	_ "github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/inmemory" // Ensure inmemory vector store is registered
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTempCompatTestConfigFile creates a temporary YAML configuration file for compatibility tests.
func createTempCompatTestConfigFile(t *testing.T) (string, string, func()) {
	t.Helper()
	configContent := `
llm_providers:
  - name: "mock-chat-for-compat-test"
    provider: "mock"
    model_name: "mock-model"
    api_key: "test-key"
    provider_specific:
      responses: 
        - "I should use a tool for this."
        - "Tool said: Echo: testing compatibility. The capital is Paris."

embedding_providers:
  - name: "openai-embed-for-compat-test"
    provider: "openai"
    model_name: "text-embedding-ada-002"
    api_key: "sk-validkey123"

vector_stores:
  - name: "inmemory-vs-for-compat-test"
    provider: "inmemory"

agents:
  - name: "test-compat-agent"
    llm_provider_name: "mock-chat-for-compat-test"
    memory_provider_name: "vectorstore-memory-for-compat" 
    memory_type: "vectorstore"
    memory_config_name: "vectorstore-memory-for-compat" 
    tool_names:
      - "EchoToolCompat"

memory_configs:
  - name: "vectorstore-memory-for-compat"
    type: "vectorstore"
    vector_store_name: "inmemory-vs-for-compat-test"
    embedding_provider_name: "openai-embed-for-compat-test" 
    input_key: "user_input"
    output_key: "agent_output"
    history_key: "conversation_history"
    top_k: 2 

tools:
  - name: "EchoToolCompat"
    provider: "EchoTool"
    description: "Echoes input for compatibility test"
`
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_compat_config.yaml")
	err := os.WriteFile(configFile, []byte(configContent), 0600)
	require.NoError(t, err)
	return tempDir, "test_compat_config", func() { os.RemoveAll(tempDir) }
}

type SimpleToolRegistry struct {
	tools map[string]tools.Tool
}

func NewSimpleToolRegistry() *SimpleToolRegistry {
	return &SimpleToolRegistry{tools: make(map[string]tools.Tool)}
}

func (r *SimpleToolRegistry) RegisterTool(tool tools.Tool) error {
	if _, exists := r.tools[tool.GetName()]; exists {
		return fmt.Errorf("tool %s already registered", tool.GetName())
	}
	r.tools[tool.GetName()] = tool
	return nil
}

func (r *SimpleToolRegistry) GetTool(name string) (tools.Tool, error) {
	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}
	return tool, nil
}

func TestProviderCompatibility_MockLLM_OpenAIEmbedder_InMemoryVectorStore(t *testing.T) {
	ctx := context.Background()

	tempDir, configName, cleanup := createTempCompatTestConfigFile(t)
	defer cleanup()

	// Set a dummy OpenAI API key environment variable for the test - this is a fallback, config should take precedence
	const dummyEnvApiKey = "sk-envkeyshouldnotbeused"
	originalApiKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", dummyEnvApiKey)
	defer os.Setenv("OPENAI_API_KEY", originalApiKey)

	vp, err := config.NewViperProvider(configName, []string{tempDir}, "BELUGA_COMPAT_TEST")
	require.NoError(t, err, "Failed to create ViperProvider")
	require.NotNil(t, vp, "ViperProvider is nil")

	llmProviderFactory, err := llmsfactory.NewLLMProviderFactory(vp)
	require.NoError(t, err, "Failed to create LLMProviderFactory")
	mockLLM, err := llmProviderFactory.GetProvider("mock-chat-for-compat-test")
	require.NoError(t, err, "Failed to get mock LLM provider")
	require.NotNil(t, mockLLM, "Mock LLM provider is nil")

	embedderProviderFactory, err := embeddingsfactory.NewEmbedderProviderFactory(vp)
	require.NoError(t, err, "Failed to create EmbedderProviderFactory. Check factory logs for APIKey issues.")

	currentEmbedder, err := embedderProviderFactory.GetProvider("openai-embed-for-compat-test")
	require.NoError(t, err, "Failed to get OpenAI embedder provider from factory.")
	require.NotNil(t, currentEmbedder)

	vsConfigs, err := vp.GetVectorStoresConfig()
	require.NoError(t, err)
	var targetVSConfig schema.VectorStoreConfig
	foundVS := false
	for _, vsCfg := range vsConfigs {
		if vsCfg.Name == "inmemory-vs-for-compat-test" {
			targetVSConfig = vsCfg
			foundVS = true
			break
		}
	}
	require.True(t, foundVS, "Vector store config not found")

	vsFactory := vectorstorefactory.NewVectorStoreFactory()
	vs, err := vsFactory.Create(targetVSConfig.Provider, targetVSConfig.ProviderSpecific)
	require.NoError(t, err)
	require.NotNil(t, vs)

	memInputKey := "user_input"
	memOutputKey := "agent_output"
	memHistoryKey := "conversation_history"
	memTopK := 2

	vectorStoreMemory := memory.NewVectorStoreMemory(
		currentEmbedder,
		vs,
		memory.WithInputKey(memInputKey),
		memory.WithOutputKey(memOutputKey),
		memory.WithMemoryKey(memHistoryKey),
		memory.WithK(memTopK),
	)
	require.NotNil(t, vectorStoreMemory)

	toolRegistry := NewSimpleToolRegistry()
	echoToolCfg, err := vp.GetToolConfig("EchoToolCompat")
	require.NoError(t, err, "Failed to get EchoToolCompat config")
	echoTool, err := providers.NewEchoTool(echoToolCfg)
	require.NoError(t, err, "Failed to create EchoTool")
	err = toolRegistry.RegisterTool(echoTool)
	require.NoError(t, err)

	userInputPhrase := "What is the capital of France?"
	memoryInput := map[string]interface{}{memInputKey: userInputPhrase}

	err = vectorStoreMemory.SaveContext(ctx, memoryInput, map[string]string{memOutputKey: "Thinking..."})
	require.NoError(t, err, "Failed to save initial context to VectorStoreMemory")

	loadedMemory, err := vectorStoreMemory.LoadMemoryVariables(ctx, memoryInput)
	require.NoError(t, err, "Failed to load memory variables from VectorStoreMemory")
	retrievedHistory, ok := loadedMemory[memHistoryKey].(string)
	require.True(t, ok, "Memory did not return string for history. Got: %T", loadedMemory[memHistoryKey])
	assert.Contains(t, retrievedHistory, userInputPhrase, "Retrieved history should contain the user input")

	aiResponseString, err := mockLLM.Invoke(ctx, userInputPhrase)
	require.NoError(t, err, "mockLLM.Invoke failed for initial query")
	require.Equal(t, "I should use a tool for this.", aiResponseString, "Unexpected first LLM response")

	constructedToolCall := schema.ToolCall{
		ID:       "tool-call-compat-123",
		Function: schema.FunctionCall{Name: "EchoToolCompat", Arguments: `{"input": "testing compatibility"}`},
	}

	toolToExecute, err := toolRegistry.GetTool(constructedToolCall.Function.Name)
	require.NoError(t, err, "Failed to get EchoToolCompat from registry")

	var toolArgs map[string]interface{}
	err = json.Unmarshal([]byte(constructedToolCall.Function.Arguments), &toolArgs)
	require.NoError(t, err)
	toolOutput, err := toolToExecute.Execute(ctx, toolArgs)
	require.NoError(t, err)
	assert.Equal(t, "Echo: testing compatibility", toolOutput)

	contextToSave := map[string]interface{}{memInputKey: userInputPhrase}
	finalAnswer := "Tool said: Echo: testing compatibility. The capital is Paris."
	outputsToSave := map[string]string{memOutputKey: finalAnswer}
	err = vectorStoreMemory.SaveContext(ctx, contextToSave, outputsToSave)
	require.NoError(t, err, "Failed to save tool observation and AI response to VectorStoreMemory")

	loadedMemoryAfterTool, err := vectorStoreMemory.LoadMemoryVariables(ctx, map[string]interface{}{memInputKey: "Tell me about that tool use."})
	require.NoError(t, err)
	retrievedHistoryAfterTool, ok := loadedMemoryAfterTool[memHistoryKey].(string)
	require.True(t, ok)
	assert.Contains(t, retrievedHistoryAfterTool, "testing compatibility", "Retrieved history should contain tool interaction details")
	assert.Contains(t, retrievedHistoryAfterTool, "capital is Paris", "Retrieved history should contain final answer")

	t.Log("Provider Compatibility Test (MockLLM, OpenAIEmbedder, InMemoryVectorStore) Passed!")
}
