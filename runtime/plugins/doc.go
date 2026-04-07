// Package plugins provides built-in runtime.Plugin implementations for the
// Beluga AI agent runtime. The following plugins are available:
//
//   - RetryAndReflect: auto-retries turns on retryable errors up to a
//     configured maximum, using core.IsRetryable to gate retry decisions.
//
//   - AuditPlugin: logs turn lifecycle events (start, end, error) to an
//     audit.Store via audit.Logger.
//
//   - CostTracking: extracts token usage from agent events after each turn
//     and records it via a cost.Tracker, enforcing optional budget limits.
//
//   - RateLimit: enforces a sliding-window request rate limit per session
//     using a token-bucket algorithm, rejecting turns that exceed the limit.
//
// All plugins implement the runtime.Plugin interface and include
// compile-time interface checks.
package plugins
