# Part 1: Your First LLM Call

In this tutorial, you'll learn how to make your first LLM (Large Language Model) API call using the Beluga AI Framework. By the end, you'll understand how to configure providers, send messages, and handle responses.

## Learning Objectives

- ✅ Configure an LLM provider (OpenAI, Anthropic, or Ollama)
- ✅ Create and send messages to an LLM
- ✅ Handle responses and errors
- ✅ Understand basic configuration options

## Prerequisites

- Go 1.24+ installed
- API key for at least one provider (or Ollama for local models)
- Beluga AI Framework installed

## Step 1: Project Setup

Create a new directory for your project:

```bash
mkdir beluga-tutorial
cd beluga-tutorial
go mod init beluga-tutorial
go get github.com/lookatitude/beluga-ai
```

## Step 2: Basic LLM Call

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

	// Step 1: Create LLM configuration
	config := llms.NewConfig(
		llms.WithProvider("openai"),                    // Provider name
		llms.WithModelName("gpt-3.5-turbo"),            // Model to use
		llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),  // API key from environment
		llms.WithTemperatureConfig(0.7),                // Creativity level (0-2)
		llms.WithMaxTokensConfig(500),                  // Maximum response length
	)

	// Step 2: Create provider using factory
	factory := llms.NewFactory()
	provider, err := factory.CreateProvider("openai", config)
	if err != nil {
		fmt.Printf("Error creating provider: %v\n", err)
		return
	}

	// Step 3: Create messages
	messages := []schema.Message{
		schema.NewSystemMessage("You are a helpful AI assistant."),
		schema.NewHumanMessage("What is the capital of France? Answer in one sentence."),
	}

	// Step 4: Generate response
	response, err := provider.Generate(ctx, messages)
	if err != nil {
		fmt.Printf("Error generating response: %v\n", err)
		return
	}

	// Step 5: Print the response
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

## Step 3: Understanding Messages

Beluga AI uses a message-based system. There are several message types:

### Message Types

```go
// System message - Sets the assistant's behavior
systemMsg := schema.NewSystemMessage("You are a helpful assistant.")

// Human message - User input
humanMsg := schema.NewHumanMessage("Hello, how are you?")

// AI message - Assistant response
aiMsg := schema.NewAIMessage("I'm doing well, thank you!")

// Tool message - Tool execution results
toolMsg := schema.NewToolMessage("42", "calculator")
```

### Conversation Example

```go
messages := []schema.Message{
	schema.NewSystemMessage("You are a math tutor."),
	schema.NewHumanMessage("What is 2 + 2?"),
	schema.NewAIMessage("2 + 2 equals 4."),
	schema.NewHumanMessage("What about 3 + 3?"),
}
```

## Step 4: Using Different Providers

### OpenAI

```go
config := llms.NewConfig(
	llms.WithProvider("openai"),
	llms.WithModelName("gpt-4"),  // or "gpt-3.5-turbo"
	llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
)
```

### Anthropic (Claude)

```go
config := llms.NewConfig(
	llms.WithProvider("anthropic"),
	llms.WithModelName("claude-3-sonnet-20240229"),
	llms.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
)
```

### Ollama (Local Models)

```go
config := llms.NewConfig(
	llms.WithProvider("ollama"),
	llms.WithModelName("llama2"),  // or any model you have locally
	llms.WithBaseURL("http://localhost:11434"),  // Default Ollama URL
)

// Make sure Ollama is running:
// ollama pull llama2
// ollama serve
```

## Step 5: Configuration Options

### Temperature

Controls randomness (0.0 = deterministic, 2.0 = very creative):

```go
config := llms.NewConfig(
	llms.WithProvider("openai"),
	llms.WithModelName("gpt-3.5-turbo"),
	llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
	llms.WithTemperatureConfig(0.7),  // Balanced creativity
)
```

### Max Tokens

Limits response length:

```go
llms.WithMaxTokensConfig(1000),  // Maximum 1000 tokens
```

### Timeout

Sets request timeout:

```go
import "time"

llms.WithTimeout(30 * time.Second),
```

### Retry Configuration

Handles transient failures:

```go
llms.WithRetryConfig(
	3,                    // Max retries
	1 * time.Second,      // Initial delay
	2.0,                  // Backoff multiplier
),
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

### Common Error Codes

- `rate_limit` - Rate limit exceeded (retryable)
- `authentication_error` - Invalid API key (not retryable)
- `invalid_request` - Invalid request format (retryable)
- `network_error` - Network issues (retryable)
- `internal_error` - Provider internal error (retryable)

## Step 7: Streaming Responses

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

## Complete Example

Here's a complete example with error handling and configuration:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create configuration
	config := llms.NewConfig(
		llms.WithProvider("openai"),
		llms.WithModelName("gpt-3.5-turbo"),
		llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		llms.WithTemperatureConfig(0.7),
		llms.WithMaxTokensConfig(500),
		llms.WithTimeout(30*time.Second),
		llms.WithRetryConfig(3, 1*time.Second, 2.0),
	)

	// Create provider
	factory := llms.NewFactory()
	provider, err := factory.CreateProvider("openai", config)
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	// Create messages
	messages := []schema.Message{
		schema.NewSystemMessage("You are a helpful AI assistant."),
		schema.NewHumanMessage("Explain quantum computing in simple terms."),
	}

	// Generate response
	response, err := provider.Generate(ctx, messages)
	if err != nil {
		if llms.IsLLMError(err) {
			code := llms.GetLLMErrorCode(err)
			log.Printf("LLM Error [%s]: %v", code, err)
			if !llms.IsRetryableError(err) {
				log.Fatal("Non-retryable error, exiting")
			}
		} else {
			log.Fatalf("Unexpected error: %v", err)
		}
		return
	}

	fmt.Printf("Response: %s\n", response.Content)
}
```

## Exercises

1. **Try different models**: Switch between `gpt-3.5-turbo` and `gpt-4` and compare responses
2. **Adjust temperature**: Try values from 0.0 to 2.0 and observe the difference
3. **Create a conversation**: Build a multi-turn conversation with context
4. **Implement streaming**: Convert the example to use streaming responses
5. **Add error handling**: Implement retry logic for retryable errors

## Common Issues

### "API key not found" error

```bash
# Make sure you've set the environment variable
export OPENAI_API_KEY="your-key-here"
# Or for Anthropic
export ANTHROPIC_API_KEY="your-key-here"
```

### "Provider not found" error

- Ensure you're using a supported provider name: `openai`, `anthropic`, `bedrock`, `ollama`
- Check that the provider is registered in the factory

### "Model not found" error

- Verify the model name is correct for your provider
- For OpenAI: `gpt-4`, `gpt-3.5-turbo`
- For Anthropic: `claude-3-sonnet-20240229`, `claude-3-opus-20240229`
- For Ollama: Use `ollama list` to see available models

## Next Steps

Congratulations! You've made your first LLM call. Next, learn how to:

- **[Part 2: Building a Simple RAG Application](./02-simple-rag.md)** - Create a retrieval-augmented generation pipeline
- **[Concepts: LLMs](../concepts/llms.md)** - Deep dive into LLM concepts
- **[Provider Documentation](../providers/llms/)** - Detailed provider guides

---

**Ready for the next step?** Continue to [Part 2: Building a Simple RAG Application](./02-simple-rag.md)!

