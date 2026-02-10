// Package syncutil provides concurrency utilities for the Beluga AI framework.
//
// This is an internal package and is not part of the public API. It is used by
// packages that need bounded parallel execution, such as the agent executor,
// parallel workflow, and RAG pipeline.
//
// # WorkerPool
//
// [WorkerPool] manages a fixed number of goroutines that process submitted work.
// It limits concurrency to a configurable maximum and provides a Wait method to
// block until all submitted work completes:
//
//	pool := syncutil.NewWorkerPool(4)
//	defer pool.Close()
//	for _, item := range items {
//	    item := item
//	    pool.Submit(func() { process(item) })
//	}
//	pool.Wait()
//
// Once closed via [WorkerPool.Close], subsequent calls to Submit return
// [ErrPoolClosed].
//
// # Semaphore
//
// [Semaphore] provides a counting semaphore backed by a buffered channel.
// It limits the number of concurrent operations to its capacity:
//
//	sem := syncutil.NewSemaphore(10)
//	sem.Acquire()      // blocks until a slot is available
//	defer sem.Release()
//
// The non-blocking [Semaphore.TryAcquire] variant returns false immediately
// if the semaphore is at capacity.
package syncutil
