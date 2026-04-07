package runtime

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestNewWorkerPool verifies constructor behaviour including size normalisation.
func TestNewWorkerPool(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantCap int
	}{
		{name: "normal size", size: 4, wantCap: 4},
		{name: "size one", size: 1, wantCap: 1},
		{name: "zero normalises to one", size: 0, wantCap: 1},
		{name: "negative normalises to one", size: -5, wantCap: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewWorkerPool(tt.size)
			if cap(p.sem) != tt.wantCap {
				t.Errorf("sem cap = %d, want %d", cap(p.sem), tt.wantCap)
			}
		})
	}
}

// TestWorkerPoolBoundedConcurrency verifies that no more than N tasks run
// at the same time. Tasks are submitted from background goroutines so the
// test goroutine is never blocked by the semaphore.
func TestWorkerPoolBoundedConcurrency(t *testing.T) {
	const limit = 3
	const tasks = 12

	pool := NewWorkerPool(limit)

	var (
		active  atomic.Int64
		maxSeen atomic.Int64
	)

	// release is closed when tasks should finish.
	release := make(chan struct{})
	// started is signalled by each task as soon as it begins executing.
	started := make(chan struct{}, tasks)

	ctx := context.Background()

	// Submit all tasks from a background goroutine so the test goroutine is
	// never blocked waiting for a semaphore slot.
	var submitWg sync.WaitGroup
	submitWg.Add(tasks)
	for i := 0; i < tasks; i++ {
		go func() {
			defer submitWg.Done()
			err := pool.Submit(ctx, func(ctx context.Context) {
				cur := active.Add(1)
				started <- struct{}{}

				// Track the high-water mark.
				for {
					old := maxSeen.Load()
					if cur <= old {
						break
					}
					if maxSeen.CompareAndSwap(old, cur) {
						break
					}
				}

				// Block until the test releases all tasks.
				select {
				case <-release:
				case <-ctx.Done():
				}
				active.Add(-1)
			})
			if err != nil {
				t.Errorf("Submit returned unexpected error: %v", err)
			}
		}()
	}

	// Wait until the pool is at capacity (all limit slots are full).
	deadline := time.Now().Add(5 * time.Second)
	for active.Load() < int64(limit) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}

	// Verify the invariant: active count must not exceed the pool limit.
	if cur := active.Load(); cur > int64(limit) {
		t.Errorf("active goroutines = %d, want <= %d", cur, limit)
	}

	// Release all tasks so the remaining queued submits can proceed.
	close(release)

	// Wait for all submit goroutines and then all work to finish.
	submitWg.Wait()
	pool.Wait()

	if got := maxSeen.Load(); got > int64(limit) {
		t.Errorf("peak concurrent goroutines = %d, want <= %d", got, limit)
	}
}

// TestWorkerPoolWait verifies that Wait blocks until all submitted work is done.
// Pool size equals task count so Submit never blocks.
func TestWorkerPoolWait(t *testing.T) {
	const tasks = 8
	pool := NewWorkerPool(tasks) // slots >= tasks: Submit never blocks
	ctx := context.Background()

	var completed atomic.Int64

	for i := 0; i < tasks; i++ {
		if err := pool.Submit(ctx, func(ctx context.Context) {
			time.Sleep(10 * time.Millisecond)
			completed.Add(1)
		}); err != nil {
			t.Fatalf("Submit: %v", err)
		}
	}

	pool.Wait()

	if got := completed.Load(); got != tasks {
		t.Errorf("completed = %d, want %d", got, tasks)
	}
}

// TestWorkerPoolSubmitContextCancelled verifies that Submit returns ctx.Err()
// when the context is cancelled while waiting for a semaphore slot.
func TestWorkerPoolSubmitContextCancelled(t *testing.T) {
	// Pool with 1 slot; fill it so the next Submit must wait.
	pool := NewWorkerPool(1)

	release := make(chan struct{})
	bgCtx := context.Background()

	// Fill the single slot.
	if err := pool.Submit(bgCtx, func(ctx context.Context) {
		<-release
	}); err != nil {
		t.Fatalf("Submit (fill slot): %v", err)
	}

	// Create a context we can cancel.
	cancelCtx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- pool.Submit(cancelCtx, func(ctx context.Context) {
			t.Error("fn should never execute after context cancel")
		})
	}()

	// Give the goroutine time to reach the blocking select, then cancel.
	time.Sleep(5 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Errorf("Submit error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Submit did not return after context cancellation")
	}

	// Clean up the goroutine holding the slot.
	close(release)
	pool.Wait()
}

// TestWorkerPoolDrainWaitsForInflight verifies that Drain waits for all
// in-flight work to complete before returning.
func TestWorkerPoolDrainWaitsForInflight(t *testing.T) {
	const tasks = 4
	pool := NewWorkerPool(tasks)
	ctx := context.Background()

	var completed atomic.Int64
	release := make(chan struct{})

	for i := 0; i < tasks; i++ {
		if err := pool.Submit(ctx, func(ctx context.Context) {
			<-release
			completed.Add(1)
		}); err != nil {
			t.Fatalf("Submit: %v", err)
		}
	}

	// Close release so all in-flight tasks can finish.
	close(release)

	drainCtx, drainCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer drainCancel()

	if err := pool.Drain(drainCtx); err != nil {
		t.Errorf("Drain returned error: %v", err)
	}

	if got := completed.Load(); got != tasks {
		t.Errorf("completed after Drain = %d, want %d", got, tasks)
	}
}

// TestWorkerPoolDrainContextExpiry verifies that Drain returns ctx.Err() when
// the context expires before all work completes.
func TestWorkerPoolDrainContextExpiry(t *testing.T) {
	pool := NewWorkerPool(2)
	bgCtx := context.Background()

	hold := make(chan struct{}) // never closed in this test — tasks block forever

	for i := 0; i < 2; i++ {
		if err := pool.Submit(bgCtx, func(ctx context.Context) {
			<-hold
		}); err != nil {
			t.Fatalf("Submit: %v", err)
		}
	}

	drainCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := pool.Drain(drainCtx)
	if err != context.DeadlineExceeded {
		t.Errorf("Drain error = %v, want context.DeadlineExceeded", err)
	}

	// Release the goroutines so they don't leak past the test.
	close(hold)
}

// TestWorkerPoolDrainRejectsNewWork verifies that after Drain, Submit returns
// an error instead of scheduling new work.
func TestWorkerPoolDrainRejectsNewWork(t *testing.T) {
	pool := NewWorkerPool(4)

	drainCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := pool.Drain(drainCtx); err != nil {
		t.Fatalf("Drain: %v", err)
	}

	err := pool.Submit(context.Background(), func(ctx context.Context) {
		t.Error("fn should never execute after Drain")
	})
	if err == nil {
		t.Error("Submit after Drain should return an error, got nil")
	}
}

// TestWorkerPoolContextCancelledDuringWork verifies that fn receives a context
// and can observe cancellation of that context during execution.
func TestWorkerPoolContextCancelledDuringWork(t *testing.T) {
	pool := NewWorkerPool(2)

	workCtx, cancel := context.WithCancel(context.Background())

	var detected atomic.Bool
	running := make(chan struct{})

	if err := pool.Submit(workCtx, func(ctx context.Context) {
		close(running)
		// Block until context is cancelled.
		<-ctx.Done()
		detected.Store(true)
	}); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Wait until the task is actually running before cancelling.
	<-running
	cancel()

	pool.Wait()

	if !detected.Load() {
		t.Error("work function did not detect context cancellation")
	}
}

// BenchmarkWorkerPoolSubmit measures the overhead of Submit for no-op tasks.
func BenchmarkWorkerPoolSubmit(b *testing.B) {
	pool := NewWorkerPool(8)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = pool.Submit(ctx, func(ctx context.Context) {})
		}
	})

	pool.Wait()
}
