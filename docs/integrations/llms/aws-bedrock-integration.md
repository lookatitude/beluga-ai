# AWS Bedrock Integration

Welcome, colleague! In this integration guide, we're going to integrate AWS Bedrock with Beluga AI's LLMs package. AWS Bedrock provides access to multiple foundation models through a unified API.

## What you will build

You will configure Beluga AI to use AWS Bedrock models (Claude, Llama, Titan, etc.) for LLM operations, enabling you to leverage AWS's managed AI infrastructure with Beluga AI's unified interface.

## Learning Objectives

- ✅ Configure AWS Bedrock with Beluga AI
- ✅ Use Bedrock models (Claude, Llama, Titan)
- ✅ Handle AWS authentication and permissions
- ✅ Understand Bedrock-specific configuration

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- AWS account with Bedrock access
- AWS credentials configured
- Bedrock models enabled in your region

## Step 1: Setup and Installation

Install AWS SDK:
bash
```bash
go get github.com/aws/aws-sdk-go-v2/service/bedrockruntime
```

Configure AWS credentials:
bash
```bash
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"
```

## Step 2: Basic Bedrock Configuration

Create a Bedrock LLM provider:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/llms/providers/bedrock"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
    ctx := context.Background()

    // Create Bedrock configuration
    config := llms.NewConfig(
        llms.WithProvider("bedrock"),
        llms.WithModelName("anthropic.claude-3-sonnet-20240229-v1:0"),
        llms.WithRegion(os.Getenv("AWS_REGION")),
    )

    // Create Bedrock provider
    provider, err := bedrock.NewBedrockLLM(ctx, "anthropic.claude-3-sonnet-20240229-v1:0", 
        llms.WithRegion(os.Getenv("AWS_REGION")),
    )
    if err != nil {
        log.Fatalf("Failed to create Bedrock provider: %v", err)
    }

    // Use the provider
    messages := []schema.Message{
        schema.NewHumanMessage("What is the capital of France?"),
    }

    response, err := provider.Generate(ctx, messages)
    if err != nil {
        log.Fatalf("Failed to generate: %v", err)
    }

    fmt.Printf("Response: %s\n", response.Content)
}
```

### Verification

Run the example:
bash
```bash
export AWS_REGION="us-east-1"
go run main.go
```

You should see a response from Claude.

## Step 3: Using Different Bedrock Models

Switch between different Bedrock models:
// Claude models
claudeProvider, _ := bedrock.NewBedrockLLM(ctx, "anthropic.claude-3-sonnet-20240229-v1:0")

// Llama models
llamaProvider, _ := bedrock.NewBedrockLLM(ctx, "meta.llama2-13b-chat-v1")

// Titan models
titanProvider, _ := bedrock.NewBedrockLLM(ctx, "amazon.titan-text-lite-v1")
```

## Step 4: Advanced Configuration

Configure Bedrock with advanced options:
provider, err := bedrock.NewBedrockLLM(ctx, "anthropic.claude-3-sonnet-20240229-v1:0",
    llms.WithRegion("us-east-1"),
    llms.WithTemperatureConfig(0.7),
    llms.WithMaxTokensConfig(1000),
    llms.WithTimeout(30*time.Second),
)
```

## Step 5: Complete Integration

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/llms/providers/bedrock"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Create Bedrock provider with full configuration
    provider, err := bedrock.NewBedrockLLM(ctx, "anthropic.claude-3-sonnet-20240229-v1:0",
        llms.WithRegion(os.Getenv("AWS_REGION")),
        llms.WithTemperatureConfig(0.7),
        llms.WithMaxTokensConfig(1000),
        llms.WithRetryConfig(3, 1*time.Second, 2.0),
    )
    if err != nil {
        log.Fatalf("Failed to create provider: %v", err)
    }

    // Create messages
    messages := []schema.Message{
        schema.NewSystemMessage("You are a helpful assistant."),
        schema.NewHumanMessage("Explain quantum computing in simple terms."),
    }

    // Generate response
    response, err := provider.Generate(ctx, messages)
    if err != nil {
        log.Fatalf("Generation failed: %v", err)
    }


    fmt.Printf("Response: %s\n", response.Content)
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `ModelName` | Bedrock model ID | - | Yes |
| `Region` | AWS region | `us-east-1` | No |
| `Temperature` | Sampling temperature | `0.7` | No |
| `MaxTokens` | Maximum tokens | `1000` | No |
| `Timeout` | Request timeout | `30s` | No |

## Common Issues

### "AccessDeniedException"

**Problem**: AWS credentials don't have Bedrock permissions.

**Solution**: Add Bedrock permissions to your IAM role:

```json
{
  "Effect": "Allow",
  "Action": [
    "bedrock:InvokeModel",
    "bedrock:InvokeModelWithResponseStream"
  ],
  "Resource": "*"
}
```

### "Model not found"

**Problem**: Model not enabled in your region.

**Solution**: Enable the model in AWS Bedrock console:
1. Go to AWS Bedrock console
2. Navigate to "Model access"
3. Enable the model you want to use

## Production Considerations

When using Bedrock in production:

- **Use IAM roles**: Prefer IAM roles over access keys
- **Enable models**: Ensure models are enabled in your region
- **Monitor costs**: Track Bedrock API usage
- **Set timeouts**: Configure appropriate timeouts
- **Handle rate limits**: Implement retry logic

## Next Steps

Congratulations! You've integrated AWS Bedrock with Beluga AI. Next, learn how to:

- **[Anthropic Claude Enterprise](./anthropic-claude-enterprise.md)** - Enterprise Claude setup
- **[LLM Providers Guide](../../guides/llm-providers.md)** - LLM configuration
- **[LLMs Package Documentation](../../api-docs/packages/llms.md)** - Deep dive into LLMs package

---

**Ready for more?** Check out the Integrations Index for more integration guides!
