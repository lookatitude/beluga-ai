---
title: "GitHub Document Loader"
description: "Load files from GitHub repositories in Beluga AI. Contents API integration with branch selection and file filtering for code RAG in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "GitHub, document loader, repository loader, Contents API, code RAG, Go, Beluga AI"
---

The GitHub loader implements the `loader.DocumentLoader` interface for loading files from GitHub repositories. It uses the GitHub Contents API to fetch file content, with support for branch/tag/SHA references.

Choose GitHub when you need to load source code, documentation, or configuration files from GitHub repositories into a RAG pipeline. The loader supports branch/tag/SHA references and works with both GitHub.com and GitHub Enterprise. For loading from other platforms, consider [Confluence](/providers/loader/confluence) or [Notion](/providers/loader/notion).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/loader/providers/github
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
    _ "github.com/lookatitude/beluga-ai/rag/loader/providers/github"
)

func main() {
    l, err := loader.New("github", config.ProviderConfig{
        APIKey: os.Getenv("GITHUB_TOKEN"),
    })
    if err != nil {
        log.Fatal(err)
    }

    docs, err := l.Load(context.Background(), "lookatitude/beluga-ai/README.md")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("SHA: %s\n", docs[0].Metadata["sha"])
    fmt.Printf("Content: %s\n", docs[0].Content)
}
```

## Configuration

| Parameter | Type | Default | Description |
|---|---|---|---|
| `APIKey` | `string` | `""` | GitHub personal access token (optional for public repos) |
| `BaseURL` | `string` | `https://api.github.com` | GitHub API base URL (use for GitHub Enterprise) |
| `Timeout` | `time.Duration` | `30s` | HTTP request timeout |
| `Options["ref"]` | `string` | `""` | Git ref (branch, tag, or commit SHA) to load from |

## Source Format

The `source` parameter uses the format `owner/repo/path/to/file`:

```go
docs, err := l.Load(ctx, "lookatitude/beluga-ai/docs/concepts.md")
```

The source must contain at least three path segments: owner, repository, and file path.

## Loading from a Specific Branch

Use `Options["ref"]` to load from a specific branch, tag, or commit:

```go
l, err := loader.New("github", config.ProviderConfig{
    APIKey: os.Getenv("GITHUB_TOKEN"),
    Options: map[string]any{
        "ref": "v2.0.0",
    },
})
docs, err := l.Load(ctx, "lookatitude/beluga-ai/README.md")
```

## GitHub Enterprise

Point to your GitHub Enterprise instance using `BaseURL`:

```go
l, err := loader.New("github", config.ProviderConfig{
    APIKey:  os.Getenv("GHE_TOKEN"),
    BaseURL: "https://github.corp.example.com/api/v3",
})
```

## Document Metadata

| Field | Type | Description |
|---|---|---|
| `source` | `string` | Original `owner/repo/path` string |
| `loader` | `string` | Always `"github"` |
| `path` | `string` | File path within the repository |
| `sha` | `string` | Git blob SHA |
| `size` | `int` | File size in bytes |
| `html_url` | `string` | GitHub web URL for the file |
| `download_url` | `string` | Direct download URL |

## Public Repositories

For public repositories, the `APIKey` is optional. However, unauthenticated requests are subject to lower rate limits (60 requests per hour vs. 5,000 authenticated):

```go
l, err := loader.New("github", config.ProviderConfig{})
docs, err := l.Load(ctx, "golang/go/README.md")
```

## Error Handling

```go
docs, err := l.Load(ctx, "owner/repo/file.go")
if err != nil {
    // Possible errors:
    // - "github: source is required" (empty source)
    // - "github: source must be in format owner/repo/path" (invalid format)
    // - "github: ... is a dir, not a file" (source points to directory)
    // - "github: fetch ...: ..." (API error, 404 or 403)
    log.Fatal(err)
}
```
