---
title: Working with LLMs
description: Configure language models, stream responses, use structured output, and build multi-provider setups.
---

The `llm` package provides a unified interface for interacting with language models. Every provider — from OpenAI to Ollama — implements the same `ChatModel` interface, making it straightforward to switch providers, add middleware, and build multi-model architectures.

## The ChatModel Interface

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

Providers register themselves via `init()`. Import the provider package, then use `llm.New()`:

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

Use `llm.List()` to discover all registered providers at runtime.

## Basic Generation

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

Control model behavior with functional options:

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

Stream responses using Go 1.23 iterators:

```go
for chunk, err := range model.Stream(ctx, msgs) {
	if err != nil {
		log.Printf("stream error: %v", err)
		break
	}
	fmt.Print(chunk.Delta) // Print text as it arrives
}
```

The `StreamChunk` contains:

```go
type StreamChunk struct {
	Delta      string       // New text content
	ToolCalls  []ToolCall   // Incremental tool call data
	FinishReason string     // Set on the final chunk
}
```

## Structured Output

Parse LLM responses into typed Go structs:

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

Attach tool definitions so the LLM can request tool calls:

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

`BindTools` returns a new `ChatModel` — the original is not modified.

## Middleware

Middleware wraps `ChatModel` to add cross-cutting behavior. Apply multiple middlewares with `ApplyMiddleware`:

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

Middleware is applied left-to-right in execution order. The first middleware in the list executes first (outermost wrapper).

### Built-in Middleware

| Middleware | Purpose |
|-----------|---------|
| `WithLogging(logger)` | Log Generate/Stream calls via `slog` |
| `WithFallback(model)` | Fall back to another model on retryable errors |
| `WithHooks(hooks)` | Attach lifecycle callbacks |

### Writing Custom Middleware

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

Hooks provide lifecycle callbacks without implementing the full `ChatModel` interface:

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

### Round-Robin Router

Distribute load across multiple providers:

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

Automatic failover when a provider is down:

```go
primary, _ := llm.New("openai", llm.ProviderConfig{Model: "gpt-4o"})
backup, _ := llm.New("anthropic", llm.ProviderConfig{Model: "claude-sonnet-4-5-20250929"})

failover := llm.NewFailoverRouter(primary, backup)

// Automatically falls back to backup on retryable errors
resp, err := failover.Generate(ctx, msgs)
```

### Custom Routing Strategy

Implement `RouterStrategy` for custom selection logic:

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

Manage conversation history within token limits:

```go
contextMgr := llm.NewContextManager(model,
	llm.WithContextMaxTokens(8000),
	llm.WithContextStrategy(llm.StrategyTruncate),
)

// Automatically truncates older messages to fit within the token budget
resp, err := contextMgr.Generate(ctx, longConversation)
```

## Next Steps

- [Building Your First Agent](/guides/first-agent/) — Use ChatModel inside an agent
- [Tools & MCP](/guides/tools-and-mcp/) — Tool binding and MCP integration
- [RAG Pipeline](/guides/rag-pipeline/) — Retrieval-augmented generation
- [Monitoring & Observability](/guides/observability/) — Track LLM usage and performance
