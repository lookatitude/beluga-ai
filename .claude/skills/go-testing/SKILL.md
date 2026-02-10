---
name: go-testing
description: Go testing patterns for Beluga AI v2. Use when writing tests, mocks, or benchmarks.
---

# Go Testing Patterns

## Table-Driven Tests

```go
func TestGenerate(t *testing.T) {
    tests := []struct {
        name    string
        input   []schema.Message
        want    *schema.AIMessage
        wantErr error
    }{
        {name: "simple", input: []schema.Message{schema.HumanMessage("hello")}, want: &schema.AIMessage{...}},
        {name: "empty input errors", input: nil, wantErr: &core.Error{Code: core.ErrInvalidInput}},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := model.Generate(context.Background(), tt.input)
            if tt.wantErr != nil { require.Error(t, err); return }
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

## Stream + Context Cancellation

```go
func TestStreamCancellation(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    count := 0
    for _, err := range model.Stream(ctx, msgs) {
        count++
        if count == 2 { cancel() }
        if err != nil { assert.ErrorIs(t, err, context.Canceled); break }
    }
    assert.LessOrEqual(t, count, 3)
}
```

## Mocks

Location: `internal/testutil/mock<package>/`. Every mock has configurable function fields + error injection + call counting.

## Rules

- `*_test.go` alongside source, not in separate dirs.
- Use `testify/assert` (non-fatal) and `testify/require` (fatal).
- Integration tests: `//go:build integration` tag, `*_integration_test.go`.
- Benchmarks: `*_bench_test.go`, `b.ReportAllocs()`, `b.RunParallel()`.
- Compile-time check: `var _ Interface = (*Impl)(nil)`.
- Always test: error paths, context cancellation for streams, nil/empty inputs.
