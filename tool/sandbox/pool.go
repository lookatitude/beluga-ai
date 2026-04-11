package sandbox

import (
	"context"
	"sync"

	"github.com/lookatitude/beluga-ai/core"
)

// poolOptions holds configuration for SandboxPool.
type poolOptions struct {
	size   int
	warmup bool
}

// PoolOption configures a SandboxPool.
type PoolOption func(*poolOptions)

// WithPoolSize sets the number of sandbox instances in the pool. Default is 4.
func WithPoolSize(n int) PoolOption {
	return func(o *poolOptions) { o.size = n }
}

// WithWarmup enables pre-creation of all sandbox instances when the pool is
// created. When false (the default), sandboxes are created lazily on first checkout.
func WithWarmup(warmup bool) PoolOption {
	return func(o *poolOptions) { o.warmup = warmup }
}

// SandboxPool manages a bounded pool of reusable Sandbox instances. It supports
// checkout/checkin for fast reuse, avoiding the overhead of creating a new
// sandbox for each execution.
//
// The pool size bounds the maximum number of concurrently-live sandboxes:
// callers that request a sandbox when all slots are checked out will block
// until one is returned or the context is cancelled.
type SandboxPool struct {
	mu       sync.Mutex
	provider string
	opts     poolOptions
	pool     chan Sandbox
	sem      chan struct{} // capacity-bounded semaphore
	closed   bool
}

// NewSandboxPool creates a new pool that uses the named sandbox provider.
// The pool pre-allocates a buffered channel of the configured size. If warmup
// is enabled, all instances are created immediately.
func NewSandboxPool(provider string, opts ...PoolOption) (*SandboxPool, error) {
	o := poolOptions{size: 4}
	for _, opt := range opts {
		opt(&o)
	}
	if o.size <= 0 {
		return nil, core.NewError("sandbox.pool", core.ErrInvalidInput, "pool size must be positive", nil)
	}

	p := &SandboxPool{
		provider: provider,
		opts:     o,
		pool:     make(chan Sandbox, o.size),
		sem:      make(chan struct{}, o.size),
	}

	if o.warmup {
		for i := 0; i < o.size; i++ {
			sb, err := NewSandbox(provider)
			if err != nil {
				// Close any already-created sandboxes.
				p.closeAll(context.Background())
				return nil, core.Errorf(core.ErrProviderDown, "sandbox.pool: warmup failed at instance %d: %w", i, err)
			}
			p.pool <- sb
		}
	}

	return p, nil
}

// Checkout retrieves a sandbox from the pool. The total number of
// concurrently live sandboxes is bounded by the pool size: if all slots are
// checked out, Checkout blocks until a sandbox is returned via Checkin or
// the context is cancelled.
func (p *SandboxPool) Checkout(ctx context.Context) (Sandbox, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, core.NewError("sandbox.pool", core.ErrToolFailed, "pool is closed", nil)
	}
	p.mu.Unlock()

	// Acquire a slot from the semaphore first. This enforces the hard
	// concurrency cap — at most opts.size sandboxes are ever live.
	select {
	case p.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, core.NewError("sandbox.pool", core.ErrTimeout, "checkout timed out", ctx.Err())
	}

	// Try to reuse an idle sandbox.
	select {
	case sb := <-p.pool:
		return sb, nil
	default:
	}

	// No idle sandbox — create a new one. If creation fails, release the
	// slot so that another caller can make progress.
	sb, err := NewSandbox(p.provider)
	if err != nil {
		<-p.sem
		return nil, core.Errorf(core.ErrProviderDown, "sandbox.pool: create sandbox: %w", err)
	}
	return sb, nil
}

// Checkin returns a sandbox to the pool and releases the semaphore slot
// previously acquired by Checkout. If the pool has been closed
// concurrently, the sandbox is closed and dropped. The deposit into the
// channel is performed while holding the pool mutex so that Close/closeAll
// and Checkin are mutually exclusive, eliminating the window where a
// sandbox could be leaked after Close has already drained the pool.
func (p *SandboxPool) Checkin(sb Sandbox) {
	if sb == nil {
		return
	}

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		_ = sb.Close(context.Background())
		p.releaseSlot()
		return
	}
	select {
	case p.pool <- sb:
		p.mu.Unlock()
		p.releaseSlot()
	default:
		p.mu.Unlock()
		_ = sb.Close(context.Background())
		p.releaseSlot()
	}
}

// releaseSlot frees one semaphore slot, if one is held.
func (p *SandboxPool) releaseSlot() {
	select {
	case <-p.sem:
	default:
	}
}

// Close closes the pool and all sandboxes currently in it. After Close,
// Checkout returns an error and Checkin closes returned sandboxes.
func (p *SandboxPool) Close(ctx context.Context) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	p.closeAll(ctx)
	return nil
}

// closeAll drains and closes all sandboxes currently in the pool channel.
func (p *SandboxPool) closeAll(ctx context.Context) {
	for {
		select {
		case sb := <-p.pool:
			_ = sb.Close(ctx)
		default:
			return
		}
	}
}
