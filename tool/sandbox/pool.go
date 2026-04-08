package sandbox

import (
	"context"
	"fmt"
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
type SandboxPool struct {
	mu       sync.Mutex
	provider string
	opts     poolOptions
	pool     chan Sandbox
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
	}

	if o.warmup {
		for i := 0; i < o.size; i++ {
			sb, err := NewSandbox(provider)
			if err != nil {
				// Close any already-created sandboxes.
				p.closeAll(context.Background())
				return nil, fmt.Errorf("sandbox.pool: warmup failed at instance %d: %w", i, err)
			}
			p.pool <- sb
		}
	}

	return p, nil
}

// Checkout retrieves a sandbox from the pool. If the pool is empty and has not
// reached capacity, a new instance is created. If the pool is empty and at
// capacity, Checkout blocks until a sandbox is returned via Checkin or the
// context is cancelled.
func (p *SandboxPool) Checkout(ctx context.Context) (Sandbox, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, core.NewError("sandbox.pool", core.ErrToolFailed, "pool is closed", nil)
	}
	p.mu.Unlock()

	// Try non-blocking first.
	select {
	case sb := <-p.pool:
		return sb, nil
	default:
	}

	// Try to create a new one (best-effort — we don't track total created
	// since sandboxes may be discarded on error).
	sb, err := NewSandbox(p.provider)
	if err == nil {
		return sb, nil
	}

	// Fall back to blocking wait.
	select {
	case sb := <-p.pool:
		return sb, nil
	case <-ctx.Done():
		return nil, core.NewError("sandbox.pool", core.ErrTimeout, "checkout timed out", ctx.Err())
	}
}

// Checkin returns a sandbox to the pool. If the pool buffer is full, the
// sandbox is closed and discarded.
func (p *SandboxPool) Checkin(sb Sandbox) {
	if sb == nil {
		return
	}

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		_ = sb.Close(context.Background())
		return
	}
	p.mu.Unlock()

	select {
	case p.pool <- sb:
	default:
		// Pool full — discard.
		_ = sb.Close(context.Background())
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
