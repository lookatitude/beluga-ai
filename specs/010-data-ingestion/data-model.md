# Data Model: Data Ingestion and Processing

**Feature**: 010-data-ingestion  
**Date**: 2026-01-11

## Entity Overview

```
┌─────────────────────┐     ┌─────────────────────┐
│   DocumentLoader    │     │    TextSplitter     │
│   (interface)       │     │    (interface)      │
├─────────────────────┤     ├─────────────────────┤
│ + Load(ctx) []Doc   │     │ + SplitText(s) []s  │
└─────────────────────┘     │ + SplitDocs() []Doc │
         △                  └─────────────────────┘
         │                           △
    ┌────┴────┐                 ┌────┴────┐
    │         │                 │         │
┌───┴───┐ ┌───┴───────┐   ┌────┴────┐ ┌───┴──────┐
│ Text  │ │ Recursive │   │Recursive│ │ Markdown │
│Loader │ │ Directory │   │Character│ │ Splitter │
└───────┘ │  Loader   │   │Splitter │ └──────────┘
          └───────────┘   └─────────┘
                │
                ▼
         ┌──────────────┐
         │   Document   │ (from pkg/schema)
         ├──────────────┤
         │ ID           │
         │ PageContent  │
         │ Metadata     │
         │ Embedding    │
         │ Score        │
         └──────────────┘
```

## Entities

### 1. Document (Existing - pkg/schema/internal/document.go)

Represents a loaded document with content and metadata. **No modifications required**.

| Field | Type | Description | Validation |
|-------|------|-------------|------------|
| `ID` | `string` | Optional unique identifier | Optional |
| `PageContent` | `string` | Text content of the document | Required (non-empty after load) |
| `Metadata` | `map[string]string` | Source information (path, size, etc.) | Required |
| `Embedding` | `[]float32` | Optional embedding vector | Optional (set by embeddings package) |
| `Score` | `float32` | Optional relevance score | Optional (set by retriever) |

**Metadata Keys** (populated by loaders):
- `source` - File path or URL
- `file_size` - Size in bytes
- `modified_at` - Last modification timestamp (RFC3339)
- `loader_type` - Loader that created this document
- `chunk_index` - Index when document is a chunk (set by splitter)
- `chunk_total` - Total chunks from source (set by splitter)

### 2. DocumentLoader (Interface - pkg/documentloaders/iface/loader.go)

Abstraction for loading documents from any source. Implements `core.Loader` interface.

```go
type DocumentLoader interface {
    Load(ctx context.Context) ([]schema.Document, error)
    LazyLoad(ctx context.Context) (<-chan any, error)
}
```

| Method | Input | Output | Description |
|--------|-------|--------|-------------|
| `Load` | `context.Context` | `([]Document, error)` | Load all documents from configured source |
| `LazyLoad` | `context.Context` | `(<-chan any, error)` | Load documents incrementally via channel (yields Document or error) |

**Contract**:
- MUST respect context cancellation
- MUST populate `PageContent` with non-empty content for each document
- MUST populate `Metadata` with at least `source` key
- MAY return partial results on error (documents loaded before failure)
- MUST return `LoaderError` on failure
- `LazyLoad()` MUST send errors on channel when encountered during streaming

### 3. TextSplitter (Interface - pkg/textsplitters/iface/splitter.go)

Abstraction for splitting text into chunks. Implements `retrievers.iface.Splitter` interface.

```go
type TextSplitter interface {
    SplitText(ctx context.Context, text string) ([]string, error)
    SplitDocuments(ctx context.Context, docs []schema.Document) ([]schema.Document, error)
    CreateDocuments(ctx context.Context, texts []string, metadatas []map[string]any) ([]schema.Document, error)
}
```

| Method | Input | Output | Description |
|--------|-------|--------|-------------|
| `SplitText` | `context.Context, string` | `([]string, error)` | Split text into chunks |
| `SplitDocuments` | `context.Context, []Document` | `([]Document, error)` | Split documents, preserving metadata |
| `CreateDocuments` | `context.Context, []string, []map[string]any` | `([]Document, error)` | Create documents from texts and metadatas, then split |

**Contract**:
- MUST respect context cancellation
- MUST produce chunks ≤ ChunkSize (by configured length function)
- MUST preserve consecutive chunk overlap = ChunkOverlap
- MUST copy and extend metadata on split documents (add chunk_index, chunk_total)
- MUST return single-element slice if input < ChunkSize
- `CreateDocuments()` MUST create Document objects with provided metadata before splitting
- MUST return `SplitterError` on failure

### 4. LoaderError (pkg/documentloaders/errors.go)

Custom error type for loader operations.

| Field | Type | Description |
|-------|------|-------------|
| `Op` | `string` | Operation that failed (e.g., "Load", "ReadFile") |
| `Err` | `error` | Underlying error |
| `Code` | `string` | Error code for programmatic handling |
| `Path` | `string` | File path if applicable |
| `Message` | `string` | Human-readable message |

**Error Codes**:
| Code | Constant | Description |
|------|----------|-------------|
| `io_error` | `ErrCodeIOError` | File system operation failed |
| `not_found` | `ErrCodeNotFound` | File or loader not found |
| `invalid_config` | `ErrCodeInvalidConfig` | Configuration validation failed |
| `cycle_detected` | `ErrCodeCycleDetected` | Symlink cycle detected |
| `file_too_large` | `ErrCodeFileTooLarge` | File exceeds MaxFileSize |
| `binary_file` | `ErrCodeBinaryFile` | Binary content detected |
| `cancelled` | `ErrCodeCancelled` | Context cancelled |

### 5. SplitterError (pkg/textsplitters/errors.go)

Custom error type for splitter operations.

| Field | Type | Description |
|-------|------|-------------|
| `Op` | `string` | Operation that failed (e.g., "SplitText") |
| `Err` | `error` | Underlying error |
| `Code` | `string` | Error code for programmatic handling |
| `Message` | `string` | Human-readable message |

**Error Codes**:
| Code | Constant | Description |
|------|----------|-------------|
| `invalid_config` | `ErrCodeInvalidConfig` | Configuration validation failed |
| `empty_input` | `ErrCodeEmptyInput` | Empty text provided |
| `cancelled` | `ErrCodeCancelled` | Context cancelled |

### 6. DirectoryConfig (pkg/documentloaders/config.go)

Configuration for RecursiveDirectoryLoader.

| Field | Type | Default | Validation | Description |
|-------|------|---------|------------|-------------|
| `Path` | `string` | - | `required` | Root directory path |
| `MaxDepth` | `int` | `10` | `min=0` | Maximum recursion depth |
| `Extensions` | `[]string` | `nil` | - | File extensions to include (nil = all) |
| `Concurrency` | `int` | `GOMAXPROCS` | `min=1` | Number of concurrent workers |
| `MaxFileSize` | `int64` | `104857600` (100MB) | `min=1` | Maximum file size in bytes |
| `FollowSymlinks` | `bool` | `true` | - | Whether to follow symbolic links |

### 7. SplitterConfig (pkg/textsplitters/config.go)

Configuration for text splitters.

| Field | Type | Default | Validation | Description |
|-------|------|---------|------------|-------------|
| `ChunkSize` | `int` | - | `required,min=1` | Target chunk size |
| `ChunkOverlap` | `int` | `0` | `min=0,ltfield=ChunkSize` | Overlap between chunks |
| `Separators` | `[]string` | `["\n\n","\n"," ",""]` | - | Separator hierarchy |
| `LengthFunction` | `func(string) int` | `len` | - | Function to measure length |

### 8. Registry (pkg/documentloaders/registry.go, pkg/textsplitters/registry.go)

Global registry for provider management.

| Method | Signature | Description |
|--------|-----------|-------------|
| `GetRegistry` | `() *Registry` | Get singleton registry instance |
| `Register` | `(name string, factory LoaderFactory)` | Register a loader factory |
| `Create` | `(name string, config *Config) (Loader, error)` | Create loader instance |
| `List` | `() []string` | List registered provider names |
| `IsRegistered` | `(name string) bool` | Check if provider exists |

**Factory Types**:
```go
type LoaderFactory func(*LoaderConfig) (DocumentLoader, error)
type SplitterFactory func(*SplitterConfig) (TextSplitter, error)
```

## State Transitions

### Document Lifecycle

```
[Source File] --Load()--> [Document] --SplitDocuments()--> [Chunked Documents]
                              │                                    │
                              │                                    │
                              ▼                                    ▼
                    Metadata: {source}              Metadata: {source, chunk_index}
                              │                                    │
                              └───────────> Embed() ────> VectorStore.Add()
```

### Loader State

No persistent state. Loaders are stateless—configuration is immutable after construction.

### Splitter State

No persistent state. Splitters are stateless—configuration is immutable after construction.

## Relationships

```
Registry 1 ──────────────────────────────────> * LoaderFactory
    │
    │ creates
    ▼
DocumentLoader 1 ────── produces ──────────> * Document
    │
    │ (user chains)
    ▼
TextSplitter 1 ──────── produces ──────────> * Document (chunked)
    │
    │ (user chains)
    ▼
Embeddings (existing) ─────────────────────> * Document (with vectors)
    │
    │ (user chains)
    ▼
VectorStore (existing) ────────────────────> Storage
```

## Validation Rules

### DirectoryConfig
1. `Path` must be non-empty and point to accessible directory
2. `MaxDepth` must be ≥ 0 (0 = root only)
3. `Concurrency` must be ≥ 1
4. `MaxFileSize` must be ≥ 1 byte
5. `Extensions` if provided, must contain valid extensions (starting with `.`)

### SplitterConfig
1. `ChunkSize` must be ≥ 1
2. `ChunkOverlap` must be ≥ 0 and < `ChunkSize`
3. `Separators` if empty, defaults to standard hierarchy

### Document (output validation)
1. `PageContent` must be non-empty after loading
2. `Metadata["source"]` must be set
3. After splitting: `chunk_index` < `chunk_total`
