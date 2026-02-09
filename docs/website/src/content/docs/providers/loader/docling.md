---
title: Docling
description: Convert documents to structured content using IBM Docling.
---

The Docling loader implements the `loader.DocumentLoader` interface using the IBM Docling API to convert documents (PDFs, DOCX, images, and more) into structured text. Docling extracts text, tables, and layout information and returns the content as Markdown or plain text.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/loader/providers/docling
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/loader"
    _ "github.com/lookatitude/beluga-ai/rag/loader/providers/docling"
)

func main() {
    l, err := loader.New("docling", config.ProviderConfig{
        BaseURL: "http://localhost:5001",
    })
    if err != nil {
        log.Fatal(err)
    }

    docs, err := l.Load(context.Background(), "/path/to/document.pdf")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Content: %s\n", docs[0].Content)
}
```

## Configuration

| Parameter | Type | Default | Description |
|---|---|---|---|
| `BaseURL` | `string` | `http://localhost:5001` | Docling API endpoint |
| `APIKey` | `string` | `""` | Optional Bearer token for authentication |
| `Timeout` | `time.Duration` | 0 (no timeout) | HTTP request timeout |

## Source Types

The loader accepts two types of sources:

### Local Files

File paths are uploaded to the Docling API as multipart form data:

```go
docs, err := l.Load(ctx, "/path/to/document.pdf")
```

### URLs

HTTP/HTTPS URLs are passed to the Docling API as a JSON body for server-side download:

```go
docs, err := l.Load(ctx, "https://example.com/report.pdf")
```

## Content Output

The Docling API returns both Markdown and plain text representations. The loader prefers Markdown content when available, falling back to plain text:

1. Markdown content (`md_content`) is used if present
2. Plain text content (`text_content`) is used as fallback
3. If both are empty, `nil` is returned (no documents)

## Document Metadata

| Field | Type | Description |
|---|---|---|
| `source` | `string` | Original file path or URL |
| `format` | `string` | Always `"docling"` |
| `loader` | `string` | Always `"docling"` |

## Supported File Types

Docling supports a wide range of document formats including:

- PDF documents
- Microsoft Word (DOCX)
- Microsoft PowerPoint (PPTX)
- Images (PNG, JPG, TIFF)
- HTML pages

Refer to the [Docling documentation](https://github.com/DS4SD/docling) for the complete list of supported formats.

## Self-Hosted Deployment

Docling can be run as a local service using Docker:

```bash
docker run -p 5001:5001 ds4sd/docling-serve
```

Once running, configure the loader to point to your local instance:

```go
l, err := loader.New("docling", config.ProviderConfig{
    BaseURL: "http://localhost:5001",
})
```

## Error Handling

```go
docs, err := l.Load(ctx, "/path/to/document.pdf")
if err != nil {
    // Possible errors:
    // - "docling: source is required" (empty source)
    // - "docling: open file: ..." (local file not found)
    // - "docling: API error (status 422): ..." (unsupported format)
    // - "docling: request: ..." (connection failure)
    log.Fatal(err)
}
```
