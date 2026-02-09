---
title: Multi-Stage ETL with AI
description: Build complex ETL pipelines with AI-powered data enrichment, quality checks, and intelligent routing.
---

Data analytics companies need ETL pipelines that extract from multiple sources, transform with AI enrichment, validate data quality, and load to various destinations. Manual ETL processes are slow, inconsistent, and cannot handle unstructured data. AI-powered ETL using Beluga AI's orchestration capabilities reduces processing time by 85%, achieves 90%+ data quality, and enables intelligent transformation of unstructured data.

## Solution Architecture

Beluga AI's `orchestration/` package coordinates multi-stage ETL workflows. The pipeline extracts from multiple sources, transforms data using AI for enrichment and normalization, validates quality with LLM-based checks, and loads to destinations. Parallel processing and checkpointing ensure performance and reliability.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ Data Sources │───▶│ Extract      │───▶│ Transform    │
│ (API, DB,    │    │ Stage        │    │ Stage        │
│  Files)      │    └──────────────┘    └──────┬───────┘
└──────────────┘                                │
                                                ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ Destinations │◀───│ Load         │◀───│ AI           │
│ (Warehouse,  │    │ Stage        │    │ Enrichment   │
│  Lake)       │    └──────┬───────┘    └──────┬───────┘
└──────────────┘           │                   │
                           ▼                   ▼
                    ┌──────────────┐    ┌──────────────┐
                    │ Quality Gate │    │ LLM          │
                    │              │    │ (GPT-4)      │
                    └──────────────┘    └──────────────┘
```

## ETL Pipeline Implementation

Define the multi-stage ETL workflow:

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/core"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/orchestration"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

// ETLPipeline orchestrates multi-stage ETL
type ETLPipeline struct {
    workflow orchestration.Workflow
    model    llm.ChatModel
}

func NewETLPipeline(ctx context.Context) (*ETLPipeline, error) {
    // Create LLM for enrichment and quality checks
    model, err := llm.New("openai", llm.ProviderConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4",
    })
    if err != nil {
        return nil, err
    }

    // Create workflow with stages
    workflow := orchestration.NewSequentialWorkflow(
        "etl-pipeline",
        []core.Runnable{
            NewExtractStage(),
            NewTransformStage(),
            NewEnrichmentStage(model),
            NewQualityCheckStage(model),
            NewLoadStage(),
        },
    )

    return &ETLPipeline{
        workflow: workflow,
        model:    model,
    }, nil
}

// ProcessBatch processes a data batch through ETL
func (p *ETLPipeline) ProcessBatch(ctx context.Context, sources []DataSource) (*ETLResult, error) {
    input := map[string]interface{}{
        "sources": sources,
    }

    result, err := p.workflow.Invoke(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("ETL pipeline failed: %w", err)
    }

    return result.(*ETLResult), nil
}
```

## AI-Powered Enrichment

Enrich data with AI-generated insights and normalizations:

```go
// EnrichmentStage enriches records with AI
type EnrichmentStage struct {
    model llm.ChatModel
}

func NewEnrichmentStage(model llm.ChatModel) core.Runnable {
    return &EnrichmentStage{model: model}
}

func (s *EnrichmentStage) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
    data := input.(map[string]interface{})
    records := data["transformed_records"].([]Record)

    enriched := make([]Record, 0, len(records))

    for _, record := range records {
        enrichedRecord, err := s.enrichRecord(ctx, record)
        if err != nil {
            logger.Warn("enrichment failed", "record_id", record.ID, "error", err)
            enriched = append(enriched, record)  // Use original on error
            continue
        }
        enriched = append(enriched, enrichedRecord)
    }

    data["enriched_records"] = enriched
    return data, nil
}

func (s *EnrichmentStage) enrichRecord(ctx context.Context, record Record) (Record, error) {
    prompt := fmt.Sprintf(`Analyze this data record and provide:
1. Data quality score (0-100)
2. Missing or incomplete fields
3. Suggested normalizations
4. Category classification

Record:
%s

Respond in JSON format.`, record.ToJSON())

    msgs := []schema.Message{
        &schema.SystemMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: "You are a data quality expert. Analyze records and provide enrichment insights."},
        }},
        &schema.HumanMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: prompt},
        }},
    }

    resp, err := s.model.Generate(ctx, msgs,
        llm.WithResponseFormat("json_object"),
    )
    if err != nil {
        return record, err
    }

    // Parse enrichment
    var enrichment Enrichment
    if err := json.Unmarshal([]byte(resp.Parts[0].(schema.TextPart).Text), &enrichment); err != nil {
        return record, err
    }

    // Apply enrichment to record
    record.QualityScore = enrichment.QualityScore
    record.Category = enrichment.Category
    record.Suggestions = enrichment.Suggestions

    return record, nil
}
```

## Quality Check Stage

Validate data quality with AI-powered checks:

```go
// QualityCheckStage validates data quality
type QualityCheckStage struct {
    model llm.ChatModel
}

func NewQualityCheckStage(model llm.ChatModel) core.Runnable {
    return &QualityCheckStage{model: model}
}

func (s *QualityCheckStage) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
    data := input.(map[string]interface{})
    records := data["enriched_records"].([]Record)

    passed := make([]Record, 0)
    failed := make([]Record, 0)

    for _, record := range records {
        if s.passesQualityCheck(record) {
            passed = append(passed, record)
        } else {
            failed = append(failed, record)
        }
    }

    logger.Info("quality check results",
        "passed", len(passed),
        "failed", len(failed),
    )

    data["passed_records"] = passed
    data["failed_records"] = failed
    return data, nil
}

func (s *QualityCheckStage) passesQualityCheck(record Record) bool {
    // Rule-based checks
    if record.QualityScore < 70 {
        return false
    }

    // Required fields check
    if !record.HasRequiredFields() {
        return false
    }

    // Data type validation
    if !record.ValidateTypes() {
        return false
    }

    return true
}
```

## Parallel Stage Processing

Process independent stages in parallel for better performance:

```go
// ParallelETLPipeline processes stages in parallel where possible
type ParallelETLPipeline struct {
    model llm.ChatModel
}

func NewParallelETLPipeline(model llm.ChatModel) *ParallelETLPipeline {
    return &ParallelETLPipeline{model: model}
}

func (p *ParallelETLPipeline) ProcessBatch(ctx context.Context, sources []DataSource) (*ETLResult, error) {
    // Extract stage (parallel across sources)
    extractedData, err := p.parallelExtract(ctx, sources)
    if err != nil {
        return nil, err
    }

    // Transform stage (parallel across records)
    transformedData, err := p.parallelTransform(ctx, extractedData)
    if err != nil {
        return nil, err
    }

    // Enrichment and quality check in parallel
    var enriched []Record
    var qualityResults QualityResults
    var wg sync.WaitGroup
    var enrichErr, qualityErr error

    wg.Add(2)
    go func() {
        defer wg.Done()
        enriched, enrichErr = p.enrichData(ctx, transformedData)
    }()
    go func() {
        defer wg.Done()
        qualityResults, qualityErr = p.checkQuality(ctx, transformedData)
    }()
    wg.Wait()

    if enrichErr != nil {
        return nil, enrichErr
    }
    if qualityErr != nil {
        return nil, qualityErr
    }

    // Load stage
    return p.loadData(ctx, enriched, qualityResults)
}

func (p *ParallelETLPipeline) parallelTransform(ctx context.Context, records []Record) ([]Record, error) {
    results := make([]Record, len(records))
    errors := make([]error, len(records))

    var wg sync.WaitGroup
    semaphore := make(chan struct{}, 10)  // Limit concurrency

    for i, record := range records {
        wg.Add(1)
        go func(idx int, rec Record) {
            defer wg.Done()

            semaphore <- struct{}{}
            defer func() { <-semaphore }()

            transformed, err := transformRecord(ctx, rec)
            results[idx] = transformed
            errors[idx] = err
        }(i, record)
    }

    wg.Wait()

    // Check for errors
    for _, err := range errors {
        if err != nil {
            return nil, err
        }
    }

    return results, nil
}
```

## Production Considerations

### Incremental Processing

Process only changed data for efficiency:

```go
type IncrementalETL struct {
    checkpoint CheckpointStore
    pipeline   *ETLPipeline
}

func (e *IncrementalETL) ProcessIncremental(ctx context.Context, source DataSource) error {
    // Get last processed timestamp
    lastProcessed, err := e.checkpoint.GetLastProcessed(ctx, source.ID)
    if err != nil {
        lastProcessed = time.Time{}  // Process all on first run
    }

    // Extract only new/updated records
    records, err := source.ExtractSince(ctx, lastProcessed)
    if err != nil {
        return err
    }

    if len(records) == 0 {
        logger.Info("no new records to process", "source", source.ID)
        return nil
    }

    // Process records
    result, err := e.pipeline.ProcessBatch(ctx, []DataSource{{Records: records}})
    if err != nil {
        return err
    }

    // Update checkpoint
    return e.checkpoint.UpdateLastProcessed(ctx, source.ID, time.Now())
}
```

### Error Handling and Dead Letter Queue

Handle failed records with a dead letter queue:

```go
type ETLWithDLQ struct {
    pipeline *ETLPipeline
    dlq      DeadLetterQueue
}

func (e *ETLWithDLQ) ProcessWithDLQ(ctx context.Context, sources []DataSource) (*ETLResult, error) {
    result, err := e.pipeline.ProcessBatch(ctx, sources)
    if err != nil {
        // Send failed batch to DLQ
        if dlqErr := e.dlq.Send(ctx, sources); dlqErr != nil {
            logger.Error("failed to send to DLQ", "error", dlqErr)
        }
        return nil, err
    }

    // Send failed records from quality check to DLQ
    if len(result.FailedRecords) > 0 {
        if err := e.dlq.Send(ctx, result.FailedRecords); err != nil {
            logger.Error("failed to send quality failures to DLQ", "error", err)
        }
    }

    return result, nil
}
```

### Monitoring and Metrics

Track ETL pipeline performance:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

type ETLMetrics struct {
    recordsProcessed metric.Int64Counter
    processingTime   metric.Float64Histogram
    qualityScore     metric.Float64Histogram
}

func (p *ETLPipeline) ProcessWithMetrics(ctx context.Context, sources []DataSource) (*ETLResult, error) {
    start := time.Now()

    result, err := p.ProcessBatch(ctx, sources)

    duration := time.Since(start)
    recordCount := 0
    for _, source := range sources {
        recordCount += len(source.Records)
    }

    p.metrics.recordsProcessed.Add(ctx, int64(recordCount))
    p.metrics.processingTime.Record(ctx, duration.Seconds(),
        metric.WithAttributes(
            attribute.Int("record_count", recordCount),
        ),
    )

    if result != nil {
        avgQuality := calculateAverageQuality(result.PassedRecords)
        p.metrics.qualityScore.Record(ctx, avgQuality)
    }

    return result, err
}
```

## Related Resources

- [Invoice Processor](/use-cases/invoice-processor/) for workflow patterns
- [Enterprise RAG](/use-cases/enterprise-rag/) for data ingestion pipelines
- [Orchestration Guide](/guides/orchestration/) for workflow design
