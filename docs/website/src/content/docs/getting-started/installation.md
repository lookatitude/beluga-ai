---
title: Installation
description: How to install and set up Beluga AI v2.
---

# Installation

## Requirements

- **Go 1.23+** (uses `iter.Seq2[T, error]` for streaming)
- **Git** for version control

## Install

```bash
go get github.com/lookatitude/beluga-ai@latest
```

## Provider-specific dependencies

Individual providers may require additional setup:

- **OpenAI/Anthropic/Google**: Set API keys via environment variables
- **SQLite Vector**: Requires CGO and sqlite-vec extension
- **Silero VAD**: Requires CGO and ONNX Runtime
- **Neo4j/Memgraph**: Requires running graph database instance

See the [Providers](/providers/) section for detailed setup instructions per provider.
