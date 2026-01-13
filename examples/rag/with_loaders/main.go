// Example: RAG Pipeline with Document Loaders and Text Splitters
//
// This example demonstrates a complete RAG pipeline using documentloaders
// to load documents from files/directories and textsplitters to chunk them
// before embedding and storage.
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
	"github.com/lookatitude/beluga-ai/pkg/documentloaders"
	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	_ "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/mock"
	_ "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/openai"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/textsplitters"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	_ "github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/inmemory"
)

func main() {
	fmt.Println("📚 Beluga AI - RAG Pipeline with Document Loaders")
	fmt.Println("==================================================")

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

	// Step 4: Create a text splitter
	splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
		textsplitters.WithRecursiveChunkSize(500),
		textsplitters.WithRecursiveChunkOverlap(100),
	)
	if err != nil {
		log.Fatalf("Failed to create splitter: %v", err)
	}
	fmt.Println("✅ Created text splitter")

	// Step 5: Load documents from directory or create sample files
	fmt.Println("\n📂 Loading documents...")
	var documents []schema.Document

	// Try to load from current directory
	fsys := os.DirFS(".")
	loader, err := documentloaders.NewDirectoryLoader(fsys,
		documentloaders.WithMaxDepth(1),
		documentloaders.WithExtensions(".txt", ".md"),
		documentloaders.WithConcurrency(2),
	)
	if err != nil {
		log.Fatalf("Failed to create directory loader: %v", err)
	}

	loadedDocs, err := loader.Load(ctx)
	if err != nil {
		log.Printf("Warning: Failed to load from directory: %v", err)
	}

	if len(loadedDocs) > 0 {
		fmt.Printf("✅ Loaded %d documents from directory\n", len(loadedDocs))
		documents = loadedDocs
	} else {
		// Fallback: Create sample documents in memory
		fmt.Println("⚠️  No documents found in directory, using sample documents")
		documents = createSampleDocuments()
	}

	// Step 6: Split documents into chunks
	fmt.Println("\n✂️  Splitting documents into chunks...")
	chunks, err := splitter.SplitDocuments(ctx, documents)
	if err != nil {
		log.Fatalf("Failed to split documents: %v", err)
	}
	fmt.Printf("✅ Split %d documents into %d chunks\n", len(documents), len(chunks))

	// Show chunk metadata
	if len(chunks) > 0 {
		fmt.Println("\n📊 Chunk metadata example:")
		exampleChunk := chunks[0]
		fmt.Printf("  Source: %s\n", exampleChunk.Metadata["source"])
		if idx, ok := exampleChunk.Metadata["chunk_index"]; ok {
			fmt.Printf("  Chunk index: %s\n", idx)
		}
		if total, ok := exampleChunk.Metadata["chunk_total"]; ok {
			fmt.Printf("  Total chunks: %s\n", total)
		}
	}

	// Step 7: Add chunks to the vector store
	fmt.Println("\n📝 Adding chunks to vector store...")
	ids, err := vectorStore.AddDocuments(ctx, chunks)
	if err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}
	fmt.Printf("✅ Added %d chunks (IDs: %d total)\n", len(chunks), len(ids))

	// Step 8: Query the knowledge base
	query := "What is artificial intelligence?"
	fmt.Printf("\n🔍 Query: %s\n", query)

	// Step 9: Retrieve relevant chunks
	relevantDocs, scores, err := vectorStore.SimilaritySearchByQuery(
		ctx, query, 3, embedder,
	)
	if err != nil {
		log.Fatalf("Failed to search: %v", err)
	}

	fmt.Printf("✅ Retrieved %d relevant chunks:\n", len(relevantDocs))
	for i, doc := range relevantDocs {
		content := doc.GetContent()
		if len(content) > 100 {
			content = content[:100] + "..."
		}
		fmt.Printf("  [Score: %.3f] %s (source: %s)\n", scores[i], content, doc.Metadata["source"])
	}

	// Step 10: Build context from retrieved chunks
	contextBuilder := strings.Builder{}
	contextBuilder.WriteString("Use the following context to answer the question:\n\n")
	for i, doc := range relevantDocs {
		contextBuilder.WriteString(fmt.Sprintf("Chunk %d: %s\n", i+1, doc.GetContent()))
	}

	context := contextBuilder.String()

	// Step 11: Generate response using LLM with context
	fmt.Println("\n🤖 Generating response with context...")
	messages := []schema.Message{
		schema.NewSystemMessage("You are a helpful assistant that answers questions based on the provided context."),
		schema.NewHumanMessage(context + "\n\nQuestion: " + query),
	}

	responses, err := llm.GenerateMessages(ctx, messages)
	if err != nil {
		log.Fatalf("Failed to generate response: %v", err)
	}

	// Step 12: Display the final answer
	if len(responses) > 0 {
		fmt.Printf("\n✅ Final Answer:\n%s\n", responses[0].GetContent())
	}

	fmt.Println("\n✨ RAG pipeline with document loaders completed successfully!")
}

// createSampleDocuments creates sample documents for demonstration.
func createSampleDocuments() []schema.Document {
	return []schema.Document{
		schema.NewDocument(
			"Artificial Intelligence (AI) is the simulation of human intelligence in machines. "+
				"AI systems are designed to perform tasks that typically require human intelligence, "+
				"such as visual perception, speech recognition, decision-making, and language translation. "+
				"Machine learning is a subset of AI that enables machines to learn from data without being explicitly programmed.",
			map[string]string{"source": "sample_ai_intro.txt", "topic": "AI"},
		),
		schema.NewDocument(
			"Deep Learning uses neural networks with multiple layers to understand complex patterns in data. "+
				"These networks are inspired by the structure of the human brain and can learn hierarchical representations. "+
				"Deep learning has revolutionized fields like computer vision, natural language processing, and speech recognition.",
			map[string]string{"source": "sample_deep_learning.txt", "topic": "Deep Learning"},
		),
		schema.NewDocument(
			"Natural Language Processing (NLP) helps computers understand, interpret, and generate human language. "+
				"NLP combines computational linguistics with machine learning to enable applications like "+
				"chatbots, translation services, and sentiment analysis. Modern NLP relies heavily on transformer architectures.",
			map[string]string{"source": "sample_nlp.txt", "topic": "NLP"},
		),
	}
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

// mockChatModel is a simple mock implementation.
type mockChatModel struct{}

func (m *mockChatModel) GenerateMessages(ctx context.Context, messages []schema.Message, options ...core.Option) ([]schema.Message, error) {
	return []schema.Message{schema.NewAIMessage("Artificial Intelligence (AI) is the simulation of human intelligence in machines. Based on the provided context, AI systems can perform tasks like visual perception, speech recognition, and decision-making. Machine learning, a subset of AI, enables machines to learn from data without explicit programming.")}, nil
}

func (m *mockChatModel) StreamMessages(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan schema.Message, error) {
	ch := make(chan schema.Message, 1)
	ch <- schema.NewAIMessage("Artificial Intelligence (AI) is the simulation of human intelligence in machines.")
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
