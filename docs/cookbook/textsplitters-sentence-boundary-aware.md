---
title: "Sentence-boundary Aware"
package: "textsplitters"
category: "text-processing"
complexity: "intermediate"
---

# Sentence-boundary Aware

## Problem

You need to split text into chunks while preserving sentence boundaries, avoiding splits in the middle of sentences which can lose context and create confusing chunks.

## Solution

Implement sentence-aware splitting that detects sentence boundaries using NLP techniques or regex patterns, splits only at sentence boundaries, and merges sentences into appropriately sized chunks. This works because sentences are natural semantic units, and splitting between them preserves meaning.

## Code Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "regexp"
    "strings"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
    
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

var tracer = otel.Tracer("beluga.textsplitters.sentence_boundary")

// SentenceBoundarySplitter splits text at sentence boundaries
type SentenceBoundarySplitter struct {
    chunkSize    int
    chunkOverlap int
    sentenceEnd  *regexp.Regexp
}

// NewSentenceBoundarySplitter creates a new splitter
func NewSentenceBoundarySplitter(chunkSize, chunkOverlap int) *SentenceBoundarySplitter {
    // Pattern to match sentence endings
    sentenceEnd := regexp.MustCompile(`[.!?]+[\s\n]+`)

    return &SentenceBoundarySplitter{
        chunkSize:    chunkSize,
        chunkOverlap: chunkOverlap,
        sentenceEnd:  sentenceEnd,
    }
}

// SplitText splits text at sentence boundaries
func (sbs *SentenceBoundarySplitter) SplitText(ctx context.Context, text string) ([]string, error) {
    ctx, span := tracer.Start(ctx, "sentence_splitter.split")
    defer span.End()
    
    span.SetAttributes(
        attribute.Int("text_length", len(text)),
        attribute.Int("chunk_size", sbs.chunkSize),
    )
    
    // Split into sentences
    sentences := sbs.splitIntoSentences(ctx, text)
    
    span.SetAttributes(attribute.Int("sentence_count", len(sentences)))
    
    // Merge sentences into chunks
    chunks := sbs.mergeSentences(ctx, sentences)
    
    span.SetAttributes(attribute.Int("chunk_count", len(chunks)))
    span.SetStatus(trace.StatusOK, "text split at sentence boundaries")
    
    return chunks, nil
}

// splitIntoSentences splits text into sentences
func (sbs *SentenceBoundarySplitter) splitIntoSentences(ctx context.Context, text string) []string {
    // Find all sentence boundaries
    indices := sbs.sentenceEnd.FindAllStringIndex(text, -1)

    if len(indices) == 0 {
        // No sentence boundaries found, return whole text
        return []string{text}
    }
    
    sentences := []string{}
    start := 0
    
    for _, match := range indices {
        end := match[1]
        sentence := strings.TrimSpace(text[start:end])
        if sentence != "" {
            sentences = append(sentences, sentence)
        }
        start = end
    }
    
    // Add remaining text
    if start < len(text) {
        remaining := strings.TrimSpace(text[start:])
        if remaining != "" {
            sentences = append(sentences, remaining)
        }
    }
    
    return sentences
}

// mergeSentences merges sentences into chunks
func (sbs *SentenceBoundarySplitter) mergeSentences(ctx context.Context, sentences []string) []string {
    chunks := []string{}
    currentChunk := ""

    for _, sentence := range sentences {
        // Calculate size if we add this sentence
        potentialChunk := currentChunk
        if potentialChunk != "" {
            potentialChunk += " "
        }
        potentialChunk += sentence
        
        // If adding would exceed chunk size, finalize current chunk
        if len(potentialChunk) > sbs.chunkSize && currentChunk != "" {
            chunks = append(chunks, currentChunk)
            
            // Start new chunk with overlap
            if sbs.chunkOverlap > 0 {
                // Take last sentences for overlap
                overlapSentences := sbs.getOverlapSentences(currentChunk, sbs.chunkOverlap)
                if len(overlapSentences) > 0 {
                    currentChunk = strings.Join(overlapSentences, " ")
                } else {
                    currentChunk = ""
                }
            } else {
                currentChunk = ""
            }
            
            // Add current sentence to new chunk
            if currentChunk != "" {
                currentChunk += " "
            }
            currentChunk += sentence
        } else {
            // Add sentence to current chunk
            currentChunk = potentialChunk
        }
    }
    
    // Add final chunk
    if currentChunk != "" {
        chunks = append(chunks, currentChunk)
    }
    
    return chunks
}

// getOverlapSentences gets last sentences for overlap
func (sbs *SentenceBoundarySplitter) getOverlapSentences(text string, overlapSize int) []string {
    // Split text back into sentences
    sentences := sbs.sentenceEnd.Split(text, -1)

    // Take last sentences that fit in overlap size
    overlapSentences := []string{}
    currentSize := 0
    
    for i := len(sentences) - 1; i >= 0; i-- {
        sentence := strings.TrimSpace(sentences[i])
        if sentence == "" {
            continue
        }
        
        if currentSize+len(sentence) > overlapSize {
            break
        }
        
        overlapSentences = append([]string{sentence}, overlapSentences...)
        currentSize += len(sentence) + 1 // +1 for space
    }
    
    return overlapSentences
}

// SplitDocuments splits documents preserving sentence boundaries
func (sbs *SentenceBoundarySplitter) SplitDocuments(ctx context.Context, documents []schema.Document) ([]schema.Document, error) {
    ctx, span := tracer.Start(ctx, "sentence_splitter.split_documents")
    defer span.End()
    
    resultDocs := []schema.Document{}
    
    for _, doc := range documents {
        chunks, err := sbs.SplitText(ctx, doc.GetContent())
        if err != nil {
            span.RecordError(err)
            continue
        }
        
        // Create documents from chunks
        for i, chunk := range chunks {
            chunkDoc := schema.NewDocument(chunk, map[string]string{
                "source":    doc.GetMetadata()["source"],
                "chunk_index": fmt.Sprintf("%d", i),
            })
            resultDocs = append(resultDocs, chunkDoc)
        }
    }
    
    span.SetAttributes(attribute.Int("output_document_count", len(resultDocs)))
    span.SetStatus(trace.StatusOK, "documents split")
    
    return resultDocs, nil
}

func main() {
    ctx := context.Background()

    // Create splitter
    splitter := NewSentenceBoundarySplitter(500, 100)
    
    // Split text
    text := "This is sentence one. This is sentence two! Is this sentence three? This is sentence four."
    
    chunks, err := splitter.SplitText(ctx, text)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    
    fmt.Printf("Split into %d chunks\n", len(chunks))
    for i, chunk := range chunks {
        fmt.Printf("Chunk %d: %s\n", i+1, chunk)
    }
}
```

## Explanation

Let's break down what's happening:

1. **Sentence detection** - Notice how we use regex to detect sentence boundaries (periods, exclamation marks, question marks followed by whitespace). This identifies natural break points.

2. **Boundary preservation** - We only split at detected sentence boundaries, never in the middle of sentences. This preserves semantic meaning.

3. **Overlap handling** - When creating overlap between chunks, we use complete sentences rather than partial text. This maintains coherence in overlapping regions.

```go
**Key insight:** Always split at sentence boundaries. Mid-sentence splits lose context and create confusing chunks that are hard for RAG systems to use effectively.

## Testing

```
Here's how to test this solution:
```go
func TestSentenceBoundarySplitter_PreservesSentences(t *testing.T) {
    splitter := NewSentenceBoundarySplitter(50, 10)
    
    text := "Sentence one. Sentence two. Sentence three."
    chunks, err := splitter.SplitText(context.Background(), text)
    
    require.NoError(t, err)
    // Verify no chunks split mid-sentence
    for _, chunk := range chunks {
        require.NotContains(t, chunk, "Sentence on")
    }
}

## Variations

### Language-specific Patterns

Use language-specific sentence patterns:
type LanguageAwareSplitter struct {
    language string
    patterns map[string]*regexp.Regexp
}
```

### Abbreviation Handling

Handle abbreviations correctly:
```go
func (sbs *SentenceBoundarySplitter) isAbbreviation(word string) bool {
    // Check if word is abbreviation (e.g., "Mr.", "Dr.")
}
```

## Related Recipes

- **[Textsplitters Advanced Code Splitting](./textsplitters-advanced-code-splitting-tree-sitter.md)** - Code-aware splitting
- **[Document Ingestion Recipes](./document-ingestion-recipes.md)** - Document loading patterns
- **[Textsplitters Package Guide](../package_design_patterns.md)** - For a deeper understanding of text splitting
