package scheduler

import (
	"context"
	"fmt"
	"time"

	// Potentially add imports for task definitions or workflow components
)

// Task represents a unit of work to be scheduled and executed.
// This is a basic representation; a real task might have more attributes
// like priority, dependencies, retry policies, etc.
type Task struct {
	ID          string
	Name        string
	ExecuteFunc func(ctx context.Context) error
	ScheduledAt time.Time
	// Add other fields like status, result, error, etc.
}

// Scheduler defines the interface for a task scheduler.
// It manages the lifecycle of tasks, including their scheduling and execution.
type Scheduler interface {
	// Schedule adds a task to be executed at a specific time or according to a schedule.
	Schedule(task Task) error
	// Start begins the scheduler's execution loop, processing scheduled tasks.
	Start(ctx context.Context) error
	// Stop gracefully shuts down the scheduler.
	Stop() error
	// GetTaskStatus returns the status of a specific task.
	// GetTaskStatus(taskID string) (string, error) // Example, might need more complex status object
}

// InMemoryScheduler is a simple in-memory implementation of the Scheduler interface.
// It's suitable for basic use cases or testing.
// For production, a persistent and more robust scheduler (e.g., backed by a DB or message queue) would be needed.
type InMemoryScheduler struct {
	tasks    []Task
	stopChan chan struct{}
	// Potentially use a priority queue or a more sophisticated data structure for tasks
}

// NewInMemoryScheduler creates a new InMemoryScheduler.
func NewInMemoryScheduler() *InMemoryScheduler {
	return &InMemoryScheduler{
		tasks:    make([]Task, 0),
		stopChan: make(chan struct{}),
	}
}

// Schedule adds a task to the in-memory scheduler.
// This simple implementation just appends it to a slice.
func (s *InMemoryScheduler) Schedule(task Task) error {
	s.tasks = append(s.tasks, task)
	fmt.Printf("Task %s scheduled for %s\n", task.Name, task.ScheduledAt.Format(time.RFC3339))
	return nil
}

// Start begins the scheduler's execution loop.
// This is a very basic loop that checks tasks every second.
// A real scheduler would be more sophisticated (e.g., using timers, worker pools).
func (s *InMemoryScheduler) Start(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	fmt.Println("In-memory scheduler started.")

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Scheduler context cancelled, stopping.")
			return ctx.Err()
		case <-s.stopChan:
			fmt.Println("Scheduler received stop signal, shutting down.")
			return nil
		case now := <-ticker.C:
			var remainingTasks []Task
			for _, task := range s.tasks {
				if now.After(task.ScheduledAt) || now.Equal(task.ScheduledAt) {
					fmt.Printf("Executing task: %s (ID: %s)\n", task.Name, task.ID)
					go func(t Task) { // Execute in a goroutine to not block the scheduler loop
						err := t.ExecuteFunc(ctx)
						if err != nil {
							fmt.Printf("Error executing task %s (ID: %s): %v\n", t.Name, t.ID, err)
						}
					}(task)
				} else {
					remainingTasks = append(remainingTasks, task)
				}
			}
			s.tasks = remainingTasks
		}
	}
}

// Stop signals the scheduler to stop processing tasks.
func (s *InMemoryScheduler) Stop() error {
	close(s.stopChan)
	fmt.Println("In-memory scheduler stop requested.")
	return nil
}

