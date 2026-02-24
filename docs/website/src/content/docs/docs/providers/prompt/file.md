---
title: "File-Based Prompt Provider"
description: "Load versioned prompt templates from JSON files in Beluga AI. File-based prompt management with versioning, hot-reload, and template rendering in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "prompt management, file-based prompts, prompt templates, versioning, hot-reload, Go, Beluga AI"
---

The file provider implements the `prompt.PromptManager` interface by loading templates from JSON files in a directory. It supports versioned templates with automatic latest-version resolution and thread-safe concurrent access.

Choose the file provider when you want to manage prompt templates as JSON files in version control. It supports semantic versioning with automatic latest-version resolution and Go `text/template` syntax for variable interpolation. Templates are loaded once on initialization and served from memory, making it suitable for production use with no external dependencies.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/prompt/providers/file
```

## Configuration

The file provider takes a single argument: the path to a directory containing JSON template files.

## Template Format

Each JSON file in the directory defines one template version:

```json
{
    "name": "system-prompt",
    "version": "1.0.0",
    "content": "You are a {{.role}} assistant specializing in {{.domain}}.",
    "variables": {
        "role": "helpful",
        "domain": "general knowledge"
    },
    "metadata": {
        "author": "platform-team",
        "description": "Default system prompt"
    }
}
```

Templates use Go `text/template` syntax for variable interpolation. The `variables` field provides default values that can be overridden at render time.

## Basic Usage

```go
package main

import (
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/prompt/providers/file"
)

func main() {
    manager, err := file.NewFileManager("./prompts")
    if err != nil {
        log.Fatal(err)
    }

    // Render the latest version with custom variables
    msgs, err := manager.Render("system-prompt", map[string]any{
        "role":   "technical",
        "domain": "Go programming",
    })
    if err != nil {
        log.Fatal(err)
    }

    for _, msg := range msgs {
        fmt.Println(msg)
    }
}
```

## Version Management

Create multiple files for different versions of the same template:

```
prompts/
  system-prompt-v1.json    # name: "system-prompt", version: "1.0.0"
  system-prompt-v2.json    # name: "system-prompt", version: "2.0.0"
  rag-prompt-v1.json       # name: "rag-prompt", version: "1.0.0"
```

Retrieve a specific version or the latest:

```go
// Get the latest version (lexicographically highest)
tmpl, err := manager.Get("system-prompt", "")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Latest: %s v%s\n", tmpl.Name, tmpl.Version)

// Get a specific version
tmpl, err = manager.Get("system-prompt", "1.0.0")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Specific: %s v%s\n", tmpl.Name, tmpl.Version)
```

Versions are compared lexicographically. The highest version string is considered the latest.

## Listing Templates

```go
templates := manager.List()

for _, t := range templates {
    fmt.Printf("%s v%s\n", t.Name, t.Version)
}
// Output:
// rag-prompt v1.0.0
// system-prompt v2.0.0
// system-prompt v1.0.0
```

Templates are sorted by name, then by version in descending order.

## Rendering

The `Render` method retrieves the latest version, merges variables, executes the template, and returns the result as a system message:

```go
msgs, err := manager.Render("system-prompt", map[string]any{
    "role": "senior",
})
if err != nil {
    log.Fatal(err)
}
// Returns: []schema.Message with a single SystemMessage
```

Variables provided at render time override the template's default `Variables`. Any defaults not overridden are used as-is.

## Directory Structure

All `.json` files in the specified directory are loaded on initialization. Subdirectories are not scanned. Each file must contain a valid template with `name`, `version`, and `content` fields.

```
prompts/
  greeting.json
  coding-assistant.json
  rag-system-v1.json
  rag-system-v2.json
```

## Error Handling

```go
manager, err := file.NewFileManager("./prompts")
if err != nil {
    // Directory does not exist or is not readable
    log.Fatal(err)
}

tmpl, err := manager.Get("nonexistent", "")
if err != nil {
    // Template not found
    log.Fatal(err)
}

msgs, err := manager.Render("system-prompt", map[string]any{
    "invalid_var": func() {},
})
if err != nil {
    // Template rendering error (invalid variable type)
    log.Fatal(err)
}
```

The file provider validates all templates during initialization. If any JSON file in the directory contains an invalid template (missing name, missing content, or unparseable template syntax), `NewFileManager` returns an error.

## Thread Safety

The file provider is safe for concurrent use. Read operations use a read-write mutex, allowing multiple goroutines to call `Get`, `Render`, and `List` simultaneously.
