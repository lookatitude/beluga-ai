package tool

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/core"
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

func (t *timeoutTool) Name() string              { return t.tool.Name() }
func (t *timeoutTool) Description() string        { return t.tool.Description() }
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

// WithRetry returns a Middleware that retries tool execution up to maxAttempts
// times on retryable errors (as determined by core.IsRetryable).
func WithRetry(maxAttempts int) Middleware {
	return func(t Tool) Tool {
		return &retryTool{tool: t, maxAttempts: maxAttempts}
	}
}

type retryTool struct {
	tool        Tool
	maxAttempts int
}

func (r *retryTool) Name() string              { return r.tool.Name() }
func (r *retryTool) Description() string        { return r.tool.Description() }
func (r *retryTool) InputSchema() map[string]any { return r.tool.InputSchema() }

func (r *retryTool) Execute(ctx context.Context, input map[string]any) (*Result, error) {
	var lastErr error
	for attempt := 0; attempt < r.maxAttempts; attempt++ {
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
	return nil, fmt.Errorf("tool %s failed after %d attempts: %w", r.tool.Name(), r.maxAttempts, lastErr)
}
