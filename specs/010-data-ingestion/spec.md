# Feature Specification: Data Ingestion and Processing

**Feature Branch**: `010-data-ingestion`  
**Created**: 2026-01-11  
**Status**: Draft  
**Input**: User description: "Data Ingestion and Processing - Add document loaders and text splitters packages for RAG pipelines, enabling loading from various sources and intelligent chunking for embeddings/vectorstores."

## Clarifications

### Session 2026-01-11

- Q: How does the system handle symbolic links in directory traversal? → A: Follow symlinks with cycle detection - follow symbolic links but detect and skip cycles to prevent infinite loops.
- Q: Should files be loaded in parallel within RecursiveDirectoryLoader? → A: Configurable bounded concurrency - default worker pool sized to GOMAXPROCS, configurable via `WithConcurrency(n)` option.
- Q: What is the maximum file size limit for loading? → A: Configurable limit with 100MB default - files exceeding limit are skipped with warning, configurable via `WithMaxFileSize(bytes)`.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Load Documents from Directory (Priority: P1)

As a developer building a RAG application, I want to load documents from a directory structure so that I can process my knowledge base files without writing custom file handling code.

**Why this priority**: Directory loading is the most common ingestion pattern for knowledge bases. This enables developers to quickly onboard existing document collections (company docs, code repositories, documentation sites) into their RAG pipelines.

**Independent Test**: Can be fully tested by pointing the loader at a test directory with mixed file types and verifying all compatible files are loaded with proper metadata (file path, size, modification time).

**Acceptance Scenarios**:

1. **Given** a directory containing text files, **When** I use the RecursiveDirectoryLoader with default settings, **Then** all text files are loaded as Document structs with PageContent and Metadata fields populated.
2. **Given** a nested directory structure with max depth of 3, **When** I configure MaxDepth=2, **Then** only files within 2 levels are loaded, and deeper files are skipped.
3. **Given** a directory with mixed file types (.txt, .md, .pdf, .html), **When** I load with file type filtering enabled, **Then** only files matching the configured extensions are processed.
4. **Given** an empty directory, **When** I attempt to load, **Then** an empty document slice is returned without error.
5. **Given** a directory containing symbolic links, **When** I load with default settings, **Then** symlinks are followed but cycles are detected and skipped with a warning logged.
6. **Given** a file exceeding 100MB (default limit), **When** I load with default settings, **Then** the file is skipped with a warning and other files continue processing.

---

### User Story 2 - Split Documents into Chunks (Priority: P1)

As a developer preparing documents for embedding, I want to split large documents into overlapping chunks so that I can maintain context while respecting embedding model token limits.

**Why this priority**: Text splitting is essential for any RAG pipeline—embeddings have token limits, and context preservation through overlap is critical for retrieval quality. This is a core dependency for vectorstore ingestion.

**Independent Test**: Can be fully tested by passing sample documents to the splitter and verifying chunk sizes, overlap patterns, and boundary handling.

**Acceptance Scenarios**:

1. **Given** a document exceeding chunk size, **When** I split with ChunkSize=1000 and ChunkOverlap=200, **Then** resulting chunks are approximately 1000 characters with 200 characters of overlap between consecutive chunks.
2. **Given** a document smaller than chunk size, **When** I split, **Then** the document is returned as a single chunk without modification.
3. **Given** a markdown document with headers, **When** I use MarkdownTextSplitter, **Then** splits occur preferentially at header boundaries while respecting chunk size limits.
4. **Given** configured separator preferences (e.g., "\n\n", "\n", " "), **When** splitting, **Then** the splitter attempts separators in order, falling back as needed.

---

### User Story 3 - Load Single Text File (Priority: P2)

As a developer with a single source document, I want to load a text file directly so that I can process standalone documents without directory traversal overhead.

**Why this priority**: Simple file loading is a common entry point for testing and small-scale applications. It provides the foundation that directory loaders build upon.

**Independent Test**: Can be fully tested by loading a sample text file and verifying content and metadata extraction.

**Acceptance Scenarios**:

1. **Given** a valid text file path, **When** I use TextLoader, **Then** the file content is returned as a single Document with file path in metadata.
2. **Given** a non-existent file path, **When** I attempt to load, **Then** a descriptive LoaderError is returned with appropriate error code.
3. **Given** a file with non-UTF8 encoding, **When** I load with encoding detection enabled, **Then** content is properly decoded or an encoding error is returned.

---

### User Story 4 - End-to-End RAG Ingestion Pipeline (Priority: P2)

As a developer building a complete RAG system, I want to chain loaders and splitters together so that I can create a seamless pipeline from raw files to vectorstore-ready chunks.

**Why this priority**: The composability of loaders and splitters is what makes this feature valuable. Developers need to see how these components integrate with existing embeddings and vectorstores packages.

**Independent Test**: Can be fully tested by running a complete pipeline: load → split → verify chunks are suitable for embedding.

**Acceptance Scenarios**:

1. **Given** a directory of documents and a configured splitter, **When** I chain loader.Load() → splitter.SplitDocuments(), **Then** I receive chunked documents ready for embedding.
2. **Given** the RAG pipeline with OTEL enabled, **When** processing documents, **Then** traces show load duration, document count, split duration, and chunk count as span attributes.
3. **Given** an error in any pipeline stage, **When** the error occurs, **Then** the error is wrapped with context (stage, file, operation) and propagated appropriately.

---

### User Story 5 - Register Custom Loader (Priority: P3)

As a developer with specialized document sources, I want to register custom loaders via a factory registry so that I can extend the framework without modifying core code.

**Why this priority**: Extensibility is an enterprise requirement. Custom loaders (database exports, API responses, proprietary formats) need first-class support through the registry pattern.

**Independent Test**: Can be fully tested by implementing a mock custom loader, registering it, and retrieving it by name.

**Acceptance Scenarios**:

1. **Given** a custom loader implementing DocumentLoader interface, **When** I register it with a unique name, **Then** the loader is available via factory.GetLoader("custom-name").
2. **Given** an attempt to register a duplicate loader name, **When** registration is attempted, **Then** an appropriate error is returned.
3. **Given** a request for an unregistered loader, **When** I call GetLoader(), **Then** an error with ErrCodeNotFound is returned.

---

### Edge Cases

- **File locked by another process**: Return LoaderError with ErrCodeIOError and continue processing other files.
- **Symbolic links**: Follow symlinks with cycle detection; skip cycles and log warning with ErrCodeCycleDetected.
- **Chunk size smaller than minimum unit**: Validate at construction; reject with ErrCodeInvalidConfig if ChunkSize < 1.
- **Binary files encountered unexpectedly**: Detect via content sniffing (first 512 bytes); skip with warning and ErrCodeBinaryFile.
- **ChunkOverlap exceeds ChunkSize**: Validate at construction; reject with ErrCodeInvalidConfig.
- **Documents with no natural split points**: Fall back to character-based splitting at ChunkSize boundary.
- **File exceeds maximum size limit**: Skip file with warning, log ErrCodeFileTooLarge, continue with other files.

## Requirements *(mandatory)*

### Functional Requirements

#### Document Loaders (pkg/documentloaders)

- **FR-001**: System MUST provide a `DocumentLoader` interface that implements `core.Loader` interface (from `pkg/core/interfaces.go`), including both `Load(ctx context.Context) ([]schema.Document, error)` and `LazyLoad(ctx context.Context) (<-chan any, error)` methods.
- **FR-002**: System MUST provide a `TextLoader` implementation that reads plain text files into a single Document.
- **FR-003**: System MUST provide a `RecursiveDirectoryLoader` implementation that traverses directories using configurable file system abstraction (fs.FS).
- **FR-004**: RecursiveDirectoryLoader MUST support configurable `MaxDepth` to limit recursion depth.
- **FR-005**: RecursiveDirectoryLoader MUST support file extension filtering to select appropriate loaders per file type.
- **FR-006**: System MUST populate Document metadata with source file path, file size, and modification timestamp.
- **FR-007**: System MUST provide a global `ProviderRegistry` for registering and retrieving loader implementations by name.
- **FR-008**: All loaders MUST respect context cancellation and return appropriate errors when cancelled.
- **FR-009**: System MUST provide custom `LoaderError` type with `Op` (operation), `Err` (wrapped error), and `Code` (error code) fields.
- **FR-010**: System MUST emit OTEL traces for Load() and LazyLoad() operations with attributes: loader_type, document_count, duration.
- **FR-011**: System MUST emit OTEL metrics: operations_total (counter), documents_loaded (counter), load_duration (histogram).
- **FR-027**: RecursiveDirectoryLoader MUST follow symbolic links with cycle detection, skipping cycles and logging warnings.
- **FR-028**: RecursiveDirectoryLoader MUST support configurable bounded concurrency via `WithConcurrency(n)` option, defaulting to GOMAXPROCS.
- **FR-029**: RecursiveDirectoryLoader MUST support configurable maximum file size via `WithMaxFileSize(bytes)` option, defaulting to 100MB.
- **FR-030**: Files exceeding MaxFileSize MUST be skipped with a warning logged and ErrCodeFileTooLarge in metadata.
- **FR-031**: System MUST detect binary files via content sniffing (first 512 bytes) and skip with ErrCodeBinaryFile.

#### Text Splitters (pkg/textsplitters)

- **FR-012**: System MUST provide a `TextSplitter` interface that implements `retrievers.iface.Splitter` interface (from `pkg/retrievers/iface/interfaces.go`), including `SplitText(ctx context.Context, text string) ([]string, error)`, `SplitDocuments(ctx context.Context, documents []schema.Document) ([]schema.Document, error)`, and `CreateDocuments(ctx context.Context, texts []string, metadatas []map[string]any) ([]schema.Document, error)` methods.
- **FR-013**: System MUST provide `RecursiveCharacterTextSplitter` with configurable `ChunkSize` and `ChunkOverlap`. *(Note: FR-013 was previously FR-014; renumbered to fill gap after FR-012)*
- **FR-014**: RecursiveCharacterTextSplitter MUST support configurable separator hierarchy (default: "\n\n", "\n", " ", "").
- **FR-015**: System MUST provide `MarkdownTextSplitter` that respects markdown structure (headers, code blocks).
- **FR-016**: System MUST support custom length functions for token-based chunking (e.g., via LLM tokenizer).
- **FR-017**: Splitters MUST provide functional options: `WithChunkSize(int)`, `WithChunkOverlap(int)`, `WithLengthFunction(func(string) int)`.
- **FR-018**: System MUST validate that ChunkOverlap is less than ChunkSize during construction.
- **FR-019**: System MUST provide a global registry for splitter implementations.
- **FR-020**: System MUST emit OTEL traces for split operations with attributes: splitter_type, input_count, output_count, duration.
- **FR-021**: System MUST emit OTEL metrics: operations_total (counter), chunks_created (counter), split_duration (histogram).

#### Configuration & Validation

- **FR-023**: System MUST provide config structs with validation tags (mapstructure, yaml, env, validate).
- **FR-024**: DirectoryConfig MUST include: Path (required), MaxDepth (default 10), Extensions (optional filter list), Concurrency (default GOMAXPROCS), MaxFileSize (default 100MB), FollowSymlinks (default true).
- **FR-025**: SplitterConfig MUST include: ChunkSize (required, min 1), ChunkOverlap (required, min 0).
- **FR-026**: System MUST validate configurations at construction time using validator library.

### Key Entities

- **Document**: Represents a loaded document with `PageContent` (string content) and `Metadata` (map of source info). Already exists in `pkg/schema`.
- **DocumentLoader**: Abstraction for loading documents from any source into standardized Document structs.
- **TextSplitter**: Abstraction for chunking text content while preserving context through configurable overlap.
- **LoaderError**: Custom error type with operation context, wrapped error, and categorized error code.
- **ProviderRegistry**: Factory pattern registry for dynamic loader/splitter instantiation by name.

### Error Codes

- **ErrCodeIOError**: File system read/write failures, locked files.
- **ErrCodeNotFound**: Requested loader/splitter not in registry, file not found.
- **ErrCodeInvalidConfig**: Configuration validation failures (e.g., ChunkOverlap >= ChunkSize).
- **ErrCodeCycleDetected**: Symbolic link cycle detected during traversal.
- **ErrCodeFileTooLarge**: File exceeds configured MaxFileSize limit.
- **ErrCodeBinaryFile**: Binary content detected in expected text file.
- **ErrCodeCancelled**: Context cancelled during operation.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Developers can load a 1000-file directory into Documents in under 5 seconds on standard hardware (8-core CPU, SSD).
- **SC-002**: Developers can split 100 documents averaging 10KB each in under 1 second.
- **SC-003**: New loader types can be added and registered without modifying existing code (open-closed principle).
- **SC-004**: All public operations emit OTEL traces visible in standard observability tools.
- **SC-005**: Error messages include operation context, file paths, and actionable error codes.
- **SC-006**: Integration tests demonstrate complete RAG pipeline: load → split → embed → store → retrieve.
- **SC-007**: 100% test coverage for core interfaces and implementations.
- **SC-008**: Table-driven tests cover all edge cases (empty files, encoding issues, permission errors, symlink cycles, oversized files).
- **SC-009**: Concurrent loading with default settings utilizes available CPU cores efficiently (measured via profiling).

## Assumptions

- The existing `pkg/schema` Document struct is sufficient for representing loaded documents.
- OTEL monitoring infrastructure from `pkg/monitoring` is available for integration.
- Standard file system operations (os, io/fs) are sufficient; no specialized file system drivers needed initially.
- Token-based length functions will be provided via injection from `pkg/llms` or external tokenizers.
- PDF and HTML parsing will use established external libraries (e.g., `github.com/ledongthuc/pdf`, `golang.org/x/net/html`).
- 100MB default file size limit is appropriate for typical RAG document collections.

## Out of Scope

- Web/URL loading (planned as future URLLoader in subsequent iteration)
- Database-specific loaders (SQLLoader, NoSQLLoader)
- Real-time file watching/streaming ingestion
- Automatic encoding detection (initial version assumes UTF-8)
- Image/audio/video content extraction (multimodal handled separately)
- Streaming/chunked reading for files exceeding memory (deferred to future iteration)

## Dependencies

- `pkg/schema` - Document struct definition
- `pkg/core` - Loader interface definition (core.Loader)
- `pkg/retrievers` - Splitter interface definition (retrievers.iface.Splitter)
- `pkg/monitoring` - OTEL tracing and metrics
- `pkg/config` - Configuration validation patterns
- External: `github.com/go-playground/validator/v10` for validation

## Architecture Notes

**Interface Alignment**: The new packages implement existing interfaces:
- `pkg/documentloaders` implementations MUST implement `core.Loader` interface
- `pkg/textsplitters` implementations MUST implement `retrievers.iface.Splitter` interface

**Package Placement Rationale**: 
- These packages are placed at the root level (`pkg/documentloaders`, `pkg/textsplitters`) rather than inside `pkg/schema` or a RAG package because:
  1. They are **processing utilities**, not data contracts (schema is for data structures)
  2. They are **reusable beyond RAG** (document analysis, preprocessing, batch processing)
  3. They follow the **same pattern** as other standalone packages (`embeddings`, `vectorstores`, `retrievers`)
  4. They provide **provider registries** and factory patterns consistent with Beluga architecture
  5. RAG is a **composition pattern** (combines loaders + splitters + embeddings + vectorstores + retrievers), not a single package
