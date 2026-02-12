---
title: Elasticsearch Keyword Search
description: "Integrate Elasticsearch BM25 keyword search with Beluga AI for hybrid RAG pipelines combining vector and full-text retrieval in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Elasticsearch, keyword search, BM25, hybrid RAG, Beluga AI, full-text search, Go search integration"
---

Vector search alone can miss results when users search for specific terms, product codes, or proper nouns that carry exact-match significance. Elasticsearch provides BM25-based full-text search that excels at these keyword-centric queries. Combining Elasticsearch keyword search with vector similarity search in a hybrid retrieval strategy captures both semantic relevance and keyword precision -- improving recall for RAG applications that handle diverse query types.

Choose Elasticsearch when you need keyword search alongside vector search, when your organization already operates Elasticsearch clusters, or when your data has structured fields that benefit from traditional search features like faceting and aggregations.

## Overview

This integration provides:

- Full-text keyword search using Elasticsearch's `multi_match` queries
- Document indexing with metadata support
- Configurable result limits and field boosting
- OpenTelemetry tracing for production observability
- Compatibility with Beluga AI's `schema.Document` type

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed
- Elasticsearch 8.x instance (local or cloud)
- Elasticsearch Go client v8

## Installation

Install the Elasticsearch Go client:

```bash
go get github.com/elastic/go-elasticsearch/v8
```

Start a local Elasticsearch instance for development:

```bash
docker run -p 9200:9200 -e "discovery.type=single-node" \
  docker.elastic.co/elasticsearch/elasticsearch:8.11.0
```

## Usage

### Creating the Elasticsearch Retriever

Define a retriever that wraps the Elasticsearch client and returns `schema.Document` results:

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/lookatitude/beluga-ai/schema"
)

// ElasticsearchRetriever performs keyword-based document retrieval.
type ElasticsearchRetriever struct {
	client    *elasticsearch.Client
	indexName string
	defaultK  int
}

type ElasticsearchDocument struct {
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

func NewElasticsearchRetriever(esURL, indexName string, defaultK int) (*ElasticsearchRetriever, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{esURL},
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &ElasticsearchRetriever{
		client:    client,
		indexName: indexName,
		defaultK:  defaultK,
	}, nil
}

func (r *ElasticsearchRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query,
				"fields": []string{"content", "title"},
				"type":   "best_fields",
			},
		},
		"size": r.defaultK,
	}

	queryJSON, err := json.Marshal(searchQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	req := esapi.SearchRequest{
		Index: []string{r.indexName},
		Body:  strings.NewReader(string(queryJSON)),
	}

	res, err := req.Do(ctx, r.client)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch error: %s", res.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode: %w", err)
	}

	hits := result["hits"].(map[string]interface{})["hits"].([]interface{})
	documents := make([]schema.Document, 0, len(hits))

	for _, hit := range hits {
		hitMap := hit.(map[string]interface{})
		source := hitMap["_source"].(map[string]interface{})

		content, _ := source["content"].(string)
		metadata, _ := source["metadata"].(map[string]interface{})

		meta := make(map[string]string)
		for k, v := range metadata {
			meta[k] = fmt.Sprintf("%v", v)
		}

		documents = append(documents, schema.NewDocument(content, meta))
	}

	return documents, nil
}
```

### Indexing Documents

Store documents in Elasticsearch for later retrieval:

```go
func (r *ElasticsearchRetriever) IndexDocument(ctx context.Context, doc schema.Document, docID string) error {
	esDoc := ElasticsearchDocument{
		Content:  doc.PageContent,
		Metadata: make(map[string]interface{}),
	}

	for k, v := range doc.Metadata {
		esDoc.Metadata[k] = v
	}

	docJSON, err := json.Marshal(esDoc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	req := esapi.IndexRequest{
		Index:      r.indexName,
		DocumentID: docID,
		Body:       strings.NewReader(string(docJSON)),
	}

	res, err := req.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("index failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch error: %s", res.String())
	}

	return nil
}
```

### End-to-End Example

```go
func main() {
	ctx := context.Background()

	retriever, err := NewElasticsearchRetriever("http://localhost:9200", "documents", 5)
	if err != nil {
		log.Fatalf("Failed to create retriever: %v", err)
	}

	// Index a document
	doc := schema.NewDocument("Machine learning is a subset of AI.",
		map[string]string{"topic": "ml"})
	if err := retriever.IndexDocument(ctx, doc, "doc1"); err != nil {
		log.Fatalf("Failed to index: %v", err)
	}

	// Retrieve documents
	docs, err := retriever.GetRelevantDocuments(ctx, "artificial intelligence")
	if err != nil {
		log.Fatalf("Retrieval failed: %v", err)
	}

	fmt.Printf("Retrieved %d documents\n", len(docs))
	for i, d := range docs {
		fmt.Printf("%d. %s\n", i+1, d.PageContent)
	}
}
```

## Advanced Topics

### Production Retriever with Observability

Add OpenTelemetry tracing and field boosting for production deployments:

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/lookatitude/beluga-ai/schema"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ProductionElasticsearchRetriever struct {
	client    *elasticsearch.Client
	indexName string
	defaultK  int
	tracer    trace.Tracer
}

func NewProductionElasticsearchRetriever(esURL, indexName string, defaultK int) (*ProductionElasticsearchRetriever, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{esURL},
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &ProductionElasticsearchRetriever{
		client:    client,
		indexName: indexName,
		defaultK:  defaultK,
		tracer:    otel.Tracer("beluga.retriever.elasticsearch"),
	}, nil
}

func (r *ProductionElasticsearchRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	ctx, span := r.tracer.Start(ctx, "elasticsearch.search",
		trace.WithAttributes(
			attribute.String("index", r.indexName),
			attribute.String("query", query),
			attribute.Int("k", r.defaultK),
		),
	)
	defer span.End()

	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query,
				"fields": []string{"content^2", "title"},
				"type":   "best_fields",
			},
		},
		"size": r.defaultK,
	}

	queryJSON, err := json.Marshal(searchQuery)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	req := esapi.SearchRequest{
		Index: []string{r.indexName},
		Body:  strings.NewReader(string(queryJSON)),
	}

	res, err := req.Do(ctx, r.client)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("search failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		err := fmt.Errorf("elasticsearch error: %s", res.String())
		span.RecordError(err)
		return nil, err
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to decode: %w", err)
	}

	hits := result["hits"].(map[string]interface{})["hits"].([]interface{})
	documents := make([]schema.Document, 0, len(hits))

	for _, hit := range hits {
		hitMap := hit.(map[string]interface{})
		source := hitMap["_source"].(map[string]interface{})

		content, _ := source["content"].(string)
		metadata, _ := source["metadata"].(map[string]interface{})

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

	retriever, err := NewProductionElasticsearchRetriever("http://localhost:9200", "documents", 5)
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}

	docs, err := retriever.GetRelevantDocuments(ctx, "machine learning")
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}

	fmt.Printf("Retrieved %d documents\n", len(docs))
}
```

## Configuration

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `ESURL` | Elasticsearch server URL | `http://localhost:9200` | No |
| `IndexName` | Index name for documents | `documents` | No |
| `DefaultK` | Number of results to return | `5` | No |
| `Timeout` | Request timeout | `30s` | No |

## Troubleshooting

### Connection refused

Elasticsearch is not running. Start it with Docker:

```bash
docker run -p 9200:9200 -e "discovery.type=single-node" \
  docker.elastic.co/elasticsearch/elasticsearch:8.11.0
```

### Index not found

The target index does not exist yet. Create it before indexing documents:

```bash
curl -X PUT "localhost:9200/documents"
```

## Production Considerations

- **Index design** -- Structure indexes to match your query patterns and field types
- **Sharding** -- Configure shard count based on data volume and query throughput
- **Replication** -- Use replicas for high availability in multi-node clusters
- **Monitoring** -- Track query latency and cluster health via Elasticsearch APIs
- **Security** -- Enable authentication and TLS for production deployments

## Related Resources

- [Weaviate RAG Connector](/integrations/weaviate-rag) -- Hybrid vector and keyword search with Weaviate
- [Vector Stores](/integrations/vector-stores) -- Vector store provider overview
