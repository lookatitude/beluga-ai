---
title: "Sentence-Boundary Splitting"
description: "Recipe for splitting text at sentence boundaries in Go to produce higher-quality embeddings and more coherent RAG retrieval results with Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, sentence splitting, Go text splitter, RAG chunking, sentence boundary, embedding quality, TextSplitter recipe"
---

## Problem

You need to split text into chunks while preserving sentence boundaries, avoiding splits in the middle of sentences which lose context and create confusing chunks for RAG retrieval.

Each chunk in a RAG pipeline becomes an independent unit for embedding and retrieval. When a chunk starts or ends mid-sentence, the embedding captures an incomplete thought. This degrades retrieval quality because the chunk's semantic representation is muddled, and it degrades generation quality because the LLM receives partial sentences that are harder to interpret. Sentences are natural semantic units -- they express complete ideas -- making them the ideal split boundary for prose text.

## Solution

Implement sentence-aware splitting that detects sentence boundaries using regex patterns, splits only at those boundaries, and merges sentences into appropriately sized chunks. The splitting process first identifies all sentence boundaries, then greedily groups consecutive sentences into chunks that stay within the size limit. This produces chunks where every sentence is complete, maximizing semantic coherence per chunk.

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
)

var tracer = otel.Tracer("beluga.textsplitters.sentence_boundary")

// SentenceBoundarySplitter splits text at sentence boundaries.
type SentenceBoundarySplitter struct {
	chunkSize    int
	chunkOverlap int
	sentenceEnd  *regexp.Regexp
}

func NewSentenceBoundarySplitter(chunkSize, chunkOverlap int) *SentenceBoundarySplitter {
	return &SentenceBoundarySplitter{
		chunkSize:    chunkSize,
		chunkOverlap: chunkOverlap,
		sentenceEnd:  regexp.MustCompile(`[.!?]+[\s\n]+`),
	}
}

// SplitText splits text at sentence boundaries.
func (sbs *SentenceBoundarySplitter) SplitText(ctx context.Context, text string) ([]string, error) {
	ctx, span := tracer.Start(ctx, "sentence_splitter.split")
	defer span.End()

	span.SetAttributes(
		attribute.Int("text_length", len(text)),
		attribute.Int("chunk_size", sbs.chunkSize),
	)

	sentences := sbs.splitIntoSentences(text)
	span.SetAttributes(attribute.Int("sentence_count", len(sentences)))

	chunks := sbs.mergeSentences(sentences)
	span.SetAttributes(attribute.Int("chunk_count", len(chunks)))
	span.SetStatus(trace.StatusOK, "text split at sentence boundaries")
	return chunks, nil
}

func (sbs *SentenceBoundarySplitter) splitIntoSentences(text string) []string {
	indices := sbs.sentenceEnd.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
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

	if start < len(text) {
		remaining := strings.TrimSpace(text[start:])
		if remaining != "" {
			sentences = append(sentences, remaining)
		}
	}
	return sentences
}

func (sbs *SentenceBoundarySplitter) mergeSentences(sentences []string) []string {
	chunks := []string{}
	currentChunk := ""

	for _, sentence := range sentences {
		potential := currentChunk
		if potential != "" {
			potential += " "
		}
		potential += sentence

		if len(potential) > sbs.chunkSize && currentChunk != "" {
			chunks = append(chunks, currentChunk)
			currentChunk = sentence
		} else {
			currentChunk = potential
		}
	}

	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}
	return chunks
}

func main() {
	ctx := context.Background()

	splitter := NewSentenceBoundarySplitter(500, 100)
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

1. **Sentence detection** -- A regex pattern matches sentence endings (periods, exclamation marks, question marks followed by whitespace). This identifies natural break points in the text where one complete thought ends and another begins. The pattern requires trailing whitespace to avoid matching abbreviations like "Dr." or "U.S." in most cases.

2. **Boundary preservation** -- Text is split only at detected sentence boundaries, never in the middle of sentences. This ensures each chunk contains complete thoughts, producing embeddings that capture coherent semantics rather than fragments.

3. **Size-aware merging** -- Sentences are greedily merged into chunks that respect the configured size limit. Short sentences are grouped together to avoid creating too many tiny chunks; long sentences may stand alone if they exceed the overlap threshold. This balances chunk count against chunk quality.

## Variations

### Abbreviation Handling

Handle abbreviations like "Dr.", "Mr.", "U.S." that contain periods but are not sentence endings:

```go
var abbreviations = map[string]bool{
	"dr.": true, "mr.": true, "mrs.": true, "ms.": true,
	"u.s.": true, "e.g.": true, "i.e.": true,
}

func isAbbreviation(word string) bool {
	return abbreviations[strings.ToLower(word)]
}
```

## Related Recipes

- **[Code Splitting with Tree-sitter](./code-splitting)** -- Structure-aware code splitting
- **[Document Ingestion](./document-ingestion)** -- Document loading patterns
