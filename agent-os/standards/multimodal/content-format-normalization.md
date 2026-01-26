# Content Format Normalization

ContentBlocks have three formats: `base64`, `url`, `file_path`.

```go
normalizer.Normalize(ctx, block, "base64")  // Convert to in-memory
normalizer.Normalize(ctx, block, "file_path") // Write to temp file
normalizer.Normalize(ctx, block, "url")      // NOT IMPLEMENTED
```

## Conversion Rules
- `base64` → `file_path`: Creates temp file, marks metadata `temp_file: true`
- `url` → `base64`: Fetches via HTTP with context-aware request
- `url` → `file_path`: Fetches and writes to temp file
- ANY → `url`: Returns error (requires external upload service - future feature)

## Temp File Cleanup
- Caller must check `block.Metadata["temp_file"]` and delete when done
- Files use pattern `multimodal_*.{ext}` in system temp dir
- Extension derived from MIME type or defaults (image→png, audio→mp3, video→mp4)

## Format Detection
```go
// Detected by field presence, not explicit flag
Data non-empty, URL/FilePath empty → base64
URL non-empty → url
FilePath non-empty → file_path
```
