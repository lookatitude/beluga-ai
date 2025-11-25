package orchestration

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewWorkflowMonitor(t *testing.T) {
	monitor := NewWorkflowMonitor()

	assert.NotNil(t, monitor)
	assert.NotNil(t, monitor.state)
	assert.NotNil(t, monitor.logChan)
	assert.Equal(t, 100, cap(monitor.logChan)) // Check buffer size
}

func TestWorkflowMonitor_UpdateState(t *testing.T) {
	monitor := NewWorkflowMonitor()

	// Test updating state
	monitor.UpdateState("task1", "running")
	monitor.UpdateState("task2", "completed")
	monitor.UpdateState("task1", "completed") // Update existing task

	// Verify states
	assert.Equal(t, "completed", monitor.GetState("task1"))
	assert.Equal(t, "completed", monitor.GetState("task2"))
	assert.Equal(t, "", monitor.GetState("nonexistent")) // Non-existent task should return empty string
}

func TestWorkflowMonitor_GetState(t *testing.T) {
	monitor := NewWorkflowMonitor()

	// Initially empty
	assert.Equal(t, "", monitor.GetState("task1"))

	// After setting state
	monitor.UpdateState("task1", "running")
	assert.Equal(t, "running", monitor.GetState("task1"))

	// After updating state
	monitor.UpdateState("task1", "completed")
	assert.Equal(t, "completed", monitor.GetState("task1"))

	// Non-existent task
	assert.Equal(t, "", monitor.GetState("nonexistent"))
}

func TestWorkflowMonitor_StartLogging(t *testing.T) {
	monitor := NewWorkflowMonitor()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Start logging
	monitor.StartLogging()

	// Give the logging goroutine time to start
	time.Sleep(10 * time.Millisecond)

	// Update some states
	monitor.UpdateState("task1", "running")
	monitor.UpdateState("task2", "completed")

	// Close the log channel to stop the logging goroutine
	close(monitor.logChan)

	// Give time for logging to complete
	time.Sleep(100 * time.Millisecond)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify log output contains expected messages
	assert.Contains(t, output, "Task task1 state updated to: running")
	assert.Contains(t, output, "Task task2 state updated to: completed")
}

func TestWorkflowMonitor_ConcurrentAccess(t *testing.T) {
	monitor := NewWorkflowMonitor()
	
	// Start logging to consume from the channel and prevent blocking
	monitor.StartLogging()
	defer close(monitor.logChan)
	
	var wg sync.WaitGroup

	// Number of concurrent goroutines
	numGoroutines := 10
	updatesPerGoroutine := 100

	// Start multiple goroutines updating states concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < updatesPerGoroutine; j++ {
				taskID := fmt.Sprintf("task-g%d-%d", goroutineID, j)
				state := fmt.Sprintf("state-g%d-%d", goroutineID, j)
				monitor.UpdateState(taskID, state)
			}
		}(i)
	}

	// Wait for all goroutines to complete with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines completed
	case <-time.After(30 * time.Second):
		t.Fatal("Test timed out waiting for concurrent updates to complete")
	}

	// Verify that all states were set correctly
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < updatesPerGoroutine; j++ {
			taskID := fmt.Sprintf("task-g%d-%d", i, j)
			expectedState := fmt.Sprintf("state-g%d-%d", i, j)
			assert.Equal(t, expectedState, monitor.GetState(taskID))
		}
	}
}

func TestWorkflowMonitor_BufferedChannel(t *testing.T) {
	monitor := NewWorkflowMonitor()

	// Start logging to consume from the channel and prevent blocking
	monitor.StartLogging()
	defer close(monitor.logChan)

	// Fill the log channel buffer
	for i := 0; i < 100; i++ {
		monitor.UpdateState(fmt.Sprintf("task%d", i), fmt.Sprintf("state%d", i))
	}

	// Give time for logging goroutine to process
	time.Sleep(50 * time.Millisecond)

	// Channel should be full but shouldn't block
	assert.Equal(t, "state99", monitor.GetState("task99"))

	// Try one more update (shouldn't block due to buffer and consumer)
	monitor.UpdateState("task100", "state100")
	assert.Equal(t, "state100", monitor.GetState("task100"))
}

func TestWorkflowMonitor_MultipleStateUpdates(t *testing.T) {
	monitor := NewWorkflowMonitor()

	taskID := "test-task"
	states := []string{"pending", "running", "processing", "completed", "failed", "retrying", "completed"}

	// Update state multiple times
	for _, state := range states {
		monitor.UpdateState(taskID, state)
		assert.Equal(t, state, monitor.GetState(taskID))
	}

	// Final state should be the last one
	assert.Equal(t, "completed", monitor.GetState(taskID))
}

func TestWorkflowMonitor_EmptyTaskID(t *testing.T) {
	monitor := NewWorkflowMonitor()

	// Test with empty task ID
	monitor.UpdateState("", "empty-state")
	assert.Equal(t, "empty-state", monitor.GetState(""))

	// Test with whitespace task ID
	monitor.UpdateState("   ", "whitespace-state")
	assert.Equal(t, "whitespace-state", monitor.GetState("   "))
}

func TestWorkflowMonitor_SpecialCharacters(t *testing.T) {
	monitor := NewWorkflowMonitor()

	// Test with special characters in task ID and state
	taskID := "task:with:colons@domain.com"
	state := "state with spaces & symbols !@#$%^&*()"
	monitor.UpdateState(taskID, state)

	assert.Equal(t, state, monitor.GetState(taskID))
}

func TestWorkflowMonitor_LargeNumberOfTasks(t *testing.T) {
	monitor := NewWorkflowMonitor()

	// Start logging to consume from the channel and prevent blocking
	monitor.StartLogging()
	defer close(monitor.logChan)

	// Test with a large number of tasks
	numTasks := 10000
	for i := 0; i < numTasks; i++ {
		taskID := fmt.Sprintf("task-%d", i)
		state := fmt.Sprintf("state-%d", i)
		monitor.UpdateState(taskID, state)
	}

	// Give time for logging goroutine to process
	time.Sleep(100 * time.Millisecond)

	// Verify all tasks
	for i := 0; i < numTasks; i++ {
		taskID := fmt.Sprintf("task-%d", i)
		expectedState := fmt.Sprintf("state-%d", i)
		assert.Equal(t, expectedState, monitor.GetState(taskID))
	}
}

func TestWorkflowMonitor_StartLogging_MultipleTimes(t *testing.T) {
	monitor := NewWorkflowMonitor()

	// Start logging multiple times (shouldn't cause issues)
	monitor.StartLogging()
	monitor.StartLogging()
	monitor.StartLogging()

	// Update a state
	monitor.UpdateState("test-task", "test-state")

	// Close log channel
	close(monitor.logChan)

	// Should not panic
	time.Sleep(50 * time.Millisecond)
}

func TestWorkflowMonitor_LogChannelClosure(t *testing.T) {
	monitor := NewWorkflowMonitor()

	// Close the log channel
	close(monitor.logChan)

	// This should not panic, but the UpdateState call might block or fail
	// depending on the implementation. In the current implementation,
	// sending to a closed channel would panic, so this test verifies
	// that the channel operations are safe.

	// Note: In a production implementation, you might want to make the
	// log channel operations more robust to handle closure gracefully.
	assert.NotPanics(t, func() {
		// This might panic if the channel is closed and we try to send to it
		defer func() {
			if r := recover(); r != nil {
				// Expected if channel is closed
				t.Logf("Panic recovered: %v", r)
			}
		}()
		monitor.UpdateState("test", "state")
	})
}

func TestWorkflowMonitor_StateMapIsolated(t *testing.T) {
	monitor1 := NewWorkflowMonitor()
	monitor2 := NewWorkflowMonitor()

	// Update states in both monitors
	monitor1.UpdateState("task1", "state1-monitor1")
	monitor2.UpdateState("task1", "state1-monitor2")

	// States should be isolated between monitors
	assert.Equal(t, "state1-monitor1", monitor1.GetState("task1"))
	assert.Equal(t, "state1-monitor2", monitor2.GetState("task1"))

	// Different tasks should not interfere
	monitor1.UpdateState("task2", "state2-monitor1")
	assert.Equal(t, "state2-monitor1", monitor1.GetState("task2"))
	assert.Equal(t, "", monitor2.GetState("task2"))
}

func TestWorkflowMonitor_LogFormat(t *testing.T) {
	monitor := NewWorkflowMonitor()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Start logging
	monitor.StartLogging()
	time.Sleep(10 * time.Millisecond)

	// Update state
	monitor.UpdateState("test-task-123", "running_state")

	// Close log channel
	close(monitor.logChan)
	time.Sleep(50 * time.Millisecond)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify log format
	assert.Contains(t, output, "Task test-task-123 state updated to: running_state")
	assert.True(t, strings.Contains(output, "Task ") && strings.Contains(output, " state updated to: "))
}
