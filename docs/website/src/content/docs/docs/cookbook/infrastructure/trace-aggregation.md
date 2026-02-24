---
title: "Trace Aggregation for Multi-Agents"
description: "Recipe for aggregating and correlating OpenTelemetry traces across multiple Go agents to see complete end-to-end flows with Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, trace aggregation, Go multi-agent tracing, OpenTelemetry, distributed tracing, agent observability, span correlation"
---

## Problem

Multi-agent systems present a unique observability challenge. When multiple agents collaborate to solve a task, each agent generates its own spans and traces. Without proper correlation, these traces appear as disconnected operations in your observability backend, making it nearly impossible to understand the complete flow of a multi-agent interaction.

This becomes critical when debugging production issues. If a user request involves three agents, and one agent fails or performs slowly, you need to see the entire agent-to-agent communication chain to identify which agent introduced the bottleneck. Without trace aggregation, you're left examining individual agent logs and manually reconstructing the interaction timeline.

## Solution

Trace aggregation solves this by treating multi-agent workflows as a single distributed trace. The approach uses OpenTelemetry's trace ID propagation to link spans across agent boundaries. When a coordinator agent invokes a worker agent, it propagates the trace context, ensuring all operations share the same trace ID. This creates a unified view where you can see agent handoffs, measure cross-agent latency, and identify coordination bottlenecks.

The aggregator acts as a centralized collector that receives span information from all participating agents, maintains parent-child relationships between spans, and computes aggregate metrics like total duration and agent count. This design separates trace collection from trace export, allowing you to batch exports and reduce observability overhead.

The key insight is that trace IDs naturally propagate through context, so you don't need to modify agent implementations. The aggregator wrapper intercepts operations, extracts trace context, and records spans transparently.

## Code Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("beluga.agents.trace_aggregation")

// TraceAggregator aggregates traces from multiple agents.
type TraceAggregator struct {
    traces   map[string]*AggregatedTrace
    mu       sync.RWMutex
    exporter TraceExporter
}

// AggregatedTrace represents a complete trace across multiple agents.
type AggregatedTrace struct {
    TraceID       string
    RootSpan      *SpanInfo
    AgentSpans    []*SpanInfo
    StartTime     time.Time
    EndTime       time.Time
    TotalDuration time.Duration
    AgentCount    int
    Status        codes.Code
}

// SpanInfo represents a span in the aggregated trace.
type SpanInfo struct {
    SpanID       string
    ParentSpanID string
    AgentName    string
    Operation    string
    StartTime    time.Time
    EndTime      time.Time
    Duration     time.Duration
    Status       codes.Code
    Attributes   map[string]string
}

// TraceExporter exports aggregated traces.
type TraceExporter interface {
    ExportTrace(ctx context.Context, trace *AggregatedTrace) error
}

// NewTraceAggregator creates a new trace aggregator.
func NewTraceAggregator(exporter TraceExporter) *TraceAggregator {
    return &TraceAggregator{
        traces:   make(map[string]*AggregatedTrace),
        exporter: exporter,
    }
}

// StartTrace starts tracking a new multi-agent trace.
func (ta *TraceAggregator) StartTrace(ctx context.Context, traceID string, rootAgent string) context.Context {
    ctx, span := tracer.Start(ctx, "aggregator.start_trace")
    defer span.End()

    ta.mu.Lock()
    defer ta.mu.Unlock()

    aggTrace := &AggregatedTrace{
        TraceID:    traceID,
        StartTime:  time.Now(),
        AgentSpans: []*SpanInfo{},
        Status:     codes.Ok,
    }

    rootSpan := &SpanInfo{
        SpanID:     span.SpanContext().SpanID().String(),
        AgentName:  rootAgent,
        Operation:  "root",
        StartTime:  time.Now(),
        Status:     codes.Ok,
        Attributes: make(map[string]string),
    }
    aggTrace.RootSpan = rootSpan

    ta.traces[traceID] = aggTrace

    span.SetAttributes(
        attribute.String("trace_id", traceID),
        attribute.String("root_agent", rootAgent),
    )

    return ta.propagateTraceContext(ctx, traceID)
}

// AddAgentSpan adds a span from an agent to the trace.
func (ta *TraceAggregator) AddAgentSpan(ctx context.Context, traceID string, agentName string, operation string, startTime, endTime time.Time, status codes.Code, attrs map[string]string) error {
    ctx, span := tracer.Start(ctx, "aggregator.add_span")
    defer span.End()

    ta.mu.Lock()
    defer ta.mu.Unlock()

    aggTrace, exists := ta.traces[traceID]
    if !exists {
        return fmt.Errorf("trace %s not found", traceID)
    }

    agentSpan := &SpanInfo{
        SpanID:     span.SpanContext().SpanID().String(),
        AgentName:  agentName,
        Operation:  operation,
        StartTime:  startTime,
        EndTime:    endTime,
        Duration:   endTime.Sub(startTime),
        Status:     status,
        Attributes: attrs,
    }

    if len(aggTrace.AgentSpans) > 0 {
        agentSpan.ParentSpanID = aggTrace.AgentSpans[len(aggTrace.AgentSpans)-1].SpanID
    } else {
        agentSpan.ParentSpanID = aggTrace.RootSpan.SpanID
    }

    aggTrace.AgentSpans = append(aggTrace.AgentSpans, agentSpan)
    aggTrace.AgentCount++

    if status == codes.Error {
        aggTrace.Status = codes.Error
    }

    if endTime.After(aggTrace.EndTime) {
        aggTrace.EndTime = endTime
        aggTrace.TotalDuration = aggTrace.EndTime.Sub(aggTrace.StartTime)
    }

    span.SetAttributes(
        attribute.String("trace_id", traceID),
        attribute.String("agent_name", agentName),
        attribute.String("operation", operation),
    )

    return nil
}

// CompleteTrace completes a trace and exports it.
func (ta *TraceAggregator) CompleteTrace(ctx context.Context, traceID string) error {
    ctx, span := tracer.Start(ctx, "aggregator.complete_trace")
    defer span.End()

    ta.mu.Lock()
    aggTrace, exists := ta.traces[traceID]
    if !exists {
        ta.mu.Unlock()
        return fmt.Errorf("trace %s not found", traceID)
    }
    delete(ta.traces, traceID)
    ta.mu.Unlock()

    if ta.exporter != nil {
        if err := ta.exporter.ExportTrace(ctx, aggTrace); err != nil {
            span.RecordError(err)
            return fmt.Errorf("failed to export trace: %w", err)
        }
    }

    span.SetAttributes(
        attribute.String("trace_id", traceID),
        attribute.Int("agent_count", aggTrace.AgentCount),
        attribute.String("total_duration", aggTrace.TotalDuration.String()),
    )

    return nil
}

// GetTraceSummary returns a summary of a trace.
func (ta *TraceAggregator) GetTraceSummary(traceID string) (*AggregatedTrace, error) {
    ta.mu.RLock()
    defer ta.mu.RUnlock()

    aggTrace, exists := ta.traces[traceID]
    if !exists {
        return nil, fmt.Errorf("trace %s not found", traceID)
    }

    return aggTrace, nil
}

// propagateTraceContext propagates trace context to agents.
func (ta *TraceAggregator) propagateTraceContext(ctx context.Context, traceID string) context.Context {
    return context.WithValue(ctx, "trace_id", traceID)
}

// AgentTraceWrapper wraps agent operations with trace aggregation.
func AgentTraceWrapper(aggregator *TraceAggregator, agentName string, operation func(context.Context) error) func(context.Context) error {
    return func(ctx context.Context) error {
        traceID := ctx.Value("trace_id").(string)
        startTime := time.Now()

        err := operation(ctx)

        endTime := time.Now()
        status := codes.Ok
        if err != nil {
            status = codes.Error
        }

        aggregator.AddAgentSpan(ctx, traceID, agentName, "operation", startTime, endTime, status, nil)

        return err
    }
}

// LogTraceExporter exports traces to logs.
type LogTraceExporter struct{}

func (e *LogTraceExporter) ExportTrace(ctx context.Context, t *AggregatedTrace) error {
    log.Printf("Trace %s: %d agents, duration: %v, status: %v",
        t.TraceID, t.AgentCount, t.TotalDuration, t.Status)
    return nil
}

func main() {
    ctx := context.Background()

    exporter := &LogTraceExporter{}
    aggregator := NewTraceAggregator(exporter)

    traceID := "trace-123"
    ctx = aggregator.StartTrace(ctx, traceID, "coordinator")

    agent1Op := AgentTraceWrapper(aggregator, "agent1", func(ctx context.Context) error {
        time.Sleep(100 * time.Millisecond)
        return nil
    })

    agent2Op := AgentTraceWrapper(aggregator, "agent2", func(ctx context.Context) error {
        time.Sleep(50 * time.Millisecond)
        return nil
    })

    if err := agent1Op(ctx); err != nil {
        log.Printf("Agent 1 error: %v", err)
    }
    if err := agent2Op(ctx); err != nil {
        log.Printf("Agent 2 error: %v", err)
    }

    if err := aggregator.CompleteTrace(ctx, traceID); err != nil {
        log.Fatalf("Failed to complete trace: %v", err)
    }

    fmt.Println("Trace aggregation completed")
}
```

## Explanation

1. **Trace correlation via shared IDs** — OpenTelemetry trace IDs provide a natural correlation mechanism across distributed operations. By ensuring all agents use the same trace ID, you create a unified trace that observability tools can render as a single flamegraph. This eliminates the need for custom correlation logic at the application layer.

2. **Parent-child span relationships** — The aggregator maintains a span hierarchy that reflects the actual agent call graph. When agent A invokes agent B, the span from B becomes a child of A's span. This relationship is critical for understanding causality. If agent B fails, you can trace back through parent spans to see exactly what inputs led to the failure.

3. **Aggregated metrics for high-level insights** — Computing metrics like total duration and agent count at the trace level provides immediate answers to common questions: How long did this multi-agent interaction take? How many agents were involved? These metrics are expensive to compute by querying individual spans, so pre-computing them during aggregation improves dashboard performance.

4. **Transparent wrapper pattern** — The `AgentTraceWrapper` function demonstrates a key design principle: observability should not require modifying core agent logic. By wrapping agent operations, you inject trace aggregation without coupling agents to the aggregation system. This makes it easy to enable or disable aggregation, or swap to a different aggregator implementation, without touching agent code.

## Variations

### Real-Time Trace Streaming

Stream trace updates as they happen:

```go
type StreamingTraceAggregator struct {
    subscribers []chan *AggregatedTrace
}
```

### Trace Sampling

Sample traces to reduce overhead in high-throughput systems:

```go
func (ta *TraceAggregator) ShouldSample(traceID string) bool {
    // Sample 10% of traces
    return hash(traceID)%100 < 10
}
```

## Related Recipes

- [Custom S2S Voice Metrics](/cookbook/s2s-voice-metrics) — Custom metrics patterns for voice operations
