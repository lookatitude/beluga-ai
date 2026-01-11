# Quickstart Guide: V2 Framework Alignment

**Feature**: V2 Framework Alignment  
**Date**: 2025-01-27  
**Status**: Complete

## Overview

This quickstart guide provides examples and migration information for the v2 framework alignment changes. All changes are backward compatible - existing code continues to work without modifications.

---

## What's New in V2 Alignment

### 1. Enhanced OTEL Observability

All packages now have comprehensive OTEL observability (metrics, tracing, logging) integrated consistently.

**No code changes required** - observability is automatically available.

**Example**: Metrics are now available for all LLM operations:

```go
// Metrics are automatically collected
llm, _ := llms.NewProvider(ctx, "openai", config)
response, err := llm.Generate(ctx, "Hello, world!")
// Metrics, traces, and logs are automatically generated
```

---

### 2. New Providers

#### Grok LLM Provider

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/llms"
    _ "github.com/lookatitude/beluga-ai/pkg/llms/providers/grok" // Auto-register
)

config := llms.Config{
    ProviderName: "grok",
    APIKey:       "your-grok-api-key",
    Model:        "grok-beta",
}

llm, err := llms.NewProvider(ctx, "grok", config)
if err != nil {
    log.Fatal(err)
}

response, err := llm.Generate(ctx, "Explain quantum computing")
```

#### Gemini LLM Provider

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/llms"
    _ "github.com/lookatitude/beluga-ai/pkg/llms/providers/gemini" // Auto-register
)

config := llms.Config{
    ProviderName: "gemini",
    APIKey:       "your-gemini-api-key",
    Model:        "gemini-pro",
}

llm, err := llms.NewProvider(ctx, "gemini", config)
if err != nil {
    log.Fatal(err)
}

response, err := llm.Generate(ctx, "What is machine learning?")
```

---

### 3. Multimodal Capabilities

#### Image Messages

```go
import "github.com/lookatitude/beluga-ai/pkg/schema"

// Create an image message
imageData := []byte{...} // Your image bytes
imgMsg := schema.NewImageMessage(imageData, "jpeg")
imgMsg.Caption = "A beautiful sunset over the ocean"

// Use with existing message processing
processMessage(imgMsg) // Works with existing Message interface
```

#### Voice Documents

```go
import "github.com/lookatitude/beluga-ai/pkg/schema"

// Create a voice document
audioData := []byte{...} // Your audio bytes
voiceDoc := schema.NewVoiceDocument(audioData, "wav")
voiceDoc.Transcript = "Hello, this is a voice message"
voiceDoc.Language = "en-US"

// Use with existing document processing
processDocument(voiceDoc) // Works with existing Document interface
```

#### Multimodal Embeddings

```go
import "github.com/lookatitude/beluga-ai/pkg/embeddings"

// Embed an image
imgMsg := schema.NewImageMessage(imageData, "jpeg")
embedding, err := embedder.Embed(ctx, []schema.Document{imgMsg})
if err != nil {
    log.Fatal(err)
}

// Embed text (still works as before)
textDoc := schema.NewTextDocument("Hello, world!")
textEmbedding, err := embedder.Embed(ctx, []schema.Document{textDoc})
```

#### Multimodal Vector Stores

```go
import "github.com/lookatitude/beluga-ai/pkg/vectorstores"

// Store multimodal vectors
imgMsg := schema.NewImageMessage(imageData, "jpeg")
embedding := []float32{...} // Image embedding
err := store.AddDocuments(ctx, []schema.Document{imgMsg}, [][]float32{embedding})

// Search with multimodal query
queryImg := schema.NewImageMessage(queryImageData, "jpeg")
queryEmbedding := []float32{...}
results, err := store.SimilaritySearch(ctx, queryEmbedding, 10)
```

---

### 4. Enhanced Testing

All packages now have comprehensive test utilities and advanced test suites.

**For Package Developers**:

```go
// Use advanced test utilities
import "github.com/lookatitude/beluga-ai/pkg/llms"

func TestLLMProvider(t *testing.T) {
    // Use AdvancedMockLLM for testing
    mockLLM := llms.NewAdvancedMockLLM(
        llms.WithMockResponse("test response"),
        llms.WithMockDelay(100*time.Millisecond),
    )
    
    // Run table-driven tests
    testCases := []struct {
        name     string
        prompt   string
        expected string
    }{
        {"simple", "Hello", "Hello, world!"},
        {"complex", "Explain AI", "AI is..."},
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result, err := mockLLM.Generate(ctx, tc.prompt)
            assert.NoError(t, err)
            assert.Equal(t, tc.expected, result)
        })
    }
}
```

---

## Migration Guide

### No Breaking Changes

**All existing code continues to work without modifications.** The v2 alignment changes are fully backward compatible.

### Optional: Adopt New Features

You can optionally adopt new features incrementally:

1. **Try New Providers**: Switch to Grok or Gemini if desired
2. **Add Multimodal Support**: Start using image/voice messages where needed
3. **Leverage Enhanced Observability**: Use OTEL metrics and traces for monitoring

### Example: Gradual Migration

```go
// Existing code (still works)
llm, _ := llms.NewProvider(ctx, "openai", openaiConfig)
response, _ := llm.Generate(ctx, "Hello")

// Optionally try new provider
grokLLM, _ := llms.NewProvider(ctx, "grok", grokConfig)
grokResponse, _ := grokLLM.Generate(ctx, "Hello")

// Optionally add multimodal
imgMsg := schema.NewImageMessage(imageData, "jpeg")
// Use with existing workflows
```

---

## Package Structure Changes

### Internal Reorganization

Some packages have been internally reorganized to match v2 structure. **This does not affect your code** - all public APIs remain unchanged.

**What Changed**:
- Files moved to standard locations (iface/, internal/, providers/)
- Missing files added (test_utils.go, advanced_test.go)
- Internal utilities moved to internal/ subdirectory

**What Stayed the Same**:
- All public APIs
- All configuration formats
- All function signatures
- All interface contracts

---

## Verification

### Check Package Compliance

```go
// Packages are automatically compliant after alignment
// No action needed from users
```

### Verify Backward Compatibility

```go
// Your existing code should work exactly as before
// If you encounter any issues, please report them
```

---

## Examples

### Complete Example: Using New Providers

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/llms"
    _ "github.com/lookatitude/beluga-ai/pkg/llms/providers/grok"
    _ "github.com/lookatitude/beluga-ai/pkg/llms/providers/gemini"
)

func main() {
    ctx := context.Background()
    
    // Use Grok provider
    grokConfig := llms.Config{
        ProviderName: "grok",
        APIKey:       "your-grok-key",
    }
    grokLLM, err := llms.NewProvider(ctx, "grok", grokConfig)
    if err != nil {
        log.Fatal(err)
    }
    
    response, err := grokLLM.Generate(ctx, "Explain AI")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Grok:", response)
    
    // Use Gemini provider
    geminiConfig := llms.Config{
        ProviderName: "gemini",
        APIKey:       "your-gemini-key",
    }
    geminiLLM, err := llms.NewProvider(ctx, "gemini", geminiConfig)
    if err != nil {
        log.Fatal(err)
    }
    
    response, err = geminiLLM.Generate(ctx, "What is machine learning?")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Gemini:", response)
}
```

### Complete Example: Multimodal Workflow

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    
    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

func main() {
    ctx := context.Background()
    
    // Create image message
    imageData, _ := os.ReadFile("image.jpg")
    imgMsg := schema.NewImageMessage(imageData, "jpeg")
    imgMsg.Caption = "A cat playing with a ball"
    
    // Embed image
    embedder, _ := embeddings.NewProvider(ctx, "openai", embeddings.Config{})
    embedding, err := embedder.Embed(ctx, []schema.Document{imgMsg})
    if err != nil {
        log.Fatal(err)
    }
    
    // Store in vector store
    store, _ := vectorstores.NewProvider(ctx, "pgvector", vectorstores.Config{})
    err = store.AddDocuments(ctx, []schema.Document{imgMsg}, embedding)
    if err != nil {
        log.Fatal(err)
    }
    
    // Search for similar images
    queryImg := schema.NewImageMessage(queryImageData, "jpeg")
    queryEmbedding, _ := embedder.Embed(ctx, []schema.Document{queryImg})
    results, err := store.SimilaritySearch(ctx, queryEmbedding[0], 5)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d similar images\n", len(results))
}
```

---

## Support

### Getting Help

- **Documentation**: See `docs/package_design_patterns.md` for framework patterns
- **Examples**: See `examples/` directory for more examples
- **Issues**: Report issues on GitHub

### Reporting Problems

If you encounter any issues with v2 alignment:
1. Verify you're using the latest framework version
2. Check that your code follows framework patterns
3. Report the issue with:
   - Package name
   - Error message
   - Code example (if possible)

---

**Status**: Quickstart guide complete, ready for users.
