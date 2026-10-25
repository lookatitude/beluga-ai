# Testing Pattern

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
