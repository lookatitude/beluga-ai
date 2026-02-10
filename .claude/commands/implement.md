---
name: implement
description: Implement a Beluga AI v2 package. Reads architecture, plans, builds, tests, verifies.
---

Implement the specified package for Beluga AI v2.

## Steps

1. Read `docs/concepts.md` and `docs/packages.md` for the package's role and interfaces.
2. Verify dependencies exist. If missing, implement them first.
3. Plan: list files, define interfaces first, identify extension points (registry, hooks, middleware).
4. Implement per CLAUDE.md conventions:
   - Compile-time interface checks: `var _ Interface = (*Impl)(nil)`
   - Doc comments on all exports
   - Registry/hooks/middleware where applicable
5. Write tests: `*_test.go` alongside source, table-driven, error paths, context cancellation.
6. Verify: `go build ./...`, `go vet ./...`, `go test ./...`

If no package specified, ask which one and suggest based on dependency order.
