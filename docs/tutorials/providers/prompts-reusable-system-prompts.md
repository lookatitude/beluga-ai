# Reusable System Prompts for Personas

In this tutorial, you'll learn how to build a library of reusable system prompts to define distinct personas for your agents, ensuring consistent behavior across your application.

## Learning Objectives

- ✅ Design effective system prompts (personas)
- ✅ Create a prompt registry
- ✅ Implement prompt versioning
- ✅ Dynamic persona switching

## Prerequisites

- Basic prompt engineering (see [Message Template Design](./prompts-message-templates.md))
- Go 1.24+

## Why Personas?

A "Persona" is a set of instructions that defines:
- **Tone**: Formal, casual, pirate-themed?
- **Expertise**: Python coder, legal expert, creative writer?
- **Constraints**: "Never mention competitor X", "Always answer in JSON".

Hardcoding these strings in every agent is unmaintainable.

## Step 1: Defining Personas

Let's define a struct to hold our personas.
```go
package main

import "github.com/lookatitude/beluga-ai/pkg/schema"

type Persona struct {
    Name        string
    Description string
    SystemPrompt string
    Variables   []string
}

var CodingExpert = Persona{
    Name: "CodingExpert",
    Description: "Senior Software Engineer",
    SystemPrompt: `You are a Senior Software Engineer.
    - Write clean, idiomatic code.
    - Always include comments.
    - Use {{.Language}} as the primary language.`,
    Variables: []string{"Language"},
}

var SupportAgent = Persona{
    Name: "SupportAgent",
    Description: "Customer Support Representative",
    SystemPrompt: `You are a helpful support agent for Beluga AI.
    - Be polite and patient.
    - If you don't know, ask clarifying questions.
    - User name: {{.UserName}}`,
    Variables: []string{"UserName"},
}
```

## Step 2: Creating a Registry

A simple map-based registry to retrieve personas.
```go
var personaRegistry = map[string]Persona{
    "coder":   CodingExpert,
    "support": SupportAgent,
}

func GetPersona(name string) (Persona, error) {
    if p, ok := personaRegistry[name]; ok {
        return p, nil
    }
    return Persona{}, fmt.Errorf("persona not found: %s", name)
}
```

## Step 3: Compiling the Prompt

Use `text/template` or Beluga's prompt package to fill variables.
```go
import (
    "bytes"
    "text/template"
    "github.com/lookatitude/beluga-ai/pkg/prompts"
)
go
func (p Persona) CreateSystemMessage(vars map[string]any) (schema.Message, error) {
    tmpl, err := template.New(p.Name).Parse(p.SystemPrompt)
    if err != nil {
        return nil, err
    }
    
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, vars); err != nil {
        return nil, err
    }

    
    return schema.NewSystemMessage(buf.String()), nil
}
```

## Step 4: Using with Agents
```go
func main() {
    // 1. Get Persona
    coder, _ := GetPersona("coder")
    
    // 2. Create System Message
    sysMsg, _ := coder.CreateSystemMessage(map[string]any{
        "Language": "Go",
    })

    

    // 3. Create Agent with this system message
    // (Usually passed as part of the conversation history init)
    history.AddMessage(sysMsg)
    
    // ... run agent ...
}
```

## Step 5: Versioning (Advanced)

Store prompts in files or a DB with version tags.

```json
{
  "name": "coder",
  "version": "1.0",
  "prompt": "You are a coder..."
}
{
  "name": "coder",
  "version": "2.0",
  "prompt": "You are an ELITE 10x developer..."
}
```

## Verification

1. Instantiate the "coder" persona with "Python".
2. Ask it to write a function. Verify it follows Python idioms.
3. Instantiate "support" with a user name. Verify it addresses the user by name.

## Next Steps

- **[Building a Research Agent](../higher-level/agents-research-agent.md)** - Apply personas to agents
- **[Message Template Design](./prompts-message-templates.md)** - Advanced templating
