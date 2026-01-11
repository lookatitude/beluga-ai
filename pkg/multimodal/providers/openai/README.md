# OpenAI Multimodal Provider

This package provides OpenAI provider implementation for multimodal models.

## Known Limitation: Import Cycle

Due to the current architecture, providers that implement the `MultimodalModel` interface must import the `multimodal` package (because the interface uses multimodal types). This creates an import cycle when providers try to auto-register via `init()`.

**Workaround**: Providers are registered manually or via a deferred registration mechanism. The `init()` function in this package uses reflection to avoid importing `multimodal` during initialization.

## Usage

```go
import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/multimodal"
    "github.com/lookatitude/beluga-ai/pkg/multimodal/providers/openai"
)

// Create OpenAI provider
config := multimodal.Config{
    Provider: "openai",
    Model:    "gpt-4o",
    APIKey:   "your-api-key",
}

// Register manually (if auto-registration doesn't work due to import cycle)
multimodal.GetRegistry().Register("openai", func(ctx context.Context, cfg multimodal.Config) (multimodal.iface.MultimodalModel, error) {
    openaiConfig := openai.FromMultimodalConfig(cfg)
    return openai.NewOpenAIProvider(openaiConfig)
})

// Use the provider
model, err := multimodal.NewMultimodalModel(ctx, "openai", config)
```

## Future Improvement

A separate `multimodal/registry` package (similar to `embeddings/registry`) should be created to break the import cycle and allow proper auto-registration.
