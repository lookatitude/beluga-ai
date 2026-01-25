# Google Drive API Scraper

Welcome, colleague! In this integration guide, we're going to create a document loader that scrapes documents from Google Drive using the Google Drive API. This enables loading documents from Google Drive into your Beluga AI RAG pipeline.

## What you will build

You will create a custom document loader that connects to Google Drive API, lists documents, and loads them into Beluga AI's document format for processing in RAG pipelines.

## Learning Objectives

- ✅ Configure Google Drive API access
- ✅ Create a Google Drive document loader
- ✅ List and download Drive documents
- ✅ Integrate with Beluga AI document loaders

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Google Cloud project
- Google Drive API enabled
- OAuth credentials

## Step 1: Setup and Installation

Install Google API client:
bash
```bash
go get google.golang.org/api/drive/v3
go get google.golang.org/api/option
```

Create OAuth credentials:
1. Go to Google Cloud Console
2. Enable Drive API
3. Create OAuth 2.0 credentials
4. Download credentials JSON

Set environment variable:
bash
```bash
export GOOGLE_APPLICATION_CREDENTIALS="path/to/credentials.json"
```

## Step 2: Create Google Drive Loader

Create a custom loader for Google Drive:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "google.golang.org/api/drive/v3"
    "google.golang.org/api/option"
    "github.com/lookatitude/beluga-ai/pkg/documentloaders/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

type GoogleDriveLoader struct {
    service *drive.Service
    folderID string
}

func NewGoogleDriveLoader(folderID string) (*GoogleDriveLoader, error) {
    ctx := context.Background()
    
    service, err := drive.NewService(ctx, option.WithCredentialsFile(
        os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
    ))
    if err != nil {
        return nil, fmt.Errorf("drive service: %w", err)
    }

    return &GoogleDriveLoader{
        service:  service,
        folderID: folderID,
    }, nil
}

func (l *GoogleDriveLoader) Load(ctx context.Context) ([]schema.Document, error) {
    // List files in folder
    query := fmt.Sprintf("'%s' in parents and trashed=false", l.folderID)
    files, err := l.service.Files.List().
        Q(query).
        Fields("files(id, name, mimeType)").
        Do()
    if err != nil {
        return nil, fmt.Errorf("list files: %w", err)
    }

    var docs []schema.Document
    for _, file := range files.Files {
        // Download file
        content, err := l.downloadFile(ctx, file.Id, file.MimeType)
        if err != nil {
            log.Printf("Failed to download %s: %v", file.Name, err)
            continue
        }

        doc := schema.Document{
            PageContent: content,
            Metadata: map[string]any{
                "source": fmt.Sprintf("drive://%s", file.Id),
                "name":   file.Name,
                "type":   file.MimeType,
            },
        }
        docs = append(docs, doc)
    }

    return docs, nil
}

func (l *GoogleDriveLoader) downloadFile(ctx context.Context, fileID, mimeType string) (string, error) {
    // Handle different file types
    switch {
    case mimeType == "application/vnd.google-apps.document":
        // Export Google Docs as text
        resp, err := l.service.Files.Export(fileID, "text/plain").Download()
        if err != nil {
            return "", err
        }
        defer resp.Body.Close()
        // Read and return content
        return readContent(resp.Body), nil
    default:
        // Download directly
        resp, err := l.service.Files.Get(fileID).Download()
        if err != nil {
            return "", err
        }
        defer resp.Body.Close()
        return readContent(resp.Body), nil
    }
}

func (l *GoogleDriveLoader) LazyLoad(ctx context.Context) (<-chan any, error) {
    ch := make(chan any)
    go func() {
        defer close(ch)
        docs, err := l.Load(ctx)
        if err != nil {
            ch <- err
            return
        }
        for _, doc := range docs {
            ch <- doc
        }
    }()
    return ch, nil
}
```

## Step 3: Handle Different File Types

Support various Google Drive file types:
```go
func (l *GoogleDriveLoader) downloadFile(ctx context.Context, fileID, mimeType string) (string, error) {
    switch mimeType {
    case "application/vnd.google-apps.document":
        // Google Docs -> text/plain
        return l.exportAsText(ctx, fileID, "text/plain")
    case "application/vnd.google-apps.spreadsheet":
        // Google Sheets -> CSV
        return l.exportAsText(ctx, fileID, "text/csv")
    case "application/vnd.google-apps.presentation":
        // Google Slides -> text/plain
        return l.exportAsText(ctx, fileID, "text/plain")
    default:
        // Regular files
        return l.downloadDirect(ctx, fileID)
    }
}
```

## Step 4: Complete Integration

Here's a complete, production-ready example:
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
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type ProductionDriveLoader struct {
    service  *drive.Service
    folderID string
    tracer   trace.Tracer
}

func NewProductionDriveLoader(folderID string) (*ProductionDriveLoader, error) {
    ctx := context.Background()
    service, err := drive.NewService(ctx, option.WithCredentialsFile(
        os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
    ))
    if err != nil {
        return nil, err
    }

    return &ProductionDriveLoader{
        service:  service,
        folderID: folderID,
        tracer:   otel.Tracer("beluga.docloaders.drive"),
    }, nil
}

func (l *ProductionDriveLoader) Load(ctx context.Context) ([]schema.Document, error) {
    ctx, span := l.tracer.Start(ctx, "drive.load")
    defer span.End()

    query := fmt.Sprintf("'%s' in parents and trashed=false", l.folderID)
    files, err := l.service.Files.List().
        Q(query).
        Fields("files(id, name, mimeType)").
        Do()
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    var docs []schema.Document
    for _, file := range files.Files {
        doc, err := l.loadFile(ctx, file)
        if err != nil {
            log.Printf("Failed: %s: %v", file.Name, err)
            continue
        }
        docs = append(docs, doc)
    }

    span.SetAttributes(attribute.Int("doc_count", len(docs)))
    return docs, nil
}

func (l *ProductionDriveLoader) loadFile(ctx context.Context, file *drive.File) (schema.Document, error) {
    ctx, span := l.tracer.Start(ctx, "drive.load_file")
    defer span.End()

    span.SetAttributes(
        attribute.String("file_id", file.Id),
        attribute.String("mime_type", file.MimeType),
    )

    content, err := l.downloadFile(ctx, file.Id, file.MimeType)
    if err != nil {
        span.RecordError(err)
        return schema.Document{}, err
    }

    return schema.Document{
        PageContent: content,
        Metadata: map[string]any{
            "source": fmt.Sprintf("drive://%s", file.Id),
            "name":   file.Name,
            "type":   file.MimeType,
        },
    }, nil
}

func (l *ProductionDriveLoader) downloadFile(ctx context.Context, fileID, mimeType string) (string, error) {
    if strings.HasPrefix(mimeType, "application/vnd.google-apps") {
        // Export Google Workspace files
        exportType := "text/plain"
        if mimeType == "application/vnd.google-apps.spreadsheet" {
            exportType = "text/csv"
        }
        resp, err := l.service.Files.Export(fileID, exportType).Download()
        if err != nil {
            return "", err
        }
        defer resp.Body.Close()
        return readContent(resp.Body), nil
    }

    // Regular files
    resp, err := l.service.Files.Get(fileID).Download()
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    return readContent(resp.Body), nil
}

func readContent(r io.Reader) string {
    var b strings.Builder
    io.Copy(&b, r)
    return b.String()
}

func (l *ProductionDriveLoader) LazyLoad(ctx context.Context) (<-chan any, error) {
    ch := make(chan any)
    go func() {
        defer close(ch)
        docs, err := l.Load(ctx)
        if err != nil {
            ch <- err
            return
        }
        for _, doc := range docs {
            ch <- doc
        }
    }()
    return ch, nil
}

func main() {
    ctx := context.Background()
    loader, _ := NewProductionDriveLoader("folder-id")
    docs, _ := loader.Load(ctx)
    fmt.Printf("Loaded %d documents\n", len(docs))
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `FolderID` | Google Drive folder ID | - | Yes |
| `CredentialsFile` | OAuth credentials path | - | Yes |
| `ExportFormat` | Export format for Workspace files | `text/plain` | No |

## Common Issues

### "API not enabled"

**Problem**: Drive API not enabled.

**Solution**: Enable Drive API in Google Cloud Console.

### "Permission denied"

**Problem**: Insufficient OAuth scopes.

**Solution**: Request `drive.readonly` scope.

## Production Considerations

When using Google Drive in production:

- **OAuth scopes**: Request minimal required scopes
- **Rate limiting**: Handle API rate limits
- **File types**: Support various Google Workspace formats
- **Error handling**: Handle API failures gracefully
- **Security**: Use service accounts for server-side access

## Next Steps

Congratulations! You've created a Google Drive loader. Next, learn how to:

- **[AWS S3 Event-Driven Loader](./aws-s3-event-driven-loader.md)** - S3 integration
- **Document Loaders Documentation** - Deep dive into document loaders
- **[RAG Guide](../../guides/rag-multimodal.md)** - RAG patterns

---

**Ready for more?** Check out the Integrations Index for more integration guides!
