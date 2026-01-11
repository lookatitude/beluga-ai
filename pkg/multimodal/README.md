# Multimodal Package

The `multimodal` package provides a unified interface for multimodal model operations (text+image/video/audio) that integrates with existing LLM and embedding providers in the Beluga AI Framework.

## Overview

This package enables:
- Processing multimodal inputs (text+image, text+audio, text+video) through a unified interface
- Routing content to appropriate providers based on capabilities
- Multimodal reasoning and generation (visual question answering, image captioning, text-to-image)
- Multimodal RAG integration (retrieval-augmented generation with multimodal data)
- Real-time multimodal streaming (live video analysis, streaming audio/video processing)
- Extending agents with multimodal capabilities

## Features

- **Unified Interface**: Single API for all multimodal operations across providers
- **Provider Registry**: Global registry pattern matching framework standards
- **Capability Detection**: Automatic routing based on provider capabilities
- **Graceful Fallbacks**: Falls back to text-only when modality not supported
- **OTEL Observability**: Comprehensive metrics, tracing, and structured logging
- **Backward Compatible**: Maintains 100% compatibility with text-only workflows

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/multimodal"
)

func main() {
    ctx := context.Background()
    
    // Create configuration
    config := multimodal.Config{
        Provider: "openai",
        Model:    "gpt-4o",
        APIKey:   "your-api-key",
    }
    
    // Create a multimodal model
    model, err := multimodal.NewMultimodalModel(ctx, "openai", config)
    if err != nil {
        panic(err)
    }
    
    // Create multimodal input
    input := &multimodal.MultimodalInput{
        ContentBlocks: []*multimodal.ContentBlock{
            {
                Type: "text",
                Data: []byte("What's in this image?"),
            },
            {
                Type: "image",
                URL:  "https://example.com/image.png",
            },
        },
    }
    
    // Process input
    output, err := model.Process(ctx, input)
    if err != nil {
        panic(err)
    }
    
    // Use output
    fmt.Printf("Output: %+v\n", output)
}
```

## Provider Registration

Providers are automatically registered via `init()` functions in their respective packages:

```go
// pkg/multimodal/providers/openai/init.go
func init() {
    multimodal.GetRegistry().Register("openai", NewOpenAIModelFromConfig)
}
```

## Configuration

Configuration supports multiple formats via struct tags:

```go
config := multimodal.Config{
    Provider:       "openai",
    Model:          "gpt-4o",
    APIKey:         "your-api-key",
    Timeout:        30 * time.Second,
    MaxRetries:     3,
    EnableStreaming: true,
}
```

## Error Handling

The package uses custom error types with error codes:

```go
if err != nil {
    if mmErr, ok := multimodal.AsMultimodalError(err); ok {
        switch mmErr.Code {
        case multimodal.ErrCodeProviderNotFound:
            // Handle provider not found
        case multimodal.ErrCodeInvalidInput:
            // Handle invalid input
        }
    }
}
```

## Observability

All operations include OTEL metrics, tracing, and structured logging:

- Metrics: `multimodal_process_total`, `multimodal_process_duration_seconds`
- Tracing: Spans for all public methods with attributes
- Logging: Structured logging with OTEL trace/span IDs

## Testing

The package includes comprehensive test utilities:

```go
// Use mock implementations
mockModel := multimodal.NewMockMultimodalModel("openai", "gpt-4o", nil)
output, err := mockModel.Process(ctx, input)
```

## Integration

The package integrates with:
- **schema**: Uses existing multimodal message types (ImageMessage, VideoMessage, VoiceDocument)
- **embeddings**: Uses MultimodalEmbedder interface for multimodal embeddings
- **vectorstores**: Stores multimodal vectors for RAG workflows
- **agents**: Extends agents with multimodal capabilities
- **orchestration**: Supports multimodal processing in orchestration graphs

## Documentation

For more details, see:
- [Package Design Patterns](../../docs/package_design_patterns.md)
- [Multimodal Concepts](../../docs/concepts/multimodal.md)
- [Quickstart Guide](../../specs/009-multimodal/quickstart.md)
