// Example: Basic text splitting
//
// This example demonstrates how to use the textsplitters package
// to split text into chunks for RAG pipelines.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/pkg/textsplitters"
)

func main() {
	ctx := context.Background()

	// Sample text to split
	longText := `This is a long document that needs to be split into smaller chunks.
It contains multiple paragraphs and sentences.

The recursive splitter will try to split at paragraph boundaries first,
then at line breaks, then at word boundaries, and finally at character level if needed.

This ensures that chunks respect natural text boundaries whenever possible.`

	// Example 1: Recursive character splitting
	fmt.Println("=== Recursive Character Splitting ===")
	recursiveSplitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
		textsplitters.WithRecursiveChunkSize(100),
		textsplitters.WithRecursiveChunkOverlap(20),
	)
	if err != nil {
		log.Fatalf("Failed to create recursive splitter: %v", err)
	}

	chunks, err := recursiveSplitter.SplitText(ctx, longText)
	if err != nil {
		log.Fatalf("Failed to split text: %v", err)
	}

	fmt.Printf("Split text into %d chunks:\n", len(chunks))
	for i, chunk := range chunks {
		fmt.Printf("  Chunk %d (%d chars): %.80s...\n", i+1, len(chunk), chunk)
	}

	// Example 2: Markdown splitting
	fmt.Println("\n=== Markdown Splitting ===")
	markdownText := `# Introduction
This is the introduction section.

## Getting Started
This section explains how to get started.

### Installation
Installation instructions go here.

## Advanced Topics
Advanced topics are covered here.`

	markdownSplitter, err := textsplitters.NewMarkdownTextSplitter(
		textsplitters.WithMarkdownChunkSize(100),
		textsplitters.WithHeadersToSplitOn("#", "##", "###"),
	)
	if err != nil {
		log.Fatalf("Failed to create markdown splitter: %v", err)
	}

	markdownChunks, err := markdownSplitter.SplitText(ctx, markdownText)
	if err != nil {
		log.Fatalf("Failed to split markdown: %v", err)
	}

	fmt.Printf("Split markdown into %d chunks:\n", len(markdownChunks))
	for i, chunk := range markdownChunks {
		fmt.Printf("  Chunk %d: %.80s...\n", i+1, chunk)
	}

	// Example 3: Using the registry
	fmt.Println("\n=== Using registry ===")
	registry := textsplitters.GetRegistry()
	fmt.Printf("Available splitters: %v\n", registry.List())

	// Create splitter via registry
	registrySplitter, err := registry.Create("recursive", map[string]any{
		"chunk_size":    150,
		"chunk_overlap": 30,
	})
	if err != nil {
		log.Fatalf("Failed to create splitter via registry: %v", err)
	}

	registryChunks, err := registrySplitter.SplitText(ctx, longText)
	if err != nil {
		log.Fatalf("Failed to split via registry: %v", err)
	}
	fmt.Printf("Split text into %d chunks via registry\n", len(registryChunks))
}
