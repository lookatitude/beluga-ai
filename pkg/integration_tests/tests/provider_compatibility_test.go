package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/agents/factory" // Assuming an agent factory exists
	"github.com/lookatitude/beluga-ai/pkg/agents/tools/providers"
	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	"github.com/lookatitude/beluga-ai/pkg/llms/mock"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	vectorstorefactory "github.com/lookatitude/beluga-ai/pkg/vectorstores/factory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProviderCompatibility_MockLLM_OpenAIEmbedder_InMemoryVectorStore tests
// the compatibility of MockLLM, OpenAIEmbedder, and InMemoryVectorStore through an agent execution flow.
func TestProviderCompatibility_MockLLM_OpenAIEmbedder_InMemoryVectorStore(t *testing.T) {
	ctx := context.Background()

	// 1. Setup Configuration
	cfg := &config.Config{
		LLMProviders: []config.LLMProviderConfig{
			{Provider: "mockllm", Name: "mock-chat-for-compat-test", Model: "mock-model", APIKey: "test-key"},
		},
		EmbeddingProviders: []config.EmbeddingProviderConfig{
			{Provider: "openai", Name: "openai-embed-for-compat-test", Model: "text-embedding-ada-002", APIKey: "sk-testkey-openai"},
		},
		VectorStores: []config.VectorStoreConfig{
			{Provider: "inmemory", Name: "inmemory-vs-for-compat-test"},
		},
		Agents: []config.AgentConfig{
			{
				Name:           "test-compat-agent",
				LLMProvider:    "mock-chat-for-compat-test",
				EmbeddingModel: "openai-embed-for-compat-test", // Link to the embedder
				Memory: config.MemoryConfig{
					Type:             "vectorstore",
					VectorStoreName:  "inmemory-vs-for-compat-test", // Link to the vector store
					InputKey:         "user_input",
					OutputKey:        "agent_output",
					HistoryKey:       "conversation_history",
					VectorStoreTopK:  2,
				},
				Tools: []string{"EchoToolCompat"},
			},
		},
		Tools: []config.ToolConfig{
			{Provider: "EchoTool", Name: "EchoToolCompat"},
		},
	}

	// 2. Initialize Components
	// Mock LLM (can be retrieved via a factory if available, or directly instantiated for tests)
	mockLLM := mock.NewMockLLMProvider(mock.MockLLMConfig{ModelName: "mock-model"})
	// Embedder
	embedderFactory := embeddings.NewEmbedderFactory(cfg)
	currentEmbedder, err := embedderFactory.GetEmbedder(ctx, "openai-embed-for-compat-test")
	require.NoError(t, err)
	require.NotNil(t, currentEmbedder)

	// Vector Store
	vsFactory, err := vectorstorefactory.NewVectorStoreFactory(cfg.VectorStores, cfg.EmbeddingProviders, embedderFactory)
	require.NoError(t, err, "Failed to create vector store factory")
	vs, err := vsFactory.GetVectorStore(ctx, "inmemory-vs-for-compat-test")
	require.NoError(t, err)
	require.NotNil(t, vs)

	// Memory (VectorStoreMemory)
	vectorStoreMemory := memory.NewVectorStoreMemory(
		currentEmbedder,
		vs,
		memory.VectorStoreMemoryWithInputKey("user_input"),
		memory.VectorStoreMemoryWithOutputKey("agent_output"),
		memory.VectorStoreMemoryWithMemoryKey("conversation_history"),
		memory.VectorStoreMemoryWithTopK(2),
	)
	require.NotNil(t, vectorStoreMemory)

	// Tool Registry & Tools
	toolRegistry := providers.NewToolRegistry()
	echoTool := providers.NewEchoTool()
	err = toolRegistry.RegisterTool(echoTool, "EchoToolCompat")
	require.NoError(t, err)

	// Agent (Simplified instantiation for test - a full agent factory would use the config)
	// For this test, we manually orchestrate like in phase1_agent_integration_test.go
	// to focus on provider interactions within memory and tool use.

	// --- Test Scenario ---
	userInputPhrase := "What is the capital of France?"
	memoryInput := map[string]interface{}{"user_input": userInputPhrase}

	// 1. Save initial user message to VectorStoreMemory
	err = vectorStoreMemory.SaveContext(ctx, memoryInput, map[string]interface{}{"agent_output": "Thinking..."}) // Simulate initial save
	require.NoError(t, err, "Failed to save initial context to VectorStoreMemory")

	// 2. Load memory - should retrieve the saved context via similarity search
	loadedMemory, err := vectorStoreMemory.LoadMemoryVariables(ctx, memoryInput) // Provide input for potential retrieval context
	require.NoError(t, err, "Failed to load memory variables from VectorStoreMemory")
	retrievedHistory, ok := loadedMemory["conversation_history"].(string) // VectorStoreMemory might return a string summary or formatted history
	require.True(t, ok, "Memory did not return string for history. Got: %T", loadedMemory["conversation_history"])
	assert.Contains(t, retrievedHistory, userInputPhrase, "Retrieved history should contain the user input")

	// 3. Simulate LLM call proposing a tool
	llmResponseWithToolCall := schema.NewAIChatMessage("I should use a tool for this.")
	llmResponseWithToolCall.ToolCalls = []schema.ToolCall{
		{ID: "tool-call-compat-123", Function: schema.ToolFunction{Name: "EchoToolCompat", Arguments: `{"input": "testing compatibility"}`}},
	}

	// 4. Execute the tool
	toolCall := llmResponseWithToolCall.ToolCalls[0]
	var toolArgs map[string]interface{}
	err = json.Unmarshal([]byte(toolCall.Function.Arguments), &toolArgs)
	require.NoError(t, err)
	toolOutput, err := echoTool.Execute(ctx, toolArgs)
	require.NoError(t, err)
	assert.Equal(t, "Echo: testing compatibility", toolOutput)

	// 5. Save tool observation and AI response to VectorStoreMemory
	contextToSave := map[string]interface{}{
		"user_input": userInputPhrase, // or the relevant input for this turn
		// schema.ToolMessage and the AI's response would be part of the history saved
	}
	outputsToSave := map[string]interface{}{
		"agent_output": fmt.Sprintf("Tool said: %s. The capital is Paris.", toolOutput), // Final AI response
	}
	err = vectorStoreMemory.SaveContext(ctx, contextToSave, outputsToSave)
	require.NoError(t, err, "Failed to save tool observation and AI response to VectorStoreMemory")

	// 6. Load memory again to see if the new interaction is stored and retrievable
	loadedMemoryAfterTool, err := vectorStoreMemory.LoadMemoryVariables(ctx, map[string]interface{}{"user_input": "Tell me about that tool use."}) 
	require.NoError(t, err)
	retrievedHistoryAfterTool, ok := loadedMemoryAfterTool["conversation_history"].(string)
	require.True(t, ok)
	assert.Contains(t, retrievedHistoryAfterTool, "testing compatibility", "Retrieved history should contain tool interaction details")
	assert.Contains(t, retrievedHistoryAfterTool, "capital is Paris", "Retrieved history should contain final answer")

	t.Log("Provider Compatibility Test (MockLLM, OpenAIEmbedder, InMemoryVectorStore) Passed!")
}

// TODO: Add more tests for other provider combinations as they become available and testable.
// For example:
// - TestProviderCompatibility_OpenAILLM_OpenAIEmbedder_PineconeVectorStore
// - TestProviderCompatibility_AnthropicLLM_CohereEmbedder_WeaviateVectorStore (once those providers are added)

