# Streaming Chunk Sizes

ProcessStream chunks large media for incremental processing.

## Chunk Sizes
```go
const videoChunkSize = 1024 * 1024 // 1MB
const audioChunkSize = 64 * 1024   // 64KB
```
- Text and images: no chunking (processed whole)
- Sizes determined empirically; may need adjustment per use case

## Chunk Metadata
Each chunk includes tracking metadata:
```go
block.Metadata = map[string]any{
    "chunk_offset": offset,    // Byte offset in original
    "chunk_size":   len(data), // This chunk's size
    "total_size":   totalSize, // Original file size
}
```

## Stream Interruption
- Calling ProcessStream with same input ID cancels previous stream
- Cancel functions stored in `streamingState.activeStreams[inputID]`
- Cleanup happens on stream completion or cancellation

## Channel Buffering
```go
ch := make(chan *MultimodalOutput, 10) // Buffered for throughput
```
