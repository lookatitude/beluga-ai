package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lookatitude/beluga-ai/pkg/chatmodels/providers/openai"
	_ "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/mock"
	_ "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/openai"
	_ "github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/inmemory"
	"github.com/lookatitude/beluga-ai/pkg/chatmodels"
	chatmodelsiface "github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
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
	vectorStore, err := vectorstores.NewInMemoryStore(ctx,
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
	ids, err := vectorStore.AddDocuments(ctx, documents)
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

	responses, err := llm.GenerateMessages(ctx, messages)
	if err != nil {
		log.Fatalf("Failed to generate response: %v", err)
	}

	// Step 10: Display the final answer
	if len(responses) > 0 {
		fmt.Printf("\n✅ Final Answer:\n%s\n", responses[0].GetContent())
	}

	fmt.Println("\n✨ RAG pipeline example completed successfully!")
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
			APIKey: apiKey,
			Model:  "text-embedding-ada-002",
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
	// Set OpenAI config via environment or use default
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

// mockChatModel is a simple mock implementation.
type mockChatModel struct{}

func (m *mockChatModel) GenerateMessages(ctx context.Context, messages []schema.Message, options ...core.Option) ([]schema.Message, error) {
	return []schema.Message{schema.NewAIMessage("Machine Learning is a subset of Artificial Intelligence that enables machines to automatically learn and improve from experience without being explicitly programmed. It uses algorithms to analyze data, identify patterns, and make decisions or predictions.")}, nil
}

func (m *mockChatModel) StreamMessages(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan schema.Message, error) {
	ch := make(chan schema.Message, 1)
	ch <- schema.NewAIMessage("Machine Learning is a subset of Artificial Intelligence that enables machines to automatically learn and improve from experience without being explicitly programmed. It uses algorithms to analyze data, identify patterns, and make decisions or predictions.")
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
	messages, err := m.GenerateMessages(ctx, []schema.Message{schema.NewHumanMessage(fmt.Sprintf("%v", input))}, options...)
	if err != nil {
		return nil, err
	}
	if len(messages) > 0 {
		return messages[0], nil
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
	msgChan, err := m.StreamMessages(ctx, []schema.Message{schema.NewHumanMessage(fmt.Sprintf("%v", input))}, options...)
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
