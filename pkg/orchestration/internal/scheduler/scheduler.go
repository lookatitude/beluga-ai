package orchestration

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Task represents a unit of work to be scheduled.
type Task struct {
	ID        string
	Execute   func() error
	DependsOn []string
}

// Scheduler manages task execution based on dependencies and priorities.
type Scheduler struct {
	tasks     map[string]*Task
	completed map[string]bool
	mutex     sync.Mutex
}

// NewScheduler initializes a new Scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks:     make(map[string]*Task),
		completed: make(map[string]bool),
	}
}

// AddTask adds a task to the scheduler.
func (s *Scheduler) AddTask(task *Task) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	s.tasks[task.ID] = task
	return nil
}

// Run executes all tasks in the correct order based on dependencies.
func (s *Scheduler) Run() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for id, task := range s.tasks {
		if err := s.runTask(id, task); err != nil {
			return err
		}
	}

	return nil
}

func (s *Scheduler) runTask(id string, task *Task) error {
	if s.completed[id] {
		return nil
	}

	for _, dep := range task.DependsOn {
		depTask, exists := s.tasks[dep]
		if !exists {
			return fmt.Errorf("dependency %s for task %s not found", dep, id)
		}
		if err := s.runTask(dep, depTask); err != nil {
			return err
		}
	}

	if err := task.Execute(); err != nil {
		return fmt.Errorf("task %s failed: %w", id, err)
	}

	s.completed[id] = true
	return nil
}

// ExecuteSequential runs tasks in a strictly sequential order.
func (s *Scheduler) ExecuteSequential() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for id, task := range s.tasks {
		if err := s.runTask(id, task); err != nil {
			return err
		}
	}

	return nil
}

// ExecuteAutonomous runs tasks without considering dependencies.
func (s *Scheduler) ExecuteAutonomous() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for id, task := range s.tasks {
		go func(taskID string, t *Task) {
			if err := t.Execute(); err != nil {
				fmt.Printf("Task %s failed: %v\n", taskID, err)
			}
		}(id, task)
	}

	return nil
}

// ExecuteConcurrent runs tasks concurrently using a worker pool
func (s *Scheduler) ExecuteConcurrent(workers int) error {
	workerPool := NewWorkerPool(workers)
	workerPool.Start()
	defer workerPool.Stop()

	results := workerPool.GetResults()

	// Submit all tasks
	for _, task := range s.tasks {
		if err := workerPool.SubmitTask(*task); err != nil {
			return fmt.Errorf("failed to submit task %s: %w", task.ID, err)
		}
	}

	// Collect results
	return s.collectWorkerResults(results, len(s.tasks))
}

// collectWorkerResults collects results from worker pool
func (s *Scheduler) collectWorkerResults(results <-chan TaskResult, totalTasks int) error {
	completedTasks := 0

	for completedTasks < totalTasks {
		result := <-results
		s.mutex.Lock()
		s.completed[result.TaskID] = result.Success
		s.mutex.Unlock()

		completedTasks++

		if !result.Success {
			fmt.Printf("Task %s failed: %v\n", result.TaskID, result.Error)
		}
	}

	return nil
}

// ExecuteWithRetry runs tasks with retry logic
func (s *Scheduler) ExecuteWithRetry(maxRetries int) error {
	retryExecutor := NewRetryExecutor(RetryConfig{
		MaxAttempts:   maxRetries,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
	})

	s.mutex.Lock()
	defer s.mutex.Unlock()

	for id, task := range s.tasks {
		if err := s.runTaskWithRetry(id, task, retryExecutor); err != nil {
			return err
		}
	}

	return nil
}

// runTaskWithRetry runs a single task with retry logic
func (s *Scheduler) runTaskWithRetry(id string, task *Task, retryExecutor *RetryExecutor) error {
	if s.completed[id] {
		return nil
	}

	// Check dependencies
	for _, dep := range task.DependsOn {
		depTask, exists := s.tasks[dep]
		if !exists {
			return fmt.Errorf("dependency %s for task %s not found", dep, id)
		}
		if err := s.runTaskWithRetry(dep, depTask, retryExecutor); err != nil {
			return err
		}
	}

	// Execute task with retry
	ctx := context.Background()
	if err := retryExecutor.ExecuteTaskWithRetry(ctx, *task); err != nil {
		return fmt.Errorf("task %s failed after retries: %w", id, err)
	}

	s.completed[id] = true
	return nil
}
