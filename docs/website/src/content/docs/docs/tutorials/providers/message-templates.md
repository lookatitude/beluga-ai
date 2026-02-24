---
title: Message Template Design
description: "Design type-safe prompt templates in Go using Beluga AI's prompt package — variable substitution, multi-role chat sequences, validation, and template composition."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, message templates, prompt design, type-safe, template composition, LLM"
---

A prompt is the programming interface for an LLM. Just as you avoid hardcoding values in Go code, you should avoid hardcoding values in prompts. String concatenation for building prompts is fragile, difficult to test, and vulnerable to prompt injection. Beluga AI's `prompt` package separates instruction logic (templates) from data (variables), making prompts reusable, testable, and versionable. This separation follows the same principle as SQL parameterized queries — the template defines the structure, and variables fill in the data.

## What You Will Build

Prompt templates ranging from simple string interpolation to multi-role chat sequences with persona definitions, variable validation, and template composition.

## Prerequisites

- Understanding of [Multi-turn Conversations](/docs/tutorials/foundation/multiturn-conversations)
- Familiarity with Go's `text/template` syntax

## Step 1: Basic String Templates

Create a simple template with variable substitution. Go's `text/template` package provides the foundation for prompt templates because it is part of the standard library, well-tested, and supports the logic constructs (conditionals, loops) needed for complex prompts.

```go
package main

import (
    "bytes"
    "fmt"
    "text/template"
)

func main() {
    tmpl, err := template.New("translator").Parse(
        "Translate the following text to {{.Language}}: {{.Text}}",
    )
    if err != nil {
        fmt.Printf("Parse error: %v\n", err)
        return
    }

    var buf bytes.Buffer
    err = tmpl.Execute(&buf, map[string]string{
        "Language": "French",
        "Text":     "The library is open late today.",
    })
    if err != nil {
        fmt.Printf("Execute error: %v\n", err)
        return
    }

    fmt.Println(buf.String())
    // Output: Translate the following text to French: The library is open late today.
}
```

## Step 2: Chat Message Templates

Build structured multi-role conversations from templates. The `ChatTemplate` struct separates the system prompt (which defines the agent's behavior) from the user prompt (which carries the query). This mirrors the message role structure that LLMs expect, where the system message sets constraints and the human message provides the task.

```go
import "github.com/lookatitude/beluga-ai/schema"

type ChatTemplate struct {
    SystemTemplate string
    UserTemplate   string
}

func (t *ChatTemplate) Format(vars map[string]any) ([]schema.Message, error) {
    systemText, err := executeTemplate("system", t.SystemTemplate, vars)
    if err != nil {
        return nil, fmt.Errorf("system template: %w", err)
    }

    userText, err := executeTemplate("user", t.UserTemplate, vars)
    if err != nil {
        return nil, fmt.Errorf("user template: %w", err)
    }

    return []schema.Message{
        schema.NewSystemMessage(systemText),
        schema.NewHumanMessage(userText),
    }, nil
}

func executeTemplate(name, text string, vars map[string]any) (string, error) {
    tmpl, err := template.New(name).Parse(text)
    if err != nil {
        return "", err
    }
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, vars); err != nil {
        return "", err
    }
    return buf.String(), nil
}
```

Usage:

```go
support := &ChatTemplate{
    SystemTemplate: `You are a Senior Linux Administrator.
You are concise and always provide code snippets when applicable.`,
    UserTemplate: `I am having this issue: {{.Query}}`,
}

msgs, err := support.Format(map[string]any{
    "Query": "My cron job isn't running on Ubuntu 22.04",
})
if err != nil {
    fmt.Printf("Format error: %v\n", err)
    return
}

// msgs[0] is the system message with the persona
// msgs[1] is the human message with the query
resp, err := model.Generate(ctx, msgs)
```

## Step 3: Persona Definitions

Define reusable personas as constants. Persona constants centralize behavior definitions that would otherwise be scattered across agent configurations. When a persona needs updating — changing tone, adding constraints, or updating expertise — you modify one constant rather than finding every agent that embeds the same text.

```go
const (
    PersonaTechSupport = `You are a Senior Linux Administrator.
- Be helpful, concise, and provide code snippets.
- Always explain the "why" behind commands.
- If the issue is ambiguous, ask clarifying questions.`

    PersonaCodeReviewer = `You are a Senior Software Engineer reviewing code.
- Focus on correctness, performance, and readability.
- Use the language: {{.Language}}.
- Cite specific line numbers when suggesting changes.`

    PersonaDataAnalyst = `You are a Data Analyst.
- Use precise numbers and cite sources.
- Present findings in structured format.
- Flag any data quality issues.`
)
```

## Step 4: Template Logic

Since templates use Go's `text/template`, you can include conditionals and loops. Template logic is useful when the prompt structure changes based on the data — for example, including a summary section only when requested, or iterating over a variable number of data points. This avoids building prompts through string concatenation, which is error-prone and difficult to maintain.

```go
const reportTemplate = `Analyze the following data points:
{{range .DataPoints}}- {{.Label}}: {{.Value}}
{{end}}
{{if .IncludeSummary}}Provide a summary at the end.{{end}}
Format: {{.Format}}`

type DataPoint struct {
    Label string
    Value string
}

vars := map[string]any{
    "DataPoints": []DataPoint{
        {Label: "Revenue", Value: "$1.2M"},
        {Label: "Growth", Value: "15%"},
    },
    "IncludeSummary": true,
    "Format":         "markdown",
}
```

## Step 5: Variable Validation

Validate that all required variables are provided before executing. Validation catches missing variables at template execution time rather than producing prompts with blank fields that confuse the LLM. The explicit `RequiredVars` approach provides clear error messages that identify which variable is missing, which is more useful than Go's template engine's generic "nil pointer" errors.

```go
func validateVars(tmplText string, vars map[string]any) error {
    tmpl, err := template.New("check").Parse(tmplText)
    if err != nil {
        return err
    }

    // Execute into a discard writer to catch missing variable errors
    var buf bytes.Buffer
    return tmpl.Execute(&buf, vars)
}
```

For production use, define required variables explicitly:

```go
type ValidatedTemplate struct {
    Template     string
    RequiredVars []string
}

func (t *ValidatedTemplate) Format(vars map[string]any) (string, error) {
    for _, required := range t.RequiredVars {
        if _, ok := vars[required]; !ok {
            return "", fmt.Errorf("missing required variable: %s", required)
        }
    }
    return executeTemplate("validated", t.Template, vars)
}
```

## Security Note

Always use template substitution — never string concatenation — to insert user input into prompts. Template execution prevents the most common prompt injection patterns by treating user input as data, not as template directives. While template substitution is not a complete defense against prompt injection, it eliminates the class of attacks where user input alters the template structure itself.

## Verification

1. Create a translator template and verify the output for different languages.
2. Use a persona template with missing variables — verify the error is descriptive.
3. Test template logic with `range` and `if` directives.

## Next Steps

- [Reusable System Prompts](/docs/tutorials/providers/reusable-system-prompts) — Build a persona library
- [Multi-turn Conversations](/docs/tutorials/foundation/multiturn-conversations) — Use templates in conversation flows
