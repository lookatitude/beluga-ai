# SafetyResult JSON Reporting

Welcome, colleague! In this integration guide, we're going to implement JSON reporting for Beluga AI safety results. This enables exporting safety check results for logging, auditing, and compliance purposes.

## What you will build

You will create a system that exports Beluga AI safety check results as JSON, enabling integration with logging systems, audit trails, and compliance reporting tools.

## Learning Objectives

- ✅ Export safety results as JSON
- ✅ Create structured safety reports
- ✅ Integrate with logging systems
- ✅ Understand compliance reporting patterns

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Understanding of JSON serialization

## Step 1: Basic JSON Export

Create JSON export functionality:
```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/safety"
)

type SafetyReport struct {
    SafetyResult safety.SafetyResult `json:"safety_result"`
    Metadata     ReportMetadata      `json:"metadata"`
}

type ReportMetadata struct {
    ReportID    string    `json:"report_id"`
    GeneratedAt time.Time `json:"generated_at"`
    Version     string    `json:"version"`
    Source      string    `json:"source"`
}

func ExportSafetyResult(result safety.SafetyResult) ([]byte, error) {
    report := SafetyReport{
        SafetyResult: result,
        Metadata: ReportMetadata{
            ReportID:    generateReportID(),
            GeneratedAt: time.Now(),
            Version:     "1.0",
            Source:      "beluga-ai",
        },
    }
    
    return json.MarshalIndent(report, "", "  ")
}
```

## Step 2: Enhanced Reporting

Add detailed reporting:
```go
type DetailedSafetyReport struct {
    SafetyResult safety.SafetyResult `json:"safety_result"`
    CheckDetails CheckDetails         `json:"check_details"`
    Metadata     ReportMetadata       `json:"metadata"`
}

type CheckDetails struct {
    CheckType    string    `json:"check_type"`
    Duration     time.Duration `json:"duration_ms"`
    PatternsChecked int     `json:"patterns_checked"`
    APIUsed      bool      `json:"api_used"`
}

func ExportDetailedResult(result safety.SafetyResult, details CheckDetails) ([]byte, error) {
    report := DetailedSafetyReport{
        SafetyResult: result,
        CheckDetails: details,
        Metadata: ReportMetadata{
            ReportID:    generateReportID(),
            GeneratedAt: time.Now(),
            Version:     "1.0",
            Source:      "beluga-ai",
        },
    }

    
    return json.MarshalIndent(report, "", "  ")
}
```

## Step 3: File Export

Export to file:
```go
func ExportToFile(result safety.SafetyResult, filepath string) error {
    jsonData, err := ExportSafetyResult(result)
    if err != nil {
        return fmt.Errorf("export failed: %w", err)
    }

    
    return os.WriteFile(filepath, jsonData, 0644)
}
```

## Step 4: Streaming Export

Stream results to external systems:
```go
type SafetyReporter struct {
    output io.Writer
    encoder *json.Encoder
}

func NewSafetyReporter(output io.Writer) *SafetyReporter {
    return &SafetyReporter{
        output:  output,
        encoder: json.NewEncoder(output),
    }
}

func (r *SafetyReporter) Report(ctx context.Context, result safety.SafetyResult) error {
    report := SafetyReport{
        SafetyResult: result,
        Metadata: ReportMetadata{
            ReportID:    generateReportID(),
            GeneratedAt: time.Now(),
            Version:     "1.0",
            Source:      "beluga-ai",
        },
    }

    
    return r.encoder.Encode(report)
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
    "os"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/safety"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type ProductionSafetyReporter struct {
    output   io.Writer
    encoder  *json.Encoder
    tracer   trace.Tracer
    reportCount int64
}

func NewProductionSafetyReporter(output io.Writer) *ProductionSafetyReporter {
    return &ProductionSafetyReporter{
        output:  output,
        encoder: json.NewEncoder(output),
        tracer:  otel.Tracer("beluga.safety.reporter"),
    }
}

func (r *ProductionSafetyReporter) Report(ctx context.Context, result safety.SafetyResult) error {
    ctx, span := r.tracer.Start(ctx, "safety.report",
        trace.WithAttributes(
            attribute.Bool("safe", result.Safe),
            attribute.Float64("risk_score", result.RiskScore),
        ),
    )
    defer span.End()
    
    report := SafetyReport{
        SafetyResult: result,
        Metadata: ReportMetadata{
            ReportID:    fmt.Sprintf("report-%d-%d", time.Now().Unix(), r.reportCount),
            GeneratedAt: time.Now(),
            Version:     "1.0",
            Source:      "beluga-ai",
        },
    }
    
    r.reportCount++
    
    if err := r.encoder.Encode(report); err != nil {
        span.RecordError(err)
        return fmt.Errorf("encode failed: %w", err)
    }
    
    span.SetAttributes(attribute.Int64("report_count", r.reportCount))
    return nil
}

func main() {
    ctx := context.Background()
    
    // Create reporter
    reporter := NewProductionSafetyReporter(os.Stdout)
    
    // Create safety checker
    checker := safety.NewSafetyChecker()
    
    // Check content
    result, err := checker.CheckContent(ctx, "test content")
    if err != nil {
        log.Fatalf("Check failed: %v", err)
    }
    
    // Report result
    if err := reporter.Report(ctx, result); err != nil {
        log.Fatalf("Report failed: %v", err)
    }
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `Output` | Output destination | `stdout` | No |
| `Format` | JSON format (pretty/compact) | `pretty` | No |
| `IncludeMetadata` | Include metadata | `true` | No |

## Common Issues

### "JSON encoding failed"

**Problem**: Invalid data structure.

**Solution**: Ensure all fields are JSON-serializable:// Use json tags on struct fields
```

### "File write failed"

**Problem**: Permission or path issues.

**Solution**: Check file permissions and path:chmod 644 reports/
```

## Production Considerations

When using JSON reporting in production:

- **Structured logging**: Integrate with logging systems
- **Audit trails**: Store reports for compliance
- **Performance**: Use streaming for high volume
- **Security**: Sanitize sensitive data before export
- **Retention**: Implement retention policies

## Next Steps

Congratulations! You've implemented JSON reporting for safety results. Next, learn how to:

- **[Third-Party Ethical API Filter](./third-party-ethical-api-filter.md)** - External safety APIs
- **Safety Package Documentation** - Deep dive into safety package
- **[Safety Guide](../../guides/llm-providers.md)** - Safety patterns

---

**Ready for more?** Check out the Integrations Index for more integration guides!
