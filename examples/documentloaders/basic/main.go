// Example: Basic document loading
//
// This example demonstrates how to use the documentloaders package
// to load documents from a directory and a single file.
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

	// Example 1: Load from a directory
	fmt.Println("=== Loading from directory ===")
	fsys := os.DirFS(".")
	loader, err := documentloaders.NewDirectoryLoader(fsys,
		documentloaders.WithMaxDepth(2),
		documentloaders.WithExtensions(".txt", ".md"),
		documentloaders.WithConcurrency(2),
	)
	if err != nil {
		log.Fatalf("Failed to create directory loader: %v", err)
	}

	docs, err := loader.Load(ctx)
	if err != nil {
		log.Fatalf("Failed to load documents: %v", err)
	}

	fmt.Printf("Loaded %d documents from directory\n", len(docs))
	for i, doc := range docs {
		if i >= 3 { // Show first 3
			fmt.Printf("... and %d more documents\n", len(docs)-3)
			break
		}
		fmt.Printf("  - %s (%d bytes)\n", doc.Metadata["source"], len(doc.PageContent))
	}

	// Example 2: Load a single file
	fmt.Println("\n=== Loading single file ===")
	textLoader, err := documentloaders.NewTextLoader("README.md")
	if err != nil {
		log.Printf("Note: README.md not found, skipping single file example")
	} else {
		textDocs, err := textLoader.Load(ctx)
		if err != nil {
			log.Fatalf("Failed to load file: %v", err)
		}

		fmt.Printf("Loaded %d document(s) from file\n", len(textDocs))
		if len(textDocs) > 0 {
			doc := textDocs[0]
			fmt.Printf("  - Source: %s\n", doc.Metadata["source"])
			fmt.Printf("  - Size: %s bytes\n", doc.Metadata["file_size"])
			fmt.Printf("  - Content preview: %.100s...\n", doc.PageContent)
		}
	}

	// Example 3: Using the registry
	fmt.Println("\n=== Using registry ===")
	registry := documentloaders.GetRegistry()
	fmt.Printf("Available loaders: %v\n", registry.List())

	// Create loader via registry
	registryLoader, err := registry.Create("directory", map[string]any{
		"path":       ".",
		"max_depth":  1,
		"extensions": []string{".go"},
	})
	if err != nil {
		log.Fatalf("Failed to create loader via registry: %v", err)
	}

	registryDocs, err := registryLoader.Load(ctx)
	if err != nil {
		log.Fatalf("Failed to load via registry: %v", err)
	}
	fmt.Printf("Loaded %d .go files via registry\n", len(registryDocs))
}
