// Package syncutil provides concurrency utilities for the Beluga AI framework.
// It includes a worker pool for bounded parallel execution and a semaphore
// for concurrency limiting.
package syncutil

import (
	"errors"
	"sync"
)

// ErrPoolClosed is returned when work is submitted to a closed pool.
var ErrPoolClosed = errors.New("syncutil: worker pool is closed")

// WorkerPool manages a fixed number of goroutines that process submitted work.
// It limits concurrency to maxWorkers simultaneous tasks and provides a Wait
// method to block until all submitted work completes.
type WorkerPool struct {
	sem    Semaphore
	wg     sync.WaitGroup
	mu     sync.Mutex
	closed bool
}

// NewWorkerPool creates a WorkerPool that runs at most maxWorkers concurrent
// tasks. If maxWorkers is less than 1, it defaults to 1.
func NewWorkerPool(maxWorkers int) *WorkerPool {
	if maxWorkers < 1 {
		maxWorkers = 1
	}
	return &WorkerPool{
		sem: NewSemaphore(maxWorkers),
	}
}

// Submit enqueues fn for execution by the pool. It returns ErrPoolClosed if
// the pool has been closed. The function fn is executed in a new goroutine
// once a worker slot becomes available.
func (p *WorkerPool) Submit(fn func()) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrPoolClosed
	}
	p.wg.Add(1)
	p.mu.Unlock()

	go func() {
		defer p.wg.Done()
		p.sem.Acquire()
		defer p.sem.Release()
		fn()
	}()

	return nil
}

// Wait blocks until all submitted work has completed.
func (p *WorkerPool) Wait() {
	p.wg.Wait()
}

// Close marks the pool as closed so that no new work can be submitted, then
// waits for all in-flight work to finish. After Close returns, Submit will
// always return ErrPoolClosed.
func (p *WorkerPool) Close() {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()
	p.wg.Wait()
}

// Semaphore provides a counting semaphore backed by a buffered channel.
// It limits the number of concurrent operations to its capacity.
type Semaphore chan struct{}

// NewSemaphore creates a Semaphore with the given capacity. Capacity must be
// at least 1; values less than 1 are clamped to 1.
func NewSemaphore(capacity int) Semaphore {
	if capacity < 1 {
		capacity = 1
	}
	return make(Semaphore, capacity)
}

// Acquire blocks until a slot is available, then claims it.
func (s Semaphore) Acquire() {
	s <- struct{}{}
}

// TryAcquire attempts to acquire a slot without blocking. It returns true
// if successful, false if the semaphore is at capacity.
func (s Semaphore) TryAcquire() bool {
	select {
	case s <- struct{}{}:
		return true
	default:
		return false
	}
}

// Release frees a previously acquired slot.
func (s Semaphore) Release() {
	<-s
}
