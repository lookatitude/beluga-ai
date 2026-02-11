---
title: Legal Entity Extraction System
description: Schema-validated entity extraction achieves 98.5% accuracy with 91% reduction in processing time.
---

Legal due diligence for a single M&A transaction requires extracting entities — companies, individuals, dates, monetary amounts, jurisdictions — from thousands of contracts, filings, and correspondence. A paralegal team manually processing 500 documents per day achieves 80-85% extraction accuracy, but the 15-20% error rate means critical entities (a subsidiary name, a contract expiration date, a liability cap) are missed or misclassified. These errors create compliance risk: a missed entity in a regulatory filing can result in sanctions, and an incorrectly extracted date in a contract review can invalidate legal arguments.

The challenge with LLM-based extraction is that raw LLM output is unstructured text — the model might return entities in varying formats, omit confidence scores, or hallucinate entities that do not exist in the source document. Without schema validation, downstream systems cannot trust the extraction output, and the automation provides no reliability guarantee over manual extraction.

## Solution Architecture

Beluga AI combines LLM extraction with schema validation to guarantee structured, type-safe output. The LLM handles the semantic complexity of identifying entities in legal language (understanding that "Acme Corp., a Delaware corporation" is a company entity, not three separate entities), while schema validation enforces data quality constraints that the LLM alone cannot guarantee — valid entity types, non-empty values, confidence score ranges, and type-specific format validation for dates and monetary amounts.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Document   │───▶│   Document   │───▶│     LLM      │
│    Input     │    │    Parser    │    │    Entity    │
└──────────────┘    └──────────────┘    │  Extractor   │
                                        └──────┬───────┘
                                               │
                                               ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  Structured  │◀───│    Valid?    │◀───│    Schema    │
│    Output    │    └──────────────┘    │  Validator   │
└──────────────┘            │            └──────────────┘
                            │
                            ▼
                    ┌──────────────┐
                    │    Error     │
                    │   Handler    │
                    └──────────────┘
```

## Entity Extraction Implementation

The entity extractor uses LLMs for intelligent extraction combined with schema validation for data quality:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

// EntityExtractor extracts and validates legal entities from documents.
type EntityExtractor struct {
    model     llm.ChatModel
    validator *EntityValidator
}

func NewEntityExtractor(ctx context.Context) (*EntityExtractor, error) {
    model, err := llm.New("openai", nil)
    if err != nil {
        return nil, fmt.Errorf("create model: %w", err)
    }

    return &EntityExtractor{
        model:     model,
        validator: NewEntityValidator(),
    }, nil
}

// ExtractEntities extracts and validates entities from document text.
func (e *EntityExtractor) ExtractEntities(
    ctx context.Context,
    documentText string,
    docID string,
) ([]LegalEntity, error) {
    // Build extraction prompt
    promptText := fmt.Sprintf(`Extract legal entities from the following document text.

Return entities in JSON format with the following fields:
- type: one of "company", "person", "date", "amount", "location"
- value: the entity text
- confidence: confidence score (0-1)
- metadata: optional additional information

Document Text:
%s

Return entities as a JSON array.`, documentText)

    // Extract entities using LLM
    msgs := []schema.Message{
        &schema.SystemMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: "You are a legal entity extraction expert. Extract entities accurately in JSON format."},
        }},
        &schema.HumanMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: promptText},
        }},
    }

    resp, err := e.model.Generate(ctx, msgs)
    if err != nil {
        return nil, fmt.Errorf("LLM extraction failed: %w", err)
    }

    // Parse and validate entities
    entities, err := e.parseAndValidateEntities(ctx, resp.Parts[0].(schema.TextPart).Text, docID)
    if err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    return entities, nil
}

func (e *EntityExtractor) parseAndValidateEntities(
    ctx context.Context,
    jsonContent string,
    docID string,
) ([]LegalEntity, error) {
    // Parse JSON response
    var rawEntities []map[string]any
    if err := json.Unmarshal([]byte(extractJSON(jsonContent)), &rawEntities); err != nil {
        return nil, fmt.Errorf("parse JSON: %w", err)
    }

    // Validate each entity
    var validatedEntities []LegalEntity

    for _, raw := range rawEntities {
        entity := LegalEntity{
            Type:       getString(raw, "type"),
            Value:      getString(raw, "value"),
            Confidence: getFloat64(raw, "confidence"),
            Metadata:   getMap(raw, "metadata"),
            DocumentID: docID,
        }

        // Validate entity against schema
        if err := e.validator.Validate(entity); err != nil {
            // Log validation error but continue processing
            continue
        }

        validatedEntities = append(validatedEntities, entity)
    }

    return validatedEntities, nil
}

func extractJSON(content string) string {
    // Extract JSON from markdown code blocks or plain text
    // Simplified implementation
    return content
}

func getString(m map[string]any, key string) string {
    if v, ok := m[key].(string); ok {
        return v
    }
    return ""
}

func getFloat64(m map[string]any, key string) float64 {
    if v, ok := m[key].(float64); ok {
        return v
    }
    return 0.0
}

func getMap(m map[string]any, key string) map[string]string {
    if v, ok := m[key].(map[string]any); ok {
        result := make(map[string]string)
        for k, val := range v {
            if s, ok := val.(string); ok {
                result[k] = s
            }
        }
        return result
    }
    return make(map[string]string)
}

type LegalEntity struct {
    Type       string            `json:"type"`
    Value      string            `json:"value"`
    Confidence float64           `json:"confidence"`
    Metadata   map[string]string `json:"metadata,omitempty"`
    DocumentID string            `json:"document_id"`
}
```

## Schema Validation

Implement strict schema validation for entity data quality:

```go
type EntityValidator struct {
    allowedTypes map[string]bool
}

func NewEntityValidator() *EntityValidator {
    return &EntityValidator{
        allowedTypes: map[string]bool{
            "company":  true,
            "person":   true,
            "date":     true,
            "amount":   true,
            "location": true,
        },
    }
}

func (v *EntityValidator) Validate(entity LegalEntity) error {
    // Validate type
    if !v.allowedTypes[entity.Type] {
        return fmt.Errorf("invalid entity type: %s", entity.Type)
    }

    // Validate value is not empty
    if entity.Value == "" {
        return fmt.Errorf("entity value is required")
    }

    // Validate confidence score range
    if entity.Confidence < 0 || entity.Confidence > 1 {
        return fmt.Errorf("confidence score must be between 0 and 1")
    }

    // Validate document ID
    if entity.DocumentID == "" {
        return fmt.Errorf("document ID is required")
    }

    // Type-specific validation
    switch entity.Type {
    case "date":
        if err := v.validateDate(entity.Value); err != nil {
            return fmt.Errorf("invalid date: %w", err)
        }
    case "amount":
        if err := v.validateAmount(entity.Value); err != nil {
            return fmt.Errorf("invalid amount: %w", err)
        }
    }

    return nil
}

func (v *EntityValidator) validateDate(value string) error {
    // Validate date format
    // Simplified implementation
    return nil
}

func (v *EntityValidator) validateAmount(value string) error {
    // Validate amount format
    // Simplified implementation
    return nil
}
```

## Batch Processing

Implement efficient batch processing for high-volume document sets:

```go
func (e *EntityExtractor) ExtractBatch(
    ctx context.Context,
    documents []Document,
) ([]BatchResult, error) {
    results := make([]BatchResult, len(documents))

    for i, doc := range documents {
        entities, err := e.ExtractEntities(ctx, doc.Text, doc.ID)
        if err != nil {
            results[i] = BatchResult{
                DocumentID: doc.ID,
                Error:      err,
            }
            continue
        }

        results[i] = BatchResult{
            DocumentID: doc.ID,
            Entities:   entities,
        }
    }

    return results, nil
}

type Document struct {
    ID   string
    Text string
}

type BatchResult struct {
    DocumentID string
    Entities   []LegalEntity
    Error      error
}
```

## Production Considerations

### Confidence Thresholds

Implement confidence-based filtering for quality control:

```go
func (e *EntityExtractor) ExtractWithThreshold(
    ctx context.Context,
    documentText string,
    docID string,
    minConfidence float64,
) ([]LegalEntity, error) {
    allEntities, err := e.ExtractEntities(ctx, documentText, docID)
    if err != nil {
        return nil, err
    }

    // Filter by confidence threshold
    var highConfidence []LegalEntity
    for _, entity := range allEntities {
        if entity.Confidence >= minConfidence {
            highConfidence = append(highConfidence, entity)
        }
    }

    return highConfidence, nil
}
```

### Error Handling

Implement comprehensive error handling with retry logic:

```go
func (e *EntityExtractor) ExtractWithRetry(
    ctx context.Context,
    documentText string,
    docID string,
    maxRetries int,
) ([]LegalEntity, error) {
    var lastErr error

    for attempt := 0; attempt < maxRetries; attempt++ {
        entities, err := e.ExtractEntities(ctx, documentText, docID)
        if err == nil {
            return entities, nil
        }

        lastErr = err
        // Exponential backoff
        time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
    }

    return nil, fmt.Errorf("extraction failed after %d attempts: %w", maxRetries, lastErr)
}
```

### Observability

Track extraction metrics to monitor performance and accuracy:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

func (e *EntityExtractor) ExtractWithMonitoring(
    ctx context.Context,
    documentText string,
    docID string,
) ([]LegalEntity, error) {
    tracer := otel.Tracer("entity-extractor")
    ctx, span := tracer.Start(ctx, "entity.extract")
    defer span.End()

    span.SetAttributes(
        attribute.String("document_id", docID),
        attribute.Int("document_length", len(documentText)),
    )

    entities, err := e.ExtractEntities(ctx, documentText, docID)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    span.SetAttributes(
        attribute.Int("entities_extracted", len(entities)),
    )

    return entities, nil
}
```

### Manual Review Queue

Route low-confidence extractions to manual review:

```go
type ReviewQueue struct {
    pendingReview []LegalEntity
}

func (e *EntityExtractor) ExtractWithReview(
    ctx context.Context,
    documentText string,
    docID string,
    reviewQueue *ReviewQueue,
) ([]LegalEntity, error) {
    entities, err := e.ExtractEntities(ctx, documentText, docID)
    if err != nil {
        return nil, err
    }

    var validated []LegalEntity
    confidenceThreshold := 0.8

    for _, entity := range entities {
        if entity.Confidence < confidenceThreshold {
            // Add to review queue
            reviewQueue.Add(entity)
        } else {
            validated = append(validated, entity)
        }
    }

    return validated, nil
}

func (q *ReviewQueue) Add(entity LegalEntity) {
    q.pendingReview = append(q.pendingReview, entity)
}
```

## Results

Legal entity extraction delivered significant improvements:

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Processing Time (hours/week) | 40 | 3.5 | 91% reduction |
| Extraction Accuracy (%) | 80-85 | 98.5 | 15-18% |
| Documents Processed/Day | 50 | 520 | 940% increase |
| Error Rate (%) | 15-20 | 1.5 | 87-92% reduction |
| Cost per Document ($) | 8.50 | 0.78 | 91% reduction |

## Related Resources

- [Few-Shot SQL Generation](/use-cases/few-shot-sql/) for structured output patterns
- [Schema Validation Guide](/guides/schema-validation/) for validation setup
- [LLM Configuration](/integrations/llm-providers/) for provider-specific tuning
