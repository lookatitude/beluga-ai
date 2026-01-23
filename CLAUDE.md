# CLAUDE.md - AI Assistant Guide for Beluga AI Framework

This document provides essential context for AI assistants working on the Beluga AI Framework codebase.

## Project Overview

**Beluga AI Framework** is a comprehensive, production-ready Go framework for building sophisticated AI and agentic applications. It serves as a Go-native alternative to Python frameworks like LangChain and CrewAI, providing robust abstractions for LLM integration, agent creation, RAG pipelines, voice processing, and workflow orchestration.

- **Language**: Go 1.24+
- **License**: MIT
- **Status**: Enterprise-grade, production-ready
- **Module Path**: `github.com/lookatitude/beluga-ai`

## Quick Reference Commands

```bash
# Build and test
make build              # Build all packages
make test               # Run all tests
make test-unit          # Unit tests only (with race detection)
make test-integration   # Integration tests only

# Code quality
make lint               # Run golangci-lint
make lint-fix           # Auto-fix lint issues
make fmt                # Format code with gofmt
make vet                # Run go vet

# Security
make security           # Run gosec, govulncheck, gitleaks

# Full CI locally
make ci-local           # Run complete CI pipeline locally

# Coverage
make test-coverage      # Generate coverage report (outputs to coverage/)
```

## Repository Structure

```
beluga-ai/
├── pkg/                    # Core framework packages (19 packages)
│   ├── agents/            # Agent framework with tools and executors
│   ├── chatmodels/        # Chat-based LLM interfaces
│   ├── config/            # Configuration management (Viper-based)
│   ├── core/              # Core utilities (errors, DI, runnable)
│   ├── documentloaders/   # Load documents from files/URLs
│   ├── embeddings/        # Text embedding providers
│   ├── llms/              # LLM provider abstractions
│   ├── memory/            # Conversation memory (buffer, summary, vector)
│   ├── messaging/         # Messaging providers (Twilio)
│   ├── monitoring/        # OpenTelemetry integration
│   ├── multimodal/        # Multi-modal processing
│   ├── orchestration/     # Workflow orchestration (chains, graphs)
│   ├── prompts/           # Prompt template management
│   ├── retrievers/        # Document retrieval for RAG
│   ├── schema/            # Core data structures (messages, documents)
│   ├── server/            # REST and MCP server APIs
│   ├── textsplitters/     # Text chunking for RAG
│   ├── vectorstores/      # Vector database providers
│   └── voice/             # Complete voice framework (STT, TTS, VAD)
│
├── cmd/                    # Command-line tools
├── examples/               # 64+ runnable examples by category
├── tests/                  # Integration testing framework
│   └── integration/       # End-to-end, package pairs, provider compat
├── docs/                   # Comprehensive documentation
├── specs/                  # Feature specifications
├── scripts/                # Utility scripts
└── .github/workflows/      # CI/CD pipelines
```

## Package Structure Convention

Every package **MUST** follow this standardized structure:

```
pkg/{package_name}/
├── iface/                    # Public interfaces and types (REQUIRED)
├── internal/                 # Private implementation details
├── providers/                # Provider implementations (multi-provider packages)
├── config.go                 # Configuration structs with validation
├── metrics.go                # OTEL metrics implementation
├── errors.go                 # Custom errors (Op/Err/Code pattern)
├── {package_name}.go         # Main API and factory functions
├── registry.go               # Global registry (multi-provider packages)
├── test_utils.go             # Test helpers and mock factories
├── advanced_test.go          # Comprehensive test suite
└── README.md                 # Package documentation
```

## Key Design Patterns

### 1. Interface Segregation (ISP)
- Small, focused interfaces with single responsibilities
- No "god interfaces" - prefer composition

```go
// Good: Focused interfaces
type LLMCaller interface {
    Generate(ctx context.Context, prompt string) (string, error)
}

type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
}
```

### 2. Global Registry Pattern (Multi-Provider Packages)
```go
// Standard pattern for provider registration
func RegisterGlobal(name string, creator func(ctx context.Context, config Config) (Interface, error))
func NewProvider(ctx context.Context, name string, config Config) (Interface, error)
```

### 3. Functional Options Pattern
```go
type Option func(*Component)

func WithTimeout(timeout time.Duration) Option {
    return func(c *Component) { c.timeout = timeout }
}
```

### 4. Error Handling Pattern
```go
type Error struct {
    Op   string    // Operation that failed
    Err  error     // Underlying error
    Code ErrorCode // Error classification
}
```

### 5. OpenTelemetry Integration (MANDATORY)
Every package must implement OTEL metrics in `metrics.go`:
```go
type Metrics struct {
    operationsTotal   metric.Int64Counter
    operationDuration metric.Float64Histogram
    errorsTotal       metric.Int64Counter
    tracer            trace.Tracer
}
```

## Code Style Guidelines

### Import Organization
```go
import (
    // Standard library
    "context"
    "fmt"

    // Third-party packages
    "go.opentelemetry.io/otel/trace"

    // Internal packages
    "github.com/lookatitude/beluga-ai/pkg/schema"
)
```

### Naming Conventions
- **Packages**: lowercase, singular forms (`agent`, `llm`, not `agents`, `llms`)
- **Interfaces**: "er" suffix for single-method (`Embedder`), nouns for multi-method (`VectorStore`)
- **Functions**: Clear action verbs (`New`, `Get`, `Register`)

### Context Propagation
Always propagate context for cancellation, timeouts, and tracing:
```go
func (c *Component) Process(ctx context.Context, input interface{}) error {
    ctx, span := c.tracer.Start(ctx, "component.process")
    defer span.End()
    // ...
}
```

## Testing Requirements

### Test Files Required Per Package
- `test_utils.go` - Advanced mocking utilities
- `advanced_test.go` - Comprehensive test suites
- `{package}_test.go` - Unit tests

### Test Patterns
```go
// Table-driven tests (required pattern)
func TestComponent(t *testing.T) {
    tests := []struct {
        name          string
        input         interface{}
        expectedError bool
    }{
        // Test cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Running Tests
```bash
# Run with race detection (recommended)
go test -race -v ./pkg/...

# Run specific package
go test -v ./pkg/llms/...

# Run integration tests
go test -v -timeout=15m ./tests/integration/...
```

## Commit Convention

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types
- `feat`: New feature (MINOR version bump)
- `fix`: Bug fix (PATCH version bump)
- `docs`: Documentation changes
- `test`: Test additions/changes
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Dependency updates, etc.

### Breaking Changes
```
feat: allow provided config object to extend other configs

BREAKING CHANGE: `extends` key in config file is now used for extending
```

## Dependencies

### Key External Dependencies
- **LLM SDKs**: `github.com/sashabaranov/go-openai`, `github.com/anthropics/anthropic-sdk-go`
- **AWS**: `github.com/aws/aws-sdk-go-v2` (for Bedrock)
- **Observability**: `go.opentelemetry.io/otel` v1.39.0
- **Configuration**: `github.com/spf13/viper`
- **Testing**: `github.com/stretchr/testify`
- **Validation**: `github.com/go-playground/validator/v10`
- **Voice**: `github.com/livekit/protocol`, `github.com/twilio/twilio-go`
- **Workflow**: `go.temporal.io/sdk`

## Common Tasks

### Adding a New Provider
1. Create provider file in `providers/` directory
2. Implement the package's main interface
3. Register in global registry with `init()` function
4. Add comprehensive tests in `advanced_test.go`
5. Update package README.md

### Adding New Functionality
1. Check existing patterns in similar packages
2. Follow the package structure convention
3. Implement OTEL metrics for observability
4. Add comprehensive tests (table-driven, concurrency, error handling)
5. Update documentation

### Running Full CI Locally
```bash
make ci-local
```
This runs: format check, lint, vet, security scans, unit tests, integration tests, coverage check, and build verification.

## Important Notes

### Linting
- Uses golangci-lint v2.6.2
- The `react` package under `pkg/agents/providers/react` is excluded due to a golangci-lint bug
- Run `make lint-fix` to auto-fix issues

### Coverage
- Advisory threshold: 80% (does not block CI)
- Reports generated in `coverage/` directory

### Security
- gosec for static security analysis
- govulncheck for dependency vulnerabilities
- gitleaks for secret detection

### Pre-commit Hooks
Configure with:
```bash
pip install pre-commit
pre-commit install
```

## Documentation References

- [Architecture](./docs/architecture.md) - System architecture and design
- [Package Design Patterns](./docs/package_design_patterns.md) - Design conventions (MUST READ)
- [Best Practices](./docs/best-practices.md) - Production patterns
- [Contributing](./CONTRIBUTING.md) - Development workflow
- [Quick Start](./docs/quickstart.md) - Getting started guide
- [Examples](./examples/README.md) - Runnable examples

## Provider Support

### LLM Providers
- OpenAI (GPT-4, GPT-3.5)
- Anthropic (Claude)
- AWS Bedrock
- Ollama (local models)
- Gemini, Google, Grok, Groq

### Vector Store Providers
- In-Memory
- PgVector (PostgreSQL)
- Pinecone

### Embedding Providers
- OpenAI
- Ollama

### Voice Providers
- LiveKit
- Twilio
- WebRTC (Pion)
