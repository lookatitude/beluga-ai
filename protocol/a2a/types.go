// Package a2a implements the Agent-to-Agent (A2A) protocol for multi-agent
// collaboration. It provides an A2A server that exposes a Beluga agent as a
// remote agent with an Agent Card, task lifecycle management, and HTTP endpoints.
//
// Usage:
//
//	// Expose an agent via A2A
//	card := a2a.AgentCard{Name: "my-agent", Endpoint: "http://localhost:9090"}
//	srv := a2a.NewServer(myAgent, card)
//	srv.Serve(ctx, ":9090")
//
//	// Connect to a remote A2A agent
//	remote, err := a2a.NewRemoteAgent("http://localhost:9090")
//	result, err := remote.Invoke(ctx, "Hello")
package a2a

// TaskStatus represents the lifecycle state of an A2A task.
type TaskStatus string

const (
	// StatusSubmitted indicates the task has been received but not yet started.
	StatusSubmitted TaskStatus = "submitted"
	// StatusWorking indicates the task is currently being processed.
	StatusWorking TaskStatus = "working"
	// StatusCompleted indicates the task has finished successfully.
	StatusCompleted TaskStatus = "completed"
	// StatusFailed indicates the task has failed.
	StatusFailed TaskStatus = "failed"
	// StatusCanceled indicates the task was canceled.
	StatusCanceled TaskStatus = "canceled"
)

// AgentCard describes a remote agent's identity and capabilities.
// It is served at the well-known URL /.well-known/agent.json.
type AgentCard struct {
	Name         string       `json:"name"`
	Description  string       `json:"description,omitempty"`
	Version      string       `json:"version,omitempty"`
	Capabilities []string     `json:"capabilities,omitempty"`
	Endpoint     string       `json:"endpoint"`
	Skills       []AgentSkill `json:"skills,omitempty"`
}

// AgentSkill describes a specific skill or capability of an agent.
type AgentSkill struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Task represents an A2A task with its lifecycle state and results.
type Task struct {
	ID       string         `json:"id"`
	Status   TaskStatus     `json:"status"`
	Input    string         `json:"input,omitempty"`
	Output   string         `json:"output,omitempty"`
	Error    string         `json:"error,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// TaskRequest is the payload for creating a new A2A task.
type TaskRequest struct {
	Input    string         `json:"input"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// TaskResponse wraps a Task in an API response.
type TaskResponse struct {
	Task Task `json:"task"`
}

// ErrorResponse wraps an error message in an API response.
type ErrorResponse struct {
	Error string `json:"error"`
}
