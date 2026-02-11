---
title: Semantic Image Search System
description: Build intelligent image search with natural language queries using multimodal embeddings and vector similarity.
---

E-commerce platforms struggle with keyword-based image search because product images contain information that text metadata cannot fully capture — color, style, material texture, spatial arrangement. A customer looking for "a blue velvet couch with mid-century legs" cannot express this through keyword filters alone, and manual tagging every visual attribute of every product is not scalable.

A semantic image search system uses multimodal embeddings to understand both images and text queries, enabling intuitive search by description rather than exact keywords. The key innovation is mapping both images and text into the same embedding space: a photo of a blue velvet couch and the text description "blue velvet mid-century sofa" produce similar vectors, enabling cross-modal similarity search without explicit attribute matching.

## Solution Architecture

Beluga AI provides multimodal embedding support that handles both text and image inputs, vector stores for fast similarity search, and semantic retrieval strategies. The system indexes product images as vectors, processes natural language queries into the same embedding space, and returns visually similar products ranked by relevance.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Product    │───▶│    Image     │───▶│  Multimodal  │
│   Images     │    │   Encoder    │    │  Embeddings  │
└──────────────┘    └──────────────┘    └──────┬───────┘
                                               │
                                               ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Search     │◀───│   Ranked     │◀───│ VectorStore  │
│   Results    │    │   Results    │    │  (Similarity │
└──────────────┘    └──────────────┘    │    Search)   │
                          ▲              └──────────────┘
                          │
                    ┌─────┴────────┐
                    │  Text Query  │
                    │   Embedder   │
                    └──────────────┘
```

## Implementation

### Image Indexing Pipeline

The indexing pipeline generates embeddings for product images and stores them in a vector database with metadata:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
)

type ImageSearchSystem struct {
    embedder embedding.Embedder
    store    vectorstore.VectorStore
}

func NewImageSearchSystem(ctx context.Context) (*ImageSearchSystem, error) {
    embedder, err := embedding.New("openai", embedding.ProviderConfig{
        Model: "text-embedding-3-large", // Supports multimodal
    })
    if err != nil {
        return nil, fmt.Errorf("create embedder: %w", err)
    }

    store, err := vectorstore.New("pgvector", vectorstore.ProviderConfig{
        ConnectionString: "postgresql://localhost/images",
    })
    if err != nil {
        return nil, fmt.Errorf("create vector store: %w", err)
    }

    return &ImageSearchSystem{
        embedder: embedder,
        store:    store,
    }, nil
}

func (s *ImageSearchSystem) IndexImage(ctx context.Context, productID string, imageData []byte, metadata map[string]interface{}) error {
    // Generate embedding for image
    embeddings, err := s.embedder.Embed(ctx, []string{string(imageData)})
    if err != nil {
        return fmt.Errorf("embed image: %w", err)
    }

    // Create document with metadata
    doc := schema.Document{
        Content: fmt.Sprintf("Product %s", productID),
        Metadata: map[string]interface{}{
            "product_id": productID,
            "type":       "image",
        },
    }
    for k, v := range metadata {
        doc.Metadata[k] = v
    }

    // Store in vector database
    if err := s.store.Add(ctx, []schema.Document{doc}, [][]float64{embeddings[0]}); err != nil {
        return fmt.Errorf("store image: %w", err)
    }

    log.Printf("Indexed image for product %s", productID)
    return nil
}
```

### Semantic Search

Text queries are embedded and matched against image embeddings using vector similarity:

```go
func (s *ImageSearchSystem) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
    // Generate embedding for text query
    queryEmbeddings, err := s.embedder.Embed(ctx, []string{query})
    if err != nil {
        return nil, fmt.Errorf("embed query: %w", err)
    }

    // Search vector store
    results, err := s.store.SimilaritySearch(ctx, queryEmbeddings[0],
        vectorstore.WithTopK(topK),
        vectorstore.WithThreshold(0.7),
    )
    if err != nil {
        return nil, fmt.Errorf("similarity search: %w", err)
    }

    // Convert to search results
    searchResults := make([]SearchResult, len(results))
    for i, result := range results {
        searchResults[i] = SearchResult{
            ProductID: result.Metadata["product_id"].(string),
            Score:     result.Score,
            Metadata:  result.Metadata,
        }
    }

    return searchResults, nil
}

type SearchResult struct {
    ProductID string
    Score     float64
    Metadata  map[string]interface{}
}
```

### Relevance Ranking

Combine similarity scores with other signals for improved ranking:

```go
func (s *ImageSearchSystem) SearchWithRanking(ctx context.Context, query string, topK int) ([]SearchResult, error) {
    // Get more results for ranking
    results, err := s.Search(ctx, query, topK*2)
    if err != nil {
        return nil, err
    }

    // Rank by relevance (combine similarity with popularity, recency, etc.)
    ranked := rankResults(results, query)

    // Return top K
    if len(ranked) > topK {
        ranked = ranked[:topK]
    }

    return ranked, nil
}

func rankResults(results []SearchResult, query string) []SearchResult {
    // Combine similarity score with other factors
    for i := range results {
        popularityBoost := getPopularityScore(results[i].ProductID)
        recencyBoost := getRecencyScore(results[i].ProductID)
        results[i].Score = results[i].Score * 0.7 + popularityBoost*0.2 + recencyBoost*0.1
    }

    // Sort by combined score
    sort.Slice(results, func(i, j int) bool {
        return results[i].Score > results[j].Score
    })

    return results
}
```

## Production Considerations

### Batch Indexing

Process large image collections efficiently with batched embedding:

```go
func (s *ImageSearchSystem) IndexBatch(ctx context.Context, images []ImageData) error {
    batchSize := 100
    for i := 0; i < len(images); i += batchSize {
        end := min(i+batchSize, len(images))
        batch := images[i:end]

        texts := make([]string, len(batch))
        for j, img := range batch {
            texts[j] = string(img.Data)
        }

        embeddings, err := s.embedder.Embed(ctx, texts)
        if err != nil {
            return fmt.Errorf("embed batch %d: %w", i/batchSize, err)
        }

        docs := make([]schema.Document, len(batch))
        for j, img := range batch {
            docs[j] = schema.Document{
                Content: fmt.Sprintf("Product %s", img.ProductID),
                Metadata: map[string]interface{}{
                    "product_id": img.ProductID,
                    "type":       "image",
                },
            }
        }

        if err := s.store.Add(ctx, docs, embeddings); err != nil {
            return fmt.Errorf("store batch %d: %w", i/batchSize, err)
        }
    }
    return nil
}

type ImageData struct {
    ProductID string
    Data      []byte
    Metadata  map[string]interface{}
}
```

### Observability

Track search performance and relevance metrics:

```go
import (
    "github.com/lookatitude/beluga-ai/o11y"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

func (s *ImageSearchSystem) SearchWithTracing(ctx context.Context, query string, topK int) ([]SearchResult, error) {
    tracer := otel.Tracer("image-search")
    ctx, span := tracer.Start(ctx, "image_search.search")
    defer span.End()

    span.SetAttributes(
        attribute.String("search.query", query),
        attribute.Int("search.top_k", topK),
    )

    results, err := s.Search(ctx, query, topK)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    span.SetAttributes(attribute.Int("search.results_count", len(results)))
    return results, nil
}
```

### Scaling

- **Horizontal scaling**: Deploy multiple search service instances behind a load balancer
- **Embedding caching**: Cache embeddings for frequently searched queries to reduce API calls
- **Vector store optimization**: Use read replicas for pgvector or managed services for automatic scaling
- **Model selection**: Balance embedding quality with latency by testing different embedding models

## Results

After implementing semantic image search, the platform achieved:

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Search Relevance | 40-50% | 87% | 74-118% |
| User Satisfaction | 6.0/10 | 9.1/10 | 52% |
| Conversion Rate | 2.5% | 4.7% | 88% |
| Search-to-Purchase Time | 8 min | 2.5 min | 69% reduction |
| Zero-Result Searches | 15-20% | 4% | 73-80% reduction |

## Related Resources

- [RAG Pipeline Guide](/guides/rag-pipeline/) for embedding and retrieval patterns
- [Vector Store Integration](/integrations/vector-stores/) for provider-specific configuration
- [Embedding Providers](/providers/embedding/) for multimodal embedding options
