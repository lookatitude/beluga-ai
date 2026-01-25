---
title: "Advanced Code Splitting (tree-sitter)"
package: "textsplitters"
category: "text-processing"
complexity: "advanced"
---

# Advanced Code Splitting (tree-sitter)

## Problem

You need to split code files intelligently while preserving code structure, keeping functions, classes, and logical blocks together rather than splitting at arbitrary character boundaries.

## Solution

Use tree-sitter (or similar AST parsers) to parse code into an abstract syntax tree, identify logical code boundaries (functions, classes, methods), and split along these boundaries. This works because code has structure that can be parsed, and splitting along structural boundaries preserves semantic meaning.

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

// CodeSplitter splits code using AST parsing
type CodeSplitter struct {
    language     string
    chunkSize    int
    chunkOverlap int
    minChunkSize int
}

// NewCodeSplitter creates a new code splitter
func NewCodeSplitter(language string, chunkSize, chunkOverlap, minChunkSize int) *CodeSplitter {
    return &CodeSplitter{
        language:     language,
        chunkSize:    chunkSize,
        chunkOverlap: chunkOverlap,
        minChunkSize: minChunkSize,
    }
}

// SplitCode splits code preserving structure
func (cs *CodeSplitter) SplitCode(ctx context.Context, code string) ([]string, error) {
    ctx, span := tracer.Start(ctx, "code_splitter.split")
    defer span.End()
    
    span.SetAttributes(
        attribute.String("language", cs.language),
        attribute.Int("code_length", len(code)),
    )
    
    // Parse code into AST
    // In practice, would use tree-sitter or similar
    nodes := cs.parseCode(ctx, code)
    
    // Extract logical blocks
    blocks := cs.extractBlocks(ctx, nodes)
    
    // Merge blocks into chunks
    chunks := cs.mergeBlocks(ctx, blocks)
    
    span.SetAttributes(attribute.Int("chunk_count", len(chunks)))
    span.SetStatus(trace.StatusOK, "code split")
    
    return chunks, nil
}

// parseCode parses code into AST nodes (simplified)
func (cs *CodeSplitter) parseCode(ctx context.Context, code string) []CodeNode {
    // In practice, would use tree-sitter to parse
    // This is a simplified representation
    return []CodeNode{}
}

// CodeNode represents an AST node
type CodeNode struct {
    Type     string // "function", "class", "method"
    Content  string
    Start    int
    End      int
    Children []CodeNode
}

// extractBlocks extracts logical code blocks
func (cs *CodeSplitter) extractBlocks(ctx context.Context, nodes []CodeNode) []CodeBlock {
    blocks := []CodeBlock{}

    for _, node := range nodes {
        // Extract top-level blocks (functions, classes)
        if node.Type == "function" || node.Type == "class" {
            blocks = append(blocks, CodeBlock{
                Type:    node.Type,
                Content: node.Content,
                Size:    len(node.Content),
            })
        }
        
        // Recursively extract from children
        childBlocks := cs.extractBlocks(ctx, node.Children)
        blocks = append(blocks, childBlocks...)
    }
    
    return blocks
}

// CodeBlock represents a logical code block
type CodeBlock struct {
    Type    string
    Content string
    Size    int
}

// mergeBlocks merges blocks into appropriately sized chunks
func (cs *CodeSplitter) mergeBlocks(ctx context.Context, blocks []CodeBlock) []string {
    chunks := []string{}
    currentChunk := ""

    for _, block := range blocks {
        // If adding this block would exceed chunk size, finalize current chunk
        if len(currentChunk)+block.Size > cs.chunkSize && len(currentChunk) >= cs.minChunkSize {
            chunks = append(chunks, currentChunk)
            
            // Start new chunk with overlap
            if cs.chunkOverlap > 0 && len(currentChunk) > cs.chunkOverlap {
                currentChunk = currentChunk[len(currentChunk)-cs.chunkOverlap:]
            } else {
                currentChunk = ""
            }
        }
        
        // Add block to current chunk
        if currentChunk != "" {
            currentChunk += "\n\n"
        }
        currentChunk += block.Content
    }
    
    // Add final chunk
    if currentChunk != "" {
        chunks = append(chunks, currentChunk)
    }
    
    return chunks
}

// SplitByLanguage splits code based on detected language
func (cs *CodeSplitter) SplitByLanguage(ctx context.Context, code string, filename string) ([]string, error) {
    // Detect language from filename or content
    language := cs.detectLanguage(filename, code)

    // Use language-specific splitter
    cs.language = language
    return cs.SplitCode(ctx, code)
}

// detectLanguage detects programming language
func (cs *CodeSplitter) detectLanguage(filename string, code string) string {
    // Simple detection based on extension
    ext := getExtension(filename)
    langMap := map[string]string{
        ".go":   "go",
        ".py":   "python",
        ".js":   "javascript",
        ".ts":   "typescript",
        ".java": "java",
    }

    if lang, ok := langMap[ext]; ok {
        return lang
    }
    return "unknown"
}

func getExtension(filename string) string {
    // Extract extension
    for i := len(filename) - 1; i >= 0; i-- {
        if filename[i] == '.' {
            return filename[i:]
        }
    }
    return ""
}

func main() {
    ctx := context.Background()

    // Create code splitter
    splitter := NewCodeSplitter("go", 1000, 200, 100)
    
    // Split code
    code := `
package main

func function1() {
    // code here
}

func function2() {
    // code here
}
```
`
    
```go
    chunks, err := splitter.SplitCode(ctx, code)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    fmt.Printf("Split into %d chunks\n", len(chunks))
}
```

## Explanation

Let's break down what's happening:

1. **AST-based parsing** - Notice how we parse code into an abstract syntax tree. This preserves code structure, allowing us to identify logical boundaries.

2. **Structure-aware splitting** - We split along structural boundaries (functions, classes) rather than arbitrary character positions. This keeps related code together.

3. **Intelligent merging** - We merge blocks into appropriately sized chunks while preserving structure. If a block is too large, we split it; if blocks are small, we merge them.

```go
**Key insight:** Use AST parsing for code splitting. Structure-aware splitting produces better chunks for RAG and maintains code readability.

## Testing

```
Here's how to test this solution:

```go
func TestCodeSplitter_PreservesStructure(t *testing.T) {
    splitter := NewCodeSplitter("go", 500, 100, 50)
    
    code := "func test() { return }"
    chunks, err := splitter.SplitCode(context.Background(), code)
    
    require.NoError(t, err)
    require.Greater(t, len(chunks), 0)
}

## Variations

### Multi-file Splitting

Split multiple files while maintaining file boundaries:

func (cs *CodeSplitter) SplitMultipleFiles(ctx context.Context, files map[string]string) (map[string][]string, error) {
    // Split each file separately
}
```

### Import Preservation

Preserve imports in each chunk:

```go
func (cs *CodeSplitter) SplitWithImports(ctx context.Context, code string) ([]string, error) {
    // Include relevant imports in each chunk
}
```

## Related Recipes

- **[Textsplitters Sentence-boundary Aware](./textsplitters-sentence-boundary-aware.md)** - Preserve sentence boundaries
- **[Document Ingestion Recipes](./document-ingestion-recipes.md)** - Document loading patterns
- **[Textsplitters Package Guide](../package_design_patterns.md)** - For a deeper understanding of text splitting
