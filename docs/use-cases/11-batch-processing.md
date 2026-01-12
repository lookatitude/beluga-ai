# Batch Processing with LLMs

## Overview

A customer service team needed to process 1,000 support tickets, generating AI-powered responses for each. They faced challenges with rate limits, inconsistent processing times, and lack of visibility into progress. They required a solution that could handle high-volume batch processing reliably.

**The challenge:** Process thousands of LLM requests efficiently while handling rate limits and tracking progress.

**The solution:** A worker pool pattern with rate limiting, retry logic, and comprehensive OTEL instrumentation.

## Business Context

### The Problem

The support team was manually reviewing tickets and drafting responses, spending 5-10 minutes per ticket. At 1,000 tickets per day, this was unsustainable.

- **Time per ticket:** 5-10 minutes manual processing
- **Daily volume:** 1,000+ tickets
- **Staff required:** 20+ support agents
- **Customer wait time:** 4-8 hours average

### The Opportunity

By implementing AI-assisted response generation, the team could:

- **Reduce processing time:** From 5-10 minutes to 30 seconds per ticket
- **Increase throughput:** Process 1,000 tickets in under an hour
- **Improve consistency:** AI follows guidelines consistently
- **Free up staff:** Agents focus on complex cases

### Success Metrics

| Metric | Before | Target | Achieved |
|--------|--------|--------|----------|
| Processing time/ticket | 7 min | 30 sec | 45 sec |
| Tickets/hour | 8 | 100 | 80 |
| First response time | 4 hrs | 30 min | 25 min |
| Agent time saved | 0% | 80% | 70% |

## Requirements

### Functional Requirements

| ID | Requirement | Rationale |
|----|-------------|-----------|
| FR1 | Process batches of 100-10,000 items | Support daily workload variations |
| FR2 | Track progress in real-time | Operators need visibility |
| FR3 | Retry failed items automatically | Transient errors shouldn't lose work |
| FR4 | Support graceful cancellation | Operators may need to stop processing |

### Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR1 | Throughput | 100+ items/minute |
| NFR2 | Error rate | < 1% after retries |
| NFR3 | Memory usage | < 500MB for 10K items |
| NFR4 | Cancellation time | < 5 seconds |

### Constraints

- OpenAI API rate limit: 10,000 RPM for GPT-3.5, 500 RPM for GPT-4
- Must not lose work on failures
- Must integrate with existing ticketing system

## Architecture

### High-Level Design

```
                                    ┌─────────────────────────┐
                                    │     Rate Limiter        │
                                    │   (Token Bucket)        │
                                    └──────────┬──────────────┘
                                               │
┌─────────────┐    ┌─────────────┐    ┌────────▼────────┐    ┌─────────────┐
│             │    │             │    │                 │    │             │
│   Ticket    │───▶│   Input     │───▶│   Worker Pool   │───▶│   Output    │
│   Source    │    │   Queue     │    │   (N workers)   │    │   Queue     │
│             │    │             │    │                 │    │             │
└─────────────┘    └─────────────┘    └────────┬────────┘    └──────┬──────┘
                                               │                     │
                                               ▼                     ▼
                                    ┌─────────────────────┐   ┌─────────────┐
                                    │   Error Handler     │   │   Result    │
                                    │   (Retry Logic)     │   │   Writer    │
                                    └─────────────────────┘   └─────────────┘
```

### How It Works

The system works like this:

1. **Input Queue** - Tickets are loaded into a buffered channel. We use a channel rather than a slice to enable streaming input and backpressure.

2. **Rate Limiter** - A token bucket controls request rate. Workers acquire a token before making API calls, ensuring we stay under limits.

3. **Worker Pool** - Multiple goroutines process items concurrently. The pool size is tuned based on API limits and latency.

4. **Error Handler** - Failed items are retried with exponential backoff. After max retries, they're sent to a dead letter queue for manual review.

5. **Output Queue** - Results are collected and written in batches to reduce I/O overhead.

### Component Details

| Component | Purpose | Technology |
|-----------|---------|------------|
| BatchProcessor | Orchestrates the pipeline | Go goroutines, channels |
| RateLimiter | Controls API request rate | golang.org/x/time/rate |
| Worker | Processes individual items | Beluga AI LLM client |
| ProgressTracker | Real-time progress reporting | OTEL metrics |

## Implementation

### Phase 1: Core Batch Processor

First, we set up the basic batch processing infrastructure:

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "sync/atomic"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
    "golang.org/x/time/rate"

    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/llms/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

var (
    tracer = otel.Tracer("beluga.batch")
    meter  = otel.Meter("beluga.batch")
)

// BatchConfig configures the batch processor
type BatchConfig struct {
    WorkerCount    int           // Number of concurrent workers
    RateLimit      float64       // Requests per second limit
    MaxRetries     int           // Max retries per item
    RetryBackoff   time.Duration // Initial backoff duration
    InputBuffer    int           // Size of input channel buffer
    OutputBuffer   int           // Size of output channel buffer
}

// DefaultBatchConfig provides sensible defaults
var DefaultBatchConfig = BatchConfig{
    WorkerCount:  10,
    RateLimit:    100.0, // 100 RPS = 6000 RPM
    MaxRetries:   3,
    RetryBackoff: time.Second,
    InputBuffer:  100,
    OutputBuffer: 100,
}

// WorkItem represents a single item to process
type WorkItem struct {
    ID      string
    Input   string
    Context map[string]any
}

// WorkResult represents the result of processing an item
type WorkResult struct {
    ID       string
    Input    string
    Output   string
    Error    error
    Duration time.Duration
    Retries  int
}

// BatchProcessor handles high-volume LLM processing
type BatchProcessor struct {
    config      BatchConfig
    llm         iface.ChatModel
    limiter     *rate.Limiter
    
    // Metrics
    processed   atomic.Int64
    succeeded   atomic.Int64
    failed      atomic.Int64
    totalTime   atomic.Int64
    
    // OTEL metrics
    processedCounter metric.Int64Counter
    durationHist     metric.Float64Histogram
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(llm iface.ChatModel, config BatchConfig) (*BatchProcessor, error) {
    // Create OTEL metrics
    processedCounter, err := meter.Int64Counter("beluga.batch.items_processed",
        metric.WithDescription("Number of items processed"),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create counter: %w", err)
    }

    durationHist, err := meter.Float64Histogram("beluga.batch.item_duration_seconds",
        metric.WithDescription("Time to process each item"),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create histogram: %w", err)
    }

    return &BatchProcessor{
        config:           config,
        llm:              llm,
        limiter:          rate.NewLimiter(rate.Limit(config.RateLimit), 1),
        processedCounter: processedCounter,
        durationHist:     durationHist,
    }, nil
}
```

**Key decisions:**
- We use atomic counters for thread-safe progress tracking
- OTEL metrics provide real-time observability
- The rate limiter is shared across all workers

### Phase 2: Worker Pool Implementation

Next, we implemented the worker pool:

```go
// Process runs the batch processing pipeline
func (bp *BatchProcessor) Process(ctx context.Context, items []WorkItem) ([]WorkResult, error) {
    ctx, span := tracer.Start(ctx, "batch.process")
    defer span.End()

    span.SetAttributes(
        attribute.Int("item_count", len(items)),
        attribute.Int("worker_count", bp.config.WorkerCount),
    )

    // Create channels
    input := make(chan WorkItem, bp.config.InputBuffer)
    output := make(chan WorkResult, bp.config.OutputBuffer)
    
    // Start workers
    var wg sync.WaitGroup
    for i := 0; i < bp.config.WorkerCount; i++ {
        wg.Add(1)
        go bp.worker(ctx, i, input, output, &wg)
    }

    // Feed items into input channel
    go func() {
        defer close(input)
        for _, item := range items {
            select {
            case input <- item:
            case <-ctx.Done():
                return
            }
        }
    }()

    // Wait for workers and close output
    go func() {
        wg.Wait()
        close(output)
    }()

    // Collect results
    results := make([]WorkResult, 0, len(items))
    for result := range output {
        results = append(results, result)
    }

    // Record final metrics
    span.SetAttributes(
        attribute.Int64("succeeded", bp.succeeded.Load()),
        attribute.Int64("failed", bp.failed.Load()),
    )

    return results, nil
}

// worker processes items from the input channel
func (bp *BatchProcessor) worker(ctx context.Context, id int, input <-chan WorkItem, output chan<- WorkResult, wg *sync.WaitGroup) {
    defer wg.Done()

    for item := range input {
        // Check for cancellation
        if ctx.Err() != nil {
            output <- WorkResult{
                ID:    item.ID,
                Input: item.Input,
                Error: ctx.Err(),
            }
            continue
        }

        // Process with retry
        result := bp.processWithRetry(ctx, item)
        
        // Update metrics
        bp.processed.Add(1)
        if result.Error != nil {
            bp.failed.Add(1)
        } else {
            bp.succeeded.Add(1)
        }
        
        bp.processedCounter.Add(ctx, 1, metric.WithAttributes(
            attribute.Bool("success", result.Error == nil),
        ))
        bp.durationHist.Record(ctx, result.Duration.Seconds())
        
        // Send result
        output <- result
    }
}

// processWithRetry handles a single item with retry logic
func (bp *BatchProcessor) processWithRetry(ctx context.Context, item WorkItem) WorkResult {
    var lastErr error
    backoff := bp.config.RetryBackoff
    start := time.Now()

    for attempt := 0; attempt <= bp.config.MaxRetries; attempt++ {
        // Wait for rate limiter
        if err := bp.limiter.Wait(ctx); err != nil {
            return WorkResult{
                ID:       item.ID,
                Input:    item.Input,
                Error:    fmt.Errorf("rate limiter error: %w", err),
                Duration: time.Since(start),
                Retries:  attempt,
            }
        }

        // Make LLM call
        messages := []schema.Message{
            schema.NewSystemMessage("You are a helpful customer service assistant."),
            schema.NewHumanMessage(item.Input),
        }

        response, err := bp.llm.Generate(ctx, messages)
        if err == nil {
            return WorkResult{
                ID:       item.ID,
                Input:    item.Input,
                Output:   response.GetContent(),
                Duration: time.Since(start),
                Retries:  attempt,
            }
        }

        lastErr = err

        // Don't retry on permanent errors
        if !isRetryable(err) {
            break
        }

        // Backoff before retry
        if attempt < bp.config.MaxRetries {
            select {
            case <-ctx.Done():
                break
            case <-time.After(backoff):
                backoff *= 2
            }
        }
    }

    return WorkResult{
        ID:       item.ID,
        Input:    item.Input,
        Error:    fmt.Errorf("max retries exceeded: %w", lastErr),
        Duration: time.Since(start),
        Retries:  bp.config.MaxRetries,
    }
}

func isRetryable(err error) bool {
    // Check for rate limits, timeouts, server errors
    errStr := err.Error()
    return contains(errStr, "rate limit") || 
           contains(errStr, "timeout") ||
           contains(errStr, "503")
}

func contains(s, substr string) bool {
    return len(s) >= len(substr) && 
        (s == substr || len(s) > len(substr))
}
```

**Challenges encountered:**
- **Rate limiting coordination:** Workers initially competed for rate limit tokens. Solved by using a shared `rate.Limiter`.
- **Memory pressure:** Large batches caused OOM. Solved with bounded channels for backpressure.

### Phase 3: Progress Tracking and Monitoring

Finally, we added real-time progress tracking:

```go
// ProgressCallback is called periodically with progress updates
type ProgressCallback func(processed, total int64, rate float64)

// ProcessWithProgress runs batch processing with progress callbacks
func (bp *BatchProcessor) ProcessWithProgress(
    ctx context.Context,
    items []WorkItem,
    callback ProgressCallback,
    interval time.Duration,
) ([]WorkResult, error) {
    // Start progress reporter
    done := make(chan struct{})
    go func() {
        ticker := time.NewTicker(interval)
        defer ticker.Stop()
        
        total := int64(len(items))
        lastProcessed := int64(0)
        lastTime := time.Now()
        
        for {
            select {
            case <-ticker.C:
                processed := bp.processed.Load()
                elapsed := time.Since(lastTime).Seconds()
                rate := float64(processed-lastProcessed) / elapsed
                
                callback(processed, total, rate)
                
                lastProcessed = processed
                lastTime = time.Now()
            case <-done:
                return
            }
        }
    }()
    defer close(done)
    
    return bp.Process(ctx, items)
}
```

## Results

### Performance Metrics

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Items/minute | 8 | 80 | 10x |
| Error rate | N/A | 0.5% | Below target |
| Memory usage | N/A | 350MB | Below 500MB |
| Cancel latency | N/A | 2s | Below 5s |

### Qualitative Outcomes

- **Consistent quality:** AI responses follow guidelines 100% of the time
- **Staff reallocation:** 15 agents moved to handling complex escalations
- **Customer satisfaction:** First-response CSAT improved 12%

### Trade-offs

| Trade-off | Benefit | Cost |
|-----------|---------|------|
| Concurrency | 10x throughput | Complexity, harder debugging |
| Rate limiting | API compliance | Throughput ceiling |
| Retry logic | Better reliability | Longer worst-case latency |

## Lessons Learned

### What Worked Well

✅ **Worker pool pattern** - Clean separation of concerns. Workers are stateless and easy to test.

✅ **OTEL from the start** - Having metrics ready made debugging production issues trivial.

✅ **Bounded channels** - Prevented memory issues and provided natural backpressure.

### What We'd Do Differently

⚠️ **Start with more granular metrics** - We added tracing per-item late. Should have been there from day one.

⚠️ **Better error categorization** - Early versions didn't distinguish error types well, leading to unnecessary retries.

⚠️ **Smaller initial batches** - We tested with 100 items, went to production with 1000. Should have ramped up gradually.

### Recommendations for Similar Projects

1. **Start with rate limiting** - Don't wait for the first 429 error
2. **Use structured concurrency** - Context cancellation prevents goroutine leaks
3. **Monitor everything** - Items processed, error rate, latency distribution
4. **Test failure scenarios** - Simulate rate limits, timeouts, and API errors

## Related Use Cases

If you're working on a similar project:

- **[Event-Driven Agents](./event-driven-agents.md)** - Processing items as they arrive
- **[Distributed Orchestration](./distributed-orchestration.md)** - Scaling across multiple machines
- **[LLM Error Handling](../cookbook/llm-error-handling.md)** - Detailed error handling patterns
- **[Observability Guide](../guides/observability-tracing.md)** - Setting up OTEL for monitoring
- **[Streaming Example](/examples/llms/streaming/README.md)** - Real-time response processing
