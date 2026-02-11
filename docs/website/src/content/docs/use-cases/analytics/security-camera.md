---
title: Security Camera Event Analysis
description: Analyze video feeds from security cameras in real-time to detect events, identify threats, and generate alerts.
---

Security camera networks generate massive volumes of video — a facility with 100 cameras at 30fps produces 260,000 frames per minute. Human operators monitoring these feeds suffer from attention fatigue after 20-30 minutes, missing critical events during exactly the periods when threats are most likely. Scaling manual monitoring linearly with camera count is economically unsustainable.

Traditional computer vision (motion detection, object recognition) catches basic events but cannot understand context: a person entering through a door is normal, but the same person entering through a window at 3 AM is a threat. Vision-capable LLMs bridge this gap by understanding scene context, human behavior patterns, and situational risk — the same capabilities that make them useful for image understanding in other domains.

The challenge is cost and latency. Analyzing every frame with an LLM at $0.01 per image would cost $2,600 per minute for 100 cameras. Frame sampling, motion pre-filtering, and priority-based analysis rates make LLM-powered video analysis economically viable.

## Solution Architecture

Beluga AI's multimodal support (via `schema.ImagePart`) sends frames to vision-capable LLMs for contextual analysis. The system samples frames at configurable intervals (2-60 seconds depending on camera priority), pre-filters with lightweight motion detection to skip static scenes, and processes multiple camera feeds in parallel using goroutines with bounded concurrency.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ Camera       │───▶│ Frame        │───▶│ Multimodal   │
│ Video Stream │    │ Extractor    │    │ LLM          │
└──────────────┘    └──────────────┘    └──────┬───────┘
                                                │
                                                ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ Alert System │◀───│ Threat       │◀───│ Event        │
│ (Security    │    │ Classifier   │    │ Detector     │
│  Team)       │    └──────────────┘    └──────────────┘
└──────────────┘
```

## Video Stream Processing

Process video streams with frame extraction and sampling:

```go
package main

import (
    "context"
    "encoding/base64"
    "fmt"
    "time"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

// SecurityAnalyzer analyzes security camera feeds
type SecurityAnalyzer struct {
    model llm.ChatModel
}

func NewSecurityAnalyzer(ctx context.Context) (*SecurityAnalyzer, error) {
    // Use vision-capable model
    model, err := llm.New("openai", llm.ProviderConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4o",
    })
    if err != nil {
        return nil, err
    }

    return &SecurityAnalyzer{model: model}, nil
}

// AnalyzeFrame analyzes a single frame for security events
func (a *SecurityAnalyzer) AnalyzeFrame(ctx context.Context, frameData []byte, cameraID string) (*SecurityEvent, error) {
    // Encode frame as base64
    imageB64 := base64.StdEncoding.EncodeToString(frameData)

    // Build multimodal message
    msgs := []schema.Message{
        &schema.SystemMessage{Parts: []schema.ContentPart{
            schema.TextPart{
                Text: `You are a security camera analyst. Analyze frames for:
- Unauthorized access or intrusion
- Suspicious activity or behavior
- Security threats or dangerous situations
- Unusual objects or vehicles

Respond with:
1. Event detected (yes/no)
2. Threat level (none/low/medium/high/critical)
3. Description of what you observed
4. Recommended action`,
            },
        }},
        &schema.HumanMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: fmt.Sprintf("Analyze this frame from camera %s:", cameraID)},
            schema.ImagePart{URL: fmt.Sprintf("data:image/jpeg;base64,%s", imageB64)},
        }},
    }

    resp, err := a.model.Generate(ctx, msgs)
    if err != nil {
        return nil, fmt.Errorf("frame analysis failed: %w", err)
    }

    // Parse LLM response into SecurityEvent
    return parseSecurityEvent(resp.Parts[0].(schema.TextPart).Text, cameraID), nil
}
```

## Stream Processing with Frame Sampling

Process video streams efficiently with frame sampling:

```go
// AnalyzeStream analyzes a video stream with frame sampling
func (a *SecurityAnalyzer) AnalyzeStream(ctx context.Context, videoStream <-chan []byte, cameraID string, sampleRate time.Duration) (<-chan SecurityEvent, error) {
    eventChan := make(chan SecurityEvent, 100)

    go func() {
        defer close(eventChan)

        ticker := time.NewTicker(sampleRate)
        defer ticker.Stop()

        var currentFrame []byte

        for {
            select {
            case <-ctx.Done():
                return
            case frame, ok := <-videoStream:
                if !ok {
                    return
                }
                currentFrame = frame
            case <-ticker.C:
                if currentFrame == nil {
                    continue
                }

                // Analyze sampled frame
                event, err := a.AnalyzeFrame(ctx, currentFrame, cameraID)
                if err != nil {
                    logger.Error("frame analysis failed", "error", err)
                    continue
                }

                // Only emit events with detections
                if event.Detected {
                    eventChan <- *event
                }
            }
        }
    }()

    return eventChan, nil
}
```

## Threat Classification and Alerting

Classify events and generate alerts for high-priority threats:

```go
type SecurityEvent struct {
    CameraID    string
    Timestamp   time.Time
    Detected    bool
    ThreatLevel string  // "none", "low", "medium", "high", "critical"
    Description string
    Action      string
}

// ProcessEvent processes detected events and sends alerts
func (a *SecurityAnalyzer) ProcessEvent(ctx context.Context, event SecurityEvent) error {
    // Log all events
    logger.Info("security event detected",
        "camera", event.CameraID,
        "threat_level", event.ThreatLevel,
        "description", event.Description,
    )

    // Send alerts for high-priority threats
    if event.ThreatLevel == "high" || event.ThreatLevel == "critical" {
        return a.sendAlert(ctx, event)
    }

    return nil
}

func (a *SecurityAnalyzer) sendAlert(ctx context.Context, event SecurityEvent) error {
    alert := Alert{
        Title:       fmt.Sprintf("Security Alert - Camera %s", event.CameraID),
        Severity:    event.ThreatLevel,
        Description: event.Description,
        Action:      event.Action,
        Timestamp:   event.Timestamp,
    }

    // Send to alerting system (PagerDuty, Slack, email, etc.)
    return alertManager.Send(ctx, alert)
}
```

## Multi-Camera Monitoring

Monitor multiple cameras simultaneously with parallel processing:

```go
// MonitorCameras monitors multiple camera feeds
func (a *SecurityAnalyzer) MonitorCameras(ctx context.Context, cameras []Camera) error {
    var wg sync.WaitGroup

    for _, camera := range cameras {
        wg.Add(1)
        go func(cam Camera) {
            defer wg.Done()

            // Connect to camera stream
            stream, err := cam.OpenStream(ctx)
            if err != nil {
                logger.Error("failed to open camera stream", "camera", cam.ID, "error", err)
                return
            }
            defer stream.Close()

            // Analyze stream (sample every 5 seconds)
            eventChan, err := a.AnalyzeStream(ctx, stream, cam.ID, 5*time.Second)
            if err != nil {
                logger.Error("failed to analyze stream", "camera", cam.ID, "error", err)
                return
            }

            // Process events
            for event := range eventChan {
                if err := a.ProcessEvent(ctx, event); err != nil {
                    logger.Error("failed to process event", "error", err)
                }
            }
        }(camera)
    }

    wg.Wait()
    return nil
}
```

## Production Considerations

### Frame Rate Optimization

Adjust frame sampling based on camera location and activity:

```go
type CameraConfig struct {
    ID         string
    SampleRate time.Duration  // How often to analyze frames
    Priority   int             // 1-5, higher = more frequent analysis
}

func computeSampleRate(priority int) time.Duration {
    switch priority {
    case 5:  // Critical areas (entrances, vaults)
        return 2 * time.Second
    case 4:  // High-traffic areas
        return 5 * time.Second
    case 3:  // Normal areas
        return 10 * time.Second
    case 2:  // Low-traffic areas
        return 30 * time.Second
    default:  // Archive/backup cameras
        return 60 * time.Second
    }
}
```

### Cost Optimization

Reduce LLM API costs with motion detection pre-filtering:

```go
// AnalyzeWithMotionDetection only analyzes frames with motion
func (a *SecurityAnalyzer) AnalyzeWithMotionDetection(ctx context.Context, frameData []byte, previousFrame []byte, cameraID string) (*SecurityEvent, error) {
    // Use lightweight motion detection
    if !detectMotion(frameData, previousFrame) {
        return &SecurityEvent{Detected: false}, nil
    }

    // Only send to LLM if motion detected
    return a.AnalyzeFrame(ctx, frameData, cameraID)
}

func detectMotion(current, previous []byte) bool {
    // Simple pixel difference threshold
    // For production, use OpenCV or similar
    if len(current) != len(previous) {
        return true
    }

    diff := 0
    threshold := 0.05  // 5% pixel change

    for i := range current {
        if current[i] != previous[i] {
            diff++
        }
    }

    return float64(diff)/float64(len(current)) > threshold
}
```

### Event Deduplication

Prevent duplicate alerts for the same event:

```go
type EventDeduplicator struct {
    recent sync.Map  // map[string]time.Time
    window time.Duration
}

func NewEventDeduplicator(window time.Duration) *EventDeduplicator {
    return &EventDeduplicator{window: window}
}

func (d *EventDeduplicator) IsDuplicate(event SecurityEvent) bool {
    key := fmt.Sprintf("%s:%s", event.CameraID, event.Description)

    if lastSeen, ok := d.recent.Load(key); ok {
        if time.Since(lastSeen.(time.Time)) < d.window {
            return true  // Duplicate within window
        }
    }

    d.recent.Store(key, time.Now())
    return false
}
```

### Performance Monitoring

Track analysis latency and throughput:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

type Metrics struct {
    framesAnalyzed   metric.Int64Counter
    eventsDetected   metric.Int64Counter
    analysisLatency  metric.Float64Histogram
}

func (a *SecurityAnalyzer) AnalyzeFrameWithMetrics(ctx context.Context, frameData []byte, cameraID string) (*SecurityEvent, error) {
    start := time.Now()

    event, err := a.AnalyzeFrame(ctx, frameData, cameraID)

    duration := time.Since(start)
    a.metrics.analysisLatency.Record(ctx, duration.Seconds(),
        metric.WithAttributes(attribute.String("camera", cameraID)),
    )
    a.metrics.framesAnalyzed.Add(ctx, 1,
        metric.WithAttributes(attribute.String("camera", cameraID)),
    )

    if err == nil && event.Detected {
        a.metrics.eventsDetected.Add(ctx, 1,
            metric.WithAttributes(
                attribute.String("camera", cameraID),
                attribute.String("threat_level", event.ThreatLevel),
            ),
        )
    }

    return event, err
}
```

## Related Resources

- [Audio-Visual Search](/use-cases/audio-visual-search/) for multimodal query patterns
- [Monitoring Dashboards](/use-cases/monitoring-dashboards/) for observability setup
- [LLM Integration Guide](/guides/llm/) for vision model configuration
