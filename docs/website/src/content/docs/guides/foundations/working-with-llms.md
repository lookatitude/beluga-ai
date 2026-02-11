---
title: Working with LLMs
description: Configure language models, stream responses, use structured output, and build multi-provider setups.
---

The `llm` package provides a unified interface for interacting with language models. Every provider — from OpenAI to Ollama — implements the same `ChatModel` interface, making it straightforward to switch providers, add middleware, and build multi-model architectures. This abstraction is the foundation of Beluga's provider-agnostic design: your application code depends on `ChatModel`, not on any specific provider SDK.

## The ChatModel Interface

The `ChatModel` interface defines four methods that capture the complete lifecycle of LLM interaction. `Generate` handles synchronous request-response patterns. `Stream` returns an `iter.Seq2[schema.StreamChunk, error]` iterator for real-time token delivery. `BindTools` returns a new model instance with tool definitions attached — it uses an immutable copy pattern so the original model is never modified. `ModelID` provides the underlying model identifier for logging and routing decisions.

This interface is deliberately small. A small interface is easier to implement (each new provider only needs four methods), easier to wrap (middleware composes cleanly), and easier to test (mocks are straightforward).

```go
type ChatModel interface {
	Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error)
	Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error]
	BindTools(tools []schema.ToolDefinition) ChatModel
	ModelID() string
}
```

| Method | Purpose |
|--------|---------|
| `Generate` | Synchronous completion — returns a full response |
| `Stream` | Streaming completion — returns an iterator of chunks |
| `BindTools` | Returns a new model with tool definitions attached |
| `ModelID` | Returns the model identifier (e.g., `"gpt-4o"`) |

## Provider Setup

Providers register themselves via `init()` — import the provider package with a blank identifier, and it becomes available through `llm.New()`. This is Beluga's registry pattern: each provider calls `llm.Register()` in its `init()` function, mapping a string name to a factory function. The advantage of this approach is zero configuration boilerplate — you declare which providers you want by importing them, and the registry handles the rest. There is no central configuration file to maintain or provider list to keep in sync.

### OpenAI

```go
import (
	"github.com/lookatitude/beluga-ai/llm"
	_ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

model, err := llm.New("openai", llm.ProviderConfig{
	APIKey: os.Getenv("OPENAI_API_KEY"),
	Model:  "gpt-4o",
})
```

### Anthropic

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"

model, err := llm.New("anthropic", llm.ProviderConfig{
	APIKey: os.Getenv("ANTHROPIC_API_KEY"),
	Model:  "claude-sonnet-4-5-20250929",
})
```

### Google (Gemini)

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/google"

model, err := llm.New("google", llm.ProviderConfig{
	APIKey: os.Getenv("GOOGLE_API_KEY"),
	Model:  "gemini-2.0-flash",
})
```

### Ollama (Local)

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/ollama"

model, err := llm.New("ollama", llm.ProviderConfig{
	BaseURL: "http://localhost:11434",
	Model:   "llama3.1",
})
```

### Available Providers

Beluga ships with 20 provider packages. Most providers that expose an OpenAI-compatible API share the same internal HTTP client (`internal/openaicompat/`), which means adding a new compatible provider requires minimal code. Use `llm.List()` to discover all registered providers at runtime — this is useful for building UIs that let users select their preferred model.

| Provider | Import Path | Config Notes |
|----------|-------------|-------------|
| OpenAI | `llm/providers/openai` | `APIKey`, `Model` |
| Anthropic | `llm/providers/anthropic` | `APIKey`, `Model` |
| Google | `llm/providers/google` | `APIKey`, `Model` |
| Ollama | `llm/providers/ollama` | `BaseURL`, `Model` |
| Azure | `llm/providers/azure` | `APIKey`, `BaseURL`, `Model` |
| Bedrock | `llm/providers/bedrock` | AWS credentials |
| Groq | `llm/providers/groq` | `APIKey`, `Model` |
| DeepSeek | `llm/providers/deepseek` | `APIKey`, `Model` |
| Mistral | `llm/providers/mistral` | `APIKey`, `Model` |
| Cohere | `llm/providers/cohere` | `APIKey`, `Model` |
| Together | `llm/providers/together` | `APIKey`, `Model` |
| Fireworks | `llm/providers/fireworks` | `APIKey`, `Model` |
| OpenRouter | `llm/providers/openrouter` | `APIKey`, `Model` |
| Perplexity | `llm/providers/perplexity` | `APIKey`, `Model` |
| xAI | `llm/providers/xai` | `APIKey`, `Model` |
| HuggingFace | `llm/providers/huggingface` | `APIKey`, `Model` |
| Cerebras | `llm/providers/cerebras` | `APIKey`, `Model` |
| SambaNova | `llm/providers/sambanova` | `APIKey`, `Model` |
| LiteLLM | `llm/providers/litellm` | `BaseURL`, `Model` |
| Bifrost | `llm/providers/bifrost` | `BaseURL`, `Model` |

## Basic Generation

The simplest interaction pattern sends a list of messages to the model and receives a complete response. Messages are typed — `SystemMessage` sets the model's behavior, `HumanMessage` carries user input, and `AIMessage` represents the model's response. The response includes both the generated text and token usage metadata, which is essential for cost tracking and context window management.

```go
ctx := context.Background()

msgs := []schema.Message{
	schema.NewSystemMessage("You are a helpful assistant."),
	schema.NewHumanMessage("Explain quantum entanglement in one paragraph."),
}

resp, err := model.Generate(ctx, msgs)
if err != nil {
	log.Fatal(err)
}

fmt.Println(resp.Text())
fmt.Printf("Tokens: %d input, %d output\n", resp.Usage.InputTokens, resp.Usage.OutputTokens)
```

## Generation Options

Functional options control model behavior on a per-request basis. This pattern is preferable to configuration structs because options are composable, optional, and self-documenting — each option function name describes exactly what it controls. Default values are set by the provider, so you only specify what you want to override.

```go
resp, err := model.Generate(ctx, msgs,
	llm.WithTemperature(0.7),
	llm.WithMaxTokens(1000),
	llm.WithTopP(0.9),
	llm.WithStopSequences("END", "STOP"),
)
```

| Option | Type | Description |
|--------|------|-------------|
| `WithTemperature(t)` | `float64` | Sampling temperature (0.0–2.0) |
| `WithMaxTokens(n)` | `int` | Maximum tokens to generate |
| `WithTopP(p)` | `float64` | Nucleus sampling (0.0–1.0) |
| `WithStopSequences(s...)` | `string...` | Stop generation on these strings |
| `WithResponseFormat(f)` | `ResponseFormat` | JSON mode or JSON Schema |
| `WithToolChoice(c)` | `ToolChoice` | `auto`, `none`, or `required` |
| `WithSpecificTool(name)` | `string` | Force a specific tool call |

## Streaming

Streaming delivers tokens as they are generated, reducing perceived latency from seconds to milliseconds for the first visible output. Beluga uses Go 1.23 `iter.Seq2` iterators for streaming instead of channels. This design avoids common channel pitfalls — goroutine leaks when consumers abandon a stream, buffer sizing decisions, and the question of who is responsible for closing the channel. With `iter.Seq2`, the stream is consumed with a standard `for range` loop and cleans up automatically when the loop exits, whether by completion or `break`.

```go
for chunk, err := range model.Stream(ctx, msgs) {
	if err != nil {
		log.Printf("stream error: %v", err)
		break
	}
	fmt.Print(chunk.Delta) // Print text as it arrives
}
```

The `StreamChunk` carries incremental data for each token delivery: the new text delta, any partial tool call data, and a finish reason on the final chunk.

```go
type StreamChunk struct {
	Delta      string       // New text content
	ToolCalls  []ToolCall   // Incremental tool call data
	FinishReason string     // Set on the final chunk
}
```

## Structured Output

When you need the model to return data in a specific format rather than free-form text, use `StructuredOutput`. It derives a JSON Schema from a Go type parameter, instructs the model to respond in JSON conforming to that schema, parses the response, and automatically retries if parsing fails. The retry mechanism includes the parse error in the conversation context so the model can self-correct.

This approach is more robust than manually writing JSON schemas because the schema stays in sync with your Go type — if you add a field to the struct, the schema updates automatically.

```go
type Sentiment struct {
	Score     float64 `json:"score"`
	Label     string  `json:"label"`
	Reasoning string  `json:"reasoning"`
}

structured := llm.NewStructured[Sentiment](model)

result, err := structured.Generate(ctx, []schema.Message{
	schema.NewHumanMessage("Analyze the sentiment: 'This product exceeded my expectations!'"),
})
if err != nil {
	log.Fatal(err)
}

fmt.Printf("Sentiment: %s (%.2f)\n", result.Label, result.Score)
fmt.Printf("Reasoning: %s\n", result.Reasoning)
```

`StructuredOutput` generates a JSON Schema from the Go type, instructs the model to respond in JSON, parses the response, and retries on parse failures:

```go
// Configure retry behavior
structured := llm.NewStructured[Sentiment](model,
	llm.WithMaxRetries(3), // Default: 2
)
```

## Tool Binding

Tool binding tells the model what external functions it can call. You provide a list of tool definitions (name, description, and JSON Schema for parameters), and the model decides when and how to invoke them. `BindTools` returns a new `ChatModel` instance — the original is not modified. This immutability is important because it means you can safely bind different tool sets to the same base model for different use cases without interference.

```go
tools := []schema.ToolDefinition{
	{
		Name:        "get_weather",
		Description: "Get current weather for a location",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"city": map[string]any{
					"type":        "string",
					"description": "City name",
				},
			},
			"required": []string{"city"},
		},
	},
}

modelWithTools := model.BindTools(tools)
resp, err := modelWithTools.Generate(ctx, msgs)

// Check for tool calls in the response
for _, tc := range resp.ToolCalls {
	fmt.Printf("Tool: %s, Args: %s\n", tc.Name, tc.Arguments)
}
```

## Middleware

Middleware wraps `ChatModel` to add cross-cutting behavior — logging, metrics, fallback, caching — without modifying the model implementation. The middleware signature is `func(ChatModel) ChatModel`: it takes a model in, returns a wrapped model out. This pattern composes naturally because the wrapped model satisfies the same interface as the original, so middleware can stack to any depth.

`ApplyMiddleware` applies middleware in right-to-left order internally so that the first middleware in your list executes first (outermost wrapper). This means the order you write matches the order of execution, which is the intuitive behavior.

```go
import "log/slog"

logger := slog.Default()

model = llm.ApplyMiddleware(model,
	llm.WithLogging(logger),           // Log all calls
	llm.WithFallback(backupModel),     // Fall back on retryable errors
	llm.WithHooks(llm.Hooks{           // Custom lifecycle hooks
		BeforeGenerate: func(ctx context.Context, msgs []schema.Message) error {
			log.Println("Generating response...")
			return nil
		},
	}),
)
```

### Built-in Middleware

| Middleware | Purpose |
|-----------|---------|
| `WithLogging(logger)` | Log Generate/Stream calls via `slog` |
| `WithFallback(model)` | Fall back to another model on retryable errors |
| `WithHooks(hooks)` | Attach lifecycle callbacks |

### Writing Custom Middleware

To create custom middleware, implement a struct that embeds or delegates to the `next` model for all four `ChatModel` methods. Add your custom logic around the delegation calls. The example below shows a metrics middleware that records latency and error counts for every `Generate` call.

```go
func WithMetrics(collector MetricsCollector) llm.Middleware {
	return func(next llm.ChatModel) llm.ChatModel {
		return &metricsModel{next: next, metrics: collector}
	}
}

type metricsModel struct {
	next    llm.ChatModel
	metrics MetricsCollector
}

func (m *metricsModel) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
	start := time.Now()
	resp, err := m.next.Generate(ctx, msgs, opts...)
	m.metrics.RecordLatency(time.Since(start))
	if err != nil {
		m.metrics.RecordError()
	}
	return resp, err
}

// Implement Stream, BindTools, and ModelID similarly...
```

## Hooks

Hooks provide lifecycle callbacks at specific points in the generation process without requiring you to implement the full `ChatModel` interface. This makes hooks lighter-weight than middleware — use hooks when you need to observe or validate, and middleware when you need to transform behavior.

All hook fields are optional. Setting a field to `nil` means it is skipped with zero overhead. The `OnError` hook can transform or suppress errors: return `nil` to suppress, return the error to propagate, or return a different error to replace it. Hooks compose with `ComposeHooks`, which merges multiple hook structs so that each callback runs in sequence.

```go
hooks := llm.Hooks{
	BeforeGenerate: func(ctx context.Context, msgs []schema.Message) error {
		// Validate, log, or modify before sending
		return nil
	},
	AfterGenerate: func(ctx context.Context, resp *schema.AIMessage, err error) {
		// Record metrics, audit, or cache responses
	},
	OnStream: func(ctx context.Context, chunk schema.StreamChunk) {
		// Monitor streaming progress
	},
	OnToolCall: func(ctx context.Context, call schema.ToolCall) {
		// Audit or filter tool calls
	},
	OnError: func(ctx context.Context, err error) error {
		// Transform or suppress errors
		return err // Return nil to suppress
	},
}

// Compose multiple hooks
combined := llm.ComposeHooks(loggingHooks, metricsHooks, auditHooks)
```

## Multi-Provider Routing

Production systems often need to distribute requests across multiple LLM providers for load balancing, cost optimization, or failover. Beluga's `Router` implements the `ChatModel` interface, so it is a drop-in replacement for any single model — your application code does not need to know whether it is talking to one model or a routing layer.

### Round-Robin Router

Round-robin distributes requests evenly across providers. This is useful for load balancing when multiple providers offer equivalent capabilities, or for staying within per-provider rate limits.

```go
openai, err := llm.New("openai", llm.ProviderConfig{
	APIKey: os.Getenv("OPENAI_API_KEY"),
	Model:  "gpt-4o",
})

anthropic, err := llm.New("anthropic", llm.ProviderConfig{
	APIKey: os.Getenv("ANTHROPIC_API_KEY"),
	Model:  "claude-sonnet-4-5-20250929",
})

router := llm.NewRouter(
	llm.WithModels(openai, anthropic),
	llm.WithStrategy(&llm.RoundRobin{}),
)

// Use router as a normal ChatModel
resp, err := router.Generate(ctx, msgs)
```

### Failover Router

Failover automatically switches to a backup provider when the primary returns a retryable error (network timeout, rate limit, server error). This provides high availability without requiring your application code to handle provider-specific failure modes.

```go
primary, _ := llm.New("openai", llm.ProviderConfig{Model: "gpt-4o"})
backup, _ := llm.New("anthropic", llm.ProviderConfig{Model: "claude-sonnet-4-5-20250929"})

failover := llm.NewFailoverRouter(primary, backup)

// Automatically falls back to backup on retryable errors
resp, err := failover.Generate(ctx, msgs)
```

### Custom Routing Strategy

Implement `RouterStrategy` for custom selection logic. The strategy receives the full list of available models and the current message list, giving it enough context to make informed routing decisions — for example, routing based on message length, content type, or cost constraints.

```go
type CostAwareStrategy struct{}

func (s *CostAwareStrategy) Select(ctx context.Context, models []llm.ChatModel, msgs []schema.Message) (llm.ChatModel, error) {
	// Route short messages to the cheaper model
	totalLen := 0
	for _, msg := range msgs {
		totalLen += len(msg.(*schema.HumanMessage).Text())
	}
	if totalLen < 500 {
		return models[0], nil // Cheaper model
	}
	return models[1], nil // More capable model
}

router := llm.NewRouter(
	llm.WithModels(cheapModel, premiumModel),
	llm.WithStrategy(&CostAwareStrategy{}),
)
```

## Context Manager

Long conversations can exceed a model's context window. The `ContextManager` wraps a `ChatModel` and automatically manages conversation history to stay within a token budget. When the conversation exceeds the limit, the configured strategy determines what to remove — truncation drops the oldest messages, while summarization condenses earlier messages into a summary. Because `ContextManager` implements `ChatModel`, it is transparent to the rest of your application.

```go
contextMgr := llm.NewContextManager(model,
	llm.WithContextMaxTokens(8000),
	llm.WithContextStrategy(llm.StrategyTruncate),
)

// Automatically truncates older messages to fit within the token budget
resp, err := contextMgr.Generate(ctx, longConversation)
```

## Next Steps

- [Building Your First Agent](/guides/foundations/first-agent/) — Use ChatModel inside an agent
- [Structured Output](/guides/foundations/structured-output/) — Deep dive into typed LLM responses
- [Tools & MCP](/guides/tools-and-mcp/) — Tool binding and MCP integration
- [RAG Pipeline](/guides/rag-pipeline/) — Retrieval-augmented generation
- [Monitoring & Observability](/guides/observability/) — Track LLM usage and performance
