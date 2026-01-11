# API Contract: pkg/textsplitters

**Feature**: 010-data-ingestion  
**Date**: 2026-01-11  
**Package**: `github.com/lookatitude/beluga-ai/pkg/textsplitters`

## Interfaces

### TextSplitter (iface/splitter.go)

```go
package iface

import (
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

// TextSplitter defines the interface for splitting text into chunks.
// Implementations should be stateless and thread-safe.
type TextSplitter interface {
    // SplitText splits a string into chunks based on the splitter's configuration.
    // Returns at least one chunk for non-empty input.
    SplitText(text string) ([]string, error)

    // SplitDocuments splits documents and returns new documents with chunk metadata.
    // Each output document inherits the source document's metadata plus:
    // - "chunk_index": 0-based index of this chunk
    // - "chunk_total": total number of chunks from source document
    SplitDocuments(docs []schema.Document) ([]schema.Document, error)
}
```

## Configuration

### SplitterConfig (config.go)

```go
package textsplitters

// SplitterConfig contains common configuration for all splitters.
type SplitterConfig struct {
    // ChunkSize is the target maximum chunk size (required).
    ChunkSize int `mapstructure:"chunk_size" yaml:"chunk_size" env:"SPLITTER_CHUNK_SIZE" validate:"required,min=1"`

    // ChunkOverlap is the number of characters/tokens to overlap between chunks.
    // Must be less than ChunkSize.
    ChunkOverlap int `mapstructure:"chunk_overlap" yaml:"chunk_overlap" env:"SPLITTER_CHUNK_OVERLAP" validate:"min=0,ltfield=ChunkSize"`

    // LengthFunction measures text length. Defaults to len() for character count.
    // Set to a tokenizer function for token-based splitting.
    LengthFunction func(string) int `mapstructure:"-" yaml:"-"`
}

// DefaultSplitterConfig returns a SplitterConfig with sensible defaults.
func DefaultSplitterConfig() *SplitterConfig {
    return &SplitterConfig{
        ChunkSize:      1000,
        ChunkOverlap:   200,
        LengthFunction: func(s string) int { return len(s) },
    }
}
```

### RecursiveConfig (config.go)

```go
package textsplitters

// RecursiveConfig contains configuration for RecursiveCharacterTextSplitter.
type RecursiveConfig struct {
    SplitterConfig `mapstructure:",squash"`

    // Separators defines the hierarchy of separators to try.
    // Default: ["\n\n", "\n", " ", ""]
    Separators []string `mapstructure:"separators" yaml:"separators" env:"SPLITTER_SEPARATORS"`
}

// DefaultRecursiveConfig returns a RecursiveConfig with sensible defaults.
func DefaultRecursiveConfig() *RecursiveConfig {
    return &RecursiveConfig{
        SplitterConfig: *DefaultSplitterConfig(),
        Separators:     []string{"\n\n", "\n", " ", ""},
    }
}
```

### MarkdownConfig (config.go)

```go
package textsplitters

// MarkdownConfig contains configuration for MarkdownTextSplitter.
type MarkdownConfig struct {
    SplitterConfig `mapstructure:",squash"`

    // HeadersToSplitOn defines which headers trigger splits.
    // Default: ["#", "##", "###", "####", "#####", "######"]
    HeadersToSplitOn []string `mapstructure:"headers_to_split_on" yaml:"headers_to_split_on"`

    // ReturnEachLine returns each line as a separate chunk if true.
    ReturnEachLine bool `mapstructure:"return_each_line" yaml:"return_each_line"`
}

// DefaultMarkdownConfig returns a MarkdownConfig with sensible defaults.
func DefaultMarkdownConfig() *MarkdownConfig {
    return &MarkdownConfig{
        SplitterConfig:   *DefaultSplitterConfig(),
        HeadersToSplitOn: []string{"#", "##", "###", "####", "#####", "######"},
        ReturnEachLine:   false,
    }
}
```

## Functional Options

```go
package textsplitters

// Option configures a splitter.
type Option func(*SplitterConfig)

// WithChunkSize sets the target chunk size.
func WithChunkSize(size int) Option {
    return func(c *SplitterConfig) {
        c.ChunkSize = size
    }
}

// WithChunkOverlap sets the overlap between consecutive chunks.
func WithChunkOverlap(overlap int) Option {
    return func(c *SplitterConfig) {
        c.ChunkOverlap = overlap
    }
}

// WithLengthFunction sets a custom length function.
// Use this for token-based splitting with LLM tokenizers.
func WithLengthFunction(fn func(string) int) Option {
    return func(c *SplitterConfig) {
        c.LengthFunction = fn
    }
}

// RecursiveOption configures a recursive splitter.
type RecursiveOption func(*RecursiveConfig)

// WithSeparators sets the separator hierarchy.
func WithSeparators(seps ...string) RecursiveOption {
    return func(c *RecursiveConfig) {
        c.Separators = seps
    }
}

// MarkdownOption configures a markdown splitter.
type MarkdownOption func(*MarkdownConfig)

// WithHeadersToSplitOn sets which markdown headers trigger splits.
func WithHeadersToSplitOn(headers ...string) MarkdownOption {
    return func(c *MarkdownConfig) {
        c.HeadersToSplitOn = headers
    }
}
```

## Factory Functions

### NewRecursiveCharacterTextSplitter

```go
// NewRecursiveCharacterTextSplitter creates a splitter that recursively splits
// text using a hierarchy of separators (paragraph, line, word, character).
func NewRecursiveCharacterTextSplitter(opts ...RecursiveOption) (iface.TextSplitter, error)
```

**Parameters**:
- `opts`: Configuration options (chunk size, overlap, separators)

**Returns**:
- `TextSplitter`: Configured recursive splitter
- `error`: `SplitterError` if configuration is invalid

**Example**:
```go
splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithChunkSize(1000),
    textsplitters.WithChunkOverlap(200),
)
if err != nil {
    log.Fatal(err)
}
chunks, err := splitter.SplitText(longText)
```

### NewMarkdownTextSplitter

```go
// NewMarkdownTextSplitter creates a splitter that respects markdown structure.
// Splits preferentially at header boundaries while respecting chunk size limits.
func NewMarkdownTextSplitter(opts ...MarkdownOption) (iface.TextSplitter, error)
```

**Parameters**:
- `opts`: Configuration options (chunk size, overlap, headers)

**Returns**:
- `TextSplitter`: Configured markdown splitter
- `error`: `SplitterError` if configuration is invalid

**Example**:
```go
splitter, err := textsplitters.NewMarkdownTextSplitter(
    textsplitters.WithChunkSize(1500),
    textsplitters.WithChunkOverlap(100),
    textsplitters.WithHeadersToSplitOn("##", "###"),
)
if err != nil {
    log.Fatal(err)
}
docs, err := splitter.SplitDocuments(loadedDocs)
```

## Registry API

```go
package textsplitters

// SplitterFactory creates a TextSplitter from configuration.
type SplitterFactory func(config map[string]any) (iface.TextSplitter, error)

// Registry manages splitter provider registration.
type Registry struct {
    // ... internal fields
}

// GetRegistry returns the global registry singleton.
func GetRegistry() *Registry

// Register adds a splitter factory to the registry.
// Panics if name is already registered (call during init only).
func (r *Registry) Register(name string, factory SplitterFactory)

// Create instantiates a splitter by name with the given configuration.
// Returns ErrCodeNotFound if the splitter is not registered.
func (r *Registry) Create(name string, config map[string]any) (iface.TextSplitter, error)

// List returns all registered splitter names.
func (r *Registry) List() []string

// IsRegistered checks if a splitter name is registered.
func (r *Registry) IsRegistered(name string) bool
```

**Registered Providers** (via init.go):
- `"recursive_character"` - RecursiveCharacterTextSplitter
- `"markdown"` - MarkdownTextSplitter

## Error Types

### SplitterError (errors.go)

```go
package textsplitters

// Error codes for splitter operations.
const (
    ErrCodeInvalidConfig = "invalid_config"
    ErrCodeEmptyInput    = "empty_input"
    ErrCodeNotFound      = "not_found"
    ErrCodeCancelled     = "cancelled"
)

// SplitterError represents an error during text splitting.
type SplitterError struct {
    Op      string // Operation (e.g., "SplitText", "SplitDocuments")
    Code    string // Error code for programmatic handling
    Message string // Human-readable message
    Err     error  // Underlying error
}

func (e *SplitterError) Error() string
func (e *SplitterError) Unwrap() error

// NewSplitterError creates a new SplitterError.
func NewSplitterError(op, code, message string, err error) *SplitterError

// IsSplitterError checks if err is a SplitterError.
func IsSplitterError(err error) bool

// GetSplitterError extracts SplitterError from err if present.
func GetSplitterError(err error) *SplitterError
```

## Metrics (metrics.go)

```go
package textsplitters

// Metrics for text splitting operations.
// Initialized via NewMetrics() and registered with OTEL meter.

// Counters:
// - textsplitters_operations_total{splitter_type, status}
// - textsplitters_chunks_created{splitter_type}
// - textsplitters_documents_processed{splitter_type}

// Histograms:
// - textsplitters_split_duration_seconds{splitter_type}
// - textsplitters_chunk_size{splitter_type}
// - textsplitters_chunks_per_document{splitter_type}
```

## Tracing

All `SplitText()` and `SplitDocuments()` operations create spans with:
- Span name: `textsplitters.{splitter_type}.SplitText` or `textsplitters.{splitter_type}.SplitDocuments`
- Attributes:
  - `splitter.type`: Splitter name (e.g., "recursive_character")
  - `splitter.chunk_size`: Configured chunk size
  - `splitter.chunk_overlap`: Configured overlap
  - `splitter.input_count`: Number of input documents (for SplitDocuments)
  - `splitter.output_count`: Number of output chunks
  - `splitter.duration_ms`: Operation duration

Errors are recorded via `span.RecordError()` with status `codes.Error`.

## Usage Patterns

### Basic RAG Pipeline

```go
// Load documents
loader, _ := documentloaders.NewDirectoryLoader(
    os.DirFS("/data/knowledge"),
    documentloaders.WithExtensions(".txt", ".md"),
)
docs, err := loader.Load(ctx)
if err != nil {
    log.Fatal(err)
}

// Split documents
splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithChunkSize(1000),
    textsplitters.WithChunkOverlap(200),
)
chunks, err := splitter.SplitDocuments(docs)
if err != nil {
    log.Fatal(err)
}

// Continue with embedding and storage...
embeddings := embeddings.NewOpenAIEmbedder(...)
vectorstore := vectorstores.NewPinecone(...)
```

### Token-Based Splitting

```go
// Use tiktoken or similar for accurate token counting
tokenizer := tiktoken.NewEncoder("cl100k_base")
countTokens := func(s string) int {
    tokens, _ := tokenizer.Encode(s, nil)
    return len(tokens)
}

splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithChunkSize(512),  // 512 tokens
    textsplitters.WithChunkOverlap(50), // 50 token overlap
    textsplitters.WithLengthFunction(countTokens),
)
```
