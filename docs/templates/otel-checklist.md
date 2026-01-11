# OTEL Integration Checklist

**Package**: [Package Name]  
**Date**: [Date]  
**Status**: [In Progress / Complete]

## Overview

This checklist ensures complete OTEL (OpenTelemetry) observability integration for a Beluga AI framework package, following v2 standards.

## Metrics Integration

- [ ] `metrics.go` file exists with OTEL metrics implementation
- [ ] PackageMetrics struct defined with appropriate metric types:
  - [ ] Counters for operations (e.g., `totalRequests`, `totalErrors`)
  - [ ] Histograms for durations (e.g., `operationDuration`)
  - [ ] Gauges for observable metrics (e.g., `activeOperations`)
- [ ] Metrics initialized in `NewPackageMetrics()` function
- [ ] Metrics follow naming conventions:
  - [ ] Use package prefix (e.g., `beluga.{package}.{metric_name}`)
  - [ ] Use descriptive, consistent names
  - [ ] Include appropriate attributes/labels
- [ ] All public methods record relevant metrics:
  - [ ] Operation counts
  - [ ] Operation durations
  - [ ] Error counts
  - [ ] Success/failure rates

## Tracing Integration

- [ ] OTEL tracing imported (`go.opentelemetry.io/otel/trace`)
- [ ] All public methods create spans:
  - [ ] Span name follows convention: `{package}.{method_name}`
  - [ ] Context propagated correctly
  - [ ] Span attributes set (operation type, parameters, etc.)
  - [ ] Errors recorded on spans (`span.RecordError()`)
  - [ ] Spans properly ended (defer or explicit)
- [ ] Span attributes include:
  - [ ] Package name
  - [ ] Operation type
  - [ ] Relevant parameters (sanitized)
  - [ ] Result status
- [ ] Trace context propagated to downstream calls
- [ ] Distributed tracing works across package boundaries

## Structured Logging

- [ ] Structured logging library integrated (zap or slog)
- [ ] Logger initialized with OTEL context support
- [ ] All log statements include:
  - [ ] Trace ID (from context)
  - [ ] Span ID (from context)
  - [ ] Structured fields (key-value pairs)
  - [ ] Appropriate log levels (Debug, Info, Warn, Error)
- [ ] Log context extracted from OTEL context:
  - [ ] `trace_id` from span context
  - [ ] `span_id` from span context
  - [ ] `trace_flags` from span context
- [ ] Logging follows framework patterns:
  - [ ] No fmt.Printf or log.Print
  - [ ] Context-aware logging
  - [ ] Structured fields for searchability

## Integration Verification

- [ ] Metrics collection verified:
  - [ ] Metrics exported to OTEL collector
  - [ ] Metrics visible in monitoring dashboard
  - [ ] Metric values are accurate
- [ ] Tracing verified:
  - [ ] Traces exported to OTEL collector
  - [ ] Traces visible in tracing dashboard
  - [ ] Trace context propagates correctly
  - [ ] Spans linked correctly
- [ ] Logging verified:
  - [ ] Logs include trace/span IDs
  - [ ] Logs searchable by trace ID
  - [ ] Log levels appropriate
- [ ] Performance impact assessed:
  - [ ] No significant performance degradation
  - [ ] Benchmarks show acceptable overhead
  - [ ] Sampling configured appropriately

## Consistency Check

- [ ] Patterns match `pkg/monitoring/` reference implementation
- [ ] Metric naming matches framework conventions
- [ ] Tracing patterns match framework conventions
- [ ] Logging patterns match framework conventions
- [ ] Code follows package design patterns document

## Documentation

- [ ] README.md updated with OTEL integration details
- [ ] Inline code comments explain OTEL usage
- [ ] Examples show OTEL integration
- [ ] Migration guide updated (if applicable)

## Testing

- [ ] Unit tests verify metrics recording
- [ ] Unit tests verify tracing spans
- [ ] Unit tests verify structured logging
- [ ] Integration tests verify OTEL export
- [ ] Test coverage meets requirements (100% for new code)

## Notes

[Add any specific notes, issues, or deviations from standard patterns]

---

**Completion Criteria**: All items checked, tests pass, documentation updated, integration verified.
