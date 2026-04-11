# Error Handling Pattern

Structured errors with ErrorCode enums and IsRetryable classification for programmatic retry decisions.

## Canonical Example

**File:** `core/errors.go:8-110`

```go
type ErrorCode string

const (
	ErrRateLimit ErrorCode = "rate_limit"
	ErrAuth ErrorCode = "auth_error"
	ErrTimeout ErrorCode = "timeout"
	ErrInvalidInput ErrorCode = "invalid_input"
	ErrToolFailed ErrorCode = "tool_failed"
	ErrProviderDown ErrorCode = "provider_unavailable"
	ErrGuardBlocked ErrorCode = "guard_blocked"
	ErrBudgetExhausted ErrorCode = "budget_exhausted"
	ErrNotFound ErrorCode = "not_found"
)

var retryableCodes = map[ErrorCode]bool{
	ErrRateLimit: true,
	ErrTimeout: true,
	ErrProviderDown: true,
}

func IsRetryable(err error) bool {
	var e *Error
	if errors.As(err, &e) {
		return retryableCodes[e.Code]
	}
	return false
}
```

## Variations

1. **NewError with operation + code** — `core/errors.go:66-73`
   - Creates structured error with Op, Code, Message, Err fields

2. **Error method with chain printing** — `core/errors.go:77-82`
   - Formats: "op [code]: message: cause"

## Anti-Patterns

- **Unclassified errors**: Generic error{} without ErrorCode; breaks retry logic
- **No operation context**: Losing caller information; hard to debug
- **Silent swallowing**: Catching error but not checking IsRetryable before deciding action
- **Non-deterministic retry decisions**: Checking error string instead of ErrorCode enum

## Invariants

- All provider/tool errors wrap in core.Error with ErrorCode
- IsRetryable checks only: ErrRateLimit, ErrTimeout, ErrProviderDown
- Error.Unwrap() always returns Err field for proper error chain traversal
- Error.Is() compares codes only; two errors match iff Code matches
