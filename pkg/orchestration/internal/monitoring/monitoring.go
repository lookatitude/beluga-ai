package orchestration

import (
	"fmt"
	"sync"
)

// WorkflowMonitor tracks the state and progress of workflows.
type WorkflowMonitor struct {
	state   map[string]string
	logChan chan string
	mutex   sync.Mutex
}

// NewWorkflowMonitor initializes a new WorkflowMonitor.
func NewWorkflowMonitor() *WorkflowMonitor {
	return &WorkflowMonitor{
		state:   make(map[string]string),
		logChan: make(chan string, 100),
	}
}

// UpdateState updates the state of a task in the workflow.
func (wm *WorkflowMonitor) UpdateState(taskID, state string) {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()
	wm.state[taskID] = state
	wm.logChan <- fmt.Sprintf("Task %s state updated to: %s", taskID, state)
}

// GetState retrieves the current state of a task.
func (wm *WorkflowMonitor) GetState(taskID string) string {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()
	return wm.state[taskID]
}

// StartLogging starts a goroutine to log state changes.
func (wm *WorkflowMonitor) StartLogging() {
	go func() {
		for logEntry := range wm.logChan {
			_, _ = fmt.Println(logEntry)
		}
	}()
}
