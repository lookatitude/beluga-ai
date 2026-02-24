---
title: "Capability-based Fallbacks"
description: "Recipe for routing multimodal requests to the right Go LLM model based on capability detection with automatic fallback when features are missing."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, capability fallback, multimodal routing, Go model selection, provider fallback, feature detection, LLM capabilities"
---

# Capability-based Fallbacks

## Problem

You need to handle cases where a multimodal model doesn't support certain capabilities (e.g., video processing, specific image formats), requiring fallback to alternative models or simplified processing.

The multimodal LLM landscape is fragmented: GPT-4o handles images but not audio natively, Gemini supports images, audio, and video, Claude supports images but not video. When building applications that accept diverse media types, hardcoding a single provider creates brittleness. If your primary model adds audio support tomorrow, or your fallback model drops video support, your application breaks. Capability-based routing decouples the application from specific provider limitations, automatically adapting to the available model ecosystem.

## Solution

Implement capability detection that checks model capabilities against request requirements, routes requests to the most appropriate model, and provides fallbacks when the primary model cannot handle the input. Models are registered with their capabilities, and the router selects the first model in priority order that satisfies all requirements. If no model fully supports the request, fallback strategies (format conversion, simplified processing) can be applied.

## Code Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("beluga.multimodal.capabilities")

// ModelCapabilities represents model capabilities
type ModelCapabilities struct {
    SupportsImages   bool
    SupportsVideo    bool
    SupportsAudio    bool
    ImageFormats     []string
    VideoFormats     []string
    MaxImageSize     int64
    MaxVideoDuration time.Duration
}

// CapabilityRouter routes requests based on capabilities
type CapabilityRouter struct {
    models        map[string]*ModelCapabilities
    fallbackOrder []string
}

// NewCapabilityRouter creates a new router
func NewCapabilityRouter() *CapabilityRouter {
    return &CapabilityRouter{
        models:        make(map[string]*ModelCapabilities),
        fallbackOrder: []string{},
    }
}

// RegisterModel registers a model with its capabilities
func (cr *CapabilityRouter) RegisterModel(modelID string, capabilities *ModelCapabilities) {
    cr.models[modelID] = capabilities
    cr.fallbackOrder = append(cr.fallbackOrder, modelID)
}

// RouteRequest routes request to appropriate model
func (cr *CapabilityRouter) RouteRequest(ctx context.Context, requestType string, mediaType string) (string, error) {
    ctx, span := tracer.Start(ctx, "capability_router.route")
    defer span.End()

    span.SetAttributes(
        attribute.String("request_type", requestType),
        attribute.String("media_type", mediaType),
    )

    // Find model with required capabilities
    for _, modelID := range cr.fallbackOrder {
        caps := cr.models[modelID]
        if cr.supportsRequest(caps, requestType, mediaType) {
            span.SetAttributes(
                attribute.String("selected_model", modelID),
                attribute.String("selection_reason", "capability_match"),
            )
            span.SetStatus(trace.StatusOK, "model selected")
            return modelID, nil
        }
    }

    // Try fallback: simplified processing
    fallbackModel := cr.findFallbackModel(ctx, requestType, mediaType)
    if fallbackModel != "" {
        span.SetAttributes(
            attribute.String("selected_model", fallbackModel),
            attribute.String("selection_reason", "fallback"),
        )
        span.SetStatus(trace.StatusOK, "fallback model selected")
        return fallbackModel, nil
    }

    err := fmt.Errorf("no model supports %s/%s", requestType, mediaType)
    span.RecordError(err)
    span.SetStatus(trace.StatusError, err.Error())
    return "", err
}

// supportsRequest checks if model supports the request
func (cr *CapabilityRouter) supportsRequest(caps *ModelCapabilities, requestType string, mediaType string) bool {
    switch requestType {
    case "image":
        if !caps.SupportsImages {
            return false
        }
        return cr.supportsFormat(caps.ImageFormats, mediaType)
    case "video":
        return caps.SupportsVideo && cr.supportsFormat(caps.VideoFormats, mediaType)
    case "audio":
        return caps.SupportsAudio
    default:
        return false
    }
}

// supportsFormat checks if format is supported
func (cr *CapabilityRouter) supportsFormat(supportedFormats []string, format string) bool {
    for _, f := range supportedFormats {
        if f == format {
            return true
        }
    }
    return false
}

// findFallbackModel finds a fallback model with partial support
func (cr *CapabilityRouter) findFallbackModel(ctx context.Context, requestType string, mediaType string) string {
    // Try to find model with partial capability
    // e.g., convert video to images and process with image model
    return ""
}

// CapabilityAwareMultimodal wraps multimodal processing with capability routing
type CapabilityAwareMultimodal struct {
    router *CapabilityRouter
    models map[string]interface{} // Map model ID to actual model
}

// NewCapabilityAwareMultimodal creates a new capability-aware processor
func NewCapabilityAwareMultimodal(router *CapabilityRouter) *CapabilityAwareMultimodal {
    return &CapabilityAwareMultimodal{
        router: router,
        models: make(map[string]interface{}),
    }
}

// Process routes and processes request
func (camm *CapabilityAwareMultimodal) Process(ctx context.Context, requestType string, mediaType string, data interface{}) (interface{}, error) {
    ctx, span := tracer.Start(ctx, "capability_aware.process")
    defer span.End()

    // Route to appropriate model
    modelID, err := camm.router.RouteRequest(ctx, requestType, mediaType)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return nil, err
    }

    // Get model
    model := camm.models[modelID]
    if model == nil {
        err := fmt.Errorf("model %s not found", modelID)
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return nil, err
    }

    // Process with selected model
    // result := model.Process(ctx, data)

    span.SetAttributes(attribute.String("model_used", modelID))
    span.SetStatus(trace.StatusOK, "processed with capability-aware routing")

    return nil, nil
}

func main() {
    ctx := context.Background()

    // Create router
    router := NewCapabilityRouter()

    // Register models
    router.RegisterModel("model1", &ModelCapabilities{
        SupportsImages: true,
        ImageFormats:   []string{"image/jpeg", "image/png"},
    })

    // Route request
    modelID, err := router.RouteRequest(ctx, "image", "image/jpeg")
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    fmt.Printf("Selected model: %s\n", modelID)
}
```

## Explanation

1. **Capability registration** -- Each model's capabilities (supported modalities, formats, size limits) are registered with the router. This creates a declarative capability map that can be updated as providers evolve. When a provider adds new capabilities, you update the registration without changing routing logic.

2. **Fallback ordering** -- Models are tried in preference order (set by registration order) until one that supports all request requirements is found. This allows you to prefer faster or cheaper models while falling back to more capable ones when needed. The priority order reflects your cost/quality/latency tradeoffs.

3. **Graceful degradation** -- If no model fully supports the request, the `findFallbackModel` method can implement degradation strategies. For example, convert video to a series of images and process with an image-capable model, or extract audio from video and transcribe separately. This ensures the system provides the best possible result rather than failing outright.

> **Key insight:** Always have fallbacks. Even if primary models don't support a capability, you can often find alternative approaches (simplified processing, format conversion, modality extraction) that provide useful results.

## Testing

```go
func TestCapabilityRouter_RoutesCorrectly(t *testing.T) {
    router := NewCapabilityRouter()
    router.RegisterModel("model1", &ModelCapabilities{SupportsImages: true})

    modelID, err := router.RouteRequest(context.Background(), "image", "image/jpeg")
    require.NoError(t, err)
    require.Equal(t, "model1", modelID)
}
```

## Variations

### Dynamic Capability Detection

Detect capabilities at runtime by probing the model:

```go
func (cr *CapabilityRouter) DetectCapabilities(ctx context.Context, modelID string) (*ModelCapabilities, error) {
    // Query model for capabilities
}
```

### Capability Negotiation

Negotiate the best capability match when no model fully satisfies requirements:

```go
func (cr *CapabilityRouter) NegotiateBestMatch(ctx context.Context, requirements *CapabilityRequirements) (string, error) {
    // Find best matching model
}
```

## Related Recipes

- [Processing Multiple Images per Prompt](/docs/cookbook/multiple-images) -- Process multiple images
- [LLMs Package Guide](/docs/guides/llm-providers) -- For a deeper understanding of multimodal
