# React Package Exclusion

Hardcoded exclusion of react package due to golangci-lint bug.

```makefile
lint: ## Run golangci-lint
    @echo "Running golangci-lint (excluding react package due to golangci-lint v2.6.2 panic bug)..."
    @packages=$$(go list ./pkg/... ./tests/... 2>/dev/null | \
        grep -vE "github.com/lookatitude/beluga-ai/pkg/agents/providers/react|..." | \
        sed 's|github.com/lookatitude/beluga-ai/||' | tr '\n' ' '); \
    golangci-lint run --timeout=5m $$packages
```

## Why Excluded?
- golangci-lint v2.6.2 panics on this specific package
- Bug in linter, not in code
- Excluding is safer than disabling linting entirely

## Affected Package
```
pkg/agents/providers/react
```

## When to Remove
When golangci-lint is upgraded past v2.6.2 and the panic is fixed:
1. Test: `golangci-lint run ./pkg/agents/providers/react`
2. If passes, remove from exclusion list

## Documentation
Comment in Makefile explains the workaround - don't remove without testing.
