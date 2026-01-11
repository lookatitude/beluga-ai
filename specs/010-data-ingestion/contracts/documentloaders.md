# API Contract: pkg/documentloaders

**Feature**: 010-data-ingestion  
**Date**: 2026-01-11  
**Package**: `github.com/lookatitude/beluga-ai/pkg/documentloaders`

## Interfaces

### DocumentLoader (iface/loader.go)

```go
package iface

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

// DocumentLoader defines the interface for loading documents from various sources.
// Implementations should be stateless and thread-safe.
type DocumentLoader interface {
    // Load reads documents from the configured source.
    // Returns all successfully loaded documents and an error if any failures occurred.
    // Implementations MUST respect context cancellation.
    // Implementations MUST populate Document.Metadata with at least "source" key.
    Load(ctx context.Context) ([]schema.Document, error)
}
```

## Configuration

### LoaderConfig (config.go)

```go
package documentloaders

// LoaderConfig contains common configuration for all loaders.
type LoaderConfig struct {
    // MaxFileSize is the maximum file size in bytes (default: 100MB).
    MaxFileSize int64 `mapstructure:"max_file_size" yaml:"max_file_size" env:"LOADER_MAX_FILE_SIZE" validate:"min=1"`
}

// DefaultLoaderConfig returns a LoaderConfig with sensible defaults.
func DefaultLoaderConfig() *LoaderConfig {
    return &LoaderConfig{
        MaxFileSize: 100 * 1024 * 1024, // 100MB
    }
}
```

### DirectoryConfig (config.go)

```go
package documentloaders

// DirectoryConfig contains configuration for RecursiveDirectoryLoader.
type DirectoryConfig struct {
    LoaderConfig `mapstructure:",squash"`

    // Path is the root directory to load from (required).
    Path string `mapstructure:"path" yaml:"path" env:"LOADER_PATH" validate:"required"`

    // MaxDepth limits recursion depth (default: 10, 0 = root only).
    MaxDepth int `mapstructure:"max_depth" yaml:"max_depth" env:"LOADER_MAX_DEPTH" validate:"min=0"`

    // Extensions filters files by extension (nil = all files).
    // Extensions should include the dot (e.g., ".txt", ".md").
    Extensions []string `mapstructure:"extensions" yaml:"extensions" env:"LOADER_EXTENSIONS"`

    // Concurrency sets the number of concurrent file readers (default: GOMAXPROCS).
    Concurrency int `mapstructure:"concurrency" yaml:"concurrency" env:"LOADER_CONCURRENCY" validate:"min=1"`

    // FollowSymlinks enables following symbolic links (default: true).
    FollowSymlinks bool `mapstructure:"follow_symlinks" yaml:"follow_symlinks" env:"LOADER_FOLLOW_SYMLINKS"`
}

// DefaultDirectoryConfig returns a DirectoryConfig with sensible defaults.
func DefaultDirectoryConfig() *DirectoryConfig {
    return &DirectoryConfig{
        LoaderConfig:   *DefaultLoaderConfig(),
        MaxDepth:       10,
        Concurrency:    runtime.GOMAXPROCS(0),
        FollowSymlinks: true,
    }
}
```

## Functional Options

```go
package documentloaders

// Option configures a loader.
type Option func(*LoaderConfig)

// WithMaxFileSize sets the maximum file size.
func WithMaxFileSize(size int64) Option {
    return func(c *LoaderConfig) {
        c.MaxFileSize = size
    }
}

// DirectoryOption configures a directory loader.
type DirectoryOption func(*DirectoryConfig)

// WithMaxDepth sets the maximum recursion depth.
func WithMaxDepth(depth int) DirectoryOption {
    return func(c *DirectoryConfig) {
        c.MaxDepth = depth
    }
}

// WithExtensions sets the file extension filter.
func WithExtensions(exts ...string) DirectoryOption {
    return func(c *DirectoryConfig) {
        c.Extensions = exts
    }
}

// WithConcurrency sets the number of concurrent workers.
func WithConcurrency(n int) DirectoryOption {
    return func(c *DirectoryConfig) {
        c.Concurrency = n
    }
}

// WithFollowSymlinks enables or disables symlink following.
func WithFollowSymlinks(follow bool) DirectoryOption {
    return func(c *DirectoryConfig) {
        c.FollowSymlinks = follow
    }
}
```

## Factory Functions

### NewTextLoader

```go
// NewTextLoader creates a loader for a single text file.
// Returns an error if the file path is invalid or inaccessible.
func NewTextLoader(path string, opts ...Option) (iface.DocumentLoader, error)
```

**Parameters**:
- `path`: Path to the text file (required)
- `opts`: Optional configuration options

**Returns**:
- `DocumentLoader`: Configured text loader
- `error`: `LoaderError` if configuration is invalid

**Example**:
```go
loader, err := documentloaders.NewTextLoader("/path/to/file.txt")
if err != nil {
    log.Fatal(err)
}
docs, err := loader.Load(ctx)
```

### NewDirectoryLoader

```go
// NewDirectoryLoader creates a loader that recursively reads files from a directory.
// The loader uses the provided file system abstraction for testability.
func NewDirectoryLoader(fsys fs.FS, opts ...DirectoryOption) (iface.DocumentLoader, error)
```

**Parameters**:
- `fsys`: File system to read from (use `os.DirFS(path)` for real file system)
- `opts`: Configuration options

**Returns**:
- `DocumentLoader`: Configured directory loader
- `error`: `LoaderError` if configuration is invalid

**Example**:
```go
loader, err := documentloaders.NewDirectoryLoader(
    os.DirFS("/data/docs"),
    documentloaders.WithMaxDepth(5),
    documentloaders.WithExtensions(".txt", ".md"),
    documentloaders.WithConcurrency(4),
)
if err != nil {
    log.Fatal(err)
}
docs, err := loader.Load(ctx)
```

## Registry API

```go
package documentloaders

// LoaderFactory creates a DocumentLoader from configuration.
type LoaderFactory func(config map[string]any) (iface.DocumentLoader, error)

// Registry manages loader provider registration.
type Registry struct {
    // ... internal fields
}

// GetRegistry returns the global registry singleton.
func GetRegistry() *Registry

// Register adds a loader factory to the registry.
// Panics if name is already registered (call during init only).
func (r *Registry) Register(name string, factory LoaderFactory)

// Create instantiates a loader by name with the given configuration.
// Returns ErrCodeNotFound if the loader is not registered.
func (r *Registry) Create(name string, config map[string]any) (iface.DocumentLoader, error)

// List returns all registered loader names.
func (r *Registry) List() []string

// IsRegistered checks if a loader name is registered.
func (r *Registry) IsRegistered(name string) bool
```

**Registered Providers** (via init.go):
- `"text"` - TextLoader for single files
- `"directory"` - RecursiveDirectoryLoader for directories

## Error Types

### LoaderError (errors.go)

```go
package documentloaders

// Error codes for loader operations.
const (
    ErrCodeIOError       = "io_error"
    ErrCodeNotFound      = "not_found"
    ErrCodeInvalidConfig = "invalid_config"
    ErrCodeCycleDetected = "cycle_detected"
    ErrCodeFileTooLarge  = "file_too_large"
    ErrCodeBinaryFile    = "binary_file"
    ErrCodeCancelled     = "cancelled"
)

// LoaderError represents an error during document loading.
type LoaderError struct {
    Op      string // Operation (e.g., "Load", "ReadFile")
    Code    string // Error code for programmatic handling
    Path    string // File path if applicable
    Message string // Human-readable message
    Err     error  // Underlying error
}

func (e *LoaderError) Error() string
func (e *LoaderError) Unwrap() error

// NewLoaderError creates a new LoaderError.
func NewLoaderError(op, code, path, message string, err error) *LoaderError

// IsLoaderError checks if err is a LoaderError.
func IsLoaderError(err error) bool

// GetLoaderError extracts LoaderError from err if present.
func GetLoaderError(err error) *LoaderError
```

## Metrics (metrics.go)

```go
package documentloaders

// Metrics for document loading operations.
// Initialized via NewMetrics() and registered with OTEL meter.

// Counters:
// - documentloaders_operations_total{loader_type, status}
// - documentloaders_documents_loaded{loader_type}
// - documentloaders_files_skipped{loader_type, reason}

// Histograms:
// - documentloaders_load_duration_seconds{loader_type}
// - documentloaders_file_size_bytes{loader_type}
```

## Tracing

All `Load()` operations create spans with:
- Span name: `documentloaders.{loader_type}.Load`
- Attributes:
  - `loader.type`: Loader name (e.g., "directory")
  - `loader.path`: Source path
  - `loader.documents_count`: Number of documents loaded
  - `loader.files_skipped`: Number of files skipped
  - `loader.duration_ms`: Operation duration

Errors are recorded via `span.RecordError()` with status `codes.Error`.
