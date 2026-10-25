---
type: standard
title: "Testing Pattern"
description: "Migrated from .wiki/patterns/testing.md."
owner: orchestrator
confidence: high
trust_tier: reviewed
importance: high
source_refs: [wiki-patterns-testing]
created_at: 2026-04-11
updated_at: 2026-04-11
expires_at: null
supersedes: null
sensitivity: internal
recall_importance: 3
---
# Testing Pattern

> Migrated from `.wiki/patterns/testing.md` on 2026-10-25 (Guild 3.x; structure kept). Verbatim original: `.guild/knowledge/sources/wiki-patterns-testing/original.md` (sha256 `9293e119b39d…`).

Table-driven tests with context cancellation, subtests, and error code validation.

## Canonical Example

**File:** `tool/tool_test.go:11-39`

```go
func TestTool(t *testing.T) {
	tests := []struct {
		name    string
		tool    Tool
		input   map[string]any
		wantErr bool
		check   func(t *testing.T, result *Result)
	}{
		{
			name: "success",
			tool: &mockTool{executeFn: func(_ map[string]any) (*Result, error) {
				return TextResult("ok"), nil
			}},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.tool.Execute(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}
```

## Variations

1. **Retry logic validation** — `tool/middleware_test.go:92-118`
   - Validates IsRetryable behavior
   - Tests retry exhaustion
   - Checks attempt count

2. **Context cancellation tests** — `tool/middleware_test.go:164-188`
   - Verifies early context cancellation stops retries
   - Validates error propagation

## Anti-Patterns

- **Single test per scenario**: Monolithic test functions; hard to isolate failures
- **Missing error code checks**: Validating only error presence, not IsRetryable classification
- **Not testing context cancellation**: Missing deadline/cancellation edge cases
- **Hardcoded sleep durations**: Flaky tests; use fake clocks or shorter timeouts

## Invariants

- Each test case runs in isolation via t.Run()
- Failed assertion includes both expected and actual values
- Context cancellation always results in ctx.Err() propagation
- Error code (e.Code) always checked with core.IsRetryable validation
