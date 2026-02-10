// Package resilience provides fault-tolerance primitives for the Beluga AI
// framework: retry with exponential backoff, circuit breakers, hedged requests,
// and provider-aware rate limiting.
//
// # Retry
//
// The generic Retry function executes a function up to a configured number of
// attempts with exponential backoff and optional jitter. It respects context
// cancellation and uses core.IsRetryable to determine which errors are
// retryable.
//
//	result, err := resilience.Retry(ctx, resilience.DefaultRetryPolicy(), func(ctx context.Context) (string, error) {
//	    return callExternalAPI(ctx)
//	})
//
// RetryPolicy controls maximum attempts, initial backoff, maximum backoff,
// backoff multiplier, jitter, and optionally restricts retries to specific
// error codes.
//
// # Circuit Breaker
//
// CircuitBreaker implements the circuit-breaker stability pattern. It wraps
// function calls and short-circuits when a failure threshold is exceeded,
// giving the downstream dependency time to recover.
//
// State transitions:
//
//   - closed → open: after failureThreshold consecutive failures
//   - open → half-open: after resetTimeout elapses
//   - half-open → closed: on a successful probe call
//   - half-open → open: on a failed probe call
//
// Usage:
//
//	cb := resilience.NewCircuitBreaker(5, 30*time.Second)
//	result, err := cb.Execute(ctx, func(ctx context.Context) (any, error) {
//	    return callService(ctx)
//	})
//	if errors.Is(err, resilience.ErrCircuitOpen) {
//	    // circuit is open, handle gracefully
//	}
//
// # Hedged Requests
//
// Hedge executes a primary function immediately. If it does not return within
// a delay, a secondary function is started concurrently. The result from
// whichever completes successfully first is returned. Both goroutines are
// cancelled once a result is available.
//
//	result, err := resilience.Hedge(ctx, primaryFn, fallbackFn, 100*time.Millisecond)
//
// # Rate Limiting
//
// RateLimiter enforces provider-specific rate limits using token-bucket
// algorithms for RPM (requests per minute) and TPM (tokens per minute),
// plus a semaphore for concurrency control. Configure limits via
// ProviderLimits:
//
//	rl := resilience.NewRateLimiter(resilience.ProviderLimits{
//	    RPM:           60,
//	    TPM:           100000,
//	    MaxConcurrent: 10,
//	})
//	if err := rl.Allow(ctx); err != nil {
//	    // rate limited or context cancelled
//	}
//	defer rl.Release()
package resilience
