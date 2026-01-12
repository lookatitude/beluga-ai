# Multimodal RAG Example

## Description

This example shows you how to build a multimodal RAG (Retrieval-Augmented Generation) system in Beluga AI. You'll learn:

- How to index documents of different types (text, images, image+text)
- How to create embeddings for mixed content
- How to perform similarity search across modalities
- How to generate answers using retrieved multimodal context

## Prerequisites

Before running this example, you need:

| Requirement | Version | Notes |
|-------------|---------|-------|
| Go | 1.24+ | [Install Go](https://go.dev/doc/install) |
| Beluga AI | latest | `go get github.com/lookatitude/beluga-ai` |
| OpenAI API Key | - | Get one at [OpenAI](https://platform.openai.com/api-keys) |

### Environment Setup

Set these environment variables:

```bash
export OPENAI_API_KEY="your-api-key-here"

# Optional: Enable debug logging
export BELUGA_LOG_LEVEL=debug
```

## Usage

### Running the Example

1. **Navigate to the example directory**:

```bash
cd examples/rag/multimodal
```

2. **Run the example**:

```bash
go run multimodal_rag.go
```

### Expected Output

When successful, you'll see:

```
=== Multimodal RAG Example ===

Indexing documents...
Indexed 6 documents

Question: What do beluga whales look like?
---
Retrieved documents:
  1. [image] Score: 0.892 - A beluga whale swimming gracefully in clear arctic wa...
  2. [text] Score: 0.845 - Beluga whales are medium-sized toothed whales known...
  3. [image] Score: 0.756 - A pod of approximately 15 beluga whales swimming tog...

Answer: Based on the context provided, beluga whales have several distinctive physical 
characteristics. They are medium-sized toothed whales known for their distinctive white 
coloration. Adult beluga whales are pure white, while younger calves show a grayish 
coloration...

==================================================

Question: How do beluga whales communicate?
---
...
```

## Code Structure

```
multimodal/
├── README.md                # This file
├── multimodal_rag.go        # Main example implementation
└── multimodal_rag_test.go   # Comprehensive test suite
```

### Key Components

| File | Purpose |
|------|---------|
| `multimodal_rag.go` | Main implementation with indexing, search, and generation |
| `multimodal_rag_test.go` | Tests for all components including similarity search |

### Design Decisions

This example demonstrates these Beluga AI patterns:

- **In-memory vector store**: Simplified for demonstration; production would use pgvector or similar
- **Text-based image embeddings**: Uses captions for image embedding (CLIP integration would be production-ready)
- **Modular architecture**: Separate indexing, retrieval, and generation phases
- **OTEL Instrumentation**: Full tracing for debugging and monitoring

## Document Types

The example supports three document types:

| Type | Description | Indexing Strategy |
|------|-------------|-------------------|
| `text` | Plain text documents | Embed content directly |
| `image` | Image with caption | Embed caption, store URL |
| `image_text` | Image + description | Embed description, store URL |

## Testing

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run with race detection
go test -race ./...

# Run with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...
```

### Test Structure

| Test | What it verifies |
|------|-----------------|
| `TestNewMultimodalRAGExample` | Example creation |
| `TestIndexDocuments` | Document indexing for all types |
| `TestQuery` | End-to-end query functionality |
| `TestSimilaritySearch` | Correct ranking by similarity |
| `TestCosineSimilarity` | Math correctness |
| `TestDocumentTypes` | All document types are represented |
| `TestTruncate` | Helper function behavior |
| `BenchmarkIndexing` | Indexing performance |
| `BenchmarkQuery` | Query performance |
| `BenchmarkSimilaritySearch` | Search performance at scale |

### Expected Test Output

```
=== RUN   TestNewMultimodalRAGExample
--- PASS: TestNewMultimodalRAGExample (0.00s)
=== RUN   TestIndexDocuments
--- PASS: TestIndexDocuments (0.01s)
=== RUN   TestQuery
--- PASS: TestQuery (0.02s)
=== RUN   TestSimilaritySearch
--- PASS: TestSimilaritySearch (0.00s)
=== RUN   TestCosineSimilarity
--- PASS: TestCosineSimilarity (0.00s)
PASS
coverage: 87.3% of statements
```

## Extending This Example

### Using a Real Vector Database

Replace the in-memory store with pgvector:

```go
import "github.com/lookatitude/beluga-ai/pkg/vectorstores"

vectorStore, _ := vectorstores.NewPgVector(
    vectorstores.WithConnectionString(os.Getenv("DATABASE_URL")),
    vectorstores.WithTableName("multimodal_docs"),
)

// Use vectorStore.AddDocument() instead of in-memory storage
```

### Adding Image Embedding Support

For true multimodal embeddings (not just caption-based):

```go
import "github.com/lookatitude/beluga-ai/pkg/multimodal"

multimodalEmbedder, _ := multimodal.NewMultimodalEmbeddings(
    multimodal.WithTextEmbedder(textEmbedder),
    multimodal.WithImageModel("clip-vit-base"),
)

// Use multimodalEmbedder for both text and images
```

### Vision-Capable Generation

Use GPT-4V for image-aware generation:

```go
llmClient, _ := llms.NewOpenAIChat(
    llms.WithModel("gpt-4-vision-preview"),
)

// Pass actual images (not just URLs) in the context
```

## Troubleshooting

### Common Issues

<details>
<summary>❌ Error: "OPENAI_API_KEY environment variable is required"</summary>

**Cause:** The `OPENAI_API_KEY` environment variable is not set.

**Solution:**
```bash
export OPENAI_API_KEY="sk-..."
```
</details>

<details>
<summary>❌ Low relevance scores for image queries</summary>

**Cause:** Text queries may not match image captions well.

**Solution:**
1. Improve image captions with more detail
2. Add keywords that users might search for
3. Consider using CLIP embeddings for true cross-modal search
</details>

<details>
<summary>❌ Slow indexing with many documents</summary>

**Cause:** Each document requires an API call for embedding.

**Solution:**
1. Batch documents in EmbedDocuments calls
2. Use a local embedding model
3. Cache embeddings and only recompute on changes
</details>

## Related Examples

After completing this example, you might want to explore:

- **[Advanced RAG](../advanced/README.md)** - More sophisticated retrieval strategies
- **[RAG with Loaders](../with_loaders/README.md)** - Load documents from files
- **[Vector Stores](/examples/vectorstores/README.md)** - Database integration

## Learn More

- **[Multimodal RAG Guide](/docs/guides/rag-multimodal.md)** - In-depth guide
- **[RAG Concepts](/docs/concepts/rag.md)** - Understanding RAG fundamentals
- **[Embeddings Guide](/docs/providers/embeddings/selection.md)** - Choosing embedding models
