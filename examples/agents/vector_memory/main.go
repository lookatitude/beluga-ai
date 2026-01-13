package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	_ "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/mock"
	_ "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/openai"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	_ "github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/inmemory"
)

func main() {
	fmt.Println("🔍 Beluga AI - Agent with Vector Store Memory Example")
	fmt.Println("=====================================================")

	ctx := context.Background()

	// Step 1: Create an LLM instance
	llm, err := createLLM(ctx)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Step 2: Create an embedder for vector operations
	embedder, err := createEmbedder(ctx)
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}

	// Step 3: Create a vector store
	vectorStore, err := vectorstores.NewInMemoryStore(ctx,
		vectorstores.WithEmbedder(embedder),
	)
	if err != nil {
		log.Fatalf("Failed to create vector store: %v", err)
	}

	// Step 4: Create vector store memory
	// Vector store memory uses semantic search to retrieve relevant past conversations
	mem := memory.NewVectorStoreRetrieverMemory(
		embedder,
		vectorStore,
	)

	fmt.Println("\n🔍 Created vector store memory with semantic retrieval")

	// Step 6: Create an agent
	agent, err := agents.NewBaseAgent("vector-memory-agent", llm, nil)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Step 7: Initialize the agent with memory
	initConfig := map[string]interface{}{
		"max_retries": 3,
		"memory":      mem,
	}
	if err := agent.Initialize(initConfig); err != nil {
		log.Fatalf("Failed to initialize agent: %v", err)
	}

	// Step 8: Have a conversation and save to memory
	conversations := []struct {
		input  string
		output string
	}{
		{"I love programming in Go", "That's great! Go is an excellent language for building efficient systems."},
		{"What's my favorite programming language?", "You mentioned you love programming in Go."},
		{"Tell me about Python", "Python is a versatile programming language known for its simplicity."},
		{"What did I say about Go?", "You said you love programming in Go."},
	}

	fmt.Println("\n💬 Starting conversation with vector memory...")
	for i, conv := range conversations {
		fmt.Printf("\n--- Turn %d ---\n", i+1)
		fmt.Printf("User: %s\n", conv.input)

		// Save previous conversation to memory
		if i > 0 {
			prevInputs := map[string]any{"input": conversations[i-1].input}
			prevOutputs := map[string]any{"output": conversations[i-1].output}
			if err := mem.SaveContext(ctx, prevInputs, prevOutputs); err != nil {
				log.Printf("Warning: Failed to save context: %v", err)
			}
		}

		// Load memory variables (semantic search for relevant past conversations)
		inputs := map[string]any{"input": conv.input}
		memoryVars, err := mem.LoadMemoryVariables(ctx, inputs)
		if err != nil {
			log.Printf("Warning: Failed to load memory: %v", err)
		} else {
			fmt.Printf("Retrieved relevant context: %v\n", memoryVars)
		}

		// Execute agent
		result, err := agent.Invoke(ctx, inputs)
		if err != nil {
			log.Fatalf("Agent execution failed: %v", err)
		}

		fmt.Printf("Agent: %s\n", result)
	}

	// Step 9: Display memory information
	fmt.Println("\n📚 Vector Memory Information:")
	fmt.Printf("Memory variables: %v\n", mem.MemoryVariables())

	fmt.Println("\n✨ Example completed successfully!")
}

// createLLM creates an LLM instance.
func createLLM(ctx context.Context) (llmsiface.LLM, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  OPENAI_API_KEY not set, using mock LLM")
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

// createEmbedder creates an embedder instance.
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

// mockLLM is a simple mock implementation.
type mockLLM struct {
	modelName    string
	providerName string
}

func (m *mockLLM) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	return "Mock response based on context", nil
}

func (m *mockLLM) GetModelName() string {
	return m.modelName
}

func (m *mockLLM) GetProviderName() string {
	return m.providerName
}

// mockEmbedder is a simple mock implementation.
type mockEmbedder struct{}

func (m *mockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	// Return mock embeddings (all zeros for simplicity)
	embeddings := make([][]float32, len(texts))
	for i := range embeddings {
		embeddings[i] = make([]float32, 1536) // OpenAI embedding dimension
	}
	return embeddings, nil
}

func (m *mockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return make([]float32, 1536), nil
}
