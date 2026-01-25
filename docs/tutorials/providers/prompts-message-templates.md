# Message Template Design

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this guide, we'll explore how to design effective prompt templates, ranging from simple greetings to complex multi-role chat sequences with specialized personas using Beluga AI's `pkg/prompts` package.

## Learning Objectives
By the end of this tutorial, you will:
1.  Create and format basic `StringPromptTemplate`s.
2.  Use `ChatPromptAdapter` to model system and user roles.
3.  Design a "Persona" system using reusable system prompts.

## Introduction
Welcome, colleague! A prompt is more than just a string. It's the "programming code" for your LLM. Just as you wouldn't hardcode variables in Go, you shouldn't hardcode values in your prompts.

Beluga AI's `pkg/prompts` package provides a structured way to manage these templates. It separates the **instruction logic** (the template) from the **data** (the variables). This makes your prompts reusable, testable, and easier to version.

## Why This Matters

*   **Consistency**: Ensure your AI always responds with the same tone and constraints.
*   **Security**: Use built-in variable validation to prevent prompt injection or malformed requests.
*   **Observability**: Track which templates are being used and which ones are failing using OpenTelemetry.

## Concepts

### String Templates
Uses the standard Go `text/template` syntax (`\{\{.VariableName\}\}`). Best for simple completions or single-message prompts.

### Chat Templates (Adapters)
AI models today are mostly "chat-tuned." They expect a sequence of roles:
- **System**: Instructions on how to behave.
- **Human/User**: The current query.
- **AI/Assistant**: Previous context.

`ChatPromptAdapter` coordinates these roles into a single object.

---

## Step-by-Step Implementation

### Step 1: Basic String Templates

Let's start simple. Suppose we are building a translation tool.
```go
package main

import (
    "context"
    "fmt"
    "log"
    "github.com/lookatitude/beluga-ai/pkg/prompts"
)

func main() {
    ctx := context.Background()

    // 1. Define the template
    // We use {{.var}} syntax
    template, err := prompts.NewStringPromptTemplate(
        "translator", 
        "Translate the following text to {{.language}}: {{.text}}",
    )
    if err != nil {
        log.Fatal(err)
    }

    // 2. Format with data
    inputs := map[string]interface{}{
        "language": "French",
        "text":     "The library is open late today.",
    }

    result, err := template.Format(ctx, inputs)
    if err != nil {
        log.Fatal(err)
    }

    // result.ToString() returns the final rendered string
    fmt.Println(result.ToString())
}
```

### Step 2: Designing Reusable Personas

One of the most powerful patterns in AI is the **Persona**. instead of writing "You are a helpful assistant" every time, we can create specialized system prompts.
```go
const (
    SystemPromptTechSupport = `You are a Senior Linux Admin. 
You are helpful, concise, and always provide code snippets when applicable. 
Your tone is professional but direct.`
    SystemPromptCreativeWriter = `You are a Nobel Prize winning novelist. 
You use descriptive language, metaphors, and prioritize emotional depth.`

)

Now, let's use these with a `ChatPromptAdapter`.
```go
func createSupportAgent(query string) {
    ctx := context.Background()

    // The ChatPromptAdapter takes:
    // 1. Name
    // 2. System Template
    // 3. User Template
    // 4. List of variable names
    adapter, _ := prompts.NewChatPromptAdapter(
        "tech_support",
        SystemPromptTechSupport,             // Use our reusable persona
        "I am having this issue: {{.query}}", // The user's input
        []string{"query"},
    )

    messages, _ := adapter.Format(ctx, map[string]interface{}{
        "query": "My cron job isn't running on Ubuntu 22.04",
    })


    // 'messages' is now a []schema.Message
    // Index 0: System Message (The Persona)
    // Index 1: Human Message (The Query)
}
```

### Step 3: Enforcing Strict Validation

What if a developer forgets to pass a variable? `pkg/prompts` can catch this before you spend money on an LLM call.

```go
manager, _ := prompts.NewPromptManager(
    prompts.WithConfig(&prompts.Config{
        ValidateVariables:   true,
        StrictVariableCheck: true,
    }),
)

// If we try to format a template missing a required variable,
// Format() will return a VariableMissingError.
```

## Pro-Tips

*   **Logic in Templates**: Since we use `text/template`, you can use `if/else` or `range` inside your prompts!

    ```go
    "Here is your list:
    {{range .items}}- {{.}}
    {{end}}"
    ```
*   **Persona Registry**: Keep your system prompts in a separate `.yaml` or `.json` file and load them at startup using `PromptManager`. This lets your non-technical team members update the AI's "vibe" without changing Go code.
*   **Prompt Injection Warning**: Always treat user input as untrusted. Never use string concatenation (`"Hello " + userInput`) to build prompts. Always use templates!

## Troubleshooting

### "Variable missing"
Double check that the key in your `map[string]interface\{\}` exactly matches the `{{.VariableName}}` in your template (including capitalization).

### "Unknown Role"
If you are passing formatted chat messages to an LLM provider (like OpenAI), ensure you are using the `ChatPromptAdapter` which correctly tags messages as `system` and `human`.

## Conclusion

You've now moved from string-mashing to proper **Prompt Engineering**. By using `pkg/prompts`, you've created a clean separation between the "Brain" (the LLM) and the "Script" (the Template). This structure is essential as you scale to multi-agent systems and complex RAG pipelines.
