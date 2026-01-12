# Multimodal RAG

## Introduction

Welcome to this guide on multimodal RAG (Retrieval-Augmented Generation) in Beluga AI. By the end, you'll understand how to build systems that can answer questions about images, videos, and text documents together.

**What you'll learn:**
- How multimodal embeddings work and when to use them
- How to create embeddings for mixed content (images + text)
- How to store and retrieve multimodal data from vector databases
- How to generate responses using retrieved multimodal context

**Why this matters:**
Traditional RAG systems only work with text. But real-world data includes images, diagrams, charts, and videos. Multimodal RAG lets you build AI systems that understand and reason about all types of content - answering questions like "What does this architecture diagram show?" or "Summarize this product image gallery."

## Prerequisites

Before we begin, make sure you have:

- **Go 1.24+** installed ([installation guide](https://go.dev/doc/install))
- **Beluga AI Framework** installed (`go get github.com/lookatitude/beluga-ai`)
- **OpenAI API key** with access to vision models (GPT-4V) and embeddings
- **Understanding of basic RAG concepts** - if you're new to this, check out our [RAG concepts guide](../concepts/rag.md)
- **A vector database** set up (we'll use pgvector in examples)

## Concepts

Before we start coding, let's understand the key concepts:

### Multimodal Embeddings

Traditional text embeddings convert words into numerical vectors. Multimodal embeddings do the same for images, audio, and video - creating vectors in a shared space where similar content (regardless of modality) is close together.

```
                    Shared Embedding Space
              ┌─────────────────────────────────┐
              │                                 │
    Text ─────│──▶ [0.12, 0.34, ...]           │
              │              ○ "cat sleeping"   │
              │                  ╲              │
   Image ─────│──▶ [0.11, 0.35, ...] ○         │
              │                      cat.jpg    │
              │                                 │
              │                ○ "dog running"  │
   Video ─────│──▶ [0.89, 0.23, ...]           │
              │                                 │
              └─────────────────────────────────┘
```

This means you can:
- Search for images using text queries
- Find similar images regardless of their descriptions
- Answer questions about image content

### Multimodal RAG Pipeline

The multimodal RAG pipeline extends traditional RAG:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        Multimodal RAG Pipeline                          │
│                                                                         │
│   Indexing Phase:                                                       │
│   ┌──────────┐    ┌──────────────┐    ┌──────────────┐    ┌──────────┐ │
│   │  Image   │───▶│  Multimodal  │───▶│   Chunking   │───▶│  Vector  │ │
│   │  + Text  │    │  Embedding   │    │  (if needed) │    │  Store   │ │
│   └──────────┘    └──────────────┘    └──────────────┘    └──────────┘ │
│                                                                         │
│   Query Phase:                                                          │
│   ┌──────────┐    ┌──────────────┐    ┌──────────────┐    ┌──────────┐ │
│   │  Query   │───▶│    Embed     │───▶│   Retrieve   │───▶│ Generate │ │
│   │  (text)  │    │    Query     │    │  Top-K Docs  │    │ Response │ │
│   └──────────┘    └──────────────┘    └──────────────┘    └──────────┘ │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Content Types in Multimodal RAG

| Content Type | Embedding Strategy | Considerations |
|--------------|-------------------|----------------|
| **Text** | Standard text embedding | Chunk appropriately |
| **Images** | CLIP or vision embeddings | Consider image captions |
| **Image + Text** | Combined embedding | Weight importance |
| **Documents with images** | Per-element embeddings | Maintain relationships |
| **Video** | Frame sampling + embedding | Extract key frames |

## Step-by-Step Tutorial

Now let's build a multimodal RAG system step by step.

### Step 1: Set Up Multimodal Embedding Provider

First, we'll create an embedding provider that handles both text and images:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "github.com/lookatitude/beluga-ai/pkg/multimodal"
)

func main() {
    ctx := context.Background()

    // Create an embedding provider that supports multimodal content
    // OpenAI's embedding models can handle text; for images, we use CLIP-style embeddings
    embeddingProvider, err := embeddings.NewOpenAIEmbeddings(
        embeddings.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        embeddings.WithModel("text-embedding-3-small"),
    )
    if err != nil {
        log.Fatalf("Failed to create embedding provider: %v", err)
    }

    // For true multimodal embeddings, we'll use the multimodal package
    multimodalProvider, err := multimodal.NewMultimodalEmbeddings(
        multimodal.WithTextEmbedder(embeddingProvider),
        // Add image embedding capability
        multimodal.WithImageModel("clip-vit-base"),
    )
    if err != nil {
        log.Fatalf("Failed to create multimodal provider: %v", err)
    }

    fmt.Println("Multimodal embedding provider created successfully")
}
```

**What you'll see:**
```
Multimodal embedding provider created successfully
```

**Why this works:** We create a composite embedding provider that can handle different modalities. Text uses standard embeddings, while images use vision-specific models like CLIP.

### Step 2: Create Embeddings for Mixed Content

Now let's embed some mixed content:

```go
    // Define some content to embed - a mix of text and images
    documents := []multimodal.Document{
        {
            ID:      "doc1",
            Type:    multimodal.TypeText,
            Content: "The Beluga whale is a medium-sized toothed whale known for its white color.",
        },
        {
            ID:       "doc2",
            Type:     multimodal.TypeImage,
            ImageURL: "https://example.com/beluga-whale.jpg",
            Caption:  "A beluga whale swimming in arctic waters",
        },
        {
            ID:      "doc3",
            Type:    multimodal.TypeText,
            Content: "Beluga whales are highly social and communicate using clicks and whistles.",
        },
        {
            ID:       "doc4",
            Type:     multimodal.TypeImageText,
            Content:  "Diagram showing beluga whale migration patterns",
            ImageURL: "https://example.com/migration-map.png",
        },
    }

    // Create embeddings for all documents
    embeddedDocs, err := multimodalProvider.EmbedDocuments(ctx, documents)
    if err != nil {
        log.Fatalf("Failed to embed documents: %v", err)
    }

    fmt.Printf("Created embeddings for %d documents\n", len(embeddedDocs))
    for _, doc := range embeddedDocs {
        fmt.Printf("  %s (%s): vector dim = %d\n", doc.ID, doc.Type, len(doc.Embedding))
    }
```

**What you'll see:**
```
Created embeddings for 4 documents
  doc1 (text): vector dim = 1536
  doc2 (image): vector dim = 1536
  doc3 (text): vector dim = 1536
  doc4 (image_text): vector dim = 1536
```

**Why this works:** Each document is embedded according to its type. Images are processed through a vision encoder, text through a text encoder, and combined content uses a fusion strategy to create a unified embedding.

### Step 3: Store in Vector Database

Now we store the embeddings in a vector database:

```go
    import "github.com/lookatitude/beluga-ai/pkg/vectorstores"

    // Create vector store connection
    vectorStore, err := vectorstores.NewPgVector(
        vectorstores.WithConnectionString(os.Getenv("POSTGRES_URL")),
        vectorstores.WithTableName("multimodal_docs"),
        vectorstores.WithDimensions(1536),
    )
    if err != nil {
        log.Fatalf("Failed to create vector store: %v", err)
    }

    // Store embedded documents
    for _, doc := range embeddedDocs {
        err := vectorStore.AddDocument(ctx, vectorstores.Document{
            ID:        doc.ID,
            Content:   doc.Content,
            Embedding: doc.Embedding,
            Metadata: map[string]any{
                "type":     string(doc.Type),
                "image_url": doc.ImageURL,
                "caption":  doc.Caption,
            },
        })
        if err != nil {
            log.Printf("Failed to store document %s: %v", doc.ID, err)
            continue
        }
    }

    fmt.Println("Documents stored in vector database")
```

**Why this works:** We store each embedding with metadata about its type and original content. This lets us reconstruct the multimodal context when retrieving.

### Step 4: Implement Multimodal Retrieval

Now let's implement retrieval that handles multimodal queries:

```go
    // Embed a text query
    query := "What do beluga whales look like?"
    queryEmbedding, err := multimodalProvider.EmbedQuery(ctx, query)
    if err != nil {
        log.Fatalf("Failed to embed query: %v", err)
    }

    // Retrieve similar documents (returns both text and images)
    results, err := vectorStore.SimilaritySearch(ctx, queryEmbedding, 
        vectorstores.WithTopK(3),
    )
    if err != nil {
        log.Fatalf("Retrieval failed: %v", err)
    }

    fmt.Printf("Found %d relevant documents for: %q\n", len(results), query)
    for _, result := range results {
        docType := result.Metadata["type"].(string)
        fmt.Printf("  [%s] %s (score: %.3f)\n", docType, result.ID, result.Score)
    }
```

**What you'll see:**
```
Found 3 relevant documents for: "What do beluga whales look like?"
  [image] doc2 (score: 0.923)
  [text] doc1 (score: 0.891)
  [image_text] doc4 (score: 0.756)
```

**Why this works:** The query embedding exists in the same vector space as both text and image embeddings. Similar content is retrieved regardless of modality - the image of a beluga whale scores highest for a visual question.

### Step 5: Generate Response with Multimodal Context

Finally, we generate a response using the retrieved multimodal context:

```go
    import (
        "github.com/lookatitude/beluga-ai/pkg/llms"
        "github.com/lookatitude/beluga-ai/pkg/schema"
    )

    // Create a vision-capable LLM
    llmClient, err := llms.NewOpenAIChat(
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llms.WithModel("gpt-4-vision-preview"),
    )
    if err != nil {
        log.Fatalf("Failed to create LLM: %v", err)
    }

    // Build multimodal context from retrieved documents
    var contextParts []schema.ContentPart
    for _, result := range results {
        docType := result.Metadata["type"].(string)
        
        switch docType {
        case "text":
            contextParts = append(contextParts, schema.TextContent{
                Text: result.Content,
            })
        case "image", "image_text":
            if imageURL, ok := result.Metadata["image_url"].(string); ok {
                contextParts = append(contextParts, schema.ImageContent{
                    URL: imageURL,
                })
            }
            if result.Content != "" {
                contextParts = append(contextParts, schema.TextContent{
                    Text: result.Content,
                })
            }
        }
    }

    // Create the prompt with multimodal context
    messages := []schema.Message{
        schema.NewSystemMessage("You are a helpful assistant that answers questions using the provided context. The context may include text and images."),
        schema.NewHumanMessageWithContent(append(
            []schema.ContentPart{schema.TextContent{Text: "Context:"}},
            append(contextParts, schema.TextContent{Text: "\n\nQuestion: " + query})...,
        )),
    }

    // Generate response
    response, err := llmClient.Generate(ctx, messages)
    if err != nil {
        log.Fatalf("Generation failed: %v", err)
    }

    fmt.Printf("\nAnswer: %s\n", response.GetContent())
```

**What you'll see:**
```
Answer: Based on the image and text provided, beluga whales are medium-sized 
toothed whales with a distinctive white coloration. In the image, you can see 
a beluga whale swimming in arctic waters, displaying its characteristic rounded 
forehead (called a melon) and lack of dorsal fin. Their white color helps them 
camouflage in icy arctic environments.
```

**Why this works:** GPT-4V can process both the text descriptions and actual images, synthesizing a comprehensive answer that references visual content.

## Code Examples

Here's a complete, production-ready multimodal RAG implementation:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/multimodal"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

var tracer = otel.Tracer("beluga.rag.multimodal")

// MultimodalRAG implements multimodal retrieval-augmented generation
type MultimodalRAG struct {
    embedder    *multimodal.MultimodalEmbeddings
    vectorStore vectorstores.VectorStore
    llm         llms.ChatModel
    topK        int
}

// NewMultimodalRAG creates a new multimodal RAG system
func NewMultimodalRAG(
    embedder *multimodal.MultimodalEmbeddings,
    vectorStore vectorstores.VectorStore,
    llm llms.ChatModel,
    topK int,
) *MultimodalRAG {
    return &MultimodalRAG{
        embedder:    embedder,
        vectorStore: vectorStore,
        llm:         llm,
        topK:        topK,
    }
}

// Index adds documents to the RAG system
func (r *MultimodalRAG) Index(ctx context.Context, documents []multimodal.Document) error {
    ctx, span := tracer.Start(ctx, "multimodal_rag.index",
        trace.WithAttributes(
            attribute.Int("document_count", len(documents)),
        ))
    defer span.End()

    start := time.Now()

    // Embed all documents
    embedded, err := r.embedder.EmbedDocuments(ctx, documents)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return fmt.Errorf("embedding failed: %w", err)
    }

    // Store in vector database
    for _, doc := range embedded {
        err := r.vectorStore.AddDocument(ctx, vectorstores.Document{
            ID:        doc.ID,
            Content:   doc.Content,
            Embedding: doc.Embedding,
            Metadata: map[string]any{
                "type":      string(doc.Type),
                "image_url": doc.ImageURL,
                "caption":   doc.Caption,
            },
        })
        if err != nil {
            span.RecordError(err)
            log.Printf("Failed to store document %s: %v", doc.ID, err)
        }
    }

    span.SetAttributes(
        attribute.Float64("duration_ms", float64(time.Since(start).Milliseconds())),
    )
    span.SetStatus(codes.Ok, "")

    return nil
}

// Query performs multimodal RAG
func (r *MultimodalRAG) Query(ctx context.Context, question string) (string, error) {
    ctx, span := tracer.Start(ctx, "multimodal_rag.query",
        trace.WithAttributes(
            attribute.String("question", question),
        ))
    defer span.End()

    start := time.Now()

    // Embed the query
    queryEmbedding, err := r.embedder.EmbedQuery(ctx, question)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return "", fmt.Errorf("query embedding failed: %w", err)
    }

    // Retrieve relevant documents
    results, err := r.vectorStore.SimilaritySearch(ctx, queryEmbedding,
        vectorstores.WithTopK(r.topK),
    )
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return "", fmt.Errorf("retrieval failed: %w", err)
    }

    span.AddEvent("documents_retrieved", trace.WithAttributes(
        attribute.Int("count", len(results)),
    ))

    // Build multimodal context
    context := r.buildContext(results)

    // Generate response
    response, err := r.generate(ctx, question, context)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return "", fmt.Errorf("generation failed: %w", err)
    }

    span.SetAttributes(
        attribute.Float64("duration_ms", float64(time.Since(start).Milliseconds())),
        attribute.Int("response_length", len(response)),
    )
    span.SetStatus(codes.Ok, "")

    return response, nil
}

func (r *MultimodalRAG) buildContext(results []vectorstores.SearchResult) []schema.ContentPart {
    var parts []schema.ContentPart

    for _, result := range results {
        docType, _ := result.Metadata["type"].(string)

        switch docType {
        case "text":
            parts = append(parts, schema.TextContent{Text: result.Content})
        case "image":
            if url, ok := result.Metadata["image_url"].(string); ok {
                parts = append(parts, schema.ImageContent{URL: url})
            }
            if caption, ok := result.Metadata["caption"].(string); ok && caption != "" {
                parts = append(parts, schema.TextContent{Text: "Image caption: " + caption})
            }
        case "image_text":
            if url, ok := result.Metadata["image_url"].(string); ok {
                parts = append(parts, schema.ImageContent{URL: url})
            }
            parts = append(parts, schema.TextContent{Text: result.Content})
        }
    }

    return parts
}

func (r *MultimodalRAG) generate(ctx context.Context, question string, context []schema.ContentPart) (string, error) {
    systemPrompt := `You are a helpful assistant that answers questions based on the provided context.
The context may include text and images. Analyze all provided content carefully and synthesize
a comprehensive answer. If you reference an image, describe what you see that's relevant.`

    // Build message with multimodal content
    userContent := append(
        []schema.ContentPart{schema.TextContent{Text: "Context:\n"}},
        context...,
    )
    userContent = append(userContent, schema.TextContent{Text: "\n\nQuestion: " + question})

    messages := []schema.Message{
        schema.NewSystemMessage(systemPrompt),
        schema.NewHumanMessageWithContent(userContent),
    }

    response, err := r.llm.Generate(ctx, messages)
    if err != nil {
        return "", err
    }

    return response.GetContent(), nil
}
```

## Testing

Testing multimodal RAG requires mocking embedding and retrieval:

```go
func TestMultimodalRAG_Query(t *testing.T) {
    tests := []struct {
        name          string
        question      string
        mockResults   []vectorstores.SearchResult
        wantErr       bool
    }{
        {
            name:     "text-only results",
            question: "What is a beluga?",
            mockResults: []vectorstores.SearchResult{
                {ID: "1", Content: "Beluga is a whale", Metadata: map[string]any{"type": "text"}},
            },
            wantErr: false,
        },
        {
            name:     "mixed results",
            question: "Show me a beluga",
            mockResults: []vectorstores.SearchResult{
                {ID: "1", Metadata: map[string]any{"type": "image", "image_url": "http://example.com/beluga.jpg"}},
                {ID: "2", Content: "White whale", Metadata: map[string]any{"type": "text"}},
            },
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create mocks
            mockEmbedder := &MockMultimodalEmbedder{}
            mockVectorStore := &MockVectorStore{results: tt.mockResults}
            mockLLM := llms.NewAdvancedMockChatModel(
                llms.WithResponses("Generated answer"),
            )

            rag := NewMultimodalRAG(mockEmbedder, mockVectorStore, mockLLM, 3)

            answer, err := rag.Query(context.Background(), tt.question)
            if (err != nil) != tt.wantErr {
                t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
            }

            if !tt.wantErr && answer == "" {
                t.Error("Query() returned empty answer")
            }
        })
    }
}
```

## Best Practices

### Do

- **Use appropriate embedding models** - CLIP for images, text-embedding-3 for text. Consider multimodal models like BLIP-2 for combined content.
- **Store modality metadata** - Know what type each document is for proper context building.
- **Caption images** - Add text captions to improve retrieval for text queries.
- **Batch embeddings** - Embed multiple documents at once for efficiency.
- **Use vision-capable LLMs** - GPT-4V, Claude 3, or Gemini for generation.

### Don't

- **Don't ignore image context** - Images without captions or metadata are harder to retrieve.
- **Don't mix embedding dimensions** - All embeddings in one store should have the same dimensions.
- **Don't skip chunking** - Long documents still need chunking; images need appropriate resolution.
- **Don't overload context** - Too many images can hit token limits.

### Performance Tips

- **Lazy load images** - Pass URLs instead of raw bytes until generation time
- **Cache embeddings** - Recompute only when content changes
- **Use hybrid search** - Combine vector similarity with keyword matching for better recall

## Troubleshooting

### Q: Images aren't being retrieved for text queries

**A:** Check that your embedding model supports cross-modal retrieval. CLIP-style models embed text and images in the same space. If using separate models, ensure they output compatible dimensions.

### Q: Generation is slow with many images

**A:** Vision models process each image, adding latency. Reduce `topK` or pre-filter to include only the most relevant images. Consider summarizing images before including them.

### Q: Retrieval quality is poor for images

**A:** Add rich captions to images during indexing. The caption text improves retrieval when users search with text queries.

## Related Resources

Now that you understand multimodal RAG, explore:

- **[Multimodal RAG Example](/examples/rag/multimodal/README.md)** - Complete implementation with tests
- **[Vector Stores Guide](../concepts/rag.md)** - Deep dive into vector databases
- **[Streaming LLM Guide](./llm-streaming-tool-calls.md)** - Stream multimodal responses
- **[Embeddings Providers](/docs/providers/embeddings/selection.md)** - Choose the right embedding model
- **[RAG Strategies Use Case](../use-cases/rag-strategies.md)** - Compare RAG approaches
