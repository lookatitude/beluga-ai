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
	err := core.NewError("ExampleOperation", "example_code", fmt.Errorf("something went wrong"))
	fmt.Printf("✅ Created error: %v\n", err)
	fmt.Printf("   Operation: %s\n", err.Op())
	fmt.Printf("   Code: %s\n", err.Code())

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
	result, err := runnable.Run(ctx, "test input")
	if err != nil {
		log.Printf("⚠️  Runnable error: %v", err)
	} else {
		fmt.Printf("✅ Runnable executed: %s\n", result)
	}

	// Example 4: Options Pattern
	fmt.Println("\n📋 Example 4: Options Pattern")
	opts := []core.Option{
		core.WithOption("key1", "value1"),
		core.WithOption("key2", 42),
		core.WithOption("key3", true),
	}
	fmt.Println("✅ Created options:")
	for _, opt := range opts {
		fmt.Printf("   Option applied\n")
	}

	// Example 5: Error Wrapping
	fmt.Println("\n📋 Example 5: Error Wrapping")
	originalErr := fmt.Errorf("original error")
	wrappedErr := core.WrapError("WrapperOperation", originalErr)
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

func (r *ExampleRunnable) Run(ctx context.Context, input any) (any, error) {
	return fmt.Sprintf("Processed by %s: %v", r.name, input), nil
}
