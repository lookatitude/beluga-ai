---
name: status
description: Check package health and structure for Beluga AI v2. Shows which packages have tests, registries, hooks, and middleware.
---

Check the health and structure of Beluga AI v2 packages.

Run the following checks and present a status table:

```bash
# Check which top-level packages exist
for pkg in core schema config o11y llm tool memory rag agent voice orchestration workflow protocol guard resilience cache hitl auth eval state prompt server internal; do
    if [ -d "$pkg" ]; then
        files=$(find "$pkg" -name "*.go" ! -name "*_test.go" | wc -l)
        tests=$(find "$pkg" -name "*_test.go" | wc -l)
        echo "✅ $pkg ($files files, $tests test files)"
    else
        echo "❌ $pkg (not found)"
    fi
done
```

For each existing package, also check:
- Does it have a registry? (grep for "func Register")
- Does it have hooks? (grep for "type Hooks struct")
- Does it have middleware? (grep for "type Middleware func")
- Does it compile? (`go build ./$pkg/...`)

Present results as a clear status table showing: Package | Files | Tests | Registry | Hooks | Middleware | Compiles
