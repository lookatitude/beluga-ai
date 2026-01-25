# Beluga AI Framework - Quick Start Guide

Welcome to Beluga AI! This guide will help you get started with the framework in just a few minutes. By the end of this guide, you'll have:

- ✅ Installed the framework
- ✅ Made your first LLM API call
- ✅ Configured a provider
- ✅ Created your first agent

## Prerequisites

Before you begin, ensure you have:

- **Go 1.24 or later** installed ([Download Go](https://golang.org/dl/))
- **API Keys** for at least one LLM provider:
  - OpenAI: [Get API Key](https://platform.openai.com/api-keys)
  - Anthropic: [Get API Key](https://console.anthropic.com/)
  - Or use Ollama for local models (no API key needed)

Verify your Go installation:
```bash
go version
# Should output: go version go1.24.x or later
```

## Step 1: Installation

Install the Beluga AI framework:

```bash
go get github.com/lookatitude/beluga-ai
```

Or if you're starting a new project:

```bash
mkdir my-beluga-app
cd my-beluga-app
go mod init my-beluga-app
go get github.com/lookatitude/beluga-ai
```

## Step 2: Your First LLM Call

Create a file `main.go`:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
	ctx := context.Background()

	// Create LLM configuration
	config := llms.NewConfig(
		llms.WithProvider("openai"),                    // or "anthropic", "ollama"
		llms.WithModelName("gpt-4"),                    // or "claude-3-sonnet-20240229"
		llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),  // Set OPENAI_API_KEY env var
		llms.WithTemperatureConfig(0.7),
		llms.WithMaxTokensConfig(500),
	)

	// Create provider using factory
	factory := llms.NewFactory()
	provider, err := factory.CreateProvider("openai", config)
	if err != nil {
		fmt.Printf("Error creating provider: %v\n", err)
		return
	}

	// Create messages
	messages := []schema.Message{
		schema.NewSystemMessage("You are a helpful AI assistant."),
		schema.NewHumanMessage("What is the capital of France? Answer in one sentence."),
	}

	// Generate response
	response, err := provider.Generate(ctx, messages)
	if err != nil {
		fmt.Printf("Error generating response: %v\n", err)
		return
	}

	// Print the response
	fmt.Printf("Response: %s\n", response.Content)
}
```

Run it:

```bash
export OPENAI_API_KEY="your-api-key-here"
go run main.go
```

**Expected Output:**
```
Response: The capital of France is Paris.
```

### Using Ollama (Local Models)

If you prefer to use local models with Ollama (no API key needed):

```go
config := llms.NewConfig(
	llms.WithProvider("ollama"),
	llms.WithModelName("llama2"),  // or any model you have locally
	llms.WithBaseURL("http://localhost:11434"),  // Default Ollama URL
)
```

Make sure Ollama is running:
```bash
# Install Ollama from https://ollama.ai
ollama pull llama2
ollama serve
```

## Step 3: Configuration with YAML

For production applications, use YAML configuration files. Create `config.yaml`:

```yaml
llm_providers:
  - name: "openai-gpt4"
    provider: "openai"
    model_name: "gpt-4"
    api_key: "${OPENAI_API_KEY}"  # Environment variable
    temperature: 0.7
    max_tokens: 1000
    timeout: "30s"
    max_retries: 3
    retry_delay: "1s"
    retry_backoff: 2.0

  - name: "anthropic-claude"
    provider: "anthropic"
    model_name: "claude-3-sonnet-20240229"
    api_key: "${ANTHROPIC_API_KEY}"
    temperature: 0.8
    max_tokens: 2048
```

Load configuration in your code:

```go
import (
	"log"
	
	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/lookatitude/beluga-ai/pkg/llms"
)

// Load configuration from YAML file
cfgProvider, err := config.NewYAMLProvider("config", []string{"."}, "")
if err != nil {
	log.Fatal(err)
}

// Get LLM config from loaded configuration using UnmarshalKey
var llmConfig llms.Config
if err := cfgProvider.UnmarshalKey("llm_providers.0", &llmConfig); err != nil {
	log.Fatal(err)
}

// Use the config
factory := llms.NewFactory()
provider, err := factory.CreateProvider("openai", &llmConfig)
if err != nil {
	log.Fatal(err)
}
```

## Step 4: Create Your First Agent

Agents are autonomous entities that can reason, plan, and execute tasks using tools. Here's a simple example:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
	ctx := context.Background()

	// 1. Create LLM provider
	llmConfig := llms.NewConfig(
		llms.WithProvider("openai"),
		llms.WithModelName("gpt-4"),
		llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
	)

	factory := llms.NewFactory()
	llmProvider, err := factory.CreateProvider("openai", llmConfig)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// 2. Create tools for the agent
	calculator := tools.NewCalculatorTool()
	echoTool := tools.NewEchoTool()

	agentTools := []tools.Tool{
		calculator,
		echoTool,
	}

	// 3. Create agent
	agent, err := agents.NewBaseAgent(
		"my-assistant",
		llmProvider,
		agentTools,
		agents.WithMaxRetries(3),
		agents.WithMaxIterations(10),
	)
	if err != nil {
		fmt.Printf("Error creating agent: %v\n", err)
		return
	}

	// 4. Initialize agent
	if err := agent.Initialize(map[string]interface{}{
		"max_retries": 3,
	}); err != nil {
		fmt.Printf("Error initializing agent: %v\n", err)
		return
	}

	// 5. Execute agent with input
	input := map[string]interface{}{
		"input": "Calculate 15 * 23 and then echo the result",
	}

	result, err := agent.Invoke(ctx, input)
	if err != nil {
		fmt.Printf("Error executing agent: %v\n", err)
		return
	}

	fmt.Printf("Agent Result: %v\n", result)
}
```

## Step 5: Streaming Responses

For real-time responses, use streaming:

```go
// Create messages
messages := []schema.Message{
	schema.NewSystemMessage("You are a helpful assistant."),
	schema.NewHumanMessage("Write a short poem about AI."),
}

// Stream the response
streamChan, err := provider.StreamChat(ctx, messages)
if err != nil {
	log.Fatal(err)
}

// Process chunks as they arrive
for chunk := range streamChan {
	if chunk.Err != nil {
		fmt.Printf("Error: %v\n", chunk.Err)
		break
	}
	fmt.Print(chunk.Content)  // Print as it arrives
}
fmt.Println()  // New line at the end
```

## Step 6: Error Handling

Beluga AI provides comprehensive error handling:

```go
response, err := provider.Generate(ctx, messages)
if err != nil {
	// Check if it's an LLM-specific error
	if llms.IsLLMError(err) {
		code := llms.GetLLMErrorCode(err)
		fmt.Printf("LLM Error Code: %s\n", code)

		// Check if error is retryable
		if llms.IsRetryableError(err) {
			fmt.Println("This error can be retried")
			// Implement retry logic here
		}
	} else {
		fmt.Printf("Non-LLM error: %v\n", err)
	}
	return
}
```

Common error codes:
- `rate_limit` - Rate limit exceeded (retryable)
- `authentication_error` - Invalid API key (not retryable)
- `invalid_request` - Invalid request format (retryable)
- `network_error` - Network issues (retryable)
- `internal_error` - Provider internal error (retryable)

## Troubleshooting

### Common Issues

**1. "API key not found" error**
```bash
# Make sure you've set the environment variable
export OPENAI_API_KEY="your-key-here"
# Or for Anthropic
export ANTHROPIC_API_KEY="your-key-here"
```

**2. "Provider not found" error**
- Ensure you're using a supported provider name: `openai`, `anthropic`, `bedrock`, `ollama`, `gemini`
- Check that the provider is registered in the factory

**3. "Model not found" error**
- Verify the model name is correct for your provider
- For OpenAI: `gpt-4`, `gpt-3.5-turbo`
- For Anthropic: `claude-3-sonnet-20240229`, `claude-3-opus-20240229`
- For Ollama: Use `ollama list` to see available models

**4. Connection timeout**
- Check your network connection
- Verify the API endpoint URL is correct
- Increase timeout in config: `llms.WithTimeout(60 * time.Second)`

## Next Steps

Now that you've completed the quick start, explore more:

1. **📚 [Architecture Documentation](./architecture.md)** - Understand the framework's design
2. **📦 [Package Design Patterns](./package_design_patterns.md)** - Learn best practices
3. **🎯 [Use Cases](./use-cases/)** - See real-world examples
4. **🔧 [API Documentation](./api-reference.md)** - Detailed API reference
5. **🤝 [Contributing Guide](https://github.com/lookatitude/beluga-ai/blob/main/CONTRIBUTING.md)** - Contribute to the project

### Recommended Learning Path

1. **Basic Usage** ✅ (You just completed this!)
2. **Memory Management** - Add conversation memory to agents
3. **Vector Stores** - Implement RAG (Retrieval-Augmented Generation)
4. **Orchestration** - Build complex workflows
5. **Observability** - Add monitoring and tracing

### Example Projects

Check out the examples directory:
```bash
cd examples/llm-usage
go run main.go
```

## Getting Help

- **Documentation**: [Full Documentation](https://github.com/lookatitude/beluga-ai/blob/main/README.md)
- **Issues**: [GitHub Issues](https://github.com/lookatitude/beluga-ai/issues)
- **Framework Comparison**: [Comparison with LangChain/CrewAI](./framework-comparison.md)

## Summary

You've learned:
- ✅ How to install Beluga AI
- ✅ How to make your first LLM call
- ✅ How to configure providers
- ✅ How to create agents with tools
- ✅ How to handle errors and stream responses

**Ready to build something amazing?** Start with a simple project and gradually add more features. The framework is designed to scale from simple scripts to enterprise applications.

Happy coding! 🐋

