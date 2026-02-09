---
title: SpaCy Sentence Tokenizer
description: Integrate SpaCy's NLP sentence tokenizer with Beluga AI text splitters for language-aware chunking that respects sentence boundaries and preserves semantic coherence.
---

## Overview

This guide covers building a custom text splitter that uses [SpaCy](https://spacy.io) for sentence-aware chunking. Standard character-based splitters may cut text mid-sentence, degrading retrieval quality and LLM comprehension. A sentence-aware splitter groups complete sentences into chunks of a target size, producing semantically coherent segments for embedding and retrieval.

The splitter implements Beluga AI's `TextSplitter` interface, making it compatible with the full RAG pipeline -- from document loading through vector store ingestion.

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed
- Python 3.8+ with SpaCy installed
- A SpaCy language model (e.g., `en_core_web_sm`)

## Installation

Install SpaCy and download a language model:

```bash
pip install spacy
python -m spacy download en_core_web_sm
```

The Go integration communicates with SpaCy through a local HTTP API or subprocess. Choose the approach that fits your deployment:

| Method | Tradeoff |
|---|---|
| HTTP microservice | Best for production; independent scaling, language isolation |
| Subprocess | Simpler setup; higher per-call latency |
| Go NLP library | No Python dependency; fewer language models available |

## Configuration

| Option | Description | Default |
|---|---|---|
| `Language` | SpaCy language code (e.g., `en`, `de`, `es`) | `en` |
| `ChunkSize` | Target chunk size in characters | `1000` |
| `ChunkOverlap` | Overlap between consecutive chunks in characters | `200` |

## Usage

### Define the Sentence API Interface

Abstract the sentence tokenization backend so you can swap implementations:

```go
// SentenceAPI provides sentence-level text operations.
type SentenceAPI interface {
    SplitSentences(text string) ([]string, error)
    DetectLanguage(text string) (string, error)
}
```

### Implement the Splitter

The splitter groups sentences into chunks that stay within the configured size, with overlap for context continuity:

```go
package main

import (
    "context"
    "fmt"
    "strings"

    "github.com/lookatitude/beluga-ai/schema"
)

// SpaCySplitter splits text into chunks at sentence boundaries.
type SpaCySplitter struct {
    language     string
    chunkSize    int
    chunkOverlap int
    sentenceAPI  SentenceAPI
}

// NewSpaCySplitter creates a sentence-aware splitter backed by SpaCy.
func NewSpaCySplitter(language string, chunkSize, chunkOverlap int) (*SpaCySplitter, error) {
    api, err := NewSpaCyAPI(language)
    if err != nil {
        return nil, fmt.Errorf("initialize spacy api: %w", err)
    }
    return &SpaCySplitter{
        language:     language,
        chunkSize:    chunkSize,
        chunkOverlap: chunkOverlap,
        sentenceAPI:  api,
    }, nil
}

// SplitText splits text into chunks that respect sentence boundaries.
func (s *SpaCySplitter) SplitText(ctx context.Context, text string) ([]string, error) {
    sentences, err := s.sentenceAPI.SplitSentences(text)
    if err != nil {
        return nil, fmt.Errorf("split sentences: %w", err)
    }

    var chunks []string
    var currentChunk []string
    currentSize := 0

    for _, sentence := range sentences {
        sentenceLen := len(sentence)

        if currentSize+sentenceLen > s.chunkSize && len(currentChunk) > 0 {
            chunks = append(chunks, strings.Join(currentChunk, " "))

            // Retain trailing sentences for overlap
            currentChunk = overlapSentences(currentChunk, s.chunkOverlap)
            currentSize = totalLen(currentChunk)
        }

        currentChunk = append(currentChunk, sentence)
        currentSize += sentenceLen
    }

    if len(currentChunk) > 0 {
        chunks = append(chunks, strings.Join(currentChunk, " "))
    }

    return chunks, nil
}

// SplitDocuments splits each document's content and propagates metadata.
func (s *SpaCySplitter) SplitDocuments(ctx context.Context, documents []schema.Document) ([]schema.Document, error) {
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

// overlapSentences returns trailing sentences whose total length is within overlapSize.
func overlapSentences(sentences []string, overlapSize int) []string {
    var result []string
    currentSize := 0
    for i := len(sentences) - 1; i >= 0 && currentSize < overlapSize; i-- {
        result = append([]string{sentences[i]}, result...)
        currentSize += len(sentences[i])
    }
    return result
}

func totalLen(sentences []string) int {
    n := 0
    for _, s := range sentences {
        n += len(s)
    }
    return n
}

func copyMetadata(src map[string]any) map[string]any {
    dst := make(map[string]any, len(src))
    for k, v := range src {
        dst[k] = v
    }
    return dst
}
```

### Implement the SpaCy API Wrapper

A minimal wrapper that calls SpaCy via a local HTTP service:

```go
// SpaCyAPI communicates with a SpaCy backend for sentence tokenization.
type SpaCyAPI struct {
    language string
    baseURL  string
}

// NewSpaCyAPI creates a client for the SpaCy sentence tokenization service.
func NewSpaCyAPI(language string) (*SpaCyAPI, error) {
    return &SpaCyAPI{
        language: language,
        baseURL:  "http://localhost:8090",
    }, nil
}

// SplitSentences sends text to the SpaCy service and returns sentence boundaries.
func (s *SpaCyAPI) SplitSentences(text string) ([]string, error) {
    // POST to {baseURL}/sentences with {"text": text, "language": language}
    // Parse JSON response: {"sentences": ["...", "..."]}
    return nil, fmt.Errorf("implement SpaCy HTTP client")
}

// DetectLanguage returns the configured language.
func (s *SpaCyAPI) DetectLanguage(text string) (string, error) {
    return s.language, nil
}
```

### Integrate with the RAG Pipeline

Use the splitter in a Beluga AI RAG pipeline:

```go
package main

import (
    "context"
    "fmt"
    "log"
)

func main() {
    ctx := context.Background()

    splitter, err := NewSpaCySplitter("en", 1000, 200)
    if err != nil {
        log.Fatalf("Failed to create splitter: %v", err)
    }

    text := `Natural language processing is a subfield of linguistics and computer science.
It focuses on the interaction between computers and human language.
Modern NLP relies heavily on machine learning approaches.`

    chunks, err := splitter.SplitText(ctx, text)
    if err != nil {
        log.Fatalf("Split failed: %v", err)
    }

    for i, chunk := range chunks {
        fmt.Printf("Chunk %d: %s\n", i, chunk)
    }
}
```

## Advanced Topics

### OpenTelemetry Instrumentation

Add tracing to monitor splitting latency and chunk distribution:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("beluga.splitter.spacy")

func (s *SpaCySplitter) SplitTextWithTracing(ctx context.Context, text string) ([]string, error) {
    ctx, span := tracer.Start(ctx, "spacy.split_text")
    defer span.End()

    span.SetAttributes(
        attribute.String("language", s.language),
        attribute.Int("chunk_size", s.chunkSize),
        attribute.Int("text_length", len(text)),
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

### Multilingual Support

Switch language models dynamically based on content:

```go
func (s *SpaCySplitter) SplitTextMultilingual(ctx context.Context, text string) ([]string, error) {
    lang, err := s.sentenceAPI.DetectLanguage(text)
    if err != nil {
        return nil, fmt.Errorf("detect language: %w", err)
    }

    if lang != s.language {
        // Create a language-specific splitter
        langSplitter, err := NewSpaCySplitter(lang, s.chunkSize, s.chunkOverlap)
        if err != nil {
            return nil, fmt.Errorf("create splitter for %s: %w", lang, err)
        }
        return langSplitter.SplitText(ctx, text)
    }

    return s.SplitText(ctx, text)
}
```

### SpaCy as an HTTP Microservice

For production, run SpaCy as a standalone service using a simple Flask or FastAPI wrapper:

```python
# spacy_service.py
from fastapi import FastAPI
import spacy

app = FastAPI()
models = {}

def get_model(lang: str):
    if lang not in models:
        models[lang] = spacy.load(f"{lang}_core_web_sm")
    return models[lang]

@app.post("/sentences")
def split_sentences(text: str, language: str = "en"):
    nlp = get_model(language)
    doc = nlp(text)
    return {"sentences": [sent.text for sent in doc.sents]}
```

## Troubleshooting

### Language model not found

**Cause**: The SpaCy language model is not installed in the Python environment.

**Resolution**: Install the required model:

```bash
python -m spacy download en_core_web_sm
```

### Sentence splitting returns empty results

**Cause**: The text contains no recognizable sentence boundaries, or the SpaCy API is unreachable.

**Resolution**: Verify the text has standard punctuation. Check that the SpaCy service is running and accessible at the configured URL.

## Related Resources

- [Tiktoken BPE Tokenizer](/integrations/tiktoken-bpe) -- Token-count-aware splitting
- [RAG Pipeline Guide](/guides/rag) -- End-to-end retrieval-augmented generation
- [Text Splitter API Reference](/api-reference/rag/splitter) -- All built-in splitter implementations
