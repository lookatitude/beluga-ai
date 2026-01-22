# Local Development with In-Memory Vector Stores

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this guide, we'll use `pkg/vectorstores` to create a transient, localized search engine that lives entirely in RAM. It's fast, free, and requires zero setup for your RAG (Retrieval-Augmented Generation) prototyping.

## Learning Objectives
By the end of this tutorial, you will:
1.  Initialize an `InMemoryStore`.
2.  Index a small dataset of "facts".
3.  Perform semantic search queries.

## Introduction
Welcome, colleague! So, you want to build a RAG app. You've got your PDFs parsed, your embeddings generated... but where do you put them?

Spinning up a PostgreSQL instance with `pgvector` or signing up for a hosted Pinecone index is great for production, but it destroys your "flow" when you just want to test an idea.

Enter the **In-Memory Vector Store**.

## Why This Matters

*   **Rapid Prototyping**: Test your retrieval logic in unit tests without mocking a database.
*   **CI/CD Friendly**: Run your entire RAG pipeline in a GitHub Action.
*   **Zero Dependencies**: No Docker, no API keys, no network calls.

## Prerequisites

*   A working Go environment.
*   An embedding model (we'll use a mocked one for this guide to keep it zero-latency, but you can swap it for OpenAI/Google).

## Concepts

### The Vector Store
A Vector Store is just a specialized database that stores `[]float32` arrays (vectors) along with the original text.

### Cosine Similarity
To find "similar" things, we calculate the angle between two vectors. Small angle = high similarity. The `InMemoryStore` does this using brute-force math (simulating what a real DB does with an index).

## Step-by-Step Implementation

### Step 1: The Setup

Let's set up our main function. We'll need a mock embedder to turn text into numbers.
```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

// MockEmbedder is a dummy implementation for demonstration
type MockEmbedder struct {}

func (m *MockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
    // In a real app, this calls an API.
    // Here, we just return random-ish vectors for demo purposes.
    // Pro-Tip: Real embeddings usually have 1536 dimensions (OpenAI) or 768 (Google).
    dims := 3 
    vectors := make([][]float32, len(texts))
    for i := range texts {
        vectors[i] = []float32{0.1, 0.2, 0.3} // Simplified!
    }
    return vectors, nil
}

func (m *MockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
    return []float32{0.1, 0.2, 0.3}, nil
}
```

### Step 2: Initializing the Store

Now, let's create the store. Notice how simple the initialization is compared to a database connection.
```go
func main() {
    ctx := context.Background()
    embedder := &MockEmbedder{}

    // Create the in-memory store
    // We inject the embedder so the store knows how to vectorize queries later
    store, err := vectorstores.NewInMemoryStore(ctx,
        vectorstores.WithEmbedder(embedder),
    )
    if err != nil {
        log.Fatal(err)
    }

    
    fmt.Println("Vector Store initialized in RAM!")
}
```

### Step 3: Adding Documents

We wrap our raw text in `schema.Document` structs. This allows us to attach metadata (like source filenames or page numbers).
```go
    // Our knowledge base
    facts := []string{
        "Beluga whales are white and live in the Arctic.",
        "The beluga sturgeon is unrelated to the whale.",
        "Golang is a statically typed programming language.",
    }

    // Convert to Documents
    var docs []schema.Document
    for _, f := range facts {
        docs = append(docs, schema.NewDocument(f, map[string]string{
            "source": "encyclopedia",
        }))
    }

    // Index them
    // This calls EmbedDocuments() behind the scenes
    ids, err := store.AddDocuments(ctx, docs)
    if err != nil {
        log.Fatal(err)
    }

    
    fmt.Printf("Indexed %d documents\n", len(ids))
```

### Step 4: Searching

Now for the payoff. Let's find documents relevant to "programming".
```go
    query := "coding languages"
    
    // Search!
    // k=1 means "give me the single best match"
    results, scores, err := store.SimilaritySearchByQuery(ctx, query, 1, embedder)
    if err != nil {
        log.Fatal(err)
    }


    if len(results) > 0 \{
        fmt.Printf("Query: %s\n", query)
        fmt.Printf("Top Match: %s\n", results[0].PageContent)
        fmt.Printf("Score: %.4f\n", scores[0])
    }
```

*Note: Since our MockEmbedder returns identical vectors, the search results won't be semantically accurate in this snippet. Swap `MockEmbedder` with `openai.NewEmbedder(...)` to see real magic!*

## Pro-Tips

*   **Filter by Metadata**: `InMemoryStore` supports filtering! You can pass a filter function to narrow down results *before* the vector comparison.

    // Hypothetical filter usage (interface dependent)
    // store.SimilaritySearch(..., vectorstores.WithFilter(func(meta map[string]any) bool \{
    //     return meta["source"] == "encyclopedia"
    // }))
*   **Dump to Disk**: While it is "in-memory", you can serialize the underlying map to JSON and save it to a file. This creates a "poor man's generic database" that you can reload on the next startup.

## Troubleshooting

### "Results are garbage"
If you consistently get irrelevant results, check your **Embedder**. The garbage-in, garbage-out rule applies. If your embeddings are random (like in our mock above), your search results will be random.

### "Out of Memory"
The `InMemoryStore` holds *everything* in a Go slice. If you try to index 1 million Wikipedia articles, your process will crash. For datasets larger than ~50k documents, switch to `pkg/vectorstores/pgvector`.

## Conclusion

You now have a functional search engine running inside your capabilities. This is the fastest way to validate RAG workflows. Once your logic works here, changing to `NewPgVectorStore` is literally a one-line code change.
