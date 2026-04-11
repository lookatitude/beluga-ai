package sleeptime

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Priority determines the execution order of tasks during idle periods.
// Lower numeric values indicate higher priority.
type Priority int

const (
	// PriorityHigh tasks run first during idle periods.
	PriorityHigh Priority = 0

	// PriorityNormal tasks run after high-priority tasks.
	PriorityNormal Priority = 10

	// PriorityLow tasks run only when higher-priority tasks are complete.
	PriorityLow Priority = 20
)

// SessionState provides read-only access to session data for task execution.
// Tasks use this to inspect session history without modifying it directly.
type SessionState struct {
	// SessionID is the unique identifier of the session.
	SessionID string

	// AgentID is the identifier of the agent that owns the session.
	AgentID string

	// TurnCount is the number of conversation turns in the session.
	TurnCount int

	// LastActivity is when the session last had user activity.
	LastActivity time.Time

	// Metadata holds arbitrary key-value data from the session state.
	Metadata map[string]any
}

// TaskResult holds the outcome of a task execution.
type TaskResult struct {
	// TaskName is the name of the task that produced this result.
	TaskName string

	// Success indicates whether the task completed without error.
	Success bool

	// Duration is how long the task took to execute.
	Duration time.Duration

	// ItemsProcessed is the number of items the task operated on.
	ItemsProcessed int

	// Error holds the error message if the task failed. Empty on success.
	Error string
}

// Task defines a unit of background work that can run during idle periods.
// Implementations must respect context cancellation for preemption.
type Task interface {
	// Name returns a unique identifier for this task.
	Name() string

	// Priority returns the execution priority of this task.
	Priority() Priority

	// ShouldRun reports whether the task should execute given the current
	// session state. Tasks that have nothing to do should return false.
	ShouldRun(ctx context.Context, state SessionState) bool

	// Run executes the task. It must respect context cancellation and return
	// promptly when the context is cancelled (preemption on wake).
	Run(ctx context.Context, state SessionState) (TaskResult, error)
}

// TaskFactory creates a Task from a configuration map.
type TaskFactory func(cfg map[string]any) (Task, error)

var (
	taskMu       sync.RWMutex
	taskRegistry = make(map[string]TaskFactory)
)

// RegisterTask registers a task factory under the given name. It is typically
// called from init() in task implementation packages. Registering a duplicate
// name overwrites the previous factory.
func RegisterTask(name string, f TaskFactory) {
	taskMu.Lock()
	defer taskMu.Unlock()
	taskRegistry[name] = f
}

// NewTask creates a task by looking up the registered factory for the given
// name and invoking it with the provided configuration.
func NewTask(name string, cfg map[string]any) (Task, error) {
	taskMu.RLock()
	f, ok := taskRegistry[name]
	taskMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("sleeptime: unknown task %q (registered: %v)", name, ListTasks())
	}
	return f(cfg)
}

// ListTasks returns the sorted names of all registered task factories.
func ListTasks() []string {
	taskMu.RLock()
	defer taskMu.RUnlock()
	names := make([]string, 0, len(taskRegistry))
	for name := range taskRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
