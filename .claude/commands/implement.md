---
name: implement
description: Start implementing a Beluga AI v2 package. Reads architecture docs, plans the implementation, then executes.
---

Implement a Beluga AI v2 package. Follow this workflow:

1. **Read architecture docs** — Read `docs/concepts.md` and `docs/packages.md` to understand the package's role, interfaces, and dependencies.

2. **Check dependencies** — Verify that packages this one depends on already exist. If not, list what's missing and implement dependencies first.

3. **Plan** — Create a brief implementation plan:
   - List all files to create
   - Define interfaces first
   - Identify extension points (registry, hooks, middleware)
   - Note which providers to implement

4. **Implement** — For each file:
   - Write the code following CLAUDE.md conventions
   - Add compile-time interface checks: `var _ Interface = (*Impl)(nil)`
   - Include doc comments on all exports
   - Follow the registry/hooks/middleware pattern where applicable

5. **Test** — Write tests alongside implementation:
   - `*_test.go` next to source
   - Table-driven tests
   - Test error paths
   - Test context cancellation for streams

6. **Verify** — Run:
   ```
   go build ./...
   go vet ./...
   go test ./...
   ```

If the user specifies a package name, implement that package. Otherwise, ask which package to implement and suggest based on package dependency order and architecture docs.
