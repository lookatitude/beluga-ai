# Third-Party Ethical API Filter

Welcome, colleague! In this integration guide, we're going to integrate third-party ethical API filters with Beluga AI's safety package. This enables advanced content safety checking using specialized safety APIs.

## What you will build

You will configure Beluga AI to use third-party ethical API filters (like Perspective API, Azure Content Moderator, etc.) for content safety validation, providing enterprise-grade safety checking beyond basic pattern matching.

## Learning Objectives

- ✅ Integrate third-party safety APIs
- ✅ Wrap external APIs with Beluga AI safety interface
- ✅ Handle API responses and errors
- ✅ Combine with built-in safety checks

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Third-party safety API account (e.g., Perspective API, Azure Content Moderator)
- API key for the service

## Step 1: Setup and Installation

Choose a safety API provider:

- **Perspective API** (Google): https://www.perspectiveapi.com
- **Azure Content Moderator**: https://azure.microsoft.com/services/cognitive-services/content-moderator/
- **AWS Comprehend**: https://aws.amazon.com/comprehend/

Install HTTP client:
bash
```bash
go get net/http
```

## Step 2: Create API Filter Wrapper

Create a wrapper for third-party safety APIs:
```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/safety"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type ThirdPartySafetyFilter struct {
    apiURL    string
    apiKey    string
    httpClient *http.Client
    tracer    trace.Tracer
}

type SafetyAPIRequest struct {
    Text string `json:"text"`
}

type SafetyAPIResponse struct {
    Safe      bool    `json:"safe"`
    RiskScore float64 `json:"risk_score"`
    Categories map[string]float64 `json:"categories"`
}

func NewThirdPartySafetyFilter(apiURL, apiKey string) *ThirdPartySafetyFilter {
    return &ThirdPartySafetyFilter{
        apiURL: apiURL,
        apiKey: apiKey,
        httpClient: &http.Client{
            Timeout: 10 * time.Second,
        },
        tracer: otel.Tracer("beluga.safety.third_party"),
    }
}

func (f *ThirdPartySafetyFilter) CheckContent(ctx context.Context, content string) (safety.SafetyResult, error) {
    ctx, span := f.tracer.Start(ctx, "third_party.check",
        trace.WithAttributes(
            attribute.String("api_url", f.apiURL),
            attribute.Int("content_length", len(content)),
        ),
    )
    defer span.End()
    
    // Prepare request
    reqBody := SafetyAPIRequest{Text: content}
    jsonData, err := json.Marshal(reqBody)
    if err != nil {
        span.RecordError(err)
        return safety.SafetyResult{}, fmt.Errorf("marshal failed: %w", err)
    }
    
    // Create HTTP request
    req, err := http.NewRequestWithContext(ctx, "POST", f.apiURL, 
        strings.NewReader(string(jsonData)))
    if err != nil {
        span.RecordError(err)
        return safety.SafetyResult{}, fmt.Errorf("request failed: %w", err)
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+f.apiKey)
    
    // Execute request
    resp, err := f.httpClient.Do(req)
    if err != nil {
        span.RecordError(err)
        return safety.SafetyResult{}, fmt.Errorf("API call failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        span.RecordError(fmt.Errorf("API error: %d", resp.StatusCode))
        return safety.SafetyResult{}, fmt.Errorf("API error: %d", resp.StatusCode)
    }
    
    // Parse response
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        span.RecordError(err)
        return safety.SafetyResult{}, fmt.Errorf("read failed: %w", err)
    }
    
    var apiResp SafetyAPIResponse
    if err := json.Unmarshal(body, &apiResp); err != nil {
        span.RecordError(err)
        return safety.SafetyResult{}, fmt.Errorf("unmarshal failed: %w", err)
    }
    
    // Convert to Beluga AI SafetyResult
    result := safety.SafetyResult{
        Safe:      apiResp.Safe,
        RiskScore: apiResp.RiskScore,
        Timestamp: time.Now(),
        Issues:    make([]safety.SafetyIssue, 0),
    }
    
    // Convert categories to issues
    for category, score := range apiResp.Categories {
        if score > 0.5 {
            result.Issues = append(result.Issues, safety.SafetyIssue{
                Type:        category,
                Description: fmt.Sprintf("%s detected (score: %.2f)", category, score),
                Severity:    f.getSeverity(score),
            })
        }
    }
    
    span.SetAttributes(
        attribute.Bool("safe", result.Safe),
        attribute.Float64("risk_score", result.RiskScore),
    )
    
    return result, nil
}

func (f *ThirdPartySafetyFilter) getSeverity(score float64) string {
    if score > 0.8 {
        return "high"
    } else if score > 0.5 {
        return "medium"
    }
    return "low"
}
```

## Step 3: Integrate with Beluga AI Safety

Use with Beluga AI safety checker:
```go
func main() {
    ctx := context.Background()
    
    // Create third-party filter
    filter := NewThirdPartySafetyFilter(
        "https://api.safety-service.com/v1/check",
        os.Getenv("SAFETY_API_KEY"),
    )
    
    // Check content
    result, err := filter.CheckContent(ctx, "User input text here")
    if err != nil {
        log.Fatalf("Check failed: %v", err)
    }
    
    if !result.Safe {
        fmt.Printf("Content unsafe! Risk score: %.2f\n", result.RiskScore)
        for _, issue := range result.Issues {
            fmt.Printf("  - %s: %s\n", issue.Type, issue.Description)
        }
    } else {
        fmt.Println("Content is safe")
    }
}
```

## Step 4: Combine with Built-in Safety

Combine third-party API with built-in checks:
```go
type HybridSafetyChecker struct \{
    builtIn  *safety.SafetyChecker
    external *ThirdPartySafetyFilter
}
go
func (h *HybridSafetyChecker) CheckContent(ctx context.Context, content string) (safety.SafetyResult, error) {
    // Run built-in check first (fast)
    builtInResult, _ := h.builtIn.CheckContent(ctx, content)
    
    // If built-in passes, check with external API
    if builtInResult.Safe {
        externalResult, err := h.external.CheckContent(ctx, content)
        if err != nil {
            // Fallback to built-in result on API failure
            return builtInResult, nil
        }
        return externalResult, nil
    }

    
    return builtInResult, nil
}
```

## Step 5: Complete Integration

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/safety"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type ProductionThirdPartyFilter struct {
    apiURL     string
    apiKey     string
    httpClient *http.Client
    tracer     trace.Tracer
}

func NewProductionThirdPartyFilter(apiURL, apiKey string) *ProductionThirdPartyFilter {
    return &ProductionThirdPartyFilter{
        apiURL: apiURL,
        apiKey: apiKey,
        httpClient: &http.Client{
            Timeout: 10 * time.Second,
        },
        tracer: otel.Tracer("beluga.safety.third_party"),
    }
}

func (f *ProductionThirdPartyFilter) CheckContent(ctx context.Context, content string) (safety.SafetyResult, error) {
    ctx, span := f.tracer.Start(ctx, "third_party.check")
    defer span.End()
    
    reqBody := map[string]string{"text": content}
    jsonData, _ := json.Marshal(reqBody)
    
    req, _ := http.NewRequestWithContext(ctx, "POST", f.apiURL,
        strings.NewReader(string(jsonData)))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+f.apiKey)
    
    resp, err := f.httpClient.Do(req)
    if err != nil {
        span.RecordError(err)
        return safety.SafetyResult{}, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        err := fmt.Errorf("API error: %d", resp.StatusCode)
        span.RecordError(err)
        return safety.SafetyResult{}, err
    }
    
    body, _ := io.ReadAll(resp.Body)
    var apiResp struct {
        Safe      bool              `json:"safe"`
        RiskScore float64           `json:"risk_score"`
        Categories map[string]float64 `json:"categories"`
    }
    json.Unmarshal(body, &apiResp)
    
    result := safety.SafetyResult{
        Safe:      apiResp.Safe,
        RiskScore: apiResp.RiskScore,
        Timestamp: time.Now(),
    }
    
    span.SetAttributes(
        attribute.Bool("safe", result.Safe),
        attribute.Float64("risk_score", result.RiskScore),
    )
    
    return result, nil
}

func main() {
    ctx := context.Background()
    
    filter := NewProductionThirdPartyFilter(
        os.Getenv("SAFETY_API_URL"),
        os.Getenv("SAFETY_API_KEY"),
    )
    
    result, err := filter.CheckContent(ctx, "test content")
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    
    fmt.Printf("Safe: %v, Risk: %.2f\n", result.Safe, result.RiskScore)
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `APIURL` | Safety API endpoint | - | Yes |
| `APIKey` | API authentication key | - | Yes |
| `Timeout` | Request timeout | `10s` | No |
| `RetryCount` | Maximum retries | `3` | No |

## Common Issues

### "API key invalid"

**Problem**: Wrong or expired API key.

**Solution**: Verify API key:export SAFETY_API_KEY="your-api-key"
```

### "Rate limit exceeded"

**Problem**: Too many API calls.

**Solution**: Implement rate limiting or caching:// Cache results for repeated content
```

## Production Considerations

When using third-party APIs in production:

- **Rate limits**: Monitor and handle rate limits
- **Cost management**: Track API usage and costs
- **Fallbacks**: Have fallback to built-in checks
- **Caching**: Cache results for repeated content
- **Error handling**: Handle API failures gracefully

## Next Steps

Congratulations! You've integrated third-party safety APIs with Beluga AI. Next, learn how to:

- **[SafetyResult JSON Reporting](./safety-result-json-reporting.md)** - Export safety results
- **Safety Package Documentation** - Deep dive into safety package
- **[Safety Guide](../../guides/llm-providers.md)** - Safety patterns

---

**Ready for more?** Check out the Integrations Index for more integration guides!
