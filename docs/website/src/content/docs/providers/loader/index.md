---
title: Document Loader Providers
description: Overview of all document loader providers available in Beluga AI v2.
---

Beluga AI v2 provides a unified `loader.DocumentLoader` interface for loading content from diverse sources — files, APIs, cloud storage, and web pages — and converting them into `schema.Document` slices for use in RAG pipelines. All providers register via `init()` and are instantiated through the global registry.

## Interface

```go
type DocumentLoader interface {
    Load(ctx context.Context, source string) ([]schema.Document, error)
}
```

The `source` parameter is provider-specific: it may be a file path, URL, page ID, or cloud storage URI depending on the loader.

## Registry Usage

```go
import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/loader"

    // Register the provider you need via blank import
    _ "github.com/lookatitude/beluga-ai/rag/loader/providers/github"
)

func main() {
    l, err := loader.New("github", config.ProviderConfig{
        APIKey: os.Getenv("GITHUB_TOKEN"),
    })
    if err != nil {
        log.Fatal(err)
    }

    docs, err := l.Load(context.Background(), "owner/repo/README.md")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Loaded %d documents\n", len(docs))
}
```

## Available Providers

| Provider | Registry Name | Source Format | Description |
|---|---|---|---|
| [Cloud Storage](/providers/loader/cloudstorage) | `cloudstorage` | `s3://`, `gs://`, `az://` URIs | AWS S3, Google Cloud Storage, Azure Blob |
| [Confluence](/providers/loader/confluence) | `confluence` | Page ID or `SPACE/page-id` | Atlassian Confluence wiki pages |
| [Docling](/providers/loader/docling) | `docling` | File path or URL | IBM Docling document conversion (PDF, DOCX, images) |
| [Firecrawl](/providers/loader/firecrawl) | `firecrawl` | URL | Web scraping with JavaScript rendering |
| [Google Drive](/providers/loader/gdrive) | `gdrive` | File ID | Google Drive files and Google Workspace exports |
| [GitHub](/providers/loader/github) | `github` | `owner/repo/path` | GitHub repository files via the Contents API |
| [Notion](/providers/loader/notion) | `notion` | Page ID | Notion pages via the Notion API |
| [Unstructured](/providers/loader/unstructured) | `unstructured` | File path | Unstructured.io document extraction |

### Built-in Loaders

The `loader` package also includes four built-in loaders that require no external dependencies:

| Loader | Registry Name | Description |
|---|---|---|
| Text | `text` | Plain text files |
| JSON | `json` | JSON files with configurable path extraction |
| CSV | `csv` | CSV files (one document per row) |
| Markdown | `markdown` | Markdown files |

## Provider Discovery

List all registered providers at runtime:

```go
names := loader.List()
// Returns sorted list: ["cloudstorage", "confluence", "csv", "docling", ...]
```

## Transformer Pipeline

Loaders can be combined with transformers in a pipeline. The pipeline runs all loaders, concatenates their results, and applies transformers to each document:

```go
import "github.com/lookatitude/beluga-ai/rag/loader"

pipeline := loader.NewPipeline(
    loader.WithLoader(githubLoader),
    loader.WithLoader(notionLoader),
    loader.WithTransformer(loader.TransformerFunc(func(ctx context.Context, doc schema.Document) (schema.Document, error) {
        doc.Metadata["processed"] = true
        return doc, nil
    })),
)

docs, err := pipeline.Load(ctx, "owner/repo/docs/")
if err != nil {
    log.Fatal(err)
}
```

## Document Structure

All loaders return `schema.Document` values with consistent metadata:

```go
type Document struct {
    ID       string
    Content  string
    Metadata map[string]any
}
```

Every loader sets at minimum:
- `source` — the original source string passed to `Load`
- `loader` — the loader name (e.g., `"github"`, `"confluence"`)

Additional metadata fields are provider-specific and documented on each provider's page.
