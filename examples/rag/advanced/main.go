package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/chatmodels"
	chatmodelsiface "github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
	_ "github.com/lookatitude/beluga-ai/pkg/chatmodels/providers/openai"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	_ "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/mock"
	_ "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/openai"
	"github.com/lookatitude/beluga-ai/pkg/documentloaders"
	"github.com/lookatitude/beluga-ai/pkg/retrievers"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/textsplitters"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	_ "github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/inmemory"
	"testing/fstest"
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

	vectorStore, err := vectorstores.NewInMemoryStore(ctx,
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
		retrievers.WithDefaultK(5), // Retrieve top 5 documents
	)
	if err != nil {
		log.Fatalf("Failed to create retriever: %v", err)
	}
	fmt.Println("✅ Created advanced retriever with filtering")

	// Step 3: Load and split documents using documentloaders and textsplitters
	fmt.Println("\n📝 Loading and splitting documents for the knowledge base...")

	// Create a mock filesystem with diverse documents
	mockFS := fstest.MapFS{
		"api/rest.txt": &fstest.MapFile{
			Data: []byte("REST APIs use HTTP methods like GET, POST, PUT, DELETE to perform operations on resources."),
		},
		"api/graphql.txt": &fstest.MapFile{
			Data: []byte("GraphQL is a query language that allows clients to request exactly the data they need from a single endpoint."),
		},
		"api/grpc.txt": &fstest.MapFile{
			Data: []byte("gRPC is a high-performance RPC framework that uses Protocol Buffers for serialization."),
		},
		"architecture/microservices.txt": &fstest.MapFile{
			Data: []byte("Microservices architecture breaks applications into small, independent services."),
		},
		"architecture/event-driven.txt": &fstest.MapFile{
			Data: []byte("Event-driven architecture uses events to trigger and communicate between decoupled services."),
		},
		"architecture/monolithic.txt": &fstest.MapFile{
			Data: []byte("Monolithic architecture consists of a single unified application with all components tightly coupled."),
		},
	}

	// Use directory loader with advanced options
	loader, err := documentloaders.NewDirectoryLoader(mockFS,
		documentloaders.WithMaxDepth(2),
		documentloaders.WithExtensions(".txt"),
		documentloaders.WithConcurrency(3), // Use 3 concurrent workers
	)
	if err != nil {
		log.Fatalf("Failed to create directory loader: %v", err)
	}

	loadedDocs, err := loader.Load(ctx)
	if err != nil {
		log.Fatalf("Failed to load documents: %v", err)
	}
	fmt.Printf("✅ Loaded %d documents from directory\n", len(loadedDocs))

	// Use recursive splitter with advanced options
	splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
		textsplitters.WithRecursiveChunkSize(200),
		textsplitters.WithRecursiveChunkOverlap(40),
	)
	if err != nil {
		log.Fatalf("Failed to create text splitter: %v", err)
	}

	chunks, err := splitter.SplitDocuments(ctx, loadedDocs)
	if err != nil {
		log.Fatalf("Failed to split documents: %v", err)
	}
	fmt.Printf("✅ Split into %d chunks\n", len(chunks))

	// Add metadata to chunks based on file path
	for i := range chunks {
		if chunks[i].Metadata == nil {
			chunks[i].Metadata = make(map[string]string)
		}
		source := chunks[i].Metadata["source"]
		if strings.Contains(source, "api/") {
			chunks[i].Metadata["topic"] = "API"
			if strings.Contains(source, "rest") {
				chunks[i].Metadata["type"] = "REST"
				chunks[i].Metadata["level"] = "beginner"
			} else if strings.Contains(source, "graphql") {
				chunks[i].Metadata["type"] = "GraphQL"
				chunks[i].Metadata["level"] = "intermediate"
			} else if strings.Contains(source, "grpc") {
				chunks[i].Metadata["type"] = "gRPC"
				chunks[i].Metadata["level"] = "advanced"
			}
		} else if strings.Contains(source, "architecture/") {
			chunks[i].Metadata["topic"] = "Architecture"
			if strings.Contains(source, "microservices") {
				chunks[i].Metadata["pattern"] = "microservices"
				chunks[i].Metadata["level"] = "intermediate"
			} else if strings.Contains(source, "event") {
				chunks[i].Metadata["pattern"] = "events"
				chunks[i].Metadata["level"] = "advanced"
			} else if strings.Contains(source, "monolithic") {
				chunks[i].Metadata["pattern"] = "monolithic"
				chunks[i].Metadata["level"] = "beginner"
			}
		}
	}

	_, err = vectorStore.AddDocuments(ctx, chunks)
	if err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}
	fmt.Printf("✅ Added %d chunks to knowledge base\n", len(chunks))

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

	responses, err := llm.GenerateMessages(ctx, messages)
	if err != nil {
		log.Fatalf("Failed to generate response: %v", err)
	}

	// Step 7: Display the final answer
	if len(responses) > 0 {
		fmt.Printf("\n✅ Comprehensive Answer:\n%s\n", responses[0].GetContent())
	}

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

// createLLM creates a ChatModel instance.
func createLLM(ctx context.Context) (chatmodelsiface.ChatModel, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  OPENAI_API_KEY not set, using mock ChatModel")
		return &mockChatModel{}, nil
	}

	config := chatmodels.DefaultConfig()
	config.DefaultProvider = "openai"
	if config.Providers == nil {
		config.Providers = make(map[string]any)
	}
	config.Providers["openai"] = map[string]any{
		"api_key": apiKey,
		"model":   "gpt-3.5-turbo",
		"enabled": true,
	}

	llm, err := chatmodels.NewChatModel("gpt-3.5-turbo", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChatModel: %w", err)
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

// mockChatModel is a simple mock implementation.
type mockChatModel struct{}

func (m *mockChatModel) GenerateMessages(ctx context.Context, messages []schema.Message, options ...core.Option) ([]schema.Message, error) {
	return []schema.Message{schema.NewAIMessage("Based on the provided context, here's a comprehensive comparison:\n\n1. REST APIs use standard HTTP methods and are widely adopted.\n2. GraphQL provides flexible querying with a single endpoint.\n3. gRPC offers high performance with Protocol Buffers.\n\nEach has its strengths depending on use case requirements.")}, nil
}

func (m *mockChatModel) StreamMessages(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan schema.Message, error) {
	ch := make(chan schema.Message, 1)
	ch <- schema.NewAIMessage("Based on the provided context, here's a comprehensive comparison:\n\n1. REST APIs use standard HTTP methods and are widely adopted.\n2. GraphQL provides flexible querying with a single endpoint.\n3. gRPC offers high performance with Protocol Buffers.\n\nEach has its strengths depending on use case requirements.")
	close(ch)
	return ch, nil
}

func (m *mockChatModel) GetModelInfo() chatmodelsiface.ModelInfo {
	return chatmodelsiface.ModelInfo{
		Name:     "mock-model",
		Provider: "mock-provider",
	}
}

func (m *mockChatModel) CheckHealth() map[string]any {
	return map[string]any{"status": "healthy"}
}

func (m *mockChatModel) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	messages, err := ensureMessages(input)
	if err != nil {
		return nil, err
	}
	responses, err := m.GenerateMessages(ctx, messages, options...)
	if err != nil {
		return nil, err
	}
	if len(responses) > 0 {
		return responses[0], nil
	}
	return nil, fmt.Errorf("no messages generated")
}

func (m *mockChatModel) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := m.Invoke(ctx, input, options...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (m *mockChatModel) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	messages, err := ensureMessages(input)
	if err != nil {
		return nil, err
	}
	msgChan, err := m.StreamMessages(ctx, messages, options...)
	if err != nil {
		return nil, err
	}
	anyChan := make(chan any)
	go func() {
		defer close(anyChan)
		for msg := range msgChan {
			anyChan <- msg
		}
	}()
	return anyChan, nil
}

func ensureMessages(input any) ([]schema.Message, error) {
	switch v := input.(type) {
	case []schema.Message:
		return v, nil
	case map[string]interface{}:
		if inputVal, ok := v["input"].(string); ok {
			return []schema.Message{schema.NewHumanMessage(inputVal)}, nil
		}
		return []schema.Message{schema.NewHumanMessage(fmt.Sprintf("%v", v))}, nil
	case string:
		return []schema.Message{schema.NewHumanMessage(v)}, nil
	default:
		return []schema.Message{schema.NewHumanMessage(fmt.Sprintf("%v", input))}, nil
	}
}
