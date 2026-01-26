# Package Design Patterns Refactor - Relevant Standards

## Global Standards

### required-files.md
Every package MUST have:
- `iface/` directory with interfaces
- `config.go` with configuration structs
- `metrics.go` with OTEL metrics
- `errors.go` with Op/Err/Code pattern
- `test_utils.go` with mock factories
- `advanced_test.go` with comprehensive tests
- `README.md` with documentation

### provider-subpackage-layout.md
Multi-provider packages use:
```
pkg/{package}/
├── providers/
│   ├── {provider1}/
│   ├── {provider2}/
│   └── ...
```

### naming.md
- Packages: lowercase, singular (agent, llm)
- Interfaces: "er" suffix for single-method, nouns for multi-method
- Functions: Clear action verbs (New, Get, Register)

## Backend Standards

### registry-shape.md
```go
type Registry struct {
    factories map[string]Factory
    mu        sync.RWMutex
}

func GetRegistry() *Registry
func (r *Registry) Register(name string, factory Factory)
func (r *Registry) GetProvider(name string, cfg Config) (Interface, error)
func (r *Registry) ListProviders() []string
```

### factory-signature.md
```go
type Factory func(ctx context.Context, config *Config) (Interface, error)
```

### op-err-code.md
```go
type Error struct {
    Op   string    // Operation name
    Err  error     // Underlying error
    Code ErrorCode // Error classification
}
```

### otel-naming.md
- Meter name: `beluga.{package}`
- Counter: `{package}.{operation}.total`
- Histogram: `{package}.{operation}.duration`
- Error counter: `{package}.errors.total`

### metrics-go-shape.md
```go
type Metrics struct {
    operationsTotal   metric.Int64Counter
    operationDuration metric.Float64Histogram
    errorsTotal       metric.Int64Counter
    tracer            trace.Tracer
}

func NewMetrics(meterProvider metric.MeterProvider) (*Metrics, error)
func (m *Metrics) RecordOperation(ctx context.Context, op string, duration time.Duration)
func (m *Metrics) RecordError(ctx context.Context, op string, err error)
```

## Testing Standards

### table-driven.md
```go
func TestComponent(t *testing.T) {
    tests := []struct {
        name    string
        input   interface{}
        want    interface{}
        wantErr bool
    }{
        // test cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

### test-utils.md
```go
// Mock factories for each interface
func NewMockInterface(opts ...MockOption) *MockInterface

// Test configuration helpers
func NewTestConfig(opts ...ConfigOption) *Config

// Context helpers with timeout
func TestContext(t *testing.T) context.Context
```

### advanced-test.md
Advanced tests MUST cover:
- Success cases
- Error cases
- Concurrency safety
- Edge cases
- Performance (benchmarks)

### concurrency-and-errors.md
```go
func TestConcurrency(t *testing.T) {
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            // concurrent operation
        }()
    }
    wg.Wait()
}
```

## Voice Standards

### backend-and-registry.md
Voice backend uses registry for providers:
- LiveKit
- Vapi
- Vocode
- Pipecat
- Cartesia
- Twilio

### stt-tts-vad-transport-etc.md
Each voice component is an independent sub-package with:
- Own registry
- Own config
- Own metrics
- Own tests

### session-and-config.md
Session management for voice interactions:
- Session state machine
- Config propagation to sub-packages
- Timeout handling

## Safety Standards

### middleware-pattern.md
Safety as middleware wrapper:
```go
type SafetyMiddleware struct {
    checker SafetyChecker
    agent   agents.Agent
}

func (m *SafetyMiddleware) Execute(ctx context.Context, input string) (string, error) {
    if result := m.checker.Check(input); !result.Safe {
        return "", result.Error()
    }
    return m.agent.Execute(ctx, input)
}
```

### risk-scoring.md
```go
type SafetyResult struct {
    Safe       bool
    RiskScore  float64
    Violations []Violation
}
```

## Index Structure (index.yml format)

```yaml
global:
  required-files:
    description: Mandatory files for every package
  provider-subpackage-layout:
    description: Provider organization in multi-provider packages
  naming:
    description: Package, interface, and function naming conventions

backend:
  registry-shape:
    description: Global registry pattern for provider management
  factory-signature:
    description: Standard factory function signature
  op-err-code:
    description: Error struct with Op, Err, Code fields
  otel-naming:
    description: OTEL meter, counter, histogram naming
  metrics-go-shape:
    description: Standard metrics.go structure

testing:
  table-driven:
    description: Table-driven test pattern
  test-utils:
    description: Test utilities and mock factories
  advanced-test:
    description: Comprehensive test suite requirements
  concurrency-and-errors:
    description: Concurrency and error testing patterns
```
