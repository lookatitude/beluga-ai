---
title: Tiktoken Byte-Pair Encoding
description: Integrate OpenAI's Tiktoken tokenizer with Beluga AI text splitters for token-count-aware chunking that respects LLM context window limits.
---

## Overview

LLMs operate on tokens, not characters. A 4,000-character chunk might contain anywhere from 800 to 1,200 tokens depending on the text content and encoding. Character-based splitters cannot guarantee that chunks fit within an LLM's context window, leading to truncation or wasted capacity.

This guide covers building a custom text splitter that uses [Tiktoken](https://github.com/openai/tiktoken) (OpenAI's byte-pair encoding tokenizer) for token-aware text chunking. By encoding text to tokens, splitting at token boundaries, and decoding back to text, the splitter ensures each chunk stays within the model's token limit. This matters most when you need precise control over context window utilization -- for example, when reserving specific token budgets for system prompts, retrieved context, and generation output.

The splitter implements Beluga AI's `TextSplitter` interface and works with any encoding supported by Tiktoken, including `cl100k_base` (GPT-4, GPT-3.5-turbo) and `o200k_base` (GPT-4o).

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed
- A Go Tiktoken library or Python with `tiktoken` installed
- Understanding of BPE tokenization concepts

## Installation

Install a Go-native Tiktoken library:

```bash
go get github.com/pkoukk/tiktoken-go
```

Alternatively, if using Python bindings:

```bash
pip install tiktoken
```

## Configuration

| Option | Description | Default |
|---|---|---|
| `EncodingName` | Tiktoken encoding name | `cl100k_base` |
| `ChunkSize` | Target chunk size in tokens | `1000` |
| `ChunkOverlap` | Overlap between chunks in tokens | `200` |

Common encoding names:

| Encoding | Models |
|---|---|
| `cl100k_base` | GPT-4, GPT-3.5-turbo, text-embedding-ada-002 |
| `o200k_base` | GPT-4o |
| `p50k_base` | Codex, text-davinci-002/003 |
| `r50k_base` | GPT-3 (davinci, curie, babbage, ada) |

## Usage

### Define the Tokenizer Interface

Abstract the tokenizer to allow swapping implementations:

```go
// Tokenizer encodes text to token IDs and decodes token IDs back to text.
type Tokenizer interface {
    Encode(text string) ([]int, error)
    Decode(tokens []int) (string, error)
    Count(text string) (int, error)
}
```

### Implement the Token-Aware Splitter

The splitter encodes text to tokens, partitions the token sequence into overlapping windows, and decodes each window back to text:

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/schema"
)

// TiktokenSplitter splits text into chunks based on token count.
type TiktokenSplitter struct {
    encodingName string
    chunkSize    int
    chunkOverlap int
    tokenizer    Tokenizer
}

// NewTiktokenSplitter creates a token-aware splitter for the given encoding.
func NewTiktokenSplitter(encodingName string, chunkSize, chunkOverlap int) (*TiktokenSplitter, error) {
    tokenizer, err := NewTiktokenTokenizer(encodingName)
    if err != nil {
        return nil, fmt.Errorf("create tokenizer: %w", err)
    }
    return &TiktokenSplitter{
        encodingName: encodingName,
        chunkSize:    chunkSize,
        chunkOverlap: chunkOverlap,
        tokenizer:    tokenizer,
    }, nil
}

// SplitText splits text into token-bounded chunks.
func (s *TiktokenSplitter) SplitText(ctx context.Context, text string) ([]string, error) {
    tokenCount, err := s.tokenizer.Count(text)
    if err != nil {
        return nil, fmt.Errorf("count tokens: %w", err)
    }

    // Text fits in a single chunk
    if tokenCount <= s.chunkSize {
        return []string{text}, nil
    }

    tokens, err := s.tokenizer.Encode(text)
    if err != nil {
        return nil, fmt.Errorf("encode text: %w", err)
    }

    stride := s.chunkSize - s.chunkOverlap
    var chunks []string

    for i := 0; i < len(tokens); i += stride {
        end := i + s.chunkSize
        if end > len(tokens) {
            end = len(tokens)
        }

        chunkText, err := s.tokenizer.Decode(tokens[i:end])
        if err != nil {
            return nil, fmt.Errorf("decode chunk at offset %d: %w", i, err)
        }
        chunks = append(chunks, chunkText)

        if end >= len(tokens) {
            break
        }
    }

    return chunks, nil
}

// SplitDocuments splits each document's content and propagates metadata.
func (s *TiktokenSplitter) SplitDocuments(ctx context.Context, documents []schema.Document) ([]schema.Document, error) {
    var result []schema.Document

    for _, doc := range documents {
        chunks, err := s.SplitText(ctx, doc.PageContent)
        if err != nil {
            return nil, err
        }

        for i, chunk := range chunks {
            chunkDoc := schema.Document{
                PageContent: chunk,
                Metadata:    copyMetadata(doc.Metadata),
            }
            chunkDoc.Metadata["chunk_index"] = i
            chunkDoc.Metadata["chunk_total"] = len(chunks)
            result = append(result, chunkDoc)
        }
    }

    return result, nil
}

// CreateDocuments creates documents from raw text slices with optional metadata.
func (s *TiktokenSplitter) CreateDocuments(ctx context.Context, texts []string, metadatas []map[string]any) ([]schema.Document, error) {
    var result []schema.Document

    for i, text := range texts {
        chunks, err := s.SplitText(ctx, text)
        if err != nil {
            return nil, err
        }

        var baseMeta map[string]any
        if i < len(metadatas) {
            baseMeta = metadatas[i]
        }

        for j, chunk := range chunks {
            doc := schema.Document{
                PageContent: chunk,
                Metadata:    copyMetadata(baseMeta),
            }
            doc.Metadata["chunk_index"] = j
            doc.Metadata["chunk_total"] = len(chunks)
            result = append(result, doc)
        }
    }

    return result, nil
}

func copyMetadata(src map[string]any) map[string]any {
    dst := make(map[string]any)
    for k, v := range src {
        dst[k] = v
    }
    return dst
}
```

### Implement the Tokenizer Wrapper

Wrap the Go Tiktoken library:

```go
import (
    tiktoken "github.com/pkoukk/tiktoken-go"
)

// TiktokenTokenizer wraps tiktoken-go for encoding and decoding.
type TiktokenTokenizer struct {
    encoding *tiktoken.Tiktoken
}

// NewTiktokenTokenizer creates a tokenizer for the given encoding name.
func NewTiktokenTokenizer(encodingName string) (*TiktokenTokenizer, error) {
    enc, err := tiktoken.GetEncoding(encodingName)
    if err != nil {
        return nil, fmt.Errorf("get encoding %s: %w", encodingName, err)
    }
    return &TiktokenTokenizer{encoding: enc}, nil
}

// Encode converts text to a sequence of token IDs.
func (t *TiktokenTokenizer) Encode(text string) ([]int, error) {
    return t.encoding.Encode(text, nil, nil), nil
}

// Decode converts token IDs back to text.
func (t *TiktokenTokenizer) Decode(tokens []int) (string, error) {
    return t.encoding.Decode(tokens), nil
}

// Count returns the number of tokens in the text.
func (t *TiktokenTokenizer) Count(text string) (int, error) {
    tokens := t.encoding.Encode(text, nil, nil)
    return len(tokens), nil
}
```

### Integrate with the RAG Pipeline

Use the token-aware splitter in a Beluga AI application:

```go
package main

import (
    "context"
    "fmt"
    "log"
)

func main() {
    ctx := context.Background()

    splitter, err := NewTiktokenSplitter("cl100k_base", 500, 50)
    if err != nil {
        log.Fatalf("Failed to create splitter: %v", err)
    }

    text := `Beluga AI v2 is a Go-native framework for building agentic AI systems.
It provides streaming-first design with iter.Seq2, protocol interoperability
via MCP and A2A, and a pluggable architecture for LLMs, tools, memory, and RAG.
The framework supports production patterns including circuit breakers, rate
limiting, human-in-the-loop approval, and OpenTelemetry instrumentation.`

    chunks, err := splitter.SplitText(ctx, text)
    if err != nil {
        log.Fatalf("Split failed: %v", err)
    }

    for i, chunk := range chunks {
        tokenCount, _ := splitter.tokenizer.Count(chunk)
        fmt.Printf("Chunk %d (%d tokens): %s...\n", i, tokenCount, chunk[:min(50, len(chunk))])
    }
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

## Advanced Topics

### OpenTelemetry Instrumentation

Add tracing to monitor tokenization and splitting performance:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("beluga.splitter.tiktoken")

func (s *TiktokenSplitter) SplitTextWithTracing(ctx context.Context, text string) ([]string, error) {
    ctx, span := tracer.Start(ctx, "tiktoken.split_text")
    defer span.End()

    span.SetAttributes(
        attribute.String("encoding", s.encodingName),
        attribute.Int("chunk_size_tokens", s.chunkSize),
        attribute.Int("text_length_chars", len(text)),
    )

    chunks, err := s.SplitText(ctx, text)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    span.SetAttributes(attribute.Int("chunk_count", len(chunks)))
    return chunks, nil
}
```

### Choosing the Right Encoding

Match the encoding to your target model to get accurate token counts:

```go
func EncodingForModel(model string) string {
    switch {
    case strings.HasPrefix(model, "gpt-4o"):
        return "o200k_base"
    case strings.HasPrefix(model, "gpt-4"),
         strings.HasPrefix(model, "gpt-3.5-turbo"):
        return "cl100k_base"
    case strings.HasPrefix(model, "text-davinci"):
        return "p50k_base"
    default:
        return "cl100k_base"
    }
}
```

### Combining with Sentence Splitting

For best results, split on sentence boundaries first, then verify each chunk fits within the token budget:

```go
func SplitSentenceThenToken(ctx context.Context, text string, sentSplitter *SpaCySplitter, tokSplitter *TiktokenSplitter) ([]string, error) {
    // First pass: sentence-aware split
    sentenceChunks, err := sentSplitter.SplitText(ctx, text)
    if err != nil {
        return nil, err
    }

    // Second pass: ensure each chunk fits token limit
    var result []string
    for _, chunk := range sentenceChunks {
        tokenChunks, err := tokSplitter.SplitText(ctx, chunk)
        if err != nil {
            return nil, err
        }
        result = append(result, tokenChunks...)
    }
    return result, nil
}
```

## Troubleshooting

### Encoding not found

**Cause**: The encoding name does not match any Tiktoken encoding.

**Resolution**: Use one of the supported encodings: `cl100k_base`, `o200k_base`, `p50k_base`, or `r50k_base`. Verify with `tiktoken.GetEncoding(name)`.

### Token count mismatch with LLM

**Cause**: The tokenizer encoding does not match the model's tokenizer.

**Resolution**: Ensure the encoding name matches your target model. GPT-4 and GPT-3.5-turbo use `cl100k_base`; GPT-4o uses `o200k_base`. Use `EncodingForModel()` to automate selection.

### Large text causes high memory usage

**Cause**: Encoding very large documents produces large token slices.

**Resolution**: Split text into paragraphs or sections before tokenization. Process documents in batches rather than loading all content at once.

## Related Resources

- [SpaCy Sentence Tokenizer](/integrations/spacy-tokenizer) -- Sentence-boundary-aware splitting
- [RAG Pipeline Guide](/guides/rag) -- End-to-end retrieval-augmented generation
- [Text Splitter API Reference](/api-reference/rag/splitter) -- All built-in splitter implementations
