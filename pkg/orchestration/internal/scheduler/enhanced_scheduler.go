package orchestration

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EnhancedScheduler provides advanced scheduling with worker pools and retry mechanisms.
type EnhancedScheduler struct {
	ctx            context.Context
	tasks          map[string]*Task
	completed      map[string]bool
	workerPool     *WorkerPool
	retryExecutor  *RetryExecutor
	circuitBreaker *CircuitBreaker
	bulkhead       *Bulkhead
	cancel         context.CancelFunc
	mutex          sync.RWMutex
}

// EnhancedTask extends the basic Task with retry and concurrency settings.
type EnhancedTask struct {
	Task
	RetryConfig            RetryConfig
	MaxRetries             int
	Priority               int
	Timeout                time.Duration
	RequiresCircuitBreaker bool
}

// NewEnhancedScheduler creates a new enhanced scheduler.
func NewEnhancedScheduler(workers int) *EnhancedScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &EnhancedScheduler{
		tasks:          make(map[string]*Task),
		completed:      make(map[string]bool),
		workerPool:     NewWorkerPool(workers),
		retryExecutor:  NewRetryExecutor(DefaultRetryConfig()),
		circuitBreaker: NewCircuitBreaker(5, 30*time.Second), // 5 failures, 30s reset
		bulkhead:       NewBulkhead(workers * 2),             // Allow 2x workers concurrent operations
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Start initializes the scheduler.
func (es *EnhancedScheduler) Start() {
	es.workerPool.Start()
}

// Stop gracefully stops the scheduler.
func (es *EnhancedScheduler) Stop() {
	es.cancel()
	es.workerPool.Stop()
}

// AddTask adds a task to the scheduler.
func (es *EnhancedScheduler) AddTask(task *Task) error {
	es.mutex.Lock()
	defer es.mutex.Unlock()

	if _, exists := es.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	es.tasks[task.ID] = task
	return nil
}

// AddEnhancedTask adds an enhanced task with retry and concurrency features.
func (es *EnhancedScheduler) AddEnhancedTask(task *EnhancedTask) error {
	es.mutex.Lock()
	defer es.mutex.Unlock()

	if _, exists := es.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	// Convert to basic task for storage
	basicTask := &Task{
		ID:        task.ID,
		Execute:   es.createEnhancedExecuteFunc(task),
		DependsOn: task.DependsOn,
	}

	es.tasks[task.ID] = basicTask
	return nil
}

// createEnhancedExecuteFunc creates an enhanced execute function with retry and circuit breaker.
func (es *EnhancedScheduler) createEnhancedExecuteFunc(task *EnhancedTask) func() error {
	return func() error {
		// Create task-specific retry config
		retryConfig := task.RetryConfig
		if retryConfig.MaxAttempts == 0 {
			retryConfig = DefaultRetryConfig()
			retryConfig.MaxAttempts = task.MaxRetries
			if task.MaxRetries == 0 {
				retryConfig.MaxAttempts = 3 // Default
			}
		}

		taskCtx, cancel := context.WithTimeout(es.ctx, task.Timeout)
		defer cancel()

		// Wrap execution with bulkhead for concurrency control
		return es.bulkhead.Execute(taskCtx, func() error {
			// Wrap with circuit breaker if required
			if task.RequiresCircuitBreaker {
				return es.circuitBreaker.Call(func() error {
					return es.retryExecutor.ExecuteWithRetry(taskCtx, task.Execute)
				})
			}

			// Execute with retry only
			return es.retryExecutor.ExecuteWithRetry(taskCtx, task.Execute)
		})
	}
}

// RunAsync executes tasks asynchronously using the worker pool.
func (es *EnhancedScheduler) RunAsync() error {
	es.mutex.RLock()
	defer es.mutex.RUnlock()

	results := es.workerPool.GetResults()

	// Submit all tasks to worker pool
	for _, task := range es.tasks {
		if err := es.submitTaskWithDependencies(task); err != nil {
			return fmt.Errorf("failed to submit task %s: %w", task.ID, err)
		}
	}

	// Collect results
	return es.collectResults(results)
}

// submitTaskWithDependencies submits a task considering its dependencies.
func (es *EnhancedScheduler) submitTaskWithDependencies(task *Task) error {
	// Check dependencies
	for _, depID := range task.DependsOn {
		if !es.completed[depID] {
			return fmt.Errorf("dependency %s for task %s not completed", depID, task.ID)
		}
	}

	return es.workerPool.SubmitTask(*task)
}

// collectResults collects and processes task results.
func (es *EnhancedScheduler) collectResults(results <-chan TaskResult) error {
	completedTasks := 0
	totalTasks := len(es.tasks)

	for completedTasks < totalTasks {
		select {
		case result := <-results:
			es.mutex.Lock()
			es.completed[result.TaskID] = result.Success
			es.mutex.Unlock()

			completedTasks++

			if !result.Success {
				_, _ = fmt.Printf("Task %s failed after %d attempts: %v\n", result.TaskID, result.Attempts, result.Error)
			} else {
				_, _ = fmt.Printf("Task %s completed successfully in %v\n", result.TaskID, result.Duration)
			}

		case <-es.ctx.Done():
			return es.ctx.Err()
		}
	}

	return nil
}

// RunSequential executes tasks sequentially with enhanced features.
func (es *EnhancedScheduler) RunSequential() error {
	es.mutex.Lock()
	defer es.mutex.Unlock()

	for id, task := range es.tasks {
		if err := es.runEnhancedTask(id, task); err != nil {
			return err
		}
	}

	return nil
}

func (es *EnhancedScheduler) runEnhancedTask(id string, task *Task) error {
	if es.completed[id] {
		return nil
	}

	// Check dependencies
	for _, dep := range task.DependsOn {
		depTask, exists := es.tasks[dep]
		if !exists {
			return fmt.Errorf("dependency %s for task %s not found", dep, id)
		}
		if err := es.runEnhancedTask(dep, depTask); err != nil {
			return err
		}
	}

	// Execute task
	if err := task.Execute(); err != nil {
		return fmt.Errorf("task %s failed: %w", id, err)
	}

	es.completed[id] = true
	return nil
}

// GetStats returns scheduler statistics.
func (es *EnhancedScheduler) GetStats() SchedulerStats {
	es.mutex.RLock()
	defer es.mutex.RUnlock()

	return SchedulerStats{
		TotalTasks:          len(es.tasks),
		CompletedTasks:      es.countCompletedTasks(),
		WorkerPoolStats:     es.workerPool.GetStats(),
		CircuitBreakerState: es.circuitBreaker.GetState(),
		BulkheadConcurrency: es.bulkhead.GetCurrentConcurrency(),
		BulkheadCapacity:    es.bulkhead.GetCapacity(),
	}
}

// countCompletedTasks counts completed tasks.
func (es *EnhancedScheduler) countCompletedTasks() int {
	count := 0
	for _, completed := range es.completed {
		if completed {
			count++
		}
	}
	return count
}

// SchedulerStats holds statistics about the scheduler.
type SchedulerStats struct {
	TotalTasks          int
	CompletedTasks      int
	WorkerPoolStats     WorkerPoolStats
	CircuitBreakerState CircuitState
	BulkheadConcurrency int
	BulkheadCapacity    int
}

// SetRetryConfig sets the retry configuration.
func (es *EnhancedScheduler) SetRetryConfig(config RetryConfig) {
	es.retryExecutor = NewRetryExecutor(config)
}

// SetCircuitBreakerConfig sets the circuit breaker configuration.
func (es *EnhancedScheduler) SetCircuitBreakerConfig(failureThreshold int, resetTimeout time.Duration) {
	es.circuitBreaker = NewCircuitBreaker(failureThreshold, resetTimeout)
}

// SetBulkheadCapacity sets the bulkhead capacity.
func (es *EnhancedScheduler) SetBulkheadCapacity(capacity int) {
	es.bulkhead = NewBulkhead(capacity)
}
