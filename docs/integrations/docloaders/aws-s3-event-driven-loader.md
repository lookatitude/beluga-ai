# AWS S3 Event-Driven Loader

Welcome, colleague! In this integration guide, we're going to create an event-driven document loader that automatically processes documents uploaded to AWS S3. This enables real-time document ingestion for RAG pipelines.

## What you will build

You will create a custom document loader that listens to S3 events (via SQS or Lambda) and automatically loads new documents into your Beluga AI RAG pipeline when they're uploaded to S3.

## Learning Objectives

- ✅ Create a custom S3 event-driven document loader
- ✅ Handle S3 object creation events
- ✅ Process documents from S3 buckets
- ✅ Integrate with Beluga AI document loaders

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- AWS account with S3 access
- AWS credentials configured

## Step 1: Setup and Installation

Install AWS SDK:
bash
```bash
go get github.com/aws/aws-sdk-go-v2/service/s3
go get github.com/aws/aws-sdk-go-v2/config
```

Configure AWS credentials:
```bash
export AWS_REGION="us-east-1"
export AWS_ACCESS_KEY_ID="your-key"
export AWS_SECRET_ACCESS_KEY="your-secret"
```
# Or use AWS CLI: aws configure
```

## Step 2: Create S3 Event-Driven Loader

Create a custom loader that processes S3 events:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/lookatitude/beluga-ai/pkg/documentloaders/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

type S3EventLoader struct {
    s3Client *s3.Client
    bucket   string
}

func NewS3EventLoader(bucket string) (*S3EventLoader, error) {
    cfg, err := config.LoadDefaultConfig(context.Background())
    if err != nil {
        return nil, fmt.Errorf("aws config: %w", err)
    }

    return &S3EventLoader{
        s3Client: s3.NewFromConfig(cfg),
        bucket:   bucket,
    }, nil
}

func (l *S3EventLoader) Load(ctx context.Context) ([]schema.Document, error) {
    // List objects in bucket
    result, err := l.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
        Bucket: aws.String(l.bucket),
    })
    if err != nil {
        return nil, fmt.Errorf("list objects: %w", err)
    }

    var docs []schema.Document
    for _, obj := range result.Contents {
        // Download object
        getObj, err := l.s3Client.GetObject(ctx, &s3.GetObjectInput{
            Bucket: aws.String(l.bucket),
            Key:    obj.Key,
        })
        if err != nil {
            log.Printf("Failed to get %s: %v", *obj.Key, err)
            continue
        }

        // Read content
        content := make([]byte, *obj.Size)
        _, err = getObj.Body.Read(content)
        getObj.Body.Close()
        if err != nil {
            log.Printf("Failed to read %s: %v", *obj.Key, err)
            continue
        }

        // Create document
        doc := schema.Document{
            PageContent: string(content),
            Metadata: map[string]any{
                "source": fmt.Sprintf("s3://%s/%s", l.bucket, *obj.Key),
                "key":    *obj.Key,
                "size":   *obj.Size,
            },
        }
        docs = append(docs, doc)
    }

    return docs, nil
}

func (l *S3EventLoader) LazyLoad(ctx context.Context) (<-chan any, error) {
    ch := make(chan any)
    go func() {
        defer close(ch)
        docs, err := l.Load(ctx)
        if err != nil {
            ch \<- err
            return
        }
        for _, doc := range docs {
            ch \<- doc
        }
    }()
    return ch, nil
}
```

## Step 3: Handle S3 Events

Process S3 events from SQS or Lambda:
```go
type S3Event struct {
    Records []struct {
        S3 struct {
            Bucket struct {
                Name string `json:"name"`
            } `json:"bucket"`
            Object struct {
                Key string `json:"key"`
            } `json:"object"`
        } `json:"s3"`
    } `json:"Records"`
}

func (l *S3EventLoader) ProcessEvent(ctx context.Context, event S3Event) error {
    for _, record := range event.Records {
        bucket := record.S3.Bucket.Name
        key := record.S3.Object.Key

        // Download and process
        getObj, err := l.s3Client.GetObject(ctx, &s3.GetObjectInput{
            Bucket: aws.String(bucket),
            Key:    aws.String(key),
        })
        if err != nil {
            return fmt.Errorf("get object: %w", err)
        }

        content := make([]byte, getObj.ContentLength)
        getObj.Body.Read(content)
        getObj.Body.Close()

        // Process document
        doc := schema.Document{
            PageContent: string(content),
            Metadata: map[string]any{
                "source": fmt.Sprintf("s3://%s/%s", bucket, key),
            },
        }

        // Add to vector store or process
        fmt.Printf("Processed: %s\n", key)
    }


    return nil
}
```

## Step 4: Lambda Handler

Create a Lambda function handler:
```go
func HandleLambdaEvent(ctx context.Context, event S3Event) error {
    loader, err := NewS3EventLoader("your-bucket")
    if err != nil {
        return err
    }


    return loader.ProcessEvent(ctx, event)
}
```

## Step 5: Complete Integration

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/lookatitude/beluga-ai/pkg/documentloaders/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type ProductionS3Loader struct {
    s3Client *s3.Client
    bucket   string
    tracer   trace.Tracer
}

func NewProductionS3Loader(bucket string) (*ProductionS3Loader, error) {
    cfg, err := config.LoadDefaultConfig(context.Background())
    if err != nil {
        return nil, err
    }

    return &ProductionS3Loader{
        s3Client: s3.NewFromConfig(cfg),
        bucket:   bucket,
        tracer:   otel.Tracer("beluga.docloaders.s3"),
    }, nil
}

func (l *ProductionS3Loader) Load(ctx context.Context) ([]schema.Document, error) {
    ctx, span := l.tracer.Start(ctx, "s3.load")
    defer span.End()

    span.SetAttributes(attribute.String("bucket", l.bucket))

    result, err := l.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
        Bucket: aws.String(l.bucket),
    })
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    var docs []schema.Document
    for _, obj := range result.Contents {
        doc, err := l.loadObject(ctx, *obj.Key)
        if err != nil {
            log.Printf("Failed to load %s: %v", *obj.Key, err)
            continue
        }
        docs = append(docs, doc)
    }

    span.SetAttributes(attribute.Int("doc_count", len(docs)))
    return docs, nil
}

func (l *ProductionS3Loader) loadObject(ctx context.Context, key string) (schema.Document, error) {
    ctx, span := l.tracer.Start(ctx, "s3.load_object")
    defer span.End()

    span.SetAttributes(attribute.String("key", key))

    getObj, err := l.s3Client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(l.bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        span.RecordError(err)
        return schema.Document{}, err
    }
    defer getObj.Body.Close()

    content := make([]byte, *getObj.ContentLength)
    if _, err := getObj.Body.Read(content); err != nil {
        span.RecordError(err)
        return schema.Document{}, err
    }

    return schema.Document{
        PageContent: string(content),
        Metadata: map[string]any{
            "source": fmt.Sprintf("s3://%s/%s", l.bucket, key),
            "key":    key,
        },
    }, nil
}

func (l *ProductionS3Loader) LazyLoad(ctx context.Context) (<-chan any, error) {
    ch := make(chan any)
    go func() {
        defer close(ch)
        docs, err := l.Load(ctx)
        if err != nil {
            ch \<- err
            return
        }
        for _, doc := range docs {
            ch \<- doc
        }
    }()
    return ch, nil
}

func main() {
    ctx := context.Background()
    loader, _ := NewProductionS3Loader("your-bucket")
    docs, _ := loader.Load(ctx)
    fmt.Printf("Loaded %d documents\n", len(docs))
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `Bucket` | S3 bucket name | - | Yes |
| `Region` | AWS region | `us-east-1` | No |
| `Prefix` | Object key prefix | - | No |

## Common Issues

### "Access denied"

**Problem**: Missing S3 permissions.

**Solution**: Ensure IAM role has `s3:GetObject` permission.

### "Bucket not found"

**Problem**: Wrong bucket name.

**Solution**: Verify bucket name and region.

## Production Considerations

When using S3 event-driven loading in production:

- **Event sources**: Use SQS or Lambda for events
- **Error handling**: Handle failed downloads gracefully
- **Rate limiting**: Respect S3 rate limits
- **Cost optimization**: Use appropriate storage classes
- **Security**: Use IAM roles, not access keys

## Next Steps

Congratulations! You've created an S3 event-driven loader. Next, learn how to:

- **[Google Drive API Scraper](./google-drive-api-scraper.md)** - Google Drive integration
- **Document Loaders Documentation** - Deep dive into document loaders
- **[RAG Guide](../../guides/rag-multimodal.md)** - RAG patterns

---

**Ready for more?** Check out the Integrations Index for more integration guides!
