---
title: LLM Recipes
description: "Production Go recipes for LLM integration: streaming responses, token counting, history trimming, error handling, and retry strategies with Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go LLM recipes, LLM streaming, token counting, history trimming, error handling, retry logic, ChatModel"
sidebar:
  order: 0
---

These recipes address the core challenges of working with LLMs in production: managing context windows, handling streaming responses, dealing with transient errors, routing between providers, and extracting structured data. Each recipe builds on Beluga AI's LLM abstraction layer -- the `ChatModel` interface, middleware composition, and hooks -- so the patterns work consistently across all providers (OpenAI, Anthropic, Google, Ollama, and others).

## Trim Chat History to Fit Context Windows

**Problem:** Conversation history grows beyond the model's context window, causing failures or truncated responses.

**Solution:** Use `llm.ContextManager` with a truncation or sliding window strategy.

Every LLM has a fixed context window (4K, 8K, 128K tokens). As conversations grow, you must decide which messages to keep and which to discard. The wrong strategy loses critical context; the right one preserves system instructions and recent messages while discarding older, less relevant history. Beluga AI's `ContextManager` provides two strategies: `truncate` (drop oldest messages first) and `sliding` (maintain a fixed-size window of recent messages). Both strategies preserve system messages by default because they contain instructions that the model needs throughout the conversation.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

func main() {
	ctx := context.Background()

	// Create a context manager that keeps system messages and truncates old history.
	cm := llm.NewContextManager(
		llm.WithContextStrategy("truncate"),  // Drop oldest messages first.
		llm.WithKeepSystemMessages(true),      // Never remove system messages.
	)

	// Build a long conversation.
	messages := []schema.Message{
		&schema.SystemMessage{Content: "You are a helpful assistant."},
	}
	for i := 0; i < 100; i++ {
		messages = append(messages, &schema.HumanMessage{
			Content: fmt.Sprintf("User message %d with some content", i),
		})
		messages = append(messages, &schema.AIMessage{
			Content: fmt.Sprintf("Response to message %d", i),
		})
	}

	// Trim to fit within 4096 tokens.
	trimmed, err := cm.Fit(ctx, messages, 4096)
	if err != nil {
		slog.Error("context fit failed", "error", err)
		return
	}

	fmt.Printf("Original: %d messages, Trimmed: %d messages\n",
		len(messages), len(trimmed))
}
```

**Sliding window strategy** keeps a fixed window of recent messages, discarding everything older:

```go
cm := llm.NewContextManager(
	llm.WithContextStrategy("sliding"),
	llm.WithKeepSystemMessages(true),
)
```

---

## Stream Chunks with Metadata

**Problem:** You need to stream LLM responses while tracking token usage, model info, and timing.

**Solution:** Use `model.Stream()` and inspect `schema.StreamChunk` fields for metadata.

Streaming responses is critical for perceived responsiveness, but you also need metadata (token counts, finish reasons, timing) for cost tracking and debugging. Beluga AI's streaming interface provides both: text deltas arrive as they're generated, while metadata fields like `FinishReason` and tool calls appear on specific chunks. The `iter.Seq2[StreamChunk, error]` pattern lets you process chunks with a simple `for range` loop, and errors propagate naturally through the second return value.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	_ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
	ctx := context.Background()

	model, err := llm.New("openai", llm.ProviderConfig{
		APIKey: "your-api-key",
		Model:  "gpt-4o",
	})
	if err != nil {
		slog.Error("model creation failed", "error", err)
		return
	}

	messages := []schema.Message{
		&schema.HumanMessage{Content: "Explain Go interfaces in one paragraph."},
	}

	start := time.Now()
	var totalChunks int
	var firstChunkTime time.Duration

	for chunk, err := range model.Stream(ctx, messages) {
		if err != nil {
			slog.Error("stream error", "error", err)
			break
		}

		totalChunks++
		if totalChunks == 1 {
			firstChunkTime = time.Since(start)
		}

		// Print the text delta as it arrives.
		if chunk.Delta != "" {
			fmt.Print(chunk.Delta)
		}

		// Check for tool calls in the stream.
		for _, tc := range chunk.ToolCalls {
			fmt.Printf("\n[tool call: %s]\n", tc.Name)
		}

		// Final chunk may contain usage metadata.
		if chunk.FinishReason != "" {
			fmt.Printf("\n\nFinish reason: %s\n", chunk.FinishReason)
		}
	}

	fmt.Printf("TTFT: %v, Chunks: %d, Total: %v\n",
		firstChunkTime, totalChunks, time.Since(start))
}
```

---

## Handle Streaming Tool Calls

**Problem:** During streaming, the model produces tool calls that arrive incrementally across multiple chunks. You need to accumulate them and execute tools mid-stream.

**Solution:** Collect tool call fragments, then execute when complete.

Tool calls in streaming responses arrive in fragments: the first chunk might contain the tool name and ID, while subsequent chunks contain pieces of the arguments JSON. You cannot execute a tool until you have the complete arguments. This recipe accumulates tool call fragments by ID, and when the stream's `FinishReason` is `"tool_calls"`, all accumulated calls are executed. This pattern lets the agent loop continue with tool results while keeping the streaming experience smooth for the user.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// StreamingToolHandler accumulates tool calls from streaming chunks.
type StreamingToolHandler struct {
	registry *tool.Registry
	pending  map[string]*schema.ToolCall // id -> accumulated call
}

func NewStreamingToolHandler(reg *tool.Registry) *StreamingToolHandler {
	return &StreamingToolHandler{
		registry: reg,
		pending:  make(map[string]*schema.ToolCall),
	}
}

// HandleChunk processes a stream chunk and executes completed tool calls.
func (h *StreamingToolHandler) HandleChunk(ctx context.Context, chunk schema.StreamChunk) ([]*tool.Result, error) {
	var results []*tool.Result

	for _, tc := range chunk.ToolCalls {
		if tc.ID != "" {
			// New tool call -- store it.
			h.pending[tc.ID] = &tc
		}
	}

	// When finish_reason is "tool_calls", execute all pending tools.
	if chunk.FinishReason == "tool_calls" {
		for id, tc := range h.pending {
			t, ok := h.registry.Get(tc.Name)
			if !ok {
				slog.Warn("unknown tool", "name", tc.Name)
				continue
			}

			result, err := t.Execute(ctx, tc.Arguments)
			if err != nil {
				results = append(results, tool.ErrorResult(err))
			} else {
				results = append(results, result)
			}
			delete(h.pending, id)
		}
	}

	return results, nil
}

func main() {
	ctx := context.Background()

	reg := tool.NewRegistry()
	type CalcInput struct {
		Expression string `json:"expression" description:"Math expression" required:"true"`
	}
	reg.Add(tool.NewFuncTool("calculate", "Evaluate math",
		func(ctx context.Context, input CalcInput) (*tool.Result, error) {
			return tool.TextResult("42"), nil
		},
	))

	handler := NewStreamingToolHandler(reg)

	model, err := llm.New("openai", llm.ProviderConfig{
		APIKey: "your-api-key",
		Model:  "gpt-4o",
	})
	if err != nil {
		slog.Error("failed to create model", "error", err)
		return
	}

	defs := make([]schema.ToolDefinition, 0)
	for _, t := range reg.All() {
		defs = append(defs, tool.ToDefinition(t))
	}
	boundModel := model.BindTools(defs)

	msgs := []schema.Message{
		&schema.HumanMessage{Content: "What is 6 * 7?"},
	}

	for chunk, err := range boundModel.Stream(ctx, msgs) {
		if err != nil {
			slog.Error("stream error", "error", err)
			break
		}

		if chunk.Delta != "" {
			fmt.Print(chunk.Delta)
		}

		results, err := handler.HandleChunk(ctx, chunk)
		if err != nil {
			slog.Error("tool handler error", "error", err)
			break
		}
		for _, r := range results {
			data, _ := json.Marshal(r.Content)
			fmt.Printf("\n[tool result: %s]\n", data)
		}
	}
}
```

---

## Count Tokens Without Performance Impact

**Problem:** You need to know token counts for cost tracking or context management, but counting tokens on every request adds latency.

**Solution:** Use the `llm.Tokenizer` interface and compose it with hooks for lazy counting.

Accurate token counting requires a tokenizer specific to the model being used (different models tokenize text differently). However, running a full tokenizer on every request adds latency that is disproportionate to the value it provides. This recipe uses hooks to count tokens inline with LLM calls -- the hook fires before and after each generation, counting tokens when they're already in memory. For most use cases, the `SimpleTokenizer` (whitespace-based estimation) is sufficient; switch to a provider-specific tokenizer only when you need exact counts for billing.

```go
package main

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// TokenTracker counts tokens asynchronously via hooks.
type TokenTracker struct {
	tokenizer    llm.Tokenizer
	inputTokens  atomic.Int64
	outputTokens atomic.Int64
}

func NewTokenTracker(t llm.Tokenizer) *TokenTracker {
	return &TokenTracker{tokenizer: t}
}

// Hooks returns LLM hooks that track token usage.
func (t *TokenTracker) Hooks() llm.Hooks {
	return llm.Hooks{
		BeforeGenerate: func(ctx context.Context, msgs []schema.Message) error {
			count := 0
			for _, msg := range msgs {
				count += t.tokenizer.Count(msg.GetContent())
			}
			t.inputTokens.Add(int64(count))
			return nil
		},
		AfterGenerate: func(ctx context.Context, resp *schema.AIMessage, err error) {
			if resp != nil {
				count := t.tokenizer.Count(resp.Content)
				t.outputTokens.Add(int64(count))
			}
		},
	}
}

// Stats returns cumulative token counts.
func (t *TokenTracker) Stats() (input, output int64) {
	return t.inputTokens.Load(), t.outputTokens.Load()
}

func main() {
	tracker := NewTokenTracker(&llm.SimpleTokenizer{})

	// Apply the tracker as middleware hooks.
	// model = llm.ApplyMiddleware(model, llm.WithHooks(tracker.Hooks()))

	// After several requests:
	input, output := tracker.Stats()
	fmt.Printf("Total tokens -- Input: %d, Output: %d\n", input, output)
}
```

**Key decisions:**
- `SimpleTokenizer` splits on whitespace (~4 chars/token heuristic). For accurate counts, use a provider-specific tokenizer.
- `atomic.Int64` ensures thread-safe counting with zero contention, which is important when multiple goroutines share a single tracker.
- Hooks run inline but counting is cheap compared to network latency, so the overhead is negligible.

---

## Robust LLM Error Handling with Retries

**Problem:** LLM API calls fail intermittently due to rate limits, timeouts, or provider outages. You need automatic retry with intelligent backoff.

**Solution:** Combine `resilience.Retry` with `core.IsRetryable` to handle transient failures.

LLM APIs are external services that fail in predictable ways: rate limits (HTTP 429), server errors (HTTP 5xx), and network timeouts. The key to robust error handling is classifying errors before deciding how to respond. Retrying an authentication error wastes API calls and delays the real fix, while not retrying a rate limit error means dropping a valid request. Beluga AI's `core.IsRetryable()` function checks the error's `ErrorCode` against known retryable categories, and the `resilience.RetryPolicy` handles exponential backoff with jitter to prevent thundering herd problems when many clients retry simultaneously.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/resilience"
	"github.com/lookatitude/beluga-ai/schema"
)

func generateWithRetry(ctx context.Context, model llm.ChatModel, msgs []schema.Message) (*schema.AIMessage, error) {
	policy := resilience.RetryPolicy{
		MaxAttempts:    5,
		InitialBackoff: 500 * time.Millisecond,
		MaxBackoff:     30 * time.Second,
		BackoffFactor:  2.0,
		Jitter:         true, // Prevents thundering herd.
		RetryableErrors: []core.ErrorCode{
			core.ErrRateLimit,
			core.ErrTimeout,
			core.ErrProviderDown,
		},
	}

	return resilience.Retry(ctx, policy, func(ctx context.Context) (*schema.AIMessage, error) {
		return model.Generate(ctx, msgs)
	})
}

func main() {
	ctx := context.Background()

	model, err := llm.New("openai", llm.ProviderConfig{
		APIKey: "your-api-key",
		Model:  "gpt-4o",
	})
	if err != nil {
		slog.Error("model creation failed", "error", err)
		return
	}

	msgs := []schema.Message{
		&schema.HumanMessage{Content: "Hello!"},
	}

	resp, err := generateWithRetry(ctx, model, msgs)
	if err != nil {
		slog.Error("all retries exhausted", "error", err)
		return
	}

	fmt.Println(resp.Content)
}
```

---

## Multi-Provider Fallback Chain

**Problem:** You want to try a primary LLM provider and fall back to alternatives if it fails.

**Solution:** Use `llm.Router` with a `FallbackStrategy`.

Provider outages are inevitable. A fallback chain ensures your application stays available even when a single provider goes down. The `llm.Router` with `FallbackStrategy` tries providers in order, moving to the next only when the current one returns an error. This is different from a load balancer -- it has a clear preference order (primary, secondary, tertiary) and only uses fallbacks when necessary. This matters for cost optimization: you might prefer a cheaper provider as primary and use a more expensive one only as a backup.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	_ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
	_ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
	ctx := context.Background()

	// Create primary and fallback models.
	primary, err := llm.New("openai", llm.ProviderConfig{
		APIKey: "openai-key",
		Model:  "gpt-4o",
	})
	if err != nil {
		slog.Error("primary model failed", "error", err)
		return
	}

	fallback, err := llm.New("anthropic", llm.ProviderConfig{
		APIKey: "anthropic-key",
		Model:  "claude-sonnet-4-5-20250929",
	})
	if err != nil {
		slog.Error("fallback model failed", "error", err)
		return
	}

	// Router with fallback strategy tries models in order.
	router := llm.NewRouter(
		llm.WithModels(primary, fallback),
		llm.WithStrategy(&llm.Fallback{}),
	)

	msgs := []schema.Message{
		&schema.HumanMessage{Content: "Hello!"},
	}

	resp, err := router.Generate(ctx, msgs)
	if err != nil {
		slog.Error("all providers failed", "error", err)
		return
	}

	fmt.Println(resp.Content)
}
```

---

## Round-Robin Load Balancing

**Problem:** You have multiple API keys or providers and want to distribute load evenly.

**Solution:** Use the default `RoundRobin` strategy with the router.

When you have multiple API keys for the same provider, or multiple providers with similar capabilities, round-robin distribution prevents any single key from hitting rate limits. The `RoundRobin` strategy rotates through models on each call, spreading the load evenly. This is the default router strategy because it provides the simplest form of load balancing without requiring health checking or latency measurement.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	_ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
	ctx := context.Background()

	// Create multiple model instances with different API keys.
	models := make([]llm.ChatModel, 0)
	for _, key := range []string{"key-1", "key-2", "key-3"} {
		m, err := llm.New("openai", llm.ProviderConfig{
			APIKey: key,
			Model:  "gpt-4o",
		})
		if err != nil {
			slog.Error("model creation failed", "error", err)
			return
		}
		models = append(models, m)
	}

	// RoundRobin is the default strategy -- rotates through models.
	router := llm.NewRouter(llm.WithModels(models...))

	msgs := []schema.Message{
		&schema.HumanMessage{Content: "Hello!"},
	}

	// Each call goes to the next model in rotation.
	for i := 0; i < 6; i++ {
		resp, err := router.Generate(ctx, msgs)
		if err != nil {
			slog.Error("generate failed", "error", err)
			continue
		}
		fmt.Printf("Request %d (model: %s): %s\n", i, resp.Model, resp.Content)
	}
}
```

---

## LLM Middleware Composition

**Problem:** You need to add logging, caching, rate limiting, and other cross-cutting concerns to LLM calls without modifying provider code.

**Solution:** Compose middleware using `llm.ApplyMiddleware`.

Middleware follows the `func(ChatModel) ChatModel` pattern: each middleware wraps the model and adds behavior around `Generate` and `Stream` calls. `ApplyMiddleware` applies middleware right-to-left, so the first argument becomes the outermost wrapper. In the example below, logging runs first (outermost), then hooks fire before/after the actual model call (innermost). This layering is important because you want logging to capture hook errors, not the other way around.

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	model, _ := llm.New("openai", llm.ProviderConfig{
		APIKey: "your-key",
		Model:  "gpt-4o",
	})

	// Stack middleware: logging wraps the hooks which wrap the model.
	// ApplyMiddleware applies in order so the first middleware is outermost.
	model = llm.ApplyMiddleware(model,
		llm.WithLogging(logger),
		llm.WithHooks(llm.Hooks{
			BeforeGenerate: func(ctx context.Context, msgs []schema.Message) error {
				slog.Info("generating", "message_count", len(msgs))
				return nil
			},
			OnError: func(ctx context.Context, err error) error {
				slog.Error("LLM error", "error", err)
				return err // Propagate the error.
			},
		}),
	)

	// Use the wrapped model as normal -- middleware is transparent.
	resp, _ := model.Generate(context.Background(), []schema.Message{
		&schema.HumanMessage{Content: "Hello!"},
	})
	if resp != nil {
		slog.Info("response", "content", resp.Content)
	}
}
```

**Key decisions:**
- Middleware is `func(ChatModel) ChatModel` -- simple to write and compose. Any function matching this signature is a valid middleware.
- `ApplyMiddleware` applies right-to-left so the first argument is outermost. This matches the visual reading order: what you see first is what runs first.
- Each middleware wraps both `Generate` and `Stream` transparently, so streaming gets the same cross-cutting behavior as non-streaming calls.

---

## Structured Output Parsing

**Problem:** You need the LLM to return data in a specific Go struct format, not free-form text.

**Solution:** Use `llm.StructuredOutput` to constrain the model's response to a JSON schema derived from a Go type.

Free-form text is difficult to parse reliably. Structured output solves this by constraining the model to produce JSON that matches a schema derived from your Go struct. The `llm.StructuredOutput[T]` generic function handles schema generation, model instruction, response parsing, and type validation in a single call. This is more reliable than parsing text with regex because the model is instructed at the API level to produce valid JSON matching the schema, and the response is deserialized into your typed struct with proper error handling.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

type MovieReview struct {
	Title     string  `json:"title" description:"Movie title"`
	Rating    float64 `json:"rating" description:"Rating from 1.0 to 10.0"`
	Pros      []string `json:"pros" description:"Positive aspects"`
	Cons      []string `json:"cons" description:"Negative aspects"`
	Recommend bool    `json:"recommend" description:"Would you recommend this movie?"`
}

func main() {
	ctx := context.Background()

	model, err := llm.New("openai", llm.ProviderConfig{
		APIKey: "your-key",
		Model:  "gpt-4o",
	})
	if err != nil {
		slog.Error("model creation failed", "error", err)
		return
	}

	// Parse the response into a typed struct.
	review, err := llm.StructuredOutput[MovieReview](ctx, model, []schema.Message{
		&schema.HumanMessage{Content: "Review the movie 'Interstellar' (2014)"},
	})
	if err != nil {
		slog.Error("structured output failed", "error", err)
		return
	}

	fmt.Printf("Title: %s\n", review.Title)
	fmt.Printf("Rating: %.1f/10\n", review.Rating)
	fmt.Printf("Recommend: %v\n", review.Recommend)
	fmt.Printf("Pros: %v\n", review.Pros)
	fmt.Printf("Cons: %v\n", review.Cons)
}
```
