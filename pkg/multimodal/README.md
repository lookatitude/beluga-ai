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

## Available Providers

The multimodal package supports the following providers:

### Commercial Providers
- **OpenAI** (`openai`): GPT-4o, GPT-4 Vision - Full multimodal support
- **Google Gemini** (`gemini`): Gemini Pro, Gemini Ultra - Text, image, audio, video
- **Google Vertex AI** (`google`): Enterprise Google AI Platform - Full multimodal support
- **Anthropic** (`anthropic`): Claude 3 Opus, Sonnet - Text and image support
- **xAI** (`xai`): Grok models - Text and image support

### Open-Source Providers
- **Qwen** (`qwen`): Alibaba's Qwen models - Full multimodal support
- **Pixtral** (`pixtral`): Mistral AI's Pixtral models - Text, image, audio, video
- **Phi** (`phi`): Microsoft's Phi models via Hugging Face - Full multimodal support
- **DeepSeek** (`deepseek`): DeepSeek models - OpenAI-compatible API
- **Gemma** (`gemma`): Google's Gemma models - Gemini-compatible API

Each provider has its own README with detailed configuration and usage examples in `pkg/multimodal/providers/{provider}/README.md`.

### Provider-Specific Configuration Examples

#### OpenAI Configuration

```go
config := multimodal.Config{
    Provider: "openai",
    Model:    "gpt-4o",
    APIKey:   os.Getenv("OPENAI_API_KEY"),
    Timeout:  30 * time.Second,
    MaxRetries: 3,
    ProviderSpecific: map[string]any{
        "temperature": 0.7,
        "max_tokens": 4096,
        "vision_detail": "high", // "low" or "high"
    },
}
model, err := multimodal.NewMultimodalModel(ctx, "openai", config)
```

#### Google Gemini Configuration

```go
config := multimodal.Config{
    Provider: "gemini",
    Model:    "gemini-pro",
    APIKey:   os.Getenv("GEMINI_API_KEY"),
    ProviderSpecific: map[string]any{
        "temperature": 0.9,
        "top_p": 0.95,
        "top_k": 40,
    },
}
model, err := multimodal.NewMultimodalModel(ctx, "gemini", config)
```

#### Anthropic Configuration

```go
config := multimodal.Config{
    Provider: "anthropic",
    Model:    "claude-3-opus-20240229",
    APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
    ProviderSpecific: map[string]any{
        "max_tokens": 4096,
        "temperature": 0.7,
    },
}
model, err := multimodal.NewMultimodalModel(ctx, "anthropic", config)
```

#### Pixtral (Mistral AI) Configuration

```go
config := multimodal.Config{
    Provider: "pixtral",
    Model:    "pixtral-12b",
    APIKey:   os.Getenv("PIXTRAL_API_KEY"),
    BaseURL:  "https://api.mistral.ai/v1",
    Timeout:  30 * time.Second,
}
model, err := multimodal.NewMultimodalModel(ctx, "pixtral", config)
```

For more provider-specific examples and configuration options, see the provider READMEs in `pkg/multimodal/providers/{provider}/README.md`.

## Advanced Examples

### Health Checks

```go
// Check if model is healthy
err := model.CheckHealth(ctx)
if err != nil {
    log.Printf("Model health check failed: %v", err)
}
```

### Error Handling with Retry Logic

```go
output, err := model.Process(ctx, input)
if err != nil {
    if multimodal.IsRetryableError(err) {
        // Retry the operation
        time.Sleep(1 * time.Second)
        output, err = model.Process(ctx, input)
    }
    
    if err != nil {
        code := multimodal.GetErrorCode(err)
        switch code {
        case multimodal.ErrCodeRateLimit:
            // Handle rate limit
        case multimodal.ErrCodeQuotaExceeded:
            // Handle quota exceeded
        case multimodal.ErrCodeAuthenticationFailed:
            // Handle authentication error
        }
    }
}
```

### Streaming with Context Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

outputChan, err := model.ProcessStream(ctx, input)
if err != nil {
    log.Fatal(err)
}

for output := range outputChan {
    select {
    case <-ctx.Done():
        log.Println("Stream cancelled")
        return
    default:
        // Process output chunk
        for _, block := range output.ContentBlocks {
            data, _ := block.GetData()
            fmt.Printf("Chunk: %s\n", string(data))
        }
    }
}
```

### Provider-Specific Configuration

```go
// OpenAI with custom settings
openaiConfig := multimodal.Config{
    Provider: "openai",
    Model:    "gpt-4o",
    APIKey:   os.Getenv("OPENAI_API_KEY"),
    ProviderSpecific: map[string]any{
        "temperature": 0.7,
        "max_tokens":  4096,
        "vision_detail": "high",
    },
}

// Google Vertex AI with project settings
googleConfig := multimodal.Config{
    Provider: "google",
    Model:    "gemini-pro",
    ProviderSpecific: map[string]any{
        "project_id": "my-gcp-project",
        "location":   "us-central1",
    },
}
```

### Custom Routing Strategy

```go
input, _ := multimodal.NewMultimodalInput(blocks,
    multimodal.WithRouting(&multimodal.RoutingConfig{
        Strategy:      "manual",
        TextProvider:  "openai",
        ImageProvider: "google",
        AudioProvider: "anthropic",
        FallbackToText: true,
    }),
)
```

## Performance Considerations

- **Streaming**: Use `ProcessStream` for large files or real-time processing
- **Format Selection**: Use URLs for large files (>10MB), base64 for small files
- **Concurrent Processing**: Models are safe for concurrent use
- **Caching**: Cache provider capabilities to avoid repeated checks

## Best Practices

1. **Always check capabilities** before processing multimodal content
2. **Use context cancellation** for long-running operations
3. **Handle errors gracefully** with custom error types
4. **Monitor performance** with OTEL metrics
5. **Use appropriate formats** (base64 for small files, URLs for large files)
6. **Implement retry logic** for retryable errors
7. **Validate inputs** before processing

## Documentation

For more details, see:
- [Package Design Patterns](../../docs/package_design_patterns.md)
- [Multimodal Concepts](../../docs/concepts/multimodal.md)
- [Quickstart Guide](../../specs/009-multimodal/quickstart.md)
- Provider-specific READMEs in `pkg/multimodal/providers/{provider}/README.md`