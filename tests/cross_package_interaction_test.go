package tests

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/config"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms"

	// Provider imports removed - they would require internal package access
	// TODO: Consider moving provider registration to public APIs
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCrossPackageInteractions covers interactions between config, llms, embeddings, memory, and vectorstores.
func TestCrossPackageInteractions(t *testing.T) {
	ctx := context.Background()

	// 1. Setup Configuration for mockConfigProvider
	cfgInstance := &config.Config{
		LLMProviders: []schema.LLMProviderConfig{
			{
				Name:      "mock_llm_for_cross_test",
				Provider:  "mock",
				ModelName: "mock-model", // Corrected field name
				APIKey:    "mock-api-key",
				DefaultCallOptions: map[string]interface{}{ // Corrected field name
					"max_tokens": 256,
					// Ensure mock LLM handles this or add specific mock config if needed
				},
			},
		},
		EmbeddingProviders: []schema.EmbeddingProviderConfig{
			{
				Name:      "mock_embedder_for_cross_test",
				Provider:  "mock", // Using mock embedder for simplicity
				ModelName: "mock-embed-model",
				APIKey:    "mock-api-key", // Mock API key
				ProviderSpecific: map[string]interface{}{ // For mock embedder
					"dimension": 128,
				},
			},
		},
		VectorStores: []schema.VectorStoreConfig{
			{
				Name:     "inmemory_vs_for_cross_test",
				Provider: "inmemory",
				// Embedding field can be omitted if default is picked up or not strictly needed by inmemory for this test setup
				ProviderSpecific: map[string]interface{}{}, // No specific config for inmemory here
			},
		},
		Application: appCfg,
	}

	mockCP := &mockConfigProvider{
		cfg: cfgInstance,
	}

	// 2. Initialize an LLM using direct construction (factories not implemented yet)
	// TODO: Implement factory pattern for LLMs
	llmInstance := &mockLLMForTest{}
	require.NotNil(t, llmInstance)

	// 3. Initialize an Embedder using direct construction (factories not implemented yet)
	// TODO: Implement factory pattern for Embedders
	embedderInstance := &mockEmbedderForTest{}
	require.NotNil(t, embedderInstance)

	// 4. Initialize a VectorStore using direct construction (factories not implemented yet)
	// TODO: Implement factory pattern for VectorStores
	vsFactory := &mockVectorStoreFactory{}

	// Get the specific config for "inmemory_vs_for_cross_test"
	var vsProviderConfig schema.VectorStoreConfig
	foundVS := false
	for _, vsCfg := range mockCP.cfg.VectorStores {
		if vsCfg.Name == "inmemory_vs_for_cross_test" {
			vsProviderConfig = vsCfg
			foundVS = true
			break
		}
	}
	require.True(t, foundVS, "Vector store config 	'inmemory_vs_for_cross_test	' not found")

	vectorStoreInstance, err := vsFactory.Create(vsProviderConfig.Provider, vsProviderConfig.ProviderSpecific)
	require.NoError(t, err)
	require.NotNil(t, vectorStoreInstance)

	// 5. Initialize Memory (e.g., BufferMemory)
	// For VectorStoreRetrieverMemory, it would need the retriever
	retriever := vectorStoreInstance.AsRetriever(vectorstoresiface.WithScoreThreshold(0.7)) // Assuming WithScoreThreshold is available
	require.NotNil(t, retriever)

	// Example with BufferMemory as it's simpler and was in the original attempt
	bufferMem := memory.NewBufferMemory(
		memory.WithChatHistory(schema.NewChatHistory()),
		memory.WithReturnMessages(true),
		memory.WithInputKey("input"),
		memory.WithMemoryKey("history"),
	)
	require.NotNil(t, bufferMem)

	// 6. Using the LLM to generate text
	llmInput := "Hello, LLM!"
	llmResult, err := llmInstance.Generate(ctx, []schema.Message{schema.NewHumanMessage(llmInput)}, llmsiface.WithStreaming(false))
	require.NoError(t, err)
	require.NotNil(t, llmResult)
	assert.NotEmpty(t, llmResult.Generations)
	t.Logf("LLM Generation: %s", llmResult.Generations[0].Text)

	// 7. Using the Embedder to create embeddings
	docToEmbed := "This is a document to embed."
	embeddingsList, err := embedderInstance.EmbedDocuments(ctx, []string{docToEmbed})
	require.NoError(t, err)
	require.Len(t, embeddingsList, 1)
	assert.NotEmpty(t, embeddingsList[0])
	dimension, _ := embedderInstance.GetDimension(ctx) // Assuming GetDimension exists and works for mock
	assert.Len(t, embeddingsList[0], dimension)
	t.Logf("Document embedded, dimension: %d", len(embeddingsList[0]))

	// 8. Using the VectorStore to add and search documents
	docs := []schema.Document{
		{PageContent: "Document 1: about apples", Metadata: map[string]interface{}{"source": "doc1"}},
		{PageContent: "Document 2: about oranges", Metadata: map[string]interface{}{"source": "doc2"}},
	}
	_, err = vectorStoreInstance.AddDocuments(ctx, docs, vectorstoresiface.WithEmbeddingProvider(embedderInstance)) // Pass embedder if AddDocuments requires it for in-memory
	require.NoError(t, err)

	query := "Tell me about apples"
	// For SimilaritySearchByQuery, ensure embedder is implicitly used or passed if needed by the inmemory store's implementation
	searchResults, err := vectorStoreInstance.SimilaritySearchByQuery(ctx, query, 2, vectorstoresiface.WithScoreThreshold(0.1), vectorstoresiface.WithEmbeddingProvider(embedderInstance))
	require.NoError(t, err)
	assert.NotEmpty(t, searchResults)
	t.Logf("VectorStore search results for 	'%s	': %d found", query, len(searchResults))
	for _, sr := range searchResults {
		t.Logf("  - %s (score: %f)", sr.PageContent, sr.Score)
	}

	// 9. Using Memory to save and load context (BufferMemory example)
	initialMemoryVars, err := bufferMem.LoadMemoryVariables(ctx, nil)
	require.NoError(t, err)
	// BufferMemory's history key might be specific, check its implementation or use a constant
	assert.Empty(t, initialMemoryVars[bufferMem.MemoryKey()])

	err = bufferMem.SaveContext(ctx, map[string]interface{}{"input": "My first message"}, map[string]interface{}{"output": "AI	's first response"})
	require.NoError(t, err)

	loadedMemoryVars, err := bufferMem.LoadMemoryVariables(ctx, map[string]interface{}{"input": "My second message"})
	require.NoError(t, err)
	assert.NotEmpty(t, loadedMemoryVars[bufferMem.MemoryKey()])
	t.Logf("Memory content after save/load: %v", loadedMemoryVars[bufferMem.MemoryKey()])

	t.Log("Cross-package interaction test completed successfully.")
}

// Mock implementations for testing when factories are not available

type mockLLMForTest struct{}

func (m *mockLLMForTest) Invoke(ctx context.Context, prompt string, options ...llmsiface.Option) (string, error) {
	return "Mock LLM response", nil
}

func (m *mockLLMForTest) GetModelName() string {
	return "mock-llm"
}

func (m *mockLLMForTest) GetProviderName() string {
	return "mock"
}

type mockEmbedderForTest struct{}

func (m *mockEmbedderForTest) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range result {
		result[i] = []float32{0.1, 0.2, 0.3}
	}
	return result, nil
}

func (m *mockEmbedderForTest) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *mockEmbedderForTest) GetDimension(ctx context.Context) (int, error) {
	return 3, nil
}

type mockVectorStoreFactory struct{}

func (m *mockVectorStoreFactory) Create(ctx context.Context, name string, config interface{}) (vectorstoresiface.VectorStore, error) {
	return &mockVectorStore{}, nil
}

type mockVectorStore struct{}

func (m *mockVectorStore) AddDocuments(ctx context.Context, documents []schema.Document, opts ...vectorstoresiface.Option) ([]string, error) {
	ids := make([]string, len(documents))
	for i := range ids {
		ids[i] = "mock-id-" + string(rune(i))
	}
	return ids, nil
}

func (m *mockVectorStore) DeleteDocuments(ctx context.Context, ids []string, opts ...vectorstoresiface.Option) error {
	return nil
}

func (m *mockVectorStore) SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...vectorstoresiface.Option) ([]schema.Document, []float32, error) {
	docs := []schema.Document{schema.NewDocument("mock content", nil)}
	scores := []float32{0.9}
	return docs, scores, nil
}

func (m *mockVectorStore) SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder vectorstoresiface.Embedder, opts ...vectorstoresiface.Option) ([]schema.Document, []float32, error) {
	docs := []schema.Document{schema.NewDocument("mock content", nil)}
	scores := []float32{0.9}
	return docs, scores, nil
}

func (m *mockVectorStore) AsRetriever(opts ...vectorstoresiface.Option) vectorstoresiface.Retriever {
	return nil
}

func (m *mockVectorStore) GetName() string {
	return "mock-vectorstore"
}
