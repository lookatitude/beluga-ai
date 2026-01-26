# Op/Err/Code Struct

Every package's main error type MUST have:

- **Op** — operation that failed (e.g. `"GetProvider"`, `"Execute"`)
- **Err** — underlying error; implement `Unwrap() error` to return it
- **Code** — stable string for callers to branch on (retries, client behavior)

Optional: **Message** when you need a safe, user-facing string different from `Err.Error()`; otherwise rely on `Err`. Optional: **Fields** / **Details** / **Context** for structured metadata.

```go
type XxxError struct {
    Op      string
    Err     error
    Code    string
    Message string // optional
}

func (e *XxxError) Error() string { /* include Op, Message or Err, Code */ }
func (e *XxxError) Unwrap() error { return e.Err }
```

**Why:** Stable `Code` lets callers decide retries, user messaging, and behavior without parsing `Error()`.
