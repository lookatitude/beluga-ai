# SpaCy Sentence Tokenizer

Welcome, colleague! In this integration guide, we're going to integrate SpaCy's sentence tokenizer with Beluga AI's text splitters package. This enables language-aware text splitting that respects sentence boundaries.

## What you will build

You will create a custom text splitter that uses SpaCy for sentence-aware splitting, ensuring chunks respect sentence boundaries and maintain semantic coherence.

## Learning Objectives

- ✅ Integrate SpaCy with Beluga AI text splitters
- ✅ Create sentence-aware text splitting
- ✅ Use SpaCy for language detection and tokenization
- ✅ Configure splitting by sentence boundaries

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Python with SpaCy installed
- SpaCy language model (e.g., `en_core_web_sm`)

## Step 1: Setup and Installation

Install SpaCy:
bash
```bash
pip install spacy
python -m spacy download en_core_web_sm
```

For Go integration, you'll need to use Python bindings or a Go wrapper:
# Option 1: Use Python via subprocess/API
# Option 2: Use Go library that wraps SpaCy
```

## Step 2: Create Sentence-Aware Splitter

Create a custom splitter using SpaCy:
```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/textsplitters/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

type SpaCySplitter struct {
    language    string
    chunkSize   int
    chunkOverlap int
    sentenceAPI SentenceAPI // Interface for sentence tokenization
}

type SentenceAPI interface {
    SplitSentences(text string) ([]string, error)
    DetectLanguage(text string) (string, error)
}

func NewSpaCySplitter(language string, chunkSize, chunkOverlap int) (*SpaCySplitter, error) {
    api, err := NewSpaCyAPI(language)
    if err != nil {
        return nil, fmt.Errorf("spacy api: %w", err)
    }

    return &SpaCySplitter{
        language:    language,
        chunkSize:   chunkSize,
        chunkOverlap: chunkOverlap,
        sentenceAPI: api,
    }, nil
}

func (s *SpaCySplitter) SplitText(ctx context.Context, text string) ([]string, error) {
    // Split into sentences
    sentences, err := s.sentenceAPI.SplitSentences(text)
    if err != nil {
        return nil, fmt.Errorf("split sentences: %w", err)
    }

    // Group sentences into chunks
    var chunks []string
    var currentChunk []string
    currentSize := 0

    for _, sentence := range sentences {
        sentenceSize := len(sentence)
        
        // If adding this sentence would exceed chunk size, finalize current chunk
        if currentSize+sentenceSize > s.chunkSize && len(currentChunk) > 0 {
            chunks = append(chunks, s.joinSentences(currentChunk))
            
            // Start new chunk with overlap
            overlapSize := s.chunkOverlap
            currentChunk = s.getOverlapSentences(currentChunk, overlapSize)
            currentSize = s.calculateSize(currentChunk)
        }
        
        currentChunk = append(currentChunk, sentence)
        currentSize += sentenceSize
    }

    // Add final chunk
    if len(currentChunk) > 0 {
        chunks = append(chunks, s.joinSentences(currentChunk))
    }

    return chunks, nil
}

func (s *SpaCySplitter) joinSentences(sentences []string) string {
    result := ""
    for i, sent := range sentences {
        if i > 0 {
            result += " "
        }
        result += sent
    }
    return result
}

func (s *SpaCySplitter) getOverlapSentences(sentences []string, overlapSize int) []string {
    // Get last N sentences for overlap
    if len(sentences) == 0 {
        return []string{}
    }
    
    result := []string{}
    currentSize := 0
    for i := len(sentences) - 1; i >= 0 && currentSize < overlapSize; i-- {
        result = append([]string{sentences[i]}, result...)
        currentSize += len(sentences[i])
    }
    return result
}

func (s *SpaCySplitter) calculateSize(sentences []string) int {
    size := 0
    for _, s := range sentences {
        size += len(s)
    }
    return size
}

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
                Metadata:    make(map[string]any),
            }
            
            for k, v := range doc.Metadata {
                chunkDoc.Metadata[k] = v
            }
            
            chunkDoc.Metadata["chunk_index"] = i
            chunkDoc.Metadata["chunk_total"] = len(chunks)
            
            result = append(result, chunkDoc)
        }
    }

    return result, nil
}

func (s *SpaCySplitter) CreateDocuments(ctx context.Context, texts []string, metadatas []map[string]any) ([]schema.Document, error) {
    var result []schema.Document
    
    for i, text := range texts {
        chunks, err := s.SplitText(ctx, text)
        if err != nil {
            return nil, err
        }

        metadata := make(map[string]any)
        if i < len(metadatas) {
            metadata = metadatas[i]
        }

        for j, chunk := range chunks {
            doc := schema.Document{
                PageContent: chunk,
                Metadata:    make(map[string]any),
            }
            
            for k, v := range metadata {
                doc.Metadata[k] = v
            }
            
            doc.Metadata["chunk_index"] = j
            doc.Metadata["chunk_total"] = len(chunks)
            
            result = append(result, doc)
        }
    }

    return result, nil
}
```

## Step 3: Implement SpaCy API Wrapper

Create a wrapper for SpaCy:
// SpaCyAPI wraps SpaCy functionality
```go
type SpaCyAPI struct {
    language string
    // Implementation depends on integration method
}

func NewSpaCyAPI(language string) (*SpaCyAPI, error) {
    // Initialize SpaCy
    // This could call Python SpaCy via subprocess, API, or Go wrapper
    return &SpaCyAPI{
        language: language,
    }, nil
}

func (s *SpaCyAPI) SplitSentences(text string) ([]string, error) {
    // Call SpaCy to split sentences
    // Implementation depends on integration method
    // Example: Call Python script or API
    return nil, nil
}

func (s *SpaCyAPI) DetectLanguage(text string) (string, error) {
    // Detect language using SpaCy
    return s.language, nil
}
```

## Step 4: Use with Beluga AI

Integrate with Beluga AI text splitters:
```go
func main() {
    ctx := context.Background()

    // Create sentence-aware splitter
    splitter, err := NewSpaCySplitter("en", 1000, 200)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    // Split text
    text := "First sentence. Second sentence. Third sentence."
    chunks, err := splitter.SplitText(ctx, text)
    if err != nil {
        log.Fatalf("Split failed: %v", err)
    }

    fmt.Printf("Split into %d chunks\n", len(chunks))
    for i, chunk := range chunks {
        fmt.Printf("Chunk %d: %s\n", i, chunk)
    }
}
```

## Step 5: Complete Integration

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/textsplitters/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type ProductionSpaCySplitter struct {
    language    string
    chunkSize   int
    chunkOverlap int
    sentenceAPI SentenceAPI
    tracer      trace.Tracer
}

func NewProductionSpaCySplitter(language string, chunkSize, chunkOverlap int) (*ProductionSpaCySplitter, error) {
    api, err := NewSpaCyAPI(language)
    if err != nil {
        return nil, err
    }

    return &ProductionSpaCySplitter{
        language:    language,
        chunkSize:   chunkSize,
        chunkOverlap: chunkOverlap,
        sentenceAPI: api,
        tracer:      otel.Tracer("beluga.textsplitters.spacy"),
    }, nil
}

func (s *ProductionSpaCySplitter) SplitText(ctx context.Context, text string) ([]string, error) {
    ctx, span := s.tracer.Start(ctx, "spacy.split_text")
    defer span.End()

    span.SetAttributes(
        attribute.String("language", s.language),
        attribute.Int("chunk_size", s.chunkSize),
    )

    sentences, err := s.sentenceAPI.SplitSentences(text)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    span.SetAttributes(attribute.Int("sentence_count", len(sentences)))

    var chunks []string
    var currentChunk []string
    currentSize := 0

    for _, sentence := range sentences {
        sentenceSize := len(sentence)
        
        if currentSize+sentenceSize > s.chunkSize && len(currentChunk) > 0 {
            chunks = append(chunks, s.joinSentences(currentChunk))
            
            overlapSize := s.chunkOverlap
            currentChunk = s.getOverlapSentences(currentChunk, overlapSize)
            currentSize = s.calculateSize(currentChunk)
        }
        
        currentChunk = append(currentChunk, sentence)
        currentSize += sentenceSize
    }

    if len(currentChunk) > 0 {
        chunks = append(chunks, s.joinSentences(currentChunk))
    }

    span.SetAttributes(attribute.Int("chunk_count", len(chunks)))
    return chunks, nil
}

func (s *ProductionSpaCySplitter) joinSentences(sentences []string) string {
    result := ""
    for i, sent := range sentences {
        if i > 0 {
            result += " "
        }
        result += sent
    }
    return result
}

func (s *ProductionSpaCySplitter) getOverlapSentences(sentences []string, overlapSize int) []string {
    if len(sentences) == 0 {
        return []string{}
    }
    
    result := []string{}
    currentSize := 0
    for i := len(sentences) - 1; i >= 0 && currentSize < overlapSize; i-- {
        result = append([]string{sentences[i]}, result...)
        currentSize += len(sentences[i])
    }
    return result
}

func (s *ProductionSpaCySplitter) calculateSize(sentences []string) int {
    size := 0
    for _, s := range sentences {
        size += len(s)
    }
    return size
}

func (s *ProductionSpaCySplitter) SplitDocuments(ctx context.Context, documents []schema.Document) ([]schema.Document, error) {
    var result []schema.Document
    
    for _, doc := range documents {
        chunks, err := s.SplitText(ctx, doc.PageContent)
        if err != nil {
            return nil, err
        }

        for i, chunk := range chunks {
            chunkDoc := schema.Document{
                PageContent: chunk,
                Metadata:    make(map[string]any),
            }
            
            for k, v := range doc.Metadata {
                chunkDoc.Metadata[k] = v
            }
            
            chunkDoc.Metadata["chunk_index"] = i
            chunkDoc.Metadata["chunk_total"] = len(chunks)
            
            result = append(result, chunkDoc)
        }
    }

    return result, nil
}

func (s *ProductionSpaCySplitter) CreateDocuments(ctx context.Context, texts []string, metadatas []map[string]any) ([]schema.Document, error) {
    var result []schema.Document
    
    for i, text := range texts {
        chunks, err := s.SplitText(ctx, text)
        if err != nil {
            return nil, err
        }

        metadata := make(map[string]any)
        if i < len(metadatas) {
            metadata = metadatas[i]
        }

        for j, chunk := range chunks {
            doc := schema.Document{
                PageContent: chunk,
                Metadata:    make(map[string]any),
            }
            
            for k, v := range metadata {
                doc.Metadata[k] = v
            }
            
            doc.Metadata["chunk_index"] = j
            doc.Metadata["chunk_total"] = len(chunks)
            
            result = append(result, doc)
        }
    }

    return result, nil
}

func main() {
    ctx := context.Background()
    splitter, _ := NewProductionSpaCySplitter("en", 1000, 200)
    chunks, _ := splitter.SplitText(ctx, "Your text here...")
    fmt.Printf("Split into %d chunks\n", len(chunks))
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `Language` | Language code (e.g., `en`, `es`) | `en` | No |
| `ChunkSize` | Target chunk size in characters | `1000` | No |
| `ChunkOverlap` | Overlap between chunks | `200` | No |

## Common Issues

### "Language model not found"

**Problem**: SpaCy language model not installed.

**Solution**: Install model:python -m spacy download en_core_web_sm
```

### "Sentence splitting failed"

**Problem**: Text format issues or API errors.

**Solution**: Verify text encoding and SpaCy API connection.

## Production Considerations

When using SpaCy in production:

- **Language models**: Install required language models
- **Performance**: Cache SpaCy instances
- **Multilingual**: Support multiple languages
- **Error handling**: Handle tokenization errors gracefully
- **Integration**: Consider API-based integration for scalability

## Next Steps

Congratulations! You've integrated SpaCy with Beluga AI. Next, learn how to:

- **[Tiktoken Byte-Pair Encoding](./tiktoken-byte-pair-encoding.md)** - Token-aware splitting
- **[Text Splitters Documentation](../../api/packages/textsplitters.md)** - Deep dive into text splitters
- **[RAG Guide](../../guides/rag-multimodal.md)** - RAG patterns

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
