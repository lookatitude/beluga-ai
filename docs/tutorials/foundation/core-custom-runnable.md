# Building a Custom Runnable

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll build a **Custom Runnable** from scratch. You'll see that it's not a black box—just a standard Go interface that you can implement for your own business logic.

## Learning Objectives
By the end of this tutorial, you will:
1.  Understand the three pillars of the `Runnable` interface: `Invoke`, `Batch`, and `Stream`.
2.  Build a custom component (a deterministic "Sentiment Analyzer").
3.  Learn how to support options and tracing.

## Introduction
In the Beluga AI Framework, everything that "does work" is a **Runnable**. Whether it's a massive LLM, a vector store retriever, or a simple formatting function, they all speak the same language: the `core.Runnable` interface.

This unity allows us to compose complex chains where a `Retriever` feeds into a `Prompt`, which feeds into an `LLM`, which feeds into a `Parser`.

## Why This Matters

*   **Interoperability**: By making your custom code a `Runnable`, it can instantly plug into Beluga AI's orchestration, error handling, and observation tools.
*   **Testing**: Runnables are easy to mock and test in isolation.
*   **Encapsulation**: You can wrap complex legacy logic or external APIs into a clean, standard interface.

## Prerequisites

*   Working Go environment.
*   Understanding of `context.Context`.
*   Familiarity with `pkg/core`.

## The Interface

The `core.Runnable` interface is simple yet powerful. It defines three methods:
```go
type Runnable interface {
    Invoke(ctx context.Context, input any, options ...Option) (any, error)
    Batch(ctx context.Context, inputs []any, options ...Option) ([]any, error)
    Stream(ctx context.Context, input any, options ...Option) (<-chan any, error)
}
```

Let's implement them one by one.

## Step-by-Step Implementation

We will build a **Keyword Sentiment Analyzer**. It's a simple component that takes a string input and returns "POSITIVE" if it sees the word "good" and "NEGATIVE" if it sees "bad".

### Step 1: Define the Struct

First, define your struct. It can hold configuration or state.
```go
package main

import (
	"context"
	"strings"
    "fmt"
    
	"github.com/lookatitude/beluga-ai/pkg/core"
)

// KeywordSentiment is a simple custom Runnable
type KeywordSentiment struct {
	DefaultSentiment string
}

// NewKeywordSentiment creates a new instance
func NewKeywordSentiment(defaultSentiment string) *KeywordSentiment {
	return &KeywordSentiment{
		DefaultSentiment: defaultSentiment,
	}
}
```

### Step 2: Implement Invoke

`Invoke` is for single-input, synchronous execution. It's the bread and butter of your component.
```go
func (k *KeywordSentiment) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
    // 1. Validate Input
	text, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("input must be string")
	}

    // 2. Perform Logic
	text = strings.ToLower(text)
	if strings.Contains(text, "good") {
		return "POSITIVE", nil
	}
	if strings.Contains(text, "bad") {
		return "NEGATIVE", nil
	}


	return k.DefaultSentiment, nil
}
```

### Step 3: Implement Batch

`Batch` handles a slice of inputs. While you *can* process them strictly sequentially, this method exists to allow for concurrency optimizations (e.g., calling an API in parallel).

For our simple CPU-bound logic, a sequential loop fits, but demonstrates the pattern.
```go
func (k *KeywordSentiment) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
    
	for i, input := range inputs {
        // reuse Invoke logic to ensure consistency
		res, err := k.Invoke(ctx, input, options...)
		if err != nil {
			return nil, err // Or handle partial failures depending on your design
		}
		results[i] = res
	}

    
	return results, nil
}
```

### Step 4: Implement Stream

`Stream` allows consumers to process output as it becomes available. For an LLM, this means token-by-token. For our sentiment analyzer, we compute the result instantly, so we just emit it as one chunk and close the channel.
```go
func (k *KeywordSentiment) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	// Create a buffered channel
    ch := make(chan any, 1)
	
	// Start a goroutine to process and send
	go func() {
		defer close(ch)
		
		result, err := k.Invoke(ctx, input, options...)
		if err != nil {
            // In a real stream, you might send an error object or handle it differently
			return 
		}
		
        // Respect context cancellation!
		select {
		case ch <- result:
		case <-ctx.Done():
		}
	}()


	return ch, nil
}
```

### Step 5: Putting It All Together

Now let's see our custom runnable in action.
```go
func main() {
	analyzer := NewKeywordSentiment("NEUTRAL")
	ctx := context.Background()

	// 1. Single Invoke
	res, _ := analyzer.Invoke(ctx, "This represents a good outcome")
	fmt.Printf("Invoke Result: %s\n", res)

	// 2. Batch Processing
	inputs := []any{
		"This is bad news",
		"Just a normal sentence",
		"Very good work",
	}
	batchRes, _ := analyzer.Batch(ctx, inputs)
	fmt.Printf("Batch Results: %v\n", batchRes)
    
    // 3. Streaming
    streamCh, _ := analyzer.Stream(ctx, "Another good example")
    for chunk := range streamCh {
        fmt.Printf("Stream Chunk: %v\n", chunk)
    }
}
```

## Pro-Tips

### 1. Adding Tracing
You don't need to implement OpenTelemetry tracing manually in every method. Beluga AI provides `core.RunnableWithTracing` to wrap your simple struct!
```text
go
go
// Wrap your analyzer
tracedAnalyzer := core.RunnableWithTracing(
    analyzer, 
    tracer, 
    metrics, 
    "sentiment_analyzer",
)
// Now all calls emit spans and metrics automatically!
tracedAnalyzer.Invoke(ctx, "input")
```

### 2. Handling Options
Notice the `options ...core.Option` argument? You can use this to pass execution-time override overrides.






```go
type SentimentOption func(*KeywordSentiment)

// You would need to update Invoke to check for these options
// and apply them to a temporary config object.

### 3. Type Safety
Go's empty interface `any` is flexible but dangerous. Always perform type assertions (`input.(string)`) at the very beginning of `Invoke` and return clear errors if the type is wrong.

## Troubleshooting

### "My Stream channel hangs"
Ensure you **close** the channel in a `defer` block within the goroutine. If you forget to close it, the consumer loop `for range ch` will wait forever.

### "Batch returns partial results"
The standard `Batch` signature returns `([]any, error)`. If one item fails, you generally return an error for the whole batch. However, some advanced implementations return a `BatchResult` struct that contains individual errors per item.

## Conclusion

Congratulations! You've built a fully compliant Beluga AI component. This `KeywordSentiment` struct can now be:
*   chained with an LLM,
*   traced with OpenTelemetry,
*   or swapped out for a complex classifier later without changing client code.

By adhering to the `Runnable` interface, you ensure your custom logic is a first-class citizen in the Beluga AI ecosystem.
