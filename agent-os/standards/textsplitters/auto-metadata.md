# Auto-Metadata Tagging

Chunks automatically receive position metadata.

```go
func (s *Splitter) SplitDocuments(ctx context.Context, documents []schema.Document) ([]schema.Document, error) {
    for _, doc := range documents {
        chunks, err := s.SplitText(ctx, doc.PageContent)

        for i, chunk := range chunks {
            chunkDoc := schema.Document{
                PageContent: chunk,
                Metadata:    make(map[string]string),
            }

            // Copy original metadata
            for k, v := range doc.Metadata {
                chunkDoc.Metadata[k] = v
            }

            // Add chunk position (always added)
            chunkDoc.Metadata["chunk_index"] = strconv.Itoa(i)
            chunkDoc.Metadata["chunk_total"] = strconv.Itoa(len(chunks))

            allChunks = append(allChunks, chunkDoc)
        }
    }
}
```

## Metadata Keys
| Key | Type | Value |
|-----|------|-------|
| `chunk_index` | string | "0", "1", "2", ... |
| `chunk_total` | string | Total chunks from this document |
| (original keys) | string | Preserved from source document |

## Current Behavior
- Always added, not configurable
- Original document metadata is copied to all chunks
- Values are strings (strconv.Itoa)

## Future Improvement
Should be configurable via option:
```go
WithChunkMetadata(enabled bool)
```
Not all use cases need position tracking.
