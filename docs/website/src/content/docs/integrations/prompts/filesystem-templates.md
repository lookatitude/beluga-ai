---
title: Filesystem Prompt Templates
description: "Manage Beluga AI prompt templates as versioned files with caching, hot-reload, and Git version control for production deployments."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "prompt templates, filesystem store, Beluga AI, template caching, hot-reload prompts, Git versioned prompts, Go prompt management"
---

Prompt engineering is iterative. Embedding prompts as string literals in Go code means every change requires a code review, recompilation, and redeployment. Filesystem-based templates externalize prompts as plain text files that can be version-controlled in Git, reviewed by non-developers (product managers, domain experts), and hot-reloaded during development. Beluga AI's `prompt` package can load templates from the local filesystem with automatic variable extraction and in-memory caching for production performance.

## Overview

A filesystem-based prompt store provides:

- File-based prompt management with Git version control
- Automatic template variable extraction from `{{.variable}}` patterns
- In-memory caching for production performance
- Directory scanning to discover available templates

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed
- Familiarity with Go's `text/template` syntax

## Setup

### Directory Structure

Create a directory to hold your prompt templates:

```bash
mkdir -p prompts/templates
```

Create a sample prompt file at `prompts/templates/rag.txt`:

```text
Use the following pieces of context to answer the question at the end.
If you don't know the answer, just say that you don't know, don't try to make up an answer.

Context: {{.context}}

Question: {{.question}}

Answer:
```

## Usage

### Creating the Filesystem Store

Define a store that reads template files and extracts variables automatically:

```go
package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/lookatitude/beluga-ai/prompt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// FilesystemPromptStore loads prompt templates from the local filesystem.
type FilesystemPromptStore struct {
	baseDir string
	tracer  trace.Tracer
}

func NewFilesystemPromptStore(baseDir string) *FilesystemPromptStore {
	return &FilesystemPromptStore{
		baseDir: baseDir,
		tracer:  otel.Tracer("beluga.prompt.filesystem"),
	}
}

func (s *FilesystemPromptStore) LoadTemplate(ctx context.Context, name string) (prompt.PromptTemplate, error) {
	ctx, span := s.tracer.Start(ctx, "filesystem.load_template",
		trace.WithAttributes(attribute.String("template_name", name)),
	)
	defer span.End()

	filePath := filepath.Join(s.baseDir, name+".txt")

	content, err := os.ReadFile(filePath)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	variables := extractVariables(string(content))

	manager, err := prompt.NewPromptManager()
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

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

// extractVariables parses {{.variable}} patterns from a template string.
func extractVariables(template string) []string {
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
```

### Listing Available Templates

Scan the directory to discover all `.txt` template files:

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
	return templates, nil
}
```

### Using the Store

```go
func main() {
	ctx := context.Background()

	store := NewFilesystemPromptStore("prompts/templates")

	// List all available templates
	templates, err := store.ListTemplates(ctx)
	if err != nil {
		log.Fatalf("Failed to list templates: %v", err)
	}
	fmt.Printf("Available templates: %v\n", templates)

	// Load and format a template
	template, err := store.LoadTemplate(ctx, "rag")
	if err != nil {
		log.Fatalf("Failed to load template: %v", err)
	}

	result, err := template.Format(ctx, map[string]any{
		"context":  "Machine learning is a subset of AI.",
		"question": "What is ML?",
	})
	if err != nil {
		log.Fatalf("Failed to format: %v", err)
	}

	fmt.Printf("Formatted prompt:\n%s\n", result.ToString())
}
```

## Advanced Topics

### Production Store with Caching

For production use, add an in-memory cache to avoid repeated file reads:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lookatitude/beluga-ai/prompt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ProductionFilesystemStore adds in-memory caching on top of file-based loading.
type ProductionFilesystemStore struct {
	baseDir string
	cache   map[string]prompt.PromptTemplate
	mu      sync.RWMutex
	tracer  trace.Tracer
}

func NewProductionFilesystemStore(baseDir string) *ProductionFilesystemStore {
	return &ProductionFilesystemStore{
		baseDir: baseDir,
		cache:   make(map[string]prompt.PromptTemplate),
		tracer:  otel.Tracer("beluga.prompt.filesystem"),
	}
}

func (s *ProductionFilesystemStore) LoadTemplate(ctx context.Context, name string) (prompt.PromptTemplate, error) {
	// Check cache first
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

	filePath := filepath.Join(s.baseDir, name+".txt")
	content, err := os.ReadFile(filePath)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to read: %w", err)
	}

	variables := extractVariables(string(content))

	manager, err := prompt.NewPromptManager()
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	template, err := manager.NewStringTemplate(string(content), variables)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	// Store in cache
	s.mu.Lock()
	s.cache[name] = template
	s.mu.Unlock()

	span.SetAttributes(
		attribute.Int("variables", len(variables)),
		attribute.Bool("cached", false),
	)

	return template, nil
}

func main() {
	ctx := context.Background()

	store := NewProductionFilesystemStore("prompts/templates")

	template, err := store.LoadTemplate(ctx, "rag")
	if err != nil {
		log.Fatalf("Failed to load: %v", err)
	}

	result, err := template.Format(ctx, map[string]any{
		"context":  "AI is transforming technology.",
		"question": "What is AI?",
	})
	if err != nil {
		log.Fatalf("Failed to format: %v", err)
	}

	fmt.Printf("Prompt: %s\n", result.ToString())
}
```

## Configuration

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `BaseDir` | Base directory for templates | `prompts/templates` | No |
| `FileExtension` | Template file extension | `.txt` | No |
| `CacheEnabled` | Enable template caching | `true` | No |

## Troubleshooting

### File not found

Verify the template file exists at the expected path:

```bash
ls prompts/templates/
```

Template names map directly to filenames: loading `"rag"` looks for `prompts/templates/rag.txt`.

### Invalid template syntax

Template variables must use Go's `text/template` syntax with the dot prefix:

```text
Context: {{.context}}
Question: {{.question}}
```

Ensure there are no unmatched `{{` or `}}` delimiters.

## Production Considerations

- **Version control** prompt files in Git alongside your application code
- **Cache loaded templates** in memory to avoid repeated disk I/O
- **Implement file watching** during development for hot-reload without restarts
- **Validate templates** on load to catch syntax errors early
- **Restrict file access** permissions to prevent unauthorized template modification

## Related Resources

- [LangChain Hub Loading](/integrations/langchain-hub) -- Load community prompts from LangChain Hub
- [Prompt Package Reference](/api-reference/prompt) -- Prompt management API
