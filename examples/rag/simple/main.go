package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

func main() {
	fmt.Println("📚 Beluga AI - Simple RAG Pipeline Example")
	fmt.Println("===========================================")

	ctx := context.Background()

	// Step 1: Create an embedder
	embedder, err := createEmbedder(ctx)
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}
	fmt.Println("✅ Created embedder")

	// Step 2: Create a vector store
	vectorStore, err := vectorstores.NewVectorStore(ctx, "inmemory",
		vectorstores.WithEmbedder(embedder),
	)
	if err != nil {
		log.Fatalf("Failed to create vector store: %v", err)
	}
	fmt.Println("✅ Created vector store")

	// Step 3: Create an LLM
	llm, err := createLLM(ctx)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}
	fmt.Println("✅ Created LLM")

	// Step 4: Prepare documents for the knowledge base
	documents := []schema.Document{
		schema.NewDocument(
			"Artificial Intelligence (AI) is the simulation of human intelligence in machines.",
			map[string]string{"topic": "AI", "level": "intro"},
		),
		schema.NewDocument(
			"Machine Learning is a subset of AI that enables machines to learn from data.",
			map[string]string{"topic": "ML", "level": "intro"},
		),
		schema.NewDocument(
			"Deep Learning uses neural networks with multiple layers to understand complex patterns.",
			map[string]string{"topic": "DL", "level": "intermediate"},
		),
		schema.NewDocument(
			"Natural Language Processing helps computers understand human language.",
			map[string]string{"topic": "NLP", "level": "intermediate"},
		),
	}

	// Step 5: Add documents to the vector store
	fmt.Println("\n📝 Adding documents to vector store...")
	ids, err := vectorStore.AddDocuments(ctx, documents,
		vectorstores.WithEmbedder(embedder),
	)
	if err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}
	fmt.Printf("✅ Added %d documents (IDs: %v)\n", len(documents), ids)

	// Step 6: Query the knowledge base
	query := "What is machine learning?"
	fmt.Printf("\n🔍 Query: %s\n", query)

	// Step 7: Retrieve relevant documents
	relevantDocs, scores, err := vectorStore.SimilaritySearchByQuery(
		ctx, query, 3, embedder,
	)
	if err != nil {
		log.Fatalf("Failed to search: %v", err)
	}

	fmt.Printf("✅ Retrieved %d relevant documents:\n", len(relevantDocs))
	for i, doc := range relevantDocs {
		fmt.Printf("  [Score: %.3f] %s\n", scores[i], doc.GetContent())
	}

	// Step 8: Build context from retrieved documents
	contextBuilder := strings.Builder{}
	contextBuilder.WriteString("Use the following context to answer the question:\n\n")
	for i, doc := range relevantDocs {
		contextBuilder.WriteString(fmt.Sprintf("Document %d: %s\n", i+1, doc.GetContent()))
	}

	context := contextBuilder.String()

	// Step 9: Generate response using LLM with context
	fmt.Println("\n🤖 Generating response with context...")
	messages := []schema.Message{
		schema.NewSystemMessage("You are a helpful assistant that answers questions based on the provided context."),
		schema.NewHumanMessage(context + "\n\nQuestion: " + query),
	}

	response, err := llm.Generate(ctx, messages)
	if err != nil {
		log.Fatalf("Failed to generate response: %v", err)
	}

	// Step 10: Display the final answer
	fmt.Printf("\n✅ Final Answer:\n%s\n", response.GetContent())

	fmt.Println("\n✨ RAG pipeline example completed successfully!")
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
		content: "Machine Learning is a subset of Artificial Intelligence that enables machines to automatically learn and improve from experience without being explicitly programmed. It uses algorithms to analyze data, identify patterns, and make decisions or predictions.",
	}, nil
}

// mockMessage implements a simple message interface
type mockMessage struct {
	content string
}

func (m *mockMessage) GetContent() string {
	return m.content
}
