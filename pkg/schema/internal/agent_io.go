package internal

// AgentAction represents an action taken by an agent, typically involving a tool.
type AgentAction struct {
	// Tool is the name of the tool to be used for this action.
	Tool string `json:"tool" yaml:"tool"`
	// ToolInput is the input to the tool. It can be a string or a structured map.
	ToolInput any `json:"tool_input" yaml:"tool_input"`
	// Log is a textual representation of the agent's thought process for taking this action.
	Log string `json:"log" yaml:"log"`
}

// AgentObservation represents the output or result obtained from executing an AgentAction.
type AgentObservation struct {
	ParsedOutput any    `json:"parsed_output,omitempty" yaml:"parsed_output,omitempty"`
	ActionLog    string `json:"action_log" yaml:"action_log"`
	Output       string `json:"output" yaml:"output"`
}

// Step represents a single step in an agent's execution trace.
// It pairs an action with its corresponding observation.
// This is useful for maintaining the history of agent interactions and for ReAct-style prompting.
// type Step struct { // Original problematic line was here, ensuring newline.
type Step struct {
	Observation AgentObservation `json:"observation" yaml:"observation"`
	Action      AgentAction      `json:"action" yaml:"action"`
}

// FinalAnswer represents the final output of an agent after it has completed its task.
// It might be a direct answer, a summary, or the result of its last action if that concludes its goal.
// type FinalAnswer struct { // Original problematic line was here, ensuring newline.
type FinalAnswer struct {
	// Output is the final response from the agent.
	Output string `json:"output" yaml:"output"`
	// SourceDocuments can be a list of documents that contributed to the final answer, especially in RAG contexts.
	SourceDocuments []any `json:"source_documents,omitempty" yaml:"source_documents,omitempty"`
	// IntermediateSteps can include the sequence of actions and observations that led to the final answer,
	// useful for transparency and debugging.
	IntermediateSteps []Step `json:"intermediate_steps,omitempty" yaml:"intermediate_steps,omitempty"`
}

// AgentFinish represents the final output from an agent when it has completed its task.
type AgentFinish struct {
	// ReturnValues is a map of key-value pairs that represent the agent's final output or result.
	ReturnValues map[string]any `json:"return_values" yaml:"return_values"`
	// Log is a textual representation of the agent's final thought process or conclusion.
	Log string `json:"log" yaml:"log"`
}

// AgentScratchPadEntry represents an intermediate step in an agent's thought process,
// consisting of an action and the observation resulting from that action.
// This is often used for logging or for constructing prompts for the LLM.
type AgentScratchPadEntry struct {
	Action      AgentAction `json:"action" yaml:"action"`
	Observation string      `json:"observation" yaml:"observation"`
}

// Agent-to-Agent (A2A) Communication Types

// AgentMessage represents a message sent between agents in A2A communication.
type AgentMessage struct {
	Payload        any              `json:"payload" yaml:"payload"`
	Metadata       map[string]any   `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	FromAgentID    string           `json:"from_agent_id" yaml:"from_agent_id"`
	ToAgentID      string           `json:"to_agent_id,omitempty" yaml:"to_agent_id,omitempty"`
	MessageID      string           `json:"message_id" yaml:"message_id"`
	ConversationID string           `json:"conversation_id,omitempty" yaml:"conversation_id,omitempty"`
	MessageType    AgentMessageType `json:"message_type" yaml:"message_type"`
	Timestamp      int64            `json:"timestamp" yaml:"timestamp"`
}

// AgentMessageType defines the type of message in A2A communication.
type AgentMessageType string

const (
	// AgentMessageRequest represents a request from one agent to another.
	AgentMessageRequest AgentMessageType = "request"
	// AgentMessageResponse represents a response to a request.
	AgentMessageResponse AgentMessageType = "response"
	// AgentMessageNotification represents a one-way notification.
	AgentMessageNotification AgentMessageType = "notification"
	// AgentMessageBroadcast represents a message sent to all agents.
	AgentMessageBroadcast AgentMessageType = "broadcast"
	// AgentMessageError represents an error message.
	AgentMessageError AgentMessageType = "error"
)

// AgentRequest represents a request payload in A2A communication.
type AgentRequest struct {
	Parameters map[string]any `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Action     string         `json:"action" yaml:"action"`
	Timeout    int            `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Priority   int            `json:"priority,omitempty" yaml:"priority,omitempty"`
}

// AgentResponse represents a response payload in A2A communication.
type AgentResponse struct {
	Result    any         `json:"result,omitempty" yaml:"result,omitempty"`
	Error     *AgentError `json:"error,omitempty" yaml:"error,omitempty"`
	RequestID string      `json:"request_id" yaml:"request_id"`
	Status    string      `json:"status" yaml:"status"`
}

// AgentError represents an error in A2A communication.
type AgentError struct {
	Details map[string]any `json:"details,omitempty" yaml:"details,omitempty"`
	Code    string         `json:"code" yaml:"code"`
	Message string         `json:"message" yaml:"message"`
}

// Event Types

// Event represents a system or domain event that can be published and consumed.
type Event struct {
	Payload   any            `json:"payload" yaml:"payload"`
	Metadata  map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	EventID   string         `json:"event_id" yaml:"event_id"`
	EventType string         `json:"event_type" yaml:"event_type"`
	Source    string         `json:"source" yaml:"source"`
	Version   string         `json:"version" yaml:"version"`
	Timestamp int64          `json:"timestamp" yaml:"timestamp"`
}

// AgentLifecycleEvent represents events related to agent lifecycle.
type AgentLifecycleEvent struct {
	// AgentID identifies the agent
	AgentID string `json:"agent_id" yaml:"agent_id"`
	// EventType specifies the lifecycle event type
	EventType AgentLifecycleEventType `json:"event_type" yaml:"event_type"`
	// PreviousState is the previous state (for state change events)
	PreviousState string `json:"previous_state,omitempty" yaml:"previous_state,omitempty"`
	// CurrentState is the current state (for state change events)
	CurrentState string `json:"current_state,omitempty" yaml:"current_state,omitempty"`
	// Reason provides additional context for the event
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
}

// AgentLifecycleEventType defines types of agent lifecycle events.
type AgentLifecycleEventType string

const (
	// AgentStarted indicates an agent has started.
	AgentStarted AgentLifecycleEventType = "agent_started"
	// AgentStopped indicates an agent has stopped.
	AgentStopped AgentLifecycleEventType = "agent_stopped"
	// AgentPaused indicates an agent has been paused.
	AgentPaused AgentLifecycleEventType = "agent_paused"
	// AgentResumed indicates an agent has been resumed.
	AgentResumed AgentLifecycleEventType = "agent_resumed"
	// AgentFailed indicates an agent has failed.
	AgentFailed AgentLifecycleEventType = "agent_failed"
	// AgentConfigUpdated indicates agent configuration was updated.
	AgentConfigUpdated AgentLifecycleEventType = "agent_config_updated"
)

// TaskEvent represents events related to task execution.
type TaskEvent struct {
	Result    any           `json:"result,omitempty" yaml:"result,omitempty"`
	Error     *AgentError   `json:"error,omitempty" yaml:"error,omitempty"`
	TaskID    string        `json:"task_id" yaml:"task_id"`
	AgentID   string        `json:"agent_id" yaml:"agent_id"`
	EventType TaskEventType `json:"event_type" yaml:"event_type"`
	TaskType  string        `json:"task_type,omitempty" yaml:"task_type,omitempty"`
	Status    string        `json:"status,omitempty" yaml:"status,omitempty"`
	Progress  int           `json:"progress,omitempty" yaml:"progress,omitempty"`
}

// TaskEventType defines types of task events.
type TaskEventType string

const (
	// TaskStarted indicates a task has started.
	TaskStarted TaskEventType = "task_started"
	// TaskProgress indicates task progress update.
	TaskProgress TaskEventType = "task_progress"
	// TaskCompleted indicates a task has completed successfully.
	TaskCompleted TaskEventType = "task_completed"
	// TaskFailed indicates a task has failed.
	TaskFailed TaskEventType = "task_failed"
	// TaskCancelled indicates a task has been canceled.
	TaskCancelled TaskEventType = "task_canceled"
)

// WorkflowEvent represents events related to workflow execution.
type WorkflowEvent struct {
	WorkflowID   string            `json:"workflow_id" yaml:"workflow_id"`
	EventType    WorkflowEventType `json:"event_type" yaml:"event_type"`
	CurrentStep  string            `json:"current_step,omitempty" yaml:"current_step,omitempty"`
	Status       string            `json:"status,omitempty" yaml:"status,omitempty"`
	Participants []string          `json:"participants,omitempty" yaml:"participants,omitempty"`
	TotalSteps   int               `json:"total_steps,omitempty" yaml:"total_steps,omitempty"`
}

// WorkflowEventType defines types of workflow events.
type WorkflowEventType string

const (
	// WorkflowStarted indicates a workflow has started.
	WorkflowStarted WorkflowEventType = "workflow_started"
	// WorkflowStepCompleted indicates a workflow step has completed.
	WorkflowStepCompleted WorkflowEventType = "workflow_step_completed"
	// WorkflowCompleted indicates a workflow has completed.
	WorkflowCompleted WorkflowEventType = "workflow_completed"
	// WorkflowFailed indicates a workflow has failed.
	WorkflowFailed WorkflowEventType = "workflow_failed"
	// WorkflowCancelled indicates a workflow has been canceled.
	WorkflowCancelled WorkflowEventType = "workflow_canceled"
)
