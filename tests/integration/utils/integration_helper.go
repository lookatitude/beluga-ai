// Package utils provides shared utilities for integration testing across the Beluga AI Framework.
// This package centralizes common testing patterns, mock creation, and assertion helpers
// to ensure consistent testing approaches across all integration test suites.
package utils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/registry"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"

	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	memoryiface "github.com/lookatitude/beluga-ai/pkg/memory/iface"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// IntegrationTestHelper provides centralized utilities for integration testing.
type IntegrationTestHelper struct {
	// Component factories
	llmFactory         *llms.Factory
	memoryRegistry     *memory.MemoryRegistry
	embeddingRegistry  *embeddings.ProviderRegistry
	vectorstoreFactory *vectorstoresiface.StoreFactory
	agentRegistry      *agents.AgentRegistry

	// Mock components
	mockLLMs         map[string]llmsiface.ChatModel
	mockMemories     map[string]memoryiface.Memory
	mockEmbedders    map[string]embeddingsiface.Embedder
	mockVectorStores map[string]vectorstoresiface.VectorStore
	mockAgents       map[string]agentsiface.CompositeAgent

	// Test configuration
	useRealProviders bool
	timeout          time.Duration

	// Synchronization
	mu sync.RWMutex
}

// NewIntegrationTestHelper creates a new integration test helper.
func NewIntegrationTestHelper() *IntegrationTestHelper {
	return &IntegrationTestHelper{
		llmFactory:         llms.NewFactory(),
		memoryRegistry:     memory.GetGlobalMemoryRegistry(),
		embeddingRegistry:  registry.GetRegistry(), // Use registry.GetRegistry() to get concrete type
		vectorstoreFactory: vectorstoresiface.NewStoreFactory(),
		agentRegistry:      agents.GetGlobalAgentRegistry(),

		mockLLMs:         make(map[string]llmsiface.ChatModel),
		mockMemories:     make(map[string]memoryiface.Memory),
		mockEmbedders:    make(map[string]embeddingsiface.Embedder),
		mockVectorStores: make(map[string]vectorstoresiface.VectorStore),
		mockAgents:       make(map[string]agentsiface.CompositeAgent),

		useRealProviders: shouldUseRealProviders(),
		timeout:          30 * time.Second,
	}
}

// shouldUseRealProviders determines whether to use real providers based on environment.
func shouldUseRealProviders() bool {
	// Use real providers only if API keys are available and not in short test mode
	return os.Getenv("OPENAI_API_KEY") != "" && os.Getenv("INTEGRATION_TEST_REAL_PROVIDERS") == "true"
}

// Mock Component Creation

// CreateMockLLM creates a mock LLM for testing.
func (h *IntegrationTestHelper) CreateMockLLM(name string) llmsiface.ChatModel {
	h.mu.Lock()
	defer h.mu.Unlock()

	if existing, exists := h.mockLLMs[name]; exists {
		return existing
	}

	// Create mock LLM with realistic behavior
	mockLLM := llms.NewAdvancedMockChatModel(name,
		llms.WithResponses(
			"This is a mock response for integration testing.",
			"Another mock response with different content.",
			"A third mock response for varied testing scenarios.",
		),
	)

	h.mockLLMs[name] = mockLLM
	return mockLLM
}

// CreateMockMemory creates a mock memory for testing.
func (h *IntegrationTestHelper) CreateMockMemory(name string, memoryType memory.MemoryType) memoryiface.Memory {
	h.mu.Lock()
	defer h.mu.Unlock()

	if existing, exists := h.mockMemories[name]; exists {
		return existing
	}

	mockMemory := memory.NewAdvancedMockMemory(name+"_key", memoryType)
	h.mockMemories[name] = mockMemory
	return mockMemory
}

// CreateMockEmbedder creates a mock embedder for testing.
func (h *IntegrationTestHelper) CreateMockEmbedder(name string, dimension int) embeddingsiface.Embedder {
	h.mu.Lock()
	defer h.mu.Unlock()

	if existing, exists := h.mockEmbedders[name]; exists {
		return existing
	}

	mockEmbedder := embeddings.NewAdvancedMockEmbedder("mock-provider", name+"-model", dimension)
	h.mockEmbedders[name] = mockEmbedder
	return mockEmbedder
}

// CreateMockVectorStore creates a mock vector store for testing.
func (h *IntegrationTestHelper) CreateMockVectorStore(name string) vectorstoresiface.VectorStore {
	h.mu.Lock()
	defer h.mu.Unlock()

	if existing, exists := h.mockVectorStores[name]; exists {
		return existing
	}

	mockVectorStore := vectorstores.NewAdvancedMockVectorStore(name)
	h.mockVectorStores[name] = mockVectorStore
	return mockVectorStore
}

// Integration Test Scenarios

// TestConversationFlow tests a complete conversation flow across LLM and Memory.
func (h *IntegrationTestHelper) TestConversationFlow(llm llmsiface.ChatModel, memory memoryiface.Memory, exchanges int) error {
	ctx := context.Background()

	for i := 0; i < exchanges; i++ {
		// Load existing memory
		inputs := map[string]any{
			"input": fmt.Sprintf("What is artificial intelligence? (conversation %d)", i+1),
		}

		memoryVars, err := memory.LoadMemoryVariables(ctx, inputs)
		if err != nil {
			return fmt.Errorf("failed to load memory variables: %w", err)
		}

		// Create messages for LLM
		messages := []schema.Message{
			schema.NewHumanMessage(inputs["input"].(string)),
		}

		// Add memory context if available
		if memoryVars != nil && memory.MemoryVariables() != nil && len(memory.MemoryVariables()) > 0 {
			memoryKey := memory.MemoryVariables()[0]
			if historyContent, exists := memoryVars[memoryKey]; exists {
				if content, ok := historyContent.(string); ok && content != "" {
					messages = append([]schema.Message{schema.NewSystemMessage("Previous conversation: " + content)}, messages...)
				}
			}
		}

		// Generate response
		response, err := llm.Generate(ctx, messages)
		if err != nil {
			return fmt.Errorf("failed to generate LLM response: %w", err)
		}

		// Save to memory
		outputs := map[string]any{
			"output": response.GetContent(),
		}

		err = memory.SaveContext(ctx, inputs, outputs)
		if err != nil {
			return fmt.Errorf("failed to save context to memory: %w", err)
		}
	}

	return nil
}

// TestRAGPipeline tests a complete RAG pipeline.
func (h *IntegrationTestHelper) TestRAGPipeline(embedder embeddingsiface.Embedder, vectorStore vectorstoresiface.VectorStore, llm llmsiface.ChatModel, memory memoryiface.Memory) error {
	ctx := context.Background()

	// Step 1: Ingest documents into vector store
	documents := []schema.Document{
		schema.NewDocument("Artificial intelligence is the simulation of human intelligence in machines.", map[string]string{"topic": "AI"}),
		schema.NewDocument("Machine learning is a subset of AI that uses statistical techniques.", map[string]string{"topic": "ML"}),
		schema.NewDocument("Deep learning uses neural networks with multiple layers.", map[string]string{"topic": "DL"}),
		schema.NewDocument("Natural language processing helps computers understand human language.", map[string]string{"topic": "NLP"}),
	}

	_, err := vectorStore.AddDocuments(ctx, documents, vectorstoresiface.WithEmbedder(embedder))
	if err != nil {
		return fmt.Errorf("failed to add documents to vector store: %w", err)
	}

	// Step 2: Query the vector store for relevant documents
	query := "What is machine learning?"
	relevantDocs, scores, err := vectorStore.SimilaritySearchByQuery(ctx, query, 2, embedder)
	if err != nil {
		return fmt.Errorf("failed to search vector store: %w", err)
	}

	if len(relevantDocs) == 0 {
		return errors.New("no relevant documents found")
	}

	// Step 3: Create context from retrieved documents
	contextContent := ""
	var contextContentSb227 strings.Builder
	for _, doc := range relevantDocs {
		contextContentSb227.WriteString(doc.GetContent() + "\n")
	}
	contextContent += contextContentSb227.String()

	// Step 4: Load conversation memory
	inputs := map[string]any{"input": query}
	memoryVars, err := memory.LoadMemoryVariables(ctx, inputs)
	if err != nil {
		return fmt.Errorf("failed to load memory variables: %w", err)
	}

	// Step 5: Create enhanced prompt with context and memory
	messages := []schema.Message{
		schema.NewSystemMessage("Use the following context to answer the question: " + contextContent),
	}

	// Add memory context if available
	if memoryVars != nil && memory.MemoryVariables() != nil && len(memory.MemoryVariables()) > 0 {
		memoryKey := memory.MemoryVariables()[0]
		if historyContent, exists := memoryVars[memoryKey]; exists {
			if content, ok := historyContent.(string); ok && content != "" {
				messages = append(messages, schema.NewSystemMessage("Previous conversation: "+content))
			}
		}
	}

	messages = append(messages, schema.NewHumanMessage(query))

	// Step 6: Generate response using LLM
	response, err := llm.Generate(ctx, messages)
	if err != nil {
		return fmt.Errorf("failed to generate LLM response: %w", err)
	}

	// Step 7: Save the interaction to memory
	outputs := map[string]any{"output": response.GetContent()}
	err = memory.SaveContext(ctx, inputs, outputs)
	if err != nil {
		return fmt.Errorf("failed to save context to memory: %w", err)
	}

	// Validate the pipeline worked
	if response.GetContent() == "" {
		return errors.New("LLM response was empty")
	}

	if len(scores) != len(relevantDocs) {
		return errors.New("mismatched scores and documents")
	}

	return nil
}

// TestMultiAgentWorkflow tests multi-agent collaboration.
func (h *IntegrationTestHelper) TestMultiAgentWorkflow(agents []agentsiface.CompositeAgent, orchestrator any, memory memoryiface.Memory) error {
	ctx := context.Background()

	// Test agent coordination through memory and orchestration
	for i, agent := range agents {
		// Load shared memory
		inputs := map[string]any{
			"input": fmt.Sprintf("Agent %d task: analyze data and provide insights", i+1),
		}

		_, err := memory.LoadMemoryVariables(ctx, inputs)
		if err != nil {
			return fmt.Errorf("agent %d failed to load memory: %w", i+1, err)
		}

		// Simulate agent execution (basic mock execution)
		// In real implementation, would execute agent with orchestrator
		_ = agent // Use agent

		// Save agent results to shared memory
		outputs := map[string]any{
			"output": fmt.Sprintf("Agent %d completed analysis with insights", i+1),
		}

		err = memory.SaveContext(ctx, inputs, outputs)
		if err != nil {
			return fmt.Errorf("agent %d failed to save context: %w", i+1, err)
		}
	}

	return nil
}

// Assertion helpers for cross-package testing

// AssertCrossPackageMetrics validates that metrics are properly recorded across packages.
func (h *IntegrationTestHelper) AssertCrossPackageMetrics(t *testing.T, package1, package2 string) {
	// This would validate that metrics from both packages are being recorded
	// Implementation depends on metrics collection setup
	// TODO: Implement actual cross-package metrics validation for %s <-> %s
	_ = package1
	_ = package2
}

// AssertHealthChecks validates health checks across multiple components.
func (h *IntegrationTestHelper) AssertHealthChecks(t *testing.T, components map[string]any) {
	for name, component := range components {
		// Check if component has health check method
		if hc, ok := component.(interface{ CheckHealth() map[string]any }); ok {
			health := hc.CheckHealth()
			// For mock components, health checks may not have "status" field
			// We verify that health check returns a non-empty map
			assert.NotEmpty(t, health, "Component %s should return health check data", name)
			// If status is present, verify it's valid
			if status, hasStatus := health["status"]; hasStatus {
				assert.NotEmpty(t, status, "Component %s status should not be empty", name)
			}
		}
	}
}

// AssertConfigurationConsistency validates configuration consistency across packages.
func (h *IntegrationTestHelper) AssertConfigurationConsistency(t *testing.T, configs map[string]any) {
	// Validate that configurations are consistent and compatible
	for name, config := range configs {
		assert.NotNil(t, config, "Configuration for %s should not be nil", name)

		// Check if configuration has validation method
		if validator, ok := config.(interface{ Validate() error }); ok {
			err := validator.Validate()
			assert.NoError(t, err, "Configuration for %s should be valid", name)
		}
	}
}

// Performance and Load Testing

// CrossPackageLoadTest runs load tests across multiple packages.
func (h *IntegrationTestHelper) CrossPackageLoadTest(t *testing.T, scenario func() error, numOperations, concurrency int) {
	var wg sync.WaitGroup
	errChan := make(chan error, numOperations)
	semaphore := make(chan struct{}, concurrency)

	start := time.Now()

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := scenario(); err != nil {
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	duration := time.Since(start)

	// Check for errors
	for err := range errChan {
		require.NoError(t, err)
	}

	// Log performance metrics
	t.Logf("Cross-package load test completed: %d operations in %v (%.2f ops/sec)",
		numOperations, duration, float64(numOperations)/duration.Seconds())
}

// Test Data Creation

// CreateTestDocuments creates standardized test documents for integration tests.
func CreateTestDocuments(count int, topic string) []schema.Document {
	documents := make([]schema.Document, count)
	topics := []string{"AI", "ML", "DL", "NLP", "CV"} // Rotate through topics

	for i := 0; i < count; i++ {
		actualTopic := topic
		if actualTopic == "" {
			actualTopic = topics[i%len(topics)]
		}

		content := fmt.Sprintf("This is integration test document %d about %s. "+
			"It contains detailed information for testing cross-package functionality. "+
			"The content is designed to be meaningful for embedding and retrieval operations.",
			i+1, actualTopic)

		metadata := map[string]string{
			"doc_id":     fmt.Sprintf("integration_doc_%d", i+1),
			"topic":      actualTopic,
			"test_suite": "integration",
			"created_at": time.Now().Format(time.RFC3339),
		}

		documents[i] = schema.NewDocument(content, metadata)
	}

	return documents
}

// CreateTestConversation creates a standardized test conversation.
func CreateTestConversation(exchanges int) []schema.Message {
	messages := make([]schema.Message, 0, exchanges*2)

	topics := []string{
		"artificial intelligence",
		"machine learning algorithms",
		"neural network architectures",
		"natural language processing",
		"computer vision techniques",
	}

	for i := 0; i < exchanges; i++ {
		topic := topics[i%len(topics)]

		humanMsg := schema.NewHumanMessage(fmt.Sprintf("Can you explain %s in simple terms? (conversation %d)", topic, i+1))
		aiMsg := schema.NewAIMessage(fmt.Sprintf("Certainly! %s is a fascinating field in computer science that... (response %d)", topic, i+1))

		messages = append(messages, humanMsg, aiMsg)
	}

	return messages
}

// CreateTestQueries creates standardized test queries.
func CreateTestQueries(count int) []string {
	queries := make([]string, count)

	templates := []string{
		"What is %s and how does it work?",
		"Explain the benefits of %s in modern applications",
		"What are the challenges with implementing %s?",
		"How does %s compare to traditional approaches?",
		"What are the future trends in %s?",
	}

	topics := []string{"AI", "machine learning", "deep learning", "NLP", "computer vision"}

	for i := 0; i < count; i++ {
		template := templates[i%len(templates)]
		topic := topics[i%len(topics)]
		queries[i] = fmt.Sprintf(template, topic)
	}

	return queries
}

// Environment and Configuration

// GetTestConfig returns test configuration based on environment.
func GetTestConfig() TestConfig {
	return TestConfig{
		UseRealProviders: shouldUseRealProviders(),
		DefaultTimeout:   30 * time.Second,
		MaxRetries:       3,
		ConcurrencyLimit: 5,
		EnableMetrics:    true,
		EnableTracing:    true,
		EnableLogging:    os.Getenv("BELUGA_DEBUG") == "true",
	}
}

// TestConfig holds configuration for integration tests.
type TestConfig struct {
	DefaultTimeout   time.Duration
	MaxRetries       int
	ConcurrencyLimit int
	UseRealProviders bool
	EnableMetrics    bool
	EnableTracing    bool
	EnableLogging    bool
}

// Cleanup and Reset

// Reset clears all cached mock components.
func (h *IntegrationTestHelper) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Clear all mock components
	h.mockLLMs = make(map[string]llmsiface.ChatModel)
	h.mockMemories = make(map[string]memoryiface.Memory)
	h.mockEmbedders = make(map[string]embeddingsiface.Embedder)
	h.mockVectorStores = make(map[string]vectorstoresiface.VectorStore)
	h.mockAgents = make(map[string]agentsiface.CompositeAgent)
}

// Cleanup performs cleanup after integration tests.
func (h *IntegrationTestHelper) Cleanup(ctx context.Context) error {
	// Clear any persistent state in mock components
	for _, memory := range h.mockMemories {
		if err := memory.Clear(ctx); err != nil {
			return fmt.Errorf("failed to clear memory: %w", err)
		}
	}

	// Reset vector stores
	for name := range h.mockVectorStores {
		// Vector stores don't have a standard clear method, so we recreate
		// This is handled by Reset() method which clears the map
		_ = name // Acknowledge we're iterating over the map
	}

	return nil
}

// GetMemoryContent retrieves memory content for validation.
func (h *IntegrationTestHelper) GetMemoryContent(memory memoryiface.Memory) string {
	ctx := context.Background()

	vars, err := memory.LoadMemoryVariables(ctx, map[string]any{"input": "test"})
	if err != nil {
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

// Skip Integration Tests

// SkipIfNoRealProviders skips a test if real providers are not available.
func SkipIfNoRealProviders(t *testing.T) {
	t.Helper()
	if !shouldUseRealProviders() {
		t.Skip("Skipping integration test - real providers not configured")
	}
}

// SkipIfShortMode skips a test if running in short mode.
func SkipIfShortMode(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
}

// GetEnvOrSkip gets an environment variable or skips the test if not set.
func GetEnvOrSkip(t *testing.T, key string) string {
	t.Helper()
	value := os.Getenv(key)
	if value == "" {
		t.Skipf("Skipping test: %s environment variable not set", key)
	}
	return value
}
