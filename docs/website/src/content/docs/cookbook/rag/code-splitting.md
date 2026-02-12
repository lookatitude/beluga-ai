---
title: "Code Splitting with Tree-sitter"
description: "Recipe for splitting source code into RAG chunks while preserving functions, classes, and logical structure using Tree-sitter for Go retrieval pipelines."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, code splitting, Tree-sitter, Go RAG, code chunking, codebase retrieval, AST splitting, code indexing"
---

## Problem

You need to split code files into chunks for RAG indexing while preserving code structure, keeping functions, classes, and logical blocks together rather than splitting at arbitrary character boundaries.

When indexing source code for retrieval, character-based splitting produces chunks that start mid-function or end mid-class, creating fragments that are semantically meaningless in isolation. An embedding of "the last 30 lines of function A plus the first 20 lines of function B" captures neither function's intent well. This degrades both retrieval precision (wrong chunks match queries) and generation quality (the LLM receives incomplete code context).

## Solution

Use AST parsing (such as tree-sitter) to parse code into an abstract syntax tree, identify logical code boundaries (functions, classes, methods), and split along those boundaries. AST-based splitting treats the code as structured data rather than plain text, producing chunks where each one represents a complete logical unit. This approach is language-aware: it knows that a Go function ends at its closing brace, a Python class ends at the next unindented line, and a JavaScript module boundary matters.

## Code Example

```go
package main

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("beluga.textsplitters.code_splitting")

// CodeNode represents an AST node.
type CodeNode struct {
	Type     string // "function", "class", "method"
	Content  string
	Start    int
	End      int
	Children []CodeNode
}

// CodeBlock represents a logical code block.
type CodeBlock struct {
	Type    string
	Content string
	Size    int
}

// CodeSplitter splits code using AST parsing.
type CodeSplitter struct {
	language     string
	chunkSize    int
	chunkOverlap int
	minChunkSize int
}

func NewCodeSplitter(language string, chunkSize, chunkOverlap, minChunkSize int) *CodeSplitter {
	return &CodeSplitter{
		language:     language,
		chunkSize:    chunkSize,
		chunkOverlap: chunkOverlap,
		minChunkSize: minChunkSize,
	}
}

// SplitCode splits code while preserving structure.
func (cs *CodeSplitter) SplitCode(ctx context.Context, code string) ([]string, error) {
	ctx, span := tracer.Start(ctx, "code_splitter.split")
	defer span.End()

	span.SetAttributes(
		attribute.String("language", cs.language),
		attribute.Int("code_length", len(code)),
	)

	nodes := cs.parseCode(ctx, code)
	blocks := cs.extractBlocks(ctx, nodes)
	chunks := cs.mergeBlocks(ctx, blocks)

	span.SetAttributes(attribute.Int("chunk_count", len(chunks)))
	span.SetStatus(trace.StatusOK, "code split")
	return chunks, nil
}

func (cs *CodeSplitter) parseCode(ctx context.Context, code string) []CodeNode {
	// In production, use tree-sitter to parse the AST.
	return []CodeNode{}
}

func (cs *CodeSplitter) extractBlocks(ctx context.Context, nodes []CodeNode) []CodeBlock {
	blocks := []CodeBlock{}
	for _, node := range nodes {
		if node.Type == "function" || node.Type == "class" {
			blocks = append(blocks, CodeBlock{
				Type:    node.Type,
				Content: node.Content,
				Size:    len(node.Content),
			})
		}
		childBlocks := cs.extractBlocks(ctx, node.Children)
		blocks = append(blocks, childBlocks...)
	}
	return blocks
}

func (cs *CodeSplitter) mergeBlocks(ctx context.Context, blocks []CodeBlock) []string {
	chunks := []string{}
	currentChunk := ""

	for _, block := range blocks {
		if len(currentChunk)+block.Size > cs.chunkSize && len(currentChunk) >= cs.minChunkSize {
			chunks = append(chunks, currentChunk)
			if cs.chunkOverlap > 0 && len(currentChunk) > cs.chunkOverlap {
				currentChunk = currentChunk[len(currentChunk)-cs.chunkOverlap:]
			} else {
				currentChunk = ""
			}
		}
		if currentChunk != "" {
			currentChunk += "\n\n"
		}
		currentChunk += block.Content
	}

	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}
	return chunks
}

func main() {
	ctx := context.Background()

	splitter := NewCodeSplitter("go", 1000, 200, 100)

	code := `package main

func function1() {
    // implementation
}

func function2() {
    // implementation
}
`

	chunks, err := splitter.SplitCode(ctx, code)
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	fmt.Printf("Split into %d chunks\n", len(chunks))
}
```

## Explanation

1. **AST-based parsing** -- The code is parsed into an abstract syntax tree, which identifies logical boundaries (function declarations, class definitions, method receivers) rather than splitting at arbitrary character positions. This is the key difference from text-based splitting: the splitter understands the code's structure, not just its characters.

2. **Structure-aware splitting** -- Splits occur at function, class, and method boundaries. Related code stays together, producing chunks that are semantically coherent. A chunk containing a complete function is far more useful for retrieval than one containing the end of one function and the start of another.

3. **Intelligent merging** -- Small blocks (short utility functions, constants) are merged into appropriately sized chunks to avoid creating too many tiny chunks, while large blocks (long functions) can stand alone. The overlap ensures context continuity for cases where adjacent functions are closely related. The minimum chunk size prevents degenerate single-line chunks.

## Variations

### Language Detection

Automatically detect the programming language from the file extension:

```go
func detectLanguage(filename string) string {
	ext := filepath.Ext(filename)
	langMap := map[string]string{
		".go": "go", ".py": "python", ".js": "javascript",
		".ts": "typescript", ".java": "java", ".rs": "rust",
	}
	if lang, ok := langMap[ext]; ok {
		return lang
	}
	return "unknown"
}
```

## Related Recipes

- **[Sentence-Boundary Splitting](./sentence-splitting)** -- Preserve sentence boundaries in prose text
- **[Document Ingestion](./document-ingestion)** -- Document loading patterns
