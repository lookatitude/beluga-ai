---
description: Go framework best practices for Beluga AI v2. Applies to all Go code in this project.
globs: "*.go"
alwaysApply: false
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
