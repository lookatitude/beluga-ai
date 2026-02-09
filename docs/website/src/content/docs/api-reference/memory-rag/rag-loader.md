---
title: "RAG Document Loaders"
description: "Document loaders for files, cloud storage, APIs, and web content"
---

## loader

```go
import "github.com/lookatitude/beluga-ai/rag/loader"
```

Package loader provides document loading capabilities for the RAG pipeline.
It defines the `DocumentLoader` interface for reading content from various
sources (files, URLs, APIs) and converting them into [schema.Document] slices.

## Interfaces

The core interface is `DocumentLoader`:

```go
type DocumentLoader interface {
    Load(ctx context.Context, source string) ([]schema.Document, error)
}
```

The `Transformer` interface allows post-load enrichment:

```go
type Transformer interface {
    Transform(ctx context.Context, doc schema.Document) (schema.Document, error)
}
```

## Registry

The package follows Beluga's registry pattern. Providers register via
init() and are instantiated with `New`:

```go
l, err := loader.New("text", config.ProviderConfig{})
if err != nil {
    log.Fatal(err)
}
docs, err := l.Load(ctx, "/path/to/file.txt")
```

Use `List` to discover all registered loader names.

## Built-in Loaders

- "text" — plain text files
- "json" — JSON files with configurable path extraction
- "csv" — CSV files (one document per row)
- "markdown" — Markdown files

## External Loaders

Available as provider imports:
- "cloudstorage" — S3, GCS, Azure Blob Storage
- "confluence" — Atlassian Confluence pages
- "docling" — IBM Docling document understanding (PDFs, DOCX, images)
- "firecrawl" — Firecrawl web scraping and crawling
- "gdrive" — Google Drive files
- "github" — GitHub repository files
- "notion" — Notion pages
- "unstructured" — Unstructured.io document extraction

## Pipeline

`LoaderPipeline` chains multiple loaders and transformers. Loaders are invoked
in order and their results concatenated, then transformers are applied to each
document:

```go
p := loader.NewPipeline(
    loader.WithLoader(textLoader),
    loader.WithTransformer(loader.TransformerFunc(func(ctx context.Context, doc schema.Document) (schema.Document, error) {
        doc.Metadata["processed"] = true
        return doc, nil
    })),
)
docs, err := p.Load(ctx, "/path/to/files")
```

## Custom Provider

To add a custom document loader:

```go
func init() {
    loader.Register("custom", func(cfg config.ProviderConfig) (loader.DocumentLoader, error) {
        return &myLoader{apiKey: cfg.APIKey}, nil
    })
}
```

---

## cloudstorage

```go
import "github.com/lookatitude/beluga-ai/rag/loader/providers/cloudstorage"
```

Package cloudstorage provides a DocumentLoader that loads files from cloud
storage services (S3, GCS, Azure Blob). It detects the provider by URL prefix
(s3://, gs://, az://) and uses direct HTTP calls with pre-signed URLs or
service-specific APIs.

## Registration

The provider registers as "cloudstorage" in the loader registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/loader/providers/cloudstorage"

l, err := loader.New("cloudstorage", config.ProviderConfig{
    APIKey: "your-access-key",
    Options: map[string]any{
        "secret_key": "your-secret-key",
        "region":     "us-east-1",
    },
})
docs, err := l.Load(ctx, "s3://bucket/path/to/file.txt")
```

## Supported Providers

- S3 — URLs starting with "s3://"
- GCS — URLs starting with "gs://"
- Azure Blob — URLs starting with "az://"

## Configuration

ProviderConfig fields:
- APIKey — access key (required)
- Options["secret_key"] — secret key
- Options["region"] — cloud region

---

## confluence

```go
import "github.com/lookatitude/beluga-ai/rag/loader/providers/confluence"
```

Package confluence provides a DocumentLoader that loads pages from Atlassian
Confluence via its REST API. It implements the [loader.DocumentLoader] interface.

## Registration

The provider registers as "confluence" in the loader registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/loader/providers/confluence"

l, err := loader.New("confluence", config.ProviderConfig{
    APIKey:  "your-api-token",
    BaseURL: "https://your-domain.atlassian.net/wiki",
    Options: map[string]any{"user": "user@example.com"},
})
docs, err := l.Load(ctx, "12345") // page ID
```

## Configuration

ProviderConfig fields:
- APIKey — Confluence API token (required)
- BaseURL — Confluence wiki base URL (required)
- Options["user"] — username for basic auth

---

## docling

```go
import "github.com/lookatitude/beluga-ai/rag/loader/providers/docling"
```

Package docling provides a DocumentLoader that uses the IBM Docling API
to convert documents (PDFs, DOCX, images, etc.) into structured content.

Docling (https://github.com/DS4SD/docling) is IBM's document understanding
service that extracts text, tables, and layout from documents.

## Registration

The provider registers as "docling" in the loader registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/loader/providers/docling"

l, err := loader.New("docling", config.ProviderConfig{
    BaseURL: "http://localhost:5001",
})
docs, err := l.Load(ctx, "/path/to/document.pdf")
```

## Configuration

ProviderConfig fields:
- BaseURL — Docling API server URL (required)

---

## firecrawl

```go
import "github.com/lookatitude/beluga-ai/rag/loader/providers/firecrawl"
```

Package firecrawl provides a DocumentLoader that uses Firecrawl to crawl
websites and extract their content as markdown.

Firecrawl (https://firecrawl.dev) is a web scraping service that handles
JavaScript rendering, anti-bot detection, and returns clean markdown.

## Registration

The provider registers as "firecrawl" in the loader registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/loader/providers/firecrawl"

l, err := loader.New("firecrawl", config.ProviderConfig{
    APIKey: "fc-...",
})
docs, err := l.Load(ctx, "https://example.com")
```

## Configuration

ProviderConfig fields:
- APIKey — Firecrawl API key (required)

---

## gdrive

```go
import "github.com/lookatitude/beluga-ai/rag/loader/providers/gdrive"
```

Package gdrive provides a DocumentLoader that loads files from Google Drive
via the Google Drive REST API. It implements the [loader.DocumentLoader]
interface using direct HTTP calls.

## Registration

The provider registers as "gdrive" in the loader registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/loader/providers/gdrive"

l, err := loader.New("gdrive", config.ProviderConfig{
    APIKey: "your-api-key-or-oauth-token",
})
docs, err := l.Load(ctx, "file-id-here")
```

## Configuration

ProviderConfig fields:
- APIKey — Google API key or OAuth token (required)

---

## github

```go
import "github.com/lookatitude/beluga-ai/rag/loader/providers/github"
```

Package github provides a DocumentLoader that loads files from GitHub
repositories via the GitHub API. It implements the [loader.DocumentLoader]
interface using direct HTTP calls.

## Registration

The provider registers as "github" in the loader registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/loader/providers/github"

l, err := loader.New("github", config.ProviderConfig{
    APIKey: "ghp_...",
})
docs, err := l.Load(ctx, "owner/repo/path/to/file.go")
```

## Configuration

ProviderConfig fields:
- APIKey — GitHub personal access token (required)

---

## notion

```go
import "github.com/lookatitude/beluga-ai/rag/loader/providers/notion"
```

Package notion provides a DocumentLoader that loads pages from Notion via
its API. It implements the [loader.DocumentLoader] interface using direct HTTP
calls to the Notion API.

## Registration

The provider registers as "notion" in the loader registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/loader/providers/notion"

l, err := loader.New("notion", config.ProviderConfig{
    APIKey: "ntn_...",
})
docs, err := l.Load(ctx, "page-id-here")
```

## Configuration

ProviderConfig fields:
- APIKey — Notion integration token (required)

---

## unstructured

```go
import "github.com/lookatitude/beluga-ai/rag/loader/providers/unstructured"
```

Package unstructured provides a DocumentLoader that uses the Unstructured.io
API to extract structured content from files (PDFs, DOCX, images, etc.).

The loader uploads files to the Unstructured.io partition API and returns
the extracted elements as documents.

## Registration

The provider registers as "unstructured" in the loader registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/loader/providers/unstructured"

l, err := loader.New("unstructured", config.ProviderConfig{
    APIKey:  "key-...",
    BaseURL: "https://api.unstructured.io",
})
docs, err := l.Load(ctx, "/path/to/document.pdf")
```

## Configuration

ProviderConfig fields:
- APIKey — Unstructured.io API key (required)
- BaseURL — API base URL (default: "https://api.unstructured.io")
