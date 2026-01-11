# Research: Data Ingestion and Processing

**Feature**: 010-data-ingestion  
**Date**: 2026-01-11  
**Status**: Complete

## 1. Document Loading Patterns

### Decision: Use `io/fs.FS` abstraction for file system operations

**Rationale**: The Go 1.16+ `io/fs` package provides a standard abstraction for file system operations that:
- Enables testing with in-memory file systems (`fstest.MapFS`)
- Supports embedded file systems (`embed.FS`)
- Allows custom implementations (cloud storage, archives)
- Is the idiomatic Go approach for file abstraction

**Alternatives Considered**:
- Direct `os` package calls: Rejected because it makes testing difficult and doesn't support alternative backends
- Custom FileSystem interface: Rejected because `io/fs.FS` is the established standard

### Decision: Use worker pool pattern for concurrent loading

**Rationale**: A bounded worker pool with configurable concurrency (default `runtime.GOMAXPROCS(0)`) provides:
- Predictable resource usage
- Protection against file descriptor exhaustion
- Linear scalability on multi-core systems
- User-controllable parallelism

**Alternatives Considered**:
- Unbounded goroutines: Rejected due to risk of resource exhaustion with large directories
- Sequential loading: Rejected due to poor performance on multi-core systems
- Automatic scaling: Rejected due to complexity and unpredictable behavior

### Decision: Use `net/http.DetectContentType` for binary detection

**Rationale**: The standard library's content type detection (based on first 512 bytes) is:
- Battle-tested and reliable
- Zero external dependencies
- Fast (reads only 512 bytes)
- Sufficient for detecting binary vs text content

**Alternatives Considered**:
- Magic number libraries (e.g., `h2non/filetype`): Rejected to avoid external dependency for core functionality
- Extension-only detection: Rejected because it doesn't catch misnamed files
- Full file read: Rejected due to performance concerns

## 2. Text Splitting Strategies

### Decision: Implement recursive character splitting as primary strategy

**Rationale**: Recursive character splitting (as in Langchaingo) provides:
- Predictable chunk sizes with configurable overlap
- Natural break points at paragraph/sentence boundaries
- Fallback hierarchy ensures all content is chunked
- Simple mental model for users

**Separator Hierarchy** (default):
1. `"\n\n"` - Paragraph breaks (preferred)
2. `"\n"` - Line breaks
3. `" "` - Word boundaries
4. `""` - Character-level (fallback)

**Alternatives Considered**:
- Sentence-based splitting: Rejected as primary (too complex for v1, can add as provider)
- Token-based only: Rejected because it requires LLM tokenizer dependency
- Fixed-size only: Rejected because it ignores natural boundaries

### Decision: Support custom length functions for token counting

**Rationale**: Embedding models have token limits, not character limits. Allowing injection of length functions enables:
- Integration with LLM tokenizers (tiktoken, etc.)
- Accurate chunk sizing for specific models
- Default to character count for simplicity

**Implementation**: `WithLengthFunction(func(string) int)` functional option

**Alternatives Considered**:
- Built-in tokenizer: Rejected to avoid LLM package dependency in splitters
- Token count only: Rejected because character count is simpler for basic use

### Decision: Markdown splitter respects structure via header detection

**Rationale**: Markdown documents have semantic structure (headers, code blocks) that should influence splitting:
- Split preferentially at header boundaries
- Keep code blocks intact when possible
- Preserve markdown formatting in chunks

**Implementation**: Regex-based header detection with special handling for fenced code blocks

**Alternatives Considered**:
- Full markdown parser (goldmark): Rejected to avoid external dependency
- Treat as plain text: Rejected because it loses structural benefits

## 3. Registry Pattern Alignment

### Decision: Match `pkg/llms/registry.go` pattern exactly

**Rationale**: Consistency across Beluga packages is critical for developer experience and maintainability:
- `GetRegistry()` singleton with `sync.Once`
- `Register(name string, factory func)` for provider registration
- `Create(name string, config *Config)` for instantiation
- Auto-registration in `providers/*/init.go`

**Pattern Reference**: See `pkg/llms/registry.go` lines 1-157

**Alternatives Considered**:
- Different registry interface: Rejected for consistency
- No registry: Rejected because it prevents extensibility

## 4. Error Handling Strategy

### Decision: Custom error types with 7 error codes

**Rationale**: Structured errors enable programmatic handling and debugging:
- `LoaderError` for document loading failures
- `SplitterError` for text splitting failures
- Error codes map to specific failure modes
- `Unwrap()` for error chain inspection

**Error Codes**:
| Code | Description | Retryable |
|------|-------------|-----------|
| `io_error` | File read/write failure | Yes |
| `not_found` | File/loader not found | No |
| `invalid_config` | Configuration validation failure | No |
| `cycle_detected` | Symlink cycle in traversal | No |
| `file_too_large` | File exceeds MaxFileSize | No |
| `binary_file` | Binary content detected | No |
| `cancelled` | Context cancelled | No |

**Alternatives Considered**:
- Simple error wrapping: Rejected because it lacks programmatic handling
- Sentinel errors only: Rejected because they can't carry context

## 5. OTEL Instrumentation

### Decision: Comprehensive tracing and metrics

**Rationale**: Observability is a constitution requirement and critical for production debugging:

**Metrics** (in `metrics.go`):
- `documentloaders_operations_total` - Counter by loader_type, status
- `documentloaders_documents_loaded` - Counter by loader_type
- `documentloaders_load_duration_seconds` - Histogram by loader_type
- `textsplitters_operations_total` - Counter by splitter_type, status
- `textsplitters_chunks_created` - Counter by splitter_type
- `textsplitters_split_duration_seconds` - Histogram by splitter_type

**Tracing**:
- Span per `Load()` call with attributes: loader_type, file_count, duration
- Span per `SplitDocuments()` call with attributes: splitter_type, input_count, output_count
- Error recording with `span.RecordError()` and status codes

**Alternatives Considered**:
- Metrics only: Rejected because traces are essential for debugging pipelines
- External instrumentation: Rejected because built-in is more reliable

## 6. Symlink Handling

### Decision: Follow with cycle detection using inode tracking

**Rationale**: Following symlinks is the expected behavior, but cycles must be detected:
- Track visited inodes using `map[uint64]bool` (from `os.FileInfo.Sys().(*syscall.Stat_t).Ino`)
- Skip already-visited inodes with warning log
- Platform-portable fallback to path tracking on Windows

**Implementation**: Check inode before descending into directories; use `filepath.EvalSymlinks` to resolve

**Alternatives Considered**:
- Never follow symlinks: Rejected because it surprises users with missing files
- Follow without protection: Rejected due to infinite loop risk
- Path-based cycle detection: Rejected because inode-based is more robust

## 7. Configuration Validation

### Decision: Use `go-playground/validator` with struct tags

**Rationale**: Validation at construction time prevents runtime errors:
- `validate:"required"` for mandatory fields
- `validate:"min=1"` for minimum values
- `validate:"gt=0,ltfield=ChunkSize"` for ChunkOverlap
- Consistent with other Beluga packages

**Config Structs**:
```go
type DirectoryConfig struct {
    Path           string   `mapstructure:"path" validate:"required"`
    MaxDepth       int      `mapstructure:"max_depth" validate:"min=0" default:"10"`
    Extensions     []string `mapstructure:"extensions"`
    Concurrency    int      `mapstructure:"concurrency" validate:"min=1"`
    MaxFileSize    int64    `mapstructure:"max_file_size" validate:"min=1"`
    FollowSymlinks bool     `mapstructure:"follow_symlinks" default:"true"`
}

type SplitterConfig struct {
    ChunkSize    int      `mapstructure:"chunk_size" validate:"required,min=1"`
    ChunkOverlap int      `mapstructure:"chunk_overlap" validate:"min=0,ltfield=ChunkSize"`
    Separators   []string `mapstructure:"separators"`
}
```

**Alternatives Considered**:
- Manual validation: Rejected for consistency and completeness
- No validation: Rejected because it leads to cryptic runtime errors

## 8. External Dependencies Assessment

### Decision: Minimize external dependencies for v1

**Rationale**: Core functionality should have minimal dependencies:

| Dependency | Status | Justification |
|------------|--------|---------------|
| `go-playground/validator` | ✅ Required | Already used in Beluga, essential for config validation |
| `otel/*` | ✅ Required | Constitution requirement for observability |
| PDF libraries | ❌ Deferred | External provider, not core functionality |
| HTML parsers | ❌ Deferred | External provider, not core functionality |

**Future Providers** (separate PRs):
- PDFLoader using `github.com/ledongthuc/pdf` or `github.com/unidoc/unipdf`
- HTMLLoader using `golang.org/x/net/html`
- URLLoader using `net/http`

## Summary

All research items resolved. No NEEDS CLARIFICATION markers remain. Ready for Phase 1 design.
