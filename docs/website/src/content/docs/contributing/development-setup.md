---
title: Development Setup
description: Set up your local development environment for Beluga AI
---

This guide walks you through setting up a local development environment for contributing to Beluga AI.

## Prerequisites

Before you begin, make sure you have the following installed:

| Tool | Version | Required | Notes |
|---|---|---|---|
| **Go** | 1.23+ | Yes | Beluga uses `iter.Seq2` and other Go 1.23 features |
| **Git** | 2.x+ | Yes | For cloning and version control |
| **Make** | Any | Recommended | Simplifies common tasks via Makefile targets |
| **golangci-lint** | Latest | Recommended | Runs all configured linters in one pass |
| **Node.js** | 22+ | Docs only | Only needed if working on the documentation website |

### Installing golangci-lint

```bash
# macOS / Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Or via Go
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Fork and Clone

1. **Fork** the repository on GitHub by clicking the "Fork" button at [github.com/lookatitude/beluga-ai](https://github.com/lookatitude/beluga-ai).

2. **Clone** your fork locally:

   ```bash
   git clone https://github.com/<your-username>/beluga-ai.git
   cd beluga-ai
   ```

3. **Add the upstream remote** to keep your fork in sync:

   ```bash
   git remote add upstream https://github.com/lookatitude/beluga-ai.git
   git fetch upstream
   ```

4. **Create a branch** for your work:

   ```bash
   git checkout -b feat/my-feature
   ```

## Building

Build the entire project to verify your setup:

```bash
# Using Make (recommended)
make build

# Or directly with Go
go build ./...
```

## Running Tests

```bash
# Run all unit tests
make test

# Run integration tests (requires external services)
make integration-test
```

See the [Testing Guide](/contributing/testing/) for more details on writing and running tests.

## Running the Linter

```bash
# Run all configured linters
make lint
```

The linter checks are the same ones that run in CI, so fix any issues before opening a PR.

## IDE Setup

### VS Code

1. Install the [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.Go) (`golang.go`).
2. Open the repository root as your workspace folder.
3. The extension will prompt you to install required tools — accept all.
4. Recommended `settings.json` additions:

   ```json
   {
     "go.lintTool": "golangci-lint",
     "go.lintFlags": ["--fast"],
     "go.testFlags": ["-v", "-count=1"],
     "editor.formatOnSave": true,
     "[go]": {
       "editor.defaultFormatter": "golang.go"
     }
   }
   ```

### GoLand

1. Open the repository root as a project.
2. Go to **Settings → Go → Go Modules** and ensure module integration is enabled.
3. Go to **Settings → Tools → File Watchers** and add `goimports` as a file watcher for automatic import formatting.
4. Configure **Settings → Editor → Code Style → Go** to match project conventions.

## Project Structure

Beluga AI follows a flat package layout with no `pkg/` prefix. Here's a high-level overview:

```
beluga-ai/
├── core/           # Foundation: Stream, Runnable, Lifecycle, Errors
├── schema/         # Shared types: Message, ContentPart, Tool, Event
├── config/         # Configuration loading and validation
├── o11y/           # Observability: OpenTelemetry, slog adapters
├── llm/            # LLM abstraction and providers
├── tool/           # Tool system: FuncTool, MCP client
├── memory/         # Multi-tier memory (Core/Recall/Archival)
├── rag/            # RAG pipeline: embedding, vectorstore, retriever
├── agent/          # Agent runtime: BaseAgent, Planner, Executor
├── voice/          # Voice pipeline: STT, TTS, S2S, transport
├── orchestration/  # Chain, Graph, Router, Parallel
├── protocol/       # MCP server/client, A2A
├── guard/          # Input/output/tool safety guards
├── resilience/     # Circuit breaker, retry, rate limiting
├── internal/       # Shared utilities (not public API)
└── docs/           # Documentation and website
```

For a detailed breakdown, see the [Architecture documentation](/architecture/packages/).

## Makefile Reference

| Target | Description |
|---|---|
| `make build` | Build all packages |
| `make test` | Run unit tests |
| `make test-verbose` | Run unit tests with verbose output |
| `make integration-test` | Run integration tests |
| `make lint` | Run golangci-lint |
| `make fmt` | Format code with gofmt and goimports |
| `make coverage` | Generate test coverage report |
| `make bench` | Run benchmarks |
| `make fuzz` | Run fuzz tests |
| `make check` | Run fmt + lint + test (full pre-PR check) |
| `make docs` | Build the documentation website |
| `make clean` | Remove build artifacts |
