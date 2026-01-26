# Concurrency and Error Coverage

**Concurrency:** Add concurrency tests (goroutines, `sync.WaitGroup`, `t.Parallel()` where useful) only when the code under test is concurrent: goroutines, channels, or `sync` primitives.

**Errors:** Aim for at least one test per distinct error branch, or per `ErrCode` when that is the main abstraction. Cover timeouts and context cancellation where the implementation handles them.
