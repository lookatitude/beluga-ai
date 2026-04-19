package credential

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// RevokerHooks provides optional callbacks for auto-revocation events.
// All fields are optional; nil hooks are skipped.
type RevokerHooks struct {
	// OnRevoke is called after a credential is automatically revoked.
	OnRevoke func(ctx context.Context, cred *AgentCredential)

	// OnScanComplete is called after each scan cycle with the number of
	// credentials revoked.
	OnScanComplete func(ctx context.Context, revoked int)

	// OnError is called when a revocation fails.
	OnError func(ctx context.Context, credID string, err error)
}

// revokerOptions holds configuration for AutoRevoker.
type revokerOptions struct {
	scanInterval time.Duration
	hooks        RevokerHooks
	logger       *slog.Logger
}

// RevokerOption configures an AutoRevoker.
type RevokerOption func(*revokerOptions)

// WithScanInterval sets how often the revoker scans for expired credentials.
// Defaults to 30 seconds.
func WithScanInterval(d time.Duration) RevokerOption {
	return func(o *revokerOptions) { o.scanInterval = d }
}

// WithRevokerHooks sets the hooks for the auto-revoker.
func WithRevokerHooks(h RevokerHooks) RevokerOption {
	return func(o *revokerOptions) { o.hooks = h }
}

// WithRevokerLogger sets the logger for the auto-revoker.
func WithRevokerLogger(l *slog.Logger) RevokerOption {
	return func(o *revokerOptions) { o.logger = l }
}

func defaultRevokerOptions() revokerOptions {
	return revokerOptions{
		scanInterval: 30 * time.Second,
		logger:       slog.Default(),
	}
}

// AutoRevoker implements core.Lifecycle and runs a background goroutine that
// periodically scans for expired credentials and revokes them.
type AutoRevoker struct {
	issuer *InMemoryIssuer
	opts   revokerOptions

	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	running bool
}

// Compile-time check that AutoRevoker implements core.Lifecycle.
var _ core.Lifecycle = (*AutoRevoker)(nil)

// NewAutoRevoker creates a new AutoRevoker that scans the given issuer.
func NewAutoRevoker(issuer *InMemoryIssuer, opts ...RevokerOption) *AutoRevoker {
	o := defaultRevokerOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return &AutoRevoker{
		issuer: issuer,
		opts:   o,
	}
}

// Start begins the background scan loop. It returns once the loop has been
// launched. The caller's context is propagated to the background loop so
// tracing, tenant IDs, and deadlines survive; a locally-derived cancellation
// is still used so Stop() can terminate the loop independently.
func (r *AutoRevoker) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return core.NewError("credential.revoker.start", core.ErrInvalidInput, "already running", nil)
	}
	if err := ctx.Err(); err != nil {
		return core.NewError("credential.revoker.start", core.ErrInvalidInput, "caller context already cancelled", err)
	}

	scanCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.done = make(chan struct{})
	r.running = true

	go r.loop(scanCtx)

	return nil
}

// Stop gracefully shuts down the background scan loop. The mutex is released
// before blocking on the loop's done channel to avoid deadlocking with hooks
// that call back into the revoker (e.g. Health()).
func (r *AutoRevoker) Stop(_ context.Context) error {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return nil
	}
	cancel := r.cancel
	done := r.done
	r.running = false
	r.mu.Unlock()

	cancel()
	<-done
	return nil
}

// Health returns the current health status of the revoker.
func (r *AutoRevoker) Health() core.HealthStatus {
	r.mu.Lock()
	running := r.running
	r.mu.Unlock()

	if running {
		return core.HealthStatus{
			Status:    core.HealthHealthy,
			Message:   "auto-revoker is running",
			Timestamp: time.Now(),
		}
	}
	return core.HealthStatus{
		Status:    core.HealthUnhealthy,
		Message:   "auto-revoker is not running",
		Timestamp: time.Now(),
	}
}

// loop runs the periodic scan until ctx is cancelled.
func (r *AutoRevoker) loop(ctx context.Context) {
	defer close(r.done)

	ticker := time.NewTicker(r.opts.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.scan(ctx)
		}
	}
}

// scan finds expired credentials and revokes them.
func (r *AutoRevoker) scan(ctx context.Context) {
	expired := r.issuer.Expired()
	revoked := 0

	for _, cred := range expired {
		if err := r.issuer.Revoke(ctx, cred.ID); err != nil {
			if r.opts.hooks.OnError != nil {
				r.opts.hooks.OnError(ctx, cred.ID, err)
			}
			r.opts.logger.ErrorContext(ctx, "credential.revoker.scan.error",
				"credential_id", cred.ID,
				"error", err,
			)
			continue
		}
		revoked++
		if r.opts.hooks.OnRevoke != nil {
			r.opts.hooks.OnRevoke(ctx, cred)
		}
	}

	// Purge fully-handled (revoked + expired) credentials so the issuer
	// does not exhaust its bounded capacity.
	r.issuer.Cleanup(ctx)

	if r.opts.hooks.OnScanComplete != nil {
		r.opts.hooks.OnScanComplete(ctx, revoked)
	}
}
