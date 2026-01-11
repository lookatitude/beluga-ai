# Prompts Basic Example

This example demonstrates how to use the Prompts package for creating and formatting prompt templates.

## Prerequisites

- Go 1.21+

## Running the Example

```bash
go run main.go
```

## What This Example Shows

1. Creating a prompt manager
2. Creating string templates with variable placeholders
3. Formatting templates with variable values
4. Creating multiple templates for different use cases

## Template Syntax

Templates use `{{variable}}` syntax for variable substitution:

```go
template := "Hello, {{name}}! Welcome to {{company}}."
variables := map[string]interface{}{
    "name": "Alice",
    "company": "Beluga AI",
}
result := template.Format(ctx, variables)
// Result: "Hello, Alice! Welcome to Beluga AI."
```

## Configuration Options

- `WithConfig`: Custom configuration
- `WithMetrics`: Metrics collection
- `WithTracer`: OpenTelemetry tracing
- `WithLogger`: Custom logger

## Advanced Usage

- Chat message templates for LLM interactions
- Template caching for performance
- Variable validation
- Conditional formatting

## See Also

- [Prompts Package Documentation](../../../pkg/prompts/README.md)
- [Agent Examples](../../agents/basic/main.go)
