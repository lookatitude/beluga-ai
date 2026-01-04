package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

func main() {
	fmt.Println("💬 Beluga AI - RAG with Memory Example")
	fmt.Println("======================================")

	ctx := context.Background()

	// Step 1: Create components
	embedder, err := createEmbedder(ctx)
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}

	vectorStore, err := vectorstores.NewVectorStore(ctx, "inmemory",
		vectorstores.WithEmbedder(embedder),
	)
	if err != nil {
		log.Fatalf("Failed to create vector store: %v", err)
	}

	llm, err := createLLM(ctx)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Step 2: Create memory for conversation history
	memConfig := memory.Config{
		Type:        memory.MemoryTypeBuffer,
		MemoryKey:   "history",
		InputKey:    "input",
		OutputKey:   "output",
		Enabled:     true,
		ReturnMessages: false,
	}

	memFactory := memory.NewFactory()
	mem, err := memFactory.CreateMemory(ctx, memConfig)
	if err != nil {
		log.Fatalf("Failed to create memory: %v", err)
	}

	fmt.Println("✅ Created RAG pipeline with conversation memory")

	// Step 3: Add documents to knowledge base
	documents := []schema.Document{
		schema.NewDocument(
			"Python is a high-level programming language known for its simplicity and readability.",
			map[string]string{"topic": "Python", "type": "language"},
		),
		schema.NewDocument(
			"Go (Golang) is a statically typed language developed by Google, known for its performance and concurrency features.",
			map[string]string{"topic": "Go", "type": "language"},
		),
		schema.NewDocument(
			"JavaScript is a dynamic programming language primarily used for web development.",
			map[string]string{"topic": "JavaScript", "type": "language"},
		),
	}

	_, err = vectorStore.AddDocuments(ctx, documents,
		vectorstores.WithEmbedder(embedder),
	)
	if err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}
	fmt.Printf("✅ Added %d documents to knowledge base\n", len(documents))

	// Step 4: Multi-turn conversation with RAG
	queries := []string{
		"What is Python?",
		"What are the main differences between Python and Go?",
		"Can you summarize our conversation about programming languages?",
	}

	fmt.Println("\n💬 Starting multi-turn RAG conversation...")
	for i, query := range queries {
		fmt.Printf("\n--- Turn %d ---\n", i+1)
		fmt.Printf("User: %s\n", query)

		// Step 5: Load conversation memory
		inputs := map[string]any{"input": query}
		memoryVars, err := mem.LoadMemoryVariables(ctx, inputs)
		if err != nil {
			log.Printf("Warning: Failed to load memory: %v", err)
		}

		// Step 6: Retrieve relevant documents from vector store
		relevantDocs, scores, err := vectorStore.SimilaritySearchByQuery(
			ctx, query, 2, embedder,
		)
		if err != nil {
			log.Fatalf("Failed to search: %v", err)
		}

		fmt.Printf("📚 Retrieved %d relevant documents (scores: %v)\n", len(relevantDocs), scores)

		// Step 7: Build context from documents and memory
		contextBuilder := strings.Builder{}
		contextBuilder.WriteString("Context from knowledge base:\n")
		for j, doc := range relevantDocs {
			contextBuilder.WriteString(fmt.Sprintf("  %d. %s\n", j+1, doc.GetContent()))
		}

		// Add memory context if available
		if memoryVars != nil {
			if history, ok := memoryVars["history"].(string); ok && history != "" {
				contextBuilder.WriteString("\nPrevious conversation:\n")
				contextBuilder.WriteString(history)
			}
		}

		context := contextBuilder.String()

		// Step 8: Generate response
		messages := []schema.Message{
			schema.NewSystemMessage("You are a helpful assistant. Use the provided context and conversation history to answer questions."),
			schema.NewHumanMessage(context + "\n\nQuestion: " + query),
		}

		response, err := llm.Generate(ctx, messages)
		if err != nil {
			log.Fatalf("Failed to generate response: %v", err)
		}

		responseText := response.GetContent()
		fmt.Printf("🤖 Assistant: %s\n", responseText)

		// Step 9: Save to memory
		outputs := map[string]any{"output": responseText}
		if err := mem.SaveContext(ctx, inputs, outputs); err != nil {
			log.Printf("Warning: Failed to save context: %v", err)
		}
	}

	fmt.Println("\n✨ RAG with memory example completed successfully!")
}

// createEmbedder creates an embedder instance.
func createEmbedder(ctx context.Context) (interface{}, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  OPENAI_API_KEY not set, using mock embedder")
		return &mockEmbedder{}, nil
	}

	config := embeddings.NewConfig(
		embeddings.WithProvider("openai"),
		embeddings.WithModel("text-embedding-ada-002"),
		embeddings.WithAPIKey(apiKey),
	)

	factory := embeddings.NewFactory()
	embedder, err := factory.NewEmbedder("openai", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	return embedder, nil
}

// createLLM creates an LLM instance.
func createLLM(ctx context.Context) (interface{}, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  OPENAI_API_KEY not set, using mock LLM")
		return &mockLLM{}, nil
	}

	config := llms.NewConfig(
		llms.WithProvider("openai"),
		llms.WithModelName("gpt-3.5-turbo"),
		llms.WithAPIKey(apiKey),
	)

	factory := llms.NewFactory()
	llm, err := factory.CreateChatModel("openai", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM: %w", err)
	}

	return llm, nil
}

// mockEmbedder is a simple mock implementation.
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

// mockLLM is a simple mock implementation.
type mockLLM struct{}

func (m *mockLLM) Generate(ctx context.Context, messages []interface{}) (interface{}, error) {
	return &mockMessage{
		content: "Based on the context provided, here is a helpful answer to your question.",
	}, nil
}

// mockMessage implements a simple message interface
type mockMessage struct {
	content string
}

func (m *mockMessage) GetContent() string {
	return m.content
}
