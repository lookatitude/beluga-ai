// Package runtime provides the agent lifecycle management layer including
// the Runner, Plugin system, session management, and bounded worker pools.
//
// # Runner
//
// [Runner] is the central lifecycle manager that hosts a single agent. It
// handles session management, plugin execution, and streaming:
//
//	r := runtime.NewRunner(myAgent,
//	    runtime.WithPlugins(loggingPlugin, metricsPlugin),
//	    runtime.WithWorkerPoolSize(8),
//	)
//
//	for event, err := range r.Run(ctx, "session-123", inputMsg) {
//	    if err != nil { break }
//	    // process event
//	}
//
// # Worker Pool
//
// [WorkerPool] provides bounded concurrency for agent execution. It ensures
// that no more than N agent tasks run concurrently, preventing resource
// exhaustion when many tasks arrive simultaneously.
//
// # Shutdown
//
// [Runner.Shutdown] drains the worker pool and waits for in-flight sessions
// to complete, respecting the provided context deadline:
//
//	if err := r.Shutdown(shutdownCtx); err != nil {
//	    // shutdown context expired before all tasks drained
//	}
package runtime
