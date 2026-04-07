package runtime

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// WorkerPool provides bounded concurrency for agent execution.
// It limits the number of goroutines that may run concurrently,
// preventing resource exhaustion under load.
//
// A WorkerPool must be created via [NewWorkerPool]. The zero value is not valid.
type WorkerPool struct {
	sem     chan struct{}
	wg      sync.WaitGroup
	drained atomic.Bool
}

// NewWorkerPool creates a pool with the given concurrency limit.
// size must be >= 1; a size < 1 is normalised to 1.
func NewWorkerPool(size int) *WorkerPool {
	if size < 1 {
		size = 1
	}
	return &WorkerPool{
		sem: make(chan struct{}, size),
	}
}

// Submit schedules fn for execution in the pool. It blocks until a worker
// slot is available or ctx is cancelled. If the pool has been drained,
// Submit returns an error immediately without executing fn.
//
// fn receives the same ctx that was passed to Submit, so work items can
// detect cancellation during execution.
//
// Returns ctx.Err() if the context is cancelled while waiting for a slot.
func (p *WorkerPool) Submit(ctx context.Context, fn func(context.Context)) error {
	if p.drained.Load() {
		return fmt.Errorf("runtime.WorkerPool: pool is drained, no new work accepted")
	}

	// Acquire a semaphore slot, honouring context cancellation.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.sem <- struct{}{}:
	}

	// Double-check drain state after acquiring the slot to avoid a race where
	// Drain is called between the drained check above and slot acquisition.
	if p.drained.Load() {
		<-p.sem // release the slot we just took
		return fmt.Errorf("runtime.WorkerPool: pool is drained, no new work accepted")
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		defer func() { <-p.sem }() // release slot when done
		fn(ctx)
	}()

	return nil
}

// Wait blocks until all submitted work completes. It does not prevent new
// work from being submitted after it returns.
func (p *WorkerPool) Wait() {
	p.wg.Wait()
}

// Drain stops accepting new work and waits for all in-flight work to finish.
// It returns nil when all work completes, or ctx.Err() if the context expires
// before the pool drains.
//
// After Drain returns, any subsequent calls to Submit will return an error.
func (p *WorkerPool) Drain(ctx context.Context) error {
	p.drained.Store(true)

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
