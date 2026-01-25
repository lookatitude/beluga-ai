# Anthropic Claude Enterprise

Welcome, colleague! In this integration guide, we're going to configure Anthropic Claude Enterprise with Beluga AI. Claude Enterprise provides enhanced security, priority access, and enterprise support.

## What you will build

You will configure Beluga AI to use Anthropic Claude Enterprise API with enhanced security features, priority access, and enterprise-grade support for production AI applications.

## Learning Objectives

- ✅ Configure Claude Enterprise API
- ✅ Use enterprise features (extended context, priority access)
- ✅ Handle enterprise authentication
- ✅ Understand enterprise-specific configuration

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Anthropic Enterprise API key
- Enterprise account access

## Step 1: Setup and Installation

Configure Anthropic API key:
bash
```bash
export ANTHROPIC_API_KEY="sk-ant-enterprise-..."
```

## Step 2: Basic Enterprise Configuration

Create an Anthropic Enterprise provider:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/llms/providers/anthropic"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
    ctx := context.Background()

    // Create Anthropic Enterprise configuration
    config := llms.NewConfig(
        llms.WithProvider("anthropic"),
        llms.WithModelName("claude-3-opus-20240229"),
        llms.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
        llms.WithBaseURL("https://api.anthropic.com"), // Enterprise endpoint
    )

    // Create provider
    factory := llms.NewFactory()
    provider, err := factory.CreateProvider("anthropic", config)
    if err != nil {
        log.Fatalf("Failed to create provider: %v", err)
    }

    // Use with extended context
    messages := []schema.Message{
        schema.NewSystemMessage("You are an enterprise AI assistant."),
        schema.NewHumanMessage("Analyze this enterprise data..."),
    }

    response, err := provider.Generate(ctx, messages)
    if err != nil {
        log.Fatalf("Failed to generate: %v", err)
    }


    fmt.Printf("Response: %s\n", response.Content)
}
```

## Step 3: Enterprise Features

Use enterprise-specific features:
```go
// Extended context window (200K tokens)
config := llms.NewConfig(
    llms.WithProvider("anthropic"),
    llms.WithModelName("claude-3-opus-20240229"),
    llms.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
    llms.WithMaxTokensConfig(4096), // Enterprise supports larger contexts
)
// Priority access with retry configuration
config = llms.NewConfig(

    llms.WithProvider("anthropic"),
    llms.WithModelName("claude-3-opus-20240229"),
    llms.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
    llms.WithRetryConfig(5, 500*time.Millisecond, 1.5), // More retries for enterprise
)

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `APIKey` | Anthropic Enterprise API key | - | Yes |
| `ModelName` | Claude model (Opus, Sonnet) | `claude-3-opus-20240229` | No |
| `BaseURL` | API endpoint | `https://api.anthropic.com` | No |
| `MaxTokens` | Maximum response tokens | `4096` | No |

## Common Issues

### "Invalid API key"

**Problem**: Using non-enterprise API key.

**Solution**: Ensure you're using an enterprise API key:export ANTHROPIC_API_KEY="sk-ant-enterprise-..."
```

## Production Considerations

When using Claude Enterprise in production:

- **Use enterprise endpoints**: Ensure you're using enterprise API endpoints
- **Monitor usage**: Track API usage and costs
- **Leverage extended context**: Use 200K token context windows
- **Priority access**: Take advantage of priority routing

## Next Steps

Congratulations! You've configured Claude Enterprise with Beluga AI. Next, learn how to:

- **[AWS Bedrock Integration](./aws-bedrock-integration.md)** - Use Bedrock models
- **[LLM Providers Guide](../../guides/llm-providers.md)** - LLM configuration
- **[LLMs Package Documentation](../../api-docs/packages/llms.md)** - Deep dive into LLMs

---

**Ready for more?** Check out the Integrations Index for more integration guides!
