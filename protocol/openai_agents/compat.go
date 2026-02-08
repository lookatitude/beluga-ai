// Package openai_agents provides a compatibility layer between Beluga AI agents
// and the OpenAI Agents SDK format. It converts Beluga agents, tools, and
// handoffs into the OpenAI Agents SDK wire format for interoperability.
//
// This allows Beluga agents to be exposed via an API compatible with the OpenAI
// Agents SDK, enabling clients built for that SDK to interact with Beluga agents.
//
// Usage:
//
//	agentDef := openai_agents.FromAgent(belugaAgent)
//	jsonBytes, _ := json.Marshal(agentDef)
package openai_agents

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// AgentDef represents an agent definition in the OpenAI Agents SDK format.
type AgentDef struct {
	Name         string     `json:"name"`
	Instructions string     `json:"instructions,omitempty"`
	Model        string     `json:"model,omitempty"`
	Tools        []ToolDef  `json:"tools,omitempty"`
	Handoffs     []Handoff  `json:"handoffs,omitempty"`
}

// ToolDef represents a tool definition in the OpenAI Agents SDK format.
type ToolDef struct {
	Type     string         `json:"type"`
	Function FunctionDef    `json:"function,omitempty"`
}

// FunctionDef represents a function tool definition.
type FunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// Handoff represents an agent handoff in the OpenAI Agents SDK format.
type Handoff struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// RunRequest represents a request to run an agent.
type RunRequest struct {
	AgentName string `json:"agent_name"`
	Input     string `json:"input"`
	Stream    bool   `json:"stream,omitempty"`
}

// RunResponse represents the response from running an agent.
type RunResponse struct {
	Output    string           `json:"output"`
	Agent     string           `json:"agent"`
	ToolCalls []ToolCallResult `json:"tool_calls,omitempty"`
}

// ToolCallResult represents a tool call result in the response.
type ToolCallResult struct {
	Name   string `json:"name"`
	Input  string `json:"input"`
	Output string `json:"output"`
}

// FromAgent converts a Beluga Agent into an OpenAI Agents SDK AgentDef.
func FromAgent(a agent.Agent) AgentDef {
	def := AgentDef{
		Name:         a.ID(),
		Instructions: a.Persona().Goal,
	}

	for _, t := range a.Tools() {
		def.Tools = append(def.Tools, ToolDef{
			Type: "function",
			Function: FunctionDef{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.InputSchema(),
			},
		})
	}

	for _, child := range a.Children() {
		def.Handoffs = append(def.Handoffs, Handoff{
			Name:        child.ID(),
			Description: child.Persona().Goal,
		})
	}

	return def
}

// FromTools converts a slice of Beluga tools into OpenAI Agents SDK ToolDefs.
func FromTools(tools []tool.Tool) []ToolDef {
	defs := make([]ToolDef, len(tools))
	for i, t := range tools {
		defs[i] = ToolDef{
			Type: "function",
			Function: FunctionDef{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.InputSchema(),
			},
		}
	}
	return defs
}

// Runner executes agents using the OpenAI Agents SDK format.
type Runner struct {
	agents map[string]agent.Agent
}

// NewRunner creates a Runner with the given agents.
func NewRunner(agents ...agent.Agent) *Runner {
	r := &Runner{
		agents: make(map[string]agent.Agent),
	}
	for _, a := range agents {
		r.agents[a.ID()] = a
	}
	return r
}

// Run executes the named agent with the given input and returns a response
// in the OpenAI Agents SDK format.
func (r *Runner) Run(ctx context.Context, req RunRequest) (RunResponse, error) {
	a, ok := r.agents[req.AgentName]
	if !ok {
		return RunResponse{}, fmt.Errorf("openai_agents: unknown agent %q", req.AgentName)
	}

	output, err := a.Invoke(ctx, req.Input)
	if err != nil {
		return RunResponse{}, fmt.Errorf("openai_agents: invoke %q: %w", req.AgentName, err)
	}

	return RunResponse{
		Output: output,
		Agent:  req.AgentName,
	}, nil
}

// ListAgents returns the AgentDefs for all registered agents.
func (r *Runner) ListAgents() []AgentDef {
	result := make([]AgentDef, 0, len(r.agents))
	for _, a := range r.agents {
		result = append(result, FromAgent(a))
	}
	return result
}

// MarshalAgentDef marshals an AgentDef to JSON bytes.
func MarshalAgentDef(def AgentDef) ([]byte, error) {
	return json.Marshal(def)
}

// UnmarshalRunRequest unmarshals a RunRequest from JSON bytes.
func UnmarshalRunRequest(data []byte) (RunRequest, error) {
	var req RunRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return RunRequest{}, fmt.Errorf("openai_agents: unmarshal request: %w", err)
	}
	return req, nil
}
