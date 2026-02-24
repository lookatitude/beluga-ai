---
title: "Corrupt Document Handling"
description: "Recipe for handling corrupt, malformed, or unreadable documents in Go RAG pipelines — skip failures, log errors, and continue processing with Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, corrupt document handling, Go error recovery, RAG resilience, document validation, ingestion pipeline, fault tolerance"
---

## Problem

You need to handle corrupt, malformed, or unreadable documents gracefully without failing the entire document loading process, logging errors while continuing to process other documents.

In production document pipelines, data quality is never perfect. Files may be truncated from incomplete uploads, encoded with unexpected character sets, contain binary data disguised by a text extension, or simply be empty. When processing hundreds or thousands of files, encountering a few corrupt documents is the norm rather than the exception. A naive pipeline that returns the first error and stops leaves all subsequent valid documents unprocessed.

## Solution

Implement error isolation that catches document-specific errors, logs them with context for debugging, skips corrupt documents, and continues processing remaining documents. The wrapper pattern lets you add resilience to any existing `DocumentLoader` without modifying it. This follows the middleware pattern (`func(T) T`) common in Beluga AI: wrap the inner loader with additional behavior while preserving the same interface.

## Code Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/rag/loader"
    "github.com/lookatitude/beluga-ai/schema"
)

var tracer = otel.Tracer("beluga.documentloaders.error_handling")

// DocumentLoadResult represents the result of loading a single document.
type DocumentLoadResult struct {
    Document schema.Document
    Error    error
    FilePath string
}

// RobustDocumentLoader loads documents with error isolation.
type RobustDocumentLoader struct {
    loader      loader.DocumentLoader
    skipErrors  bool
    errorLogger func(string, error)
}

// NewRobustDocumentLoader creates a new robust loader.
func NewRobustDocumentLoader(l loader.DocumentLoader, skipErrors bool) *RobustDocumentLoader {
    return &RobustDocumentLoader{
        loader:     l,
        skipErrors: skipErrors,
        errorLogger: func(filePath string, err error) {
            log.Printf("Error loading %s: %v", filePath, err)
        },
    }
}

// LoadDocuments loads documents with per-file error handling.
func (rdl *RobustDocumentLoader) LoadDocuments(ctx context.Context, filePaths []string) ([]schema.Document, []DocumentLoadResult, error) {
    ctx, span := tracer.Start(ctx, "robust_loader.load_documents")
    defer span.End()

    span.SetAttributes(
        attribute.Int("file_count", len(filePaths)),
        attribute.Bool("skip_errors", rdl.skipErrors),
    )

    documents := []schema.Document{}
    results := []DocumentLoadResult{}
    errorCount := 0

    for _, filePath := range filePaths {
        result := rdl.loadDocument(ctx, filePath)
        results = append(results, result)

        if result.Error != nil {
            errorCount++
            rdl.errorLogger(filePath, result.Error)

            if !rdl.skipErrors {
                span.RecordError(result.Error)
                span.SetStatus(trace.StatusError, "error encountered, not skipping")
                return nil, results, fmt.Errorf("failed to load %s: %w", filePath, result.Error)
            }
        } else {
            documents = append(documents, result.Document)
        }
    }

    span.SetAttributes(
        attribute.Int("loaded_count", len(documents)),
        attribute.Int("error_count", errorCount),
    )

    return documents, results, nil
}

// loadDocument loads a single document with error handling and panic recovery.
func (rdl *RobustDocumentLoader) loadDocument(ctx context.Context, filePath string) DocumentLoadResult {
    ctx, span := tracer.Start(ctx, "robust_loader.load_document")
    defer span.End()

    span.SetAttributes(attribute.String("file_path", filePath))

    defer func() {
        if r := recover(); r != nil {
            err := fmt.Errorf("panic loading %s: %v", filePath, r)
            span.RecordError(err)
            span.SetStatus(trace.StatusError, "panic recovered")
        }
    }()

    // Validate file exists
    if _, err := os.Stat(filePath); err != nil {
        err := fmt.Errorf("file not found: %w", err)
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return DocumentLoadResult{FilePath: filePath, Error: err}
    }

    // Try to load document
    doc, err := rdl.loader.Load(ctx)
    if err != nil {
        if rdl.isRecoverableError(err) {
            doc, err = rdl.attemptRecovery(ctx, filePath, err)
        }

        if err != nil {
            span.RecordError(err)
            span.SetStatus(trace.StatusError, err.Error())
            return DocumentLoadResult{FilePath: filePath, Error: err}
        }
    }

    span.SetStatus(trace.StatusOK, "document loaded")
    return DocumentLoadResult{FilePath: filePath, Document: doc}
}

// isRecoverableError checks if an error is recoverable.
func (rdl *RobustDocumentLoader) isRecoverableError(err error) bool {
    errStr := err.Error()
    recoverablePatterns := []string{"encoding", "corrupt", "truncated", "partial"}

    for _, pattern := range recoverablePatterns {
        if contains(errStr, pattern) {
            return true
        }
    }
    return false
}

// attemptRecovery attempts to recover from a loading error.
func (rdl *RobustDocumentLoader) attemptRecovery(ctx context.Context, filePath string, err error) (schema.Document, error) {
    // Try to load with error-tolerant parser
    // In practice, would try different parsing strategies
    return nil, err
}

func contains(s, substr string) bool {
    return len(s) >= len(substr)
}

// ErrorSummary provides a summary of loading errors.
type ErrorSummary struct {
    TotalFiles  int
    Successful  int
    Failed      int
    ErrorTypes  map[string]int
    FailedFiles []string
}

// GetErrorSummary creates an error summary from results.
func GetErrorSummary(results []DocumentLoadResult) *ErrorSummary {
    summary := &ErrorSummary{
        TotalFiles:  len(results),
        ErrorTypes:  make(map[string]int),
        FailedFiles: []string{},
    }

    for _, result := range results {
        if result.Error != nil {
            summary.Failed++
            summary.FailedFiles = append(summary.FailedFiles, result.FilePath)
            errorType := getErrorType(result.Error)
            summary.ErrorTypes[errorType]++
        } else {
            summary.Successful++
        }
    }

    return summary
}

func getErrorType(err error) string {
    errStr := err.Error()
    if contains(errStr, "not found") {
        return "file_not_found"
    }
    if contains(errStr, "permission") {
        return "permission_denied"
    }
    if contains(errStr, "corrupt") {
        return "corrupt_file"
    }
    return "unknown"
}

func main() {
    ctx := context.Background()

    // Create robust loader (loader is your DocumentLoader instance)
    // robustLoader := NewRobustDocumentLoader(loader, true)

    filePaths := []string{"doc1.txt", "corrupt.doc", "doc3.txt"}
    docs, results, err := robustLoader.LoadDocuments(ctx, filePaths)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    summary := GetErrorSummary(results)
    fmt.Printf("Loaded: %d, Failed: %d\n", summary.Successful, summary.Failed)
}
```

## Explanation

1. **Error isolation** -- Each document load is wrapped in its own error handler with independent success/failure tracking. One corrupt document does not prevent loading other valid documents in the batch. The `DocumentLoadResult` struct captures per-file outcomes, giving callers full visibility into what succeeded and what failed.

2. **Panic recovery** -- Document loading is wrapped in `defer/recover` to handle unexpected panics from third-party parsers or malformed input that triggers nil pointer dereferences. Without this, one pathological file can crash the entire process. This is defense-in-depth: the error handling catches expected failures, while panic recovery catches unexpected ones.

3. **Error classification** -- Errors are classified by type (file not found, permission denied, corrupt) using the `ErrorSummary` struct. This helps identify systemic issues (e.g., "90% of failures are permission denied" suggests a misconfigured directory) versus random corruption. Error classification informs whether to retry, skip, or alert.

4. **Configurable behavior** -- The `skipErrors` flag controls whether errors are collected and skipped or propagated immediately. Strict mode (`skipErrors=false`) is useful during development to catch issues early; lenient mode (`skipErrors=true`) is appropriate for production batch ingestion where partial success is preferable to total failure.

## Variations

### Retry on Transient Errors

Retry loading on transient errors like temporary permission issues or filesystem glitches:

```go
func (rdl *RobustDocumentLoader) LoadWithRetry(ctx context.Context, filePath string, maxRetries int) DocumentLoadResult {
    // Retry logic with exponential backoff
}
```

### Partial Document Recovery

Recover partial content from corrupt documents where possible:

```go
func (rdl *RobustDocumentLoader) RecoverPartial(ctx context.Context, filePath string) (schema.Document, error) {
    // Extract readable portions from corrupt files
}
```

## Related Recipes

- [Parallel File Loading](/cookbook/parallel-file-loading) -- Parallel loading with worker pools
- [Document Ingestion Recipes](/cookbook/document-ingestion) -- Additional document loading patterns
