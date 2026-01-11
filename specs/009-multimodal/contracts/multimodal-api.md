# Multimodal API Contract

**Feature**: Multimodal Models Support  
**Date**: 2025-01-27  
**Status**: Complete

## Overview

This document defines the API contract for the multimodal models support package. It specifies interfaces, methods, parameters, return values, and error handling.

## Package: `pkg/multimodal`

### Core Interfaces

#### MultimodalModel

Main interface for multimodal model operations.

```go
type MultimodalModel interface {
    // Process processes multimodal input and returns output
    Process(ctx context.Context, input MultimodalInput) (MultimodalOutput, error)
    
    // ProcessStream processes multimodal input with streaming results
    ProcessStream(ctx context.Context, input MultimodalInput) (<-chan MultimodalOutput, error)
    
    // GetCapabilities returns the capabilities of this model
    GetCapabilities(ctx context.Context) (ModalityCapabilities, error)
    
    // SupportsModality checks if the model supports a specific modality
    SupportsModality(ctx context.Context, modality string) (bool, error)
}
```

**Methods**:

1. **Process**
   - **Parameters**:
     - `ctx` (context.Context): Context for cancellation and timeout
     - `input` (MultimodalInput): Multimodal input to process
   - **Returns**:
     - `MultimodalOutput`: Generated output
     - `error`: Error if processing fails
   - **Errors**:
     - `ErrCodeInvalidInput`: Input validation failed
     - `ErrCodeProviderError`: Provider returned an error
     - `ErrCodeUnsupportedModality`: Modality not supported
     - `ErrCodeTimeout`: Operation timed out
     - `ErrCodeCancelled`: Operation was cancelled

2. **ProcessStream**
   - **Parameters**:
     - `ctx` (context.Context): Context for cancellation and timeout
     - `input` (MultimodalInput): Multimodal input to process
   - **Returns**:
     - `<-chan MultimodalOutput`: Channel of incremental outputs
     - `error`: Error if streaming setup fails
   - **Errors**: Same as Process

3. **GetCapabilities**
   - **Parameters**:
     - `ctx` (context.Context): Context for cancellation
   - **Returns**:
     - `ModalityCapabilities`: Model capabilities
     - `error`: Error if capabilities cannot be retrieved

4. **SupportsModality**
   - **Parameters**:
     - `ctx` (context.Context): Context for cancellation
     - `modality` (string): Modality to check ("text", "image", "audio", "video")
   - **Returns**:
     - `bool`: True if modality is supported
     - `error`: Error if check fails

#### MultimodalProvider

Interface for provider implementations.

```go
type MultimodalProvider interface {
    // CreateModel creates a new model instance
    CreateModel(ctx context.Context, config Config) (MultimodalModel, error)
    
    // GetName returns the provider name
    GetName() string
    
    // GetCapabilities returns provider capabilities
    GetCapabilities() ModalityCapabilities
    
    // ValidateConfig validates provider configuration
    ValidateConfig(config Config) error
}
```

**Methods**:

1. **CreateModel**
   - **Parameters**:
     - `ctx` (context.Context): Context for cancellation
     - `config` (Config): Provider configuration
   - **Returns**:
     - `MultimodalModel`: Model instance
     - `error`: Error if creation fails

2. **GetName**
   - **Returns**: `string`: Provider name

3. **GetCapabilities**
   - **Returns**: `ModalityCapabilities`: Provider capabilities

4. **ValidateConfig**
   - **Parameters**:
     - `config` (Config): Configuration to validate
   - **Returns**:
     - `error`: Error if validation fails

#### ContentBlock

Interface for content blocks.

```go
type ContentBlock interface {
    // GetType returns the content type
    GetType() string
    
    // GetData returns the content data
    GetData() ([]byte, error)
    
    // GetURL returns the content URL if available
    GetURL() (string, bool)
    
    // GetFilePath returns the file path if available
    GetFilePath() (string, bool)
    
    // GetMIMEType returns the MIME type
    GetMIMEType() string
    
    // GetSize returns the content size in bytes
    GetSize() int64
    
    // GetMetadata returns additional metadata
    GetMetadata() map[string]any
}
```

### Factory Functions

#### NewMultimodalModel

Creates a new multimodal model instance.

```go
func NewMultimodalModel(ctx context.Context, provider string, config Config) (MultimodalModel, error)
```

**Parameters**:
- `ctx` (context.Context): Context for cancellation
- `provider` (string): Provider name (e.g., "openai", "google")
- `config` (Config): Model configuration

**Returns**:
- `MultimodalModel`: Model instance
- `error`: Error if creation fails

**Errors**:
- `ErrCodeProviderNotFound`: Provider not registered
- `ErrCodeInvalidConfig`: Configuration invalid

#### NewMultimodalInput

Creates a new multimodal input.

```go
func NewMultimodalInput(blocks []ContentBlock, opts ...InputOption) (MultimodalInput, error)
```

**Parameters**:
- `blocks` ([]ContentBlock): Content blocks
- `opts` (...InputOption): Optional configuration

**Returns**:
- `MultimodalInput`: Input instance
- `error`: Error if creation fails

**Errors**:
- `ErrCodeInvalidInput`: Input validation failed

#### NewContentBlock

Creates a new content block.

```go
func NewContentBlock(blockType string, data []byte, opts ...BlockOption) (ContentBlock, error)
func NewContentBlockFromURL(blockType string, url string, opts ...BlockOption) (ContentBlock, error)
func NewContentBlockFromFile(blockType string, filePath string, opts ...BlockOption) (ContentBlock, error)
```

**Parameters**:
- `blockType` (string): Content type ("text", "image", "audio", "video")
- `data` ([]byte): Content data (for NewContentBlock)
- `url` (string): Content URL (for NewContentBlockFromURL)
- `filePath` (string): File path (for NewContentBlockFromFile)
- `opts` (...BlockOption): Optional configuration

**Returns**:
- `ContentBlock`: Content block instance
- `error`: Error if creation fails

**Errors**:
- `ErrCodeInvalidFormat`: Format validation failed
- `ErrCodeFileNotFound`: File not found (for file-based blocks)

### Registry Functions

#### GetRegistry

Returns the global provider registry.

```go
func GetRegistry() *Registry
```

**Returns**:
- `*Registry`: Global registry instance

#### Registry Methods

```go
type Registry struct {
    // Register registers a provider factory
    Register(name string, factory ProviderFactory) error
    
    // Create creates a model instance using a registered provider
    Create(ctx context.Context, name string, config Config) (MultimodalModel, error)
    
    // ListProviders returns all registered provider names
    ListProviders() []string
    
    // IsRegistered checks if a provider is registered
    IsRegistered(name string) bool
}
```

### Configuration

#### Config

Configuration struct for multimodal models.

```go
type Config struct {
    Provider        string         `mapstructure:"provider" yaml:"provider" validate:"required"`
    Model           string         `mapstructure:"model" yaml:"model" validate:"required"`
    APIKey          string         `mapstructure:"api_key" yaml:"api_key" validate:"required_unless=Provider mock"`
    BaseURL         string         `mapstructure:"base_url" yaml:"base_url"`
    Timeout         time.Duration  `mapstructure:"timeout" yaml:"timeout" default:"30s" validate:"min=1s,max=5m"`
    MaxRetries      int            `mapstructure:"max_retries" yaml:"max_retries" default:"3" validate:"gte=0,lte=10"`
    RetryDelay      time.Duration  `mapstructure:"retry_delay" yaml:"retry_delay" default:"1s" validate:"min=100ms,max=30s"`
    EnableStreaming bool           `mapstructure:"enable_streaming" yaml:"enable_streaming" default:"true"`
    StreamChunkSize int64          `mapstructure:"stream_chunk_size" yaml:"stream_chunk_size" default:"1048576" validate:"min=1024"`
    ProviderSpecific map[string]any `mapstructure:"provider_specific" yaml:"provider_specific"`
}
```

### Error Types

#### MultimodalError

Custom error type for multimodal operations.

```go
type MultimodalError struct {
    Op      string
    Err     error
    Code    string
    Message string
}
```

**Error Codes**:
- `ErrCodeProviderNotFound`: Provider not registered
- `ErrCodeInvalidConfig`: Configuration invalid
- `ErrCodeInvalidInput`: Input validation failed
- `ErrCodeInvalidFormat`: Format validation failed
- `ErrCodeProviderError`: Provider returned an error
- `ErrCodeUnsupportedModality`: Modality not supported
- `ErrCodeTimeout`: Operation timed out
- `ErrCodeCancelled`: Operation was cancelled
- `ErrCodeFileNotFound`: File not found

### Functional Options

#### InputOption

Options for creating multimodal inputs.

```go
type InputOption func(*MultimodalInput)

func WithRouting(routing RoutingConfig) InputOption
func WithMetadata(metadata map[string]any) InputOption
func WithFormat(format string) InputOption
```

#### BlockOption

Options for creating content blocks.

```go
type BlockOption func(*ContentBlock)

func WithMIMEType(mimeType string) BlockOption
func WithMetadata(metadata map[string]any) BlockOption
func WithFormat(format string) BlockOption
```

## Usage Examples

### Basic Usage

```go
// Create a multimodal model
model, err := multimodal.NewMultimodalModel(ctx, "openai", multimodal.Config{
    Provider: "openai",
    Model:    "gpt-4o",
    APIKey:   os.Getenv("OPENAI_API_KEY"),
})
if err != nil {
    log.Fatal(err)
}

// Create multimodal input
textBlock, _ := multimodal.NewContentBlock("text", []byte("Describe this image"))
imageBlock, _ := multimodal.NewContentBlockFromURL("image", "https://example.com/image.png")
input, _ := multimodal.NewMultimodalInput([]multimodal.ContentBlock{textBlock, imageBlock})

// Process input
output, err := model.Process(ctx, input)
if err != nil {
    log.Fatal(err)
}

// Access output
for _, block := range output.ContentBlocks {
    fmt.Printf("Type: %s, Content: %s\n", block.GetType(), block.GetData())
}
```

### Streaming Usage

```go
// Process with streaming
outputChan, err := model.ProcessStream(ctx, input)
if err != nil {
    log.Fatal(err)
}

// Receive incremental outputs
for output := range outputChan {
    for _, block := range output.ContentBlocks {
        fmt.Printf("Received: %s\n", block.GetData())
    }
}
```

### Provider Registration

```go
// In providers/openai/init.go
func init() {
    multimodal.GetRegistry().Register("openai", func(ctx context.Context, config multimodal.Config) (multimodal.MultimodalModel, error) {
        return NewOpenAIMultimodalModel(ctx, config)
    })
}
```

## Notes

- All methods respect context cancellation
- All methods include OTEL tracing and metrics
- Error handling uses custom error types with codes
- Configuration uses struct tags for validation
- Provider registration happens via init() functions
