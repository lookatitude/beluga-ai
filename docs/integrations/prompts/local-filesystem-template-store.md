# Local Filesystem Template Store

Welcome, colleague! In this integration guide, we're going to create a local filesystem-based prompt template store for Beluga AI. This enables you to manage prompts as files, version control them, and load them dynamically.

## What you will build

You will create a system that loads and manages prompt templates from the local filesystem, enabling version control, easy editing, and dynamic prompt loading in your Beluga AI applications.

## Learning Objectives

- ✅ Create a filesystem prompt store
- ✅ Load prompts from files
- ✅ Manage prompt templates locally
- ✅ Understand file-based prompt patterns

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Understanding of file I/O in Go

## Step 1: Setup Directory Structure

Create a prompts directory:
```
mkdir -p prompts/templates

Create a sample prompt file `prompts/templates/rag.txt`:
Use the following pieces of context to answer the question at the end.
If you don't know the answer, just say that you don't know, don't try to make up an answer.

Context: {{.context}}

Question: {{.question}}

Answer:
```

## Step 2: Create Filesystem Store

Create a filesystem prompt loader:
```go
package main

import (
    "context"
    "fmt"
    "io/fs"
    "os"
    "path/filepath"
    "strings"

    "github.com/lookatitude/beluga-ai/pkg/prompts"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type FilesystemPromptStore struct {
    baseDir string
    tracer  trace.Tracer
}

func NewFilesystemPromptStore(baseDir string) *FilesystemPromptStore {
    return &FilesystemPromptStore{
        baseDir: baseDir,
        tracer:  otel.Tracer("beluga.prompts.filesystem"),
    }
}

func (s *FilesystemPromptStore) LoadTemplate(ctx context.Context, name string) (prompts.PromptTemplate, error) {
    ctx, span := s.tracer.Start(ctx, "filesystem.load_template",
        trace.WithAttributes(attribute.String("template_name", name)),
    )
    defer span.End()
    
    // Construct file path
    filePath := filepath.Join(s.baseDir, name+".txt")
    
    // Read file
    content, err := os.ReadFile(filePath)
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("failed to read template: %w", err)
    }
    
    // Extract variables
    variables := s.extractVariables(string(content))
    
    // Create manager
    manager, err := prompts.NewPromptManager()
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("failed to create manager: %w", err)
    }
    
    // Create template
    template, err := manager.NewStringTemplate(string(content), variables)
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("failed to create template: %w", err)
    }
    
    span.SetAttributes(
        attribute.Int("variables_count", len(variables)),
        attribute.Int("content_length", len(content)),
    )
    
    return template, nil
}

func (s *FilesystemPromptStore) extractVariables(template string) []string {
    // Extract {{.variable}} patterns
    var variables []string
    seen := make(map[string]bool)
    
    // Simple regex-like extraction
    start := 0
    for {
        idx := strings.Index(template[start:], "{{.")
        if idx == -1 {
            break
        }
        start += idx + 3
        
        end := strings.Index(template[start:], "}}")
        if end == -1 {
            break
        }
        
        varName := strings.TrimSpace(template[start : start+end])
        if !seen[varName] {
            variables = append(variables, varName)
            seen[varName] = true
        }
        start += end + 2
    }
    
    return variables
}
```

## Step 3: List Available Templates

List all templates in the directory:
```go
func (s *FilesystemPromptStore) ListTemplates(ctx context.Context) ([]string, error) {
    ctx, span := s.tracer.Start(ctx, "filesystem.list_templates")
    defer span.End()
    
    var templates []string
    
    err := filepath.WalkDir(s.baseDir, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        
        if !d.IsDir() && strings.HasSuffix(path, ".txt") {
            relPath, _ := filepath.Rel(s.baseDir, path)
            name := strings.TrimSuffix(relPath, ".txt")
            templates = append(templates, name)
        }

        

        return nil
    })
    
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("failed to walk directory: %w", err)
    }
    
    span.SetAttributes(attribute.Int("template_count", len(templates)))
text
    return templates, nil
}
```

## Step 4: Use Filesystem Store

Use the store in your application:
```go
func main() {
    ctx := context.Background()
    
    // Create store
    store := NewFilesystemPromptStore("prompts/templates")
    
    // List templates
    templates, err := store.ListTemplates(ctx)
    if err != nil {
        log.Fatalf("Failed to list templates: %v", err)
    }
    
    fmt.Printf("Available templates: %v\n", templates)
    
    // Load template
    template, err := store.LoadTemplate(ctx, "rag")
    if err != nil {
        log.Fatalf("Failed to load template: %v", err)
    }
    
    // Use template
    result, err := template.Format(ctx, map[string]any{
        "context": "Machine learning is a subset of AI.",
        "question": "What is ML?",
    })
    if err != nil {
        log.Fatalf("Failed to format: %v", err)
    }

    
    fmt.Printf("Formatted prompt:\n%s\n", result.ToString())
}
```

## Step 5: Complete Integration

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "fmt"
    "io/fs"
    "log"
    "os"
    "path/filepath"
    "strings"
    "sync"

    "github.com/lookatitude/beluga-ai/pkg/prompts"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type ProductionFilesystemStore struct {
    baseDir string
    cache   map[string]prompts.PromptTemplate
    mu      sync.RWMutex
    tracer  trace.Tracer
}

func NewProductionFilesystemStore(baseDir string) *ProductionFilesystemStore {
    return &ProductionFilesystemStore{
        baseDir: baseDir,
        cache:   make(map[string]prompts.PromptTemplate),
        tracer:  otel.Tracer("beluga.prompts.filesystem"),
    }
}

func (s *ProductionFilesystemStore) LoadTemplate(ctx context.Context, name string) (prompts.PromptTemplate, error) {
    // Check cache
    s.mu.RLock()
    if cached, ok := s.cache[name]; ok {
        s.mu.RUnlock()
        return cached, nil
    }
    s.mu.RUnlock()
    
    ctx, span := s.tracer.Start(ctx, "filesystem.load_template",
        trace.WithAttributes(attribute.String("template_name", name)),
    )
    defer span.End()
    
    // Load from file
    filePath := filepath.Join(s.baseDir, name+".txt")
    content, err := os.ReadFile(filePath)
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("failed to read: %w", err)
    }
    
    // Extract variables and create template
    variables := s.extractVariables(string(content))
    manager, _ := prompts.NewPromptManager()
    template, err := manager.NewStringTemplate(string(content), variables)
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("failed to create template: %w", err)
    }
    
    // Cache it
    s.mu.Lock()
    s.cache[name] = template
    s.mu.Unlock()
    
    span.SetAttributes(
        attribute.Int("variables", len(variables)),
        attribute.Bool("cached", false),
    )
    
    return template, nil
}

func (s *ProductionFilesystemStore) extractVariables(template string) []string {
    var variables []string
    seen := make(map[string]bool)
    start := 0
    
    for {
        idx := strings.Index(template[start:], "{{.")
        if idx == -1 {
            break
        }
        start += idx + 3
        end := strings.Index(template[start:], "}}")
        if end == -1 {
            break
        }
        varName := strings.TrimSpace(template[start : start+end])
        if !seen[varName] {
            variables = append(variables, varName)
            seen[varName] = true
        }
        start += end + 2
    }
    
    return variables
}

func main() {
    ctx := context.Background()
    
    store := NewProductionFilesystemStore("prompts/templates")
    
    template, err := store.LoadTemplate(ctx, "rag")
    if err != nil {
        log.Fatalf("Failed to load: %v", err)
    }
    
    result, err := template.Format(ctx, map[string]any{
        "context": "AI is transforming technology.",
        "question": "What is AI?",
    })
    if err != nil {
        log.Fatalf("Failed to format: %v", err)
    }

    
    fmt.Printf("Prompt: %s\n", result.ToString())
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `BaseDir` | Base directory for templates | `prompts/templates` | No |
| `FileExtension` | Template file extension | `.txt` | No |
| `CacheEnabled` | Enable template caching | `true` | No |

## Common Issues

### "File not found"

**Problem**: Template file doesn't exist.

**Solution**: Verify file path and name:ls prompts/templates/
```

### "Invalid template syntax"

**Problem**: Template variables not properly formatted.

**Solution**: Use `{{.variable}}` format:Context: {{.context}}
Question: {{.question}}
```

## Production Considerations

When using filesystem store in production:

- **Version control**: Use Git for prompt templates
- **Caching**: Cache loaded templates for performance
- **Hot reload**: Implement file watching for development
- **Validation**: Validate templates on load
- **Security**: Restrict file access permissions

## Next Steps

Congratulations! You've created a filesystem prompt store. Next, learn how to:

- **[LangChain Hub Loading](./langchain-hub-loading.md)** - Load from LangChain Hub
- **[Prompts Package Documentation](../../api/packages/prompts.md)** - Deep dive into prompts package
- **[Prompt Templates Guide](../../guides/)** - Advanced prompt patterns

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
