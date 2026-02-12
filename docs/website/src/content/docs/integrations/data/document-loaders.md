---
title: Document Loaders
description: "Load documents from filesystems, cloud storage, databases, and APIs into Beluga AI RAG pipelines for embedding and retrieval."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "document loaders, Beluga AI, file ingestion, cloud storage loader, RAG pipeline, data loading Go, document processing"
---

Before documents can be embedded and searched, they need to be loaded from wherever they live -- local filesystems, cloud storage buckets, databases, APIs, or SaaS platforms. Document loaders handle this ingestion step, converting source data into Beluga AI's `schema.Document` type with content and metadata preserved.

This guide covers the available loader integrations and shows how to implement custom loaders for specialized sources.

## Available Loaders

### Filesystem Loaders

#### Directory Loader

Load documents recursively from local filesystem.

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/documentloaders"
)

fsys := os.DirFS("/path/to/documents")
loader, err := documentloaders.NewDirectoryLoader(fsys,
    documentloaders.WithMaxDepth(10),
    documentloaders.WithExtensions(".md", ".txt", ".pdf"),
    documentloaders.WithExclusions("**/node_modules/**", "**/.git/**"),
)

docs, err := loader.Load(ctx)
```

**Supported formats**: txt, md, html, pdf, docx, csv, json

#### PDF Loader

Extract text from PDF files.

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/documentloaders/providers/pdf"
)

loader := pdf.NewLoader("/path/to/file.pdf",
    pdf.WithExtractImages(false),
    pdf.WithPreserveFormatting(true),
    pdf.WithPageSeparator("\n---\n"),
)

docs, err := loader.Load(ctx)
```

### Cloud Storage Loaders

#### AWS S3 Loader

Load documents from S3 buckets.

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/documentloaders/providers/s3"
    "github.com/aws/aws-sdk-go/aws/session"
)

sess, _ := session.NewSession(&aws.Config{
    Region: aws.String("us-west-2"),
})

loader := s3.NewLoader(sess, "my-bucket",
    s3.WithPrefix("documents/"),
    s3.WithRecursive(true),
    s3.WithIncludeExtensions(".txt", ".md", ".pdf"),
)

docs, err := loader.Load(ctx)
```

**Configuration**:
- Supports IAM roles and access keys
- Handles large files via multipart downloads
- Supports S3-compatible storage (MinIO, DigitalOcean Spaces)

#### Google Cloud Storage

Load from GCS buckets.

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/documentloaders/providers/gcs"
    "cloud.google.com/go/storage"
)

client, _ := storage.NewClient(ctx)

loader := gcs.NewLoader(client, "my-bucket",
    gcs.WithPrefix("documents/"),
    gcs.WithRecursive(true),
)

docs, err := loader.Load(ctx)
```

#### Azure Blob Storage

Load from Azure containers.

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/documentloaders/providers/azureblob"
)

loader := azureblob.NewLoader(
    "account-name",
    "container-name",
    azureblob.WithAccountKey(os.Getenv("AZURE_STORAGE_KEY")),
    azureblob.WithPrefix("documents/"),
)

docs, err := loader.Load(ctx)
```

### Web Loaders

#### HTML Loader

Scrape and parse HTML content.

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/documentloaders/providers/html"
)

loader := html.NewLoader("https://example.com",
    html.WithRemoveScripts(true),
    html.WithRemoveStyles(true),
    html.WithExtractMetadata(true),
    html.WithFollowLinks(false),
)

docs, err := loader.Load(ctx)
```

#### Sitemap Crawler

Crawl entire websites via sitemaps.

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/documentloaders/providers/sitemap"
)

loader := sitemap.NewLoader("https://example.com/sitemap.xml",
    sitemap.WithMaxDepth(3),
    sitemap.WithConcurrency(5),
    sitemap.WithRateLimit(10), // requests per second
)

docs, err := loader.Load(ctx)
```

### Database Loaders

#### SQL Loader

Load data from SQL databases.

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/documentloaders/providers/sql"
    "database/sql"
    _ "github.com/lib/pq"
)

db, _ := sql.Open("postgres", "postgres://user:pass@localhost/db")

loader := sql.NewLoader(db,
    sql.WithQuery("SELECT id, title, content FROM articles"),
    sql.WithContentColumn("content"),
    sql.WithMetadataColumns("id", "title"),
)

docs, err := loader.Load(ctx)
```

#### MongoDB Loader

Load documents from MongoDB collections.

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/documentloaders/providers/mongodb"
    "go.mongodb.org/mongo-driver/mongo"
)

client, _ := mongo.Connect(ctx)

loader := mongodb.NewLoader(
    client.Database("mydb").Collection("documents"),
    mongodb.WithFilter(bson.M{"status": "published"}),
    mongodb.WithContentField("content"),
    mongodb.WithMetadataFields("title", "author", "created_at"),
)

docs, err := loader.Load(ctx)
```

### API Loaders

#### REST API Loader

Load data from REST APIs.

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/documentloaders/providers/restapi"
)

loader := restapi.NewLoader("https://api.example.com/documents",
    restapi.WithHeaders(map[string]string{
        "Authorization": "Bearer " + os.Getenv("API_TOKEN"),
    }),
    restapi.WithPagination(restapi.PaginationConfig{
        Type:      "offset",
        PageSize:  100,
        PageParam: "offset",
    }),
    restapi.WithContentPath("data.content"),
)

docs, err := loader.Load(ctx)
```

#### Google Drive Loader

Load documents from Google Drive.

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/documentloaders/providers/googledrive"
    "google.golang.org/api/drive/v3"
)

service, _ := drive.NewService(ctx)

loader := googledrive.NewLoader(service,
    googledrive.WithFolderID("folder-id"),
    googledrive.WithRecursive(true),
    googledrive.WithMimeTypes("application/pdf", "text/plain"),
)

docs, err := loader.Load(ctx)
```

## Implementing Custom Loaders

Create custom loaders for specialized sources.

```go
package custom

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

type CustomLoader struct {
    source     string
    apiClient  *APIClient
    maxResults int
}

func NewCustomLoader(source string, apiKey string) *CustomLoader {
    return &CustomLoader{
        source:     source,
        apiClient:  NewAPIClient(apiKey),
        maxResults: 1000,
    }
}

func (cl *CustomLoader) Load(ctx context.Context) ([]schema.Document, error) {
    // Fetch data from custom source
    items, err := cl.apiClient.FetchItems(ctx, cl.source, cl.maxResults)
    if err != nil {
        return nil, fmt.Errorf("fetch items: %w", err)
    }

    // Convert to Beluga documents
    docs := make([]schema.Document, len(items))
    for i, item := range items {
        docs[i] = schema.Document{
            PageContent: item.Content,
            Metadata: map[string]interface{}{
                "source":     cl.source,
                "id":         item.ID,
                "created_at": item.CreatedAt,
                "author":     item.Author,
            },
        }
    }

    return docs, nil
}

func (cl *CustomLoader) LazyLoad(ctx context.Context) (chan schema.DocumentResult, error) {
    ch := make(chan schema.DocumentResult, 10)

    go func() {
        defer close(ch)

        offset := 0
        pageSize := 100

        for {
            items, err := cl.apiClient.FetchItemsPaginated(ctx, cl.source, offset, pageSize)
            if err != nil {
                ch <- schema.DocumentResult{Error: err}
                return
            }

            if len(items) == 0 {
                return
            }

            for _, item := range items {
                doc := schema.Document{
                    PageContent: item.Content,
                    Metadata: map[string]interface{}{
                        "source": cl.source,
                        "id":     item.ID,
                    },
                }

                select {
                case <-ctx.Done():
                    ch <- schema.DocumentResult{Error: ctx.Err()}
                    return
                case ch <- schema.DocumentResult{Document: doc}:
                }
            }

            offset += pageSize
        }
    }()

    return ch, nil
}
```

## Event-Driven Loading

Load documents reactively based on events.

### S3 Event-Driven Loader

Process new files as they arrive in S3.

```go
import (
    "github.com/aws/aws-sdk-go/service/sqs"
)

type S3EventLoader struct {
    sqsClient *sqs.SQS
    queueURL  string
    s3Loader  *s3.Loader
}

func NewS3EventLoader(sess *session.Session, queueURL string, bucket string) *S3EventLoader {
    return &S3EventLoader{
        sqsClient: sqs.New(sess),
        queueURL:  queueURL,
        s3Loader:  s3.NewLoader(sess, bucket),
    }
}

func (sel *S3EventLoader) ProcessEvents(ctx context.Context) error {
    for {
        // Poll SQS for S3 events
        result, err := sel.sqsClient.ReceiveMessageWithContext(ctx, &sqs.ReceiveMessageInput{
            QueueUrl:            aws.String(sel.queueURL),
            MaxNumberOfMessages: aws.Int64(10),
            WaitTimeSeconds:     aws.Int64(20),
        })
        if err != nil {
            return err
        }

        for _, msg := range result.Messages {
            // Parse S3 event
            event, err := parseS3Event(msg.Body)
            if err != nil {
                log.Printf("Parse event error: %v", err)
                continue
            }

            // Load document
            doc, err := sel.s3Loader.LoadSingleFile(ctx, event.ObjectKey)
            if err != nil {
                log.Printf("Load document error: %v", err)
                continue
            }

            // Process document (e.g., embed and index)
            if err := processDocument(ctx, doc); err != nil {
                log.Printf("Process document error: %v", err)
                continue
            }

            // Delete message from queue
            sel.sqsClient.DeleteMessage(&sqs.DeleteMessageInput{
                QueueUrl:      aws.String(sel.queueURL),
                ReceiptHandle: msg.ReceiptHandle,
            })
        }

        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
    }
}
```

## Best Practices

When working with document loaders:

1. **Use lazy loading** for large datasets
2. **Implement retry logic** for network failures
3. **Add rate limiting** to respect API limits
4. **Cache credentials** securely
5. **Validate document content** before processing
6. **Track loading metrics** (throughput, errors)
7. **Handle pagination** correctly
8. **Process in batches** for efficiency
9. **Add comprehensive metadata** for filtering
10. **Implement checkpointing** for resumability

## Next Steps

- Learn about [Text Splitter Integrations](/integrations/text-splitters) for chunking
- Explore [Document Processing](/guides/document-processing) for complete pipelines
- Read [RAG Pipeline](/guides/rag-pipeline) for integration patterns
