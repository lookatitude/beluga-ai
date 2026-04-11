# Pattern: Testing

**Status:** stub — populate with `/wiki-learn`

## Contract

- `*_test.go` alongside source in the same package.
- Table-driven tests. Subtests via `t.Run(name, ...)`.
- Cover: happy path, error paths, edge cases, context cancellation for streams.
- `-race` flag always. Integration tests behind `//go:build integration`.
- Benchmarks for hot paths.
- Use `internal/testutil/` mocks — never hand-rolled.

## Canonical example

(populate via `/wiki-learn`)

## Anti-patterns

- Tests that depend on wall clock.
- Tests that share state between cases via package-level variables.
- Goroutine-leaking tests (always assert cleanup via `goleak` or explicit joins).
- Skipping tests in CI without tracking.

## Related

- `patterns/streaming.md`
- `architecture/invariants.md`
