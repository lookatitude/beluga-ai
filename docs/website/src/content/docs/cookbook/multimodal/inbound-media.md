---
title: "Handling Inbound Media"
description: "Handle inbound media attachments from messaging platforms by downloading, processing, and cleaning up media files."
---

## Problem

You need to handle inbound media attachments (images, audio, video) sent via messaging platforms like Twilio, which provide MediaSIDs (media identifiers) that need to be downloaded and processed.

## Solution

Implement a media handler that receives MediaSIDs, downloads media files from the messaging platform, stores them temporarily, processes them with appropriate services (vision models, audio transcription), and cleans up after processing. Messaging platforms provide APIs to download media using MediaSIDs.

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

// MediaHandler handles inbound media files.
type MediaHandler struct {
    downloadClient *http.Client
    tempDir        string
    cleanupDelay   time.Duration
}

// NewMediaHandler creates a new media handler.
func NewMediaHandler(tempDir string, cleanupDelay time.Duration) *MediaHandler {
    return &MediaHandler{
        downloadClient: &http.Client{Timeout: 30 * time.Second},
        tempDir:        tempDir,
        cleanupDelay:   cleanupDelay,
    }
}

// DownloadMedia downloads media from a platform URL using the MediaSID for identification.
func (mh *MediaHandler) DownloadMedia(ctx context.Context, mediaSID string, mediaURL string) (string, error) {
    ctx, span := tracer.Start(ctx, "media_handler.download")
    defer span.End()

    span.SetAttributes(
        attribute.String("media_sid", mediaSID),
        attribute.String("media_url", mediaURL),
    )

    tempFile := filepath.Join(mh.tempDir, mediaSID+".tmp")
    file, err := os.Create(tempFile)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return "", fmt.Errorf("failed to create temp file: %w", err)
    }
    defer file.Close()

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

    if _, err = io.Copy(file, resp.Body); err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return "", fmt.Errorf("failed to save: %w", err)
    }

    span.SetAttributes(attribute.String("temp_file", tempFile))
    span.SetStatus(trace.StatusOK, "media downloaded")

    return tempFile, nil
}

// ProcessMedia processes downloaded media based on MIME type.
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

// processImage processes image media using a vision model.
func (mh *MediaHandler) processImage(ctx context.Context, imagePath string) (interface{}, error) {
    // Process with vision model
    // result := visionModel.Process(ctx, imagePath)
    return "image processed", nil
}

// processAudio processes audio media using STT.
func (mh *MediaHandler) processAudio(ctx context.Context, audioPath string) (interface{}, error) {
    // Process with STT provider
    // result := sttProvider.Transcribe(ctx, audioPath)
    return "audio transcribed", nil
}

// ScheduleCleanup schedules media file cleanup after a delay.
func (mh *MediaHandler) ScheduleCleanup(ctx context.Context, mediaPath string) {
    go func() {
        time.Sleep(mh.cleanupDelay)
        os.Remove(mediaPath)
    }()
}

func main() {
    ctx := context.Background()

    handler := NewMediaHandler("/tmp/media", 1*time.Hour)

    mediaURL := "https://api.twilio.com/2010-04-01/Accounts/.../Media/ME123"
    mediaPath, err := handler.DownloadMedia(ctx, "ME123", mediaURL)
    if err != nil {
        log.Fatalf("Failed to download: %v", err)
    }

    result, err := handler.ProcessMedia(ctx, mediaPath, "image/jpeg")
    if err != nil {
        log.Fatalf("Failed to process: %v", err)
    }

    handler.ScheduleCleanup(ctx, mediaPath)

    fmt.Printf("Processed: %v\n", result)
}
```

## Explanation

1. **Media download** — Media is downloaded using the platform-provided URL. The MediaSID serves as a unique identifier for temporary file naming and tracking.

2. **Type-based processing** — Media is routed to appropriate processors based on MIME type. Images go to vision models, audio to STT providers, and so on. This allows extensible media handling.

3. **Automatic cleanup** — Temporary files are scheduled for cleanup after processing. This prevents disk space issues while allowing sufficient time for processing to complete.

4. **Context propagation** — HTTP requests use `http.NewRequestWithContext` to propagate cancellation and deadlines through the download operation.

## Variations

### Streaming Processing

Process media as it downloads:

```go
func (mh *MediaHandler) StreamProcess(ctx context.Context, mediaSID string, mediaURL string) (<-chan interface{}, error) {
    // Process while downloading for lower latency
}
```

### Media Caching

Cache frequently accessed media to avoid re-downloading:

```go
type CachedMediaHandler struct {
    cache map[string]string
}
```

## Related Recipes

- [Conversation Expiry Logic](/cookbook/conversation-expiry) — Manage conversation lifecycle
- [Memory TTL and Cleanup](/cookbook/memory-ttl-cleanup) — Automatic resource cleanup strategies
