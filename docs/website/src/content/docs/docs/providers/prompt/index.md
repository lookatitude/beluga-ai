---
title: "Prompt Providers"
description: "Prompt management providers with versioned templates and cache-optimal ordering. Manage and version prompts for LLM apps in Go with Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "prompt management, prompt templates, prompt versioning, prompt cache, Go prompts, LLM prompts, Beluga AI"
---

Beluga AI v2 includes a prompt management system for versioned templates, cache-optimal message ordering, and pluggable storage backends. The `prompt` package defines a `PromptManager` interface for template storage, a `Template` type with Go text/template rendering, and a `Builder` for constructing prompt sequences optimized for LLM prompt caching.

## PromptManager Interface

All prompt providers implement the `PromptManager` interface:

```go
type PromptManager interface {
    Get(name string, version string) (*Template, error)
    Render(name string, vars map[string]any) ([]schema.Message, error)
    List() []TemplateInfo
}
```

- `Get` retrieves a template by name and version. Pass an empty string for `version` to get the latest version.
- `Render` retrieves the template, renders it with the provided variables, and returns it as a message slice.
- `List` returns summary information for all available templates (no error return).

## Template

The `Template` type represents a versioned prompt template:

```go
type Template struct {
    Name      string            `json:"name"`
    Version   string            `json:"version"`
    Content   string            `json:"content"`
    Variables map[string]string `json:"variables"`
    Metadata  map[string]any    `json:"metadata"`
}
```

Templates use Go's `text/template` syntax for variable substitution:

```go
tmpl := &prompt.Template{
    Name:    "greeting",
    Version: "1.0.0",
    Content: "You are a {{.role}} assistant. Help users with {{.topic}}.",
    Variables: map[string]string{
        "role":  "helpful",
        "topic": "general questions",
    },
}

err := tmpl.Validate()
if err != nil {
    log.Fatal(err)
}

rendered, err := tmpl.Render(map[string]any{
    "role":  "technical",
    "topic": "Go programming",
})
if err != nil {
    log.Fatal(err)
}
// "You are a technical assistant. Help users with Go programming."
```

Variables provided to `Render` are merged with the template's default `Variables`, with render-time values taking precedence.

## Prompt Builder

The `Builder` constructs prompt message sequences in an order optimized for LLM prompt caching. Static content is placed first so the prefix can be cached across requests:

```
Slot 1: System prompt       (most static)
Slot 2: Tool definitions    (semi-static)
Slot 3: Static context      (semi-static)
Slot 4: Cache breakpoint    (explicit boundary)
Slot 5: Dynamic context     (per-session)
Slot 6: User input          (always changes)
```

```go
import (
    "github.com/lookatitude/beluga-ai/prompt"
    "github.com/lookatitude/beluga-ai/schema"
)

msgs := prompt.NewBuilder(
    prompt.WithSystemPrompt("You are a helpful coding assistant."),
    prompt.WithToolDefinitions(tools),
    prompt.WithStaticContext([]string{
        "The codebase uses Go 1.23 with the iter package.",
    }),
    prompt.WithCacheBreakpoint(),
    prompt.WithDynamicContext(conversationHistory),
    prompt.WithUserInput(schema.NewHumanMessage("How do I use iter.Seq2?")),
).Build()
```

The `Build` method returns the ordered message list, skipping any empty slots.

## Template Info

The `List` method returns `TemplateInfo` summaries:

```go
type TemplateInfo struct {
    Name     string         `json:"name"`
    Version  string         `json:"version"`
    Metadata map[string]any `json:"metadata"`
}
```

```go
templates := manager.List()

for _, t := range templates {
    fmt.Printf("%s (v%s)\n", t.Name, t.Version)
}
```

## Available Providers

| Provider | Description |
|---|---|
| [File](/docs/providers/prompt/file) | Load versioned templates from JSON files in a directory |
