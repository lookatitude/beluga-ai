---
title: Pydantic / Go Struct Bridge
description: "Bridge data exchange between Python Pydantic models and Go structs using JSON Schema for cross-language Beluga AI integration."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Pydantic Go bridge, cross-language AI, JSON Schema, Beluga AI, Python Go interop, data serialization, struct validation"
---

Many AI teams run Python for model training and evaluation while using Go for production serving. When these services need to exchange messages, tool results, or evaluation data, mismatched serialization causes silent data corruption and hard-to-debug failures. This bridge pattern establishes a shared contract between Pydantic models and Go structs using JSON Schema as the source of truth, enabling reliable interoperability between Python and Go services.

## Overview

The bridge pattern works in two directions:

1. **Python to Go**: Pydantic models serialize to JSON, which Go deserializes into structs that convert to Beluga AI `schema.Message` types.
2. **Go to Python**: Beluga AI messages convert to Go structs, serialize to JSON, and are consumed by Pydantic models on the Python side.

JSON Schema serves as the shared contract between both sides.

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed (`github.com/lookatitude/beluga-ai`)
- Python 3.8+ with Pydantic (for the Python side)
- Familiarity with JSON serialization in both languages

## Installation

Go side:

```bash
go get github.com/lookatitude/beluga-ai
go get github.com/invopop/jsonschema
```

Python side:

```bash
pip install pydantic
```

## Configuration

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `Timezone` | Timezone for timestamp conversion | `UTC` | No |
| `ValidateOnConvert` | Validate data during conversion | `true` | No |
| `StrictMode` | Reject unknown JSON fields | `false` | No |

## Usage

### Define the Go Struct

Create a Go struct that mirrors the Pydantic model and provides conversion to Beluga AI message types.

```go
package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/schema"
)

// MessageData is the Go representation of the shared message format.
// JSON tags align with the Pydantic model field names.
type MessageData struct {
	Role      string            `json:"role"`
	Content   string            `json:"content"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ToMessage converts the bridge struct to a Beluga AI Message.
func (m *MessageData) ToMessage() schema.Message {
	switch m.Role {
	case "system":
		return schema.NewSystemMessage(m.Content)
	case "human":
		return schema.NewHumanMessage(m.Content)
	case "ai":
		return schema.NewAIMessage(m.Content)
	default:
		return schema.NewHumanMessage(m.Content)
	}
}

// FromMessage converts a Beluga AI Message to a bridge struct.
func FromMessage(msg schema.Message) *MessageData {
	// Extract text content from the message parts
	var content string
	for _, part := range msg.GetContent() {
		if tp, ok := part.(schema.TextPart); ok {
			content = tp.Text
			break
		}
	}

	return &MessageData{
		Role:      string(msg.GetRole()),
		Content:   content,
		Timestamp: time.Now(),
	}
}
```

### Define the Pydantic Model (Python Side)

Create the corresponding Pydantic model with matching field names.

```python
from pydantic import BaseModel, Field
from datetime import datetime
from typing import Optional, Dict


class MessageData(BaseModel):
    role: str = Field(..., description="Message role: system, human, ai, or tool")
    content: str = Field(..., description="Message text content")
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    metadata: Optional[Dict[str, str]] = None

    model_config = {
        "json_schema_extra": {
            "example": {
                "role": "human",
                "content": "Hello from Python!",
                "timestamp": "2025-01-01T00:00:00Z",
                "metadata": {"source": "api"},
            }
        }
    }
```

### Conversion Functions

Implement bidirectional JSON conversion on the Go side.

```go
package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/schema"
)

// FromPydanticJSON deserializes Pydantic JSON into a Go bridge struct.
func FromPydanticJSON(jsonData []byte) (*MessageData, error) {
	var msg MessageData
	if err := json.Unmarshal(jsonData, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal pydantic JSON: %w", err)
	}

	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	return &msg, nil
}

// ToPydanticJSON serializes a Beluga AI Message as Pydantic-compatible JSON.
func ToPydanticJSON(msg schema.Message) ([]byte, error) {
	data := FromMessage(msg)
	return json.Marshal(data)
}
```

### JSON Schema Generation

Generate a shared JSON Schema from the Go struct to serve as the contract between services.

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/invopop/jsonschema"
)

func main() {
	r := new(jsonschema.Reflector)
	r.ExpandedStruct = true

	s := r.Reflect(&MessageData{})
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		log.Fatalf("generate schema: %v", err)
	}

	fmt.Println(string(data))
}
```

The generated schema can be shared with the Python team and used with the [JSON Schema Validation](/docs/integrations/json-schema-validation/) integration to validate payloads on both sides.

### Complete Bridge Example

A full round-trip example demonstrating both directions.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/schema"
)

func main() {
	ctx := context.Background()
	_ = ctx // Available for instrumented operations

	// Direction 1: Python (Pydantic) -> Go (Beluga AI)
	pydanticJSON := []byte(`{
		"role": "human",
		"content": "Hello from Python!",
		"timestamp": "2025-01-01T00:00:00Z"
	}`)

	msgData, err := FromPydanticJSON(pydanticJSON)
	if err != nil {
		log.Fatalf("conversion failed: %v", err)
	}

	msg := msgData.ToMessage()
	fmt.Printf("Received from Python: role=%s\n", msg.GetRole())

	// Direction 2: Go (Beluga AI) -> Python (Pydantic)
	aiMsg := schema.NewAIMessage("Hello from Go!")
	jsonData, err := ToPydanticJSON(aiMsg)
	if err != nil {
		log.Fatalf("serialization failed: %v", err)
	}

	fmt.Printf("Sending to Python: %s\n", string(jsonData))

	// Verify round-trip fidelity
	var roundTrip MessageData
	if err := json.Unmarshal(jsonData, &roundTrip); err != nil {
		log.Fatalf("round-trip failed: %v", err)
	}

	fmt.Printf("Round-trip role: %s, content: %s\n", roundTrip.Role, roundTrip.Content)
}
```

## Advanced Topics

### Instrumented Bridge with OTel

Add OpenTelemetry tracing to conversion operations for cross-service observability.

```go
package main

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/schema"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Bridge provides instrumented bidirectional conversion.
type Bridge struct {
	tracer trace.Tracer
}

// NewBridge creates an instrumented bridge.
func NewBridge() *Bridge {
	return &Bridge{
		tracer: otel.Tracer("beluga.schema.bridge"),
	}
}

// ConvertFromPydantic deserializes Pydantic JSON into a Beluga AI message.
func (b *Bridge) ConvertFromPydantic(ctx context.Context, jsonData []byte) (schema.Message, error) {
	_, span := b.tracer.Start(ctx, "bridge.from_pydantic",
		trace.WithAttributes(
			attribute.Int("json.size", len(jsonData)),
		),
	)
	defer span.End()

	msgData, err := FromPydanticJSON(jsonData)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("conversion failed: %w", err)
	}

	msg := msgData.ToMessage()
	span.SetAttributes(
		attribute.String("message.role", string(msg.GetRole())),
	)

	return msg, nil
}

// ConvertToPydantic serializes a Beluga AI message as Pydantic-compatible JSON.
func (b *Bridge) ConvertToPydantic(ctx context.Context, msg schema.Message) ([]byte, error) {
	_, span := b.tracer.Start(ctx, "bridge.to_pydantic",
		trace.WithAttributes(
			attribute.String("message.role", string(msg.GetRole())),
		),
	)
	defer span.End()

	data, err := ToPydanticJSON(msg)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("serialization failed: %w", err)
	}

	span.SetAttributes(attribute.Int("json.size", len(data)))
	return data, nil
}
```

### Custom Type Handling

For complex types that do not have a direct JSON mapping, implement custom marshalers.

```go
// CustomMessageData handles non-standard timestamp formats from Python.
type CustomMessageData struct {
	MessageData
}

// UnmarshalJSON provides custom deserialization for non-standard formats.
func (m *CustomMessageData) UnmarshalJSON(data []byte) error {
	// Attempt standard parsing first
	type Alias MessageData
	aux := &struct {
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Alias: (*Alias)(&m.MessageData),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	// Try multiple timestamp formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, aux.Timestamp); err == nil {
			m.Timestamp = t
			return nil
		}
	}

	return fmt.Errorf("unsupported timestamp format: %s", aux.Timestamp)
}
```

### Batch Conversion

Convert slices of messages for bulk operations between services.

```go
// ConvertBatchFromPydantic converts multiple Pydantic JSON messages to Beluga AI messages.
func ConvertBatchFromPydantic(jsonMessages [][]byte) ([]schema.Message, error) {
	messages := make([]schema.Message, 0, len(jsonMessages))

	for i, jsonData := range jsonMessages {
		msgData, err := FromPydanticJSON(jsonData)
		if err != nil {
			return nil, fmt.Errorf("message %d: %w", i, err)
		}
		messages = append(messages, msgData.ToMessage())
	}

	return messages, nil
}
```

## Troubleshooting

### Timestamp parsing error

Python and Go use different default timestamp formats. Ensure both sides use RFC 3339 (`2006-01-02T15:04:05Z07:00` in Go, ISO 8601 in Python). Use the `CustomMessageData` pattern above if you need to handle multiple formats.

### Type conversion error

Python's dynamic typing can produce JSON values that do not match Go struct field types. Use `json.Number` for numeric fields that may arrive as integers or floats, and use custom unmarshalers for complex mappings.

### Unknown field errors in strict mode

When `StrictMode` is enabled, unknown JSON fields cause unmarshaling to fail. Either add `json:"-"` tags for fields that should be ignored, or use `json.Decoder` with `DisallowUnknownFields()`.

## Production Considerations

- **Schema as contract**: Generate JSON Schema from the Go struct and share it with the Python team. Both sides validate against the same schema.
- **Versioning**: Version your message format and include a `version` field in the JSON payload to support backward-compatible evolution.
- **Performance**: Cache `MessageData` instances when converting large batches. Avoid repeated allocations for high-throughput pipelines.
- **Validation**: Validate incoming JSON on both sides. Use the [JSON Schema Validation](/docs/integrations/json-schema-validation/) integration for schema-based validation.

## Related Resources

- [JSON Schema Validation](/docs/integrations/json-schema-validation/) -- Validate data structures with JSON Schema
- [Schema Package](/docs/api-reference/schema/) -- Core message and document types
- [A2A Protocol](/docs/guides/a2a/) -- Agent-to-agent communication across services
