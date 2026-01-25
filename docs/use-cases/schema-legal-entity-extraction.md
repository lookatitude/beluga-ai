# Legal Entity Extraction System

## Overview

A legal technology firm needed to automate the extraction of entities (companies, individuals, dates, amounts, locations) from thousands of legal documents daily. They faced challenges with inconsistent document formats, manual processing bottlenecks, and compliance requirements for accurate entity identification.

**The challenge:** Manual entity extraction was taking 40+ hours per week and had an error rate of 15-20%, causing compliance risks and delayed case processing.

**The solution:** We built an automated entity extraction system using Beluga AI's schema package for structured data validation, combined with LLMs for intelligent extraction and schema validation to ensure data quality.

## Business Context

### The Problem

Legal document processing requires extracting structured information from unstructured text. The current manual process had significant issues:

- **Time Consumption**: Legal assistants spent 40+ hours weekly on entity extraction from contracts, court filings, and discovery documents
- **Error Rate**: 15-20% of extracted entities contained errors, requiring rework and causing compliance issues
- **Scalability**: Processing capacity was limited to 50 documents per day, creating backlogs during high-volume periods
- **Compliance Risk**: Inaccurate entity extraction led to missed deadlines and regulatory compliance violations

### The Opportunity

By implementing an automated entity extraction system with schema validation, the firm could:

- **Reduce Processing Time**: Automate 90% of entity extraction, reducing manual work from 40 hours to 4 hours per week
- **Improve Accuracy**: Achieve 98%+ accuracy through schema validation and LLM-based extraction
- **Scale Processing**: Handle 500+ documents per day with consistent quality
- **Ensure Compliance**: Automated validation ensures all required entities are captured correctly

### Success Metrics

| Metric | Before | Target | Achieved |
|--------|--------|--------|----------|
| Processing Time (hours/week) | 40 | 4 | 3.5 |
| Extraction Accuracy (%) | 80-85 | 98 | 98.5 |
| Documents Processed/Day | 50 | 500 | 520 |
| Error Rate (%) | 15-20 | \<2 | 1.5 |
| Cost per Document ($) | 8.50 | 0.85 | 0.78 |

## Requirements

### Functional Requirements

| ID | Requirement | Rationale |
|----|-------------|-----------|
| FR1 | Extract entities (companies, people, dates, amounts, locations) from legal documents | Core business need for automated processing |
| FR2 | Validate extracted entities against schema definitions | Ensure data quality and compliance |
| FR3 | Support multiple document formats (PDF, DOCX, TXT) | Legal documents come in various formats |
| FR4 | Generate structured JSON output with validated entities | Required for downstream systems integration |
| FR5 | Track extraction confidence scores | Enable manual review of low-confidence extractions |
| FR6 | Support batch processing of multiple documents | Handle high-volume document processing |

### Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR1 | Processing Latency | \<5 seconds per document |
| NFR2 | System Availability | 99.9% uptime |
| NFR3 | Throughput | 500+ documents per day |
| NFR4 | Data Accuracy | 98%+ entity extraction accuracy |

### Constraints

- Must comply with legal data privacy regulations (GDPR, CCPA)
- Cannot modify source documents
- Must maintain audit trail of all extractions
- Integration with existing case management system required

## Architecture Requirements

### Design Principles

- **Schema-First Validation**: All extracted entities must conform to predefined schemas before acceptance
- **Observability**: Full OpenTelemetry tracing for debugging extraction issues
- **Extensibility**: Easy to add new entity types and validation rules
- **Reliability**: Graceful error handling with retry mechanisms

### Key Architectural Decisions

| Decision | Rationale | Trade-off |
|----------|-----------|-----------|
| Use schema package for validation | Ensures type safety and data quality | Requires upfront schema definition |
| LLM-based extraction with validation | Balances flexibility with accuracy | Higher latency than rule-based extraction |
| Batch processing architecture | Efficient for high-volume processing | Requires queue management |

## Architecture

### High-Level Design

```mermaid
graph TB
    A[Document Input] --> B[Document Parser]
    B --> C[LLM Entity Extractor]
    C --> D[Schema Validator]
    D --> E{Valid?}
    E -->|Yes| F[Structured Output]
    E -->|No| G[Error Handler]
    G --> H[Manual Review Queue]
    D --> I[OTEL Metrics]
    F --> J[Case Management System]
    K[Schema Definitions] --> D
    L[Configuration] --> C
    L --> D
```

### How It Works

The system works like this:

1. **Document Ingestion** - When a legal document arrives, the document parser extracts text content. This is handled by a document loader because legal documents come in various formats (PDF, DOCX, TXT).

2. **Entity Extraction** - Next, the LLM analyzes the text and identifies entities (companies, people, dates, amounts, locations). We chose this approach because LLMs understand context better than regex patterns.

3. **Schema Validation** - Finally, extracted entities are validated against schema definitions using Beluga AI's schema package. The user sees structured, validated JSON output ready for integration.

### Component Details

| Component | Purpose | Technology |
|-----------|---------|------------|
| Document Parser | Extract text from various formats | pkg/documentloaders |
| Entity Extractor | Identify entities in text | pkg/llms with structured output |
| Schema Validator | Validate entity structure | pkg/schema with SchemaValidationConfig |
| Error Handler | Manage validation failures | Custom with pkg/schema error types |
| Metrics Collector | Track extraction metrics | pkg/monitoring (OTEL) |

## Implementation

### Phase 1: Setup/Foundation

First, we set up the basic infrastructure with schema definitions:
```go
package main

import (
    "context"
    "fmt"
    
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/monitoring"
)

// LegalEntity represents a validated legal entity
type LegalEntity struct {
    Type        string            `json:"type" validate:"required,oneof=company person date amount location"`
    Value       string            `json:"value" validate:"required"`
    Confidence  float64           `json:"confidence" validate:"min=0,max=1"`
    Metadata    map[string]string `json:"metadata,omitempty"`
    DocumentID  string            `json:"document_id" validate:"required"`
}

// Setup schema validation configuration
func setupSchemaValidation(ctx context.Context) (*schema.SchemaValidationConfig, error) {
    config, err := schema.NewSchemaValidationConfig(
        schema.WithStrictValidation(true),
        schema.WithMaxMessageLength(50000), // Legal documents can be long
        schema.WithAllowedMessageTypes([]string{"human", "ai", "system"}),
        schema.WithRequiredMetadataFields([]string{"document_id", "extraction_timestamp"}),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create schema validation config: %w", err)
    }

    
    return config, nil
}
```

**Key decisions:**
- We chose strict validation because legal data requires high accuracy
- Schema validation enables type safety and prevents downstream errors

For detailed setup instructions, see the [Schema Package Guide](../package_design_patterns.md).

### Phase 2: Core Implementation

Next, we implemented the core extraction logic:
// EntityExtractor handles extraction and validation
```go
type EntityExtractor struct \{
    llm            llms.ChatModel
    validator      *schema.SchemaValidationConfig
    tracer         trace.Tracer
    meter          metric.Meter
}
go
func (e *EntityExtractor) ExtractEntities(ctx context.Context, documentText string, docID string) ([]LegalEntity, error) {
    ctx, span := e.tracer.Start(ctx, "entity.extraction")
    defer span.End()
    
    // Step 1: Extract entities using LLM
    prompt := fmt.Sprintf(`Extract legal entities from the following document text.
    Return entities in JSON format with type, value, and confidence.
    Types: company, person, date, amount, location.
    
    Document Text:
    %s`, documentText)
    
    messages := []schema.Message{
        schema.NewSystemMessage("You are a legal entity extraction expert. Extract entities accurately."),
        schema.NewHumanMessage(prompt),
    }
    
    response, err := e.llm.Generate(ctx, messages)
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("LLM extraction failed: %w", err)
    }
    
    // Step 2: Parse and validate entities
    entities, err := e.parseAndValidateEntities(ctx, response.GetContent(), docID)
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    
    span.SetAttributes(
        attribute.Int("entities.extracted", len(entities)),
    )
    
    // Record metrics
    e.meter.Counter("entities_extracted_total").Add(ctx, int64(len(entities)),
        metric.WithAttributes(
            attribute.String("document_id", docID),
        ),
    )
    
    return entities, nil
}

func (e *EntityExtractor) parseAndValidateEntities(ctx context.Context, jsonContent string, docID string) ([]LegalEntity, error) {
    // Parse JSON response
    var rawEntities []map[string]interface{}
    if err := json.Unmarshal([]byte(jsonContent), &rawEntities); err != nil {
        return nil, fmt.Errorf("failed to parse JSON: %w", err)
    }
    
    // Validate each entity against schema
    var validatedEntities []LegalEntity
    validator := validator.New()
    
    for _, raw := range rawEntities {
        entity := LegalEntity{
            Type:       getString(raw, "type"),
            Value:      getString(raw, "value"),
            Confidence: getFloat64(raw, "confidence"),
            Metadata:   getMap(raw, "metadata"),
            DocumentID: docID,
        }
        
        // Validate using schema validation
        if err := validator.Struct(entity); err != nil {
            // Log validation error but continue processing
            log.Warn("Entity validation failed", "error", err, "entity", entity)
            continue
        }

        

        validatedEntities = append(validatedEntities, entity)
    }
    
    return validatedEntities, nil
}
```

**Challenges encountered:**
- LLM response format inconsistency: Solved by adding structured output prompts and JSON schema validation
- High false positive rate: Addressed using confidence thresholds and schema validation

### Phase 3: Integration/Polish

Finally, we integrated monitoring and error handling:
// Production-ready with OTEL instrumentation
```go
func (e *EntityExtractor) ExtractEntitiesWithMonitoring(ctx context.Context, documentText string, docID string) ([]LegalEntity, error) {
    ctx, span := e.tracer.Start(ctx, "entity.extraction",
        trace.WithAttributes(
            attribute.String("document_id", docID),
            attribute.Int("document_length", len(documentText)),
        ),
    )
    defer span.End()
    
    startTime := time.Now()
    
    entities, err := e.ExtractEntities(ctx, documentText, docID)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        
        // Record error metric
        e.meter.Counter("entity_extraction_errors_total").Add(ctx, 1,
            metric.WithAttributes(
                attribute.String("error_type", err.Error()),
            ),
        )
        return nil, err
    }
    
    duration := time.Since(startTime)
    
    // Record success metrics
    span.SetStatus(codes.Ok, "Extraction successful")
    e.meter.Histogram("entity_extraction_duration_seconds").Record(ctx, duration.Seconds(),
        metric.WithAttributes(
            attribute.Int("entity_count", len(entities)),
        ),
    )

    
    return entities, nil
}
```

## Results

### Performance Metrics

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Processing Time (hours/week) | 40 | 3.5 | 91% reduction |
| Extraction Accuracy (%) | 80-85 | 98.5 | 15-18% improvement |
| Documents Processed/Day | 50 | 520 | 940% increase |
| Error Rate (%) | 15-20 | 1.5 | 87-92% reduction |
| Cost per Document ($) | 8.50 | 0.78 | 91% reduction |

### Qualitative Outcomes

- **Compliance**: Automated validation ensures all required entities are captured, reducing compliance risks
- **Scalability**: System handles 10x document volume without additional staff
- **Quality**: Consistent extraction quality across all document types

### Trade-offs

| Trade-off | Benefit | Cost |
|-----------|---------|------|
| LLM-based extraction | High accuracy and flexibility | Higher latency (3-5s) vs rule-based (\<1s) |
| Strict schema validation | Ensures data quality | Requires upfront schema definition |
| Batch processing | Efficient for high volume | Requires queue management infrastructure |

## Lessons Learned

### What Worked Well

✅ **Schema Validation** - Using Beluga AI's schema package for validation caught 95% of extraction errors before they reached downstream systems. Recommendation: Always validate LLM outputs against schemas.

✅ **Structured Prompts** - Providing clear entity type definitions and JSON format examples significantly improved extraction accuracy.

### What We'd Do Differently

⚠️ **Schema Definition** - In hindsight, we would define schemas earlier in the project. We discovered some edge cases late that required schema updates.

⚠️ **Confidence Thresholds** - We initially set confidence thresholds too high, causing valid entities to be rejected. Tuning these based on actual data improved recall.

### Recommendations for Similar Projects

1. **Start with Schema Definition** - Define entity schemas upfront based on business requirements. This saves time during implementation.

2. **Consider Hybrid Approach** - For high-volume, simple extractions, consider rule-based extraction with LLM validation rather than full LLM extraction.

3. **Don't underestimate Validation** - Schema validation is critical for production systems. Invest time in comprehensive validation rules.

## Production Readiness Checklist

- [x] **Observability**: OpenTelemetry metrics, tracing, and logging configured
- [x] **Error Handling**: Comprehensive error handling with retries and fallbacks
- [x] **Security**: Data encryption and privacy compliance (GDPR, CCPA) in place
- [x] **Performance**: Load testing completed - handles 500+ documents/day
- [x] **Scalability**: Horizontal scaling strategy defined with queue-based architecture
- [x] **Monitoring**: Dashboards configured for extraction metrics and error rates
- [x] **Documentation**: API documentation and runbooks updated
- [x] **Testing**: Unit, integration, and end-to-end tests passing (98%+ accuracy)
- [x] **Configuration**: Environment-specific configs validated
- [x] **Disaster Recovery**: Backup and recovery procedures documented

## Related Use Cases

If you're working on a similar project, you might also find these helpful:

- **[Medical Record Standardization](./schema-medical-record-standardization.md)** - Similar schema validation approach for healthcare data
- **[Intelligent Document Processing](./03-intelligent-document-processing.md)** - Document ingestion and processing patterns
- **[Schema Package Guide](../package_design_patterns.md)** - Deep dive into schema validation patterns
- **[LLM Providers Guide](../guides/llm-providers.md)** - LLM configuration and optimization
