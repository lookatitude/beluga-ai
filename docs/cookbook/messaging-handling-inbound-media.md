---
title: "Handling Inbound Media (MediaSIDs)"
package: "messaging"
category: "messaging"
complexity: "intermediate"
---

# Handling Inbound Media (MediaSIDs)

## Problem

You need to handle inbound media attachments (images, audio, video) sent via messaging platforms like Twilio, which provide MediaSIDs (media identifiers) that need to be downloaded and processed.

## Solution

Implement a media handler that receives MediaSIDs, downloads media files from the messaging platform, stores them temporarily, processes them with appropriate services (vision models, audio transcription), and cleans up after processing. This works because messaging platforms provide APIs to download media using MediaSIDs.

## Code Example
```go
package main

import (
    "context"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "time"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("beluga.messaging.media")

// MediaHandler handles inbound media files
type MediaHandler struct {
    downloadClient *http.Client
    tempDir        string
    cleanupDelay   time.Duration
}

// NewMediaHandler creates a new media handler
func NewMediaHandler(tempDir string, cleanupDelay time.Duration) *MediaHandler {
    return &MediaHandler{
        downloadClient: &http.Client{Timeout: 30 * time.Second},
        tempDir:        tempDir,
        cleanupDelay:   cleanupDelay,
    }
}

// DownloadMedia downloads media from MediaSID
func (mh *MediaHandler) DownloadMedia(ctx context.Context, mediaSID string, mediaURL string) (string, error) {
    ctx, span := tracer.Start(ctx, "media_handler.download")
    defer span.End()
    
    span.SetAttributes(
        attribute.String("media_sid", mediaSID),
        attribute.String("media_url", mediaURL),
    )
    
    // Create temp file
    tempFile := filepath.Join(mh.tempDir, mediaSID+".tmp")
    file, err := os.Create(tempFile)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return "", fmt.Errorf("failed to create temp file: %w", err)
    }
    defer file.Close()
    
    // Download media
    req, err := http.NewRequestWithContext(ctx, "GET", mediaURL, nil)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return "", err
    }
    
    resp, err := mh.downloadClient.Do(req)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return "", fmt.Errorf("failed to download: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        err := fmt.Errorf("unexpected status: %d", resp.StatusCode)
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return "", err
    }
    
    // Copy to file
    _, err = io.Copy(file, resp.Body)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return "", fmt.Errorf("failed to save: %w", err)
    }
    
    span.SetAttributes(attribute.String("temp_file", tempFile))
    span.SetStatus(trace.StatusOK, "media downloaded")
    
    return tempFile, nil
}

// ProcessMedia processes downloaded media
func (mh *MediaHandler) ProcessMedia(ctx context.Context, mediaPath string, mediaType string) (interface{}, error) {
    ctx, span := tracer.Start(ctx, "media_handler.process")
    defer span.End()
    
    span.SetAttributes(
        attribute.String("media_path", mediaPath),
        attribute.String("media_type", mediaType),
    )
    
    switch mediaType {
    case "image/jpeg", "image/png":
        return mh.processImage(ctx, mediaPath)
    case "audio/wav", "audio/mpeg":
        return mh.processAudio(ctx, mediaPath)
    default:
        return nil, fmt.Errorf("unsupported media type: %s", mediaType)
    }
}

// processImage processes image media
func (mh *MediaHandler) processImage(ctx context.Context, imagePath string) (interface{}, error) {
    // Process with vision model
    // result := visionModel.Process(ctx, imagePath)
    return "image processed", nil
}

// processAudio processes audio media
func (mh *MediaHandler) processAudio(ctx context.Context, audioPath string) (interface{}, error) {
    // Process with STT
    // result := sttProvider.Transcribe(ctx, audioPath)
    return "audio transcribed", nil
}

// ScheduleCleanup schedules media file cleanup
func (mh *MediaHandler) ScheduleCleanup(ctx context.Context, mediaPath string) {
    go func() {
        time.Sleep(mh.cleanupDelay)
        os.Remove(mediaPath)
    }()
}

func main() {
    ctx := context.Background()

    // Create handler
    handler := NewMediaHandler("/tmp/media", 1*time.Hour)
    
    // Download and process
    mediaURL := "https://api.twilio.com/2010-04-01/Accounts/.../Media/ME123"
    mediaPath, err := handler.DownloadMedia(ctx, "ME123", mediaURL)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    
    // Process
    result, err := handler.ProcessMedia(ctx, mediaPath, "image/jpeg")
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    // Schedule cleanup
    handler.ScheduleCleanup(ctx, mediaPath)
```
    
    fmt.Printf("Processed: %v\n", result)
}

## Explanation

Let's break down what's happening:

1. **Media download** - Notice how we download media using the provided URL. The MediaSID is used for identification, while the URL is used for actual download.

2. **Type-based processing** - We route media to appropriate processors based on MIME type. Images go to vision models, audio to STT, etc.

3. **Automatic cleanup** - We schedule cleanup of temporary files after processing. This prevents disk space issues while allowing time for processing.

```go
**Key insight:** Always clean up temporary media files. Store them temporarily during processing, then delete them to prevent disk space issues.

## Testing

```
Here's how to test this solution:
```go
func TestMediaHandler_DownloadsMedia(t *testing.T) {
    handler := NewMediaHandler(t.TempDir(), 1*time.Hour)
    
    // Mock download
    mediaPath, err := handler.DownloadMedia(context.Background(), "ME123", "http://example.com/media")
    require.NoError(t, err)
    require.FileExists(t, mediaPath)
}

## Variations

### Streaming Processing

Process media as it downloads:
func (mh *MediaHandler) StreamProcess(ctx context.Context, mediaSID string, mediaURL string) (<-chan interface{}, error) {
    // Process while downloading
}
```

### Media Caching

Cache frequently accessed media:
```go
type CachedMediaHandler struct {
    cache map[string]string
}
```
## Related Recipes

- **[Messaging Conversation Expiry Logic](./messaging-conversation-expiry-logic.md)** - Manage conversation lifecycle
- **[Multimodal Processing Multiple Images](./multimodal-processing-multiple-images-per-prompt.md)** - Process multiple images
- **[Messaging Package Guide](../package_design_patterns.md)** - For a deeper understanding of messaging
