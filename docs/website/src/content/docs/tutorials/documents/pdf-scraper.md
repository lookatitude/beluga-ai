---
title: Directory and PDF Recursive Scraper
description: Ingest entire directories of files including nested folders and PDFs using document loader pipelines with custom format handlers and metadata enrichment.
---

Manually uploading files one by one does not scale. To build a production knowledge base, you need to point your ingestion pipeline at a directory and have it recursively process everything from Markdown to PDFs. The `rag/loader` package provides loaders for common file formats and a pipeline for chaining them with transformers.

## What You Will Build

A directory scraper that recursively loads files of different formats, handles PDF extraction via custom loaders, applies automatic metadata enrichment, and filters out unwanted files. The result is a clean collection of `schema.Document` values ready for splitting and embedding.

## Prerequisites

- Familiarity with `schema.Document` and the `loader` package
- For PDF support: a Go PDF library such as `github.com/ledongthuc/pdf`

## Step 1: Basic Directory Loading

Load all text and Markdown files from a directory tree:

```go
package main

import (
    "context"
    "fmt"
    "io/fs"
    "os"
    "path/filepath"
    "strings"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/loader"
    "github.com/lookatitude/beluga-ai/schema"
)

func loadDirectory(ctx context.Context, rootDir string) ([]schema.Document, error) {
    textLoader, err := loader.New("text", config.ProviderConfig{})
    if err != nil {
        return nil, fmt.Errorf("create loader: %w", err)
    }

    markdownLoader, err := loader.New("markdown", config.ProviderConfig{})
    if err != nil {
        return nil, fmt.Errorf("create loader: %w", err)
    }

    loaders := map[string]loader.DocumentLoader{
        ".txt": textLoader,
        ".md":  markdownLoader,
    }

    var docs []schema.Document
    err = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if d.IsDir() {
            return nil
        }

        ext := strings.ToLower(filepath.Ext(path))
        l, ok := loaders[ext]
        if !ok {
            return nil // Skip unsupported formats.
        }

        loaded, err := l.Load(ctx, path)
        if err != nil {
            return fmt.Errorf("loading %s: %w", path, err)
        }
        docs = append(docs, loaded...)
        return nil
    })

    return docs, err
}
```

## Step 2: Implement a Custom PDF Loader

Create a custom loader for PDF files. This example shows the pattern -- use your preferred PDF library for the actual text extraction:

```go
// PDFLoader extracts text from PDF files.
type PDFLoader struct{}

func (l *PDFLoader) Load(ctx context.Context, source string) ([]schema.Document, error) {
    // In production, use a PDF parsing library.
    // Example with github.com/ledongthuc/pdf:
    //
    //   f, r, err := pdf.Open(source)
    //   if err != nil { return nil, err }
    //   defer f.Close()
    //   var buf bytes.Buffer
    //   for i := 1; i <= r.NumPage(); i++ {
    //       p := r.Page(i)
    //       text, _ := p.GetPlainText(nil)
    //       buf.WriteString(text)
    //   }

    // Placeholder for demonstration.
    content := fmt.Sprintf("PDF content from: %s", source)

    return []schema.Document{
        {
            ID:      source,
            Content: content,
            Metadata: map[string]any{
                "source": source,
                "format": "pdf",
            },
        },
    }, nil
}
```

## Step 3: Filter Unwanted Files

Exclude directories and files that should not be indexed:

```go
// shouldSkip returns true for files and directories that should not be loaded.
func shouldSkip(path string, d fs.DirEntry) bool {
    name := d.Name()

    // Skip hidden files and directories.
    if strings.HasPrefix(name, ".") {
        if d.IsDir() {
            return true
        }
        return true
    }

    // Skip common non-content directories.
    skipDirs := map[string]bool{
        "node_modules": true,
        "vendor":       true,
        "__pycache__":  true,
        ".git":         true,
    }
    if d.IsDir() && skipDirs[name] {
        return true
    }

    return false
}
```

Use the filter in the directory walk:

```go
err = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
    if err != nil {
        return err
    }
    if shouldSkip(path, d) {
        if d.IsDir() {
            return filepath.SkipDir
        }
        return nil
    }
    // ... load files ...
    return nil
})
```

## Step 4: Automatic Metadata Enrichment

Add file system metadata to every loaded document:

```go
func enrichMetadata(docs []schema.Document) []schema.Document {
    for i := range docs {
        source, ok := docs[i].Metadata["source"].(string)
        if !ok {
            continue
        }

        info, err := os.Stat(source)
        if err != nil {
            continue
        }

        docs[i].Metadata["file_size"] = info.Size()
        docs[i].Metadata["last_modified"] = info.ModTime().UTC().Format("2006-01-02T15:04:05Z")
        docs[i].Metadata["file_name"] = filepath.Base(source)
        docs[i].Metadata["directory"] = filepath.Dir(source)
    }
    return docs
}
```

## Step 5: Use the Pipeline

Combine loaders and transformers using `LoaderPipeline`:

```go
func buildIngestionPipeline() *loader.LoaderPipeline {
    textLoader, _ := loader.New("text", config.ProviderConfig{})

    enricher := loader.TransformerFunc(
        func(ctx context.Context, doc schema.Document) (schema.Document, error) {
            if doc.Metadata == nil {
                doc.Metadata = make(map[string]any)
            }
            doc.Metadata["ingested_at"] = time.Now().UTC().Format(time.RFC3339)

            // Add word count.
            doc.Metadata["word_count"] = len(strings.Fields(doc.Content))
            return doc, nil
        },
    )

    return loader.NewPipeline(
        loader.WithLoader(textLoader),
        loader.WithTransformer(enricher),
    )
}
```

## Step 6: Full Directory Scraper

Combine all components into a complete scraper:

```go
func scrapeDirectory(ctx context.Context, rootDir string) ([]schema.Document, error) {
    textLoader, _ := loader.New("text", config.ProviderConfig{})
    markdownLoader, _ := loader.New("markdown", config.ProviderConfig{})
    pdfLoader := &PDFLoader{}

    loaders := map[string]loader.DocumentLoader{
        ".txt": textLoader,
        ".md":  markdownLoader,
        ".pdf": pdfLoader,
    }

    var allDocs []schema.Document

    err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if shouldSkip(path, d) {
            if d.IsDir() {
                return filepath.SkipDir
            }
            return nil
        }
        if d.IsDir() {
            return nil
        }

        ext := strings.ToLower(filepath.Ext(path))
        l, ok := loaders[ext]
        if !ok {
            return nil
        }

        docs, err := l.Load(ctx, path)
        if err != nil {
            fmt.Printf("Warning: failed to load %s: %v\n", path, err)
            return nil // Continue on error.
        }
        allDocs = append(allDocs, docs...)
        return nil
    })
    if err != nil {
        return nil, err
    }

    allDocs = enrichMetadata(allDocs)

    fmt.Printf("Loaded %d documents from %s\n", len(allDocs), rootDir)
    return allDocs, nil
}
```

## Verification

1. Create a test directory with nested subdirectories and mixed file types (.txt, .md, .pdf).
2. Run the scraper. Verify the number of loaded documents matches the file count.
3. Check that metadata includes file size, modification date, and word count.
4. Verify that `.git`, `node_modules`, and hidden files are excluded.

## Next Steps

- [Lazy-Loading](/tutorials/documents/lazy-loading) -- Process millions of files with constant memory usage
- [Markdown Chunking](/tutorials/documents/markdown-chunking) -- Split loaded Markdown documents by heading structure
