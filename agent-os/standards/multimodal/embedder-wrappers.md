# Embedder Wrappers

Wrapper types adapt embedders to `vectorstores.Embedder` interface.

## Why Wrappers?
1. **Import cycle avoidance** - multimodal can't import vectorstores directly
2. **Interface adaptation** - different embedder types work with vectorstore without modification

## Wrapper Types
```go
// For regular text embedders
type embedderWrapper struct {
    embedder embeddingsiface.Embedder
}

// For multimodal embedders (preserves multimodal capability)
type multimodalEmbedderWrapper struct {
    embedder  embeddingsiface.MultimodalEmbedder
    documents []schema.Document // For multimodal context
}
```

## Limitation
Multimodal wrapper converts text to Document, losing multimodal metadata:
```go
// Simplified conversion - loses image/audio metadata
docs[i] = schema.NewDocument(text, nil)
```

## Usage in StoreMultimodalDocuments
```go
if m.multimodalEmbedder != nil {
    embedder = &multimodalEmbedderWrapper{...}
} else if m.embedder != nil {
    embedder = &embedderWrapper{...}
}
opts := []vectorstores.Option{vectorstores.WithEmbedder(embedder)}
```
