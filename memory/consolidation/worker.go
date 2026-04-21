package consolidation

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
)

const (
	// defaultInterval is the default consolidation cycle interval.
	defaultInterval = 1 * time.Hour

	// defaultMaxRecords is the default maximum records processed per cycle.
	defaultMaxRecords = 1000
)

// workerOptions holds configurable parameters for the Worker.
type workerOptions struct {
	interval           time.Duration
	maxRecordsPerCycle int
	policy             Evaluator
	compressor         Compressor
	hooks              Hooks
}

// Option configures a Worker.
type Option func(*workerOptions)

// WithInterval sets the consolidation cycle interval. The minimum is 1 second;
// smaller values are clamped.
func WithInterval(d time.Duration) Option {
	return func(o *workerOptions) {
		if d < time.Second {
			d = time.Second
		}
		o.interval = d
	}
}

// WithPolicy sets the consolidation policy used to evaluate records.
func WithPolicy(p Evaluator) Option {
	return func(o *workerOptions) { o.policy = p }
}

// WithCompressor sets the compressor used for records marked ActionCompress.
// If nil, records marked for compression are pruned instead.
func WithCompressor(c Compressor) Option {
	return func(o *workerOptions) { o.compressor = c }
}

// WithMaxRecordsPerCycle limits the number of records evaluated per cycle.
func WithMaxRecordsPerCycle(n int) Option {
	return func(o *workerOptions) {
		if n > 0 {
			o.maxRecordsPerCycle = n
		}
	}
}

// WithHooks attaches lifecycle hooks to the worker.
func WithHooks(h Hooks) Option {
	return func(o *workerOptions) { o.hooks = h }
}

// Worker runs periodic consolidation cycles against a ConsolidationStore.
// It implements core.Lifecycle for integration with the application lifecycle
// manager.
type Worker struct {
	store ConsolidationStore
	opts  workerOptions

	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	running bool
	health  core.HealthStatus
}

// Compile-time interface check.
var _ core.Lifecycle = (*Worker)(nil)

// NewWorker creates a consolidation worker that operates on the given store.
// A default ThresholdPolicy is used if none is provided via WithPolicy.
func NewWorker(store ConsolidationStore, opts ...Option) *Worker {
	o := workerOptions{
		interval:           defaultInterval,
		maxRecordsPerCycle: defaultMaxRecords,
		policy:             NewThresholdPolicy(),
	}
	for _, opt := range opts {
		opt(&o)
	}

	return &Worker{
		store: store,
		opts:  o,
		health: core.HealthStatus{
			Status:    core.HealthUnhealthy,
			Message:   "not started",
			Timestamp: time.Now(),
		},
	}
}

// Start begins the consolidation loop. It blocks until the worker is ready
// and returns immediately; the loop runs in a background goroutine.
func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return core.Errorf(core.ErrInvalidInput, "consolidation: worker already running")
	}

	// The worker daemon runs until Stop() is explicitly called. We derive
	// the loop context from ctx so that parent context cancellation is
	// respected during shutdown.
	loopCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	w.done = make(chan struct{})
	w.running = true
	w.health = core.HealthStatus{
		Status:    core.HealthHealthy,
		Message:   "running",
		Timestamp: time.Now(),
	}

	go w.loop(loopCtx)
	return nil
}

// Stop gracefully shuts down the worker and waits for the current cycle
// to finish.
func (w *Worker) Stop(ctx context.Context) error {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return nil
	}
	w.cancel()
	done := w.done
	w.mu.Unlock()

	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}

	w.mu.Lock()
	w.running = false
	w.health = core.HealthStatus{
		Status:    core.HealthUnhealthy,
		Message:   "stopped",
		Timestamp: time.Now(),
	}
	w.mu.Unlock()
	return nil
}

// Health returns the current health status of the worker.
func (w *Worker) Health() core.HealthStatus {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.health
}

// loop runs the ticker+jitter consolidation cycle until ctx is cancelled.
func (w *Worker) loop(ctx context.Context) {
	defer close(w.done)

	ticker := time.NewTicker(w.opts.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Add jitter: up to 10% of interval.
			jitter := w.jitter(w.opts.interval / 10)
			jitterTimer := time.NewTimer(jitter)
			select {
			case <-ctx.Done():
				jitterTimer.Stop()
				return
			case <-jitterTimer.C:
			}

			metrics := w.runCycle(ctx)

			w.mu.Lock()
			if len(metrics.Errors) > 0 {
				w.health = core.HealthStatus{
					Status:    core.HealthDegraded,
					Message:   fmt.Sprintf("cycle completed with %d errors", len(metrics.Errors)),
					Timestamp: time.Now(),
				}
			} else {
				w.health = core.HealthStatus{
					Status:    core.HealthHealthy,
					Message:   fmt.Sprintf("cycle ok: evaluated=%d pruned=%d compressed=%d", metrics.RecordsEvaluated, metrics.RecordsPruned, metrics.RecordsCompressed),
					Timestamp: time.Now(),
				}
			}
			w.mu.Unlock()

			if w.opts.hooks.OnCycleComplete != nil {
				w.opts.hooks.OnCycleComplete(metrics)
			}
		}
	}
}

// runCycle executes a single consolidation cycle.
func (w *Worker) runCycle(ctx context.Context) CycleMetrics {
	metrics := CycleMetrics{CycleStart: time.Now()}

	records, err := w.store.ListRecords(ctx, 0, w.opts.maxRecordsPerCycle)
	if err != nil {
		metrics.Errors = append(metrics.Errors, core.Errorf(core.ErrProviderDown, "list records: %w", err))
		metrics.CycleEnd = time.Now()
		return metrics
	}

	if len(records) == 0 {
		metrics.CycleEnd = time.Now()
		return metrics
	}

	metrics.RecordsEvaluated = len(records)

	decisions, err := w.opts.policy.Evaluate(ctx, records)
	if err != nil {
		metrics.Errors = append(metrics.Errors, core.Errorf(core.ErrInvalidInput, "evaluate: %w", err))
		metrics.CycleEnd = time.Now()
		return metrics
	}

	toPrune, toCompress := w.partitionDecisions(decisions)
	toPrune = w.applyCompression(ctx, &metrics, toCompress, toPrune)
	w.applyPrune(ctx, &metrics, toPrune)

	metrics.CycleEnd = time.Now()
	return metrics
}

// partitionDecisions splits decisions into records to prune and records to compress.
func (w *Worker) partitionDecisions(decisions []Decision) (toPrune, toCompress []Record) {
	for _, d := range decisions {
		switch d.Action {
		case ActionPrune:
			toPrune = append(toPrune, d.Record)
		case ActionCompress:
			toCompress = append(toCompress, d.Record)
		}
	}
	return toPrune, toCompress
}

// applyCompression compresses records when a compressor is available; otherwise they
// are added to toPrune. Returns the updated toPrune slice.
func (w *Worker) applyCompression(ctx context.Context, metrics *CycleMetrics, toCompress, toPrune []Record) []Record {
	if len(toCompress) == 0 {
		return toPrune
	}
	if w.opts.compressor == nil {
		return append(toPrune, toCompress...)
	}

	compressed, err := w.opts.compressor.Compress(ctx, toCompress)
	if err != nil {
		metrics.Errors = append(metrics.Errors, core.Errorf(core.ErrProviderDown, "compress: %w", err))
		return append(toPrune, toCompress...)
	}

	if err := w.store.UpdateRecords(ctx, compressed); err != nil {
		metrics.Errors = append(metrics.Errors, core.Errorf(core.ErrProviderDown, "update compressed: %w", err))
	} else {
		metrics.RecordsCompressed = len(compressed)
		if w.opts.hooks.OnCompressed != nil {
			w.opts.hooks.OnCompressed(toCompress, compressed)
		}
	}
	return toPrune
}

// applyPrune deletes the given records from the store and updates metrics.
func (w *Worker) applyPrune(ctx context.Context, metrics *CycleMetrics, toPrune []Record) {
	if len(toPrune) == 0 {
		return
	}
	ids := make([]string, len(toPrune))
	for i, r := range toPrune {
		ids[i] = r.ID
	}
	if err := w.store.DeleteRecords(ctx, ids); err != nil {
		metrics.Errors = append(metrics.Errors, core.Errorf(core.ErrProviderDown, "delete: %w", err))
	} else {
		metrics.RecordsPruned = len(toPrune)
		if w.opts.hooks.OnPruned != nil {
			w.opts.hooks.OnPruned(toPrune)
		}
	}
}

// jitter returns a random duration in [0, max) using crypto/rand.
func (w *Worker) jitter(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return max / 2 // fallback to half on error
	}
	return time.Duration(n.Int64())
}
