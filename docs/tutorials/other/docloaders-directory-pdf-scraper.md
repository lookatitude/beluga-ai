# Directory & PDF Recursive Scraper

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll use Beluga AI's document loaders to ingest entire directories of files, including nested folders and PDFs. You'll learn how to configure recursive scanning, implement custom loaders for specialized formats, and handle document metadata automatically.

## Learning Objectives
- ✅ Use the `DirectoryLoader` for recursive scanning
- ✅ Configure file extensions and exclusions
- ✅ Implement a PDF extraction wrapper
- ✅ Handle document metadata automatically

## Introduction
Welcome, colleague! Manually uploading files one by one doesn't scale. To build a serious knowledge base, you need to be able to point your agent at a folder and say "learn this." Let's look at how to build a robust directory scraper that can handle everything from Markdown to PDFs.

## Prerequisites

- Go 1.24+
- `pkg/documentloaders` package

## Step 1: Basic Directory Loading

Load all `.txt` and `.md` files in a folder.
```go
package main

import (
    "context"
    "fmt"
    "os"
    "github.com/lookatitude/beluga-ai/pkg/documentloaders"
)

func main() {
    // 1. Create a filesystem
    fsys := os.DirFS("./docs_repo")
    
    // 2. Initialize Directory Loader
    loader, err := documentloaders.NewDirectoryLoader(fsys,
        documentloaders.WithMaxDepth(5),              // Go deep
        documentloaders.WithExtensions(".md", ".txt"), // Filter types
    )
    
    // 3. Load documents
    docs, _ := loader.Load(context.Background())
    
    for _, doc := range docs {
        fmt.Printf("Loaded: %s (Size: %d)\n", doc.Metadata["source"], len(doc.PageContent))
    }
}
```

## Step 2: Handling PDFs

PDF loading usually requires an external library (like `rsc.io/pdf` or a wrapper). Beluga AI allows custom loaders per extension.
// Custom PDF Loader logic
```go
type PDFLoader struct {
    path string
}

func (p *PDFLoader) Load(ctx context.Context) ([]schema.Document, error) {
    // Logic to extract text from PDF...
}

// Register in DirectoryLoader
loader, _ := documentloaders.NewDirectoryLoader(fsys,
    documentloaders.WithCustomLoader(".pdf", func(path string) documentloaders.Loader {
        return &PDFLoader{path: path}
    }),
)
```

## Step 3: Filtering and Exclusions

Avoid loading `node_modules`, `.git`, or temporary files.
loader, _ := documentloaders.NewDirectoryLoader(fsys,
```
    documentloaders.WithExclusions("**/node_modules/**", "**/.DS_Store"),
)

## Step 4: Metadata Enrichment

Automatically add the file creation date or owner to document metadata.
loader, _ := documentloaders.NewDirectoryLoader(fsys,
go
```go
    documentloaders.WithMetadataFunc(func(path string) map[string]any {
        info, _ := os.Stat(path)
        return map[string]any{
            "last_modified": info.ModTime(),
            "file_name":     path,
        }
    }),
)
```

## Verification

1. Create a folder with nested subfolders and sample files.
2. Run the scraper.
3. Verify the number of loaded documents matches your folder structure.

## Next Steps

- **[Lazy-loading Large Data Lakes](./docloaders-lazy-loading.md)** - Optimize for millions of files.
- **[Markdown-aware Chunking](./textsplitters-markdown-chunking.md)** - Split your loaded docs.
