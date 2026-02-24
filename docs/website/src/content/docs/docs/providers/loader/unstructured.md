---
title: "Unstructured Document Loader"
description: "Extract structured content from documents with Unstructured.io in Beluga AI. PDF, DOCX, and image parsing with OCR support for RAG in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Unstructured, document loader, PDF parser, OCR, document extraction, RAG pipeline, Go, Beluga AI"
---

The Unstructured loader implements the `loader.DocumentLoader` interface using the [Unstructured.io](https://unstructured.io) API to extract structured content from a wide range of file types (PDFs, DOCX, images, HTML, and more). It uploads files to the Unstructured partition API and returns the extracted elements as a single consolidated document.

Choose Unstructured when you need to process a wide variety of document formats (PDFs, DOCX, images with OCR, emails, and more) through a single loader. Unstructured provides element-level extraction with metadata about document structure. It can be self-hosted via Docker for data privacy. For Markdown-focused document conversion, consider [Docling](/docs/providers/loader/docling). For web scraping, consider [Firecrawl](/docs/providers/loader/firecrawl).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/loader/providers/unstructured
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/loader"
    _ "github.com/lookatitude/beluga-ai/rag/loader/providers/unstructured"
)

func main() {
    l, err := loader.New("unstructured", config.ProviderConfig{
        APIKey:  os.Getenv("UNSTRUCTURED_API_KEY"),
        BaseURL: "https://api.unstructured.io",
    })
    if err != nil {
        log.Fatal(err)
    }

    docs, err := l.Load(context.Background(), "/path/to/document.pdf")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Elements extracted: %v\n", docs[0].Metadata["elements"])
    fmt.Printf("Content: %s\n", docs[0].Content)
}
```

## Configuration

| Parameter | Type | Default | Description |
|---|---|---|---|
| `APIKey` | `string` | `""` | Unstructured API key (set via `unstructured-api-key` header) |
| `BaseURL` | `string` | `https://api.unstructured.io` | API endpoint |
| `Timeout` | `time.Duration` | 0 (no timeout) | HTTP request timeout |

## Source Format

The `source` parameter is a local file path. The file is uploaded to the Unstructured API as multipart form data:

```go
docs, err := l.Load(ctx, "/path/to/document.pdf")
```

## Content Extraction

The Unstructured API returns an array of structured elements (titles, narrative text, tables, etc.). The loader combines all text elements into a single document, separated by double newlines. Empty elements are skipped.

## Document Metadata

| Field | Type | Description |
|---|---|---|
| `source` | `string` | Original file path |
| `format` | `string` | Always `"unstructured"` |
| `loader` | `string` | Always `"unstructured"` |
| `filename` | `string` | Base filename extracted from the path |
| `elements` | `int` | Number of elements returned by the API |

## Supported File Types

Unstructured supports a comprehensive set of document formats:

- PDF, DOCX, DOC, PPTX, PPT, XLSX
- HTML, XML, Markdown, RST
- Plain text, CSV, TSV
- Images (PNG, JPG, TIFF) with OCR
- Email formats (EML, MSG)
- EPUB

## Self-Hosted Deployment

You can run Unstructured locally using Docker:

```bash
docker run -p 8000:8000 quay.io/unstructured-io/unstructured-api:latest
```

Then point the loader to your local instance:

```go
l, err := loader.New("unstructured", config.ProviderConfig{
    BaseURL: "http://localhost:8000",
})
```

When self-hosting, the `APIKey` is optional.

## Error Handling

```go
docs, err := l.Load(ctx, "/path/to/document.pdf")
if err != nil {
    // Possible errors:
    // - "unstructured: source file path is required" (empty source)
    // - "unstructured: open file: ..." (file not found)
    // - "unstructured: API error (status 422): ..." (unsupported format)
    // - "unstructured: request: ..." (connection failure)
    log.Fatal(err)
}
```
