---
name: Design Component
description: Component design following ISP, DIP, SRP principles
personas:
  - architect
---

# Design Component

This skill guides you through designing a new component for Beluga AI following established architectural principles and patterns.

## Prerequisites

- Clear understanding of the component's purpose
- Knowledge of which layer it belongs to
- List of required functionality

## Steps

### 1. Define Component Purpose

Answer these questions:

1. **What problem does this solve?**
2. **What are the inputs and outputs?**
3. **What dependencies does it need?**
4. **How will it be extended in the future?**

Document in design spec:

```markdown
## Component: [Name]

### Purpose
[What problem this solves]

### Responsibilities
1. [Primary responsibility]
2. [Secondary responsibility]

### Non-Responsibilities
- [What this component does NOT do]
```

### 2. Identify Architecture Layer

Determine where this component fits:

```
1. Application Layer      - examples/
2. Agent Layer           - pkg/agents/, pkg/orchestration/
3. LLM Layer             - pkg/llms/, pkg/chatmodels/
4. RAG Layer             - pkg/retrievers/, pkg/vectorstores/
5. Memory Layer          - pkg/memory/
6. Infrastructure Layer  - pkg/core/, pkg/config/, pkg/monitoring/
```

**Rule**: Dependencies can only point downward.

### 3. Design Interfaces (ISP)

Apply Interface Segregation Principle:

```go
// BAD: God interface
type Storage interface {
    Get(key string) ([]byte, error)
    Set(key string, value []byte) error
    Delete(key string) error
    List(prefix string) ([]string, error)
    Watch(key string) <-chan Event
    Transaction(ops []Op) error
    Backup(path string) error
    Restore(path string) error
}

// GOOD: Segregated interfaces
type Reader interface {
    Get(ctx context.Context, key string) ([]byte, error)
}

type Writer interface {
    Set(ctx context.Context, key string, value []byte) error
    Delete(ctx context.Context, key string) error
}

type Lister interface {
    List(ctx context.Context, prefix string) ([]string, error)
}

type Watcher interface {
    Watch(ctx context.Context, key string) <-chan Event
}

// Compose when needed
type ReadWriter interface {
    Reader
    Writer
}
```

### 4. Define Dependencies (DIP)

Apply Dependency Inversion Principle:

```go
// BAD: Concrete dependency
type Agent struct {
    llm *openai.Client  // Concrete type
}

// GOOD: Interface dependency
type Agent struct {
    llm LLMCaller  // Interface
}

// Constructor injection
func NewAgent(llm LLMCaller, opts ...Option) *Agent {
    return &Agent{llm: llm}
}
```

### 5. Design Configuration

```go
// Configuration with validation
type Config struct {
    // Required fields
    Name    string        `mapstructure:"name" validate:"required"`
    Timeout time.Duration `mapstructure:"timeout" validate:"required,min=1s"`

    // Optional with defaults
    MaxRetries int  `mapstructure:"max_retries" validate:"gte=0,lte=10"`
    Debug      bool `mapstructure:"debug"`
}

func DefaultConfig() Config {
    return Config{
        Timeout:    30 * time.Second,
        MaxRetries: 3,
    }
}

// Functional options for runtime configuration
type Option func(*Component)

func WithLogger(l Logger) Option {
    return func(c *Component) { c.logger = l }
}

func WithMetrics(m *Metrics) Option {
    return func(c *Component) { c.metrics = m }
}
```

### 6. Plan Error Handling

```go
// Error codes for this component
type ErrorCode string

const (
    ErrCodeNotFound     ErrorCode = "NOT_FOUND"
    ErrCodeInvalidInput ErrorCode = "INVALID_INPUT"
    ErrCodeTimeout      ErrorCode = "TIMEOUT"
    ErrCodeConflict     ErrorCode = "CONFLICT"
)

// Structured error type
type Error struct {
    Op      string    // Operation: "Component.Method"
    Err     error     // Underlying error
    Code    ErrorCode // Classification
    Details map[string]interface{} // Additional context
}
```

### 7. Design for Observability

```go
// Metrics to implement
type Metrics struct {
    // Counters
    operationsTotal metric.Int64Counter   // {component}_operations_total
    errorsTotal     metric.Int64Counter   // {component}_errors_total

    // Histograms
    operationDuration metric.Float64Histogram // {component}_operation_duration_seconds

    // Gauges (if applicable)
    activeConnections metric.Int64Gauge // {component}_active_connections

    // Tracer
    tracer trace.Tracer
}
```

### 8. Plan Extension Points

If component supports multiple implementations:

```go
// Provider registry pattern
type ProviderRegistry struct {
    mu       sync.RWMutex
    creators map[string]CreatorFunc
}

type CreatorFunc func(ctx context.Context, config Config) (Interface, error)

func RegisterGlobal(name string, creator CreatorFunc)
func NewProvider(ctx context.Context, name string, config Config) (Interface, error)
```

### 9. Document Design

Create design document:

```markdown
## Component Design: [Name]

### Purpose
[Problem being solved]

### Architecture Layer
[Which layer and why]

### Interfaces
```go
type [Name] interface {
    // Methods
}
```

### Dependencies
- [Dependency 1]: [Why needed]
- [Dependency 2]: [Why needed]

### Configuration
- [Option 1]: [Purpose]
- [Option 2]: [Purpose]

### Error Handling
- [Error code]: [When raised]

### Observability
- Metrics: [List]
- Traces: [Spans]

### Extension Points
- [How to extend]

### Trade-offs
- Chose X over Y because [reason]
```

## Design Checklist

### Interfaces
- [ ] Small and focused (ISP)
- [ ] Single-method uses `-er` suffix
- [ ] Multi-method uses descriptive noun
- [ ] Context is first parameter
- [ ] Returns error as last return value

### Dependencies
- [ ] All dependencies are interfaces (DIP)
- [ ] Constructor injection used
- [ ] No global mutable state
- [ ] Dependencies point downward only

### Single Responsibility
- [ ] One clear purpose
- [ ] No mixed concerns
- [ ] Clear boundaries

### Extensibility
- [ ] Functional options for configuration
- [ ] Provider pattern if multi-implementation
- [ ] Open for extension, closed for modification

### Observability
- [ ] OTEL metrics planned
- [ ] Tracing spans defined
- [ ] Structured logging approach

## Output

A complete design document with:
- Interface definitions
- Dependency analysis
- Configuration design
- Error handling plan
- Observability design
- Trade-off documentation
