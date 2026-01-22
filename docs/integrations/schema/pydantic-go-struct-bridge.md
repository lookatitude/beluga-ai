# Pydantic/Go Struct Bridge

Welcome, colleague! In this integration guide, we're going to create a bridge between Pydantic models (Python) and Go structs for Beluga AI. This enables seamless data exchange between Python and Go services.

## What you will build

You will create a conversion system that translates between Pydantic models and Go structs, enabling interoperability between Python and Go services in your Beluga AI application.

## Learning Objectives

- ✅ Convert Pydantic models to Go structs
- ✅ Convert Go structs to Pydantic-compatible JSON
- ✅ Handle type conversions and validation
- ✅ Understand cross-language data exchange patterns

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Python 3.8+ (for Pydantic side)
- Understanding of JSON Schema

## Step 1: Define Go Struct

Create a Go struct that matches your Pydantic model:
```go
package main

import (
    "encoding/json"
    "fmt"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/schema"
)

// Go struct matching Pydantic model
type MessageData struct {
    Role      string            `json:"role"`
    Content   string            `json:"content"`
    Timestamp time.Time         `json:"timestamp"`
    Metadata  map[string]string `json:"metadata,omitempty"`
}

// Convert to Beluga AI Message
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
```

## Step 2: JSON Schema Generation

Generate JSON Schema from Go struct:
```go
import (
    "github.com/invopop/jsonschema"
)
go
func GenerateJSONSchema(v interface{}) ([]byte, error) {
    r := new(jsonschema.Reflector)
    r.ExpandedStruct = true
    
    schema := r.Reflect(v)
    return json.MarshalIndent(schema, "", "  ")
}

func main() {
    schema, err := GenerateJSONSchema(&MessageData{})
    if err != nil {
        log.Fatalf("Failed to generate schema: %v", err)
    }

    
    fmt.Println(string(schema))
}
```

## Step 3: Pydantic Model (Python Side)

Create the corresponding Pydantic model:
```python
from pydantic import BaseModel, Field
from datetime import datetime
from typing import Optional, Dict



class MessageData(BaseModel):
    role: str = Field(..., description="Message role")
    content: str = Field(..., description="Message content")
    timestamp: datetime = Field(default_factory=datetime.now)
    metadata: Optional[Dict[str, str]] = None
    
    class Config:
        json_schema_extra = {
            "example": {
                "role": "human",
                "content": "Hello, world!",
                "timestamp": "2024-01-01T00:00:00Z",
                "metadata": {"source": "api"}
            }
        }
```

## Step 4: Conversion Functions

Create conversion utilities:
```go
package main

import (
    "encoding/json"
    "fmt"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/schema"
)

// FromPydanticJSON converts Pydantic JSON to Go struct
func FromPydanticJSON(jsonData []byte) (*MessageData, error) {
    var msg MessageData
    if err := json.Unmarshal(jsonData, &msg); err != nil {
        return nil, fmt.Errorf("failed to unmarshal: %w", err)
    }
    
    // Set default timestamp if missing
    if msg.Timestamp.IsZero() {
        msg.Timestamp = time.Now()
    }
    
    return &msg, nil
}

// ToPydanticJSON converts Go struct to Pydantic-compatible JSON
func (m *MessageData) ToPydanticJSON() ([]byte, error) {
    return json.Marshal(m)
}

// FromMessage converts Beluga AI Message to Go struct
func FromMessage(msg schema.Message) *MessageData {
    return &MessageData{
        Role:      string(msg.GetRole()),
        Content:   msg.GetContent(),
        Timestamp: time.Now(),
    }
}
```

## Step 5: Complete Example

Here's a complete example with error handling:
```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/schema"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type PydanticBridge struct {
    tracer trace.Tracer
}

func NewPydanticBridge() *PydanticBridge {
    return &PydanticBridge{
        tracer: otel.Tracer("beluga.schema.pydantic_bridge"),
    }
}

func (b *PydanticBridge) ConvertFromPydantic(ctx context.Context, jsonData []byte) (schema.Message, error) {
    ctx, span := b.tracer.Start(ctx, "bridge.ConvertFromPydantic")
    defer span.End()

    msgData, err := FromPydanticJSON(jsonData)
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("conversion failed: %w", err)
    }

    msg := msgData.ToMessage()
    span.SetAttributes(
        attribute.String("message.role", string(msg.GetRole())),
        attribute.Int("message.length", len(msg.GetContent())),
    )

    return msg, nil
}

func (b *PydanticBridge) ConvertToPydantic(ctx context.Context, msg schema.Message) ([]byte, error) {
    ctx, span := b.tracer.Start(ctx, "bridge.ConvertToPydantic")
    defer span.End()

    msgData := FromMessage(msg)
    jsonData, err := msgData.ToPydanticJSON()
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("conversion failed: %w", err)
    }

    span.SetAttributes(
        attribute.Int("json.size", len(jsonData)),
    )

    return jsonData, nil
}

func main() {
    ctx := context.Background()
    bridge := NewPydanticBridge()

    // Example: Convert from Pydantic JSON
    pydanticJSON := []byte(`{
        "role": "human",
        "content": "Hello from Python!",
        "timestamp": "2024-01-01T00:00:00Z"
    }`)

    msg, err := bridge.ConvertFromPydantic(ctx, pydanticJSON)
    if err != nil {
        log.Fatalf("Conversion failed: %v", err)
    }

    fmt.Printf("Converted message: %s\n", msg.GetContent())

    // Example: Convert to Pydantic JSON
    belugaMsg := schema.NewAIMessage("Hello from Go!")
    jsonData, err := bridge.ConvertToPydantic(ctx, belugaMsg)
    if err != nil {
        log.Fatalf("Conversion failed: %v", err)
    }


    fmt.Printf("Pydantic JSON: %s\n", string(jsonData))
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `Timezone` | Timezone for timestamp conversion | `UTC` | No |
| `ValidateOnConvert` | Validate data during conversion | `true` | No |
| `StrictMode` | Fail on unknown fields | `false` | No |

## Common Issues

### "Timestamp parsing error"

**Problem**: Timestamp format mismatch between Python and Go.

**Solution**: Use RFC3339 format consistently:const RFC3339 = "2006-01-02T15:04:05Z07:00"
timestamp, err := time.Parse(RFC3339, timestampStr)
```

### "Type conversion error"

**Problem**: Type mismatches between Pydantic and Go.

**Solution**: Use custom unmarshalers for complex types:func (m *MessageData) UnmarshalJSON(data []byte) error {
    // Custom unmarshaling logic
}
```

## Production Considerations

When using Pydantic/Go bridge in production:

- **Performance**: Cache conversion results when possible
- **Error Handling**: Provide clear error messages for conversion failures
- **Validation**: Validate data on both sides
- **Monitoring**: Track conversion success/failure rates

## Next Steps

Congratulations! You've created a Pydantic/Go struct bridge. Next, learn how to:

- **[JSON Schema Validation](./json-schema-validation.md)** - Validate data structures
- **[Schema Package Documentation](../../api/packages/schema.md)** - Deep dive into schema package
- **[Use Case: Multi-Language Services](../../use-cases/)** - Cross-language integration patterns

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
