# Advanced Inference Options

In this tutorial, you'll learn how to fine-tune your LLM generation using advanced inference parameters like temperature, penalty, and top-p/top-k sampling.

## Learning Objectives

- ✅ Understand inference parameters (Temperature, TopP, TopK, etc.)
- ✅ Control creativity vs. determinism
- ✅ Reduce repetition with frequency/presence penalties
- ✅ Configure advanced streaming options

## Prerequisites

- Basic LLM usage (see [Your First LLM Call](../../getting-started/01-first-llm-call.md))
- Go 1.24+

## Why Inference Options Matter

Default settings work for general queries, but specific tasks require tuning:
- **Code Generation**: Needs low temperature (determinism).
- **Creative Writing**: Needs high temperature (creativity).
- **Q&A**: Needs balanced settings to avoid hallucinations while maintaining fluency.

## Step 1: Temperature and TopP

Temperature controls randomness. TopP (Nucleus Sampling) restricts the token pool to the top cumulative probability.
```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/llms"
)

func main() {
    ctx := context.Background()

    // High creativity configuration
    creativeConfig := llms.NewConfig(
        llms.WithProvider("openai"),
        llms.WithModelName("gpt-4"),
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        // High temperature = more random/creative
        llms.WithTemperatureConfig(1.2),
        // TopP = 0.9 means consider top 90% probability mass
        llms.WithTopPConfig(0.9),
    )
    
    // Low creativity (deterministic) configuration
    deterministicConfig := llms.NewConfig(
        llms.WithProvider("openai"),
        llms.WithModelName("gpt-4"),
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        // Temperature 0 = almost deterministic
        llms.WithTemperatureConfig(0.0),
    )
    
    // ... create provider and generate ...
}
```

## Step 2: Penalties (Frequency & Presence)

Prevent the model from repeating itself.

- **Frequency Penalty**: Penalizes tokens based on how many times they've appeared. Good for reducing verbatim repetition.
- **Presence Penalty**: Penalizes tokens if they've appeared at all. Good for encouraging new topics.
```go
config := llms.NewConfig(
    llms.WithProvider("openai"),
    llms.WithModelName("gpt-4"),
    llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),

    
    // Penalize repetition strongly
    llms.WithFrequencyPenaltyConfig(1.0), // Range usually -2.0 to 2.0
    llms.WithPresencePenaltyConfig(0.5),
)
```

## Step 3: TopK (Provider Dependent)

TopK limits the next token selection to the top K most likely tokens. Not all providers support this (OpenAI doesn't, Anthropic/Cohere might).
```go
config := llms.NewConfig(
    llms.WithProvider("anthropic"),
    llms.WithModelName("claude-3-opus-20240229"),
    llms.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),

    
    // Only consider top 50 tokens
    llms.WithTopKConfig(50),
)
```

## Step 4: Max Tokens and Stop Sequences

Control output length and stopping conditions.
```go
config := llms.NewConfig(
    llms.WithProvider("openai"),
    llms.WithModelName("gpt-3.5-turbo"),
    llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),

    

    // Limit response length
    llms.WithMaxTokensConfig(100),
    
    // Stop generating when these sequences appear
    llms.WithStopSequencesConfig([]string{"\n", "User:"}),
)
```

## Step 5: Runtime Options

You can also override options per-call using `Generate` options (if supported by the provider interface extension, or creating a new provider instance). Currently, Beluga AI emphasizes configuration at creation time, but some advanced patterns allow runtime overrides.

## Verification

1. Create a "Creative Writer" script with High Temp/Presence Penalty.
2. Create a "Data Extractor" script with Low Temp/Low Penalty.
3. Compare the outputs for the same prompt (e.g., "Write a poem about rust").

## Next Steps

- **[Adding a New LLM Provider](./llms-new-provider.md)** - Extend the framework
- **[Fine-tuning Embedding Strategies](./embeddings-finetuning-strategies.md)** - Optimize retrieval
