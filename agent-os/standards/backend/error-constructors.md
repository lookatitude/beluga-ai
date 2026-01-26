# Error Constructors and Helpers

**Base constructor:** `NewXxxError(op, code, err)` — use for all errors.

**Message constructor:** `NewXxxErrorWithMessage(op, code, message, err)` — use when you need a custom, client-facing message instead of `Err.Error()`.

Optional: `WithField` / `WithDetails` / `AddContext` for structured metadata on the error value.

**Shortcuts:** Helpers like `ErrTimeout(op, err)`, `ErrInvalidConfig(op, err)` are allowed. Define them in `errors.go` for the main package type, or in `iface/errors.go` when the error type lives in iface. Shortcuts call the base constructor with the right code.

```go
func ErrTimeout(op string, err error) *XxxError {
    return NewXxxError(op, ErrCodeTimeout, err)
}
```
