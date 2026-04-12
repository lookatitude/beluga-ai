# DOC-19: Prompt Management

**Audience:** Extension developers, prompt engineers integrating versioned templates with agents.
**Prerequisites:** [03 — Extensibility Patterns](./03-extensibility-patterns.md).
**Related:** [DOC-14 — Observability](./14-observability.md), [patterns/registry-factory.md](../patterns/registry-factory.md), [Reference: Providers](../reference/providers.md).

## Overview

The `prompt` package (Layer 3 — Capability) provides three complementary building blocks: versioned template management, cache-optimised prompt assembly, and OTel instrumentation.

**Versioned template management** decouples prompt content from application code. A `PromptManager` stores templates as named, versioned artefacts. The `file` provider loads them from JSON files on disk; any other backend (database, remote API, A/B-test system) can be wired in by implementing the same three-method interface and wrapping it with middleware at construction time.

**Cache-optimised prompt assembly** is the `Builder`. LLM providers (Anthropic, OpenAI) cache the longest static prefix of a conversation at a discount. The Builder enforces a fixed slot ordering — system prompt first, tool definitions second, static context third, dynamic history fourth, user input last — so that the static portion of every request is bit-for-bit identical and always falls inside the provider's cache window. An explicit `WithCacheBreakpoint()` marker lets providers that expose a cache control API know exactly where the cached prefix ends.

The package follows the same extensibility model used throughout Layer 3: a small interface, a `Middleware` type with `ApplyMiddleware`, and a `WithTracing()` middleware that instruments every operation with OTel GenAI spans. There are no hooks in this package because `PromptManager` is a read-only query surface with no mutable lifecycle events to intercept.

## Interface

**Source:** `prompt/manager.go:20-32`

```go
// PromptManager provides versioned access to prompt templates.
type PromptManager interface {
    // Get retrieves a template by name and version.
    // If version is empty, the latest version is returned.
    Get(name, version string) (*Template, error)

    // Render retrieves a template by name (latest version), renders it with
    // the given variables, and returns the result as a slice of schema.Message.
    Render(name string, vars map[string]any) ([]schema.Message, error)

    // List returns summary information for all available templates.
    List() []TemplateInfo
}
```

Three methods, all read-only. Implementations carry no mutable state beyond their backing store.

### Template

**Source:** `prompt/template.go:12-23`

```go
type Template struct {
    Name      string            `json:"name"`
    Version   string            `json:"version"`
    Content   string            `json:"content"`
    Variables map[string]string `json:"variables,omitempty"`
    Metadata  map[string]any    `json:"metadata,omitempty"`
}
```

`Content` uses Go `text/template` syntax. `Variables` holds default values; variables passed to `Render` take precedence over defaults.

Key methods on `*Template`:

| Method | Signature | What it does |
|---|---|---|
| `Validate` | `() error` | Parses the template content; returns error if name/content empty or parse fails (`prompt/template.go:26-39`) |
| `Render` | `(vars map[string]any) (string, error)` | Merges defaults with `vars`, executes the template, returns rendered string (`prompt/template.go:44-69`) |

### TemplateInfo

**Source:** `prompt/manager.go:8-15`

```go
type TemplateInfo struct {
    Name     string
    Version  string
    Metadata map[string]any
}
```

Returned by `PromptManager.List()` — a summary record without the full template body.

## Middleware

**Source:** `prompt/middleware.go:6-16`

```go
type Middleware func(PromptManager) PromptManager

func ApplyMiddleware(manager PromptManager, mws ...Middleware) PromptManager
```

`ApplyMiddleware` wraps `manager` in reverse-slice order so the first argument in `mws` is the outermost (first to execute). This is the same outside-in composition pattern used in `tool`, `llm`, and `memory`.

Usage:

```go
import (
    "github.com/lookatitude/beluga-ai/prompt"
    promptfile "github.com/lookatitude/beluga-ai/prompt/providers/file"
)

base, err := promptfile.NewFileManager("/etc/prompts")
if err != nil {
    return err
}
// ApplyMiddleware returns prompt.PromptManager, not *file.FileManager.
// Declare mgr as the interface type before wrapping.
var mgr prompt.PromptManager = base
mgr = prompt.ApplyMiddleware(mgr, prompt.WithTracing())
```

## Tracing

**Source:** `prompt/tracing.go:18-78`

`WithTracing()` returns a `Middleware` that wraps every `PromptManager` operation in an OTel span following the GenAI semantic conventions. The compile-time guard `var _ PromptManager = (*tracedManager)(nil)` (`prompt/tracing.go:78`) ensures the wrapper stays in sync with the interface.

| Span name | Attributes set |
|---|---|
| `prompt.get` | `gen_ai.operation.name`, `prompt.get.name`, `prompt.get.version` |
| `prompt.render` | `gen_ai.operation.name`, `prompt.render.name`, `prompt.render.message_count` |
| `prompt.list` | `gen_ai.operation.name`, `prompt.list.result_count` |

Errors are recorded on the span with `span.RecordError(err)` and status is set to `StatusError`. On success, status is `StatusOK`.

`WithTracing` is the only built-in middleware. Additional cross-cutting concerns (caching rendered results, metrics, rate limiting) are intended to be composed on top via the `Middleware` type.

## Hooks

The `prompt` package does not define a hooks struct. `PromptManager` is a read-only query surface — there are no mutable lifecycle points (start/end of a plan cycle, tool dispatch) that benefit from interception. If you need to observe renders at the agent level, use agent-level hooks and inspect the message list that `PromptManager.Render` returns.

## Providers

### `prompt/providers/file` — filesystem-backed manager

**Source:** `prompt/providers/file/file.go`

`FileManager` implements `PromptManager` by scanning a directory of JSON files at construction time. There is no `init()` side-effect registration: because `PromptManager` is not a named-string registry (unlike `llm` or `tool`), callers construct the provider directly and wrap it with middleware.

```go
// NewFileManager loads all .json files in dir.
// Returns an error if the directory cannot be read or any file fails validation.
func NewFileManager(dir string) (*FileManager, error)
```

Each JSON file must unmarshal into `prompt.Template` and pass `Validate()`. Multiple versions of the same template name are supported by placing multiple files with the same `name` field and different `version` fields. `Get` with an empty version string returns the lexicographically highest version.

**Template file format:**

```json
{
    "name": "system-assistant",
    "version": "1.2.0",
    "content": "You are {{.role}}. {{.instructions}}",
    "variables": {
        "role": "a helpful assistant",
        "instructions": "Be concise."
    }
}
```

## Cache interaction

The `prompt` package does not import `cache/` — it does not perform LLM-level caching itself. Instead, `Builder` produces a message ordering that maximises the probability of a *provider-native* cache hit:

```
System prompt  (slot 1)  ← most static → cached by provider
Tool defs      (slot 2)  ← semi-static →
Static context (slot 3)  ← semi-static →
── cache_breakpoint ──   ← explicit boundary via WithCacheBreakpoint()
Dynamic history (slot 4) ← per-session →
User input     (slot 5)  ← always changes →
```

The breakpoint is a `schema.SystemMessage` carrying `Metadata["cache_breakpoint"] = true` (`prompt/builder.go:116-119`). LLM providers that support explicit cache control (Anthropic's `cache_control` block, OpenAI's cached prefix API) can inspect this metadata to know where to anchor the cache boundary.

The `cache/` package (Layer 3) implements exact-match, semantic, and prompt caches that wrap the LLM layer. Those two concerns are orthogonal: `Builder` orders content for provider-native prefix caching; `cache/` deduplicates full prompt+response pairs at the framework layer.

### Builder API

**Source:** `prompt/builder.go`

| Function / Method | Signature | What it does |
|---|---|---|
| `NewBuilder` | `(opts ...BuilderOption) *Builder` | Creates a Builder with the given options applied |
| `WithSystemPrompt` | `(prompt string) BuilderOption` | Slot 1: static system instructions |
| `WithToolDefinitions` | `(tools []schema.ToolDefinition) BuilderOption` | Slot 2: tool definitions (semi-static) |
| `WithStaticContext` | `(docs []string) BuilderOption` | Slot 3: reference documents (semi-static) |
| `WithCacheBreakpoint` | `() BuilderOption` | Inserts a cache boundary marker between static and dynamic content |
| `WithDynamicContext` | `(msgs []schema.Message) BuilderOption` | Slot 4: per-session conversation history |
| `WithUserInput` | `(msg schema.Message) BuilderOption` | Slot 5: the current user turn |
| `Build` | `() []schema.Message` | Returns the fully ordered message list, skipping nil/empty slots |

## Quick start

```go
package main

import (
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/prompt"
    promptfile "github.com/lookatitude/beluga-ai/prompt/providers/file"
    "github.com/lookatitude/beluga-ai/schema"
)

func main() {
    base, err := promptfile.NewFileManager("/etc/prompts")
    if err != nil {
        log.Fatal(err)
    }
    // ApplyMiddleware returns prompt.PromptManager, not *file.FileManager.
    var mgr prompt.PromptManager = base
    mgr = prompt.ApplyMiddleware(mgr, prompt.WithTracing())

    msgs, err := mgr.Render("system-assistant", map[string]any{"role": "a Go expert"})
    if err != nil {
        log.Fatal(err)
    }

    all := prompt.NewBuilder(
        prompt.WithSystemPrompt(msgs[0].(*schema.SystemMessage).Parts[0].(schema.TextPart).Text),
        prompt.WithCacheBreakpoint(),
        prompt.WithDynamicContext([]schema.Message{
            schema.NewHumanMessage("What is an interface?"),
        }),
        prompt.WithUserInput(schema.NewHumanMessage("How do I implement one?")),
    ).Build()

    fmt.Printf("%d messages assembled\n", len(all))
}
```

## Full example

The following example shows versioned template retrieval, manual rendering with variable overrides, and cache-optimised assembly with tool definitions.

```go
package main

import (
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/prompt"
    promptfile "github.com/lookatitude/beluga-ai/prompt/providers/file"
    "github.com/lookatitude/beluga-ai/schema"
)

func main() {
    base, err := promptfile.NewFileManager("/etc/prompts")
    if err != nil {
        log.Fatal(err)
    }
    // ApplyMiddleware returns prompt.PromptManager, not *file.FileManager.
    var mgr prompt.PromptManager = base
    mgr = prompt.ApplyMiddleware(mgr, prompt.WithTracing())

    // List all available templates.
    for _, info := range mgr.List() {
        fmt.Printf("template: %s  version: %s\n", info.Name, info.Version)
    }

    // Retrieve a specific version.
    tmpl, err := mgr.Get("system-assistant", "1.1.0")
    if err != nil {
        log.Fatal(err)
    }

    // Render manually to get a raw string (not messages).
    rendered, err := tmpl.Render(map[string]any{"role": "a security expert"})
    if err != nil {
        log.Fatal(err)
    }

    // Assemble the final message list with tools and dynamic history.
    history := []schema.Message{
        schema.NewHumanMessage("What are the OWASP top 10?"),
        schema.NewSystemMessage("Here are the OWASP top 10 risks: ..."),
    }
    tools := []schema.ToolDefinition{
        {Name: "cve_lookup", Description: "Look up CVE details by ID"},
    }
    msgs := prompt.NewBuilder(
        prompt.WithSystemPrompt(rendered),
        prompt.WithToolDefinitions(tools),
        prompt.WithCacheBreakpoint(),
        prompt.WithDynamicContext(history),
        prompt.WithUserInput(schema.NewHumanMessage("Is CVE-2024-1234 critical?")),
    ).Build()

    fmt.Printf("%d messages in final prompt\n", len(msgs))
}
```

## Common mistakes

**Using `Render` when you need version control.** `PromptManager.Render` always fetches the latest version. If you need a pinned version in production, call `Get(name, "1.2.0")` followed by `tmpl.Render(vars)` directly.

**Ignoring slot ordering in `Builder`.** Placing dynamic content in a slot before static content breaks provider-native prefix caching — every request appears as a cache miss. Use `WithCacheBreakpoint()` to mark the boundary explicitly so LLM providers can identify the cacheable prefix.

**Not calling `Validate()` on hand-constructed templates.** `Template.Render` calls `Validate` internally, but if you store templates in a registry or database, validate at write time to surface parse errors early rather than at render time.

**Constructing `FileManager` on every request.** `NewFileManager` reads and parses all files from disk on creation. Construct it once at startup and reuse the instance.

**Expecting `prompt/` to import `cache/`.** The prompt package does not import `cache/`. Provider-native prefix caching is handled by the LLM provider itself; framework-level response deduplication is handled by `cache/` wrapping the LLM layer. These are separate mechanisms at separate layers.

**Implementing `PromptManager` without a compile-time assertion.** Add `var _ prompt.PromptManager = (*MyManager)(nil)` to your implementation file so interface drift is caught at compile time, not at first use.

## Related reading

- [03 — Extensibility Patterns](./03-extensibility-patterns.md) — the four-ring model (Interface, Registry, Hooks, Middleware) that `prompt` follows.
- [14 — Observability](./14-observability.md) — the `WithTracing()` template and GenAI span naming conventions shared by all 17 instrumented packages.
- [patterns/registry-factory.md](../patterns/registry-factory.md) — registry + `init()` pattern; note that `prompt` uses direct construction rather than a named registry because `PromptManager` backends are not interchangeable by string key.
- [patterns/middleware-chain.md](../patterns/middleware-chain.md) — `ApplyMiddleware` outside-in composition.
- [Reference: Providers](../reference/providers.md) — full provider catalog.
- [DOC-09 — Memory Architecture](./09-memory-architecture.md) — where rendered prompts enter the context window alongside retrieved memories.
