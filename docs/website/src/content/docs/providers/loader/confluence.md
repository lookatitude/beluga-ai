---
title: Confluence
description: Load wiki pages from Atlassian Confluence via the REST API.
---

The Confluence loader implements the `loader.DocumentLoader` interface for loading pages from Atlassian Confluence. It fetches page content via the Confluence REST API and extracts plain text from the HTML storage representation.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/loader/providers/confluence
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
    _ "github.com/lookatitude/beluga-ai/rag/loader/providers/confluence"
)

func main() {
    l, err := loader.New("confluence", config.ProviderConfig{
        APIKey:  os.Getenv("CONFLUENCE_API_TOKEN"),
        BaseURL: "https://your-domain.atlassian.net/wiki",
        Options: map[string]any{
            "user": "user@example.com",
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    docs, err := l.Load(context.Background(), "12345")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Title: %s\n", docs[0].Metadata["title"])
    fmt.Printf("Content: %s\n", docs[0].Content)
}
```

## Configuration

| Parameter | Type | Default | Description |
|---|---|---|---|
| `APIKey` | `string` | (required) | Confluence API token |
| `BaseURL` | `string` | (required) | Confluence instance URL (e.g., `https://your-domain.atlassian.net/wiki`) |
| `Timeout` | `time.Duration` | `30s` | HTTP request timeout |
| `Options["user"]` | `string` | `""` | Confluence username for Basic auth; omit for Bearer token auth |

## Authentication

The loader supports two authentication modes:

### Basic Auth (Atlassian Cloud)

When `Options["user"]` is set, the loader uses HTTP Basic authentication with the user email and API token:

```go
l, err := loader.New("confluence", config.ProviderConfig{
    APIKey:  os.Getenv("CONFLUENCE_API_TOKEN"),
    BaseURL: "https://your-domain.atlassian.net/wiki",
    Options: map[string]any{
        "user": "user@example.com",
    },
})
```

### Bearer Token (Data Center / Server)

When `Options["user"]` is omitted, the loader uses Bearer token authentication:

```go
l, err := loader.New("confluence", config.ProviderConfig{
    APIKey:  os.Getenv("CONFLUENCE_PAT"),
    BaseURL: "https://confluence.internal.corp/wiki",
})
```

## Source Format

The `source` parameter accepts:

- **Page ID**: A numeric string like `"12345"`
- **Space/Page ID**: A path like `"ENGINEERING/12345"` (the space key prefix is ignored; only the page ID is used for the API call)

## Document Metadata

| Field | Type | Description |
|---|---|---|
| `source` | `string` | Original source string |
| `loader` | `string` | Always `"confluence"` |
| `page_id` | `string` | Confluence page ID |
| `title` | `string` | Page title |
| `space` | `string` | Space key the page belongs to |

## Content Extraction

The loader fetches pages using the Confluence REST API with `body.storage` expansion and strips HTML tags to produce plain text content. This means formatting information (bold, italic, links) is removed, but all text content is preserved.

## Error Handling

```go
docs, err := l.Load(ctx, "12345")
if err != nil {
    // Possible errors:
    // - "confluence: base URL is required" (missing BaseURL)
    // - "confluence: API key/token is required" (missing APIKey)
    // - "confluence: fetch page ...: ..." (API request failure)
    log.Fatal(err)
}
```
