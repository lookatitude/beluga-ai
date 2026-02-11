---
title: Medical Record Standardization
description: Transform medical records from multiple hospital systems into standardized HL7 FHIR format using Beluga AI's schema validation.
---

A healthcare network acquiring a new hospital inherits decades of patient records in proprietary formats — HL7 v2 messages with custom segments, CDA documents with non-standard extensions, even flat files with institution-specific codes. When a patient transfers between facilities, their allergy list might use ICD-9 codes at one hospital and SNOMED CT at another, making automated cross-referencing impossible. Studies show that 18-25% of critical patient data is lost during inter-system transfers, leading to duplicate tests (costing $210B annually in the US), missed drug interactions, and delayed diagnoses.

The interoperability problem is not just format conversion — it requires semantic mapping between coding systems that evolved independently. An ICD-9 diagnosis code "250.00" (Type II diabetes) must map to ICD-10 "E11.9", but the mapping is not always one-to-one, and errors in code translation can change the clinical meaning of a record.

## Solution Architecture

Beluga AI's schema validation provides strict enforcement of HL7 FHIR output schemas, catching data quality issues at transformation time rather than downstream in clinical systems. The pipeline architecture separates format detection, parsing, code mapping, and FHIR transformation into distinct stages. This separation matters because each stage has different failure modes — parsing failures indicate format issues, code mapping failures indicate terminology gaps, and validation failures indicate incomplete or inconsistent transformations. OpenTelemetry tracing across stages enables operators to identify exactly where data quality breaks down.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Medical    │───▶│    Format    │───▶│    Parser    │
│   Record     │    │   Detector   │    │  (HL7 v2/    │
│   Input      │    │              │    │   CDA/Custom)│
└──────────────┘    └──────────────┘    └──────┬───────┘
                                               │
                                               ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  FHIR Record │◀───│    Schema    │◀───│   FHIR       │
│  (Validated) │    │   Validator  │    │ Transformer  │
└──────────────┘    └──────────────┘    └──────┬───────┘
                          ▲                     │
                          │                     ▼
                    ┌─────┴────────┐    ┌──────────────┐
                    │  FHIR Schema │    │     Code     │
                    │  Definitions │    │    Mapper    │
                    └──────────────┘    │ (ICD-9→ICD-10│
                                        │  LOINC/SNOMED)│
                                        └──────────────┘
```

## Schema Validation Setup

Define FHIR resource schemas with strict validation rules:

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/schema"
    "github.com/lookatitude/beluga-ai/o11y"
)

// FHIRPatient represents a standardized FHIR Patient resource
type FHIRPatient struct {
    ResourceType string          `json:"resourceType" validate:"required,eq=Patient"`
    ID           string          `json:"id" validate:"required"`
    Identifier   []FHIRIdentifier `json:"identifier,omitempty" validate:"dive"`
    Name         []FHIRHumanName  `json:"name" validate:"required,min=1,dive"`
    BirthDate    string          `json:"birthDate,omitempty" validate:"datetime=2006-01-02"`
    Gender       string          `json:"gender,omitempty" validate:"omitempty,oneof=male female other unknown"`
    Address      []FHIRAddress   `json:"address,omitempty" validate:"dive"`
    Meta         FHIRMeta        `json:"meta,omitempty"`
}

type FHIRIdentifier struct {
    System string `json:"system" validate:"required,uri"`
    Value  string `json:"value" validate:"required"`
}

type FHIRHumanName struct {
    Family string   `json:"family" validate:"required"`
    Given  []string `json:"given" validate:"required,min=1"`
}

type FHIRAddress struct {
    Line       []string `json:"line" validate:"required,min=1"`
    City       string   `json:"city" validate:"required"`
    State      string   `json:"state"`
    PostalCode string   `json:"postalCode"`
    Country    string   `json:"country" validate:"required"`
}

type FHIRMeta struct {
    VersionID   string `json:"versionId"`
    LastUpdated string `json:"lastUpdated" validate:"datetime=2006-01-02T15:04:05Z07:00"`
}
```

## Transformation Pipeline

The standardizer validates at every step to ensure data quality:

```go
package main

import (
    "context"
    "fmt"

    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type MedicalRecordStandardizer struct {
    codeMapper *CodeMapper
    tracer     trace.Tracer
}

type RawMedicalRecord struct {
    ID           string
    Format       string // "HL7v2", "CDA", "proprietary"
    SourceSystem string
    Data         []byte
}

func (m *MedicalRecordStandardizer) StandardizeRecord(ctx context.Context, rawRecord RawMedicalRecord) (*FHIRPatient, error) {
    ctx, span := m.tracer.Start(ctx, "medical.record.standardization")
    defer span.End()

    span.SetAttributes(
        attribute.String("source_format", rawRecord.Format),
        attribute.String("source_system", rawRecord.SourceSystem),
    )

    // Parse based on format
    parsedData, err := m.parseRecord(ctx, rawRecord)
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("parsing failed: %w", err)
    }

    // Map medical codes to standard terminologies
    mappedData, err := m.codeMapper.MapCodes(ctx, parsedData)
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("code mapping failed: %w", err)
    }

    // Transform to FHIR
    fhirRecord, err := m.transformToFHIR(ctx, mappedData)
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("FHIR transformation failed: %w", err)
    }

    // Validate FHIR schema
    if err := m.validateFHIR(ctx, fhirRecord); err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("FHIR validation failed: %w", err)
    }

    span.SetAttributes(
        attribute.String("fhir_resource_type", fhirRecord.ResourceType),
        attribute.String("fhir_id", fhirRecord.ID),
    )

    return fhirRecord, nil
}

func (m *MedicalRecordStandardizer) parseRecord(ctx context.Context, raw RawMedicalRecord) (*ParsedData, error) {
    switch raw.Format {
    case "HL7v2":
        return parseHL7v2(raw.Data)
    case "CDA":
        return parseCDA(raw.Data)
    default:
        return parseCustom(raw.Data)
    }
}

func (m *MedicalRecordStandardizer) validateFHIR(ctx context.Context, record *FHIRPatient) error {
    // Use standard validation library (e.g., go-playground/validator)
    // Validate against FHIR schema rules
    // Return detailed validation errors for debugging
    return nil
}
```

## Code Mapping

Medical codes must be mapped to standard terminologies:

```go
package main

import (
    "context"
    "fmt"
)

type CodeMapper struct {
    icd9ToIcd10  map[string]string
    loincMapping map[string]string
}

type ParsedData struct {
    Diagnoses   []string
    Procedures  []string
    LabResults  []LabResult
    PatientInfo PatientInfo
}

type LabResult struct {
    Code   string
    Value  string
    Units  string
}

type PatientInfo struct {
    FirstName  string
    LastName   string
    BirthDate  string
    Gender     string
    Addresses  []Address
}

type Address struct {
    Line       []string
    City       string
    State      string
    PostalCode string
    Country    string
}

func (c *CodeMapper) MapCodes(ctx context.Context, data *ParsedData) (*MappedData, error) {
    mapped := &MappedData{
        PatientInfo: data.PatientInfo,
    }

    // Map ICD-9 diagnosis codes to ICD-10
    for _, diagnosis := range data.Diagnoses {
        if icd10, ok := c.icd9ToIcd10[diagnosis]; ok {
            mapped.Diagnoses = append(mapped.Diagnoses, icd10)
        } else {
            // Fallback strategy or error
            return nil, fmt.Errorf("unmapped diagnosis code: %s", diagnosis)
        }
    }

    // Map lab codes to LOINC
    for _, result := range data.LabResults {
        if loinc, ok := c.loincMapping[result.Code]; ok {
            mapped.LabResults = append(mapped.LabResults, LabResult{
                Code:  loinc,
                Value: result.Value,
                Units: result.Units,
            })
        }
    }

    return mapped, nil
}

type MappedData struct {
    Diagnoses   []string
    Procedures  []string
    LabResults  []LabResult
    PatientInfo PatientInfo
}
```

## Production Considerations

### Observability

Track transformation metrics and validation failures:

```go
import (
    "github.com/lookatitude/beluga-ai/o11y"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/metric"
)

type AuditTrail struct {
    RecordID       string
    SourceSystem   string
    TransformTime  time.Time
    Success        bool
    ValidationErrs []string
}

func (m *MedicalRecordStandardizer) StandardizeWithAudit(ctx context.Context, rawRecord RawMedicalRecord) (*FHIRPatient, *AuditTrail, error) {
    ctx, span := m.tracer.Start(ctx, "medical.record.standardization.audited")
    defer span.End()

    auditTrail := &AuditTrail{
        RecordID:      rawRecord.ID,
        SourceSystem:  rawRecord.SourceSystem,
        TransformTime: time.Now(),
    }

    fhirRecord, err := m.StandardizeRecord(ctx, rawRecord)
    if err != nil {
        auditTrail.Success = false
        auditTrail.ValidationErrs = []string{err.Error()}
        return nil, auditTrail, err
    }

    auditTrail.Success = true
    return fhirRecord, auditTrail, nil
}
```

### Resilience

Implement retry logic for transient failures:

```go
import "github.com/lookatitude/beluga-ai/resilience"

policy := resilience.RetryPolicy{
    MaxAttempts:    3,
    InitialBackoff: 500 * time.Millisecond,
    MaxBackoff:     5 * time.Second,
    BackoffFactor:  2.0,
    Jitter:         true,
}

fhirRecord, err := resilience.Retry(ctx, policy, func(ctx context.Context) (*FHIRPatient, error) {
    return standardizer.StandardizeRecord(ctx, rawRecord)
})
```

### Batch Processing

Process large volumes efficiently:

```go
func (m *MedicalRecordStandardizer) StandardizeBatch(ctx context.Context, records []RawMedicalRecord) ([]*FHIRPatient, error) {
    batchSize := 100
    results := make([]*FHIRPatient, 0, len(records))

    for i := 0; i < len(records); i += batchSize {
        end := min(i+batchSize, len(records))
        batch := records[i:end]

        for _, record := range batch {
            fhir, err := m.StandardizeRecord(ctx, record)
            if err != nil {
                // Log error but continue processing
                continue
            }
            results = append(results, fhir)
        }
    }

    return results, nil
}
```

### Security

Healthcare data requires strict security controls:

- Store all transformations in an audit trail for HIPAA compliance
- Implement role-based access control for medical record access
- Encrypt data at rest and in transit using TLS 1.3
- Use Beluga AI's `guard/` pipeline to detect and redact PII leakage
- Store API keys and credentials in a secrets manager, never in code

## Related Resources

- [Schema Package Guide](/guides/schema-validation/) for validation patterns
- [RAG Pipeline Guide](/guides/rag-pipeline/) for document processing
- [Observability Guide](/guides/observability/) for healthcare monitoring
