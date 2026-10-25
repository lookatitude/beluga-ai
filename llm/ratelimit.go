package llm

import (
	"context"
	"iter"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// ProviderLimits describes the rate limits for a provider. Zero values mean
// no limit for that dimension.
type ProviderLimits struct {
	// RPM is the maximum requests per minute.
	RPM int
	// TPM is the maximum tokens per minute. Usage is recorded from each
	// response's Usage.TotalTokens (for streams, from the chunks that carry
	// Usage). A new request is admitted while the tokens used in the trailing
	// minute are below TPM; it is not admitted once the budget is spent.
	TPM int
	// MaxConcurrent is the maximum number of concurrent requests.
	MaxConcurrent int
	// CooldownOnRetry is the duration to wait before retrying after hitting a limit.
	CooldownOnRetry time.Duration
}

// WithProviderLimits returns middleware that enforces the given rate limits:
// requests per minute, tokens per minute and maximum concurrent requests.
//
// For Stream, limits are acquired lazily when the returned iterator is first
// ranged (not when Stream is called) and the concurrency slot is released on
// every exit path: normal completion, an error, or the consumer breaking
// early. A Seq that is never ranged therefore holds no slot, and ranging the
// same Seq twice acquires and releases twice.
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
	if limits.TPM > 0 {
		rl.tpm = &tokenWindow{maxTokens: limits.TPM, window: time.Minute}
	}
	return func(next ChatModel) ChatModel {
		return &rateLimitedModel{next: next, limiter: rl}
	}
}

type rateLimiter struct {
	limits ProviderLimits
	sem    chan struct{}
	rpm    *slidingWindow
	tpm    *tokenWindow
}

// acquireTPM checks the token budget, optionally waiting for a cooldown
// period before re-checking. Returns an error if the budget is spent.
func (rl *rateLimiter) acquireTPM(ctx context.Context) error {
	if rl.tpm == nil || rl.tpm.allow() {
		return nil
	}
	if rl.limits.CooldownOnRetry <= 0 {
		return core.NewError("llm.ratelimit", core.ErrRateLimit, "TPM limit exceeded", nil)
	}
	select {
	case <-time.After(rl.limits.CooldownOnRetry):
		if !rl.tpm.allow() {
			return core.NewError("llm.ratelimit", core.ErrRateLimit, "TPM limit exceeded", nil)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// recordUsage adds the tokens consumed by a completed request.
func (rl *rateLimiter) recordUsage(u *schema.Usage) {
	if rl.tpm == nil || u == nil || u.TotalTokens <= 0 {
		return
	}
	rl.tpm.add(u.TotalTokens)
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
	if err := rl.acquireTPM(ctx); err != nil {
		return err
	}
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

// tokenWindow tracks token consumption in a rolling time window.
type tokenWindow struct {
	mu        sync.Mutex
	maxTokens int
	window    time.Duration
	entries   []tokenEntry
}

type tokenEntry struct {
	at     time.Time
	tokens int
}

// usedLocked prunes expired entries and returns the tokens used in the window.
// The caller must hold w.mu.
func (w *tokenWindow) usedLocked(now time.Time) int {
	cutoff := now.Add(-w.window)
	valid, total := 0, 0
	for _, e := range w.entries {
		if e.at.After(cutoff) {
			w.entries[valid] = e
			valid++
			total += e.tokens
		}
	}
	w.entries = w.entries[:valid]
	return total
}

// allow reports whether the token budget for the current window is not yet
// spent.
func (w *tokenWindow) allow() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.usedLocked(time.Now()) < w.maxTokens
}

// add records tokens consumed now.
func (w *tokenWindow) add(tokens int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now()
	w.usedLocked(now)
	w.entries = append(w.entries, tokenEntry{at: now, tokens: tokens})
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
	resp, err := m.next.Generate(ctx, msgs, opts...)
	if resp != nil {
		m.limiter.recordUsage(&resp.Usage)
	}
	return resp, err
}

// Stream acquires the limits lazily, on first iteration, and releases the
// concurrency slot on every exit path (see WithProviderLimits).
func (m *rateLimitedModel) Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	return func(yield func(schema.StreamChunk, error) bool) {
		if err := m.limiter.acquire(ctx); err != nil {
			yield(schema.StreamChunk{}, err)
			return
		}
		defer m.limiter.release()
		for chunk, err := range m.next.Stream(ctx, msgs, opts...) {
			m.limiter.recordUsage(chunk.Usage)
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
