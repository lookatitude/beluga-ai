---
title: Multi-query Retrieval Chains
description: Improve RAG recall by generating multiple search queries from a single user question using LLM query expansion.
---

Users rarely use the same terminology as your documentation. When a user asks "How do I add a helper?" but your docs describe "middleware implementation," vector search may miss the connection. Multi-query retrieval uses an LLM to generate multiple query variations, capturing different angles and synonyms to improve recall.

## What You Will Build

A multi-query retriever that generates query variations using an LLM, retrieves documents for each variation, and deduplicates the combined results.

## Prerequisites

- Understanding of [Hybrid Search](/tutorials/rag/hybrid-search) and the [ChatModel interface](/guides/llm)
- A configured retriever and LLM provider

## The Problem

User query: "How do I add a helper?"

Generated variations:
1. "How to add a helper function"
2. "Implementing middleware patterns"
3. "Extending functionality with wrappers"

Each variation searches a different part of the semantic space, capturing more relevant documents.

## Step 1: Query Generation

Use an LLM to generate diverse search queries:

```go
package main

import (
    "context"
    "fmt"
    "strings"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

func generateQueries(ctx context.Context, model llm.ChatModel, question string, numQueries int) ([]string, error) {
    msgs := []schema.Message{
        schema.NewSystemMessage(fmt.Sprintf(`Generate %d different search queries for the given question.
Each query should approach the topic from a different angle or use different terminology.
Output one query per line, with no numbering or bullets.`, numQueries)),
        schema.NewHumanMessage(question),
    }

    resp, err := model.Generate(ctx, msgs,
        llm.WithTemperature(0.7),
        llm.WithMaxTokens(200),
    )
    if err != nil {
        return nil, fmt.Errorf("generate queries: %w", err)
    }

    // Split response into individual queries
    lines := strings.Split(resp.Text(), "\n")
    var queries []string
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        if trimmed != "" {
            queries = append(queries, trimmed)
        }
    }

    return queries, nil
}
```

## Step 2: Multi-query Retriever

Retrieve documents for each query variation and deduplicate:

```go
// Retriever searches for relevant documents.
type Retriever interface {
    Retrieve(ctx context.Context, query string, topK int) ([]schema.Document, error)
}

// MultiQueryRetriever expands a single query into multiple variations.
type MultiQueryRetriever struct {
    model       llm.ChatModel
    retriever   Retriever
    numQueries  int
}

func NewMultiQueryRetriever(model llm.ChatModel, retriever Retriever, numQueries int) *MultiQueryRetriever {
    return &MultiQueryRetriever{
        model:      model,
        retriever:  retriever,
        numQueries: numQueries,
    }
}

func (m *MultiQueryRetriever) Retrieve(ctx context.Context, query string, topK int) ([]schema.Document, error) {
    // Generate query variations
    queries, err := generateQueries(ctx, m.model, query, m.numQueries)
    if err != nil {
        return nil, err
    }

    // Include the original query
    queries = append([]string{query}, queries...)

    // Retrieve for each query
    var allDocs []schema.Document
    for _, q := range queries {
        docs, err := m.retriever.Retrieve(ctx, q, topK)
        if err != nil {
            return nil, fmt.Errorf("retrieve for %q: %w", q, err)
        }
        allDocs = append(allDocs, docs...)
    }

    // Deduplicate
    return deduplicateDocs(allDocs, topK), nil
}
```

## Step 3: Deduplication

Remove duplicate documents based on content or ID:

```go
func deduplicateDocs(docs []schema.Document, maxDocs int) []schema.Document {
    seen := make(map[string]bool)
    var unique []schema.Document

    for _, doc := range docs {
        key := doc.ID
        if key == "" {
            key = doc.Content
        }

        if seen[key] {
            continue
        }
        seen[key] = true
        unique = append(unique, doc)

        if len(unique) >= maxDocs {
            break
        }
    }

    return unique
}
```

## Step 4: Parallel Retrieval

For better latency, retrieve for all queries concurrently:

```go
import "sync"

func (m *MultiQueryRetriever) RetrieveParallel(ctx context.Context, query string, topK int) ([]schema.Document, error) {
    queries, err := generateQueries(ctx, m.model, query, m.numQueries)
    if err != nil {
        return nil, err
    }
    queries = append([]string{query}, queries...)

    var mu sync.Mutex
    var allDocs []schema.Document
    var firstErr error
    var wg sync.WaitGroup

    wg.Add(len(queries))
    for _, q := range queries {
        go func(q string) {
            defer wg.Done()
            docs, err := m.retriever.Retrieve(ctx, q, topK)

            mu.Lock()
            defer mu.Unlock()
            if err != nil && firstErr == nil {
                firstErr = err
                return
            }
            allDocs = append(allDocs, docs...)
        }(q)
    }
    wg.Wait()

    if firstErr != nil {
        return nil, firstErr
    }

    return deduplicateDocs(allDocs, topK), nil
}
```

## Step 5: Integration with RAG

```go
func main() {
    ctx := context.Background()

    mqRetriever := NewMultiQueryRetriever(queryModel, baseRetriever, 3)

    // User asks a vague question
    docs, err := mqRetriever.Retrieve(ctx, "How do I add a helper?", 5)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Retrieved %d documents:\n", len(docs))
    for _, doc := range docs {
        fmt.Printf("  - %s\n", doc.Content[:80])
    }

    // Use retrieved docs as context for generation
    var contextStr string
    for _, doc := range docs {
        contextStr += doc.Content + "\n---\n"
    }

    msgs := []schema.Message{
        schema.NewSystemMessage("Answer using the following context:\n" + contextStr),
        schema.NewHumanMessage("How do I add a helper?"),
    }

    resp, err := answerModel.Generate(ctx, msgs)
    if err != nil {
        fmt.Printf("Generate error: %v\n", err)
        return
    }
    fmt.Println(resp.Text())
}
```

## Verification

1. Ask a vague question ("How do I add a helper?").
2. Log the generated query variations — verify they cover different angles.
3. Verify that at least one generated query matches the terminology in your documentation.

## Next Steps

- [Hybrid Search](/tutorials/rag/hybrid-search) — Combine with keyword search for even better recall
- [Fine-tuning Embeddings](/tutorials/providers/finetuning-embeddings) — Optimize the embedding model
