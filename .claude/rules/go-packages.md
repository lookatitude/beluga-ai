---
description: Go package rules for Beluga AI v2. Auto-loaded when editing Go files in framework packages.
globs: "*.go"
alwaysApply: false
---

# Go Package Rules

## Provider Implementation Checklist

- [ ] Interface defined (≤4 methods)
- [ ] Registry with `Register`/`New`/`List` (sync.RWMutex-protected)
- [ ] Hooks struct — all fields optional, nil-safe
- [ ] Middleware type `func(T) T` + `ApplyMiddleware()`
- [ ] Provider auto-registers in `init()`
- [ ] Conformance test in `_test.go`
- [ ] OTel span at every exported method
- [ ] Errors wrapped with `core.Error` + `ErrorCode`

## Pre-commit verification gate (MANDATORY)

Before ANY `git commit` on a Go change, run and pass:

```bash
go build ./...                    # compile
go vet ./...                      # stdlib static analysis
go test -race ./...               # tests with race detector
go mod tidy && git diff --exit-code go.mod go.sum
gofmt -l . | grep -v ".claude/worktrees"
golangci-lint run ./...           # primary linter
gosec -quiet ./...                # security scanner
govulncheck ./...                 # known-CVE scanner
```

CI runs all of these on every PR (see `.github/workflows/_ci-checks.yml`
and `_security-checks.yml`). Catching issues locally is faster than
pushing and waiting for CI to fail.

**gosec focus areas** (these are the most commonly flagged in this repo):
- `G107` — avoid HTTP requests with tainted variables (sanitize URLs)
- `G112` — always set `http.Server.ReadHeaderTimeout`
- `G115` — integer overflow on type conversion (use range checks)
- `G201/G202` — SQL string formatting → use parameterised queries or
  validate the table/column name against an allowlist before interpolating
- `G304` — file inclusion via variable (call `filepath.Clean` and check
  the result is inside an allowed prefix before `os.ReadFile`)
- `G404` — use `crypto/rand` not `math/rand` for anything security-sensitive
- `errcheck` unhandled errors — use `_ = fn()` when intentional,
  otherwise propagate with `core.Errorf(code, "...: %w", err)`
- `G601` — context cancellation function must be called (always `defer cancel()`)

## Anti-rationalization

| Excuse | Counter |
|---|---|
| "Tests aren't needed for this small change" | Every change gets a test. No exceptions. Red/Green TDD. |
| "I'll add OTel instrumentation later" | Instrumentation ships WITH the code, never after. |
| "This interface needs 5 methods" | Split into two interfaces. Max 4 per interface. |
| "I'll use `interface{}` for flexibility" | Use generics. `interface{}` is never acceptable in public APIs. |
| "The existing code has no tests so I'll skip" | Write tests for the existing code AND your changes. |
| "I'll clean up error handling later" | Use `core.Error` + `ErrorCode` now. |

---

# Go Framework Rules

## Package Design

- Package names: lowercase, single-word. No `pkg/` prefix.
- One package = one responsibility. If a package does two things, split it.
- Export only what consumers need. Keep internal implementation unexported.
- Every exported type and function must have a doc comment.

## Interface Design

- Interfaces have 1-4 methods maximum. Compose larger surfaces from small interfaces.
- Define interfaces in the consumer package, not the provider.
- Every implementation must have a compile-time check: `var _ Interface = (*Impl)(nil)`
- Accept interfaces, return concrete types.

## Configuration

- Use functional options `WithX()` for all configurable types.
- Never require config structs in constructors — use variadic options.
- Validate config at construction time, not at usage time.
- Sensible defaults for every option — zero-config must work.

## Registry Pattern (ALL extensible packages)

Every extensible package MUST follow this exact pattern:

```go
var registry = make(map[string]Factory)

func Register(name string, f Factory) { registry[name] = f }  // called in init()
func New(name string, cfg Config) (Interface, error) { ... }   // factory lookup
func List() []string { ... }                                    // discovery
```

- Registration happens only in `init()`. No runtime mutations.
- `New()` returns a typed error if the name is not registered.

## Streaming

- Public streaming API uses `iter.Seq2[T, error]` — never channels.
- Use `iter.Pull()` only when pull semantics are specifically needed.
- Always respect `context.Context` cancellation in stream producers.
- Consumer pattern: `for event, err := range stream { if err != nil { break } }`

## Middleware

- Signature: `func(T) T` where T is the interface being wrapped.
- Apply outside-in (last added runs first).
- Middleware must preserve the full interface contract.

## Hooks

- All hook fields are optional (nil = skip).
- Composable via `ComposeHooks()`.
- Hooks observe and augment — they don't replace core logic.
- Hook errors are handled, not swallowed.

## Error Handling

- Return `(T, error)` — never panic for recoverable errors.
- Use typed errors from `core/errors.go` with `ErrorCode`.
- Wrap errors with `%w` to preserve the chain.
- Check `IsRetryable()` for LLM and tool errors before retry.
- Never expose internal details in errors returned to callers.

## Context

- `context.Context` is always the first parameter of every public function. No exceptions.
- Propagate context through all layers — never drop it.
- Use context for: cancellation, tracing, tenant ID, auth.
- Never store context in a struct field.

## Dependencies

- `core/` and `schema/`: zero external deps beyond stdlib + otel.
- No circular imports — dependency flows downward through layers.
- Prefer stdlib: `slog` for logging, `net/http` for transports, `encoding/json` for serialization.
- Provider packages may import provider SDKs.

## Concurrency

- Use bounded worker pools, not unbounded goroutines.
- Every goroutine must have a cancellation path via context.
- Use `sync.Pool` for hot-path allocations.
- Protect shared state with `sync.Mutex` or `sync.RWMutex` — channels only for signaling.
- Always `defer` cleanup for resources (connections, files, locks).

## Testing

- `*_test.go` alongside source in the same package.
- Table-driven tests preferred.
- Test: happy path, error paths, edge cases, context cancellation.
- Integration tests use `//go:build integration`.
- Benchmarks for hot paths.
- Use `internal/testutil/` mocks — every interface has a mock.
