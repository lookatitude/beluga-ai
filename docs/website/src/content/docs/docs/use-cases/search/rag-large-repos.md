---
title: RAG for Large Code Repositories
description: "Build efficient code search with AST-aware splitting that respects function boundaries. Reduce chunks by 50-60% and improve retrieval."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "code search RAG, code-aware splitting, AST text splitter, large repo search, code retrieval, Beluga AI, Go, developer tools"
---

Software companies building code search systems face challenges when repositories exceed 100,000 files. Standard text splitting (fixed-size chunks or line-based splitting) does not understand code structure — it splits mid-function, separates a function signature from its body, or puts half a class in one chunk and half in another. These broken chunks produce poor embeddings because the semantic unit (a complete function or method) has been fragmented, degrading retrieval accuracy.

Code-aware splitting reduces chunk count by 50-60%, preserves semantic boundaries, and improves retrieval accuracy while cutting costs. The key technique is using AST (Abstract Syntax Tree) parsing to identify function and class boundaries, then splitting at those natural boundaries rather than at arbitrary character or line counts.

## Solution Architecture

Beluga AI's splitter package supports language-specific separators and AST-based boundary detection. The code splitter uses language parsers to extract function boundaries, applies hierarchical splitting that respects code structure, and validates chunks against token limits to optimize embedding costs.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│     Code     │───▶│   Language   │───▶│    Code      │
│  Repository  │    │   Detector   │    │   Parser     │
│  (100K+      │    │              │    │ (AST-based)  │
│   files)     │    │              │    │              │
└──────────────┘    └──────────────┘    └──────┬───────┘
                                               │
                                               ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  Embeddings  │◀───│    Token     │◀───│  Hierarchical│
│  + Vector    │    │   Validator  │    │   Splitter   │
│    Store     │    │              │    │ (Function-   │
└──────────────┘    └──────────────┘    │  Boundary)   │
                                        └──────┬───────┘
                                               │
                                               ▼
                                        ┌──────────────┐
                                        │   Metadata   │
                                        │  Enrichment  │
                                        │(File/Function│
                                        │ /Line #s)    │
                                        └──────────────┘
```

## Go Code Splitter with AST

Use Go's AST parser to extract function boundaries:

```go
package main

import (
    "context"
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
    "strings"

    "github.com/lookatitude/beluga-ai/rag/splitter"
    "github.com/lookatitude/beluga-ai/schema"
)

type GoCodeSplitter struct {
    splitter  splitter.TextSplitter
    chunkSize int
}

func NewGoCodeSplitter(chunkSize, chunkOverlap int) (*GoCodeSplitter, error) {
    // Go-specific separators that respect code structure
    separators := []string{
        "\n\nfunc ",    // Function declarations
        "\n\ntype ",    // Type declarations
        "\n\nvar ",     // Variable declarations
        "\n\nconst ",   // Constant declarations
        "\n\n// ",      // Comment blocks
        "\n\n",         // Blank lines
        "\n",           // Single lines
        " ",            // Spaces
    }

    textSplitter, err := splitter.New("recursive", &splitter.Config{
        ChunkSize:    chunkSize,
        ChunkOverlap: chunkOverlap,
        Separators:   separators,
    })
    if err != nil {
        return nil, fmt.Errorf("create splitter: %w", err)
    }

    return &GoCodeSplitter{
        splitter:  textSplitter,
        chunkSize: chunkSize,
    }, nil
}

func (g *GoCodeSplitter) SplitCode(ctx context.Context, source, filePath string) ([]schema.Document, error) {
    // Parse Go AST to extract function boundaries
    fset := token.NewFileSet()
    node, err := parser.ParseFile(fset, filePath, source, parser.ParseComments)
    if err != nil {
        // Fallback to basic splitting if parsing fails
        return g.splitter.SplitText(ctx, source)
    }

    // Extract function boundaries and content
    functions := g.extractFunctions(fset, node, source)

    // Split at function boundaries when possible
    chunks := []schema.Document{}
    currentChunk := ""
    currentLine := 1

    for _, fn := range functions {
        // If adding this function exceeds chunk size, finalize current chunk
        if len(currentChunk)+len(fn.Content) > g.chunkSize && currentChunk != "" {
            doc := schema.Document{
                Content: currentChunk,
                Metadata: map[string]interface{}{
                    "source":      filePath,
                    "language":    "go",
                    "start_line":  currentLine,
                    "end_line":    fn.StartLine - 1,
                },
            }
            chunks = append(chunks, doc)
            currentChunk = ""
            currentLine = fn.StartLine
        }

        currentChunk += fn.Content + "\n\n"
    }

    // Add remaining code
    if currentChunk != "" {
        doc := schema.Document{
            Content: currentChunk,
            Metadata: map[string]interface{}{
                "source":     filePath,
                "language":   "go",
                "start_line": currentLine,
            },
        }
        chunks = append(chunks, doc)
    }

    return chunks, nil
}

type Function struct {
    Name      string
    Content   string
    StartLine int
    EndLine   int
}

func (g *GoCodeSplitter) extractFunctions(fset *token.FileSet, node *ast.File, source string) []Function {
    functions := []Function{}

    ast.Inspect(node, func(n ast.Node) bool {
        if fn, ok := n.(*ast.FuncDecl); ok {
            start := fset.Position(fn.Pos()).Line
            end := fset.Position(fn.End()).Line
            content := extractLines(source, start, end)

            functions = append(functions, Function{
                Name:      fn.Name.Name,
                Content:   content,
                StartLine: start,
                EndLine:   end,
            })
        }
        return true
    })

    return functions
}

func extractLines(text string, start, end int) string {
    lines := strings.Split(text, "\n")
    if start < 1 || start > len(lines) {
        return ""
    }
    if end > len(lines) {
        end = len(lines)
    }
    return strings.Join(lines[start-1:end], "\n")
}
```

## Language-Specific Separators

Define separators for different programming languages:

```go
package main

import (
    "github.com/lookatitude/beluga-ai/rag/splitter"
)

func GetSeparatorsForLanguage(language string) []string {
    switch language {
    case "go":
        return []string{
            "\n\nfunc ",   // Functions
            "\n\ntype ",   // Types
            "\n\nvar ",    // Variables
            "\n\nconst ",  // Constants
            "\n\n",        // Blank lines
            "\n",          // Lines
            " ",           // Spaces
        }

    case "python":
        return []string{
            "\n\nclass ",  // Classes
            "\n\ndef ",    // Functions
            "\n\n    ",    // Indented blocks (methods)
            "\n\n",        // Blank lines
            "\n",          // Lines
            " ",           // Spaces
        }

    case "javascript", "typescript":
        return []string{
            "\n\nclass ",     // Classes
            "\n\nfunction ",  // Functions
            "\n\nconst ",     // Constants
            "\n\nlet ",       // Variables
            "\n\n",           // Blank lines
            "\n",             // Lines
            " ",              // Spaces
        }

    case "java":
        return []string{
            "\n\nclass ",     // Classes
            "\n\npublic ",    // Public members
            "\n\nprivate ",   // Private members
            "\n\nprotected ", // Protected members
            "\n\n",           // Blank lines
            "\n",             // Lines
            " ",              // Spaces
        }

    default:
        // Generic code separators
        return []string{
            "\n\n",  // Blank lines
            "\n{",   // Opening braces
            "\n}",   // Closing braces
            "\n",    // Lines
            " ",     // Spaces
        }
    }
}

func CreateCodeSplitter(language string, chunkSize, chunkOverlap int) (splitter.TextSplitter, error) {
    separators := GetSeparatorsForLanguage(language)

    return splitter.New("recursive", &splitter.Config{
        ChunkSize:    chunkSize,
        ChunkOverlap: chunkOverlap,
        Separators:   separators,
    })
}
```

## Token-Aware Chunk Optimization

Validate chunks against embedding model token limits:

```go
package main

import (
    "context"

    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type TokenAwareOptimizer struct {
    tokenizer func(string) int
    maxTokens int
    tracer    trace.Tracer
}

func NewTokenAwareOptimizer(maxTokens int) *TokenAwareOptimizer {
    return &TokenAwareOptimizer{
        tokenizer: SimpleTokenCounter,
        maxTokens: maxTokens,
    }
}

func (o *TokenAwareOptimizer) OptimizeChunks(ctx context.Context, chunks []schema.Document) ([]schema.Document, error) {
    ctx, span := o.tracer.Start(ctx, "chunk.optimize")
    defer span.End()

    span.SetAttributes(attribute.Int("chunks.original", len(chunks)))

    optimized := []schema.Document{}

    for _, chunk := range chunks {
        tokenCount := o.tokenizer(chunk.Content)

        if tokenCount <= o.maxTokens {
            // Chunk within limits
            optimized = append(optimized, chunk)
        } else {
            // Chunk exceeds limits, split further
            subChunks, err := o.splitOversizedChunk(ctx, chunk)
            if err != nil {
                continue
            }
            optimized = append(optimized, subChunks...)
        }
    }

    span.SetAttributes(attribute.Int("chunks.optimized", len(optimized)))

    return optimized, nil
}

func (o *TokenAwareOptimizer) splitOversizedChunk(ctx context.Context, chunk schema.Document) ([]schema.Document, error) {
    // Estimate characters per token (roughly 4 chars per token)
    targetSize := o.maxTokens * 4

    splitter, _ := splitter.New("recursive", &splitter.Config{
        ChunkSize:    targetSize,
        ChunkOverlap: targetSize / 10,
    })

    subChunks, err := splitter.SplitDocuments(ctx, []schema.Document{chunk})
    if err != nil {
        return nil, err
    }

    return subChunks, nil
}

func SimpleTokenCounter(text string) int {
    // Estimate: 4 characters per token (adjust based on your tokenizer)
    return len(text) / 4
}
```

## Language Detection

Identify file language before splitting:

```go
package main

import (
    "path/filepath"
    "strings"
)

type LanguageDetector struct {
    extensions map[string]string
}

func NewLanguageDetector() *LanguageDetector {
    return &LanguageDetector{
        extensions: map[string]string{
            ".go":   "go",
            ".py":   "python",
            ".js":   "javascript",
            ".ts":   "typescript",
            ".jsx":  "javascript",
            ".tsx":  "typescript",
            ".java": "java",
            ".c":    "c",
            ".cpp":  "cpp",
            ".h":    "c",
            ".hpp":  "cpp",
            ".cs":   "csharp",
            ".rb":   "ruby",
            ".php":  "php",
            ".rs":   "rust",
        },
    }
}

func (d *LanguageDetector) DetectLanguage(filePath string) string {
    ext := strings.ToLower(filepath.Ext(filePath))
    if lang, ok := d.extensions[ext]; ok {
        return lang
    }
    return "unknown"
}

func (d *LanguageDetector) SplitCodeFile(ctx context.Context, filePath, content string) ([]schema.Document, error) {
    language := d.DetectLanguage(filePath)

    // Use language-specific splitter
    switch language {
    case "go":
        splitter, _ := NewGoCodeSplitter(1500, 200)
        return splitter.SplitCode(ctx, content, filePath)

    case "python":
        // Use Python-specific splitter with indentation awareness
        splitter, _ := CreateCodeSplitter("python", 1500, 200)
        return splitter.SplitText(ctx, content)

    default:
        // Use generic code splitter
        splitter, _ := CreateCodeSplitter("generic", 1500, 200)
        return splitter.SplitText(ctx, content)
    }
}
```

## Batch Processing Pipeline

Process large repositories efficiently:

```go
package main

import (
    "context"
    "io/fs"
    "path/filepath"
    "sync"
)

type CodeProcessor struct {
    detector  *LanguageDetector
    optimizer *TokenAwareOptimizer
    tracer    trace.Tracer
}

func (c *CodeProcessor) ProcessRepository(ctx context.Context, repoPath string) error {
    var wg sync.WaitGroup
    fileChan := make(chan string, 100)

    // Start worker pool
    numWorkers := 10
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for filePath := range fileChan {
                if err := c.processFile(ctx, filePath); err != nil {
                    // Log error but continue
                    continue
                }
            }
        }()
    }

    // Walk repository and send files to workers
    err := filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }

        if d.IsDir() {
            // Skip common directories
            if d.Name() == ".git" || d.Name() == "node_modules" || d.Name() == "vendor" {
                return filepath.SkipDir
            }
            return nil
        }

        // Only process code files
        if c.detector.DetectLanguage(path) != "unknown" {
            fileChan <- path
        }

        return nil
    })

    close(fileChan)
    wg.Wait()

    return err
}

func (c *CodeProcessor) processFile(ctx context.Context, filePath string) error {
    ctx, span := c.tracer.Start(ctx, "code.process.file")
    defer span.End()

    span.SetAttributes(
        attribute.String("file.path", filePath),
        attribute.String("file.language", c.detector.DetectLanguage(filePath)),
    )

    // Read file content
    content, err := os.ReadFile(filePath)
    if err != nil {
        return err
    }

    // Split into chunks
    chunks, err := c.detector.SplitCodeFile(ctx, filePath, string(content))
    if err != nil {
        return err
    }

    // Optimize chunks for token limits
    optimized, err := c.optimizer.OptimizeChunks(ctx, chunks)
    if err != nil {
        return err
    }

    // Store in vector database
    return c.storeChunks(ctx, optimized)
}
```

## Production Considerations

### Observability

Track splitting metrics and boundary preservation:

```go
import (
    "github.com/lookatitude/beluga-ai/o11y"
    "go.opentelemetry.io/otel/attribute"
)

func (c *CodeProcessor) ProcessWithObservability(ctx context.Context, repoPath string) error {
    ctx, span := c.tracer.Start(ctx, "code.process.repository")
    defer span.End()

    span.SetAttributes(
        attribute.String("repo.path", repoPath),
    )

    start := time.Now()
    err := c.ProcessRepository(ctx, repoPath)
    duration := time.Since(start)

    if err != nil {
        span.RecordError(err)
        return err
    }

    span.SetAttributes(
        attribute.Float64("duration.seconds", duration.Seconds()),
    )

    meter.RecordHistogram(ctx, "code.process.duration", duration.Seconds())
    meter.IncrementCounter(ctx, "repositories.processed")

    return nil
}
```

### Caching

Cache parsed ASTs to avoid repeated parsing:

```go
import "github.com/lookatitude/beluga-ai/cache"

type CachedGoCodeSplitter struct {
    GoCodeSplitter
    cache cache.Cache
}

func (c *CachedGoCodeSplitter) SplitCode(ctx context.Context, source, filePath string) ([]schema.Document, error) {
    cacheKey := fmt.Sprintf("ast:%s", filePath)

    // Check cache first
    if cached, ok := c.cache.Get(ctx, cacheKey); ok {
        return cached.([]schema.Document), nil
    }

    // Parse and split
    chunks, err := c.GoCodeSplitter.SplitCode(ctx, source, filePath)
    if err != nil {
        return nil, err
    }

    // Cache for 1 hour
    c.cache.Set(ctx, cacheKey, chunks, time.Hour)

    return chunks, nil
}
```

### Quality Validation

Verify function boundary preservation:

```go
func ValidateFunctionBoundaries(chunks []schema.Document) error {
    for _, chunk := range chunks {
        // Count opening and closing braces
        opens := strings.Count(chunk.Content, "{")
        closes := strings.Count(chunk.Content, "}")

        // Warn if imbalanced (may indicate split function)
        if opens != closes {
            return fmt.Errorf("imbalanced braces in chunk from %s (line %d): opens=%d, closes=%d",
                chunk.Metadata["source"],
                chunk.Metadata["start_line"],
                opens, closes)
        }
    }
    return nil
}
```

## Related Resources

- [Text Splitter Guide](/guides/text-splitting/) for splitting strategies
- [Scientific Paper Processing](/use-cases/scientific-papers/) for academic splitting
- [RAG Pipeline Guide](/guides/rag-pipeline/) for complete RAG setup
- [Embedding Guide](/guides/embeddings/) for cost optimization
