# Chain: Memory, Retry, Streaming

**Memory:** One `LoadMemoryVariables` before the steps and one `SaveContext` after, per chain run. Merge loaded variables with the direct input; when `Memory` is set, expect `map[string]any` or `string` (single input key) to match the memory contract. After the last step, `SaveContext` with the combined input and the final output (as `map[string]any` or wrapped via `GetOutputKeys`).

**Retry:** Use `scheduler.RetryExecutor` (or equivalent) with `RetryConfig` (MaxAttempts, InitialDelay, MaxDelay, BackoffFactor, JitterFactor). Call `iface.IsRetryable(err)` before retrying; if not retryable, fail immediately.

**Streaming:** For each step that supports `Stream`, use `Stream`; otherwise use `Invoke`. Run non-streaming steps first, then stream from the first step that supports it, or stream only the last step when it's the only one that supports streaming. Do not require every step to support streaming.
