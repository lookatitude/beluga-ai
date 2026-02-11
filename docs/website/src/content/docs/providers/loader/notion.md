---
title: Notion
description: Load pages from Notion workspaces via the Notion API.
---

The Notion loader implements the `loader.DocumentLoader` interface for loading pages from Notion. It fetches page metadata and block children via the Notion API and converts block content into plain text.

Choose Notion when your team's knowledge base or documentation lives in Notion and you want to feed page content into a RAG pipeline. The loader extracts plain text from paragraphs, headings, lists, quotes, callouts, and code blocks. For Atlassian wikis, consider [Confluence](/providers/loader/confluence). For Google-based content, consider [Google Drive](/providers/loader/gdrive).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/loader/providers/notion
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
    _ "github.com/lookatitude/beluga-ai/rag/loader/providers/notion"
)

func main() {
    l, err := loader.New("notion", config.ProviderConfig{
        APIKey: os.Getenv("NOTION_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }

    docs, err := l.Load(context.Background(), "page-id-here")
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
| `APIKey` | `string` | (required) | Notion integration token (starts with `ntn_`) |
| `BaseURL` | `string` | `https://api.notion.com` | Notion API base URL |
| `Timeout` | `time.Duration` | `30s` | HTTP request timeout |

## Source Format

The `source` parameter is a Notion page ID. Both formatted and unformatted IDs are accepted:

```go
// With hyphens
docs, err := l.Load(ctx, "a1b2c3d4-e5f6-7890-abcd-ef1234567890")

// Without hyphens
docs, err := l.Load(ctx, "a1b2c3d4e5f67890abcdef1234567890")
```

You can find the page ID in the Notion URL:
`https://www.notion.so/Page-Title-<PAGE_ID>`

## Supported Block Types

The loader extracts plain text from the following Notion block types:

| Block Type | Extraction |
|---|---|
| `paragraph` | Rich text content |
| `heading_1`, `heading_2`, `heading_3` | Heading text |
| `bulleted_list_item` | List item text |
| `numbered_list_item` | List item text |
| `toggle` | Toggle header text |
| `quote` | Quote content |
| `callout` | Callout content |
| `code` | Code content (without language annotation) |

Blocks are joined with double newlines to create readable plain text output.

## Document Metadata

| Field | Type | Description |
|---|---|---|
| `source` | `string` | Original page ID string |
| `loader` | `string` | Always `"notion"` |
| `page_id` | `string` | Resolved Notion page ID |
| `title` | `string` | Page title extracted from properties |

## Setting Up a Notion Integration

1. Go to [My Integrations](https://www.notion.so/my-integrations)
2. Create a new internal integration
3. Copy the integration token
4. Share the target pages with the integration (click "Share" on the page and add the integration)

```go
l, err := loader.New("notion", config.ProviderConfig{
    APIKey: "ntn_your_integration_token",
})
```

## Error Handling

```go
docs, err := l.Load(ctx, "page-id")
if err != nil {
    // Possible errors:
    // - "notion: API key (integration token) is required" (missing APIKey)
    // - "notion: page ID is required" (empty source)
    // - "notion: fetch page ...: ..." (page not found or not shared)
    // - "notion: fetch blocks ...: ..." (block retrieval failure)
    log.Fatal(err)
}
```
