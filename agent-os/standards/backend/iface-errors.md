# iface/errors.go

Add `iface/errors.go` only when a **public interface** in `iface/` has methods that return or document this error type or its codes. If the error is only used inside the package, use `errors.go` at the package root.

**Put in `iface/errors.go`:**

- The error struct and `Error()` / `Unwrap()`
- `ErrCodeXxx` constants
- `NewXxxError` and related constructors (incl. `NewXxxErrorWithMessage`, shortcuts)
- Predicates that are part of the contract for implementors: `IsXxxError`, `IsRetryable`, `GetXxxError`, `GetXxxErrorCode`, etc.

This lets both the package and external implementors depend on the same types and codes.
