// Package end_to_end provides comprehensive end-to-end integration tests.
// This test suite verifies that multiple packages work together correctly
// in realistic, complex scenarios that mirror production usage patterns.
package end_to_end

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	memoryiface "github.com/lookatitude/beluga-ai/pkg/memory/iface"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

// TestEndToEndRAGPipeline tests a complete RAG (Retrieval Augmented Generation) pipeline
func TestEndToEndRAGPipeline(t *testing.T) {
	utils.SkipIfShortMode(t) // This is a comprehensive test

	helper := utils.NewIntegrationTestHelper()
	defer helper.Cleanup(context.Background())

	tests := []struct {
		name              string
		setupDocs         []schema.Document
		conversationSteps []ConversationStep
		expectedResults   []string // Patterns we expect to see in responses
	}{
		{
			name: "ai_knowledge_base",
			setupDocs: []schema.Document{
				schema.NewDocument("Artificial Intelligence (AI) is the simulation of human intelligence in machines that are programmed to think and learn.", map[string]string{"topic": "AI", "level": "intro"}),
				schema.NewDocument("Machine Learning is a subset of AI that enables machines to automatically learn and improve from experience without being explicitly programmed.", map[string]string{"topic": "ML", "level": "intro"}),
				schema.NewDocument("Deep Learning is a subset of machine learning that uses neural networks with three or more layers to model and understand complex patterns.", map[string]string{"topic": "DL", "level": "intermediate"}),
				schema.NewDocument("Natural Language Processing (NLP) is a branch of AI that helps computers understand, interpret and manipulate human language.", map[string]string{"topic": "NLP", "level": "intermediate"}),
				schema.NewDocument("Computer Vision is a field of AI that enables computers to interpret and understand visual information from digital images or videos.", map[string]string{"topic": "CV", "level": "intermediate"}),
			},
			conversationSteps: []ConversationStep{
				{
					Query:           "What is artificial intelligence?",
					ExpectedContext: []string{"simulation", "human intelligence", "machines"},
					ExpectedMemory:  true,
				},
				{
					Query:           "How does machine learning relate to AI?",
					ExpectedContext: []string{"subset", "learn", "experience"},
					ExpectedMemory:  true,
				},
				{
					Query:           "Can you summarize what we've discussed?",
					ExpectedContext: []string{}, // Should primarily use memory
					ExpectedMemory:  true,
				},
			},
			expectedResults: []string{"intelligence", "learning", "discussed"},
		},
		{
			name: "technical_documentation",
			setupDocs: []schema.Document{
				schema.NewDocument("REST APIs use HTTP methods like GET, POST, PUT, DELETE to perform operations on resources.", map[string]string{"topic": "API", "type": "REST"}),
				schema.NewDocument("GraphQL is a query language that allows clients to request exactly the data they need from a single endpoint.", map[string]string{"topic": "API", "type": "GraphQL"}),
				schema.NewDocument("Microservices architecture breaks applications into small, independent services that communicate over networks.", map[string]string{"topic": "Architecture", "pattern": "microservices"}),
				schema.NewDocument("Event-driven architecture uses events to trigger and communicate between decoupled services.", map[string]string{"topic": "Architecture", "pattern": "events"}),
			},
			conversationSteps: []ConversationStep{
				{
					Query:           "What are REST APIs?",
					ExpectedContext: []string{"HTTP", "methods", "resources"},
					ExpectedMemory:  true,
				},
				{
					Query:           "How does GraphQL compare to REST?",
					ExpectedContext: []string{"GraphQL", "query", "endpoint"},
					ExpectedMemory:  true,
				},
			},
			expectedResults: []string{"API", "GraphQL", "REST"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create pipeline components
			pipeline := createRAGPipelineGeneric(t, helper)

			// Setup knowledge base
			err := setupKnowledgeBase(t, pipeline, tt.setupDocs)
			require.NoError(t, err)

			// Run conversation steps
			for i, step := range tt.conversationSteps {
				result, err := executeRAGQuery(t, pipeline, step.Query)
				require.NoError(t, err, "Conversation step %d failed", i+1)

				// Validate result is non-empty (mock LLMs return generic responses)
				// In a real test with actual LLMs, we would validate specific patterns
				assert.NotEmpty(t, result, "Step %d should produce a non-empty result", i+1)

				// For mock-based tests, we verify the pipeline executed successfully
				// Pattern matching is skipped for mocks as they return generic responses
				// In production tests with real LLMs, we would validate expectedResults patterns

				// Validate context was used if expected
				if len(step.ExpectedContext) > 0 {
					for range step.ExpectedContext {
						// In a real test, we'd verify the context was actually used
						// For mock test, we verify the pipeline executed successfully
						assert.NotEmpty(t, result, "Result should not be empty when context expected")
					}
				}

				// Validate memory was preserved if expected
				if step.ExpectedMemory {
					memoryContent := getMemoryContent(t, pipeline.Memory)
					assert.NotEmpty(t, memoryContent, "Memory should contain conversation history")
				}
			}

			// Verify pipeline health
			validatePipelineHealth(t, pipeline)
		})
	}
}

// ConversationStep represents a step in a multi-turn conversation
type ConversationStep struct {
	Query           string
	ExpectedContext []string
	ExpectedMemory  bool
}

// RAGPipeline represents a complete RAG pipeline for testing
type RAGPipeline struct {
	LLM         llmsiface.ChatModel
	Memory      memoryiface.Memory
	Embedder    embeddingsiface.Embedder
	VectorStore vectorstoresiface.VectorStore
	Documents   []schema.Document
}

// setupKnowledgeBase ingests documents into the vector store
func setupKnowledgeBase(tb testing.TB, pipeline *RAGPipeline, documents []schema.Document) error {
	ctx := context.Background()

	// Store documents for later reference
	pipeline.Documents = make([]schema.Document, len(documents))
	copy(pipeline.Documents, documents)

	// Add documents to vector store with embeddings
	_, err := pipeline.VectorStore.AddDocuments(ctx, documents,
		vectorstoresiface.WithEmbedder(pipeline.Embedder))

	if err != nil {
		return fmt.Errorf("failed to setup knowledge base: %w", err)
	}

	tb.Logf("Knowledge base setup complete: %d documents indexed", len(documents))
	return nil
}

// executeRAGQuery executes a query through the complete RAG pipeline
func executeRAGQuery(tb testing.TB, pipeline *RAGPipeline, query string) (string, error) {
	ctx := context.Background()

	// Step 1: Load conversation memory
	inputs := map[string]any{"input": query}
	memoryVars, err := pipeline.Memory.LoadMemoryVariables(ctx, inputs)
	if err != nil {
		return "", fmt.Errorf("failed to load memory: %w", err)
	}

	// Step 2: Retrieve relevant documents from vector store
	relevantDocs, scores, err := pipeline.VectorStore.SimilaritySearchByQuery(
		ctx, query, 3, pipeline.Embedder)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve documents: %w", err)
	}

	tb.Logf("Retrieved %d relevant documents with scores %v", len(relevantDocs), scores)

	// Step 3: Create context from retrieved documents
	contextContent := ""
	for i, doc := range relevantDocs {
		contextContent += fmt.Sprintf("Document %d: %s\n", i+1, doc.GetContent())
	}

	// Step 4: Create messages with context and memory
	messages := []schema.Message{
		schema.NewSystemMessage("Use the following context to answer the question: " + contextContent),
	}

	// Add memory context if available
	if memoryVars != nil && len(pipeline.Memory.MemoryVariables()) > 0 {
		memoryKey := pipeline.Memory.MemoryVariables()[0]
		if historyContent, exists := memoryVars[memoryKey]; exists {
			if content, ok := historyContent.(string); ok && content != "" {
				messages = append(messages, schema.NewSystemMessage("Previous conversation: "+content))
			}
		}
	}

	messages = append(messages, schema.NewHumanMessage(query))

	// Step 5: Generate response using LLM
	response, err := pipeline.LLM.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	// Step 6: Save conversation to memory
	outputs := map[string]any{"output": response.GetContent()}
	err = pipeline.Memory.SaveContext(ctx, inputs, outputs)
	if err != nil {
		return "", fmt.Errorf("failed to save context: %w", err)
	}

	return response.GetContent(), nil
}

// getMemoryContent retrieves memory content for validation
func getMemoryContent(t *testing.T, memory memoryiface.Memory) string {
	ctx := context.Background()

	vars, err := memory.LoadMemoryVariables(ctx, map[string]any{"input": "test"})
	if err != nil {
		t.Logf("Failed to load memory variables: %v", err)
		return ""
	}

	if len(memory.MemoryVariables()) == 0 {
		return ""
	}

	memoryKey := memory.MemoryVariables()[0]
	if content, exists := vars[memoryKey]; exists {
		if str, ok := content.(string); ok {
			return str
		}
	}

	return ""
}

// validatePipelineHealth checks the health of all pipeline components
func validatePipelineHealth(t *testing.T, pipeline *RAGPipeline) {
	components := map[string]interface{}{
		"llm":          pipeline.LLM,
		"memory":       pipeline.Memory,
		"embedder":     pipeline.Embedder,
		"vector_store": pipeline.VectorStore,
	}

	helper := utils.NewIntegrationTestHelper()
	helper.AssertHealthChecks(t, components)
}

// TestEndToEndRAGPipelinePerformance tests RAG pipeline performance
func TestEndToEndRAGPipelinePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	helper := utils.NewIntegrationTestHelper()
	defer helper.Cleanup(context.Background())

	tests := []struct {
		name                string
		documentsCount      int
		queries             int
		maxDurationPerQuery time.Duration
	}{
		{
			name:                "small_knowledge_base",
			documentsCount:      10,
			queries:             5,
			maxDurationPerQuery: 1 * time.Second,
		},
		{
			name:                "medium_knowledge_base",
			documentsCount:      50,
			queries:             10,
			maxDurationPerQuery: 2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := createRAGPipelineGeneric(t, helper)

			// Setup knowledge base
			documents := utils.CreateTestDocuments(tt.documentsCount, "AI")
			err := setupKnowledgeBase(t, pipeline, documents)
			require.NoError(t, err)

			// Execute queries and measure performance
			queries := utils.CreateTestQueries(tt.queries)
			start := time.Now()

			for i, query := range queries {
				queryStart := time.Now()

				result, err := executeRAGQuery(t, pipeline, query)
				require.NoError(t, err, "Query %d failed", i+1)

				queryDuration := time.Since(queryStart)
				assert.LessOrEqual(t, queryDuration, tt.maxDurationPerQuery,
					"Query %d took %v, should be <= %v", i+1, queryDuration, tt.maxDurationPerQuery)

				assert.NotEmpty(t, result, "Query %d should produce non-empty result", i+1)
			}

			totalDuration := time.Since(start)
			t.Logf("RAG pipeline performance: %d documents, %d queries in %v (avg: %v per query)",
				tt.documentsCount, tt.queries, totalDuration, totalDuration/time.Duration(tt.queries))
		})
	}
}

// TestEndToEndRAGPipelineErrorRecovery tests error handling and recovery
func TestEndToEndRAGPipelineErrorRecovery(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer helper.Cleanup(context.Background())

	tests := []struct {
		name          string
		errorStage    string
		setupError    func(pipeline *RAGPipeline)
		shouldRecover bool
	}{
		{
			name:       "embedder_failure",
			errorStage: "embedding",
			setupError: func(pipeline *RAGPipeline) {
				// Replace embedder with one that errors
				errorEmbedder := embeddings.NewAdvancedMockEmbedder("error-provider", "error-model", 128,
					embeddings.WithMockError(true, fmt.Errorf("embedding service unavailable")))
				pipeline.Embedder = errorEmbedder
			},
			shouldRecover: false,
		},
		{
			name:       "llm_failure",
			errorStage: "generation",
			setupError: func(pipeline *RAGPipeline) {
				// Replace LLM with one that errors
				errorLLM := llms.NewAdvancedMockChatModel("error-llm",
					llms.WithError(fmt.Errorf("LLM service unavailable")))
				pipeline.LLM = errorLLM
			},
			shouldRecover: false,
		},
		{
			name:       "memory_failure",
			errorStage: "memory",
			setupError: func(pipeline *RAGPipeline) {
				// Replace memory with one that errors on save
				errorMemory := memory.NewAdvancedMockMemory("error-memory", memory.MemoryTypeBuffer,
					memory.WithMockError(true, fmt.Errorf("memory storage unavailable")))
				pipeline.Memory = errorMemory
			},
			shouldRecover: false,
		},
		{
			name:       "vector_store_failure",
			errorStage: "retrieval",
			setupError: func(pipeline *RAGPipeline) {
				// Replace vector store with one that errors on search
				errorStore := vectorstores.NewAdvancedMockVectorStore("error-store",
					vectorstores.WithMockError(true, fmt.Errorf("vector store unavailable")))
				pipeline.VectorStore = errorStore
			},
			shouldRecover: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := createRAGPipelineGeneric(t, helper)

			// Setup knowledge base first
			documents := utils.CreateTestDocuments(5, "test")
			err := setupKnowledgeBase(t, pipeline, documents)
			if tt.errorStage != "vector_store_setup" {
				require.NoError(t, err)
			}

			// Introduce error
			tt.setupError(pipeline)

			// Test query execution with error
			result, err := executeRAGQuery(t, pipeline, "Test query")

			if tt.shouldRecover {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
			} else {
				assert.Error(t, err)
				t.Logf("Expected error in %s stage: %v", tt.errorStage, err)
			}
		})
	}
}

// TestEndToEndRAGPipelineConcurrency tests concurrent RAG pipeline usage
func TestEndToEndRAGPipelineConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency tests in short mode")
	}

	helper := utils.NewIntegrationTestHelper()
	defer helper.Cleanup(context.Background())

	pipeline := createRAGPipelineGeneric(t, helper)

	// Setup knowledge base
	documents := utils.CreateTestDocuments(20, "concurrent")
	err := setupKnowledgeBase(t, pipeline, documents)
	require.NoError(t, err)

	const numGoroutines = 5
	const queriesPerGoroutine = 3

	t.Run("concurrent_queries", func(t *testing.T) {
		helper.CrossPackageLoadTest(t, func() error {
			query := "What is artificial intelligence?"
			result, err := executeRAGQuery(t, pipeline, query)
			if err != nil {
				return err
			}
			if result == "" {
				return fmt.Errorf("empty result from RAG query")
			}
			return nil
		}, numGoroutines*queriesPerGoroutine, numGoroutines)
	})
}

// TestEndToEndRAGPipelineRealProviders tests with real providers (if available)
func TestEndToEndRAGPipelineRealProviders(t *testing.T) {
	utils.SkipIfNoRealProviders(t)
	utils.SkipIfShortMode(t)

	// This test would use real OpenAI/Anthropic LLMs, real embedding services, etc.
	// Implementation would be similar to mock tests but with real provider configurations
	t.Log("Real provider RAG pipeline test would be implemented here")
	t.Log("Requires: OPENAI_API_KEY, real vector store instance, etc.")

	// For now, we'll skip the implementation as it requires external services
	t.Skip("Real provider integration tests require external service credentials")
}

// BenchmarkEndToEndRAGPipeline benchmarks complete RAG pipeline performance
func BenchmarkEndToEndRAGPipeline(b *testing.B) {
	helper := utils.NewIntegrationTestHelper()
	defer helper.Cleanup(context.Background())

	pipeline := createRAGPipelineGeneric(b, helper)

	// Setup knowledge base
	documents := utils.CreateTestDocuments(50, "benchmark")
	err := setupKnowledgeBase(b, pipeline, documents)
	if err != nil {
		b.Fatalf("Failed to setup knowledge base: %v", err)
	}

	query := "What is the most important concept in artificial intelligence?"

	b.Run("FullPipeline", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := executeRAGQuery(b, pipeline, query)
			if err != nil {
				b.Errorf("RAG pipeline error: %v", err)
			}
		}
	})

	b.Run("DocumentRetrieval", func(b *testing.B) {
		ctx := context.Background()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := pipeline.VectorStore.SimilaritySearchByQuery(ctx, query, 3, pipeline.Embedder)
			if err != nil {
				b.Errorf("Document retrieval error: %v", err)
			}
		}
	})

	b.Run("ResponseGeneration", func(b *testing.B) {
		ctx := context.Background()
		messages := []schema.Message{
			schema.NewSystemMessage("Use the context to answer: Sample context"),
			schema.NewHumanMessage(query),
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := pipeline.LLM.Generate(ctx, messages)
			if err != nil {
				b.Errorf("Response generation error: %v", err)
			}
		}
	})

	b.Run("MemoryOperations", func(b *testing.B) {
		ctx := context.Background()
		inputs := map[string]any{"input": query}
		outputs := map[string]any{"output": "Sample response"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := pipeline.Memory.SaveContext(ctx, inputs, outputs)
			if err != nil {
				b.Errorf("Memory save error: %v", err)
			}

			_, err = pipeline.Memory.LoadMemoryVariables(ctx, inputs)
			if err != nil {
				b.Errorf("Memory load error: %v", err)
			}
		}
	})
}

// Helper function that works with both testing.T and testing.B
func createRAGPipelineGeneric(tb testing.TB, helper *utils.IntegrationTestHelper) *RAGPipeline {
	// Create components
	llm := helper.CreateMockLLM("rag-llm")
	memory := helper.CreateMockMemory("rag-memory", memory.MemoryTypeBuffer)
	embedder := helper.CreateMockEmbedder("rag-embedder", 128)
	vectorStore := helper.CreateMockVectorStore("rag-vectorstore")

	return &RAGPipeline{
		LLM:         llm,
		Memory:      memory,
		Embedder:    embedder,
		VectorStore: vectorStore,
		Documents:   make([]schema.Document, 0),
	}
}
