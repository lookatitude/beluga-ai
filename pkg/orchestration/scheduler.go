package orchestration

import (
	"fmt"
	"sync"
)

// Task represents a unit of work to be scheduled.
type Task struct {
	ID       string
	Execute  func() error
	DependsOn []string
}

// Scheduler manages task execution based on dependencies and priorities.
type Scheduler struct {
	tasks      map[string]*Task
	completed  map[string]bool
	mutex      sync.Mutex
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