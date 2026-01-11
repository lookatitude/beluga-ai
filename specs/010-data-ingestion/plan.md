# Implementation Plan: Data Ingestion and Processing

**Branch**: `010-data-ingestion` | **Date**: 2026-01-11 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/010-data-ingestion/spec.md`

## Summary

Add two new packages (`pkg/documentloaders` and `pkg/textsplitters`) to provide document ingestion and text chunking capabilities for RAG pipelines. These packages follow the established Beluga v2 architecture with interface-driven design, global registries for extensibility, comprehensive OTEL observability, and enterprise-grade error handling. The implementation bridges a key gap compared to Langchaingo, enabling developers to load documents from various sources and split them into embedding-ready chunks without custom code.

## Technical Context

**Language/Version**: Go 1.22+  
**Primary Dependencies**: 
- `github.com/go-playground/validator/v10` (config validation)
- `go.opentelemetry.io/otel` (tracing/metrics)
- `github.com/lookatitude/beluga-ai/pkg/schema` (Document struct)
- `github.com/lookatitude/beluga-ai/pkg/core` (Loader interface - `core.Loader`)
- `github.com/lookatitude/beluga-ai/pkg/retrievers` (Splitter interface - `retrievers.iface.Splitter`)
- `github.com/lookatitude/beluga-ai/pkg/monitoring` (OTEL helpers)

**Storage**: N/A (file system operations via `io/fs` abstraction)  
**Testing**: Go standard `testing` package, table-driven tests, benchmarks  
**Target Platform**: Linux/macOS/Windows (anywhere Go runs)  
**Project Type**: Go library package (single project structure)  
**Performance Goals**: 
- Load 1000 files in <5 seconds (with default concurrency)
- Split 100x10KB documents in <1 second
- Memory: O(file_size) per concurrent worker

**Constraints**: 
- Max file size: 100MB default (configurable)
- UTF-8 encoding assumed (no auto-detection in v1)
- No external C dependencies for core loaders

**Scale/Scope**: 
- Initial providers: TextLoader, RecursiveDirectoryLoader, RecursiveCharacterTextSplitter, MarkdownTextSplitter
- Future providers: PDFLoader, HTMLLoader, URLLoader (out of scope for v1)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| **I. Package Structure** | ✅ PASS | Both packages follow v2 structure with iface/, internal/, providers/, config.go, metrics.go, errors.go |
| **II. Interface Design (ISP)** | ✅ PASS | Small, focused interfaces: `DocumentLoader` implements `core.Loader`, `TextSplitter` implements `retrievers.iface.Splitter` |
| **III. Provider Registry Pattern** | ✅ PASS | Global registry with GetRegistry(), Register(), Create() matching pkg/llms pattern |
| **IV. OTEL Observability** | ✅ PASS | metrics.go with counters/histograms, tracing for Load/Split operations |
| **V. Error Handling** | ✅ PASS | Custom LoaderError/SplitterError with Op/Err/Code pattern, 7 error codes defined |
| **VI. Configuration** | ✅ PASS | Config structs with mapstructure/yaml/env/validate tags, functional options |
| **VII. Testing** | ✅ PASS | Table-driven tests, mocks in internal/mock/, benchmarks planned |
| **VIII. Backward Compatibility** | ✅ PASS | New packages, no breaking changes to existing APIs |

**Gate Result**: ✅ PASSED - All constitution principles satisfied

## Project Structure

### Documentation (this feature)

```text
specs/010-data-ingestion/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (Go interface definitions)
│   ├── documentloaders.md
│   └── textsplitters.md
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
pkg/
├── documentloaders/
│   ├── iface/
│   │   └── loader.go           # DocumentLoader interface
│   ├── internal/
│   │   └── mock/
│   │       └── loader.go       # Mock implementations for testing
│   ├── providers/
│   │   ├── text/
│   │   │   ├── init.go         # Auto-registration
│   │   │   └── loader.go       # TextLoader implementation
│   │   └── directory/
│   │       ├── init.go         # Auto-registration
│   │       └── loader.go       # RecursiveDirectoryLoader implementation
│   ├── config.go               # DirectoryConfig, LoaderConfig structs
│   ├── documentloaders.go      # Package entry point, factory functions
│   ├── errors.go               # LoaderError, error codes
│   ├── metrics.go              # OTEL metrics (operations_total, documents_loaded, etc.)
│   ├── registry.go             # Global loader registry
│   ├── test_utils.go           # AdvancedMockLoader, test helpers
│   ├── advanced_test.go        # Table-driven tests, concurrency tests, benchmarks
│   └── README.md               # Package documentation
│
├── textsplitters/
│   ├── iface/
│   │   └── splitter.go         # TextSplitter interface
│   ├── internal/
│   │   └── mock/
│   │       └── splitter.go     # Mock implementations for testing
│   ├── providers/
│   │   ├── recursive/
│   │   │   ├── init.go         # Auto-registration
│   │   │   └── splitter.go     # RecursiveCharacterTextSplitter
│   │   └── markdown/
│   │       ├── init.go         # Auto-registration
│   │       └── splitter.go     # MarkdownTextSplitter
│   ├── config.go               # SplitterConfig struct
│   ├── textsplitters.go        # Package entry point, factory functions
│   ├── errors.go               # SplitterError, error codes
│   ├── metrics.go              # OTEL metrics (chunks_created, split_duration, etc.)
│   ├── registry.go             # Global splitter registry
│   ├── test_utils.go           # AdvancedMockSplitter, test helpers
│   ├── advanced_test.go        # Table-driven tests, edge cases, benchmarks
│   └── README.md               # Package documentation
│
└── schema/
    └── internal/
        └── document.go         # Existing Document struct (no changes needed)

tests/
└── integration/
    └── package_pairs/
        └── documentloaders_textsplitters_test.go  # RAG pipeline integration tests
```

**Structure Decision**: Single project structure (Go library). Both new packages follow the established pattern from `pkg/llms/` with iface/, internal/, providers/ subdirectories and global registries.

**Interface Alignment**: 
- `pkg/documentloaders` implementations MUST implement `core.Loader` interface (from `pkg/core/interfaces.go`)
- `pkg/textsplitters` implementations MUST implement `retrievers.iface.Splitter` interface (from `pkg/retrievers/iface/interfaces.go`)
- This ensures compatibility with existing RAG pipeline components that expect these interfaces

## Complexity Tracking

> No constitution violations requiring justification.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |

## Phase Summary

| Phase | Artifact | Status |
|-------|----------|--------|
| 0 | research.md | ✅ Created |
| 1 | data-model.md | ✅ Created |
| 1 | contracts/documentloaders.md | ✅ Created |
| 1 | contracts/textsplitters.md | ✅ Created |
| 1 | quickstart.md | ✅ Created |
| 2 | tasks.md | Pending `/speckit.tasks` |
