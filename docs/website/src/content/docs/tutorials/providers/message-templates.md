---
title: Message Template Design
description: Design effective prompt templates using Beluga AI's prompt package for type-safe, reusable LLM interactions.
---

A prompt is the programming interface for an LLM. Just as you avoid hardcoding values in Go code, you should avoid hardcoding values in prompts. Beluga AI's `prompt` package separates instruction logic (templates) from data (variables), making prompts reusable, testable, and versionable.

## What You Will Build

Prompt templates ranging from simple string interpolation to multi-role chat sequences with persona definitions, variable validation, and template composition.

## Prerequisites

- Understanding of [Multi-turn Conversations](/tutorials/foundation/multiturn-conversations)
- Familiarity with Go's `text/template` syntax

## Step 1: Basic String Templates

Create a simple template with variable substitution:

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

Build structured multi-role conversations from templates:

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

Define reusable personas as constants:

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

Since templates use Go's `text/template`, you can include conditionals and loops:

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

Validate that all required variables are provided before executing:

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

Always use template substitution — never string concatenation — to insert user input into prompts. Template execution prevents the most common prompt injection patterns by treating user input as data, not as template directives.

## Verification

1. Create a translator template and verify the output for different languages.
2. Use a persona template with missing variables — verify the error is descriptive.
3. Test template logic with `range` and `if` directives.

## Next Steps

- [Reusable System Prompts](/tutorials/providers/reusable-system-prompts) — Build a persona library
- [Multi-turn Conversations](/tutorials/foundation/multiturn-conversations) — Use templates in conversation flows
