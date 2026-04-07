// Package runtime provides the agent lifecycle management layer including
// the Runner, Team composition, Plugin system, and session management.
//
// # Worker Pool
//
// [WorkerPool] provides bounded concurrency for agent execution. It ensures
// that no more than N agent tasks run concurrently, preventing resource
// exhaustion when many tasks arrive simultaneously:
//
//	pool := runtime.NewWorkerPool(8)
//
//	for _, task := range tasks {
//	    task := task // capture loop variable
//	    if err := pool.Submit(ctx, func(ctx context.Context) {
//	        task.Execute(ctx)
//	    }); err != nil {
//	        // context cancelled while waiting for a slot
//	        break
//	    }
//	}
//
//	pool.Wait() // block until all submitted work completes
//
// [WorkerPool.Drain] stops accepting new work and waits for all in-flight
// tasks to finish, respecting the provided context deadline:
//
//	if err := pool.Drain(shutdownCtx); err != nil {
//	    // shutdown context expired before all tasks drained
//	}
package runtime
