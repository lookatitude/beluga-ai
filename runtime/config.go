package runtime

import "time"

// StreamingMode determines how the Runner streams events to callers.
type StreamingMode int

const (
	// StreamingNone disables streaming; events are collected and yielded at the end.
	StreamingNone StreamingMode = iota
	// StreamingSSE indicates Server-Sent Events streaming mode.
	StreamingSSE
	// StreamingWebSocket indicates WebSocket streaming mode.
	StreamingWebSocket
)

// RunnerConfig holds configuration for a Runner.
type RunnerConfig struct {
	// MaxConcurrentSessions is the maximum number of concurrent Run() calls.
	// Defaults to 0 (no limit beyond the worker pool size).
	MaxConcurrentSessions int

	// SessionTTL is the default time-to-live for sessions created by the Runner.
	// A zero value means sessions do not expire.
	SessionTTL time.Duration

	// StreamingMode determines how events are streamed to callers.
	StreamingMode StreamingMode

	// WorkerPoolSize is the number of concurrent workers in the pool.
	// Defaults to defaultWorkerPoolSize if not set.
	WorkerPoolSize int

	// GracefulShutdownTimeout is the maximum time to wait for in-flight
	// sessions to complete during Shutdown. Defaults to defaultShutdownTimeout.
	GracefulShutdownTimeout time.Duration
}

const (
	// defaultWorkerPoolSize is the default number of concurrent workers.
	defaultWorkerPoolSize = 10

	// defaultShutdownTimeout is the default graceful shutdown timeout.
	defaultShutdownTimeout = 30 * time.Second
)

// defaults returns a RunnerConfig with all defaults applied.
func defaults() RunnerConfig {
	return RunnerConfig{
		WorkerPoolSize:          defaultWorkerPoolSize,
		GracefulShutdownTimeout: defaultShutdownTimeout,
		StreamingMode:           StreamingNone,
	}
}
