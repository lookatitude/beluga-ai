---
title: "Llama (Meta)"
description: "Integration guide for Meta's Llama models with Beluga AI."
---

The Llama provider is a meta-provider that enables running Meta's Llama models through any of several hosting backends. Since Meta does not offer a direct inference API, this provider delegates to Together AI, Fireworks AI, Groq, SambaNova, Cerebras, or Ollama depending on the selected backend.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/llama
```

You also need to import the backend provider you intend to use:

```go
import (
    _ "github.com/lookatitude/beluga-ai/llm/providers/llama"
    _ "github.com/lookatitude/beluga-ai/llm/providers/together"  // or your chosen backend
)
```

## Configuration

| Field    | Required | Default | Description                              |
|----------|----------|---------|------------------------------------------|
| `Model`  | Yes      | —       | Llama model ID (format depends on backend) |
| `APIKey` | Varies   | —       | API key for the backend provider         |
| `BaseURL`| No       | Backend default | Override API endpoint              |
| `Timeout`| No       | `30s`   | Request timeout                          |

**Provider-specific options (via `Options` map):**

| Key       | Default      | Description                                                      |
|-----------|--------------|------------------------------------------------------------------|
| `backend` | `"together"` | Backend provider: `together`, `fireworks`, `groq`, `sambanova`, `cerebras`, `ollama` |

**Supported backends and their default base URLs:**

| Backend      | Default Base URL                           |
|--------------|--------------------------------------------|
| `together`   | `https://api.together.xyz/v1`             |
| `fireworks`  | `https://api.fireworks.ai/inference/v1`   |
| `groq`       | `https://api.groq.com/openai/v1`         |
| `sambanova`  | `https://api.sambanova.ai/v1`            |
| `cerebras`   | `https://api.cerebras.ai/v1`             |
| `ollama`     | `http://localhost:11434/v1`               |

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
    _ "github.com/lookatitude/beluga-ai/llm/providers/llama"
    _ "github.com/lookatitude/beluga-ai/llm/providers/together"
)

func main() {
    model, err := llm.New("llama", config.ProviderConfig{
        Model:  "meta-llama/Llama-3.3-70B-Instruct",
        APIKey: os.Getenv("TOGETHER_API_KEY"),
        Options: map[string]any{
            "backend": "together",
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    msgs := []schema.Message{
        schema.NewSystemMessage("You are a helpful assistant."),
        schema.NewHumanMessage("What is the capital of France?"),
    }

    resp, err := model.Generate(context.Background(), msgs)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Text())
}
```

## Switching Backends

### Groq (ultra-fast inference)

```go
model, err := llm.New("llama", config.ProviderConfig{
    Model:  "llama-3.3-70b-versatile",
    APIKey: os.Getenv("GROQ_API_KEY"),
    Options: map[string]any{
        "backend": "groq",
    },
})
```

### Ollama (local inference)

```go
model, err := llm.New("llama", config.ProviderConfig{
    Model: "llama3.2",
    Options: map[string]any{
        "backend": "ollama",
    },
})
```

### Fireworks AI

```go
model, err := llm.New("llama", config.ProviderConfig{
    Model:  "accounts/fireworks/models/llama-v3p1-70b-instruct",
    APIKey: os.Getenv("FIREWORKS_API_KEY"),
    Options: map[string]any{
        "backend": "fireworks",
    },
})
```

## Streaming

```go
for chunk, err := range model.Stream(context.Background(), msgs) {
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(chunk.Delta)
}
fmt.Println()
```

## Advanced Features

All features available on the underlying backend provider are supported: tool calling, structured output, generation options, etc. The Llama provider simply delegates to the chosen backend.

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithTemperature(0.7),
    llm.WithMaxTokens(2048),
)
```

## Error Handling

```go
resp, err := model.Generate(ctx, msgs)
if err != nil {
    // Error prefix depends on the backend provider
    log.Fatal(err)
}
```

## Direct Construction

```go
import (
    "github.com/lookatitude/beluga-ai/llm/providers/llama"
    _ "github.com/lookatitude/beluga-ai/llm/providers/together"
)

model, err := llama.New(config.ProviderConfig{
    Model:  "meta-llama/Llama-3.3-70B-Instruct",
    APIKey: os.Getenv("TOGETHER_API_KEY"),
    Options: map[string]any{"backend": "together"},
})
```

## Model Naming by Backend

Note that Llama model IDs vary by backend:

| Backend    | Example Model ID                                          |
|------------|-----------------------------------------------------------|
| Together   | `meta-llama/Llama-3.3-70B-Instruct`                     |
| Fireworks  | `accounts/fireworks/models/llama-v3p1-70b-instruct`     |
| Groq       | `llama-3.3-70b-versatile`                                |
| SambaNova  | `Meta-Llama-3.3-70B-Instruct`                           |
| Cerebras   | `llama-3.3-70b`                                          |
| Ollama     | `llama3.2`                                               |
