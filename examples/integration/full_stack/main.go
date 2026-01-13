package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	_ "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/mock"
	_ "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/openai"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	_ "github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/inmemory"
)

func main() {
	fmt.Println("🏗️  Beluga AI - Full Stack Integration Example")
	fmt.Println("===============================================")

	ctx := context.Background()

	// Step 1: Initialize all components
	fmt.Println("\n📦 Initializing components...")

	// 1.1: Create embedder
	embedder, err := createEmbedder(ctx)
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}
	fmt.Println("  ✅ Embedder created")

	// 1.2: Create vector store
	vectorStore, err := vectorstores.NewInMemoryStore(ctx,
		vectorstores.WithEmbedder(embedder),
	)
	if err != nil {
		log.Fatalf("Failed to create vector store: %v", err)
	}
	fmt.Println("  ✅ Vector store created")

	// 1.3: Create LLM
	llm, err := createLLM(ctx)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}
	fmt.Println("  ✅ LLM created")

	// 1.4: Create memory
	memConfig := memory.Config{
		Type:           memory.MemoryTypeBuffer,
		MemoryKey:      "history",
		InputKey:       "input",
		OutputKey:      "output",
		Enabled:        true,
		ReturnMessages: false,
	}
	memFactory := memory.NewFactory()
	mem, err := memFactory.CreateMemory(ctx, memConfig)
	if err != nil {
		log.Fatalf("Failed to create memory: %v", err)
	}
	fmt.Println("  ✅ Memory created")

	// 1.5: Create agent
	agent, err := agents.NewBaseAgent("full-stack-agent", llm, nil)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	agent.Initialize(map[string]interface{}{"memory": mem})
	fmt.Println("  ✅ Agent created")

	// Step 2: Set up knowledge base
	fmt.Println("\n📚 Setting up knowledge base...")
	documents := []schema.Document{
		schema.NewDocument(
			"Beluga AI is a comprehensive framework for building AI applications in Go.",
			map[string]string{"topic": "framework", "type": "overview"},
		),
		schema.NewDocument(
			"The framework supports agents, RAG pipelines, orchestration, and multi-agent systems.",
			map[string]string{"topic": "features", "type": "capabilities"},
		),
	}

	_, err = vectorStore.AddDocuments(ctx, documents)
	if err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}
	fmt.Printf("  ✅ Added %d documents\n", len(documents))

	// Step 3: Ready for orchestration
	fmt.Println("\n🔗 Ready for orchestration...")

	// Step 4: Execute full-stack workflow
	fmt.Println("\n🚀 Executing full-stack workflow...")
	query := "What is Beluga AI and what are its main features?"

	// Step 4a: RAG retrieval
	fmt.Println("\n  Step 1: RAG Retrieval")
	relevantDocs, scores, err := vectorStore.SimilaritySearchByQuery(
		ctx, query, 2, embedder,
	)
	if err != nil {
		log.Fatalf("RAG retrieval failed: %v", err)
	}
	fmt.Printf("    Retrieved %d documents (scores: %v)\n", len(relevantDocs), scores)

	// Step 4b: Build context
	contextBuilder := strings.Builder{}
	contextBuilder.WriteString("Context:\n")
	for i, doc := range relevantDocs {
		contextBuilder.WriteString(fmt.Sprintf("  %d. %s\n", i+1, doc.GetContent()))
	}
	context := contextBuilder.String()

	// Step 4c: Load memory
	memoryVars, err := mem.LoadMemoryVariables(ctx, map[string]any{"input": query})
	if err == nil && memoryVars != nil {
		if history, ok := memoryVars["history"].(string); ok {
			context += "\nPrevious conversation:\n" + history
		}
	}

	// Step 4d: Agent processing
	fmt.Println("\n  Step 2: Agent Processing")
	agentInput := map[string]interface{}{
		"input":   query,
		"context": context,
	}
	agentResult, err := agent.Invoke(ctx, agentInput)
	if err != nil {
		log.Fatalf("Agent processing failed: %v", err)
	}
	fmt.Printf("    Agent result: %v\n", agentResult)

	// Step 4e: Save to memory
	err = mem.SaveContext(ctx, map[string]any{"input": query}, map[string]any{"output": agentResult})
	if err != nil {
		log.Printf("Warning: Failed to save context: %v", err)
	}

	// Step 5: Display final result
	fmt.Printf("\n✅ Full-Stack Result:\n")
	fmt.Printf("  Query: %s\n", query)
	fmt.Printf("  Context Retrieved: %d documents\n", len(relevantDocs))
	fmt.Printf("  Agent Response: %v\n", agentResult)

	fmt.Println("\n✨ Full-stack integration example completed successfully!")
}

// createEmbedder creates an embedder instance
func createEmbedder(ctx context.Context) (embeddingsiface.Embedder, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  OPENAI_API_KEY not set, using mock embedder")
		config := &embeddings.Config{
			Mock: &embeddings.MockConfig{
				Dimension: 1536,
				Enabled:   true,
			},
		}
		config.SetDefaults()
		return embeddings.NewEmbedder(ctx, "mock", *config)
	}

	config := &embeddings.Config{
		OpenAI: &embeddings.OpenAIConfig{
			APIKey:  apiKey,
			Model:   "text-embedding-ada-002",
			Enabled: true,
		},
	}
	config.SetDefaults()

	embedder, err := embeddings.NewEmbedder(ctx, "openai", *config)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	return embedder, nil
}

// createLLM creates an LLM instance
func createLLM(ctx context.Context) (llmsiface.LLM, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return &mockLLM{
			modelName:    "mock-model",
			providerName: "mock-provider",
		}, nil
	}

	config := llms.NewConfig(
		llms.WithProvider("openai"),
		llms.WithModelName("gpt-3.5-turbo"),
		llms.WithAPIKey(apiKey),
	)

	factory := llms.NewFactory()
	llm, err := factory.CreateProvider("openai", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM: %w", err)
	}

	return llm, nil
}

// mockEmbedder is a simple mock implementation
type mockEmbedder struct{}

func (m *mockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i := range embeddings {
		embeddings[i] = make([]float32, 1536)
	}
	return embeddings, nil
}

func (m *mockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return make([]float32, 1536), nil
}

// mockLLM is a simple mock implementation
type mockLLM struct {
	modelName    string
	providerName string
}

func (m *mockLLM) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	return "Beluga AI is a comprehensive framework for building AI applications in Go. It supports agents, RAG pipelines, orchestration, and multi-agent systems.", nil
}

func (m *mockLLM) GetModelName() string {
	return m.modelName
}

func (m *mockLLM) GetProviderName() string {
	return m.providerName
}
