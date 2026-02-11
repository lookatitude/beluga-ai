---
title: AWS Bedrock Integration
description: Integrate AWS Bedrock with Beluga AI to access Claude, Llama, Titan, and other foundation models through a unified interface with IAM authentication.
---

Many organizations standardize on AWS for infrastructure and require all AI API traffic to flow through their AWS account. AWS Bedrock makes this possible by providing managed access to foundation models from Anthropic, Meta, Amazon, and others through AWS APIs, using IAM for authentication instead of vendor-specific API keys.

Choose Bedrock when your organization requires AWS-native billing, IAM-based access control, VPC PrivateLink for network isolation, or when you want to access multiple model families (Claude, Llama, Titan) through a single provider without managing separate API keys.

## Overview

The Bedrock provider uses the AWS SDK v2 Converse API, which provides a consistent interface across all Bedrock models. This means the same Beluga code works whether you are calling Claude, Llama, or Titan -- only the model ID changes.

Key benefits:
- IAM-based authentication (no separate API keys)
- Access to multiple model families through a single provider
- AWS VPC and PrivateLink support for network isolation
- Built-in usage tracking through AWS Cost Explorer

## Prerequisites

- Go 1.23 or later
- A Beluga AI project initialized with `go mod init`
- An AWS account with Bedrock access enabled
- AWS credentials configured (IAM role, environment variables, or AWS config file)
- Target models enabled in the Bedrock console for your region

## Installation

Install the Bedrock provider and AWS SDK dependencies:

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/bedrock
```

Configure AWS credentials using one of the standard methods:

```bash
# Option 1: Environment variables
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"

# Option 2: AWS CLI configuration (recommended for development)
aws configure
```

## Configuration

### Basic Setup

Create a Bedrock ChatModel using the registry:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	_ "github.com/lookatitude/beluga-ai/llm/providers/bedrock"
	"github.com/lookatitude/beluga-ai/schema"
)

func main() {
	ctx := context.Background()

	// Create a Bedrock model via the registry.
	model, err := llm.New("bedrock", config.ProviderConfig{
		Model: "us.anthropic.claude-sonnet-4-5-20250929-v1:0",
		Options: map[string]any{
			"region": os.Getenv("AWS_REGION"),
		},
	})
	if err != nil {
		log.Fatalf("Failed to create Bedrock model: %v", err)
	}

	msgs := []schema.Message{
		schema.NewHumanMessage("What is the capital of France?"),
	}

	resp, err := model.Generate(ctx, msgs)
	if err != nil {
		log.Fatalf("Generate failed: %v", err)
	}

	fmt.Printf("Response: %s\n", resp.Text())
}
```

### Using Different Model Families

Switch between Bedrock models by changing only the model ID:

```go
// Anthropic Claude on Bedrock
claudeModel, err := llm.New("bedrock", config.ProviderConfig{
	Model:   "us.anthropic.claude-sonnet-4-5-20250929-v1:0",
	Options: map[string]any{"region": "us-east-1"},
})

// Meta Llama on Bedrock
llamaModel, err := llm.New("bedrock", config.ProviderConfig{
	Model:   "meta.llama3-70b-instruct-v1:0",
	Options: map[string]any{"region": "us-east-1"},
})

// Amazon Titan on Bedrock
titanModel, err := llm.New("bedrock", config.ProviderConfig{
	Model:   "amazon.titan-text-lite-v1",
	Options: map[string]any{"region": "us-east-1"},
})
```

### Static Credentials

For environments where IAM roles are not available, pass credentials directly:

```go
model, err := llm.New("bedrock", config.ProviderConfig{
	Model:  "us.anthropic.claude-sonnet-4-5-20250929-v1:0",
	APIKey: os.Getenv("AWS_ACCESS_KEY_ID"),
	Options: map[string]any{
		"region":     "us-east-1",
		"secret_key": os.Getenv("AWS_SECRET_ACCESS_KEY"),
	},
})
```

## Usage

### Streaming

Stream responses for real-time output:

```go
for chunk, err := range model.Stream(ctx, msgs) {
	if err != nil {
		log.Fatalf("Stream error: %v", err)
	}
	fmt.Print(chunk.Delta)
}
fmt.Println()
```

### Generation Options

Control model behavior with per-request options:

```go
resp, err := model.Generate(ctx, msgs,
	llm.WithTemperature(0.7),
	llm.WithMaxTokens(1000),
)
if err != nil {
	log.Fatalf("Generate failed: %v", err)
}
```

### Complete Production Example

A production-ready example with context timeout and error handling:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	_ "github.com/lookatitude/beluga-ai/llm/providers/bedrock"
	"github.com/lookatitude/beluga-ai/schema"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	model, err := llm.New("bedrock", config.ProviderConfig{
		Model:   "us.anthropic.claude-sonnet-4-5-20250929-v1:0",
		Timeout: 30 * time.Second,
		Options: map[string]any{
			"region": os.Getenv("AWS_REGION"),
		},
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	msgs := []schema.Message{
		schema.NewSystemMessage("You are a helpful assistant."),
		schema.NewHumanMessage("Explain quantum computing in simple terms."),
	}

	resp, err := model.Generate(ctx, msgs,
		llm.WithTemperature(0.7),
		llm.WithMaxTokens(1000),
	)
	if err != nil {
		log.Fatalf("Generate failed: %v", err)
	}

	fmt.Printf("Response: %s\n", resp.Text())
}
```

## Advanced Topics

### IAM Policy

The IAM role or user must have Bedrock invocation permissions:

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

For production, scope the `Resource` to specific model ARNs rather than using a wildcard.

### Cross-Region Model Access

Some models are available only in specific AWS regions. Use the `region` option to target the correct region:

```go
// Access a model available only in us-west-2
model, err := llm.New("bedrock", config.ProviderConfig{
	Model:   "us.anthropic.claude-opus-4-20250514-v1:0",
	Options: map[string]any{"region": "us-west-2"},
})
```

### LLM Router with Bedrock

Use Beluga's LLM Router to route between Bedrock models based on cost, latency, or capability:

```go
import "github.com/lookatitude/beluga-ai/llm"

router := llm.NewRouter(
	llm.Route("complex", complexModel),
	llm.Route("simple", simpleModel),
)
```

## Configuration Reference

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `Model` | Bedrock model ID (e.g., `us.anthropic.claude-sonnet-4-5-20250929-v1:0`) | -- | Yes |
| `APIKey` | AWS access key ID (if not using IAM roles) | From AWS config | No |
| `Timeout` | Maximum request duration | `30s` | No |
| `region` (Options) | AWS region | `us-east-1` | No |
| `secret_key` (Options) | AWS secret access key (if using static credentials) | From AWS config | No |

## Troubleshooting

### "AccessDeniedException"

The AWS credentials do not have Bedrock invocation permissions. Verify:
1. The IAM role or user has the `bedrock:InvokeModel` permission.
2. The resource ARN in the policy matches the model you are invoking.
3. Credentials are correctly configured in the environment.

### "Model not found" or "ValidationException"

The specified model is not enabled in your region. To resolve:
1. Open the AWS Bedrock console.
2. Navigate to **Model access** in the left sidebar.
3. Request access to the model you want to use.
4. Wait for access approval before retrying.

### "ExpiredTokenException"

Temporary AWS credentials (from STS or instance profiles) have expired. Refresh your credentials or ensure the IAM role's session duration is sufficient for your workload.

## Related Resources

- [LLM Providers Overview](/integrations/llm-providers) -- All supported LLM providers
- [Anthropic Claude Enterprise](/integrations/anthropic-enterprise) -- Direct Anthropic API access
- [Resilience Package](/guides/resilience) -- Retry and circuit breaker patterns
