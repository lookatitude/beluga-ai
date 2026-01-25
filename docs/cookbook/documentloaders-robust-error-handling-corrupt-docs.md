---
title: "Robust Error Handling for Corrupt Docs"
package: "documentloaders"
category: "resilience"
complexity: "intermediate"
---

# Robust Error Handling for Corrupt Docs

## Problem

You need to handle corrupt, malformed, or unreadable documents gracefully without failing the entire document loading process, logging errors while continuing to process other documents.

## Solution

Implement error isolation that catches document-specific errors, logs them with context, skips corrupt documents, and continues processing remaining documents. This works because you can wrap document loading in error handlers and collect errors separately from successful documents.

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
    
    "github.com/lookatitude/beluga-ai/pkg/documentloaders"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

var tracer = otel.Tracer("beluga.documentloaders.error_handling")

// DocumentLoadResult represents result of loading a document
type DocumentLoadResult struct {
    Document schema.Document
    Error    error
    FilePath string
}

// RobustDocumentLoader loads documents with error handling
type RobustDocumentLoader struct {
    loader      documentloaders.DocumentLoader
    skipErrors  bool
    errorLogger func(string, error)
}

// NewRobustDocumentLoader creates a new robust loader
func NewRobustDocumentLoader(loader documentloaders.DocumentLoader, skipErrors bool) *RobustDocumentLoader {
    return &RobustDocumentLoader{
        loader:     loader,
        skipErrors: skipErrors,
        errorLogger: func(filePath string, err error) {
            log.Printf("Error loading %s: %v", filePath, err)
        },
    }
}

// LoadDocuments loads documents with error handling
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
    
    if errorCount > 0 {
        span.SetAttributes(attribute.Bool("errors_skipped", rdl.skipErrors))
        span.SetStatus(trace.StatusOK, fmt.Sprintf("loaded %d, skipped %d errors", len(documents), errorCount))
    } else {
        span.SetStatus(trace.StatusOK, "all documents loaded")
    }
    
    return documents, results, nil
}

// loadDocument loads a single document with error handling
func (rdl *RobustDocumentLoader) loadDocument(ctx context.Context, filePath string) DocumentLoadResult {
    ctx, span := tracer.Start(ctx, "robust_loader.load_document")
    defer span.End()
    
    span.SetAttributes(attribute.String("file_path", filePath))
    
    // Wrap in panic recovery
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
        return DocumentLoadResult{
            FilePath: filePath,
            Error:    err,
        }
    }
    
    // Try to load document
    doc, err := rdl.loader.Load(ctx)
    if err != nil {
        // Check error type
        if rdl.isRecoverableError(err) {
            // Try recovery
            doc, err = rdl.attemptRecovery(ctx, filePath, err)
        }
        
        if err != nil {
            span.RecordError(err)
            span.SetStatus(trace.StatusError, err.Error())
            return DocumentLoadResult{
                FilePath: filePath,
                Error:    err,
            }
        }
    }
    
    span.SetStatus(trace.StatusOK, "document loaded")
    return DocumentLoadResult{
        FilePath: filePath,
        Document: doc,
    }
}

// isRecoverableError checks if error is recoverable
func (rdl *RobustDocumentLoader) isRecoverableError(err error) bool {
    errStr := err.Error()
    recoverablePatterns := []string{
        "encoding",
        "corrupt",
        "truncated",
        "partial",
    }
    
    for _, pattern := range recoverablePatterns {
        if contains(errStr, pattern) {
            return true
        }
    }
    return false
}

// attemptRecovery attempts to recover from error
func (rdl *RobustDocumentLoader) attemptRecovery(ctx context.Context, filePath string, err error) (schema.Document, error) {
    // Try to load with error-tolerant parser
    // In practice, would try different parsing strategies
    return nil, err
}

func contains(s, substr string) bool {
    return len(s) >= len(substr)
}

// ErrorSummary provides summary of loading errors
type ErrorSummary struct {
    TotalFiles    int
    Successful    int
    Failed        int
    ErrorTypes    map[string]int
    FailedFiles   []string
}

// GetErrorSummary creates error summary
func (rdl *RobustDocumentLoader) GetErrorSummary(results []DocumentLoadResult) *ErrorSummary {
    summary := &ErrorSummary{
        TotalFiles: len(results),
        ErrorTypes: make(map[string]int),
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

    // Create robust loader
    // loader := yourDocumentLoader
    robustLoader := NewRobustDocumentLoader(loader, true)
    
    // Load documents
    filePaths := []string{"doc1.txt", "corrupt.doc", "doc3.txt"}
    docs, results, err := robustLoader.LoadDocuments(ctx, filePaths)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    
    // Get summary
    summary := robustLoader.GetErrorSummary(results)
    fmt.Printf("Loaded: %d, Failed: %d\n", summary.Successful, summary.Failed)
}
```

## Explanation

Let's break down what's happening:

1. **Error isolation** - Notice how we handle errors per document, not failing the entire batch. Each document load is wrapped in error handling that allows continuing with other documents.

2. **Panic recovery** - We wrap document loading in panic recovery to handle unexpected panics gracefully. This prevents one corrupt document from crashing the entire loader.

3. **Error classification** - We classify errors to understand failure patterns. This helps identify systemic issues vs. individual corrupt files.

```go
**Key insight:** Isolate errors per document. One corrupt document shouldn't prevent loading other valid documents. Collect errors separately for analysis.

## Testing

```
Here's how to test this solution:
```go
func TestRobustDocumentLoader_HandlesErrors(t *testing.T) {
    mockLoader := &MockDocumentLoader{shouldError: true}
    robustLoader := NewRobustDocumentLoader(mockLoader, true)
    
    docs, results, err := robustLoader.LoadDocuments(context.Background(), []string{"test.txt"})
    require.NoError(t, err) // Should not fail even with errors
    require.Len(t, docs, 0) // No documents loaded
    require.Len(t, results, 1)
    require.NotNil(t, results[0].Error)
}

## Variations

### Retry on Transient Errors

Retry loading on transient errors:
func (rdl *RobustDocumentLoader) LoadWithRetry(ctx context.Context, filePath string, maxRetries int) DocumentLoadResult {
    // Retry logic
}
```

### Partial Document Recovery

Recover partial content from corrupt documents:
```go
func (rdl *RobustDocumentLoader) RecoverPartial(ctx context.Context, filePath string) (schema.Document, error) {
    // Extract readable portions
}
```

## Related Recipes

- **[Documentloaders Parallel File Walkers](./documentloaders-parallel-file-walkers.md)** - Parallel loading
- **[Document Ingestion Recipes](./document-ingestion-recipes.md)** - Additional patterns
- **[Documentloaders Package Guide](../package_design_patterns.md)** - For a deeper understanding of document loaders
