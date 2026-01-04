package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/retrievers"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

func main() {
	fmt.Println("🚀 Beluga AI - Advanced RAG Pipeline Example")
	fmt.Println("==============================================")

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

	// Step 2: Create a retriever with advanced configuration
	retriever, err := retrievers.NewVectorStoreRetriever(
		vectorStore,
		retrievers.WithDefaultK(5),           // Retrieve top 5 documents
		retrievers.WithScoreThreshold(0.7),   // Minimum similarity score
	)
	if err != nil {
		log.Fatalf("Failed to create retriever: %v", err)
	}
	fmt.Println("✅ Created advanced retriever with filtering")

	// Step 3: Add diverse documents to knowledge base
	documents := []schema.Document{
		schema.NewDocument(
			"REST APIs use HTTP methods like GET, POST, PUT, DELETE to perform operations on resources.",
			map[string]string{"topic": "API", "type": "REST", "level": "beginner"},
		),
		schema.NewDocument(
			"GraphQL is a query language that allows clients to request exactly the data they need from a single endpoint.",
			map[string]string{"topic": "API", "type": "GraphQL", "level": "intermediate"},
		),
		schema.NewDocument(
			"gRPC is a high-performance RPC framework that uses Protocol Buffers for serialization.",
			map[string]string{"topic": "API", "type": "gRPC", "level": "advanced"},
		),
		schema.NewDocument(
			"Microservices architecture breaks applications into small, independent services.",
			map[string]string{"topic": "Architecture", "pattern": "microservices", "level": "intermediate"},
		),
		schema.NewDocument(
			"Event-driven architecture uses events to trigger and communicate between decoupled services.",
			map[string]string{"topic": "Architecture", "pattern": "events", "level": "advanced"},
		),
		schema.NewDocument(
			"Monolithic architecture consists of a single unified application with all components tightly coupled.",
			map[string]string{"topic": "Architecture", "pattern": "monolithic", "level": "beginner"},
		),
	}

	_, err = vectorStore.AddDocuments(ctx, documents,
		vectorstores.WithEmbedder(embedder),
	)
	if err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}
	fmt.Printf("✅ Added %d documents to knowledge base\n", len(documents))

	// Step 4: Advanced query with multiple retrieval strategies
	query := "What are the different API types and how do they compare?"
	fmt.Printf("\n🔍 Query: %s\n", query)

	// Strategy 1: Direct similarity search
	fmt.Println("\n📊 Strategy 1: Direct Similarity Search")
	docs1, scores1, err := vectorStore.SimilaritySearchByQuery(
		ctx, query, 3, embedder,
	)
	if err != nil {
		log.Fatalf("Failed to search: %v", err)
	}
	fmt.Printf("Retrieved %d documents:\n", len(docs1))
	for i, doc := range docs1 {
		fmt.Printf("  [Score: %.3f] %s\n", scores1[i], truncate(doc.GetContent(), 80))
	}

	// Strategy 2: Using retriever (with filtering)
	fmt.Println("\n📊 Strategy 2: Retriever with Filtering")
	docs2, err := retriever.GetRelevantDocuments(ctx, query)
	if err != nil {
		log.Fatalf("Failed to retrieve: %v", err)
	}
	fmt.Printf("Retrieved %d documents (after filtering):\n", len(docs2))
	for i, doc := range docs2 {
		fmt.Printf("  %d. %s\n", i+1, truncate(doc.GetContent(), 80))
	}

	// Step 5: Build comprehensive context from multiple sources
	contextBuilder := strings.Builder{}
	contextBuilder.WriteString("Use the following information to answer the question comprehensively:\n\n")

	// Add documents from similarity search
	contextBuilder.WriteString("Relevant Information:\n")
	for i, doc := range docs1 {
		contextBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, doc.GetContent()))
	}

	context := contextBuilder.String()

	// Step 6: Generate enhanced response
	fmt.Println("\n🤖 Generating comprehensive response...")
	messages := []schema.Message{
		schema.NewSystemMessage("You are an expert technical assistant. Provide comprehensive, well-structured answers based on the provided context."),
		schema.NewHumanMessage(context + "\n\nQuestion: " + query + "\n\nProvide a detailed comparison and explanation."),
	}

	response, err := llm.Generate(ctx, messages)
	if err != nil {
		log.Fatalf("Failed to generate response: %v", err)
	}

	// Step 7: Display the final answer
	fmt.Printf("\n✅ Comprehensive Answer:\n%s\n", response.GetContent())

	// Step 8: Demonstrate metadata filtering (if supported)
	fmt.Println("\n🔍 Advanced: Metadata Filtering Example")
	fmt.Println("(This would filter documents by metadata attributes like 'level', 'type', etc.)")

	fmt.Println("\n✨ Advanced RAG pipeline example completed successfully!")
}

// truncate truncates a string to the specified length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
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
		content: "Based on the provided context, here's a comprehensive comparison:\n\n1. REST APIs use standard HTTP methods and are widely adopted.\n2. GraphQL provides flexible querying with a single endpoint.\n3. gRPC offers high performance with Protocol Buffers.\n\nEach has its strengths depending on use case requirements.",
	}, nil
}

// mockMessage implements a simple message interface
type mockMessage struct {
	content string
}

func (m *mockMessage) GetContent() string {
	return m.content
}
