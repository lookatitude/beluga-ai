package llm

import (
	"context"
	"iter"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/schema"
)

// ProviderLimits describes the rate limits for a provider. Zero values mean
// no limit for that dimension.
type ProviderLimits struct {
	// RPM is the maximum requests per minute.
	RPM int
	// TPM is the maximum tokens per minute.
	TPM int
	// MaxConcurrent is the maximum number of concurrent requests.
	MaxConcurrent int
	// CooldownOnRetry is the duration to wait before retrying after hitting a limit.
	CooldownOnRetry time.Duration
}

// WithProviderLimits returns middleware that enforces the given rate limits.
func WithProviderLimits(limits ProviderLimits) Middleware {
	rl := &rateLimiter{
		limits: limits,
	}
	if limits.MaxConcurrent > 0 {
		rl.sem = make(chan struct{}, limits.MaxConcurrent)
	}
	if limits.RPM > 0 {
		rl.rpm = &slidingWindow{
			maxCount: limits.RPM,
			window:   time.Minute,
		}
	}
	return func(next ChatModel) ChatModel {
		return &rateLimitedModel{next: next, limiter: rl}
	}
}

type rateLimiter struct {
	limits ProviderLimits
	sem    chan struct{}
	rpm    *slidingWindow
}

// acquireRPM checks the RPM sliding window, optionally waiting for a cooldown
// period before retrying. Returns an error if the limit is exceeded.
func (rl *rateLimiter) acquireRPM(ctx context.Context) error {
	if rl.rpm == nil || rl.rpm.allow() {
		return nil
	}
	if rl.limits.CooldownOnRetry <= 0 {
		return core.NewError("llm.ratelimit", core.ErrRateLimit, "RPM limit exceeded", nil)
	}
	select {
	case <-time.After(rl.limits.CooldownOnRetry):
		if !rl.rpm.allow() {
			return core.NewError("llm.ratelimit", core.ErrRateLimit, "RPM limit exceeded", nil)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// acquireConcurrency blocks until a concurrency slot is available or ctx is done.
func (rl *rateLimiter) acquireConcurrency(ctx context.Context) error {
	if rl.sem == nil {
		return nil
	}
	select {
	case rl.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (rl *rateLimiter) acquire(ctx context.Context) error {
	if err := rl.acquireRPM(ctx); err != nil {
		return err
	}
	return rl.acquireConcurrency(ctx)
}

func (rl *rateLimiter) release() {
	if rl.sem != nil {
		<-rl.sem
	}
}

// slidingWindow tracks request counts in a rolling time window.
type slidingWindow struct {
	mu       sync.Mutex
	maxCount int
	window   time.Duration
	times    []time.Time
}

func (w *slidingWindow) allow() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-w.window)

	// Remove expired entries.
	valid := 0
	for _, t := range w.times {
		if t.After(cutoff) {
			w.times[valid] = t
			valid++
		}
	}
	w.times = w.times[:valid]

	if len(w.times) >= w.maxCount {
		return false
	}
	w.times = append(w.times, now)
	return true
}

type rateLimitedModel struct {
	next    ChatModel
	limiter *rateLimiter
}

func (m *rateLimitedModel) Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
	if err := m.limiter.acquire(ctx); err != nil {
		return nil, err
	}
	defer m.limiter.release()
	return m.next.Generate(ctx, msgs, opts...)
}

func (m *rateLimitedModel) Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	if err := m.limiter.acquire(ctx); err != nil {
		return func(yield func(schema.StreamChunk, error) bool) {
			yield(schema.StreamChunk{}, err)
		}
	}
	inner := m.next.Stream(ctx, msgs, opts...)
	return func(yield func(schema.StreamChunk, error) bool) {
		defer m.limiter.release()
		for chunk, err := range inner {
			if !yield(chunk, err) {
				return
			}
			if err != nil {
				return
			}
		}
	}
}

func (m *rateLimitedModel) BindTools(tools []schema.ToolDefinition) ChatModel {
	return &rateLimitedModel{next: m.next.BindTools(tools), limiter: m.limiter}
}

func (m *rateLimitedModel) ModelID() string { return m.next.ModelID() }
