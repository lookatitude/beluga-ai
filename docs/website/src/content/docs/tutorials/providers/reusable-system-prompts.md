---
title: Reusable System Prompts
description: Build a library of versioned, reusable system prompts to define consistent agent personas.
---

A persona is a set of instructions that defines an agent's tone, expertise, and constraints. Hardcoding these strings in every agent is unmaintainable — when a persona needs updating, you must find and change every occurrence. A prompt registry with versioning allows consistent behavior across agents while enabling non-technical team members to update prompts without changing Go code. This separation of prompt content from application logic also supports A/B testing of different persona versions in production.

## What You Will Build

A prompt registry with persona definitions, template variable support, versioning, and integration with Beluga AI agents.

## Prerequisites

- Understanding of [Message Template Design](/tutorials/providers/message-templates)
- Familiarity with the [schema package](/guides/schema)

## Step 1: Define Personas

Create structured persona definitions with template support. Each persona uses Go's `text/template` syntax for variable substitution, which separates the prompt structure (owned by prompt engineers) from the runtime data (owned by application code). The `Variables` field documents which template variables the persona expects, serving as a contract between prompt authors and developers.

```go
package main

import (
    "bytes"
    "fmt"
    "text/template"

    "github.com/lookatitude/beluga-ai/schema"
)

// Persona defines a reusable agent identity.
type Persona struct {
    Name         string
    Description  string
    SystemPrompt string
    Variables    []string
    Version      string
}

var CodingExpert = Persona{
    Name:        "coding-expert",
    Description: "Senior Software Engineer",
    SystemPrompt: `You are a Senior Software Engineer.
- Write clean, idiomatic {{.Language}} code.
- Include error handling in all examples.
- Explain trade-offs when multiple approaches exist.`,
    Variables: []string{"Language"},
    Version:   "1.0",
}

var SupportAgent = Persona{
    Name:        "support-agent",
    Description: "Customer Support Representative",
    SystemPrompt: `You are a support agent for {{.Product}}.
- Be polite and patient.
- If you cannot resolve the issue, escalate clearly.
- Address the user as {{.UserName}}.`,
    Variables: []string{"Product", "UserName"},
    Version:   "1.0",
}
```

## Step 2: Create a System Message from a Persona

Compile the template and produce a `schema.Message`. The template is compiled and executed in a single method call, which ensures that template syntax errors surface immediately rather than being silently ignored. The resulting `*schema.SystemMessage` integrates directly into any `[]schema.Message` conversation.

```go
func (p Persona) CreateSystemMessage(vars map[string]any) (*schema.SystemMessage, error) {
    tmpl, err := template.New(p.Name).Parse(p.SystemPrompt)
    if err != nil {
        return nil, fmt.Errorf("parse persona %s: %w", p.Name, err)
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, vars); err != nil {
        return nil, fmt.Errorf("execute persona %s: %w", p.Name, err)
    }

    return schema.NewSystemMessage(buf.String()), nil
}
```

## Step 3: Build a Persona Registry

Centralize persona management with lookup and discovery. This follows Beluga AI's registry pattern (`Register()` + `Get()` + `List()`), which provides a consistent API for managing extensible collections. The registry centralizes persona definitions so that agents reference personas by name rather than embedding prompt text directly.

```go
type PersonaRegistry struct {
    personas map[string]Persona
}

func NewPersonaRegistry() *PersonaRegistry {
    return &PersonaRegistry{
        personas: make(map[string]Persona),
    }
}

func (r *PersonaRegistry) Register(p Persona) {
    r.personas[p.Name] = p
}

func (r *PersonaRegistry) Get(name string) (Persona, error) {
    p, ok := r.personas[name]
    if !ok {
        return Persona{}, fmt.Errorf("persona not found: %s", name)
    }
    return p, nil
}

func (r *PersonaRegistry) List() []string {
    names := make([]string, 0, len(r.personas))
    for name := range r.personas {
        names = append(names, name)
    }
    return names
}
```

## Step 4: Use with an LLM

Integrate personas into LLM conversations. The persona generates a system message that sets the model's behavior, which is then combined with the user's query in a standard message slice. This keeps the persona definition separate from the conversation logic — the same persona can be used across different agents and conversation flows.

```go
import (
    "context"

    "github.com/lookatitude/beluga-ai/llm"
)

func main() {
    ctx := context.Background()

    registry := NewPersonaRegistry()
    registry.Register(CodingExpert)
    registry.Register(SupportAgent)

    // Get persona
    persona, err := registry.Get("coding-expert")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    // Create system message with variables
    sysMsg, err := persona.CreateSystemMessage(map[string]any{
        "Language": "Go",
    })
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    // Build conversation
    msgs := []schema.Message{
        sysMsg,
        schema.NewHumanMessage("How do I implement a worker pool?"),
    }

    resp, err := model.Generate(ctx, msgs)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    fmt.Println(resp.Text())
}
```

## Step 5: Prompt Versioning

Support multiple versions of the same persona for A/B testing and gradual rollouts. The versioned registry uses a nested map (`name -> version -> Persona`) to store multiple versions of each persona independently. This enables comparing different prompt strategies in production without affecting the registry's lookup API.

```go
type VersionedRegistry struct {
    personas map[string]map[string]Persona // name -> version -> Persona
}

func (r *VersionedRegistry) Register(p Persona) {
    if r.personas[p.Name] == nil {
        r.personas[p.Name] = make(map[string]Persona)
    }
    r.personas[p.Name][p.Version] = p
}

func (r *VersionedRegistry) Get(name, version string) (Persona, error) {
    versions, ok := r.personas[name]
    if !ok {
        return Persona{}, fmt.Errorf("persona not found: %s", name)
    }
    p, ok := versions[version]
    if !ok {
        return Persona{}, fmt.Errorf("version %s not found for persona %s", version, name)
    }
    return p, nil
}
```

## Step 6: Load Personas from Configuration

Store personas in JSON files for non-developer editing. Externalizing prompt content into JSON files enables prompt engineers to iterate on persona definitions without requiring Go builds or code reviews for text changes. The application loads these files at startup, making prompt updates a configuration change rather than a code change.

```json
{
    "name": "coding-expert",
    "version": "2.0",
    "description": "Senior Software Engineer (v2)",
    "system_prompt": "You are an elite software engineer specializing in {{.Language}}.\n- Write production-grade code.\n- Include benchmarks when discussing performance.",
    "variables": ["Language"]
}
```

```go
import "encoding/json"

func loadPersonaFromFile(path string) (Persona, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return Persona{}, err
    }

    var p Persona
    if err := json.Unmarshal(data, &p); err != nil {
        return Persona{}, err
    }
    return p, nil
}
```

## Verification

1. Instantiate the "coding-expert" persona with "Python" — verify the system message mentions Python.
2. Instantiate "support-agent" with a user name — verify it addresses the user by name.
3. Register two versions of the same persona — verify both can be retrieved independently.

## Next Steps

- [Research Agent](/tutorials/agents/research-agent) — Apply personas to autonomous agents
- [Message Template Design](/tutorials/providers/message-templates) — Advanced template patterns
