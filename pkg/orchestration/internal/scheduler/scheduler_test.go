package orchestration

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScheduler(t *testing.T) {
	scheduler := NewScheduler()
	require.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.tasks)
	assert.NotNil(t, scheduler.completed)
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
	require.NoError(t, err)
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
	require.NoError(t, err)

	// Try to add duplicate
	err = scheduler.AddTask(task)
	require.Error(t, err)
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

	_ = scheduler.AddTask(task1)
	_ = scheduler.AddTask(task2)

	err := scheduler.Run()
	require.NoError(t, err)
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
		DependsOn: []string{"nonexistent"},
	}

	_ = scheduler.AddTask(task)

	err := scheduler.Run()
	require.Error(t, err)
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
	}

	_ = scheduler.AddTask(task1)
	_ = scheduler.AddTask(task2)

	err := scheduler.Run()
	require.Error(t, err)
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
		Execute: func() error {
			executionOrder = append(executionOrder, "task2")
			return nil
		},
	}

	_ = scheduler.AddTask(task1)
	_ = scheduler.AddTask(task2)

	err := scheduler.ExecuteSequential()
	require.NoError(t, err)
	assert.Equal(t, []string{"task1", "task2"}, executionOrder)
}

func TestScheduler_ExecuteAutonomous(t *testing.T) {
	scheduler := NewScheduler()

	var mu sync.Mutex
	executedTasks := make(map[string]bool)

	task1 := &Task{
		ID: "task1",
		Execute: func() error {
			time.Sleep(10 * time.Millisecond) // Small delay to ensure concurrent execution
			mu.Lock()
			executedTasks["task1"] = true
			mu.Unlock()
			return nil
		},
	}

	task2 := &Task{
		ID: "task2",
		Execute: func() error {
			time.Sleep(10 * time.Millisecond)
			mu.Lock()
			executedTasks["task2"] = true
			mu.Unlock()
			return nil
		},
	}

	_ = scheduler.AddTask(task1)
	_ = scheduler.AddTask(task2)

	err := scheduler.ExecuteAutonomous()
	require.NoError(t, err)

	// Give some time for goroutines to complete
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	task1Executed := executedTasks["task1"]
	task2Executed := executedTasks["task2"]
	mu.Unlock()
	assert.True(t, task1Executed)
	assert.True(t, task2Executed)
}

func TestScheduler_ExecuteConcurrent(t *testing.T) {
	scheduler := NewScheduler()

	var mu sync.Mutex
	executedTasks := make(map[string]bool)

	task1 := &Task{
		ID: "task1",
		Execute: func() error {
			time.Sleep(20 * time.Millisecond)
			mu.Lock()
			executedTasks["task1"] = true
			mu.Unlock()
			return nil
		},
	}

	task2 := &Task{
		ID: "task2",
		Execute: func() error {
			time.Sleep(20 * time.Millisecond)
			mu.Lock()
			executedTasks["task2"] = true
			mu.Unlock()
			return nil
		},
	}

	_ = scheduler.AddTask(task1)
	_ = scheduler.AddTask(task2)

	err := scheduler.ExecuteConcurrent(2)
	require.NoError(t, err)

	mu.Lock()
	task1Executed := executedTasks["task1"]
	task2Executed := executedTasks["task2"]
	mu.Unlock()
	assert.True(t, task1Executed)
	assert.True(t, task2Executed)
}

func TestScheduler_ExecuteWithRetry(t *testing.T) {
	scheduler := NewScheduler()

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

	_ = scheduler.AddTask(task)

	err := scheduler.ExecuteWithRetry(5)
	require.NoError(t, err)
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

	_ = scheduler.AddTask(task)

	err := scheduler.ExecuteWithRetry(2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed after retries")
	assert.Equal(t, 2, attemptCount)
}
