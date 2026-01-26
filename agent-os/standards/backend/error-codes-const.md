# Error Codes as Const

Use `ErrCodeXxx` for the Go symbol and `"snake_case"` for the string. No exceptions.

```go
const (
    ErrCodeInvalidConfig = "invalid_config"
    ErrCodeTimeout       = "timeout"
    ErrCodeStreamError   = "stream_error"
)
```

**Shared codes:** For config-related errors, use codes from `pkg/config` (e.g. `config.ErrCodeInvalidConfig`, `config/iface.ErrCodeLoadFailed`). Do not redefine them in other packages.

**Package-specific codes:** All other codes are defined in the package that uses them — in `errors.go` or, when used by public interfaces, in `iface/errors.go`.
