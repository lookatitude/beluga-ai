# Provider Package

The provider package provides unified provider discovery utilities. It offers a simplified interface for listing and discovering available providers across the Beluga AI Framework.

## Features

- **Provider Discovery**: List available providers by type
- **Unified API**: Consistent interface across all provider types
- **Provider Information**: Access provider metadata and descriptions

## Usage

### Listing Providers

```go
import "github.com/lookatitude/beluga-ai/pkg/convenience/provider"

// List available LLM providers
llmProviders := provider.ListLLMs()
fmt.Println("Available LLMs:", llmProviders)

// List available STT providers
sttProviders := provider.ListSTTs()
fmt.Println("Available STT providers:", sttProviders)

// List available TTS providers
ttsProviders := provider.ListTTSs()
fmt.Println("Available TTS providers:", ttsProviders)
```

### Getting All Providers

```go
// Get all providers organized by type
allProviders := provider.GetAllProviders()

for providerType, infos := range allProviders {
    fmt.Printf("%s providers:\n", providerType)
    for _, info := range infos {
        fmt.Printf("  - %s: %s\n", info.Name, info.Description)
    }
}
```

## Types

### ProviderInfo

```go
type ProviderInfo struct {
    Name        string
    Type        string
    Description string
}
```

## Creating Provider Instances

For creating actual provider instances, use the respective packages directly:

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "github.com/lookatitude/beluga-ai/pkg/stt"
    "github.com/lookatitude/beluga-ai/pkg/tts"
)

// Create LLM provider
llm, err := llms.NewProvider(ctx, "openai", llmConfig)

// Create embedding provider
embedder, err := embeddings.NewProvider(ctx, "openai", embeddingConfig)

// Create STT provider
speechToText, err := stt.NewProvider(ctx, "deepgram", sttConfig)

// Create TTS provider
textToSpeech, err := tts.NewProvider(ctx, "elevenlabs", ttsConfig)
```

## Provider Types

The package supports the following provider categories:

- **LLM Providers**: Language model providers (OpenAI, Anthropic, etc.)
- **STT Providers**: Speech-to-text providers (Deepgram, etc.)
- **TTS Providers**: Text-to-speech providers (ElevenLabs, etc.)
