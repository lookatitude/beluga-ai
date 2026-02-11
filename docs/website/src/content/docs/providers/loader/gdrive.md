---
title: Google Drive
description: Load files from Google Drive via the REST API.
---

The Google Drive loader implements the `loader.DocumentLoader` interface for loading files from Google Drive. It uses the Google Drive REST API v3 to fetch file content, with automatic export support for Google Workspace documents (Docs, Sheets, Slides).

Choose Google Drive when you need to load documents from Google Drive, especially Google Workspace files (Docs, Sheets, Slides). The loader automatically exports Workspace formats to text, avoiding manual conversion. For other cloud storage (S3, GCS, Azure), use [Cloud Storage](/providers/loader/cloudstorage). For Atlassian wikis, consider [Confluence](/providers/loader/confluence).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/loader/providers/gdrive
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
    _ "github.com/lookatitude/beluga-ai/rag/loader/providers/gdrive"
)

func main() {
    l, err := loader.New("gdrive", config.ProviderConfig{
        APIKey: os.Getenv("GOOGLE_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }

    docs, err := l.Load(context.Background(), "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("File: %s\n", docs[0].Metadata["file_name"])
    fmt.Printf("Content: %s\n", docs[0].Content)
}
```

## Configuration

| Parameter | Type | Default | Description |
|---|---|---|---|
| `APIKey` | `string` | (required) | Google API key or OAuth access token |
| `BaseURL` | `string` | `https://www.googleapis.com` | Google API base URL |
| `Timeout` | `time.Duration` | `60s` | HTTP request timeout |

## Source Format

The `source` parameter is a Google Drive file ID:

```go
docs, err := l.Load(ctx, "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms")
```

You can find the file ID in the Google Drive URL:
`https://docs.google.com/document/d/<FILE_ID>/edit`

## Google Workspace Export

The loader automatically detects Google Workspace formats and exports them as text:

| MIME Type | Export Format |
|---|---|
| `application/vnd.google-apps.document` | Plain text |
| `application/vnd.google-apps.spreadsheet` | CSV |
| `application/vnd.google-apps.presentation` | Plain text |
| Other `vnd.google-apps.*` types | Plain text |

Non-Google files (PDFs, text files, etc.) are downloaded directly.

## Document Metadata

| Field | Type | Description |
|---|---|---|
| `source` | `string` | File ID passed to `Load` |
| `loader` | `string` | Always `"gdrive"` |
| `file_id` | `string` | Google Drive file ID |
| `file_name` | `string` | File name from Drive metadata |
| `mime_type` | `string` | File MIME type |

## Authentication

### API Key

For public files or files accessible via API key:

```go
l, err := loader.New("gdrive", config.ProviderConfig{
    APIKey: os.Getenv("GOOGLE_API_KEY"),
})
```

### OAuth Token

For private files, pass an OAuth 2.0 access token as the `APIKey`:

```go
l, err := loader.New("gdrive", config.ProviderConfig{
    APIKey: oauthToken.AccessToken,
})
```

## Error Handling

```go
docs, err := l.Load(ctx, "file-id")
if err != nil {
    // Possible errors:
    // - "gdrive: API key or OAuth token is required" (missing APIKey)
    // - "gdrive: file ID is required" (empty source)
    // - "gdrive: get metadata ...: ..." (file not found or access denied)
    // - "gdrive: get content ...: export failed (status 403): ..." (auth error)
    log.Fatal(err)
}
```
