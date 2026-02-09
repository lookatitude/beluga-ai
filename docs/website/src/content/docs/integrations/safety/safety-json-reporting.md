---
title: Safety Result JSON Reporting
description: Export Beluga AI guard pipeline results as structured JSON for logging, auditing, and compliance reporting.
---

Beluga AI's `guard` package produces `GuardResult` values from its three-stage safety pipeline. This guide demonstrates how to serialize those results as JSON for integration with logging systems, audit trails, and compliance dashboards.

## Overview

The guard pipeline validates content at three stages -- input, output, and tool -- and returns `GuardResult` structs indicating whether content was allowed, blocked, or modified. By wrapping these results in a structured report and serializing to JSON, you can feed safety data into any downstream system that consumes structured logs.

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed (`github.com/lookatitude/beluga-ai`)
- Familiarity with the `guard` package and its `GuardResult` type

## Installation

No additional dependencies are required beyond the standard library and the Beluga AI framework.

```bash
go get github.com/lookatitude/beluga-ai
```

## Configuration

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `Output` | Output destination (`io.Writer`) | `os.Stdout` | No |
| `Format` | JSON format: `"pretty"` or `"compact"` | `"pretty"` | No |
| `IncludeMetadata` | Include report metadata | `true` | No |

## Usage

### Basic JSON Export

Define a report structure that wraps `GuardResult` with metadata, then serialize it to JSON.

```go
package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/guard"
)

// SafetyReport wraps a GuardResult with reporting metadata.
type SafetyReport struct {
	Result   guard.GuardResult `json:"result"`
	Metadata ReportMetadata    `json:"metadata"`
}

// ReportMetadata provides context about when and where the report was generated.
type ReportMetadata struct {
	ReportID    string    `json:"report_id"`
	GeneratedAt time.Time `json:"generated_at"`
	Version     string    `json:"version"`
	Source      string    `json:"source"`
}

// ExportGuardResult serializes a GuardResult as a JSON report.
func ExportGuardResult(result guard.GuardResult, reportID string) ([]byte, error) {
	report := SafetyReport{
		Result: result,
		Metadata: ReportMetadata{
			ReportID:    reportID,
			GeneratedAt: time.Now(),
			Version:     "1.0",
			Source:      "beluga-ai",
		},
	}

	return json.MarshalIndent(report, "", "  ")
}

func main() {
	result := guard.GuardResult{
		Allowed:   true,
		GuardName: "pii-redactor",
	}

	data, err := ExportGuardResult(result, "rpt-001")
	if err != nil {
		fmt.Printf("export failed: %v\n", err)
		return
	}

	fmt.Println(string(data))
}
```

### Enhanced Reporting with Check Details

Add pipeline execution details to the report for deeper audit visibility.

```go
package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/guard"
)

// DetailedSafetyReport extends the basic report with pipeline execution details.
type DetailedSafetyReport struct {
	Result       guard.GuardResult `json:"result"`
	CheckDetails CheckDetails      `json:"check_details"`
	Metadata     ReportMetadata    `json:"metadata"`
}

// CheckDetails captures execution context for a guard validation.
type CheckDetails struct {
	Stage           string        `json:"stage"`
	Duration        time.Duration `json:"duration_ns"`
	GuardsEvaluated int           `json:"guards_evaluated"`
}

// ExportDetailedResult serializes a GuardResult with execution details.
func ExportDetailedResult(result guard.GuardResult, details CheckDetails, reportID string) ([]byte, error) {
	report := DetailedSafetyReport{
		Result:       result,
		CheckDetails: details,
		Metadata: ReportMetadata{
			ReportID:    reportID,
			GeneratedAt: time.Now(),
			Version:     "1.0",
			Source:      "beluga-ai",
		},
	}

	return json.MarshalIndent(report, "", "  ")
}

func main() {
	result := guard.GuardResult{
		Allowed:   false,
		Reason:    "PII detected: credit card number",
		Modified:  "I need help with my card ending in ****",
		GuardName: "pii-redactor",
	}

	details := CheckDetails{
		Stage:           "input",
		Duration:        2 * time.Millisecond,
		GuardsEvaluated: 3,
	}

	data, err := ExportDetailedResult(result, details, "rpt-002")
	if err != nil {
		fmt.Printf("export failed: %v\n", err)
		return
	}

	fmt.Println(string(data))
}
```

### Streaming Reports to External Systems

For high-throughput pipelines, stream reports to an `io.Writer` using `json.Encoder` instead of buffering entire reports in memory.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	"github.com/lookatitude/beluga-ai/guard"
)

// SafetyReporter streams JSON-encoded safety reports to a writer.
type SafetyReporter struct {
	encoder     *json.Encoder
	reportCount atomic.Int64
}

// NewSafetyReporter creates a reporter that writes to the given destination.
func NewSafetyReporter(output io.Writer) *SafetyReporter {
	return &SafetyReporter{
		encoder: json.NewEncoder(output),
	}
}

// Report serializes and writes a GuardResult to the output stream.
func (r *SafetyReporter) Report(ctx context.Context, result guard.GuardResult) error {
	count := r.reportCount.Add(1)

	report := SafetyReport{
		Result: result,
		Metadata: ReportMetadata{
			ReportID:    fmt.Sprintf("rpt-%d-%d", time.Now().Unix(), count),
			GeneratedAt: time.Now(),
			Version:     "1.0",
			Source:      "beluga-ai",
		},
	}

	if err := r.encoder.Encode(report); err != nil {
		return fmt.Errorf("encode report: %w", err)
	}

	return nil
}

func main() {
	ctx := context.Background()
	reporter := NewSafetyReporter(os.Stdout)

	// Run guard pipeline
	pipeline := guard.NewPipeline(
		guard.Input(guard.NewPIIRedactor(guard.DefaultPIIPatterns...)),
	)

	result, err := pipeline.ValidateInput(ctx, "My card is 4111-1111-1111-1111")
	if err != nil {
		fmt.Printf("validation failed: %v\n", err)
		return
	}

	if err := reporter.Report(ctx, result); err != nil {
		fmt.Printf("report failed: %v\n", err)
	}
}
```

## Advanced Topics

### OpenTelemetry Integration

Combine JSON reporting with OpenTelemetry spans for correlated safety observability.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/lookatitude/beluga-ai/guard"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedReporter adds OTel tracing to safety reporting.
type InstrumentedReporter struct {
	encoder     *json.Encoder
	tracer      trace.Tracer
	reportCount atomic.Int64
}

// NewInstrumentedReporter creates a reporter with OTel instrumentation.
func NewInstrumentedReporter(output io.Writer) *InstrumentedReporter {
	return &InstrumentedReporter{
		encoder: json.NewEncoder(output),
		tracer:  otel.Tracer("beluga.guard.reporter"),
	}
}

// Report writes a traced safety report.
func (r *InstrumentedReporter) Report(ctx context.Context, result guard.GuardResult) error {
	ctx, span := r.tracer.Start(ctx, "guard.report",
		trace.WithAttributes(
			attribute.Bool("guard.allowed", result.Allowed),
			attribute.String("guard.name", result.GuardName),
		),
	)
	defer span.End()

	count := r.reportCount.Add(1)

	report := SafetyReport{
		Result: result,
		Metadata: ReportMetadata{
			ReportID:    fmt.Sprintf("rpt-%d-%d", time.Now().Unix(), count),
			GeneratedAt: time.Now(),
			Version:     "1.0",
			Source:      "beluga-ai",
		},
	}

	if err := r.encoder.Encode(report); err != nil {
		span.RecordError(err)
		return fmt.Errorf("encode report: %w", err)
	}

	span.SetAttributes(attribute.Int64("guard.report_count", count))
	return nil
}
```

### File Export

Write safety reports to disk for offline analysis or regulatory archival.

```go
package main

import (
	"fmt"
	"os"

	"github.com/lookatitude/beluga-ai/guard"
)

// ExportToFile writes a single GuardResult report to the specified path.
func ExportToFile(result guard.GuardResult, reportID, path string) error {
	data, err := ExportGuardResult(result, reportID)
	if err != nil {
		return fmt.Errorf("export guard result: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}
```

### Batch Reporting

Aggregate multiple guard results into a single report for batch processing scenarios.

```go
// BatchReport collects multiple guard results into a single document.
type BatchReport struct {
	Results  []guard.GuardResult `json:"results"`
	Summary  BatchSummary        `json:"summary"`
	Metadata ReportMetadata      `json:"metadata"`
}

// BatchSummary provides aggregate statistics across all results.
type BatchSummary struct {
	Total   int `json:"total"`
	Allowed int `json:"allowed"`
	Blocked int `json:"blocked"`
}
```

## Troubleshooting

### JSON encoding fails

Ensure all fields in your report struct are JSON-serializable. Use `json` struct tags on every field and verify that custom types implement `json.Marshaler` if needed.

### File write permission denied

Verify the target directory exists and the process has write permissions. Use `0600` file mode for reports containing sensitive safety data.

## Production Considerations

- **Structured logging**: Pipe JSON reports into your centralized logging system (ELK, Datadog, Loki) for searchable safety audit trails.
- **Audit compliance**: Store reports with immutable storage for regulatory requirements. Include report IDs for traceability.
- **Performance**: Use the streaming reporter (`json.Encoder`) for high-throughput pipelines to avoid buffering entire reports in memory.
- **Data sensitivity**: Sanitize or redact content fields before writing reports to disk if they may contain PII or other sensitive data.
- **Retention policies**: Implement time-based or size-based retention for safety report archives.

## Related Resources

- [Guard Package](/guides/safety-guards/) -- Guard pipeline documentation
- [Ethical API Filter](/integrations/ethical-api-filter/) -- Third-party safety API integration
- [Monitoring and Observability](/integrations/monitoring/) -- OTel instrumentation
