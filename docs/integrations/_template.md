# {Integration Title}

Welcome, colleague! In this integration guide, we're going to integrate {external service/tool} with Beluga AI's {package} package. We'll keep it quick and functional so you can see results immediately.

## What you will build

You will create a {brief description of what the integration enables}. This integration allows you to {key benefit/functionality}.

## Learning Objectives

- ✅ Configure {external service/tool} with Beluga AI
- ✅ {Specific task 1}
- ✅ {Specific task 2}
- ✅ Understand configuration options and best practices

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- \{External service/tool\} account/access (if required)
- {Any other dependencies}

## Step 1: Setup and Installation

\{Installation instructions for any required dependencies or setup\}
# Example installation commands
bash
```bash
go get github.com/example/package
```

## Step 2: Configuration

Create the configuration for the integration:
```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/{package}"
)

func main() {
    ctx := context.Background()

    // Step 1: Configure the integration
    config := {package}.NewConfig(
        {package}.With{Option}("value"),
        {package}.With{AnotherOption}(os.Getenv("API_KEY")),
    )

    // Step 2: Create the provider/client
    provider, err := {package}.New{Provider}(config)
    if err != nil {
        fmt.Printf("Error creating provider: %v\n", err)
        return
    }

    // Step 3: Use the integration
    result, err := provider.{Method}(ctx, /* parameters */)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }


    fmt.Printf("Result: %v\n", result)
}
```

### Verification

To verify it worked, run:
bash
```bash
export {ENV_VAR}="your-value"
go run main.go
```

You should see:{Expected output}
```

## Step 3: {Next Step Title}

{Description of what this step does}
// Code example for this step
```

### Verification

{How to verify this step works}

## Step 4: {Advanced Usage or Configuration}

{Description of advanced features or configuration options}
```text
go
go
// Advanced configuration example
config := {package}.NewConfig(
    {package}.With{AdvancedOption}("value"),
    // ... more options
)
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `{Option1}` | {Description} | `{default}` | Yes/No |
| `{Option2}` | {Description} | `{default}` | Yes/No |

## Common Issues

### "{Error Message}"

**Problem**: {Description of the problem}

**Solution**: # Solution steps
```

### "{Another Error}"

**Problem**: {Description}

**Solution**: {Solution steps}

## Production Considerations

When using this integration in production:

- **Error Handling**: {Best practices for error handling}
- **Retries**: {Retry configuration recommendations}
- **Monitoring**: {Observability considerations}
- **Security**: {Security best practices}

## Complete Example

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/{package}"
    "go.opentelemetry.io/otel"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Create configuration
    config := {package}.NewConfig(
        {package}.With{Option}("value"),
        {package}.WithAPIKey(os.Getenv("API_KEY")),
        {package}.WithTimeout(30*time.Second),
    )

    // Create provider
    provider, err := {package}.New{Provider}(config)
    if err != nil {
        log.Fatalf("Failed to create provider: %v", err)
    }

    // Use the integration with error handling
    result, err := provider.{Method}(ctx, /* parameters */)
    if err != nil {
        log.Fatalf("Operation failed: %v", err)
    }


    fmt.Printf("Success: %v\n", result)
}
```

## Next Steps

Congratulations! You've integrated {external service/tool} with Beluga AI. Next, learn how to:

- **[Related Integration Guide](./related-integration.md)** - {Description}
- **[Package Documentation](../../api-docs/packages/{package}.md)** - Deep dive into {package} package
- **[Use Case Example](../../use-cases/{related-use-case}.md)** - Real-world usage example
- **[Cookbook Recipe](../../cookbook/{related-recipe}.md)** - Quick reference recipe

---

**Ready for more?** Check out the Integrations Index for more integration guides!
