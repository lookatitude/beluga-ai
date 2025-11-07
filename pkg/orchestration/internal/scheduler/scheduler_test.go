package orchestration

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestNewScheduler(t *testing.T) {
	scheduler := NewScheduler()
	require.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.tasks)
	assert.NotNil(t, scheduler.completed)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func TestScheduler_AddTask(t *testing.T) {
	scheduler := NewScheduler()

	task := &Task{
		ID: "test-task",
		Execute: func() error {
			return nil
		},
	}

	err := scheduler.AddTask(task)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	assert.NoError(t, err)
	assert.Contains(t, scheduler.tasks, "test-task")
}

func TestScheduler_AddTask_DuplicateID(t *testing.T) {
	scheduler := NewScheduler()

	task := &Task{
		ID: "test-task",
		Execute: func() error {
			return nil
		},
	}

	// Add first task
	err := scheduler.AddTask(task)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	// Try to add duplicate
	err = scheduler.AddTask(task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestScheduler_Run(t *testing.T) {
	scheduler := NewScheduler()

	executedTasks := make(map[string]bool)

	task1 := &Task{
		ID: "task1",
		Execute: func() error {
			executedTasks["task1"] = true
			return nil
		},
	}

	task2 := &Task{
		ID: "task2",
		Execute: func() error {
			executedTasks["task2"] = true
			return nil
		},
		DependsOn: []string{"task1"},
	}

	scheduler.AddTask(task1)
	scheduler.AddTask(task2)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	err := scheduler.Run()
	assert.NoError(t, err)
	assert.True(t, executedTasks["task1"])
	assert.True(t, executedTasks["task2"])
	assert.True(t, scheduler.completed["task1"])
	assert.True(t, scheduler.completed["task2"])
}

func TestScheduler_Run_WithMissingDependency(t *testing.T) {
	scheduler := NewScheduler()

	task := &Task{
		ID: "task1",
		Execute: func() error {
			return nil
		},
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		DependsOn: []string{"nonexistent"},
	}

	scheduler.AddTask(task)

	err := scheduler.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestScheduler_Run_WithTaskFailure(t *testing.T) {
	scheduler := NewScheduler()

	task1 := &Task{
		ID: "task1",
		Execute: func() error {
			return errors.New("task failed")
		},
	}

	task2 := &Task{
		ID: "task2",
		Execute: func() error {
			return nil
		},
		DependsOn: []string{"task1"},
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	}

	scheduler.AddTask(task1)
	scheduler.AddTask(task2)

	err := scheduler.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task failed")
	// task2 should not be marked as completed due to dependency failure
	assert.False(t, scheduler.completed["task2"])
}

func TestScheduler_ExecuteSequential(t *testing.T) {
	scheduler := NewScheduler()

	executionOrder := make([]string, 0)

	task1 := &Task{
		ID: "task1",
		Execute: func() error {
			executionOrder = append(executionOrder, "task1")
			return nil
		},
	}

	task2 := &Task{
		ID: "task2",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		Execute: func() error {
			executionOrder = append(executionOrder, "task2")
			return nil
		},
	}

	scheduler.AddTask(task1)
	scheduler.AddTask(task2)

	err := scheduler.ExecuteSequential()
	assert.NoError(t, err)
	assert.Equal(t, []string{"task1", "task2"}, executionOrder)
}

func TestScheduler_ExecuteAutonomous(t *testing.T) {
	scheduler := NewScheduler()

	executedTasks := make(map[string]bool)

	task1 := &Task{
		ID: "task1",
		Execute: func() error {
			time.Sleep(10 * time.Millisecond) // Small delay to ensure concurrent execution
			executedTasks["task1"] = true
			return nil
		},
	}

	task2 := &Task{
		ID: "task2",
		Execute: func() error {
			time.Sleep(10 * time.Millisecond)
			executedTasks["task2"] = true
			return nil
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		},
	}

	scheduler.AddTask(task1)
	scheduler.AddTask(task2)

	err := scheduler.ExecuteAutonomous()
	assert.NoError(t, err)

	// Give some time for goroutines to complete
	time.Sleep(50 * time.Millisecond)

	assert.True(t, executedTasks["task1"])
	assert.True(t, executedTasks["task2"])
}

func TestScheduler_ExecuteConcurrent(t *testing.T) {
	scheduler := NewScheduler()

	executedTasks := make(map[string]bool)

	task1 := &Task{
		ID: "task1",
		Execute: func() error {
			time.Sleep(20 * time.Millisecond)
			executedTasks["task1"] = true
			return nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	task2 := &Task{
		ID: "task2",
		Execute: func() error {
			time.Sleep(20 * time.Millisecond)
			executedTasks["task2"] = true
			return nil
		},
	}

	scheduler.AddTask(task1)
	scheduler.AddTask(task2)

	err := scheduler.ExecuteConcurrent(2)
	assert.NoError(t, err)
	assert.True(t, executedTasks["task1"])
	assert.True(t, executedTasks["task2"])
}

func TestScheduler_ExecuteWithRetry(t *testing.T) {
	scheduler := NewScheduler()

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	attemptCount := 0

	task := &Task{
		ID: "retry-task",
		Execute: func() error {
			attemptCount++
			if attemptCount < 3 {
				return errors.New("temporary failure")
			}
			return nil
		},
	}

	scheduler.AddTask(task)

	err := scheduler.ExecuteWithRetry(5)
	assert.NoError(t, err)
	assert.Equal(t, 3, attemptCount)
}

func TestScheduler_ExecuteWithRetry_ExhaustRetries(t *testing.T) {
	scheduler := NewScheduler()

	attemptCount := 0

	task := &Task{
		ID: "failing-task",
		Execute: func() error {
			attemptCount++
			return errors.New("persistent failure")
		},
	}

	scheduler.AddTask(task)

	err := scheduler.ExecuteWithRetry(2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed after retries")
	assert.Equal(t, 2, attemptCount)
}
