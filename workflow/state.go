package workflow

import "time"

// EventType identifies the kind of history event.
type EventType string

const (
	// EventWorkflowStarted records the start of a workflow.
	EventWorkflowStarted EventType = "workflow_started"
	// EventWorkflowCompleted records successful completion.
	EventWorkflowCompleted EventType = "workflow_completed"
	// EventWorkflowFailed records a workflow failure.
	EventWorkflowFailed EventType = "workflow_failed"
	// EventWorkflowCanceled records cancellation.
	EventWorkflowCanceled EventType = "workflow_canceled"
	// EventActivityStarted records the start of an activity.
	EventActivityStarted EventType = "activity_started"
	// EventActivityCompleted records activity completion.
	EventActivityCompleted EventType = "activity_completed"
	// EventActivityFailed records an activity failure.
	EventActivityFailed EventType = "activity_failed"
	// EventSignalReceived records an incoming signal.
	EventSignalReceived EventType = "signal_received"
	// EventTimerFired records a sleep/timer completion.
	EventTimerFired EventType = "timer_fired"
)

// HistoryEvent is a single recorded event in the workflow's execution history.
type HistoryEvent struct {
	// ID is the sequential event identifier.
	ID int
	// Type identifies the event kind.
	Type EventType
	// Timestamp is when the event occurred.
	Timestamp time.Time
	// ActivityName identifies the activity (for activity events).
	ActivityName string
	// Input is the activity/workflow input.
	Input any
	// Result is the activity/workflow result.
	Result any
	// Error records any error message.
	Error string
	// SignalName is the signal name (for signal events).
	SignalName string
	// SignalPayload is the signal data.
	SignalPayload any
	// Duration records the sleep duration (for timer events).
	Duration time.Duration
}

// WorkflowState holds the complete state of a workflow execution.
type WorkflowState struct {
	// WorkflowID is the unique workflow identifier.
	WorkflowID string
	// RunID is the unique run identifier.
	RunID string
	// Status is the current workflow status.
	Status WorkflowStatus
	// Input is the original workflow input.
	Input any
	// Result is the workflow result (if completed).
	Result any
	// Error is the error message (if failed).
	Error string
	// History is the ordered sequence of events.
	History []HistoryEvent
	// CreatedAt is when the workflow was created.
	CreatedAt time.Time
	// UpdatedAt is when the workflow was last updated.
	UpdatedAt time.Time
}

// WorkflowFilter specifies criteria for listing workflows.
type WorkflowFilter struct {
	// Status filters by workflow status. Empty means all statuses.
	Status WorkflowStatus
	// Limit is the maximum number of results.
	Limit int
}
