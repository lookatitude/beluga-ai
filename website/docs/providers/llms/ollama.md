---
title: Ollama
sidebar_position: 1
---

# Ollama Provider Guide

Complete guide to using local Ollama models with Beluga AI Framework.

## Overview

Ollama enables running LLMs locally for privacy and cost savings.

## Setup

### Install Ollama

```bash
# Download from https://ollama.ai
# Or use package manager
brew install ollama  # macOS
```

### Start Ollama

```bash
ollama serve
```

### Pull Models

```bash
ollama pull llama2
ollama pull mistral
```

## Configuration

```go
config := llms.NewConfig(
    llms.WithProvider("ollama"),
    llms.WithModelName("llama2"),
    llms.WithBaseURL("http://localhost:11434"),
)
```

## Benefits

- **Privacy**: Data stays local
- **Cost**: Free (compute only)
- **Control**: Full model control

## Limitations

- Requires local resources
- Model quality varies
- Setup complexity

## Best Practices

- Choose appropriate models for hardware
- Monitor resource usage
- Use for privacy-sensitive applications

---

**Next:** [Provider Comparison](./comparison)

