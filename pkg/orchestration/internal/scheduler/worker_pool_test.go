package orchestration

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestNewWorkerPool(t *testing.T) {
	pool := NewWorkerPool(3)
	require.NotNil(t, pool)
	assert.Equal(t, 3, pool.workers)
	assert.NotNil(t, pool.taskQueue)
	assert.NotNil(t, pool.results)
	assert.False(t, pool.running)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func TestWorkerPool_StartStop(t *testing.T) {
	pool := NewWorkerPool(2)

	// Test starting
	pool.Start()
	assert.True(t, pool.running)

	// Test stopping
	pool.Stop()
	assert.False(t, pool.running)
}

func TestWorkerPool_SubmitTask(t *testing.T) {
	pool := NewWorkerPool(2)
	pool.Start()
	defer pool.Stop()

	task := Task{
		ID: "test-task",
		Execute: func() error {
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	}

	err := pool.SubmitTask(task)
	assert.NoError(t, err)

	// Wait for result
	select {
	case result := <-pool.GetResults():
		assert.Equal(t, "test-task", result.TaskID)
		assert.True(t, result.Success)
		assert.Nil(t, result.Error)
	case <-time.After(100 * time.Millisecond):
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		t.Fatal("Expected result not received")
	}
}

func TestWorkerPool_SubmitTaskWhenNotRunning(t *testing.T) {
	pool := NewWorkerPool(2)
	// Don't start the pool

	task := Task{
		ID:      "test-task",
		Execute: func() error { return nil },
	}

	err := pool.SubmitTask(task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestWorkerPool_ConcurrentTasks(t *testing.T) {
	pool := NewWorkerPool(3)
	pool.Start()
	defer pool.Stop()

	var wg sync.WaitGroup
	results := pool.GetResults()
	resultCount := 0

	// Submit multiple tasks
	for i := 0; i < 5; i++ {
		wg.Add(1)
		taskID := i
		task := Task{
			ID: string(rune('A' + taskID)),
			Execute: func() error {
				time.Sleep(20 * time.Millisecond)
				wg.Done()
				return nil
			},
		}

		err := pool.SubmitTask(task)
		require.NoError(t, err)
	}

	// Wait for all tasks to complete
	wg.Wait()

	// Collect all results
	timeout := time.After(200 * time.Millisecond)
	for resultCount < 5 {
		select {
		case result := <-results:
			assert.True(t, result.Success)
			assert.Nil(t, result.Error)
			resultCount++
		case <-timeout:
			t.Fatalf("Only received %d results out of 5", resultCount)
		}
	}
}

func TestWorkerPool_TaskFailure(t *testing.T) {
	pool := NewWorkerPool(2)
	pool.Start()
	defer pool.Stop()

	task := Task{
		ID: "failing-task",
		Execute: func() error {
			return assert.AnError // Using testify's assert.AnError
		},
	}

	err := pool.SubmitTask(task)
	assert.NoError(t, err)

	select {
	case result := <-pool.GetResults():
		assert.Equal(t, "failing-task", result.TaskID)
		assert.False(t, result.Success)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		assert.NotNil(t, result.Error)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected result not received")
	}
}

func TestWorkerPool_GetStats(t *testing.T) {
	pool := NewWorkerPool(3)

	stats := pool.GetStats()
	assert.Equal(t, 3, stats.Workers)
	assert.False(t, stats.Running)
	assert.Equal(t, 0, stats.QueueSize)
	assert.Equal(t, 0, stats.ResultsSize)

	pool.Start()
	defer pool.Stop()

	stats = pool.GetStats()
	assert.True(t, stats.Running)
}
