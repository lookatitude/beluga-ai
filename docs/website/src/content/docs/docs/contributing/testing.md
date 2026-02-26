---
title: "Testing Guide — Beluga AI"
description: "Write and run tests for Beluga AI. Table-driven tests, mocks, integration tests, fuzz testing, benchmarks, and streaming code testing patterns."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI testing, Go test patterns, table-driven tests, mock interfaces, integration tests Go, fuzz testing, benchmarks"
---

Testing is a critical part of the Beluga AI development process. With 2,885 tests across 157 packages, the test suite serves as both a safety net and a specification — tests document how each interface behaves, what edge cases are handled, and how components interact. This guide covers how to run tests, write new ones, and follow the testing conventions used throughout the project.

## Running Tests

| Command | Description |
|---|---|
| `make test` | Run all unit tests |
| `make test-verbose` | Run unit tests with verbose output |
| `make integration-test` | Run integration tests (requires external services) |
| `make coverage` | Generate an HTML coverage report |
| `make bench` | Run benchmarks |
| `make fuzz` | Run fuzz tests |

You can also run tests for a specific package:

```bash
go test ./llm/...
go test -v ./agent/... -run TestAgentHandoff
```

## Unit Test Conventions

### File Placement

Test files live alongside the source code they test. This co-location makes it easy to find tests for any source file and ensures that tests are updated when the source changes:

```
llm/
├── router.go
├── router_test.go
├── structured.go
└── structured_test.go
```

### Table-Driven Tests

Table-driven tests are the preferred pattern throughout Beluga AI. They make it easy to add new cases, provide clear failure messages that identify which specific case failed, and separate test data from test logic. When you need to test a function with multiple inputs, a table-driven test is almost always the right approach:

```go
func TestParseTemperature(t *testing.T) {
    tests := []struct {
        name    string
        input   float64
        wantErr bool
    }{
        {name: "valid zero", input: 0.0, wantErr: false},
        {name: "valid mid", input: 1.0, wantErr: false},
        {name: "valid max", input: 2.0, wantErr: false},
        {name: "negative", input: -0.1, wantErr: true},
        {name: "too high", input: 2.1, wantErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateTemperature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("validateTemperature(%v) error = %v, wantErr %v",
                    tt.input, err, tt.wantErr)
            }
        })
    }
}
```

### Subtests

Use `t.Run()` to create subtests for logical grouping:

```go
func TestAgent(t *testing.T) {
    t.Run("Run", func(t *testing.T) { /* ... */ })
    t.Run("Stream", func(t *testing.T) { /* ... */ })
    t.Run("Handoff", func(t *testing.T) { /* ... */ })
}
```

## Using Mocks

Every public interface in Beluga AI has a corresponding mock in `internal/testutil/`. Use these shared mocks in your tests instead of creating ad-hoc implementations. This ensures consistent test behavior across the codebase and reduces the maintenance burden of keeping multiple mock implementations in sync with interface changes:

```go
import "github.com/lookatitude/beluga-ai/internal/testutil"

func TestAgentWithMockModel(t *testing.T) {
    mock := &testutil.MockChatModel{
        GenerateFunc: func(ctx context.Context, msgs []schema.Message, opts ...llm.Option) (*schema.Message, error) {
            return &schema.Message{
                Role:    schema.RoleAssistant,
                Content: "Hello from mock!",
            }, nil
        },
    }

    a := agent.New("test-agent", agent.WithModel(mock))
    result, err := a.Run(context.Background(), "Hi")
    if err != nil {
        t.Fatal(err)
    }
    if result != "Hello from mock!" {
        t.Errorf("unexpected result: %s", result)
    }
}
```

## Integration Tests

Integration tests interact with external services (databases, APIs, etc.) and are separated from unit tests using a build tag. This separation ensures that `make test` runs quickly without requiring external infrastructure, while `make integration-test` provides full end-to-end validation when needed.

### Build Tag

Add the following build constraint at the top of integration test files:

```go
//go:build integration

package llm_test
```

### Running

```bash
# Run all integration tests
make integration-test

# Run integration tests for a specific package
go test -tags=integration ./llm/providers/openai/...
```

Integration tests are run in CI on every PR but may require environment variables for API keys and service URLs. Check each provider's test file for required configuration.

## Fuzz Testing

Fuzz tests help find edge cases and unexpected inputs that table-driven tests might miss. They are especially valuable for parsing functions, serialization logic, and anything that processes untrusted input. Use the standard Go fuzzing framework:

```bash
# Run all fuzz tests
make fuzz

# Run a specific fuzz test
go test -fuzz=FuzzParseMessage ./schema/...
```

### Naming Convention

Fuzz test functions must start with `Fuzz`:

```go
func FuzzParseMessage(f *testing.F) {
    f.Add([]byte(`{"role":"user","content":"hello"}`))

    f.Fuzz(func(t *testing.T, data []byte) {
        msg, err := schema.ParseMessage(data)
        if err != nil {
            return // invalid input is fine
        }
        // If parsing succeeds, re-marshaling should not fail
        _, err = json.Marshal(msg)
        if err != nil {
            t.Errorf("re-marshal failed: %v", err)
        }
    })
}
```

## Benchmarks

Write benchmarks for hot paths such as streaming, tool execution, and retrieval. Benchmarks are especially important for the `core/` stream utilities and the agent executor loop, where per-event overhead directly impacts latency:

```bash
# Run all benchmarks
make bench

# Run benchmarks for a specific package
go test -bench=. -benchmem ./core/...
```

### Writing Benchmarks

```go
func BenchmarkStreamProcessing(b *testing.B) {
    agent := setupTestAgent()
    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        for event, err := range agent.Stream(ctx, "test input") {
            if err != nil {
                b.Fatal(err)
            }
            _ = event
        }
    }
}
```

## Code Coverage

Aim for high test coverage, especially for core packages and public APIs. Coverage reports help identify untested code paths, but coverage alone does not guarantee correctness — focus on meaningful assertions that verify behavior, not just line execution:

```bash
# Generate coverage report
make coverage

# View in browser (generates coverage.html)
go tool cover -html=coverage.out -o coverage.html
```

Coverage reports are generated in CI and included in PR checks.

## Testing Streaming Code

Testing `iter.Seq2` based streaming requires collecting events into a slice before making assertions. The `collectEvents` helper function below is the standard pattern used throughout the test suite. This pattern converts the push-based iterator into a collected slice that you can assert on with standard Go testing tools:

```go
func collectEvents(seq iter.Seq2[schema.Event, error]) ([]schema.Event, error) {
    var events []schema.Event
    for event, err := range seq {
        if err != nil {
            return events, err
        }
        events = append(events, event)
    }
    return events, nil
}

func TestAgentStream(t *testing.T) {
    a := setupTestAgent()
    events, err := collectEvents(a.Stream(context.Background(), "Hello"))
    if err != nil {
        t.Fatal(err)
    }
    if len(events) == 0 {
        t.Error("expected at least one event")
    }
}
```

## CI Checks

All tests run automatically on every pull request via GitHub Actions. The CI pipeline runs:

1. **Lint** — `go vet` and `golangci-lint` (13 linters including gosec, staticcheck, errcheck)
2. **Build** — `go build ./...` and `go mod tidy` verification
3. **Unit tests** — `go test -race` with coverage reporting
4. **Integration tests** — `go test -race -tags integration`
5. **Security scans** — Snyk (dependency vulnerabilities), Trivy (filesystem scanning), govulncheck (Go vulnerability database), gosec (static security analysis), Gitleaks (secret detection), go-licenses (license compliance)
6. **SonarCloud** — Code quality, duplication detection, and maintainability analysis
7. **Greptile** — AI-powered code review (internal PRs)

All checks must pass before a PR can be merged. See the [Pull Request Process](/docs/contributing/pull-requests/) for details.
