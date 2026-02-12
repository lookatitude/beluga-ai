---
title: Structured Output with LLMs
description: "Extract typed Go structs from LLM responses with Beluga AI — auto-generate JSON schemas, validate and retry on parse failures, and build classification pipelines."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, structured output, JSON schema, LLM, generics, type-safe, validation"
---

Language models generate unstructured text by default, but production systems require typed, validated responses. Beluga AI's `StructuredOutput[T]` generic type bridges this gap: it derives a JSON Schema from any Go struct, instructs the model to respond in conformant JSON, parses the response into a typed value, and retries with self-correction feedback when parsing fails. This eliminates the manual schema writing, parsing boilerplate, and error recovery that structured extraction normally requires.

## What You'll Learn

This guide covers:
- Using `llm.NewStructured[T]` to get typed responses from any provider
- How JSON schemas are auto-generated from Go struct tags
- Nested structures and complex types
- Classification patterns with enums
- Streaming structured output with `iter.Seq2`
- Error handling, validation, and fallback strategies

## When to Use Structured Output

Structured output is essential when your application needs to process the model's response programmatically rather than display it as text. Without structured output, you would need to write fragile regex or string parsing logic to extract data from free-form text. With structured output, the model's response is guaranteed to conform to a schema and deserialize directly into a Go type.

Common use cases include:
- **Extracting data** from documents (invoices, receipts, forms)
- **Building tool calls** where the LLM needs to return function parameters
- **Generating JSON** for APIs or database records
- **Classification tasks** where you need specific categories
- **Multi-step workflows** that require machine-readable intermediate results

## Prerequisites

Before starting this guide:
- Complete [Working with LLMs](/guides/foundations/working-with-llms/)
- Understand Go struct tags and JSON marshaling
- Have an LLM provider configured (OpenAI, Anthropic, etc.)

## Basic Structured Output

The `llm.NewStructured[T]` generic constructor creates a `StructuredOutput[T]` that wraps any `ChatModel`. It uses Go's reflection to generate a JSON Schema from the type parameter `T`, then passes that schema to the model as a response format constraint. The `Generate` method returns a value of type `T` directly — no manual JSON parsing needed. If the model produces invalid JSON, the wrapper automatically retries by appending the parse error to the conversation, giving the model a chance to self-correct.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
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

    // Create LLM instance using the registry pattern
    model, err := llm.New("openai", llm.ProviderConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4o",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create a typed structured output wrapper
    structured := llm.NewStructured[Person](model)

    // Generate — returns Person directly, not raw text
    messages := []schema.Message{
        schema.NewHumanMessage("Extract information: John Doe is 32 years old and lives in Seattle. His email is john@example.com"),
    }

    person, err := structured.Generate(ctx, messages)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Extracted: %+v\n", person)
    // Output: Extracted: {Name:John Doe Age:32 Email:john@example.com City:Seattle}
}
```

## Schema Generation from Go Structs

`StructuredOutput[T]` generates JSON schemas automatically from Go struct tags using the `internal/jsonutil` package. The `json` tag controls field names, `jsonschema` tags control validation constraints (required, description, minimum, maximum, enum), and nested structs produce nested schema objects. This approach eliminates schema drift — if you add, rename, or remove a struct field, the schema updates automatically at compile time.

Standard `jsonschema` tag directives include:
- `required` — marks the field as required in the schema
- `description=...` — adds a description the model uses to understand the field's purpose
- `enum=a|b|c` — restricts values to a predefined set
- `minimum=N`, `maximum=N` — numeric range constraints
- `minItems=N` — minimum array length for slice fields

## Complex Nested Structures

Structured output handles nested objects and arrays. This is useful for extracting hierarchical data like invoices, where a single document contains a vendor object, an array of line items, and scalar totals. Each nested struct generates its own sub-schema, and the model produces a single JSON object that maps to the entire Go type hierarchy.

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

func ExtractInvoice(ctx context.Context, model llm.ChatModel, imageData []byte) (Invoice, error) {
    structured := llm.NewStructured[Invoice](model)

    messages := []schema.Message{
        schema.NewSystemMessage("You are an expert at extracting structured data from invoices."),
        schema.NewHumanMessage("Extract all information from this invoice."),
        // Add image content part here
    }

    return structured.Generate(ctx, messages)
}
```

## Classification with Enums

Structured output is well-suited for classification tasks where the model must choose from predefined categories. By using a Go string type with an `enum` constraint in the schema, the model is restricted to valid labels. The `confidence` field with `minimum=0,maximum=1` constraints ensures the model reports a properly bounded confidence score. Combined with a `reasoning` field, this pattern produces auditable classification results.

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

func AnalyzeSentiment(ctx context.Context, model llm.ChatModel, text string) (SentimentAnalysis, error) {
    structured := llm.NewStructured[SentimentAnalysis](model)

    messages := []schema.Message{
        schema.NewSystemMessage("Analyze the sentiment of user reviews."),
        schema.NewHumanMessage(fmt.Sprintf("Review: %s", text)),
    }

    return structured.Generate(ctx, messages)
}
```

## Streaming Structured Output

Some providers support streaming structured output token-by-token. This is useful for real-time UI updates where you want to show extraction progress as fields are populated. Because JSON is generated left-to-right, partial parsing can reveal completed fields before the full response arrives. Note that streaming structured output uses the model's `Stream` method directly with a response format option, rather than the `StructuredOutput` wrapper.

```go
func StreamStructuredOutput(ctx context.Context, model llm.ChatModel) error {
    // Use StructuredOutput to get the schema, then stream with the model directly
    s := llm.NewStructured[Person](model)
    schemaMap := s.Schema()

    messages := []schema.Message{
        schema.NewHumanMessage("Extract: Sarah Chen, 28, lives in Tokyo, sarah.chen@example.com"),
    }

    var buffer strings.Builder
    for chunk, err := range model.Stream(ctx, messages,
        llm.WithResponseFormat(llm.ResponseFormat{
            Type:   "json_schema",
            Schema: schemaMap,
        }),
    ) {
        if err != nil {
            return err
        }

        buffer.WriteString(chunk.Delta)

        // Try to parse partial JSON as fields complete
        var person Person
        if err := json.Unmarshal([]byte(buffer.String()), &person); err == nil {
            fmt.Printf("Progress: %+v\n", person)
        }
    }

    return nil
}
```

## Error Handling and Validation

`StructuredOutput[T]` handles JSON parse failures automatically through its retry mechanism, but production systems should also validate the extracted data against business rules. Schema validation ensures the JSON structure is correct, while business validation catches semantically invalid values (negative ages, impossible dates, missing cross-references). Layer both types of validation for defense in depth.

```go
func ExtractWithValidation(ctx context.Context, model llm.ChatModel, text string) (Person, error) {
    structured := llm.NewStructured[Person](model,
        llm.WithMaxRetries(3),
    )

    messages := []schema.Message{
        schema.NewHumanMessage(text),
    }

    person, err := structured.Generate(ctx, messages)
    if err != nil {
        return Person{}, fmt.Errorf("generate: %w", err)
    }

    // Business rule validation
    if person.Age < 0 || person.Age > 150 {
        return Person{}, fmt.Errorf("invalid age: %d", person.Age)
    }

    if person.Name == "" {
        return Person{}, fmt.Errorf("name is required")
    }

    return person, nil
}
```

## Fallback Strategies

Not all providers support native JSON Schema response formats. When working with a provider that lacks native support, implement a fallback strategy that uses prompt engineering to request JSON output, then parses the result manually. This two-tier approach lets you use native structured output where available for reliability, and fall back to prompt-based extraction otherwise.

```go
func ExtractWithFallback(ctx context.Context, model llm.ChatModel, text string) (Person, error) {
    // Try native structured output first
    structured := llm.NewStructured[Person](model)

    messages := []schema.Message{
        schema.NewHumanMessage(text),
    }

    person, err := structured.Generate(ctx, messages)
    if err == nil {
        return person, nil
    }

    // Fallback: Use prompt engineering for providers without native support
    return extractWithPromptEngineering(ctx, model, text)
}

func extractWithPromptEngineering(ctx context.Context, model llm.ChatModel, text string) (Person, error) {
    messages := []schema.Message{
        schema.NewSystemMessage("You are a data extraction expert. Always respond with valid JSON only."),
        schema.NewHumanMessage(fmt.Sprintf(
            "Extract person information from this text and return as JSON with fields: name, age, email, city.\n\nText: %s\n\nJSON:",
            text,
        )),
    }

    resp, err := model.Generate(ctx, messages,
        llm.WithTemperature(0.0),
    )
    if err != nil {
        return Person{}, err
    }

    // Extract JSON from response (may have markdown code blocks)
    jsonStr := extractJSON(resp.Text())

    var person Person
    if err := json.Unmarshal([]byte(jsonStr), &person); err != nil {
        return Person{}, fmt.Errorf("fallback parsing failed: %w", err)
    }

    return person, nil
}

func extractJSON(text string) string {
    // Remove markdown code blocks
    text = strings.TrimPrefix(text, "```json\n")
    text = strings.TrimPrefix(text, "```\n")
    text = strings.TrimSuffix(text, "\n```")
    return strings.TrimSpace(text)
}
```

## Tool Binding as Structured Output

An alternative approach to structured output is using tool binding. When you define a tool with a specific parameter schema and force the model to call it, the tool call arguments are guaranteed to match the schema. This technique works across all providers that support function calling, even those without native JSON Schema response formats. It is the same mechanism the agent runtime uses to get structured parameters for tool invocations.

```go
func ExtractWithToolBinding(ctx context.Context, model llm.ChatModel, text string) (Person, error) {
    tools := []schema.ToolDefinition{
        {
            Name:        "extract_person",
            Description: "Extract person information from text",
            InputSchema: map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "name":  map[string]any{"type": "string", "description": "Full name"},
                    "age":   map[string]any{"type": "integer", "description": "Age in years"},
                    "email": map[string]any{"type": "string", "description": "Email address"},
                    "city":  map[string]any{"type": "string", "description": "City of residence"},
                },
                "required": []string{"name", "age"},
            },
        },
    }

    modelWithTools := model.BindTools(tools)

    messages := []schema.Message{
        schema.NewHumanMessage(text),
    }

    resp, err := modelWithTools.Generate(ctx, messages,
        llm.WithSpecificTool("extract_person"),
    )
    if err != nil {
        return Person{}, err
    }

    var person Person
    if len(resp.ToolCalls) > 0 {
        if err := json.Unmarshal([]byte(resp.ToolCalls[0].Arguments), &person); err != nil {
            return Person{}, fmt.Errorf("parse tool call: %w", err)
        }
    }

    return person, nil
}
```

## Production Best Practices

When using structured output in production:

1. **Set temperature to 0.0** for deterministic extraction — higher temperatures increase the chance of malformed JSON
2. **Configure retries** via `llm.WithMaxRetries()` — the default of 2 retries handles most transient parse failures
3. **Validate beyond the schema** — check business rules, cross-field consistency, and value ranges after extraction
4. **Monitor parse failure rates** — a sudden increase in retries may indicate a model regression or prompt degradation
5. **Version your struct types** — schema changes can silently break extraction if the model sees a different schema than expected
6. **Use descriptive field tags** — the `description` in `jsonschema` tags gives the model crucial context about what each field should contain
7. **Test with edge cases** — empty strings, missing optional fields, Unicode text, and extremely long values
8. **Prefer `NewStructured[T]` over manual schemas** — auto-generated schemas stay in sync with your types and reduce maintenance burden

## Next Steps

Now that you understand structured output:
- Learn about [Prompt Engineering](/guides/foundations/prompt-engineering/) for better extraction prompts
- Explore [Working with LLMs](/guides/foundations/working-with-llms/) for provider-specific capabilities
- Read [Tools & MCP](/guides/tools-and-mcp/) for tool-based structured extraction patterns
