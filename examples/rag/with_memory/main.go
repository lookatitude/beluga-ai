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
	"github.com/lookatitude/beluga-ai/pkg/documentloaders"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/textsplitters"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	"testing/fstest"
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

	// Step 3: Load documents using documentloaders
	fmt.Println("\n📝 Loading documents for the knowledge base using documentloaders...")

	// Create a mock filesystem with programming language documents
	mockFS := fstest.MapFS{
		"languages/python.txt": &fstest.MapFile{
			Data: []byte("Python is a high-level programming language known for its simplicity and readability."),
		},
		"languages/go.txt": &fstest.MapFile{
			Data: []byte("Go (Golang) is a statically typed language developed by Google, known for its performance and concurrency features."),
		},
		"languages/javascript.txt": &fstest.MapFile{
			Data: []byte("JavaScript is a dynamic programming language primarily used for web development."),
		},
	}

	// Use directory loader to load documents
	loader, err := documentloaders.NewDirectoryLoader(mockFS,
		documentloaders.WithMaxDepth(1),
		documentloaders.WithExtensions(".txt"),
	)
	if err != nil {
		log.Fatalf("Failed to create directory loader: %v", err)
	}

	loadedDocs, err := loader.Load(ctx)
	if err != nil {
		log.Fatalf("Failed to load documents: %v", err)
	}
	fmt.Printf("✅ Loaded %d documents from directory\n", len(loadedDocs))

	// Use text splitter to split documents into chunks
	splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
		textsplitters.WithRecursiveChunkSize(100),
		textsplitters.WithRecursiveChunkOverlap(20),
	)
	if err != nil {
		log.Fatalf("Failed to create text splitter: %v", err)
	}

	chunks, err := splitter.SplitDocuments(ctx, loadedDocs)
	if err != nil {
		log.Fatalf("Failed to split documents: %v", err)
	}

	// Add metadata to chunks based on file path
	for i := range chunks {
		if chunks[i].Metadata == nil {
			chunks[i].Metadata = make(map[string]string)
		}
		source := chunks[i].Metadata["source"]
		if strings.Contains(source, "python") {
			chunks[i].Metadata["topic"] = "Python"
			chunks[i].Metadata["type"] = "language"
		} else if strings.Contains(source, "go") {
			chunks[i].Metadata["topic"] = "Go"
			chunks[i].Metadata["type"] = "language"
		} else if strings.Contains(source, "javascript") {
			chunks[i].Metadata["topic"] = "JavaScript"
			chunks[i].Metadata["type"] = "language"
		}
	}

	_, err = vectorStore.AddDocuments(ctx, chunks)
	if err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}
	fmt.Printf("✅ Added %d chunks to knowledge base\n", len(chunks))

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

		responses, err := llm.GenerateMessages(ctx, messages)
		if err != nil {
			log.Fatalf("Failed to generate response: %v", err)
		}

		var responseText string
		if len(responses) > 0 {
			responseText = responses[0].GetContent()
		}
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
	return []schema.Message{schema.NewAIMessage("Based on the context provided, here is a helpful answer to your question.")}, nil
}

func (m *mockChatModel) StreamMessages(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan schema.Message, error) {
	ch := make(chan schema.Message, 1)
	ch <- schema.NewAIMessage("Based on the context provided, here is a helpful answer to your question.")
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
