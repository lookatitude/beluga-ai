# Standards Applied

## 1. backend/op-err-code

All errors use the Op/Err/Code pattern:

```go
type Error struct {
    Op      string    // Operation that failed (e.g., "Build", "Run")
    Err     error     // Underlying error
    Code    string    // Error classification (e.g., "missing_llm")
    Fields  map[string]any // Additional context
}

func (e *Error) Error() string {
    return fmt.Sprintf("convenience/agent %s: %v (code: %s)", e.Op, e.Err, e.Code)
}

func (e *Error) Unwrap() error {
    return e.Err
}
```

## 2. backend/factory-signature

Build() methods follow the factory signature pattern:

```go
func (b *Builder) Build(ctx context.Context) (Interface, error)
```

Key requirements:
- Context as first parameter for cancellation/tracing
- Return interface type (not concrete struct)
- Return error as second value
- Never panic

## 3. backend/otel-spans

Tracing in key operations:

```go
func (a *convenienceAgent) Run(ctx context.Context, input string) (string, error) {
    ctx, span := a.tracer.Start(ctx, "convenience.agent.run")
    defer span.End()

    // Record attributes
    span.SetAttributes(
        attribute.String("input_length", fmt.Sprintf("%d", len(input))),
    )

    // ... implementation

    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    }
    return result, nil
}
```

## 4. global/required-files

Each package MUST have:
- `errors.go` - Error types and codes
- `metrics.go` - OTEL metrics implementation
- `test_utils.go` - Mock factories and test helpers
- `advanced_test.go` - Comprehensive test suite

## 5. testing/advanced-test

Tests must include:
- Table-driven tests for all public functions
- Concurrency tests for thread-safe operations
- Error branch coverage (all error codes tested)
- Mock-based isolation

```go
func TestAgent_Run(t *testing.T) {
    tests := []struct {
        name          string
        input         string
        setupMock     func(*MockLLM)
        expectedError string
        wantErr       bool
    }{
        {
            name:  "successful run",
            input: "Hello",
            setupMock: func(m *MockLLM) {
                m.EXPECT().Invoke(gomock.Any(), "Hello").Return("Hi", nil)
            },
            wantErr: false,
        },
        {
            name:  "LLM error",
            input: "Hello",
            setupMock: func(m *MockLLM) {
                m.EXPECT().Invoke(gomock.Any(), "Hello").Return("", errors.New("api error"))
            },
            expectedError: "execution_failed",
            wantErr:       true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ... test implementation
        })
    }
}
```

## 6. Package Structure Convention

```
pkg/convenience/{package}/
├── {package}.go        # Builder and NewBuilder()
├── types.go            # Public interfaces
├── {package}_impl.go   # Implementation
├── errors.go           # Error types and codes
├── metrics.go          # OTEL metrics
├── test_utils.go       # Test helpers
├── {package}_test.go   # Unit tests
├── advanced_test.go    # Comprehensive tests
└── README.md           # Documentation
```
