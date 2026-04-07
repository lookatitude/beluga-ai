// Package plugins provides built-in implementations of the [runtime.Plugin]
// interface for use with the Beluga AI agent runtime.
//
// The following plugins are available:
//
//   - [NewRetryAndReflect] — retries retryable errors up to a configurable limit.
//   - [NewAuditPlugin] — records turn start, end, and error events to an audit store.
//   - [NewCostTracking] — records LLM usage after every turn via a cost tracker.
//   - [NewRateLimit] — enforces a per-minute request cap using a token bucket.
//
// Plugins are intended to be composed via [runtime.NewPluginChain].
package plugins
