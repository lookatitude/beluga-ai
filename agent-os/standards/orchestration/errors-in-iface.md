# Errors in iface

**OrchestratorError:** `Op`, `Err`, `Code`. `NewOrchestratorError(op, err, code)`. Helpers: `ErrInvalidConfig`, `ErrExecutionFailed`, `ErrTimeout`, `ErrInvalidState`, `ErrNotFound`, `ErrDependencyFailed`, `ErrResourceExhausted`, `ErrCircuitBreakerOpen`, `ErrRateLimitExceeded`, `ErrInvalidInput`, `ErrWorkflowDeadlock`, `ErrTaskCancelled`, `ErrMaxRetriesExceeded`. Define in `iface/errors.go` so all orchestration implementations share them and iface stays self-contained.

**IsRetryable(err):** Return true for `ErrCodeTimeout`, `ErrCodeDependencyFailed`, `ErrCodeResourceExhausted`, `ErrCodeCircuitBreakerOpen`, `ErrCodeRateLimitExceeded`; false for `ErrCodeInvalidConfig`, `ErrCodeInvalidState`, `ErrCodeNotFound`, `ErrCodeInvalidInput`, `ErrCodeWorkflowDeadlock`. Providers may add their own retry logic on top.
