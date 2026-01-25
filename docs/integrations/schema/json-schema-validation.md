# JSON Schema Validation

Welcome, colleague! In this integration guide, we're going to integrate JSON Schema validation with Beluga AI's schema package. We'll keep it quick and functional so you can see results immediately.

## What you will build

You will create a validation system that uses JSON Schema to validate data structures in your Beluga AI application. This integration allows you to ensure data integrity and catch errors early in your pipeline.

## Learning Objectives

- ✅ Configure JSON Schema validation with Beluga AI
- ✅ Validate message structures and document schemas
- ✅ Handle validation errors gracefully
- ✅ Understand schema validation best practices

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Understanding of JSON Schema format

## Step 1: Setup and Installation

Install the JSON Schema validation library:
bash
```bash
go get github.com/xeipuuv/gojsonschema
```

## Step 2: Basic Schema Validation

Create a simple validation example:
```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/schema"
    "github.com/xeipuuv/gojsonschema"
)

// Define a JSON Schema for messages
const messageSchema = `{
  "type": "object",
  "properties": {
    "role": {
      "type": "string",
      "enum": ["system", "human", "ai", "tool"]
    },
    "content": {
      "type": "string"
    }
  },
  "required": ["role", "content"]
}`

func main() {
    ctx := context.Background()

    // Create a message
    msg := schema.NewHumanMessage("Hello, world!")

    // Convert to JSON for validation
    msgJSON, err := json.Marshal(msg)
    if err != nil {
        log.Fatalf("Failed to marshal message: %v", err)
    }

    // Validate against schema
    schemaLoader := gojsonschema.NewStringLoader(messageSchema)
    documentLoader := gojsonschema.NewBytesLoader(msgJSON)

    result, err := gojsonschema.Validate(schemaLoader, documentLoader)
    if err != nil {
        log.Fatalf("Validation error: %v", err)
    }

    if result.Valid() {
        fmt.Println("Message is valid!")
    } else {
        fmt.Printf("Message is invalid. Errors:\n")
        for _, desc := range result.Errors() {
            fmt.Printf("- %s\n", desc)
        }
    }
}
```

### Verification

Run the example:
bash
```bash
go run main.go
```

You should see:Message is valid!
```

## Step 3: Schema Validator Wrapper

Create a reusable validator wrapper:
```go
package main

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/lookatitude/beluga-ai/pkg/schema"
    "github.com/xeipuuv/gojsonschema"
)

type SchemaValidator struct {
    schemaLoader gojsonschema.JSONLoader
}

func NewSchemaValidator(schemaJSON string) (*SchemaValidator, error) {
    loader := gojsonschema.NewStringLoader(schemaJSON)
    return &SchemaValidator{schemaLoader: loader}, nil
}

func (v *SchemaValidator) ValidateMessage(ctx context.Context, msg schema.Message) error {
    msgJSON, err := json.Marshal(msg)
    if err != nil {
        return fmt.Errorf("failed to marshal message: %w", err)
    }

    documentLoader := gojsonschema.NewBytesLoader(msgJSON)
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

func main() {
    ctx := context.Background()

    // Create validator
    validator, err := NewSchemaValidator(messageSchema)
    if err != nil {
        log.Fatalf("Failed to create validator: %v", err)
    }

    // Validate multiple messages
    messages := []schema.Message{
        schema.NewSystemMessage("You are a helpful assistant."),
        schema.NewHumanMessage("What is the capital of France?"),
    }

    for _, msg := range messages {
        if err := validator.ValidateMessage(ctx, msg); err != nil {
            fmt.Printf("Validation failed: %v\n", err)
        } else {
            fmt.Printf("Message validated: %s\n", msg.GetContent())
        }
    }
}
```

## Step 4: Document Schema Validation

Validate document structures:
```go
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

func (v *SchemaValidator) ValidateDocument(ctx context.Context, doc schema.Document) error {
    docJSON, err := json.Marshal(doc)
    if err != nil {
        return fmt.Errorf("failed to marshal document: %w", err)
    }

    documentLoader := gojsonschema.NewBytesLoader(docJSON)
    result, err := gojsonschema.Validate(v.schemaLoader, documentLoader)
    if err != nil {
        return fmt.Errorf("validation error: %w", err)
    }

    if !result.Valid() {
        var errors []string
        for _, desc := range result.Errors() {
            errors = append(errors, desc.String())
        }
        return fmt.Errorf("document validation failed: %v", errors)
    }


    return nil
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `Schema` | JSON Schema string | - | Yes |
| `Strict` | Fail on unknown properties | `false` | No |
| `ValidateRequired` | Validate required fields | `true` | No |

## Common Issues

### "Validation error: invalid character"

**Problem**: The JSON Schema string contains invalid JSON.

**Solution**: Validate your schema JSON before using it:var schemaMap map[string]interface\{\}
```text
go
go
if err := json.Unmarshal([]byte(schemaJSON), &schemaMap); err != nil {
    return fmt.Errorf("invalid schema JSON: %w", err)
}
```

### "Validation failed: missing required field"

**Problem**: Required fields are missing from the data.

**Solution**: Ensure all required fields are present in your messages or documents.

## Production Considerations

When using JSON Schema validation in production:

- **Performance**: Cache compiled schemas to avoid re-parsing
- **Error Handling**: Provide clear, actionable error messages
- **Monitoring**: Track validation failures for debugging
- **Security**: Validate schemas themselves before use

## Complete Example

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "sync"

    "github.com/lookatitude/beluga-ai/pkg/schema"
    "github.com/xeipuuv/gojsonschema"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type SchemaValidator struct {
    schemaLoader gojsonschema.JSONLoader
    tracer       trace.Tracer
    mu           sync.RWMutex
}

func NewSchemaValidator(schemaJSON string) (*SchemaValidator, error) {
    // Validate schema JSON first
    var schemaMap map[string]interface{}
    if err := json.Unmarshal([]byte(schemaJSON), &schemaMap); err != nil {
        return nil, fmt.Errorf("invalid schema JSON: %w", err)
    }

    return &SchemaValidator{
        schemaLoader: gojsonschema.NewStringLoader(schemaJSON),
        tracer:       otel.Tracer("beluga.schema.validator"),
    }, nil
}

func (v *SchemaValidator) ValidateMessage(ctx context.Context, msg schema.Message) error {
    ctx, span := v.tracer.Start(ctx, "validator.ValidateMessage",
        trace.WithAttributes(
            attribute.String("message.role", string(msg.GetRole())),
        ),
    )
    defer span.End()

    msgJSON, err := json.Marshal(msg)
    if err != nil {
        span.RecordError(err)
        return fmt.Errorf("failed to marshal message: %w", err)
    }

    documentLoader := gojsonschema.NewBytesLoader(msgJSON)
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

func main() {
    ctx := context.Background()

    validator, err := NewSchemaValidator(messageSchema)
    if err != nil {
        log.Fatalf("Failed to create validator: %v", err)
    }

    msg := schema.NewHumanMessage("Hello, world!")
    if err := validator.ValidateMessage(ctx, msg); err != nil {
        log.Fatalf("Validation failed: %v", err)
    }


    fmt.Println("Validation successful!")
}
```

## Next Steps

Congratulations! You've integrated JSON Schema validation with Beluga AI. Next, learn how to:

- **[Pydantic/Go Struct Bridge](./pydantic-go-struct-bridge.md)** - Convert between Pydantic and Go structs
- **[Schema Package Documentation](../../api-docs/packages/schema.md)** - Deep dive into schema package
- **[Use Case: Document Processing](../../use-cases/03-intelligent-document-processing.md)** - Real-world validation usage

---

**Ready for more?** Check out the Integrations Index for more integration guides!
