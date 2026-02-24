---
title: Semantic Splitting for Better Embeddings
description: "Create semantically coherent chunks in Go by detecting topic transitions via embedding cosine similarity — produce optimized chunks for RAG retrieval with Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, semantic splitting, embeddings, topic detection, cosine similarity, RAG"
---

Even structural splitting can be arbitrary if a single section covers two distinct topics. A Markdown section titled "Background" might discuss both the problem statement and the historical context -- these are different topics that would produce better embeddings as separate chunks. Semantic splitting uses embedding similarity to detect where one topic ends and another begins, producing chunks that are optimized for retrieval quality because each chunk focuses on a single coherent idea.

## What You Will Build

A semantic splitting pipeline that divides text into sentences, embeds each sentence, calculates cosine similarity between consecutive sentences, and breaks chunks at points where similarity drops below a threshold. This produces chunks where each one focuses on a single coherent topic.

## Prerequisites

- Familiarity with the `rag/splitter` and `rag/embedding` packages
- An embedding provider (OpenAI, Ollama, or similar)

## Core Concepts

### How Semantic Splitting Works

1. Split the text into individual sentences
2. Embed every sentence using an embedding model
3. Calculate cosine similarity between sentence `i` and sentence `i+1`
4. When similarity drops below a threshold, start a new chunk
5. Group consecutive sentences into coherent chunks

The key insight is that consecutive sentences about the same topic produce similar embeddings. When the topic changes, the embedding vectors shift direction, and cosine similarity drops. By detecting these drops, the splitter identifies natural topic boundaries in the text.

### Threshold Tuning

The breakpoint threshold controls chunk granularity. A higher threshold is more sensitive to topic shifts and produces smaller, more specific chunks. A lower threshold is more tolerant and produces larger chunks that may span related subtopics. The right threshold depends on your content type and retrieval requirements.

| Threshold | Effect |
|-----------|--------|
| 0.95 (high) | Many small, highly specific chunks |
| 0.85 (medium) | Balanced chunks for most use cases |
| 0.70 (low) | Fewer, larger chunks with more context |

## Step 1: Set Up the Embedding Model

Create an embedding model via the registry pattern. The `text-embedding-3-small` model is a good default because it balances embedding quality with cost and speed. Semantic splitting requires embedding every sentence, so the embedding cost scales linearly with document length -- a cheaper model keeps the splitting step affordable for large document collections.

```go
package main

import (
    "context"
    "fmt"
    "math"
    "os"
    "strings"

    "github.com/lookatitude/beluga-ai/rag/embedding"

    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
)

func main() {
    ctx := context.Background()

    embedder, err := embedding.New("openai", embedding.ProviderConfig{
        Options: map[string]any{
            "api_key": os.Getenv("OPENAI_API_KEY"),
            "model":   "text-embedding-3-small",
        },
    })
    if err != nil {
        fmt.Printf("embedder creation failed: %v\n", err)
        return
    }

    _ = embedder
    _ = ctx
}
```

## Step 2: Implement Sentence Splitting

Break text into individual sentences for embedding. Sentence-level granularity is the right unit for semantic splitting because sentences are the smallest unit that typically carries a complete idea. Splitting at the word or phrase level would produce embeddings that are too noisy for reliable similarity comparison.

```go
// splitSentences breaks text into sentences using basic heuristics.
func splitSentences(text string) []string {
    // Simple sentence splitting. For production, use a more robust approach.
    separators := []string{". ", "! ", "? ", ".\n", "!\n", "?\n"}

    sentences := []string{text}
    for _, sep := range separators {
        var newSentences []string
        for _, s := range sentences {
            parts := strings.Split(s, sep)
            for i, part := range parts {
                trimmed := strings.TrimSpace(part)
                if trimmed == "" {
                    continue
                }
                // Re-add the punctuation except for the last part.
                if i < len(parts)-1 {
                    trimmed += string(sep[0])
                }
                newSentences = append(newSentences, trimmed)
            }
        }
        sentences = newSentences
    }

    return sentences
}
```

## Step 3: Calculate Cosine Similarity

Cosine similarity measures the angle between two embedding vectors, producing a value between -1 and 1. A value near 1 means the vectors point in the same direction (similar meaning), while a value near 0 means they are orthogonal (unrelated topics). This metric is preferred over Euclidean distance for comparing embeddings because it is invariant to vector magnitude -- two sentences about the same topic will have high cosine similarity regardless of their length.

```go
// cosineSimilarity computes the cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float64 {
    if len(a) != len(b) {
        return 0
    }

    var dotProduct, normA, normB float64
    for i := range a {
        dotProduct += float64(a[i]) * float64(b[i])
        normA += float64(a[i]) * float64(a[i])
        normB += float64(b[i]) * float64(b[i])
    }

    if normA == 0 || normB == 0 {
        return 0
    }

    return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
```

## Step 4: Build the Semantic Splitter

Combine embedding and similarity to detect topic boundaries. The splitter embeds all sentences in a single batch call (`EmbedStrings`) rather than one at a time, which is more efficient because embedding APIs support batch input and amortize the network round-trip cost. The loop compares each consecutive pair of sentence embeddings, and when similarity drops below the threshold, a new chunk begins. This linear scan is O(n) in the number of sentences, making it efficient even for long documents.

```go
// SemanticSplitter splits text based on embedding similarity between sentences.
type SemanticSplitter struct {
    embedder  embedding.Embedder
    threshold float64
}

func NewSemanticSplitter(embedder embedding.Embedder, threshold float64) *SemanticSplitter {
    return &SemanticSplitter{
        embedder:  embedder,
        threshold: threshold,
    }
}

func (s *SemanticSplitter) Split(ctx context.Context, text string) ([]string, error) {
    sentences := splitSentences(text)
    if len(sentences) <= 1 {
        return sentences, nil
    }

    // Embed all sentences.
    embeddings, err := s.embedder.EmbedStrings(ctx, sentences)
    if err != nil {
        return nil, fmt.Errorf("embed sentences: %w", err)
    }

    // Find breakpoints where similarity drops below threshold.
    var chunks []string
    var currentChunk []string
    currentChunk = append(currentChunk, sentences[0])

    for i := 1; i < len(sentences); i++ {
        similarity := cosineSimilarity(embeddings[i-1], embeddings[i])

        if similarity < s.threshold {
            // Topic change detected -- start a new chunk.
            chunks = append(chunks, strings.Join(currentChunk, " "))
            currentChunk = nil
        }
        currentChunk = append(currentChunk, sentences[i])
    }

    // Flush the last chunk.
    if len(currentChunk) > 0 {
        chunks = append(chunks, strings.Join(currentChunk, " "))
    }

    return chunks, nil
}
```

## Step 5: Use the Semantic Splitter

The example text below transitions from astronomy to zoology mid-paragraph. A character-based splitter would not detect this boundary, but the semantic splitter identifies the topic change because the embedding vectors for "sun" sentences are dissimilar from "cats" sentences.

```go
func demonstrateSemanticSplit(ctx context.Context, embedder embedding.Embedder) {
    splitter := NewSemanticSplitter(embedder, 0.85)

    text := `The sun is a star located at the center of the Solar System. ` +
        `It provides light and heat to all the planets. ` +
        `The sun is approximately 4.6 billion years old. ` +
        `Cats are feline mammals commonly kept as pets. ` +
        `They are known for their independence and agility. ` +
        `Domestic cats have been companions to humans for thousands of years.`

    chunks, err := splitter.Split(ctx, text)
    if err != nil {
        fmt.Printf("split error: %v\n", err)
        return
    }

    for i, chunk := range chunks {
        fmt.Printf("--- Chunk %d ---\n%s\n\n", i, chunk)
    }

    // Expected: two chunks, one about the sun, one about cats.
}
```

## Step 6: Combine Structural and Semantic Splitting

For best results, apply Markdown splitting first (for structure), then semantic splitting for any oversized sections. This two-pass approach uses each method's strengths: structural splitting respects document organization (headings, sections), while semantic splitting handles cases where a single section covers multiple topics. The generous initial chunk size (3000 characters) ensures the Markdown splitter preserves large sections intact, leaving the semantic splitter to subdivide only where topic changes are detected.

```go
import "github.com/lookatitude/beluga-ai/rag/splitter"

func hybridSplit(ctx context.Context, text string, embedder embedding.Embedder) ([]string, error) {
    // First pass: structural splitting by Markdown headings.
    mdSplitter := splitter.NewMarkdownSplitter(
        splitter.WithMarkdownChunkSize(3000), // Generous initial size.
        splitter.WithPreserveHeaders(true),
    )

    structuralChunks, err := mdSplitter.Split(ctx, text)
    if err != nil {
        return nil, fmt.Errorf("markdown split: %w", err)
    }

    // Second pass: semantic splitting on large chunks.
    semanticSplitter := NewSemanticSplitter(embedder, 0.85)

    var finalChunks []string
    for _, chunk := range structuralChunks {
        if len(chunk) > 1000 {
            subChunks, err := semanticSplitter.Split(ctx, chunk)
            if err != nil {
                return nil, fmt.Errorf("semantic split: %w", err)
            }
            finalChunks = append(finalChunks, subChunks...)
        } else {
            finalChunks = append(finalChunks, chunk)
        }
    }

    return finalChunks, nil
}
```

## Tuning the Threshold

Finding the right threshold depends on your content and embedding model. Start with these guidelines:

- **Technical documentation**: 0.80-0.85 (topics shift clearly between sections)
- **Narrative text**: 0.85-0.90 (topics shift gradually)
- **Conversational content**: 0.75-0.80 (topics shift frequently)

Evaluate by inspecting chunk boundaries and measuring retrieval Hit Rate.

## Verification

1. Run the semantic splitter on a document that clearly changes topics.
2. Verify that chunks do not contain mixed topics.
3. Adjust the threshold and observe how chunk boundaries change.
4. Compare retrieval accuracy (Hit Rate) between semantic splitting and fixed-size splitting.

## Next Steps

- [Markdown Chunking](/docs/tutorials/documents/markdown-chunking) -- Combine structural and semantic splitting strategies
- [Lazy-Loading Documents](/docs/tutorials/documents/lazy-loading) -- Process large document collections efficiently
