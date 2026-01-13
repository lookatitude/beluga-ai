package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
)

func main() {
	fmt.Println("🔄 Beluga AI Core Package Usage Example")
	fmt.Println("=======================================")

	ctx := context.Background()

	// Example 1: Error Handling
	fmt.Println("\n📋 Example 1: Error Handling")
	validationErr := core.NewValidationError("something went wrong", fmt.Errorf("underlying error"))
	fmt.Printf("✅ Created error: %v\n", validationErr)
	fmt.Printf("   Type: %s\n", validationErr.Type)
	fmt.Printf("   Message: %s\n", validationErr.Message)

	// Example 2: Context with Timeout
	fmt.Println("\n📋 Example 2: Context with Timeout")
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		fmt.Println("✅ Context timeout example (would timeout after 2 seconds)")
	case <-time.After(100 * time.Millisecond):
		fmt.Println("✅ Context created successfully")
	}

	// Example 3: Runnable Interface
	fmt.Println("\n📋 Example 3: Runnable Interface")
	runnable := &ExampleRunnable{name: "test-runnable"}
	result, err := runnable.Invoke(ctx, "test input")
	if err != nil {
		log.Printf("⚠️  Runnable error: %v", err)
	} else {
		fmt.Printf("✅ Runnable executed: %v\n", result)
	}

	// Example 4: Options Pattern
	fmt.Println("\n📋 Example 4: Options Pattern")
	opts := []core.Option{
		core.WithOption("key1", "value1"),
		core.WithOption("key2", 42),
		core.WithOption("key3", true),
	}
	fmt.Println("✅ Created options:")
	for range opts {
		fmt.Printf("   Option applied\n")
	}

	// Example 5: Error Wrapping
	fmt.Println("\n📋 Example 5: Error Wrapping")
	originalErr := fmt.Errorf("original error")
	wrappedErr := core.WrapError(originalErr, "wrapper message")
	fmt.Printf("✅ Wrapped error: %v\n", wrappedErr)

	fmt.Println("\n✨ All examples completed successfully!")
	fmt.Println("\nFor more examples, see:")
	fmt.Println("  - Package documentation: pkg/core/README.md")
	fmt.Println("  - Other examples that use core utilities")
}

// ExampleRunnable implements the core.Runnable interface
type ExampleRunnable struct {
	name string
}

func (r *ExampleRunnable) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	return fmt.Sprintf("Processed by %s: %v", r.name, input), nil
}

func (r *ExampleRunnable) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := r.Invoke(ctx, input, options...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (r *ExampleRunnable) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	go func() {
		defer close(ch)
		result, err := r.Invoke(ctx, input, options...)
		if err != nil {
			ch <- err
			return
		}
		ch <- result
	}()
	return ch, nil
}
