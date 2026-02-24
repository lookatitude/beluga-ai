---
title: Weaviate RAG Connector
description: "Integrate Weaviate hybrid vector and keyword search with Beluga AI RAG pipelines for semantic similarity and metadata filtering in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Weaviate, hybrid search, vector database, Beluga AI, RAG connector, semantic search Go, BM25 vector fusion"
---

Hybrid search -- combining vector similarity with keyword matching -- consistently outperforms either approach alone in retrieval benchmarks. Weaviate provides this natively: a single query can blend vector scores with BM25 keyword scores using a configurable `alpha` parameter, without requiring separate Elasticsearch and vector database deployments.

Choose Weaviate when you need hybrid search in a single system, want graph-based data modeling alongside vector search, or prefer a self-hosted or Weaviate Cloud Services deployment with built-in multi-tenancy support.

## Overview

This integration provides:

- Vector similarity search using pre-computed embeddings
- Hybrid search combining vector and keyword retrieval with configurable weighting
- Integration with Beluga AI's `Embedder` interface for query embedding
- OpenTelemetry tracing for production observability
- Results returned as `schema.Document` for seamless pipeline integration

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed
- Weaviate instance (local or Weaviate Cloud Services)
- Weaviate Go client v4

## Installation

Install the Weaviate Go client:

```bash
go get github.com/weaviate/weaviate-go-client/v4
```

Start a local Weaviate instance for development:

```bash
docker run -d -p 8080:8080 -p 50051:50051 \
  -e QUERY_DEFAULTS_LIMIT=25 \
  -e AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED=true \
  -e PERSISTENCE_DATA_PATH='/var/lib/weaviate' \
  semitechnologies/weaviate:latest
```

## Usage

### Creating the Weaviate Retriever

Define a retriever that embeds queries and searches Weaviate via its GraphQL API:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/schema"
)

// WeaviateRetriever performs vector similarity search against a Weaviate instance.
type WeaviateRetriever struct {
	client    *weaviate.Client
	className string
	embedder  embedding.Embedder
	defaultK  int
}

func NewWeaviateRetriever(weaviateURL, className string, embedder embedding.Embedder, defaultK int) (*WeaviateRetriever, error) {
	cfg := weaviate.Config{
		Host:   weaviateURL,
		Scheme: "http",
	}

	client, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &WeaviateRetriever{
		client:    client,
		className: className,
		embedder:  embedder,
		defaultK:  defaultK,
	}, nil
}

func (r *WeaviateRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	// Generate query embedding
	queryEmbedding, err := r.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	vector := make([]float32, len(queryEmbedding))
	for i, v := range queryEmbedding {
		vector[i] = float32(v)
	}

	// Build GraphQL query with vector similarity
	builder := r.client.GraphQL().Get().
		WithClassName(r.className).
		WithFields(graphql.Field{
			Name: "content",
		}, graphql.Field{
			Name: "metadata",
		}).
		WithNearVector(&graphql.NearVectorArgumentBuilder{}.
			WithVector(vector).
			WithCertainty(0.7))

	result, err := builder.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Parse results into schema.Document
	get := result.Data["Get"].(map[string]interface{})
	items := get[r.className].([]interface{})

	documents := make([]schema.Document, 0, len(items))
	for _, item := range items {
		itemMap := item.(map[string]interface{})
		content, _ := itemMap["content"].(string)
		metadata, _ := itemMap["metadata"].(map[string]interface{})

		meta := make(map[string]string)
		for k, v := range metadata {
			meta[k] = fmt.Sprintf("%v", v)
		}

		documents = append(documents, schema.NewDocument(content, meta))
	}

	return documents, nil
}
```

### Hybrid Search

Weaviate supports hybrid search that blends vector similarity with BM25 keyword matching. The `alpha` parameter controls the balance: `0.0` is pure keyword, `1.0` is pure vector.

```go
func (r *WeaviateRetriever) HybridSearch(ctx context.Context, query string, alpha float32) ([]schema.Document, error) {
	queryEmbedding, err := r.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	vector := make([]float32, len(queryEmbedding))
	for i, v := range queryEmbedding {
		vector[i] = float32(v)
	}

	builder := r.client.GraphQL().Get().
		WithClassName(r.className).
		WithFields(graphql.Field{Name: "content"}, graphql.Field{Name: "metadata"}).
		WithHybrid(&graphql.HybridArgumentBuilder{}.
			WithQuery(query).
			WithVector(vector).
			WithAlpha(alpha))

	result, err := builder.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("hybrid search failed: %w", err)
	}

	get := result.Data["Get"].(map[string]interface{})
	items := get[r.className].([]interface{})

	documents := make([]schema.Document, 0, len(items))
	for _, item := range items {
		itemMap := item.(map[string]interface{})
		content, _ := itemMap["content"].(string)
		metadata, _ := itemMap["metadata"].(map[string]interface{})

		meta := make(map[string]string)
		for k, v := range metadata {
			meta[k] = fmt.Sprintf("%v", v)
		}

		documents = append(documents, schema.NewDocument(content, meta))
	}

	return documents, nil
}
```

### End-to-End Example

```go
func main() {
	ctx := context.Background()

	// Create embedder via Beluga AI registry
	emb, err := embedding.New("openai", config.ProviderConfig{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "text-embedding-ada-002",
	})
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}

	// Create Weaviate retriever
	retriever, err := NewWeaviateRetriever("localhost:8080", "Document", emb, 5)
	if err != nil {
		log.Fatalf("Failed to create retriever: %v", err)
	}

	// Vector similarity search
	docs, err := retriever.GetRelevantDocuments(ctx, "machine learning")
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	fmt.Printf("Retrieved %d documents\n", len(docs))
}
```

## Advanced Topics

### Production Retriever with Observability

Add OpenTelemetry tracing for production deployments:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ProductionWeaviateRetriever struct {
	client    *weaviate.Client
	className string
	embedder  embedding.Embedder
	defaultK  int
	tracer    trace.Tracer
}

func NewProductionWeaviateRetriever(weaviateURL, className string, embedder embedding.Embedder, defaultK int) (*ProductionWeaviateRetriever, error) {
	cfg := weaviate.Config{
		Host:   weaviateURL,
		Scheme: "http",
	}

	client, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &ProductionWeaviateRetriever{
		client:    client,
		className: className,
		embedder:  embedder,
		defaultK:  defaultK,
		tracer:    otel.Tracer("beluga.retriever.weaviate"),
	}, nil
}

func (r *ProductionWeaviateRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	ctx, span := r.tracer.Start(ctx, "weaviate.search",
		trace.WithAttributes(
			attribute.String("class", r.className),
			attribute.String("query", query),
		),
	)
	defer span.End()

	queryEmbedding, err := r.embedder.EmbedQuery(ctx, query)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("embedding failed: %w", err)
	}

	vector := make([]float32, len(queryEmbedding))
	for i, v := range queryEmbedding {
		vector[i] = float32(v)
	}

	builder := r.client.GraphQL().Get().
		WithClassName(r.className).
		WithFields(graphql.Field{Name: "content"}, graphql.Field{Name: "metadata"}).
		WithLimit(r.defaultK).
		WithNearVector(&graphql.NearVectorArgumentBuilder{}.
			WithVector(vector).
			WithCertainty(0.7))

	result, err := builder.Do(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("search failed: %w", err)
	}

	get := result.Data["Get"].(map[string]interface{})
	items := get[r.className].([]interface{})

	documents := make([]schema.Document, 0, len(items))
	for _, item := range items {
		itemMap := item.(map[string]interface{})
		content, _ := itemMap["content"].(string)
		metadata, _ := itemMap["metadata"].(map[string]interface{})

		meta := make(map[string]string)
		for k, v := range metadata {
			meta[k] = fmt.Sprintf("%v", v)
		}

		documents = append(documents, schema.NewDocument(content, meta))
	}

	span.SetAttributes(attribute.Int("documents_retrieved", len(documents)))
	return documents, nil
}

func main() {
	ctx := context.Background()

	emb, err := embedding.New("openai", config.ProviderConfig{
		APIKey: os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}

	retriever, err := NewProductionWeaviateRetriever("localhost:8080", "Document", emb, 5)
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}

	docs, err := retriever.GetRelevantDocuments(ctx, "AI and machine learning")
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}

	fmt.Printf("Retrieved %d documents\n", len(docs))
}
```

## Configuration

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `WeaviateURL` | Weaviate server host | `localhost:8080` | No |
| `ClassName` | Weaviate class name | `Document` | No |
| `DefaultK` | Number of results to return | `5` | No |
| `Alpha` | Hybrid search weight (0=keyword, 1=vector) | `0.5` | No |

## Troubleshooting

### Connection refused

Weaviate is not running. Start it with Docker:

```bash
docker run -d -p 8080:8080 -p 50051:50051 \
  -e AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED=true \
  semitechnologies/weaviate:latest
```

### Class not found

The Weaviate class has not been created yet. Create it via the Weaviate REST API or Go client before performing queries.

## Production Considerations

- **Schema design** -- Define Weaviate classes that match your data model and query patterns
- **Vectorization modules** -- Configure server-side vectorization or use client-side embeddings
- **Hybrid search** -- Use hybrid search with an `alpha` of `0.5` as a starting point, then tune based on evaluation
- **Monitoring** -- Track query latency and resource usage via Weaviate metrics endpoints
- **Multi-tenancy** -- Use Weaviate's multi-tenancy support for isolating tenant data

## Related Resources

- [Elasticsearch Keyword Search](/docs/integrations/elasticsearch-search) -- Full-text keyword search with Elasticsearch
- [Vector Stores](/docs/integrations/vector-stores) -- Vector store provider overview
- [Embedding Providers](/docs/integrations/embedding-providers) -- Embedding provider configuration
