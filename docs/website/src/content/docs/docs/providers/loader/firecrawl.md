---
title: "Firecrawl Document Loader"
description: "Scrape websites and extract clean Markdown with Firecrawl in Beluga AI. Web crawling with JavaScript rendering and clean content for RAG in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Firecrawl, web scraper, document loader, web crawling, Markdown extraction, RAG pipeline, Go, Beluga AI"
---

The Firecrawl loader implements the `loader.DocumentLoader` interface using the [Firecrawl](https://firecrawl.dev) web scraping service. Firecrawl handles JavaScript rendering, anti-bot detection, and page extraction, returning clean Markdown content suitable for RAG pipelines.

Choose Firecrawl when you need to scrape web pages — including JavaScript-rendered content — and extract clean Markdown for RAG pipelines. Firecrawl handles anti-bot detection and page extraction, producing content ready for splitting and embedding. For loading local documents, consider [Docling](/docs/providers/loader/docling) or [Unstructured](/docs/providers/loader/unstructured).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/loader/providers/firecrawl
```

This provider depends on the official Firecrawl Go SDK (`github.com/mendableai/firecrawl-go`).

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
    _ "github.com/lookatitude/beluga-ai/rag/loader/providers/firecrawl"
)

func main() {
    l, err := loader.New("firecrawl", config.ProviderConfig{
        APIKey: os.Getenv("FIRECRAWL_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }

    docs, err := l.Load(context.Background(), "https://example.com")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Content: %s\n", docs[0].Content)
}
```

## Configuration

| Parameter | Type | Default | Description |
|---|---|---|---|
| `APIKey` | `string` | (required) | Firecrawl API key (starts with `fc-`) |
| `BaseURL` | `string` | `https://api.firecrawl.dev` | Firecrawl API endpoint |

## Source Format

The `source` parameter must be a valid URL to scrape:

```go
docs, err := l.Load(ctx, "https://docs.example.com/guide")
```

## Document Metadata

| Field | Type | Description |
|---|---|---|
| `source` | `string` | Original URL |
| `format` | `string` | Always `"markdown"` |
| `loader` | `string` | Always `"firecrawl"` |
| `title` | `string` | Page title (if available) |
| `description` | `string` | Page meta description (if available) |
| `source_url` | `string` | Resolved source URL (if available) |

## Self-Hosted Firecrawl

For self-hosted deployments, point to your own Firecrawl instance:

```go
l, err := loader.New("firecrawl", config.ProviderConfig{
    APIKey:  os.Getenv("FIRECRAWL_API_KEY"),
    BaseURL: "http://localhost:3002",
})
```

## Error Handling

```go
docs, err := l.Load(ctx, "https://example.com")
if err != nil {
    // Possible errors:
    // - "firecrawl: source URL is required" (empty source)
    // - "firecrawl: init client: ..." (invalid API key or URL)
    // - "firecrawl: scrape ...: ..." (scraping failure)
    log.Fatal(err)
}
```
