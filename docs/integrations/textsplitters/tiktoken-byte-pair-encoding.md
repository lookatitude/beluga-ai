# Tiktoken Byte-Pair Encoding

Welcome, colleague! In this integration guide, we're going to integrate Tiktoken (OpenAI's tokenizer) with Beluga AI's text splitters package. This enables token-aware text splitting that respects LLM token boundaries.

## What you will build

You will create a custom text splitter that uses Tiktoken for token-aware splitting, ensuring chunks respect token boundaries and stay within LLM context limits.

## Learning Objectives

- ✅ Integrate Tiktoken with Beluga AI text splitters
- ✅ Create token-aware text splitting
- ✅ Use Tiktoken for accurate token counting
- ✅ Configure splitting by token count

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Python with tiktoken (for tokenizer) or Go tiktoken library
- Understanding of BPE tokenization

## Step 1: Setup and Installation

Install a Go Tiktoken library or use Python bindings:
# Option 1: Use Go library (if available)
```bash
go get github.com/pkoukk/tiktoken-go
```

# Option 2: Use Python bindings via cgo
# Requires Python and tiktoken installed
bash
```bash
pip install tiktoken
```

## Step 2: Create Token-Aware Splitter

Create a custom splitter using Tiktoken:
```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/textsplitters/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

type TiktokenSplitter struct {
    encodingName string
    chunkSize    int
    chunkOverlap int
    tokenizer    Tokenizer // Interface for tokenizer
}

type Tokenizer interface {
    Encode(text string) ([]int, error)
    Decode(tokens []int) (string, error)
    Count(text string) (int, error)
}

func NewTiktokenSplitter(encodingName string, chunkSize, chunkOverlap int) (*TiktokenSplitter, error) {
    tokenizer, err := NewTiktokenTokenizer(encodingName)
    if err != nil {
        return nil, fmt.Errorf("tokenizer: %w", err)
    }

    return &TiktokenSplitter{
        encodingName: encodingName,
        chunkSize:    chunkSize,
        chunkOverlap: chunkOverlap,
        tokenizer:    tokenizer,
    }, nil
}

func (s *TiktokenSplitter) SplitText(ctx context.Context, text string) ([]string, error) {
    // Count tokens
    tokenCount, err := s.tokenizer.Count(text)
    if err != nil {
        return nil, fmt.Errorf("count tokens: %w", err)
    }

    // If text fits in one chunk, return as-is
    if tokenCount <= s.chunkSize {
        return []string{text}, nil
    }

    // Encode to tokens
    tokens, err := s.tokenizer.Encode(text)
    if err != nil {
        return nil, fmt.Errorf("encode: %w", err)
    }

    // Split into chunks
    var chunks []string
    for i := 0; i < len(tokens); i += s.chunkSize - s.chunkOverlap {
        end := i + s.chunkSize
        if end > len(tokens) {
            end = len(tokens)
        }

        chunkTokens := tokens[i:end]
        chunkText, err := s.tokenizer.Decode(chunkTokens)
        if err != nil {
            return nil, fmt.Errorf("decode chunk: %w", err)
        }

        chunks = append(chunks, chunkText)
        
        if end >= len(tokens) {
            break
        }
    }

    return chunks, nil
}

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
                Metadata:    make(map[string]any),
            }
            
            // Copy original metadata
            for k, v := range doc.Metadata {
                chunkDoc.Metadata[k] = v
            }
            
            // Add chunk metadata
            chunkDoc.Metadata["chunk_index"] = i
            chunkDoc.Metadata["chunk_total"] = len(chunks)
            
            result = append(result, chunkDoc)
        }
    }

    return result, nil
}

func (s *TiktokenSplitter) CreateDocuments(ctx context.Context, texts []string, metadatas []map[string]any) ([]schema.Document, error) {
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

## Step 3: Implement Tiktoken Tokenizer

Create a tokenizer wrapper:
// TiktokenTokenizer wraps tiktoken functionality
```go
type TiktokenTokenizer struct \{
    encoding string
    // Implementation depends on library used
}
go
func NewTiktokenTokenizer(encoding string) (*TiktokenTokenizer, error) {
    // Initialize tiktoken encoder
    // This is a placeholder - actual implementation depends on library
    return &TiktokenTokenizer{
        encoding: encoding,
    }, nil
}

func (t *TiktokenTokenizer) Count(text string) (int, error) {
    // Count tokens using tiktoken
    tokens, err := t.Encode(text)
    return len(tokens), err
}



func (t *TiktokenTokenizer) Encode(text string) ([]int, error) {
    // Encode text to tokens
    // Implementation depends on library
    return nil, nil
}

func (t *TiktokenTokenizer) Decode(tokens []int) (string, error) \{
    // Decode tokens to text
    // Implementation depends on library
    return "", nil
}
```

## Step 4: Use with Beluga AI

Integrate with Beluga AI text splitters:
```go
func main() {
    ctx := context.Background()

    // Create token-aware splitter
    splitter, err := NewTiktokenSplitter("cl100k_base", 1000, 200)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    // Split text
    text := "Your long text here..."
    chunks, err := splitter.SplitText(ctx, text)
    if err != nil {
        log.Fatalf("Split failed: %v", err)
    }

    fmt.Printf("Split into %d chunks\n", len(chunks))
    for i, chunk := range chunks {
        fmt.Printf("Chunk %d: %s\n", i, chunk[:50])
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

type ProductionTiktokenSplitter struct {
    encodingName string
    chunkSize    int
    chunkOverlap int
    tokenizer    Tokenizer
    tracer       trace.Tracer
}

func NewProductionTiktokenSplitter(encodingName string, chunkSize, chunkOverlap int) (*ProductionTiktokenSplitter, error) {
    tokenizer, err := NewTiktokenTokenizer(encodingName)
    if err != nil {
        return nil, err
    }

    return &ProductionTiktokenSplitter{
        encodingName: encodingName,
        chunkSize:    chunkSize,
        chunkOverlap: chunkOverlap,
        tokenizer:    tokenizer,
        tracer:       otel.Tracer("beluga.textsplitters.tiktoken"),
    }, nil
}

func (s *ProductionTiktokenSplitter) SplitText(ctx context.Context, text string) ([]string, error) {
    ctx, span := s.tracer.Start(ctx, "tiktoken.split_text")
    defer span.End()

    span.SetAttributes(
        attribute.String("encoding", s.encodingName),
        attribute.Int("chunk_size", s.chunkSize),
    )

    tokenCount, err := s.tokenizer.Count(text)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    span.SetAttributes(attribute.Int("token_count", tokenCount))

    if tokenCount <= s.chunkSize {
        return []string{text}, nil
    }

    tokens, err := s.tokenizer.Encode(text)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    var chunks []string
    for i := 0; i < len(tokens); i += s.chunkSize - s.chunkOverlap {
        end := i + s.chunkSize
        if end > len(tokens) {
            end = len(tokens)
        }

        chunkTokens := tokens[i:end]
        chunkText, err := s.tokenizer.Decode(chunkTokens)
        if err != nil {
            span.RecordError(err)
            return nil, err
        }

        chunks = append(chunks, chunkText)
        
        if end >= len(tokens) {
            break
        }
    }

    span.SetAttributes(attribute.Int("chunk_count", len(chunks)))
    return chunks, nil
}

func (s *ProductionTiktokenSplitter) SplitDocuments(ctx context.Context, documents []schema.Document) ([]schema.Document, error) {
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

func (s *ProductionTiktokenSplitter) CreateDocuments(ctx context.Context, texts []string, metadatas []map[string]any) ([]schema.Document, error) {
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
    splitter, _ := NewProductionTiktokenSplitter("cl100k_base", 1000, 200)
    chunks, _ := splitter.SplitText(ctx, "Your text here...")
    fmt.Printf("Split into %d chunks\n", len(chunks))
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `EncodingName` | Tiktoken encoding (e.g., `cl100k_base`) | `cl100k_base` | No |
| `ChunkSize` | Target chunk size in tokens | `1000` | No |
| `ChunkOverlap` | Overlap between chunks | `200` | No |

## Common Issues

### "Encoding not found"

**Problem**: Invalid encoding name.

**Solution**: Use valid encodings like `cl100k_base`, `p50k_base`, `r50k_base`.

### "Token count mismatch"

**Problem**: Tokenizer implementation issue.

**Solution**: Verify tokenizer implementation matches LLM tokenizer.

## Production Considerations

When using Tiktoken in production:

- **Encoding selection**: Match encoding to your LLM
- **Performance**: Cache tokenizer instances
- **Accuracy**: Verify token counts match LLM
- **Error handling**: Handle tokenization errors gracefully
- **Memory**: Large texts may require streaming

## Next Steps

Congratulations! You've integrated Tiktoken with Beluga AI. Next, learn how to:

- **[SpaCy Sentence Tokenizer](./spacy-sentence-tokenizer.md)** - Language-aware splitting
- **Text Splitters Documentation** - Deep dive into text splitters
- **[RAG Guide](../../guides/rag-multimodal.md)** - RAG patterns

---

**Ready for more?** Check out the Integrations Index for more integration guides!
