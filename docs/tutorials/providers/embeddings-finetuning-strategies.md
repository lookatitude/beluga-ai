# Fine-tuning Embedding Strategies

In this tutorial, you'll learn advanced strategies for generating and optimizing text embeddings to improve retrieval performance in RAG applications.

## Learning Objectives

- ✅ Understand embedding dimensions and models
- ✅ Implement batch embedding for performance
- ✅ Choose the right embedding model for your domain
- ✅ Optimize chunking for better embeddings

## Prerequisites

- Basic RAG knowledge (see [Simple RAG](../../getting-started/02-simple-rag.md))
- Go 1.24+

## Why Optimization Matters?

Standard embeddings (like OpenAI's `text-embedding-3-small`) are good, but:
- **Cost**: Embedding millions of documents adds up.
- **Latency**: Real-time embedding needs to be fast.
- **Accuracy**: Generic models might miss domain-specific jargon (e.g., medical, legal).

## Step 1: Choosing a Model

Beluga AI supports multiple embedders.
```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/embeddings"
)

func main() {
    ctx := context.Background()

    // 1. OpenAI (General Purpose, Paid)
    openaiConfig := &embeddings.Config{
        OpenAI: &embeddings.OpenAIConfig{
            APIKey: os.Getenv("OPENAI_API_KEY"),
            Model:  "text-embedding-3-large", // Higher dimension = better accuracy
        },
    }
    
    // 2. Ollama (Local, Free, Privacy)
    ollamaConfig := &embeddings.Config{
        Ollama: &embeddings.OllamaConfig{
            ServerURL: "http://localhost:11434",
            Model:     "nomic-embed-text", // Optimized for retrieval
        },
    }
    
    // ... factory creation ...
}
```

## Step 2: Batch Processing

Embedding one document at a time is slow. Use `EmbedDocuments` for batching.
```go
func batchEmbed(embedder embeddingsiface.Embedder, docs []string, batchSize int) ([][]float32, error) {
    var allEmbeddings [][]float32
    
    for i := 0; i < len(docs); i += batchSize {
        end := i + batchSize
        if end > len(docs) {
            end = len(docs)
        }
        
        batch := docs[i:end]
        fmt.Printf("Embedding batch %d-%d\n", i, end)
        
        embeddings, err := embedder.EmbedDocuments(context.Background(), batch)
        if err != nil {
            return nil, err
        }
        allEmbeddings = append(allEmbeddings, embeddings...)
    }

    
    return allEmbeddings, nil
}
```

## Step 3: Domain Adaptation (Concept)

While you can't "fine-tune" the OpenAI API model directly via Beluga AI, you can:
1. **Use a specialized local model**: Use Ollama with a fine-tuned BERT model.
2. **Hybrid Search**: Combine embeddings with keyword search (BM25) to cover jargon.

## Step 4: Chunking for Embeddings

The quality of embeddings depends heavily on chunking.

- **Too small**: Loss of context.
- **Too large**: Diluted meaning (the vector is an "average" of too many topics).

Recommended strategy: **Semantic Chunking**.
```
// Pseudo-code for semantic chunking logic
// 1. Split text into sentences.
// 2. Embed each sentence.
// 3. Group sentences with high cosine similarity.
// 4. Embed the groups.

## Verification

1. Embed a list of 100 sentences using batching.
2. Measure the time taken vs. sequential embedding.
3. Compare results from `text-embedding-3-small` vs `nomic-embed-text`.

## Next Steps

- **[Multimodal Embeddings](./embeddings-multimodal-google.md)** - Embed images and video
- **[Production pgvector Sharding](./vectorstores-pgvector-sharding.md)** - Store millions of vectors
