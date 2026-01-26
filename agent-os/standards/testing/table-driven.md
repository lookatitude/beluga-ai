# Table-Driven Tests

Use table-driven tests for almost all tests. One long, linear test is acceptable only in rare cases (e.g. a single integration scenario).

**Shape:**

```go
tests := []struct {
    name        string
    description string // optional, for t.Logf
    setup       func() T
    input       X
    wantErr     bool
    validate    func(*testing.T, Result, error)
}{
    {name: "happy_path", description: "…", setup: ..., input: ..., validate: ...},
    {name: "error_case", wantErr: true, ...},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        if tt.description != "" { t.Logf("%s", tt.description) }
        // setup, act, validate
    })
}
```

**Naming:** `name` is a short `snake_case` id for `t.Run`. `description` is optional and used in `t.Logf`.
