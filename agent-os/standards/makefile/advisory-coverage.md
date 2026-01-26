# Advisory Coverage Threshold

Coverage check exits 0 even when below threshold.

```makefile
test-coverage-threshold: ## Check if coverage meets 80% threshold (advisory)
    @pct=$$(go tool cover -func=$(COVERAGE_FILE) | tail -n1 | awk '{print $$3}' | sed 's/%//'); \
    threshold=80; \
    if awk "BEGIN {exit !($$pct < $$threshold)}"; then \
        echo "⚠️  Coverage $$pct% is below minimum $$threshold% (advisory check - does not block)"; \
        exit 0;  # Still exit 0!
    else \
        echo "✅ Coverage $$pct% meets minimum $$threshold% requirement"; \
    fi
```

## Why Advisory (exit 0)?
- **Coverage is a metric, not a gate** - Track trend over time
- **Matches CI behavior** - CI also doesn't block on coverage
- **Avoids blocking development** - Let devs iterate freely

## Output Indicators
| Symbol | Meaning |
|--------|---------|
| ⚠️ | Below threshold (advisory warning) |
| ✅ | Meets threshold |
| ❌ | Error calculating coverage |

## Note
The `exit 0` is intentional. Change to `exit 1` only if coverage should block builds.
