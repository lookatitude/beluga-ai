---
title: Cross-lingual Document Retrieval
description: Enable users to search documents in any language and retrieve relevant results across all languages.
---

A global knowledge management platform needed to enable users to search documents in any language and retrieve relevant results regardless of the document's original language. The conventional approach — translating all documents into a common language or translating every query into every document language — is expensive, lossy (translations miss nuance), and operationally complex (every new language multiplies the translation workload).

Users could only search documents in their own language, causing 30-40% of relevant multilingual content to be missed and reducing knowledge discovery across language barriers. A cross-lingual retrieval system with multilingual embeddings enables search in any language to find documents in all languages with 90%+ relevance. The key insight is using a multilingual embedding model that maps text from different languages into a unified semantic space — a Spanish query and an English document about the same topic produce similar vectors, enabling cross-lingual similarity search without any translation step.

## Solution Architecture

Beluga AI's RAG embedding package supports multilingual embedding models that map text from different languages into a unified semantic space. The retrieval system generates language-agnostic embeddings for both documents and queries, enabling cross-lingual similarity search without translation.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ Multilingual │───▶│  Multilingual│───▶│    Vector    │
│  Documents   │    │   Embedder   │    │    Store     │
│              │    │              │    │              │
└──────────────┘    └──────────────┘    └──────────────┘
                                               │
                                               ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  Search      │◀───│   Ranked     │◀───│Cross-lingual │
│  Results     │    │   Results    │    │    Search    │
│              │    │              │    │              │
└──────────────┘    └──────────────┘    └──────────────┘
```

## Cross-lingual Retrieval System

The system uses multilingual embeddings for language-agnostic search.

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    "github.com/lookatitude/beluga-ai/rag/retriever"
    "github.com/lookatitude/beluga-ai/schema"
)

// CrossLingualRetrievalSystem implements cross-lingual document retrieval.
type CrossLingualRetrievalSystem struct {
    embedder     embedding.Embedder
    vectorStore  vectorstore.VectorStore
    langDetector *LanguageDetector
}

// NewCrossLingualRetrievalSystem creates a new cross-lingual retrieval system.
func NewCrossLingualRetrievalSystem(ctx context.Context) (*CrossLingualRetrievalSystem, error) {
    // Use multilingual embedding model
    emb, err := embedding.New("openai", map[string]any{
        "model": "text-embedding-3-large",
    })
    if err != nil {
        return nil, fmt.Errorf("create embedder: %w", err)
    }

    store, err := vectorstore.New("pgvector", map[string]any{
        "connection_string": "postgresql://localhost/vectordb",
    })
    if err != nil {
        return nil, fmt.Errorf("create vector store: %w", err)
    }

    return &CrossLingualRetrievalSystem{
        embedder:     emb,
        vectorStore:  store,
        langDetector: NewLanguageDetector(),
    }, nil
}
```

## Document Indexing

Index documents with language metadata for cross-lingual search.

```go
// IndexDocument indexes a multilingual document.
func (s *CrossLingualRetrievalSystem) IndexDocument(
    ctx context.Context,
    docID string,
    content string,
    language string,
    metadata map[string]string,
) error {
    // Generate language-agnostic embedding
    embeddings, err := s.embedder.Embed(ctx, []string{content})
    if err != nil {
        return fmt.Errorf("generate embedding: %w", err)
    }

    // Create document with language metadata
    doc := schema.Document{
        Content: content,
        Metadata: map[string]any{
            "doc_id":   docID,
            "language": language,
        },
    }

    // Add user metadata
    for k, v := range metadata {
        doc.Metadata[k] = v
    }

    // Store in vector store
    if err := s.vectorStore.Add(ctx, []schema.Document{doc}, embeddings); err != nil {
        return fmt.Errorf("store document: %w", err)
    }

    return nil
}

// IndexBatch indexes multiple documents efficiently.
func (s *CrossLingualRetrievalSystem) IndexBatch(ctx context.Context, documents []MultilingualDocument) error {
    texts := make([]string, len(documents))
    docs := make([]schema.Document, len(documents))

    for i, doc := range documents {
        texts[i] = doc.Content
        docs[i] = schema.Document{
            Content: doc.Content,
            Metadata: map[string]any{
                "doc_id":   doc.ID,
                "language": doc.Language,
            },
        }
    }

    // Generate embeddings in batch
    embeddings, err := s.embedder.Embed(ctx, texts)
    if err != nil {
        return fmt.Errorf("generate embeddings: %w", err)
    }

    // Store documents
    if err := s.vectorStore.Add(ctx, docs, embeddings); err != nil {
        return fmt.Errorf("store documents: %w", err)
    }

    return nil
}

type MultilingualDocument struct {
    ID       string
    Content  string
    Language string
}
```

## Cross-lingual Search

Search across all languages using multilingual embeddings.

```go
type SearchResult struct {
    DocID    string
    Content  string
    Language string
    Score    float64
}

// Search searches across all languages.
func (s *CrossLingualRetrievalSystem) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
    // Detect query language
    queryLang, err := s.langDetector.Detect(ctx, query)
    if err != nil {
        queryLang = "unknown"
    }

    // Generate query embedding (same space as documents)
    embeddings, err := s.embedder.Embed(ctx, []string{query})
    if err != nil {
        return nil, fmt.Errorf("generate query embedding: %w", err)
    }

    // Create retriever
    ret, err := retriever.New("basic", map[string]any{
        "vectorstore": s.vectorStore,
        "k":           limit * 2, // Get more for ranking
    })
    if err != nil {
        return nil, fmt.Errorf("create retriever: %w", err)
    }

    // Search vector store (language-agnostic)
    docs, err := ret.Retrieve(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("similarity search: %w", err)
    }

    // Convert to search results
    results := make([]SearchResult, 0, len(docs))
    for _, doc := range docs {
        results = append(results, SearchResult{
            DocID:    doc.Metadata["doc_id"].(string),
            Content:  doc.Content,
            Language: doc.Metadata["language"].(string),
            Score:    doc.Metadata["score"].(float64),
        })
    }

    // Rank by cross-lingual relevance
    ranked := s.rankCrossLingualResults(results, queryLang)

    // Return top results
    if len(ranked) > limit {
        ranked = ranked[:limit]
    }

    return ranked, nil
}
```

## Cross-lingual Ranking

Rank results considering cross-lingual relevance.

```go
func (s *CrossLingualRetrievalSystem) rankCrossLingualResults(results []SearchResult, queryLang string) []SearchResult {
    // Sort by score (descending)
    sort.Slice(results, func(i, j int) bool {
        // Small boost for same-language results
        scoreI := results[i].Score
        scoreJ := results[j].Score

        if results[i].Language == queryLang {
            scoreI *= 1.05
        }
        if results[j].Language == queryLang {
            scoreJ *= 1.05
        }

        return scoreI > scoreJ
    })

    return results
}
```

## Language Detection

Detect query language for optimization.

```go
type LanguageDetector struct {
    // Language detection implementation
}

func NewLanguageDetector() *LanguageDetector {
    return &LanguageDetector{}
}

// Detect determines the language of text.
func (ld *LanguageDetector) Detect(ctx context.Context, text string) (string, error) {
    // Use a language detection library or service
    // This is a simplified example

    // Check for common language indicators
    if containsChinese(text) {
        return "zh", nil
    }
    if containsJapanese(text) {
        return "ja", nil
    }
    if containsArabic(text) {
        return "ar", nil
    }

    // Default to English for simplicity
    return "en", nil
}

func containsChinese(text string) bool {
    for _, r := range text {
        if r >= 0x4E00 && r <= 0x9FFF {
            return true
        }
    }
    return false
}

func containsJapanese(text string) bool {
    for _, r := range text {
        if (r >= 0x3040 && r <= 0x309F) || (r >= 0x30A0 && r <= 0x30FF) {
            return true
        }
    }
    return false
}

func containsArabic(text string) bool {
    for _, r := range text {
        if r >= 0x0600 && r <= 0x06FF {
            return true
        }
    }
    return false
}
```

## Production Considerations

### Multilingual Model Selection

Choose embedding models specifically trained for multilingual tasks. Models like OpenAI's text-embedding-3-large, Cohere's multilingual-22 embedding, or open-source models like multilingual-e5-large provide strong cross-lingual performance. Benchmark models on your language pairs.

### Embedding Quality

Cross-lingual embedding quality varies by language pair. Test retrieval accuracy across your supported languages. Some language pairs (e.g., English-Spanish) typically have higher similarity than distant pairs (e.g., English-Japanese).

### Language Detection Accuracy

Automatic language detection improves user experience but can be inaccurate for short queries or code-switched text. Consider allowing users to specify their query language explicitly as a fallback.

### Ranking Strategy

The small same-language boost (5%) balances cross-lingual retrieval with same-language preference. Tune this parameter based on user feedback. Some applications may want no language preference.

### Observability

Track cross-lingual search metrics:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

func (s *CrossLingualRetrievalSystem) recordMetrics(ctx context.Context, queryLang string, resultLanguages []string) {
    meter := otel.Meter("cross-lingual-retrieval")

    counter, _ := meter.Int64Counter("cross_lingual_searches_total")
    counter.Add(ctx, 1,
        metric.WithAttributes(
            attribute.String("query_language", queryLang),
        ),
    )

    // Track language diversity in results
    uniqueLangs := make(map[string]bool)
    for _, lang := range resultLanguages {
        uniqueLangs[lang] = true
    }

    histogram, _ := meter.Int64Histogram("result_language_diversity")
    histogram.Record(ctx, int64(len(uniqueLangs)),
        metric.WithAttributes(
            attribute.String("query_language", queryLang),
        ),
    )
}
```

### Scaling

For large multilingual document collections, consider language-specific sharding in the vector store. This can improve query performance while maintaining cross-lingual capabilities. Use approximate nearest neighbor search for sub-second retrieval at scale.

## Results

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Cross-lingual Relevance | 60-70% | 92% | 31-53% improvement |
| Content Discovery Rate | 60-70% | 91% | 30-52% improvement |
| Search Efficiency (searches/query) | 2-3 | 1 | 50-67% reduction |
| User Satisfaction Score | 6.5/10 | 9.2/10 | 42% improvement |
| Multilingual Content Usage | 40% | 87% | 118% increase |

## Related Resources

- [RAG Pipeline Guide](/guides/rag-pipeline/) for retrieval patterns
- [Embedding Package Guide](/guides/embeddings/) for embedding strategies
- [Enterprise RAG](/use-cases/enterprise-rag/) for complete RAG system setup
