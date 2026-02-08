package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// Handoff describes a transfer from one agent to another. Handoffs are
// automatically converted to tools that appear in the LLM's tool list,
// enabling the model to decide when to hand off to a specialist.
type Handoff struct {
	// TargetAgent is the agent to hand off to.
	TargetAgent Agent
	// Description explains when to use this handoff.
	Description string
	// InputFilter optionally transforms the input before handing off.
	InputFilter func(HandoffInput) HandoffInput
	// OnHandoff is called when the handoff is triggered.
	OnHandoff func(ctx context.Context) error
	// IsEnabled dynamically controls whether this handoff is available.
	IsEnabled func(ctx context.Context) bool
}

// HandoffInput carries the input data for a handoff.
type HandoffInput struct {
	// Message is the text being handed off.
	Message string
	// Context carries additional context from the source agent.
	Context map[string]any
}

// HandoffTo creates a simple handoff to the target agent.
func HandoffTo(target Agent, description string) Handoff {
	return Handoff{
		TargetAgent: target,
		Description: description,
	}
}

// HandoffsToTools converts handoffs into tool.Tool instances that the LLM
// can call. Each handoff generates a transfer_to_{id} tool.
func HandoffsToTools(handoffs []Handoff) []tool.Tool {
	tools := make([]tool.Tool, 0, len(handoffs))
	for _, h := range handoffs {
		tools = append(tools, newHandoffTool(h))
	}
	return tools
}

// handoffTool wraps a Handoff as a tool.Tool.
type handoffTool struct {
	handoff Handoff
	name    string
	desc    string
}

func newHandoffTool(h Handoff) *handoffTool {
	name := fmt.Sprintf("transfer_to_%s", h.TargetAgent.ID())
	desc := h.Description
	if desc == "" {
		desc = fmt.Sprintf("Transfer the conversation to %s.", h.TargetAgent.ID())
	}
	return &handoffTool{
		handoff: h,
		name:    name,
		desc:    desc,
	}
}

func (t *handoffTool) Name() string        { return t.name }
func (t *handoffTool) Description() string  { return t.desc }
func (t *handoffTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{
				"type":        "string",
				"description": "Message or context to pass to the target agent",
			},
		},
	}
}

func (t *handoffTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
	// Check if handoff is enabled
	if t.handoff.IsEnabled != nil && !t.handoff.IsEnabled(ctx) {
		return tool.ErrorResult(fmt.Errorf("handoff to %s is currently disabled", t.handoff.TargetAgent.ID())), nil
	}

	// Fire OnHandoff callback
	if t.handoff.OnHandoff != nil {
		if err := t.handoff.OnHandoff(ctx); err != nil {
			return nil, err
		}
	}

	// Extract the message
	msg, _ := input["message"].(string)

	// Apply input filter
	hi := HandoffInput{Message: msg}
	if ctx.Value(handoffContextKey{}) != nil {
		if m, ok := ctx.Value(handoffContextKey{}).(map[string]any); ok {
			hi.Context = m
		}
	}
	if t.handoff.InputFilter != nil {
		hi = t.handoff.InputFilter(hi)
	}

	// Invoke the target agent
	result, err := t.handoff.TargetAgent.Invoke(ctx, hi.Message)
	if err != nil {
		return nil, fmt.Errorf("handoff to %s failed: %w", t.handoff.TargetAgent.ID(), err)
	}

	return tool.TextResult(result), nil
}

type handoffContextKey struct{}

// WithHandoffContext adds handoff context to the given context.
func WithHandoffContext(ctx context.Context, data map[string]any) context.Context {
	return context.WithValue(ctx, handoffContextKey{}, data)
}

// IsHandoffTool reports whether the given tool call is a handoff tool.
func IsHandoffTool(call schema.ToolCall) bool {
	return len(call.Name) > len("transfer_to_") && call.Name[:len("transfer_to_")] == "transfer_to_"
}

// HandoffTargetID extracts the target agent ID from a handoff tool call name.
func HandoffTargetID(call schema.ToolCall) string {
	if !IsHandoffTool(call) {
		return ""
	}
	return call.Name[len("transfer_to_"):]
}

// handoffToolInput is the schema for handoff tool arguments.
type handoffToolInput struct {
	Message string `json:"message"`
}

// ParseHandoffInput parses the JSON arguments of a handoff tool call.
func ParseHandoffInput(args string) (string, error) {
	var input handoffToolInput
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return "", fmt.Errorf("invalid handoff input: %w", err)
	}
	return input.Message, nil
}
