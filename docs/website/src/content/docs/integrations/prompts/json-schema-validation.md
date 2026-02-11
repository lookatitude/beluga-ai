---
title: JSON Schema Validation
description: Validate Beluga AI message and document structures using JSON Schema with the gojsonschema library.
---

Malformed input is the most common cause of unexpected LLM behavior. A missing field, wrong type, or truncated message can produce misleading model responses that are expensive to debug. JSON Schema validation catches these issues at the application boundary -- before data reaches the LLM -- providing clear error messages instead of silent failures. This guide demonstrates how to validate Beluga AI message and document structures against JSON Schema definitions using the `gojsonschema` library.

## Overview

JSON Schema validation is useful for verifying that data entering or leaving your Beluga AI application conforms to expected structures. Common use cases include validating API request payloads before converting them to Beluga AI messages, verifying document metadata before ingestion into a RAG pipeline, and enforcing contract compliance in multi-service architectures.

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed (`github.com/lookatitude/beluga-ai`)
- Familiarity with JSON Schema format (draft-07 or later)

## Installation

Install the JSON Schema validation library:

```bash
go get github.com/xeipuuv/gojsonschema
```

## Configuration

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `Schema` | JSON Schema definition string | -- | Yes |
| `Strict` | Reject unknown properties | `false` | No |
| `ValidateRequired` | Enforce required field constraints | `true` | No |

## Usage

### Basic Message Validation

Define a JSON Schema for message structures and validate serialized messages against it.

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/xeipuuv/gojsonschema"
)

// messageSchema defines the expected structure for serialized messages.
const messageSchema = `{
  "type": "object",
  "properties": {
    "role": {
      "type": "string",
      "enum": ["system", "human", "ai", "tool"]
    },
    "parts": {
      "type": "array",
      "minItems": 1
    }
  },
  "required": ["role", "parts"]
}`

// messageDTO is a serialization-friendly representation of a message.
type messageDTO struct {
	Role  string `json:"role"`
	Parts []any  `json:"parts"`
}

func main() {
	// Create a message using the schema package
	msg := schema.NewHumanMessage("What is the capital of France?")

	// Convert to a serializable DTO
	dto := messageDTO{
		Role:  string(msg.GetRole()),
		Parts: toAnySlice(msg.GetContent()),
	}

	msgJSON, err := json.Marshal(dto)
	if err != nil {
		log.Fatalf("marshal failed: %v", err)
	}

	// Validate against schema
	schemaLoader := gojsonschema.NewStringLoader(messageSchema)
	documentLoader := gojsonschema.NewBytesLoader(msgJSON)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		log.Fatalf("validation error: %v", err)
	}

	if result.Valid() {
		fmt.Println("Message is valid")
	} else {
		fmt.Println("Message is invalid:")
		for _, desc := range result.Errors() {
			fmt.Printf("  - %s\n", desc)
		}
	}
}

func toAnySlice(parts []schema.ContentPart) []any {
	result := make([]any, len(parts))
	for i, p := range parts {
		result[i] = p
	}
	return result
}
```

### Reusable Validator Wrapper

Create a reusable validator that caches the compiled schema for repeated use.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

// SchemaValidator validates JSON documents against a pre-loaded schema.
type SchemaValidator struct {
	schemaLoader gojsonschema.JSONLoader
}

// NewSchemaValidator creates a validator from a JSON Schema string.
// It returns an error if the schema itself is invalid JSON.
func NewSchemaValidator(schemaJSON string) (*SchemaValidator, error) {
	// Validate schema JSON is well-formed
	var probe map[string]any
	if err := json.Unmarshal([]byte(schemaJSON), &probe); err != nil {
		return nil, fmt.Errorf("invalid schema JSON: %w", err)
	}

	return &SchemaValidator{
		schemaLoader: gojsonschema.NewStringLoader(schemaJSON),
	}, nil
}

// Validate checks a JSON-serializable value against the schema.
func (v *SchemaValidator) Validate(ctx context.Context, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}

	documentLoader := gojsonschema.NewBytesLoader(jsonData)
	result, err := gojsonschema.Validate(v.schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if !result.Valid() {
		var errors []string
		for _, desc := range result.Errors() {
			errors = append(errors, desc.String())
		}
		return fmt.Errorf("validation failed: %v", errors)
	}

	return nil
}
```

### Document Schema Validation

Validate document structures before ingesting them into a RAG pipeline.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/schema"
)

const documentSchema = `{
  "type": "object",
  "properties": {
    "page_content": {
      "type": "string",
      "minLength": 1
    },
    "metadata": {
      "type": "object"
    }
  },
  "required": ["page_content"]
}`

// documentDTO is a serialization-friendly representation of a document.
type documentDTO struct {
	PageContent string         `json:"page_content"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

func main() {
	ctx := context.Background()

	validator, err := NewSchemaValidator(documentSchema)
	if err != nil {
		log.Fatalf("create validator: %v", err)
	}

	doc := schema.Document{
		PageContent: "Beluga AI is a Go-native agentic framework.",
		Metadata: map[string]any{
			"source": "docs",
		},
	}

	dto := documentDTO{
		PageContent: doc.PageContent,
		Metadata:    doc.Metadata,
	}

	if err := validator.Validate(ctx, dto); err != nil {
		fmt.Printf("Document invalid: %v\n", err)
		return
	}

	fmt.Println("Document is valid")
}
```

## Advanced Topics

### Instrumented Validation with OTel

Add OpenTelemetry tracing to track validation success rates and latency.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedValidator adds OTel tracing to schema validation.
type InstrumentedValidator struct {
	schemaLoader gojsonschema.JSONLoader
	tracer       trace.Tracer
}

// NewInstrumentedValidator creates a traced validator.
func NewInstrumentedValidator(schemaJSON string) (*InstrumentedValidator, error) {
	var probe map[string]any
	if err := json.Unmarshal([]byte(schemaJSON), &probe); err != nil {
		return nil, fmt.Errorf("invalid schema JSON: %w", err)
	}

	return &InstrumentedValidator{
		schemaLoader: gojsonschema.NewStringLoader(schemaJSON),
		tracer:       otel.Tracer("beluga.schema.validator"),
	}, nil
}

// Validate checks data with tracing.
func (v *InstrumentedValidator) Validate(ctx context.Context, data any) error {
	ctx, span := v.tracer.Start(ctx, "schema.validate")
	defer span.End()

	jsonData, err := json.Marshal(data)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("marshal data: %w", err)
	}

	documentLoader := gojsonschema.NewBytesLoader(jsonData)
	result, err := gojsonschema.Validate(v.schemaLoader, documentLoader)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("validation error: %w", err)
	}

	if !result.Valid() {
		var errors []string
		for _, desc := range result.Errors() {
			errors = append(errors, desc.String())
		}
		err := fmt.Errorf("validation failed: %v", errors)
		span.RecordError(err)
		span.SetAttributes(attribute.StringSlice("validation.errors", errors))
		return err
	}

	span.SetAttributes(attribute.Bool("validation.valid", true))
	return nil
}
```

### Pipeline Integration

Use schema validation as a guard in the safety pipeline to reject malformed input before it reaches the LLM.

```go
// SchemaGuard validates input content against a JSON Schema before allowing
// it through the guard pipeline. This is useful for structured input validation.
type SchemaGuard struct {
	validator *SchemaValidator
}

func (g *SchemaGuard) Name() string { return "schema-validator" }

func (g *SchemaGuard) Validate(ctx context.Context, input guard.GuardInput) (guard.GuardResult, error) {
	if err := g.validator.Validate(ctx, input.Content); err != nil {
		return guard.GuardResult{
			Allowed:   false,
			Reason:    err.Error(),
			GuardName: g.Name(),
		}, nil
	}

	return guard.GuardResult{
		Allowed:   true,
		GuardName: g.Name(),
	}, nil
}
```

## Troubleshooting

### Validation error: invalid character

The JSON Schema string contains invalid JSON. Validate your schema before passing it to the constructor:

```go
var probe map[string]any
if err := json.Unmarshal([]byte(schemaJSON), &probe); err != nil {
    return fmt.Errorf("invalid schema JSON: %w", err)
}
```

### Validation failed: missing required field

Required fields declared in the schema are missing from the data. Ensure all required fields are populated in your DTOs before validation.

## Production Considerations

- **Schema caching**: Create `SchemaValidator` instances once and reuse them. Avoid re-parsing schemas on every request.
- **Error messages**: Aggregate validation errors into structured error types for better API responses.
- **Monitoring**: Track validation failure rates to detect data quality issues early.
- **Schema versioning**: Version your schemas and validate against the correct version for each API version.

## Related Resources

- [Go Struct Bridge](/integrations/go-struct-bridge/) -- Cross-language data exchange with Pydantic
- [Schema Package](/api-reference/schema/) -- Core message and document types
- [Guard Pipeline](/guides/safety-guards/) -- Safety pipeline integration
