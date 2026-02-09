---
title: Structured Output with LLMs
description: Learn how to extract typed, structured data from LLM responses using JSON schemas and Go struct binding.
---

Language models generate unstructured text by default, but production systems require typed, validated responses. Beluga AI provides structured output capabilities that let you extract data in predictable formats from your LLM calls.

## What You'll Learn

This guide covers:
- Using structured output with LLMs to get typed responses
- Defining JSON schemas for validation
- Binding responses directly to Go structs
- Handling structured output with streaming
- Error handling and fallback strategies

## When to Use Structured Output

Structured output is essential when:
- **Extracting data** from documents (invoices, receipts, forms)
- **Building tool calls** where the LLM needs to return function parameters
- **Generating JSON** for APIs or database records
- **Classification tasks** where you need specific categories
- **Multi-step workflows** that require machine-readable intermediate results

## Prerequisites

Before starting this guide:
- Complete [Working with LLMs](/guides/working-with-llms)
- Understand Go struct tags and JSON marshaling
- Have an LLM provider configured (OpenAI, Anthropic, etc.)

## Basic Structured Output

The simplest way to get structured output is using the `StructuredOutput` option when generating completions.

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

// Define the structure you want
type Person struct {
    Name  string `json:"name" jsonschema:"required,description=Full name of the person"`
    Age   int    `json:"age" jsonschema:"required,description=Age in years"`
    Email string `json:"email" jsonschema:"description=Email address"`
    City  string `json:"city" jsonschema:"description=City of residence"`
}

func main() {
    ctx := context.Background()

    // Create LLM instance
    config := llms.NewConfig(
        llms.WithProvider("openai"),
        llms.WithModelName("gpt-4o"),
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    )

    factory := llms.NewFactory()
    llm, err := factory.CreateLLM("openai", config)
    if err != nil {
        log.Fatal(err)
    }

    // Define your schema
    schemaJSON := `{
        "type": "object",
        "properties": {
            "name": {"type": "string", "description": "Full name"},
            "age": {"type": "integer", "description": "Age in years"},
            "email": {"type": "string", "description": "Email address"},
            "city": {"type": "string", "description": "City of residence"}
        },
        "required": ["name", "age"]
    }`

    // Generate with structured output
    messages := []schema.Message{
        schema.NewHumanMessage("Extract information: John Doe is 32 years old and lives in Seattle. His email is john@example.com"),
    }

    response, err := llm.Generate(ctx, messages,
        llms.WithStructuredOutput(schemaJSON),
        llms.WithTemperature(0.0), // Use deterministic output
    )
    if err != nil {
        log.Fatal(err)
    }

    // Parse the response
    var person Person
    if err := json.Unmarshal([]byte(response.Content), &person); err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Extracted: %+v\n", person)
    // Output: Extracted: {Name:John Doe Age:32 Email:john@example.com City:Seattle}
}
```

## Auto-Generating Schemas from Go Structs

Manually writing JSON schemas is tedious. Use struct tags to auto-generate them.

```go
import "github.com/invopop/jsonschema"

// GenerateSchema creates a JSON schema from a Go struct
func GenerateSchema[T any]() (string, error) {
    reflector := &jsonschema.Reflector{
        AllowAdditionalProperties: false,
        RequiredFromJSONSchemaTags: true,
    }

    var example T
    schema := reflector.Reflect(example)

    schemaBytes, err := json.Marshal(schema)
    if err != nil {
        return "", err
    }

    return string(schemaBytes), nil
}

// Usage
schemaJSON, err := GenerateSchema[Person]()
if err != nil {
    log.Fatal(err)
}
```

## Complex Nested Structures

Structured output supports nested objects and arrays.

```go
type Invoice struct {
    InvoiceNumber string    `json:"invoice_number" jsonschema:"required"`
    Date          string    `json:"date" jsonschema:"required"`
    Vendor        Vendor    `json:"vendor" jsonschema:"required"`
    Items         []LineItem `json:"items" jsonschema:"required,minItems=1"`
    Total         float64   `json:"total" jsonschema:"required"`
}

type Vendor struct {
    Name    string `json:"name" jsonschema:"required"`
    Address string `json:"address"`
    TaxID   string `json:"tax_id"`
}

type LineItem struct {
    Description string  `json:"description" jsonschema:"required"`
    Quantity    int     `json:"quantity" jsonschema:"required,minimum=1"`
    UnitPrice   float64 `json:"unit_price" jsonschema:"required,minimum=0"`
    Amount      float64 `json:"amount" jsonschema:"required,minimum=0"`
}

func ExtractInvoice(ctx context.Context, llm llms.LLM, imageData []byte) (*Invoice, error) {
    // Generate schema
    schemaJSON, err := GenerateSchema[Invoice]()
    if err != nil {
        return nil, fmt.Errorf("generate schema: %w", err)
    }

    // Create multimodal message with image
    messages := []schema.Message{
        schema.NewSystemMessage("You are an expert at extracting structured data from invoices."),
        schema.NewHumanMessage("Extract all information from this invoice."),
        // Add image content part here
    }

    response, err := llm.Generate(ctx, messages,
        llms.WithStructuredOutput(schemaJSON),
        llms.WithTemperature(0.0),
    )
    if err != nil {
        return nil, fmt.Errorf("generate: %w", err)
    }

    var invoice Invoice
    if err := json.Unmarshal([]byte(response.Content), &invoice); err != nil {
        return nil, fmt.Errorf("unmarshal: %w", err)
    }

    return &invoice, nil
}
```

## Classification with Enums

Use structured output for classification tasks with predefined categories.

```go
type SentimentAnalysis struct {
    Sentiment  Sentiment `json:"sentiment" jsonschema:"required,enum=positive|negative|neutral"`
    Confidence float64   `json:"confidence" jsonschema:"required,minimum=0,maximum=1"`
    Reasoning  string    `json:"reasoning" jsonschema:"required"`
}

type Sentiment string

const (
    SentimentPositive Sentiment = "positive"
    SentimentNegative Sentiment = "negative"
    SentimentNeutral  Sentiment = "neutral"
)

func AnalyzeSentiment(ctx context.Context, llm llms.LLM, text string) (*SentimentAnalysis, error) {
    schemaJSON, err := GenerateSchema[SentimentAnalysis]()
    if err != nil {
        return nil, err
    }

    messages := []schema.Message{
        schema.NewSystemMessage("Analyze the sentiment of user reviews."),
        schema.NewHumanMessage(fmt.Sprintf("Review: %s", text)),
    }

    response, err := llm.Generate(ctx, messages,
        llms.WithStructuredOutput(schemaJSON),
    )
    if err != nil {
        return nil, err
    }

    var analysis SentimentAnalysis
    if err := json.Unmarshal([]byte(response.Content), &analysis); err != nil {
        return nil, err
    }

    return &analysis, nil
}
```

## Streaming Structured Output

Some providers support streaming structured output token-by-token. This is useful for real-time UI updates.

```go
func StreamStructuredOutput(ctx context.Context, llm llms.LLM) error {
    schemaJSON, _ := GenerateSchema[Person]()

    messages := []schema.Message{
        schema.NewHumanMessage("Extract: Sarah Chen, 28, lives in Tokyo, sarah.chen@example.com"),
    }

    stream, err := llm.Stream(ctx, messages,
        llms.WithStructuredOutput(schemaJSON),
    )
    if err != nil {
        return err
    }

    var buffer strings.Builder
    for chunk := range stream {
        if chunk.Error != nil {
            return chunk.Error
        }

        buffer.WriteString(chunk.Content)

        // Try to parse partial JSON
        var person Person
        if err := json.Unmarshal([]byte(buffer.String()), &person); err == nil {
            // Valid JSON so far
            fmt.Printf("Progress: %+v\n", person)
        }
    }

    return nil
}
```

## Error Handling and Validation

Always validate structured output against your schema and business rules.

```go
import "github.com/xeipuuv/gojsonschema"

func ValidateStructuredOutput(schemaJSON, responseJSON string) error {
    schemaLoader := gojsonschema.NewStringLoader(schemaJSON)
    documentLoader := gojsonschema.NewStringLoader(responseJSON)

    result, err := gojsonschema.Validate(schemaLoader, documentLoader)
    if err != nil {
        return fmt.Errorf("validation error: %w", err)
    }

    if !result.Valid() {
        var errMsgs []string
        for _, desc := range result.Errors() {
            errMsgs = append(errMsgs, desc.String())
        }
        return fmt.Errorf("schema validation failed: %s", strings.Join(errMsgs, "; "))
    }

    return nil
}

func ExtractWithValidation(ctx context.Context, llm llms.LLM, text string) (*Person, error) {
    schemaJSON, _ := GenerateSchema[Person]()

    messages := []schema.Message{
        schema.NewHumanMessage(text),
    }

    response, err := llm.Generate(ctx, messages,
        llms.WithStructuredOutput(schemaJSON),
    )
    if err != nil {
        return nil, fmt.Errorf("generate: %w", err)
    }

    // Validate against schema
    if err := ValidateStructuredOutput(schemaJSON, response.Content); err != nil {
        return nil, fmt.Errorf("invalid output: %w", err)
    }

    var person Person
    if err := json.Unmarshal([]byte(response.Content), &person); err != nil {
        return nil, fmt.Errorf("unmarshal: %w", err)
    }

    // Business rule validation
    if person.Age < 0 || person.Age > 150 {
        return nil, fmt.Errorf("invalid age: %d", person.Age)
    }

    return &person, nil
}
```

## Fallback Strategies

Not all providers support structured output natively. Implement fallback strategies.

```go
func ExtractWithFallback(ctx context.Context, llm llms.LLM, text string) (*Person, error) {
    schemaJSON, _ := GenerateSchema[Person]()

    // Try native structured output first
    messages := []schema.Message{
        schema.NewHumanMessage(text),
    }

    response, err := llm.Generate(ctx, messages,
        llms.WithStructuredOutput(schemaJSON),
    )

    if err != nil {
        // Fallback: Use prompt engineering
        return extractWithPromptEngineering(ctx, llm, text)
    }

    var person Person
    if err := json.Unmarshal([]byte(response.Content), &person); err != nil {
        // Fallback if parsing fails
        return extractWithPromptEngineering(ctx, llm, text)
    }

    return &person, nil
}

func extractWithPromptEngineering(ctx context.Context, llm llms.LLM, text string) (*Person, error) {
    messages := []schema.Message{
        schema.NewSystemMessage("You are a data extraction expert. Always respond with valid JSON only."),
        schema.NewHumanMessage(fmt.Sprintf(
            "Extract person information from this text and return as JSON with fields: name, age, email, city.\n\nText: %s\n\nJSON:",
            text,
        )),
    }

    response, err := llm.Generate(ctx, messages,
        llms.WithTemperature(0.0),
    )
    if err != nil {
        return nil, err
    }

    // Extract JSON from response (may have markdown code blocks)
    jsonStr := extractJSON(response.Content)

    var person Person
    if err := json.Unmarshal([]byte(jsonStr), &person); err != nil {
        return nil, fmt.Errorf("fallback parsing failed: %w", err)
    }

    return &person, nil
}

func extractJSON(text string) string {
    // Remove markdown code blocks
    text = strings.TrimPrefix(text, "```json\n")
    text = strings.TrimPrefix(text, "```\n")
    text = strings.TrimSuffix(text, "\n```")
    return strings.TrimSpace(text)
}
```

## Provider-Specific Features

Different providers have varying levels of structured output support.

### OpenAI Function Calling

OpenAI's function calling is a form of structured output.

```go
func ExtractWithFunctionCalling(ctx context.Context, llm llms.LLM, text string) (*Person, error) {
    // Define function schema
    functionSchema := schema.Tool{
        Name:        "extract_person",
        Description: "Extract person information from text",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "name":  map[string]string{"type": "string"},
                "age":   map[string]string{"type": "integer"},
                "email": map[string]string{"type": "string"},
                "city":  map[string]string{"type": "string"},
            },
            "required": []string{"name", "age"},
        },
    }

    messages := []schema.Message{
        schema.NewHumanMessage(text),
    }

    response, err := llm.Generate(ctx, messages,
        llms.WithTools([]schema.Tool{functionSchema}),
        llms.WithToolChoice("extract_person"),
    )
    if err != nil {
        return nil, err
    }

    // Parse tool call arguments
    var person Person
    if len(response.ToolCalls) > 0 {
        if err := json.Unmarshal([]byte(response.ToolCalls[0].Arguments), &person); err != nil {
            return nil, err
        }
    }

    return &person, nil
}
```

### Anthropic Structured Output

Anthropic models support JSON mode and schema validation.

```go
config := llms.NewConfig(
    llms.WithProvider("anthropic"),
    llms.WithModelName("claude-3-5-sonnet-20241022"),
    llms.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
    llms.WithExtraOptions(map[string]interface{}{
        "json_mode": true,
    }),
)
```

## Production Best Practices

When using structured output in production:

1. **Always validate schemas** before making LLM calls
2. **Set temperature to 0.0** for deterministic extraction
3. **Implement retry logic** with exponential backoff
4. **Monitor schema compliance** using observability hooks
5. **Version your schemas** and track changes
6. **Cache schema generation** for frequently used types
7. **Test with edge cases** (missing fields, malformed data)
8. **Log validation failures** for continuous improvement

```go
type StructuredOutputConfig struct {
    MaxRetries       int
    ValidationStrict bool
    CacheSchemas     bool
    LogFailures      bool
}

func ExtractWithRetry(
    ctx context.Context,
    llm llms.LLM,
    text string,
    config StructuredOutputConfig,
) (*Person, error) {
    var lastErr error

    for i := 0; i < config.MaxRetries; i++ {
        person, err := ExtractWithValidation(ctx, llm, text)
        if err == nil {
            return person, nil
        }

        lastErr = err

        if config.LogFailures {
            log.Printf("Extraction attempt %d failed: %v", i+1, err)
        }

        // Exponential backoff
        time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second)
    }

    return nil, fmt.Errorf("extraction failed after %d retries: %w", config.MaxRetries, lastErr)
}
```

## Next Steps

Now that you understand structured output:
- Learn about [Multi-Agent Systems](/guides/multi-agent-systems) for coordinated extraction
- Explore [Document Processing](/guides/document-processing) for batch extraction
- Read [Prompt Engineering](/guides/prompt-engineering) for better extraction prompts
- Check out [LLM Recipes](/cookbook/llm-recipes) for real-world examples
