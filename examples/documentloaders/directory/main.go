// Example: Advanced directory loading
//
// This example demonstrates advanced usage of RecursiveDirectoryLoader
// with various configuration options including depth limits, extension filtering,
// concurrency control, file size limits, and symlink following.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/pkg/documentloaders"
)

func main() {
	ctx := context.Background()

	// Example 1: Basic directory loading with depth limit
	fmt.Println("=== Example 1: Limited depth ===")
	fsys1 := os.DirFS(".")
	loader1, err := documentloaders.NewDirectoryLoader(fsys1,
		documentloaders.WithMaxDepth(1), // Only load files in current directory
	)
	if err != nil {
		log.Fatalf("Failed to create loader: %v", err)
	}

	docs1, err := loader1.Load(ctx)
	if err != nil {
		log.Fatalf("Failed to load: %v", err)
	}
	fmt.Printf("Loaded %d documents (max depth 1)\n", len(docs1))

	// Example 2: Extension filtering
	fmt.Println("\n=== Example 2: Extension filtering ===")
	loader2, err := documentloaders.NewDirectoryLoader(fsys1,
		documentloaders.WithExtensions(".go", ".md"), // Only .go and .md files
		documentloaders.WithMaxDepth(2),
	)
	if err != nil {
		log.Fatalf("Failed to create loader: %v", err)
	}

	docs2, err := loader2.Load(ctx)
	if err != nil {
		log.Fatalf("Failed to load: %v", err)
	}
	fmt.Printf("Loaded %d .go and .md files\n", len(docs2))

	// Example 3: Concurrency control
	fmt.Println("\n=== Example 3: Custom concurrency ===")
	loader3, err := documentloaders.NewDirectoryLoader(fsys1,
		documentloaders.WithConcurrency(4), // Use 4 workers
		documentloaders.WithMaxDepth(2),
	)
	if err != nil {
		log.Fatalf("Failed to create loader: %v", err)
	}

	docs3, err := loader3.Load(ctx)
	if err != nil {
		log.Fatalf("Failed to load: %v", err)
	}
	fmt.Printf("Loaded %d documents with 4 concurrent workers\n", len(docs3))

	// Example 4: File size limit
	fmt.Println("\n=== Example 4: File size limit ===")
	loader4, err := documentloaders.NewDirectoryLoader(fsys1,
		documentloaders.WithDirectoryMaxFileSize(100*1024), // Max 100KB per file
		documentloaders.WithMaxDepth(1),
	)
	if err != nil {
		log.Fatalf("Failed to create loader: %v", err)
	}

	docs4, err := loader4.Load(ctx)
	if err != nil {
		log.Fatalf("Failed to load: %v", err)
	}
	fmt.Printf("Loaded %d documents (files >100KB skipped)\n", len(docs4))

	// Example 5: Symlink following
	fmt.Println("\n=== Example 5: Symlink following ===")
	loader5, err := documentloaders.NewDirectoryLoader(fsys1,
		documentloaders.WithFollowSymlinks(true), // Follow symlinks with cycle detection
		documentloaders.WithMaxDepth(3),
	)
	if err != nil {
		log.Fatalf("Failed to create loader: %v", err)
	}

	docs5, err := loader5.Load(ctx)
	if err != nil {
		log.Fatalf("Failed to load: %v", err)
	}
	fmt.Printf("Loaded %d documents (symlinks followed)\n", len(docs5))

	// Example 6: Lazy loading (streaming)
	fmt.Println("\n=== Example 6: Lazy loading (streaming) ===")
	loader6, err := documentloaders.NewDirectoryLoader(fsys1,
		documentloaders.WithMaxDepth(1),
		documentloaders.WithExtensions(".go"),
	)
	if err != nil {
		log.Fatalf("Failed to create loader: %v", err)
	}

	ch, err := loader6.LazyLoad(ctx)
	if err != nil {
		log.Fatalf("Failed to start lazy load: %v", err)
	}

	count := 0
	for item := range ch {
		switch v := item.(type) {
		case error:
			log.Printf("Error during lazy load: %v", v)
		default:
			count++
			if count <= 3 {
				fmt.Printf("  Received document %d\n", count)
			}
		}
	}
	fmt.Printf("Streamed %d documents via LazyLoad\n", count)

	// Example 7: Combined options
	fmt.Println("\n=== Example 7: Combined options ===")
	loader7, err := documentloaders.NewDirectoryLoader(fsys1,
		documentloaders.WithMaxDepth(2),
		documentloaders.WithExtensions(".go", ".md", ".txt"),
		documentloaders.WithConcurrency(8),
		documentloaders.WithDirectoryMaxFileSize(500*1024), // 500KB limit
		documentloaders.WithFollowSymlinks(false), // Don't follow symlinks
	)
	if err != nil {
		log.Fatalf("Failed to create loader: %v", err)
	}

	docs7, err := loader7.Load(ctx)
	if err != nil {
		log.Fatalf("Failed to load: %v", err)
	}
	fmt.Printf("Loaded %d documents with combined options\n", len(docs7))
}
