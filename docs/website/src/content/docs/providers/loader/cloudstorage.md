---
title: Cloud Storage
description: Load documents from AWS S3, Google Cloud Storage, and Azure Blob Storage.
---

The Cloud Storage loader implements the `loader.DocumentLoader` interface for loading files from cloud object storage services. It detects the cloud provider automatically by URL prefix (`s3://`, `gs://`, `az://`) and fetches content via direct HTTP calls.

Choose Cloud Storage when your documents are stored in AWS S3, Google Cloud Storage, or Azure Blob Storage. The loader auto-detects the cloud provider from the URL prefix, so a single loader instance handles all three. For loading from SaaS knowledge bases, consider [Confluence](/providers/loader/confluence), [Notion](/providers/loader/notion), or [Google Drive](/providers/loader/gdrive).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/loader/providers/cloudstorage
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
    _ "github.com/lookatitude/beluga-ai/rag/loader/providers/cloudstorage"
)

func main() {
    l, err := loader.New("cloudstorage", config.ProviderConfig{
        APIKey: os.Getenv("AWS_ACCESS_KEY_ID"),
        Options: map[string]any{
            "secret_key": os.Getenv("AWS_SECRET_ACCESS_KEY"),
            "region":     "us-east-1",
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    docs, err := l.Load(context.Background(), "s3://my-bucket/documents/report.txt")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Content: %s\n", docs[0].Content)
}
```

## Configuration

| Parameter | Type | Default | Description |
|---|---|---|---|
| `APIKey` | `string` | `""` | Access key or bearer token for authentication |
| `Timeout` | `time.Duration` | `60s` | HTTP request timeout |
| `Options["secret_key"]` | `string` | `""` | Secret key (used with S3) |
| `Options["region"]` | `string` | `us-east-1` | AWS region for S3 bucket URLs |

## Supported Providers

The loader determines the cloud provider from the URL prefix:

| Prefix | Provider | URL Format |
|---|---|---|
| `s3://` | AWS S3 | `s3://bucket/key` |
| `gs://` | Google Cloud Storage | `gs://bucket/object` |
| `az://` | Azure Blob Storage | `az://container/blob` |

## Provider-Specific Examples

### AWS S3

```go
docs, err := l.Load(ctx, "s3://my-bucket/data/document.pdf")
```

### Google Cloud Storage

```go
l, err := loader.New("cloudstorage", config.ProviderConfig{
    APIKey: os.Getenv("GCS_ACCESS_TOKEN"),
})
docs, err := l.Load(ctx, "gs://my-bucket/reports/summary.txt")
```

### Azure Blob Storage

```go
l, err := loader.New("cloudstorage", config.ProviderConfig{
    APIKey: os.Getenv("AZURE_STORAGE_TOKEN"),
})
docs, err := l.Load(ctx, "az://my-container/files/data.csv")
```

## Document Metadata

Each loaded document includes the following metadata fields:

| Field | Type | Description |
|---|---|---|
| `source` | `string` | Original cloud storage URI |
| `loader` | `string` | Always `"cloudstorage"` |
| `provider` | `string` | Cloud provider: `"s3"`, `"gcs"`, or `"azure"` |
| `bucket` | `string` | Bucket or container name |
| `key` | `string` | Object key or blob path |
| `filename` | `string` | Extracted filename from the key |

## Error Handling

The loader returns descriptive errors for invalid URLs and failed requests:

```go
docs, err := l.Load(ctx, "s3://my-bucket/path/to/file.txt")
if err != nil {
    // Possible errors:
    // - "cloudstorage: invalid S3 URL ..." (malformed URL)
    // - "cloudstorage: fetch ... failed (status 403): ..." (auth failure)
    // - "cloudstorage: fetch ...: context deadline exceeded" (timeout)
    log.Fatal(err)
}
```
