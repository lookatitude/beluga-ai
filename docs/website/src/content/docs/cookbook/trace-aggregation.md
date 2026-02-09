---
title: "Trace Aggregation for Multi-Agents"
description: "Aggregate and correlate traces from multiple agents to see complete end-to-end flows across agent boundaries."
---

## Problem

You need to aggregate and correlate traces from multiple agents working together in a multi-agent system, so you can see the complete end-to-end flow across agent boundaries and identify bottlenecks or failures in agent coordination.

## Solution

Implement trace aggregation that collects spans from multiple agents, correlates them by trace ID and parent-child relationships, and provides aggregated views of multi-agent workflows. OpenTelemetry uses trace IDs to link related spans, and you can propagate these IDs through agent-to-agent communication.

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

1. **Trace correlation** — A single trace ID is used across all agents. Each agent's spans are linked to this trace, allowing you to see the complete flow. The trace ID propagates through context, ensuring all agent operations are correlated.

2. **Span hierarchy** — Parent-child relationships between spans are maintained. This creates a tree structure showing how agents call each other, making it straightforward to identify which agent triggered which operation.

3. **Aggregated metrics** — Aggregate metrics like total duration and agent count are computed. This provides high-level insights into multi-agent performance without needing to query individual spans.

4. **Agent wrapper** — The `AgentTraceWrapper` function transparently adds trace aggregation to any agent operation. This avoids scattering tracing code throughout agent implementations.

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
