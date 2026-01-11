# Quickstart: Multimodal Models Support

**Feature**: Multimodal Models Support  
**Date**: 2025-01-27  
**Status**: Complete

## Overview

This quickstart guide provides examples for using the multimodal models support package. It covers basic usage, provider configuration, streaming, RAG integration, and agent extensions.

## Installation

The multimodal package is part of the Beluga AI Framework. No additional installation is required.

```go
import "github.com/lookatitude/beluga-ai/pkg/multimodal"
```

## Basic Usage

### Creating a Multimodal Model

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"
    
    "github.com/lookatitude/beluga-ai/pkg/multimodal"
    "github.com/lookatitude/beluga-ai/pkg/multimodal/types"
)

func main() {
    ctx := context.Background()
    
    // Create a multimodal model
    model, err := multimodal.NewMultimodalModel(ctx, "openai", multimodal.Config{
        Provider: "openai",
        Model:    "gpt-4o",
        APIKey:   os.Getenv("OPENAI_API_KEY"),
        Timeout:  30 * time.Second,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Check capabilities
    caps, err := model.GetCapabilities(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Supports text: %v\n", caps.Text)
    fmt.Printf("Supports image: %v\n", caps.Image)
    fmt.Printf("Supports audio: %v\n", caps.Audio)
    fmt.Printf("Supports video: %v\n", caps.Video)
}
```

### Processing Multimodal Input

```go
// Create text content block
textBlock, err := multimodal.NewContentBlock("text", []byte("What's in this image?"))
if err != nil {
    log.Fatal(err)
}

// Create image content block from URL
imageBlock, err := multimodal.NewContentBlockFromURL(ctx, "image", "https://example.com/image.png")
if err != nil {
    log.Fatal(err)
}

// Create multimodal input
input, err := multimodal.NewMultimodalInput([]*multimodal.ContentBlock{textBlock, imageBlock})
if err != nil {
    log.Fatal(err)
}

// Process input
output, err := model.Process(ctx, input)
if err != nil {
    log.Fatal(err)
}

// Access results
for _, block := range output.ContentBlocks {
    fmt.Printf("Type: %s\n", block.Type)
    fmt.Printf("Content: %s\n", string(block.Data))
}
```

### Processing from File

```go
// Create image content block from file
imageBlock, err := multimodal.NewContentBlockFromFile(ctx, "image", "/path/to/image.jpg")
if err != nil {
    log.Fatal(err)
}

input, _ := multimodal.NewMultimodalInput([]*multimodal.ContentBlock{imageBlock})
output, err := model.Process(ctx, input)
```

## Streaming

### Streaming Multimodal Processing

```go
// Process with streaming
outputChan, err := model.ProcessStream(ctx, input)
if err != nil {
    log.Fatal(err)
}

// Receive incremental outputs
for output := range outputChan {
    for _, block := range output.ContentBlocks {
        fmt.Printf("Received chunk: %s\n", string(block.Data))
    }
}

// Check for errors
select {
case <-ctx.Done():
    fmt.Println("Context cancelled")
default:
    fmt.Println("Streaming completed")
}
```

## Provider Configuration

### Using Different Providers

```go
// OpenAI
openaiModel, _ := multimodal.NewMultimodalModel(ctx, "openai", multimodal.Config{
    Provider: "openai",
    Model:    "gpt-4o",
    APIKey:   os.Getenv("OPENAI_API_KEY"),
})

// Google Gemini
geminiModel, _ := multimodal.NewMultimodalModel(ctx, "gemini", multimodal.Config{
    Provider: "gemini",
    Model:    "gemini-pro",
    APIKey:   os.Getenv("GOOGLE_API_KEY"),
})

// Anthropic Claude
claudeModel, _ := multimodal.NewMultimodalModel(ctx, "anthropic", multimodal.Config{
    Provider: "anthropic",
    Model:    "claude-4-opus",
    APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
})
```

### Provider-Specific Configuration

```go
config := multimodal.Config{
    Provider: "openai",
    Model:    "gpt-4o",
    APIKey:   os.Getenv("OPENAI_API_KEY"),
    ProviderSpecific: map[string]any{
        "max_tokens": 4096,
        "temperature": 0.7,
        "vision_detail": "high",
    },
}
model, _ := multimodal.NewMultimodalModel(ctx, "openai", config)
```

## Multimodal RAG Integration

### Storing Multimodal Documents

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
    vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

// Create multimodal embedder
embedder, _ := embeddings.NewEmbedder(ctx, "openai", embeddings.Config{
    OpenAI: &embeddings.OpenAIConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "text-embedding-3-large",
    },
})

// Check if embedder supports multimodal
if multiEmbedder, ok := embedder.(embeddingsiface.MultimodalEmbedder); ok && multiEmbedder.SupportsMultimodal() {
    // Create document with image
    doc := schema.Document{
        PageContent: "A beautiful sunset over the ocean",
        Metadata: map[string]string{
            "image_url": "https://example.com/sunset.jpg",
            "image_type": "image/jpeg",
        },
    }
    
    // Generate multimodal embedding
    embedding, err := multiEmbedder.EmbedQueryMultimodal(ctx, doc)
    if err != nil {
        log.Fatal(err)
    }
    
    // Store in vector store
    store, _ := vectorstores.NewVectorStore(ctx, "qdrant", vectorstoresiface.Config{
        Embedder: embedder,
        // ... other config
    })
    
    doc.Embedding = embedding
    _, err = store.AddDocuments(ctx, []schema.Document{doc})
    if err != nil {
        log.Fatal(err)
    }
}
```

### Multimodal Search

```go
// Create query document with image
queryDoc := schema.Document{
    PageContent: "Find similar images",
    Metadata: map[string]string{
        "image_url": "https://example.com/query.jpg",
        "image_type": "image/jpeg",
    },
}

// Generate query embedding
queryEmbedding, _ := multiEmbedder.EmbedQueryMultimodal(ctx, queryDoc)

// Search
docs, scores, err := store.SimilaritySearch(ctx, queryEmbedding, 5)
if err != nil {
    log.Fatal(err)
}

// Access results
for i, doc := range docs {
    fmt.Printf("Result %d (score: %.2f): %s\n", i+1, scores[i], doc.PageContent)
    if imageURL, ok := doc.Metadata["image_url"]; ok {
        fmt.Printf("  Image: %s\n", imageURL)
    }
}
```

## Agent Integration

### Multimodal Agent

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/agents"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

// Create LLM that supports multimodal
config := llms.NewConfig(
    llms.WithProvider("openai"),
    llms.WithModelName("gpt-4o"),
    llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
)
llm, _ := llms.GetRegistry().GetLLM("openai", config)

// Create agent with multimodal support
agent, _ := agents.NewBaseAgent("multimodal-agent", llm, nil)

// Create multimodal message
imageMsg := schema.NewImageMessage("https://example.com/image.png", "What's in this image?")

// Process with agent
response, err := agent.Invoke(ctx, []schema.Message{imageMsg})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Agent response: %s\n", response)
```

### Multimodal ReAct Agent

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/agents"
    "github.com/lookatitude/beluga-ai/pkg/agents/tools"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

// Create ChatModel for ReAct agent
config := llms.NewConfig(
    llms.WithProvider("openai"),
    llms.WithModelName("gpt-4o"),
    llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
)
chatModel, _ := llms.GetRegistry().GetProvider("openai", config)

// ReAct agent automatically handles multimodal inputs
reactAgent, _ := agents.NewReActAgent("react-multimodal-agent", chatModel, []tools.Tool{
    // ... tools
}, nil)

// Process multimodal input
imageMsg := schema.NewImageMessage("https://example.com/image.png", "Analyze this image and describe what you see")

response, err := reactAgent.Invoke(ctx, []schema.Message{imageMsg})
```

## Error Handling

### Custom Error Types

```go
output, err := model.Process(ctx, input)
if err != nil {
    if multiErr, ok := multimodal.AsMultimodalError(err); ok {
        switch multiErr.Code {
        case multimodal.ErrCodeUnsupportedModality:
            fmt.Printf("Modality not supported: %s\n", multiErr.Message)
        case multimodal.ErrCodeProviderError:
            fmt.Printf("Provider error: %s\n", multiErr.Err)
        case multimodal.ErrCodeTimeout:
            fmt.Printf("Operation timed out\n")
        default:
            fmt.Printf("Error: %s\n", multiErr.Message)
        }
    } else {
        fmt.Printf("Unknown error: %v\n", err)
    }
}
```

## Advanced Usage

### Custom Routing

```go
// Note: Routing configuration is typically handled at the model level
// For manual routing, configure providers when creating the model
// The types package focuses on data structures, routing is handled by the multimodal package
input, _ := multimodal.NewMultimodalInput(blocks)
```

### Capability Detection

```go
// Check if model supports specific modality
supportsImage, _ := model.SupportsModality(ctx, "image")
if !supportsImage {
    fmt.Println("Image processing not supported, falling back to text")
}

// Get full capabilities
caps, _ := model.GetCapabilities(ctx)
fmt.Printf("Max image size: %d bytes\n", caps.MaxImageSize)
fmt.Printf("Supported formats: %v\n", caps.SupportedImageFormats)
```

### Provider Registry

```go
// List available providers
registry := multimodal.GetRegistry()
providers := registry.ListProviders()
fmt.Printf("Available providers: %v\n", providers)

// Check if provider is registered
if registry.IsRegistered("openai") {
    fmt.Println("OpenAI provider is available")
}
```

## Best Practices

1. **Always check capabilities** before processing multimodal content
2. **Use streaming** for large files or real-time processing
3. **Handle errors gracefully** with custom error types
4. **Use context cancellation** for long-running operations
5. **Validate inputs** before processing
6. **Monitor performance** with OTEL metrics
7. **Cache provider capabilities** to avoid repeated checks
8. **Use appropriate formats** (base64 for small files, URLs for large files)

## Next Steps

- See [data-model.md](./data-model.md) for entity definitions
- See [contracts/multimodal-api.md](./contracts/multimodal-api.md) for API specifications
- See package README for detailed documentation
- See examples/ directory for more examples
