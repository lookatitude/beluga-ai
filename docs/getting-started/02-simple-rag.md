# Part 2: Building a Simple RAG Application

In this tutorial, you'll learn how to build a Retrieval-Augmented Generation (RAG) application from scratch. RAG combines information retrieval with language generation to create AI systems that can answer questions using your own documents.

## Learning Objectives

- ✅ Understand RAG architecture
- ✅ Generate embeddings for documents
- ✅ Store documents in a vector store
- ✅ Retrieve relevant documents
- ✅ Generate answers using retrieved context

## Prerequisites

- Completed [Part 1: Your First LLM Call](./01-first-llm-call.md)
- API key for OpenAI (for embeddings) or Ollama installed locally
- Basic understanding of vector embeddings

## What is RAG?

RAG (Retrieval-Augmented Generation) is a technique that:
1. **Retrieves** relevant documents from a knowledge base
2. **Augments** the LLM prompt with retrieved context
3. **Generates** answers based on the augmented context

This allows LLMs to answer questions using information not in their training data.

## Step 1: Project Setup

Create a new file `rag_example.go`:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
)
```

## Step 2: Initialize Components

### Create Embedder

```go
func setupEmbedder(ctx context.Context) (embeddingsiface.Embedder, error) {
	config := &embeddings.Config{
		OpenAI: &embeddings.OpenAIConfig{
			APIKey: os.Getenv("OPENAI_API_KEY"),
			Model:  "text-embedding-ada-002",
		},
	}

	factory, err := embeddings.NewEmbedderFactory(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder factory: %w", err)
	}

	embedder, err := factory.NewEmbedder("openai")
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	return embedder, nil
}
```

### Create Vector Store

```go
func setupVectorStore(ctx context.Context, embedder embeddingsiface.Embedder) (vectorstoresiface.VectorStore, error) {
	// Use in-memory store for simplicity
	store, err := vectorstores.NewInMemoryStore(ctx,
		vectorstores.WithEmbedder(embedder),
		vectorstores.WithSearchK(5), // Return top 5 results
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector store: %w", err)
	}

	return store, nil
}
```

### Create LLM

```go
func setupLLM(ctx context.Context) (llmsiface.ChatModel, error) {
	config := llms.NewConfig(
		llms.WithProvider("openai"),
		llms.WithModelName("gpt-3.5-turbo"),
		llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		llms.WithTemperatureConfig(0.7),
	)

	factory := llms.NewFactory()
	provider, err := factory.CreateProvider("openai", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}

	return provider, nil
}
```

## Step 3: Add Documents to Vector Store

```go
func addDocuments(ctx context.Context, store vectorstoresiface.VectorStore, embedder embeddingsiface.Embedder) error {
	// Create sample documents
	documents := []schema.Document{
		schema.NewDocument(
			"Artificial intelligence is the simulation of human intelligence in machines.",
			map[string]string{"topic": "AI", "source": "intro.md"},
		),
		schema.NewDocument(
			"Machine learning is a subset of AI that uses statistical techniques to enable computers to learn.",
			map[string]string{"topic": "ML", "source": "ml.md"},
		),
		schema.NewDocument(
			"Deep learning uses neural networks with multiple layers to learn complex patterns.",
			map[string]string{"topic": "DL", "source": "dl.md"},
		),
		schema.NewDocument(
			"Natural language processing helps computers understand and generate human language.",
			map[string]string{"topic": "NLP", "source": "nlp.md"},
		),
	}

	// Add documents to vector store
	ids, err := store.AddDocuments(ctx, documents,
		vectorstoresiface.WithEmbedder(embedder),
	)
	if err != nil {
		return fmt.Errorf("failed to add documents: %w", err)
	}

	fmt.Printf("Added %d documents with IDs: %v\n", len(ids), ids)
	return nil
}
```

## Step 4: Retrieve Relevant Documents

```go
func retrieveDocuments(ctx context.Context, store vectorstoresiface.VectorStore, embedder embeddingsiface.Embedder, query string, k int) ([]schema.Document, []float32, error) {
	// Search for relevant documents
	docs, scores, err := store.SimilaritySearchByQuery(ctx, query, k, embedder)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to search: %w", err)
	}

	fmt.Printf("Retrieved %d documents:\n", len(docs))
	for i, doc := range docs {
		fmt.Printf("  [Score: %.3f] %s\n", scores[i], doc.GetContent())
	}

	return docs, scores, nil
}
```

## Step 5: Generate Answer with Context

```go
func generateAnswer(ctx context.Context, llm llmsiface.ChatModel, query string, contextDocs []schema.Document) (string, error) {
	// Build context from retrieved documents
	contextText := ""
	for i, doc := range contextDocs {
		contextText += fmt.Sprintf("Document %d: %s\n", i+1, doc.GetContent())
	}

	// Create prompt with context
	messages := []schema.Message{
		schema.NewSystemMessage(
			"You are a helpful assistant. Use the following context to answer the question. " +
				"If the context doesn't contain the answer, say so.\n\n" +
				"Context:\n" + contextText,
		),
		schema.NewHumanMessage(query),
	}

	// Generate response
	response, err := llm.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	return response.Content, nil
}
```

## Step 6: Complete RAG Pipeline

```go
func main() {
	ctx := context.Background()

	// Step 1: Setup components
	fmt.Println("Setting up components...")
	embedder, err := setupEmbedder(ctx)
	if err != nil {
		fmt.Printf("Error setting up embedder: %v\n", err)
		return
	}

	store, err := setupVectorStore(ctx, embedder)
	if err != nil {
		fmt.Printf("Error setting up vector store: %v\n", err)
		return
	}

	llm, err := setupLLM(ctx)
	if err != nil {
		fmt.Printf("Error setting up LLM: %v\n", err)
		return
	}

	// Step 2: Add documents
	fmt.Println("\nAdding documents to knowledge base...")
	if err := addDocuments(ctx, store, embedder); err != nil {
		fmt.Printf("Error adding documents: %v\n", err)
		return
	}

	// Step 3: Query
	query := "What is machine learning?"
	fmt.Printf("\nQuery: %s\n", query)

	// Step 4: Retrieve relevant documents
	fmt.Println("\nRetrieving relevant documents...")
	relevantDocs, scores, err := retrieveDocuments(ctx, store, embedder, query, 2)
	if err != nil {
		fmt.Printf("Error retrieving documents: %v\n", err)
		return
	}

	// Step 5: Generate answer
	fmt.Println("\nGenerating answer...")
	answer, err := generateAnswer(ctx, llm, query, relevantDocs)
	if err != nil {
		fmt.Printf("Error generating answer: %v\n", err)
		return
	}

	// Step 6: Display result
	fmt.Printf("\nAnswer: %s\n", answer)
}
```

## Step 7: Using Ollama for Local RAG

For privacy-sensitive applications, use Ollama:

```go
func setupOllamaEmbedder(ctx context.Context) (embeddingsiface.Embedder, error) {
	config := &embeddings.Config{
		Ollama: &embeddings.OllamaConfig{
			ServerURL: "http://localhost:11434",
			Model:     "nomic-embed-text",
		},
	}

	factory, err := embeddings.NewEmbedderFactory(config)
	if err != nil {
		return nil, err
	}

	return factory.NewEmbedder("ollama")
}
```

Make sure Ollama is running:
```bash
ollama pull nomic-embed-text
ollama serve
```

## Step 8: Advanced: Using PgVector for Persistence

For production, use PostgreSQL with pgvector:

```go
func setupPgVectorStore(ctx context.Context, embedder embeddingsiface.Embedder) (vectorstoresiface.VectorStore, error) {
	store, err := vectorstores.NewPgVectorStore(ctx,
		vectorstores.WithEmbedder(embedder),
		vectorstores.WithProviderConfig("connection_string", 
			"postgres://user:password@localhost/dbname?sslmode=disable"),
		vectorstores.WithProviderConfig("table_name", "documents"),
		vectorstores.WithSearchK(10),
	)
	return store, err
}
```

## Complete Example

Here's the complete RAG application:

```go
package main

import (
	"context"
	"fmt"
	"os"

	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

func main() {
	ctx := context.Background()

	// Setup
	embedder, _ := setupEmbedder(ctx)
	store, _ := setupVectorStore(ctx, embedder)
	llm, _ := setupLLM(ctx)

	// Add documents
	documents := []schema.Document{
		schema.NewDocument("AI is the simulation of human intelligence.", 
			map[string]string{"topic": "AI"}),
		schema.NewDocument("ML uses statistical techniques to enable learning.", 
			map[string]string{"topic": "ML"}),
	}
	store.AddDocuments(ctx, documents, vectorstoresiface.WithEmbedder(embedder))

	// Query
	query := "What is machine learning?"
	docs, _, _ := store.SimilaritySearchByQuery(ctx, query, 2, embedder)

	// Generate answer
	contextText := ""
	for _, doc := range docs {
		contextText += doc.GetContent() + "\n"
	}

	messages := []schema.Message{
		schema.NewSystemMessage("Answer using this context: " + contextText),
		schema.NewHumanMessage(query),
	}

	response, _ := llm.Generate(ctx, messages)
	fmt.Printf("Answer: %s\n", response.Content)
}
```

## Exercises

1. **Add more documents**: Expand the knowledge base with more topics
2. **Experiment with k**: Try different values for the number of retrieved documents
3. **Improve prompts**: Enhance the system message for better answers
4. **Add metadata filtering**: Filter documents by metadata (topic, source, etc.)
5. **Implement chunking**: Split large documents into smaller chunks

## Next Steps

Congratulations! You've built a RAG application. Next, learn how to:

- **[Part 3: Creating Your First Agent](./03-first-agent.md)** - Build autonomous agents
- **[Part 5: Memory Management](./05-memory-management.md)** - Add conversation memory
- **[Concepts: RAG](../concepts/rag.md)** - Deep dive into RAG concepts

---

**Ready for the next step?** Continue to [Part 3: Creating Your First Agent](./03-first-agent.md)!

