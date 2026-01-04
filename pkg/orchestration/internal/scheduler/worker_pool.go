package orchestration

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// WorkerPool manages a pool of workers for concurrent task execution.
type WorkerPool struct {
	ctx       context.Context //nolint:containedctx // Context is necessary for worker pool lifecycle management
	taskQueue chan Task
	results   chan TaskResult
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	workers   int
	mu        sync.RWMutex
	running   bool
}

// TaskResult represents the result of a task execution.
type TaskResult struct {
	CompletedAt time.Time
	Error       error
	TaskID      string
	Attempts    int
	Duration    time.Duration
	Success     bool
}

// NewWorkerPool creates a new worker pool.
func NewWorkerPool(workers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		workers:   workers,
		taskQueue: make(chan Task, workers*2), // Buffer for better performance
		results:   make(chan TaskResult, workers*2),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start begins the worker pool execution.
func (wp *WorkerPool) Start() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.running {
		return
	}

	wp.running = true

	// Start workers
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// Stop gracefully stops the worker pool.
func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if !wp.running {
		return
	}

	wp.running = false
	wp.cancel()

	close(wp.taskQueue)
	wp.wg.Wait()

	close(wp.results)
}

// SubmitTask submits a task to the worker pool.
func (wp *WorkerPool) SubmitTask(task Task) error {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	if !wp.running {
		return errors.New("worker pool is not running")
	}

	select {
	case wp.taskQueue <- task:
		return nil
	case <-wp.ctx.Done():
		return errors.New("worker pool is shutting down")
	default:
		return errors.New("task queue is full")
	}
}

// GetResults returns a channel for receiving task results.
func (wp *WorkerPool) GetResults() <-chan TaskResult {
	return wp.results
}

// IsRunning returns whether the worker pool is running.
func (wp *WorkerPool) IsRunning() bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.running
}

// worker is the main worker goroutine.
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	for {
		select {
		case task, ok := <-wp.taskQueue:
			if !ok {
				return // Channel closed
			}

			startTime := time.Now()
			result := TaskResult{
				TaskID:      task.ID,
				CompletedAt: startTime,
			}

			// Execute task
			err := task.Execute()
			duration := time.Since(startTime)

			result.Success = err == nil
			result.Error = err
			result.Duration = duration
			result.Attempts = 1

			// Send result (non-blocking)
			select {
			case wp.results <- result:
			case <-wp.ctx.Done():
				return
			default:
				// Result channel full, log warning in production
				fmt.Printf("Warning: result channel full for task %s\n", task.ID)
			}

		case <-wp.ctx.Done():
			return
		}
	}
}

// GetStats returns current worker pool statistics.
func (wp *WorkerPool) GetStats() WorkerPoolStats {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	return WorkerPoolStats{
		Workers:     wp.workers,
		Running:     wp.running,
		QueueSize:   len(wp.taskQueue),
		ResultsSize: len(wp.results),
	}
}

// WorkerPoolStats holds statistics about the worker pool.
type WorkerPoolStats struct {
	Workers     int
	Running     bool
	QueueSize   int
	ResultsSize int
}
