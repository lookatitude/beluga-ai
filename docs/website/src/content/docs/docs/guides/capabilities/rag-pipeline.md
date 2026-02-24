---
title: RAG Pipeline Guide
description: "Build retrieval-augmented generation pipelines in Go with Beluga AI — document loading, embeddings, vector stores, hybrid search, CRAG, and HyDE strategies."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, RAG, retrieval-augmented generation, embeddings, vector store, hybrid search, retriever"
---

Language models generate answers from their training data, but they cannot access your private documents, recent data, or domain-specific knowledge. Retrieval-Augmented Generation (RAG) solves this by fetching relevant documents at query time and injecting them into the LLM's context window. The model then generates answers grounded in your actual data rather than relying on potentially outdated or hallucinated information.

The `rag/` package provides a complete, modular pipeline for building RAG systems. Each stage — loading, splitting, embedding, storing, and retrieving — is a separate package with its own interface, registry, and providers. This decomposition lets you swap any component independently: change your vector database without touching your embedding logic, or upgrade your retrieval strategy without modifying your document pipeline.

## Pipeline Architecture

```mermaid
graph LR
  subgraph Indexing
    A[Documents] --> B[Loader] --> C[Splitter] --> D[Embedder] --> E[VectorStore]
  end
  subgraph Query
    F[Query] --> G[Embedder] --> H[Retriever] --> I[Relevant Docs] --> J[LLM] --> K[Response]
  end
  E -.-> H
```

Each stage is a separate package with its own interface, registry, and providers:

| Package | Interface | Purpose |
|---------|-----------|---------|
| `rag/loader` | `DocumentLoader` | Load content from files, URLs, APIs |
| `rag/splitter` | `TextSplitter` | Chunk documents for embedding |
| `rag/embedding` | `Embedder` | Convert text to vectors |
| `rag/vectorstore` | `VectorStore` | Store and search embeddings |
| `rag/retriever` | `Retriever` | Find relevant documents |

## Document Loading

The first step in any RAG pipeline is getting your data into a structured format. Document loaders read content from various sources — files, URLs, APIs — and produce `schema.Document` values with both content and metadata. The registry pattern means you can add new loader types (databases, cloud storage, custom APIs) without modifying existing code.

Load documents from various sources:

```go
import (
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/loader"
)

// Load a text file
textLoader, err := loader.New("text", config.ProviderConfig{})
if err != nil {
	log.Fatal(err)
}

docs, err := textLoader.Load(ctx, "/path/to/document.txt")
if err != nil {
	log.Fatal(err)
}
```

### Built-in Loaders

| Loader | Format | Description |
|--------|--------|-------------|
| `text` | Plain text | Simple file loading |
| `json` | JSON | Configurable path extraction |
| `csv` | CSV | One document per row |
| `markdown` | Markdown | Structure-aware loading |

### Document Transformers

After loading, you often need to enrich documents with additional metadata for filtering and auditing. Transformers let you add source attribution, timestamps, or any custom metadata before documents enter the splitting stage. This metadata is preserved through splitting and stored alongside embeddings, enabling filtered searches later.

```go
// Add metadata to every document
addSource := loader.TransformerFunc(func(ctx context.Context, doc schema.Document) (schema.Document, error) {
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]any)
	}
	doc.Metadata["source"] = "internal-docs"
	doc.Metadata["loaded_at"] = time.Now().Format(time.RFC3339)
	return doc, nil
})
```

## Text Splitting

Embedding models have token limits — typically 512 to 8192 tokens depending on the model. Documents that exceed these limits must be split into smaller chunks. But splitting is not just about fitting token budgets: smaller, focused chunks improve retrieval precision because each chunk's embedding captures a narrower semantic meaning, making it easier to match against specific queries.

The `chunk_overlap` parameter controls how many characters overlap between adjacent chunks. Overlap prevents information loss at split boundaries — without it, a sentence that spans two chunks would be cut in half, and neither chunk would contain the complete thought. An overlap of 10-20% of the chunk size is typically sufficient to preserve context.

Split documents into chunks optimized for embedding:

```go
import (
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/splitter"
)

s, err := splitter.New("recursive", config.ProviderConfig{
	Options: map[string]any{
		"chunk_size":    1000,
		"chunk_overlap": 200,
	},
})
if err != nil {
	log.Fatal(err)
}

// Split raw text
chunks, err := s.Split(ctx, longText)

// Or split documents (preserves metadata)
chunkedDocs, err := s.SplitDocuments(ctx, docs)
```

`SplitDocuments` preserves the original metadata and adds `chunk_index`, `chunk_total`, and `parent_id` to each chunk.

### Built-in Splitters

| Splitter | Strategy | Best For |
|----------|----------|----------|
| `recursive` | Recursive character boundaries | General-purpose text |
| `markdown` | Heading hierarchy | Markdown documents |
| `token` | Token-based boundaries | Precise token-budget chunks |

## Embeddings

Embeddings convert text into dense vector representations where semantically similar texts are close together in vector space. This is the core mechanism that enables semantic search — finding documents by meaning rather than exact keyword matches. The embedding model you choose affects both the quality of retrieval and the dimensionality (and therefore storage cost) of your vectors.

Convert text to vector representations:

```go
import (
	"github.com/lookatitude/beluga-ai/rag/embedding"
	_ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
)

embedder, err := embedding.New("openai", embedding.ProviderConfig{
	APIKey: os.Getenv("OPENAI_API_KEY"),
	Model:  "text-embedding-3-small",
})
if err != nil {
	log.Fatal(err)
}

// Embed a batch of texts
vectors, err := embedder.Embed(ctx, []string{"hello world", "goodbye world"})

// Embed a single text
vec, err := embedder.EmbedSingle(ctx, "search query")

// Check dimensions
fmt.Println("Dimensions:", embedder.Dimensions())
```

### Embedding Providers

| Provider | Import Path | Models |
|----------|-------------|--------|
| OpenAI | `rag/embedding/providers/openai` | `text-embedding-3-small`, `text-embedding-3-large` |
| Google | `rag/embedding/providers/google` | `text-embedding-004` |
| Cohere | `rag/embedding/providers/cohere` | `embed-english-v3.0` |
| Voyage | `rag/embedding/providers/voyage` | `voyage-3` |
| Mistral | `rag/embedding/providers/mistral` | `mistral-embed` |
| Jina | `rag/embedding/providers/jina` | `jina-embeddings-v3` |
| Ollama | `rag/embedding/providers/ollama` | Local models |
| Sentence Transformers | `rag/embedding/providers/sentence_transformers` | Local models |
| In-Memory | `rag/embedding/providers/inmemory` | Test/dev (random vectors) |

## Vector Store

Vector stores persist embeddings and support efficient similarity search over them. When a query arrives, it is embedded using the same model, and the vector store finds the nearest neighbors — the documents most semantically similar to the query. Different backends offer different trade-offs between latency, scalability, filtering capabilities, and operational complexity.

Store and search embeddings:

```go
import (
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	_ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
)

store, err := vectorstore.New("pgvector", vectorstore.ProviderConfig{
	ConnectionString: os.Getenv("DATABASE_URL"),
})
if err != nil {
	log.Fatal(err)
}

// Add documents with embeddings
err = store.Add(ctx, chunkedDocs, vectors)

// Search for similar documents
queryVec, err := embedder.EmbedSingle(ctx, "What is Go?")
results, err := store.Search(ctx, queryVec, 10,
	vectorstore.WithThreshold(0.7),
	vectorstore.WithFilter(map[string]any{"source": "internal-docs"}),
)
```

### Search Options

| Option | Description |
|--------|-------------|
| `WithThreshold(t)` | Minimum similarity score (0.0–1.0) |
| `WithFilter(meta)` | Match metadata key-value pairs |
| `WithStrategy(s)` | Distance metric: `Cosine`, `DotProduct`, `Euclidean` |

### Vector Store Providers

| Provider | Import Path | Type |
|----------|-------------|------|
| In-Memory | `rag/vectorstore/providers/inmemory` | Development/testing |
| pgvector | `rag/vectorstore/providers/pgvector` | PostgreSQL extension |
| Pinecone | `rag/vectorstore/providers/pinecone` | Managed cloud |
| Qdrant | `rag/vectorstore/providers/qdrant` | Open-source |
| Weaviate | `rag/vectorstore/providers/weaviate` | Open-source |
| Milvus | `rag/vectorstore/providers/milvus` | Open-source |
| Chroma | `rag/vectorstore/providers/chroma` | Open-source |
| Redis | `rag/vectorstore/providers/redis` | Redis Stack |
| Elasticsearch | `rag/vectorstore/providers/elasticsearch` | Elastic |
| MongoDB | `rag/vectorstore/providers/mongodb` | Atlas Vector Search |
| SQLite-vec | `rag/vectorstore/providers/sqlitevec` | Embedded |
| Vespa | `rag/vectorstore/providers/vespa` | Enterprise search |
| Turbopuffer | `rag/vectorstore/providers/turbopuffer` | Serverless |

## Retriever

The `Retriever` interface abstracts the search step, decoupling your application from specific vector store implementations and search strategies. Retrievers can combine multiple backends, apply reranking, or implement advanced strategies like CRAG and HyDE. This abstraction is where the most impactful RAG quality improvements happen — choosing the right retrieval strategy often matters more than choosing the right embedding model.

```go
import "github.com/lookatitude/beluga-ai/rag/retriever"

docs, err := r.Retrieve(ctx, "What is quantum computing?",
	retriever.WithTopK(5),
	retriever.WithThreshold(0.7),
	retriever.WithMetadata(map[string]any{"topic": "physics"}),
)
```

### Retrieval Strategies

| Strategy | Description | When to Use |
|----------|-------------|-------------|
| `vector` | Pure vector similarity search | Simple use cases |
| `hybrid` | Vector + BM25 with RRF fusion | **Recommended default** |
| `crag` | Corrective RAG with quality grading | Quality-critical applications |
| `hyde` | Hypothetical Document Embeddings | Sparse-data domains |
| `adaptive` | Adjusts strategy based on query | Variable query patterns |
| `ensemble` | Combines multiple retriever outputs | Maximum recall |

### Hybrid Search (Recommended)

Pure vector search excels at finding semantically similar content but can miss documents that contain the exact keywords a user is looking for. Conversely, BM25 keyword matching finds exact term matches but misses paraphrases and synonyms. Hybrid search combines both signals using Reciprocal Rank Fusion (RRF), which merges the ranked results from each method into a single list. This is the recommended default because it handles both precise keyword queries ("error code 404") and conceptual queries ("how to handle missing pages") effectively.

```go
hybridRetriever, err := retriever.New("hybrid", retriever.ProviderConfig{
	Options: map[string]any{
		"vector_store": store,
		"embedder":     embedder,
		"bm25_weight":  0.3,
		"vector_weight": 0.7,
	},
})

docs, err := hybridRetriever.Retrieve(ctx, "Go concurrency patterns",
	retriever.WithTopK(10),
)
```

### CRAG (Corrective RAG)

A fundamental problem with naive RAG is that retrieved documents may be irrelevant to the query. When an LLM receives irrelevant context, it often generates plausible-sounding but incorrect answers — a form of hallucination. Corrective RAG addresses this by using an LLM to grade each retrieved document for relevance before passing it to the generation step. Documents below the confidence threshold are discarded, and if too few relevant documents remain, CRAG can trigger a web search as a fallback. This quality-gating step significantly reduces hallucination in production systems.

```go
cragRetriever, err := retriever.New("crag", retriever.ProviderConfig{
	Options: map[string]any{
		"base_retriever": baseRetriever,
		"grader_llm":     model,
		"threshold":      0.6,
	},
})
```

### HyDE (Hypothetical Document Embeddings)

Short or vague user queries often produce poor embeddings because there is not enough semantic content to capture the user's intent. For example, the query "auth" generates a very different embedding than a paragraph explaining authentication flows. HyDE solves this by first asking an LLM to generate a hypothetical document that would answer the query, then embedding that hypothetical answer instead of the raw query. The hypothetical document's embedding is much closer in vector space to the actual relevant documents, dramatically improving recall for sparse-data domains and terse queries.

```go
hydeRetriever, err := retriever.New("hyde", retriever.ProviderConfig{
	Options: map[string]any{
		"base_retriever": baseRetriever,
		"llm":            model,
		"embedder":       embedder,
	},
})
```

## Complete Pipeline Example

The following example demonstrates the full RAG pipeline from end to end: loading a text file, splitting it into chunks, embedding the chunks, storing them in an in-memory vector database, retrieving relevant context for a query, and generating an answer with an LLM. In production, you would replace the in-memory store with a persistent backend like pgvector or Pinecone.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/rag/loader"
	"github.com/lookatitude/beluga-ai/rag/retriever"
	"github.com/lookatitude/beluga-ai/rag/splitter"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/config"

	_ "github.com/lookatitude/beluga-ai/llm/providers/openai"
	_ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
	_ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/inmemory"
)

func main() {
	ctx := context.Background()

	// 1. Load documents
	l, err := loader.New("text", config.ProviderConfig{})
	if err != nil {
		log.Fatal(err)
	}
	docs, err := l.Load(ctx, "knowledge-base.txt")
	if err != nil {
		log.Fatal(err)
	}

	// 2. Split into chunks
	s, err := splitter.New("recursive", config.ProviderConfig{
		Options: map[string]any{"chunk_size": 500, "chunk_overlap": 50},
	})
	if err != nil {
		log.Fatal(err)
	}
	chunks, err := s.SplitDocuments(ctx, docs)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Embed chunks
	emb, err := embedding.New("openai", embedding.ProviderConfig{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "text-embedding-3-small",
	})
	if err != nil {
		log.Fatal(err)
	}
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Content
	}
	vectors, err := emb.Embed(ctx, texts)
	if err != nil {
		log.Fatal(err)
	}

	// 4. Store in vector database
	store, err := vectorstore.New("inmemory", vectorstore.ProviderConfig{})
	if err != nil {
		log.Fatal(err)
	}
	err = store.Add(ctx, chunks, vectors)
	if err != nil {
		log.Fatal(err)
	}

	// 5. Retrieve relevant context
	query := "How does error handling work?"
	queryVec, err := emb.EmbedSingle(ctx, query)
	if err != nil {
		log.Fatal(err)
	}
	relevant, err := store.Search(ctx, queryVec, 5)
	if err != nil {
		log.Fatal(err)
	}

	// 6. Generate answer with context
	model, err := llm.New("openai", llm.ProviderConfig{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-4o",
	})
	if err != nil {
		log.Fatal(err)
	}

	contextStr := ""
	for _, doc := range relevant {
		contextStr += doc.Content + "\n\n"
	}

	msgs := []schema.Message{
		schema.NewSystemMessage("Answer the question using the provided context. If unsure, say so."),
		schema.NewHumanMessage(fmt.Sprintf("Context:\n%s\nQuestion: %s", contextStr, query)),
	}

	resp, err := model.Generate(ctx, msgs)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.Text())
}
```

## Retriever Hooks

Beluga AI uses the hooks pattern across all subsystems for lifecycle observation without wrapping. Retriever hooks let you log queries, measure latency, audit which documents were retrieved, and track reranking behavior. Hooks are optional function fields — any nil hook is simply skipped, so you only pay for the observation you need.

```go
hooks := retriever.Hooks{
	BeforeRetrieve: func(ctx context.Context, query string) error {
		log.Printf("Retrieving for: %q", query)
		return nil
	},
	AfterRetrieve: func(ctx context.Context, docs []schema.Document, err error) {
		log.Printf("Found %d documents", len(docs))
	},
	OnRerank: func(ctx context.Context, query string, before, after []schema.Document) {
		log.Printf("Reranked: %d → %d documents", len(before), len(after))
	},
}
```

## Next Steps

- [Working with LLMs](/guides/working-with-llms/) — The ChatModel that generates answers
- [Memory System](/guides/memory-system/) — Persistent memory with vector search
- [Tools & MCP](/guides/tools-and-mcp/) — Give agents retrieval as a tool
- [Monitoring & Observability](/guides/observability/) — Trace RAG pipeline performance
