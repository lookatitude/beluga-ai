---
title: Third-Party Ethical API Filter
description: Integrate external content safety APIs (Perspective API, Azure Content Safety, AWS Comprehend) with the Beluga AI guard pipeline.
---

Built-in content filters cover common safety patterns, but enterprise deployments often require specialized safety services that are maintained by dedicated teams, updated with the latest threat models, and backed by SLAs. Google Perspective API excels at toxicity detection, Azure Content Safety provides category-level scoring, and AWS Comprehend detects PII and sentiment. By wrapping these services as Beluga AI guards, you compose them with built-in guards in the three-stage safety pipeline (input, output, tool) and get defense in depth without replacing your existing safety infrastructure.

## Overview

While Beluga AI includes built-in guards for prompt injection detection, PII redaction, and content filtering, enterprise deployments often require integration with specialized external safety services. By implementing the `guard.Guard` interface, any external API can participate in the three-stage guard pipeline (input, output, tool) alongside built-in guards.

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed (`github.com/lookatitude/beluga-ai`)
- API credentials for your chosen safety service
- Familiarity with the `guard` package and the `Guard` interface

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```

No additional Go dependencies are required. The external API is called via the standard `net/http` client.

## Configuration

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `APIURL` | Safety API endpoint URL | -- | Yes |
| `APIKey` | API authentication key | -- | Yes |
| `Timeout` | HTTP request timeout | `10s` | No |
| `RetryCount` | Maximum retry attempts on failure | `3` | No |

## Usage

### Implementing the Guard Interface

Create a guard that calls an external safety API and translates the response into a `GuardResult`.

```go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lookatitude/beluga-ai/guard"
)

// ThirdPartySafetyGuard wraps an external content safety API as a Beluga AI guard.
type ThirdPartySafetyGuard struct {
	apiURL     string
	apiKey     string
	httpClient *http.Client
}

// NewThirdPartySafetyGuard creates a guard that delegates validation to an external API.
func NewThirdPartySafetyGuard(apiURL, apiKey string) *ThirdPartySafetyGuard {
	return &ThirdPartySafetyGuard{
		apiURL: apiURL,
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the guard identifier.
func (g *ThirdPartySafetyGuard) Name() string {
	return "third-party-safety"
}

// safetyAPIRequest is the request payload for the external API.
type safetyAPIRequest struct {
	Text string `json:"text"`
}

// safetyAPIResponse is the response payload from the external API.
type safetyAPIResponse struct {
	Safe       bool               `json:"safe"`
	RiskScore  float64            `json:"risk_score"`
	Categories map[string]float64 `json:"categories"`
}

// Validate calls the external API and converts the response to a GuardResult.
func (g *ThirdPartySafetyGuard) Validate(ctx context.Context, input guard.GuardInput) (guard.GuardResult, error) {
	reqBody := safetyAPIRequest{Text: input.Content}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return guard.GuardResult{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.apiURL, bytes.NewReader(jsonData))
	if err != nil {
		return guard.GuardResult{}, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return guard.GuardResult{}, fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return guard.GuardResult{}, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return guard.GuardResult{}, fmt.Errorf("read response: %w", err)
	}

	var apiResp safetyAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return guard.GuardResult{}, fmt.Errorf("unmarshal response: %w", err)
	}

	// Convert API response to GuardResult
	result := guard.GuardResult{
		Allowed:   apiResp.Safe,
		GuardName: g.Name(),
	}

	if !apiResp.Safe {
		result.Reason = g.buildReason(apiResp.Categories)
	}

	return result, nil
}

// buildReason summarizes flagged categories into a human-readable reason.
func (g *ThirdPartySafetyGuard) buildReason(categories map[string]float64) string {
	reason := "content flagged by external API:"
	for category, score := range categories {
		if score > 0.5 {
			reason += fmt.Sprintf(" %s(%.2f)", category, score)
		}
	}
	return reason
}
```

### Integrating with the Guard Pipeline

Register the external guard alongside built-in guards in the pipeline.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/guard"
)

func main() {
	ctx := context.Background()

	// Create the external safety guard
	externalGuard := NewThirdPartySafetyGuard(
		os.Getenv("SAFETY_API_URL"),
		os.Getenv("SAFETY_API_KEY"),
	)

	// Compose with built-in guards in the pipeline
	pipeline := guard.NewPipeline(
		guard.Input(
			guard.NewPromptInjectionDetector(),  // Fast, local check first
			externalGuard,                        // External API check second
		),
		guard.Output(
			guard.NewPIIRedactor(guard.DefaultPIIPatterns...),
		),
	)

	// Validate user input
	result, err := pipeline.ValidateInput(ctx, "Hello, can you help me?")
	if err != nil {
		log.Fatalf("validation error: %v", err)
	}

	if result.Allowed {
		fmt.Println("Content is safe")
	} else {
		fmt.Printf("Content blocked: %s\n", result.Reason)
	}
}
```

### Hybrid Safety with Fallback

Combine built-in and external guards with graceful degradation when the external API is unavailable.

```go
package main

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/guard"
)

// FallbackSafetyGuard tries the external API first and falls back to a local guard on failure.
type FallbackSafetyGuard struct {
	primary  guard.Guard
	fallback guard.Guard
}

// NewFallbackSafetyGuard wraps a primary guard with a local fallback.
func NewFallbackSafetyGuard(primary, fallback guard.Guard) *FallbackSafetyGuard {
	return &FallbackSafetyGuard{
		primary:  primary,
		fallback: fallback,
	}
}

// Name returns the guard identifier.
func (g *FallbackSafetyGuard) Name() string {
	return "fallback-safety"
}

// Validate attempts the primary guard and falls back on error.
func (g *FallbackSafetyGuard) Validate(ctx context.Context, input guard.GuardInput) (guard.GuardResult, error) {
	result, err := g.primary.Validate(ctx, input)
	if err != nil {
		// Log the external API failure and fall back
		fmt.Printf("primary guard failed, using fallback: %v\n", err)
		return g.fallback.Validate(ctx, input)
	}
	return result, nil
}
```

## Advanced Topics

### OpenTelemetry Instrumentation

Add tracing to external API calls for latency monitoring and error tracking.

```go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lookatitude/beluga-ai/guard"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedSafetyGuard adds OTel tracing to external API calls.
type InstrumentedSafetyGuard struct {
	apiURL     string
	apiKey     string
	httpClient *http.Client
	tracer     trace.Tracer
}

// NewInstrumentedSafetyGuard creates a traced external safety guard.
func NewInstrumentedSafetyGuard(apiURL, apiKey string) *InstrumentedSafetyGuard {
	return &InstrumentedSafetyGuard{
		apiURL: apiURL,
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		tracer: otel.Tracer("beluga.guard.third_party"),
	}
}

// Name returns the guard identifier.
func (g *InstrumentedSafetyGuard) Name() string {
	return "instrumented-third-party-safety"
}

// Validate calls the external API with OTel tracing.
func (g *InstrumentedSafetyGuard) Validate(ctx context.Context, input guard.GuardInput) (guard.GuardResult, error) {
	ctx, span := g.tracer.Start(ctx, "guard.third_party.validate",
		trace.WithAttributes(
			attribute.String("guard.api_url", g.apiURL),
			attribute.String("guard.stage", input.Role),
			attribute.Int("guard.content_length", len(input.Content)),
		),
	)
	defer span.End()

	reqBody := safetyAPIRequest{Text: input.Content}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		span.RecordError(err)
		return guard.GuardResult{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.apiURL, bytes.NewReader(jsonData))
	if err != nil {
		span.RecordError(err)
		return guard.GuardResult{}, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		return guard.GuardResult{}, fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("API returned status %d", resp.StatusCode)
		span.RecordError(err)
		return guard.GuardResult{}, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.RecordError(err)
		return guard.GuardResult{}, fmt.Errorf("read response: %w", err)
	}

	var apiResp safetyAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		span.RecordError(err)
		return guard.GuardResult{}, fmt.Errorf("unmarshal response: %w", err)
	}

	result := guard.GuardResult{
		Allowed:   apiResp.Safe,
		GuardName: g.Name(),
	}

	span.SetAttributes(
		attribute.Bool("guard.allowed", result.Allowed),
		attribute.Float64("guard.risk_score", apiResp.RiskScore),
	)

	return result, nil
}
```

### Rate Limiting and Caching

For production deployments, wrap external API calls with the `resilience` and `cache` packages to manage throughput and reduce costs.

```go
// Use resilience.RateLimiter to throttle external API calls
// Use cache.ExactCache to avoid re-checking identical content

// Example: cache guard results by content hash
// See the resilience and cache package documentation for details.
```

## Troubleshooting

### API key invalid

Verify your API key is correctly set in the environment. Rotate keys if they have expired.

```bash
export SAFETY_API_KEY="your-api-key"
```

### Rate limit exceeded

Implement request throttling using the `resilience` package or add a caching layer to avoid repeated checks on identical content.

### Connection timeouts

Increase the HTTP client timeout or use the `FallbackSafetyGuard` pattern to degrade gracefully when the external service is slow.

## Production Considerations

- **Guard ordering**: Place fast local guards (prompt injection, content filter) before external API guards to minimize unnecessary API calls.
- **Rate limits**: Monitor external API quotas and implement rate limiting to stay within limits.
- **Cost management**: Cache results for repeated content to reduce API costs.
- **Fallback strategy**: Always provide a local fallback guard for when external APIs are unavailable.
- **Error handling**: External API failures should not crash the pipeline. Log errors and degrade gracefully.
- **Latency budgets**: External API calls add latency. Set appropriate timeouts and monitor p99 latency.

## Related Resources

- [Guard Package](/guides/safety-guards/) -- Guard pipeline documentation
- [Safety Result JSON Reporting](/integrations/safety-json-reporting/) -- Export guard results as JSON
- [Resilience Package](/guides/resilience/) -- Circuit breaker, retry, and rate limiting
- [Monitoring and Observability](/integrations/monitoring/) -- OTel instrumentation
