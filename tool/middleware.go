package tool

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// Middleware wraps a Tool and returns a new Tool with added behavior.
// Middleware functions compose via ApplyMiddleware.
type Middleware func(Tool) Tool

// ApplyMiddleware applies middleware functions to a tool in reverse order,
// so that the first middleware in the list is the outermost wrapper.
func ApplyMiddleware(t Tool, mws ...Middleware) Tool {
	for i := len(mws) - 1; i >= 0; i-- {
		t = mws[i](t)
	}
	return t
}

// WithTimeout returns a Middleware that enforces a maximum execution duration.
// If the tool's Execute exceeds d, the context is cancelled and a timeout
// error is returned.
func WithTimeout(d time.Duration) Middleware {
	return func(t Tool) Tool {
		return &timeoutTool{tool: t, timeout: d}
	}
}

type timeoutTool struct {
	tool    Tool
	timeout time.Duration
}

func (t *timeoutTool) Name() string                { return t.tool.Name() }
func (t *timeoutTool) Description() string         { return t.tool.Description() }
func (t *timeoutTool) InputSchema() map[string]any { return t.tool.InputSchema() }

func (t *timeoutTool) Execute(ctx context.Context, input map[string]any) (*Result, error) {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	result, err := t.tool.Execute(ctx, input)
	if err != nil && ctx.Err() == context.DeadlineExceeded {
		return nil, core.NewError(
			"tool.execute",
			core.ErrTimeout,
			fmt.Sprintf("tool %s timed out after %s", t.tool.Name(), t.timeout),
			err,
		)
	}
	return result, err
}

// Default backoff used by WithRetry.
const (
	defaultRetryInitialBackoff = 100 * time.Millisecond
	defaultRetryMaxBackoff     = 2 * time.Second
)

// WithRetry returns a Middleware that retries tool execution up to maxAttempts
// times on retryable errors (as determined by core.IsRetryable), waiting with
// jittered exponential backoff (100ms doubling up to 2s) between attempts.
// maxAttempts below 1 is treated as 1 (a single attempt, no retries).
func WithRetry(maxAttempts int) Middleware {
	return WithRetryBackoff(maxAttempts, defaultRetryInitialBackoff, defaultRetryMaxBackoff)
}

// WithRetryBackoff is like WithRetry with an explicit backoff: the first
// retry waits about initial, each later one doubles the wait up to max, and
// every wait is jittered to between half and the full value. A zero initial
// retries immediately.
func WithRetryBackoff(maxAttempts int, initial, maxBackoff time.Duration) Middleware {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	return func(t Tool) Tool {
		return &retryTool{tool: t, maxAttempts: maxAttempts, initial: initial, max: maxBackoff}
	}
}

type retryTool struct {
	tool        Tool
	maxAttempts int
	initial     time.Duration
	max         time.Duration
}

func (r *retryTool) Name() string                { return r.tool.Name() }
func (r *retryTool) Description() string         { return r.tool.Description() }
func (r *retryTool) InputSchema() map[string]any { return r.tool.InputSchema() }

func (r *retryTool) Execute(ctx context.Context, input map[string]any) (*Result, error) {
	var lastErr error
	backoff := r.initial
	for attempt := 0; attempt < r.maxAttempts; attempt++ {
		if attempt > 0 {
			if err := sleepWithContext(ctx, jitterBackoff(backoff)); err != nil {
				return nil, err
			}
			backoff = nextRetryBackoff(backoff, r.max)
		}
		result, err := r.tool.Execute(ctx, input)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !core.IsRetryable(err) {
			return nil, err
		}
		// Check context before retrying.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	return nil, core.Wrapf(core.ErrToolFailed, "tool %s failed after %d attempts: %w", r.tool.Name(), r.maxAttempts, lastErr)
}

// nextRetryBackoff doubles d, capped at maxBackoff when it is positive.
func nextRetryBackoff(d, maxBackoff time.Duration) time.Duration {
	d *= 2
	if maxBackoff > 0 && d > maxBackoff {
		d = maxBackoff
	}
	return d
}

// jitterBackoff returns a duration in [d/2, d].
func jitterBackoff(d time.Duration) time.Duration {
	if d <= 1 {
		return d
	}
	half := d / 2
	return half + time.Duration(rand.Int64N(int64(half)+1)) // #nosec G404 -- jitter, not security sensitive
}

// sleepWithContext waits for d or until ctx is done, returning ctx's error
// in the latter case.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
