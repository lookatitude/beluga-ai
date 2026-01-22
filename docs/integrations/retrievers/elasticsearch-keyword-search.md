# Elasticsearch Keyword Search

Welcome, colleague! In this integration guide, we're going to integrate Elasticsearch for keyword-based retrieval with Beluga AI's retrievers package. Elasticsearch provides powerful full-text search capabilities for hybrid RAG systems.

## What you will build

You will configure Beluga AI to use Elasticsearch for keyword-based document retrieval, enabling hybrid search that combines semantic similarity with exact keyword matching for improved RAG accuracy.

## Learning Objectives

- ✅ Configure Elasticsearch with Beluga AI
- ✅ Create Elasticsearch retriever
- ✅ Perform keyword searches
- ✅ Combine with vector search for hybrid retrieval

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Elasticsearch instance (local or cloud)
- Elasticsearch Go client

## Step 1: Setup and Installation

Install Elasticsearch Go client:
bash
```bash
go get github.com/elastic/go-elasticsearch/v8
```

Start Elasticsearch (local):
```bash
docker run -p 9200:9200 -e "discovery.type=single-node" docker.elastic.co/elasticsearch/elasticsearch:8.11.0


## Step 2: Create Elasticsearch Retriever

Create an Elasticsearch-based retriever:
```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"

    "github.com/elastic/go-elasticsearch/v8"
    "github.com/elastic/go-elasticsearch/v8/esapi"
    "github.com/lookatitude/beluga-ai/pkg/core"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

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
    // Build search query
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
    
    queryJSON, _ := json.Marshal(searchQuery)
    
    // Perform search
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
    
    // Parse results
    var result map[string]interface{}
    if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode: %w", err)
    }
    
    // Extract documents
    hits := result["hits"].(map[string]interface{})["hits"].([]interface{})
    documents := make([]schema.Document, 0, len(hits))
    
    for _, hit := range hits {
        hitMap := hit.(map[string]interface{})
        source := hitMap["_source"].(map[string]interface{})
        
        content, _ := source["content"].(string)
        metadata, _ := source["metadata"].(map[string]interface{})
        
        // Convert metadata
        meta := make(map[string]string)
        for k, v := range metadata {
            meta[k] = fmt.Sprintf("%v", v)
        }
        
        documents = append(documents, schema.NewDocument(content, meta))
    }
    
    return documents, nil
}
```

## Step 3: Index Documents

Index documents in Elasticsearch:
```go
func (r *ElasticsearchRetriever) IndexDocument(ctx context.Context, doc schema.Document, docID string) error {
    esDoc := ElasticsearchDocument{
        Content:  doc.PageContent,
        Metadata: make(map[string]interface{}),
    }
    
    for k, v := range doc.Metadata {
        esDoc.Metadata[k] = v
    }
    
    docJSON, _ := json.Marshal(esDoc)
    
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

## Step 4: Use with Beluga AI

Use Elasticsearch retriever in RAG pipeline:
```go
func main() {
    ctx := context.Background()
    
    // Create retriever
    retriever, err := NewElasticsearchRetriever("http://localhost:9200", "documents", 5)
    if err != nil {
        log.Fatalf("Failed to create retriever: %v", err)
    }
    
    // Index documents
    doc := schema.NewDocument("Machine learning is a subset of AI.", 
        map[string]string{"topic": "ml"})
    retriever.IndexDocument(ctx, doc, "doc1")
    
    // Retrieve documents
    docs, err := retriever.GetRelevantDocuments(ctx, "artificial intelligence")
    if err != nil {
        log.Fatalf("Retrieval failed: %v", err)
    }
    
    fmt.Printf("Retrieved %d documents\n", len(docs))
    for i, doc := range docs {
        fmt.Printf("%d. %s\n", i+1, doc.PageContent)
    }
}
```

## Step 5: Complete Integration

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "strings"
    "time"

    "github.com/elastic/go-elasticsearch/v8"
    "github.com/elastic/go-elasticsearch/v8/esapi"
    "github.com/lookatitude/beluga-ai/pkg/core"
    "github.com/lookatitude/beluga-ai/pkg/schema"
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
        tracer:    otel.Tracer("beluga.retrievers.elasticsearch"),
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
    
    queryJSON, _ := json.Marshal(searchQuery)
    
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

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `ESURL` | Elasticsearch URL | `http://localhost:9200` | No |
| `IndexName` | Index name | `documents` | No |
| `DefaultK` | Default number of results | `5` | No |
| `Timeout` | Request timeout | `30s` | No |

## Common Issues

### "Connection refused"

**Problem**: Elasticsearch not running.

**Solution**: Start Elasticsearch:docker run -p 9200:9200 docker.elastic.co/elasticsearch/elasticsearch:8.11.0
```

### "Index not found"

**Problem**: Index doesn't exist.

**Solution**: Create index:curl -X PUT "localhost:9200/documents"
```

## Production Considerations

When using Elasticsearch in production:

- **Index design**: Design indexes for your query patterns
- **Sharding**: Configure appropriate shard count
- **Replication**: Use replicas for high availability
- **Monitoring**: Monitor query performance
- **Security**: Enable authentication and TLS

## Next Steps

Congratulations! You've integrated Elasticsearch with Beluga AI. Next, learn how to:

- **[Weaviate RAG Connector](./weaviate-rag-connector.md)** - Weaviate integration
- **[Retrievers Package Documentation](../../api/packages/retrievers.md)** - Deep dive into retrievers
- **[RAG Tutorial](../../getting-started/02-simple-rag.md)** - Build RAG applications

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
