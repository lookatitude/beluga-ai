---
title: "Composio"
description: "Access SaaS integrations as tools via the Composio API."
---

The Composio provider connects Beluga AI to the [Composio](https://composio.dev/) platform, which provides access to 250+ SaaS integrations (Gmail, Slack, GitHub, Jira, and more) as callable tools. It implements tool discovery and execution through the Composio REST API, returning tools as native `tool.Tool` instances.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/protocol/mcp/providers/composio
```

## Configuration

| Option | Type | Default | Description |
|---|---|---|---|
| `WithAPIKey(key)` | `string` | (required) | Composio API key |
| `WithBaseURL(url)` | `string` | `https://backend.composio.dev` | API endpoint |
| `WithTimeout(d)` | `time.Duration` | `30s` | HTTP request timeout |

**Environment variables:**

| Variable | Maps to |
|---|---|
| `COMPOSIO_API_KEY` | `WithAPIKey` |

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/protocol/mcp/providers/composio"
)

func main() {
    client, err := composio.New(
        composio.WithAPIKey(os.Getenv("COMPOSIO_API_KEY")),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Discover all available tools
    tools, err := client.ListTools(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    for _, t := range tools {
        fmt.Printf("%s: %s\n", t.Name(), t.Description())
    }
}
```

## Tool Execution

Each discovered tool implements the `tool.Tool` interface and can be executed directly:

```go
client, err := composio.New(
    composio.WithAPIKey(os.Getenv("COMPOSIO_API_KEY")),
)
if err != nil {
    log.Fatal(err)
}

tools, err := client.ListTools(ctx)
if err != nil {
    log.Fatal(err)
}

// Find a specific tool
for _, t := range tools {
    if t.Name() == "GMAIL_SEND_EMAIL" {
        result, err := t.Execute(ctx, map[string]any{
            "to":      "user@example.com",
            "subject": "Hello from Beluga AI",
            "body":    "This email was sent via Composio.",
        })
        if err != nil {
            log.Fatal(err)
        }
        for _, part := range result.Content {
            fmt.Println(part)
        }
        break
    }
}
```

## With Agents

Composio tools integrate directly with Beluga agents since they implement `tool.Tool`:

```go
import (
    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/protocol/mcp/providers/composio"
)

client, err := composio.New(
    composio.WithAPIKey(os.Getenv("COMPOSIO_API_KEY")),
)
if err != nil {
    log.Fatal(err)
}

tools, err := client.ListTools(ctx)
if err != nil {
    log.Fatal(err)
}

// Pass Composio tools to an agent
myAgent, err := agent.New("assistant",
    agent.WithModel(model),
    agent.WithTools(tools...),
    agent.WithInstructions("You are an assistant that can send emails and manage tasks."),
)
```

## With Tool Registry

Add Composio tools to a local tool registry for centralized management:

```go
import "github.com/lookatitude/beluga-ai/tool"

registry := tool.NewRegistry()

tools, err := client.ListTools(ctx)
if err != nil {
    log.Fatal(err)
}

for _, t := range tools {
    err := registry.Add(t)
    if err != nil {
        log.Printf("failed to register %s: %v", t.Name(), err)
    }
}

// Access tools by name
emailTool, err := registry.Get("GMAIL_SEND_EMAIL")
if err != nil {
    log.Fatal(err)
}
```

## Tool Schema

Each Composio tool exposes its input schema as a JSON Schema object, which includes parameter names, types, and descriptions from the Composio platform. Use `InputSchema()` to inspect the expected inputs:

```go
for _, t := range tools {
    fmt.Printf("Tool: %s\n", t.Name())
    fmt.Printf("Schema: %v\n", t.InputSchema())
}
```

## API Endpoints

The provider uses two Composio API endpoints:

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/actions` | `GET` | List all available actions (tools) |
| `/api/v1/actions/{name}/execute` | `POST` | Execute a specific action |

## Error Handling

```go
tools, err := client.ListTools(ctx)
if err != nil {
    // Errors include authentication failures, network issues, and API errors
    log.Fatal(err)
}

result, err := tools[0].Execute(ctx, input)
if err != nil {
    // Execution errors from the Composio API
    log.Printf("tool execution failed: %v", err)
}

// Check for tool-level errors
if result.IsError {
    fmt.Println("Tool returned an error result")
}
```

The provider checks the `successful` field in execution responses. If the API reports `successful: false`, the tool returns an error result with the error message from the API.

## Authentication

Composio uses API key authentication. The key is sent in the `x-api-key` HTTP header on every request.
