---
title: "Capability-based Fallbacks"
package: "multimodal"
category: "resilience"
complexity: "intermediate"
---

# Capability-based Fallbacks

## Problem

You need to handle cases where a multimodal model doesn't support certain capabilities (e.g., video processing, specific image formats), requiring fallback to alternative models or simplified processing.

## Solution

Implement capability detection that checks model capabilities, routes requests to appropriate models based on capabilities, and provides fallbacks when capabilities are unavailable. This works because you can query model capabilities and route requests accordingly.

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
    SupportsImages    bool
    SupportsVideo     bool
    SupportsAudio     bool
    ImageFormats      []string
    VideoFormats      []string
    MaxImageSize      int64
    MaxVideoDuration  time.Duration
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
    router  *CapabilityRouter
    models  map[string]interface{} // Map model ID to actual model
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

Let's break down what's happening:

1. **Capability registration** - Notice how we register each model's capabilities. This allows the router to match requests to appropriate models.

2. **Fallback ordering** - We maintain a fallback order, trying models in preference order until finding one that supports the request.

3. **Graceful degradation** - If no model fully supports the request, we can find fallback models with partial support (e.g., convert video to images).

```go
**Key insight:** Always have fallbacks. Even if primary models don't support a capability, you can often find alternative approaches (simplified processing, format conversion, etc.).

## Testing

```
Here's how to test this solution:
```go
func TestCapabilityRouter_RoutesCorrectly(t *testing.T) {
    router := NewCapabilityRouter()
    router.RegisterModel("model1", &ModelCapabilities{SupportsImages: true})
    
    modelID, err := router.RouteRequest(context.Background(), "image", "image/jpeg")
    require.NoError(t, err)
    require.Equal(t, "model1", modelID)
}

## Variations

### Dynamic Capability Detection

Detect capabilities at runtime:
func (cr *CapabilityRouter) DetectCapabilities(ctx context.Context, modelID string) (*ModelCapabilities, error) {
    // Query model for capabilities
}
```

### Capability Negotiation

Negotiate best capability match:
```go
func (cr *CapabilityRouter) NegotiateBestMatch(ctx context.Context, requirements *CapabilityRequirements) (string, error) {
    // Find best matching model
}
```

## Related Recipes

- **[Multimodal Processing Multiple Images](./multimodal-processing-multiple-images-per-prompt.md)** - Process multiple images
- **[LLMs Package Guide](../guides/llm-providers.md)** - For a deeper understanding of multimodal
