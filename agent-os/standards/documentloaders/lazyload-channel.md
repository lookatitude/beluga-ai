# LazyLoad Channel Pattern

Streaming document loading via single channel.

```go
type Loader interface {
    Load(ctx context.Context) ([]schema.Document, error)
    LazyLoad(ctx context.Context) (<-chan any, error)  // Returns docs OR errors
}
```

## Consumer Pattern
```go
ch, _ := loader.LazyLoad(ctx)
for item := range ch {
    switch v := item.(type) {
    case schema.Document:
        // handle document
    case error:
        // handle error (stream ends after error)
    }
}
```

## Current Design
- Single `<-chan any` yields Document or error
- First error terminates the stream
- Channel closed on completion

## Note: Planned Refactoring
This pattern is not idiomatic Go. Future versions should use:
```go
LazyLoad(ctx context.Context) (<-chan schema.Document, <-chan error, error)
```
or
```go
LazyLoad(ctx context.Context) (<-chan LoadResult, error)
type LoadResult struct {
    Document schema.Document
    Err      error
}
```
