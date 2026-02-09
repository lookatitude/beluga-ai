---
title: Google Drive API Scraper
description: Build a custom document loader that fetches documents from Google Drive using the Drive API for Beluga AI RAG pipelines.
---

Beluga AI's document loader interface enables loading content from any source. This guide demonstrates how to build a custom Google Drive loader that connects to the Drive API, lists documents in a folder, and converts them into `schema.Document` values for processing in RAG pipelines.

## Overview

The Google Drive loader uses the Drive API to list and download files from a specified folder. It handles Google Workspace formats (Docs, Sheets, Slides) by exporting them to text, and downloads regular files directly. The loader implements patterns compatible with Beluga AI's `DocumentLoader` interface from the `rag/loader` package.

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed
- Google Cloud project with the Drive API enabled
- OAuth 2.0 credentials or a service account key

## Installation

Install the Google API client library:

```bash
go get google.golang.org/api/drive/v3
go get google.golang.org/api/option
```

### Setting Up Credentials

1. Open the [Google Cloud Console](https://console.cloud.google.com)
2. Create a project or select an existing one
3. Enable the Google Drive API under **APIs & Services**
4. Create credentials:
   - For server-side access: create a **Service Account** and download the JSON key
   - For user-delegated access: create **OAuth 2.0 Client ID** credentials
5. Set the credentials path:

```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"
```

## The DocumentLoader Interface

The loader satisfies the `DocumentLoader` interface:

```go
// From github.com/lookatitude/beluga-ai/rag/loader
type DocumentLoader interface {
    Load(ctx context.Context, source string) ([]schema.Document, error)
}
```

For Google Drive, the `source` parameter is interpreted as a folder ID.

## Usage

### Basic Drive Loader

Build a loader that fetches all files from a Google Drive folder:

```go
package main

import (
    "context"
    "fmt"
    "io"
    "log"
    "os"
    "strings"

    "google.golang.org/api/drive/v3"
    "google.golang.org/api/option"
    "github.com/lookatitude/beluga-ai/schema"
)

// DriveLoader loads documents from a Google Drive folder.
type DriveLoader struct {
    service *drive.Service
}

// NewDriveLoader creates a new Google Drive loader using the credentials
// file specified by the GOOGLE_APPLICATION_CREDENTIALS environment variable.
func NewDriveLoader(ctx context.Context) (*DriveLoader, error) {
    credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
    if credsFile == "" {
        return nil, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS not set")
    }

    svc, err := drive.NewService(ctx, option.WithCredentialsFile(credsFile))
    if err != nil {
        return nil, fmt.Errorf("create drive service: %w", err)
    }

    return &DriveLoader{service: svc}, nil
}

// Load fetches all non-trashed files from the given folder ID and returns
// them as documents.
func (l *DriveLoader) Load(ctx context.Context, folderID string) ([]schema.Document, error) {
    query := fmt.Sprintf("'%s' in parents and trashed=false", folderID)
    files, err := l.service.Files.List().
        Q(query).
        Fields("files(id, name, mimeType)").
        Context(ctx).
        Do()
    if err != nil {
        return nil, fmt.Errorf("list files: %w", err)
    }

    var docs []schema.Document
    for _, file := range files.Files {
        content, err := l.downloadFile(ctx, file.Id, file.MimeType)
        if err != nil {
            log.Printf("skipping %s: %v", file.Name, err)
            continue
        }

        docs = append(docs, schema.Document{
            Content: content,
            Metadata: map[string]any{
                "source":    fmt.Sprintf("drive://%s", file.Id),
                "name":      file.Name,
                "mime_type": file.MimeType,
            },
        })
    }

    return docs, nil
}
```

### Handling Google Workspace File Types

Google Workspace files (Docs, Sheets, Slides) cannot be downloaded directly. They must be exported to a supported format:

```go
// downloadFile retrieves the content of a Drive file. Google Workspace
// files are exported to text; regular files are downloaded directly.
func (l *DriveLoader) downloadFile(ctx context.Context, fileID, mimeType string) (string, error) {
    switch mimeType {
    case "application/vnd.google-apps.document":
        return l.export(ctx, fileID, "text/plain")
    case "application/vnd.google-apps.spreadsheet":
        return l.export(ctx, fileID, "text/csv")
    case "application/vnd.google-apps.presentation":
        return l.export(ctx, fileID, "text/plain")
    default:
        return l.download(ctx, fileID)
    }
}

func (l *DriveLoader) export(ctx context.Context, fileID, exportMIME string) (string, error) {
    resp, err := l.service.Files.Export(fileID, exportMIME).Context(ctx).Download()
    if err != nil {
        return "", fmt.Errorf("export %s: %w", fileID, err)
    }
    defer resp.Body.Close()
    return readAll(resp.Body)
}

func (l *DriveLoader) download(ctx context.Context, fileID string) (string, error) {
    resp, err := l.service.Files.Get(fileID).Context(ctx).Download()
    if err != nil {
        return "", fmt.Errorf("download %s: %w", fileID, err)
    }
    defer resp.Body.Close()
    return readAll(resp.Body)
}

func readAll(r io.Reader) (string, error) {
    var b strings.Builder
    if _, err := io.Copy(&b, r); err != nil {
        return "", err
    }
    return b.String(), nil
}
```

| Workspace Type | MIME Type | Export Format |
|---------------|-----------|--------------|
| Google Docs | `application/vnd.google-apps.document` | `text/plain` |
| Google Sheets | `application/vnd.google-apps.spreadsheet` | `text/csv` |
| Google Slides | `application/vnd.google-apps.presentation` | `text/plain` |

### Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
)

func main() {
    ctx := context.Background()

    loader, err := NewDriveLoader(ctx)
    if err != nil {
        log.Fatal(err)
    }

    folderID := os.Getenv("GOOGLE_DRIVE_FOLDER_ID")
    if folderID == "" {
        log.Fatal("GOOGLE_DRIVE_FOLDER_ID not set")
    }

    docs, err := loader.Load(ctx, folderID)
    if err != nil {
        log.Fatal(err)
    }

    for _, doc := range docs {
        fmt.Printf("Loaded: %s (%d bytes)\n", doc.Metadata["name"], len(doc.Content))
    }
}
```

## Advanced Topics

### Observability with OpenTelemetry

Add tracing to the loader for production monitoring:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type TracedDriveLoader struct {
    *DriveLoader
    tracer trace.Tracer
}

func NewTracedDriveLoader(ctx context.Context) (*TracedDriveLoader, error) {
    base, err := NewDriveLoader(ctx)
    if err != nil {
        return nil, err
    }
    return &TracedDriveLoader{
        DriveLoader: base,
        tracer:      otel.Tracer("beluga.loader.drive"),
    }, nil
}

func (l *TracedDriveLoader) Load(ctx context.Context, folderID string) ([]schema.Document, error) {
    ctx, span := l.tracer.Start(ctx, "drive.load")
    defer span.End()

    span.SetAttributes(attribute.String("drive.folder_id", folderID))

    docs, err := l.DriveLoader.Load(ctx, folderID)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    span.SetAttributes(attribute.Int("drive.document_count", len(docs)))
    return docs, nil
}
```

### Pagination for Large Folders

The Drive API returns a maximum of 100 files per page by default. Handle pagination for folders with many files:

```go
func (l *DriveLoader) LoadAll(ctx context.Context, folderID string) ([]schema.Document, error) {
    query := fmt.Sprintf("'%s' in parents and trashed=false", folderID)
    var docs []schema.Document
    pageToken := ""

    for {
        call := l.service.Files.List().
            Q(query).
            Fields("nextPageToken, files(id, name, mimeType)").
            PageSize(100).
            Context(ctx)

        if pageToken != "" {
            call = call.PageToken(pageToken)
        }

        result, err := call.Do()
        if err != nil {
            return nil, fmt.Errorf("list page: %w", err)
        }

        for _, file := range result.Files {
            content, err := l.downloadFile(ctx, file.Id, file.MimeType)
            if err != nil {
                log.Printf("skipping %s: %v", file.Name, err)
                continue
            }
            docs = append(docs, schema.Document{
                Content: content,
                Metadata: map[string]any{
                    "source":    fmt.Sprintf("drive://%s", file.Id),
                    "name":      file.Name,
                    "mime_type": file.MimeType,
                },
            })
        }

        if result.NextPageToken == "" {
            break
        }
        pageToken = result.NextPageToken
    }

    return docs, nil
}
```

## Configuration

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to service account or OAuth credentials JSON | - | Yes |
| `FolderID` | Google Drive folder ID (from the folder URL) | - | Yes |
| Export format | MIME type for exporting Workspace files | `text/plain` | No |

## Troubleshooting

**"Google Drive API has not been used in project"** -- Enable the Drive API in the Google Cloud Console under **APIs & Services > Library**.

**"The caller does not have permission"** -- Verify that the service account or OAuth user has access to the target folder. For service accounts, share the folder with the service account email address.

**"File not exportable"** -- Only Google Workspace files support the Export endpoint. Regular files (PDFs, images, etc.) must use the direct download path.

## Production Considerations

- Use service accounts for server-side access; OAuth 2.0 is better suited for user-facing applications
- Request the minimal OAuth scope needed: `https://www.googleapis.com/auth/drive.readonly`
- Handle Drive API rate limits (currently 12,000 queries per minute per project) with exponential backoff
- Implement incremental sync using `modifiedTime` filters to avoid re-processing unchanged files
- For large folders, use pagination and process files in batches
- Store processed file IDs to enable deduplication across runs

## Related Resources

- [Document Loaders](/integrations/document-loaders) -- All document loader integrations
- [S3 Event-Driven Loader](/integrations/s3-event-loader) -- AWS S3 document loading
- [Embedding Providers](/integrations/embedding-providers) -- Generating embeddings for loaded documents
