package syncutil

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestNewWorkerPool(t *testing.T) {
	tests := []struct {
		name       string
		maxWorkers int
	}{
		{"positive workers", 4},
		{"single worker", 1},
		{"zero defaults to 1", 0},
		{"negative defaults to 1", -5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewWorkerPool(tt.maxWorkers)
			if pool == nil {
				t.Fatal("expected non-nil pool")
			}
			pool.Close()
		})
	}
}

func TestWorkerPoolSubmitAndWait(t *testing.T) {
	pool := NewWorkerPool(4)
	var count atomic.Int64

	for i := 0; i < 100; i++ {
		err := pool.Submit(func() {
			count.Add(1)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	pool.Wait()
	if got := count.Load(); got != 100 {
		t.Errorf("expected 100 tasks completed, got %d", got)
	}
	pool.Close()
}

func TestWorkerPoolConcurrencyLimit(t *testing.T) {
	pool := NewWorkerPool(2)
	var concurrent atomic.Int64
	var maxConcurrent atomic.Int64

	for i := 0; i < 20; i++ {
		err := pool.Submit(func() {
			cur := concurrent.Add(1)
			// Track max concurrent
			for {
				old := maxConcurrent.Load()
				if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			concurrent.Add(-1)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	pool.Wait()
	if got := maxConcurrent.Load(); got > 2 {
		t.Errorf("expected max concurrency <= 2, got %d", got)
	}
	pool.Close()
}

func TestWorkerPoolClosePreventsSubmit(t *testing.T) {
	pool := NewWorkerPool(2)
	pool.Close()

	err := pool.Submit(func() {})
	if err != ErrPoolClosed {
		t.Errorf("expected ErrPoolClosed, got %v", err)
	}
}

func TestSemaphoreAcquireRelease(t *testing.T) {
	sem := NewSemaphore(2)
	sem.Acquire()
	sem.Acquire()

	if sem.TryAcquire() {
		t.Error("expected TryAcquire to fail when semaphore is full")
	}

	sem.Release()

	if !sem.TryAcquire() {
		t.Error("expected TryAcquire to succeed after release")
	}

	sem.Release()
	sem.Release()
}

func TestNewSemaphoreClamp(t *testing.T) {
	sem := NewSemaphore(0)
	if cap(sem) != 1 {
		t.Errorf("expected capacity 1, got %d", cap(sem))
	}

	sem2 := NewSemaphore(-10)
	if cap(sem2) != 1 {
		t.Errorf("expected capacity 1, got %d", cap(sem2))
	}
}
