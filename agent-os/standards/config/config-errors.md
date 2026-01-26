# Config Errors

Config errors follow the same **Op/Err/Code** pattern as other packages (see `backend/op-err-code`): `Op`, `Err` (with `Unwrap`), `Code`, optional `Message`.

**Constructors:** `NewConfigError(op, code, message string, args ...any)`, `WrapError(underlying error, code, message string, args ...any)`.

**ErrCodeXxx** in `iface/errors.go`: e.g. `ErrCodeInvalidConfig`, `ErrCodeValidationFailed`, `ErrCodeKeyNotFound`, `ErrCodeAllProvidersFailed`, `ErrCodeParseFailed`, `ErrCodeInvalidParameters`, etc.

**Predicates:**

- `IsConfigError(err, code string) bool` — unwraps the error chain and returns true if any `ConfigError` has the given `Code`.
- `AsConfigError(err error, target **ConfigError) bool` — sets `target` to the first `ConfigError` in the chain and returns true.
