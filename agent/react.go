package agent

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

func init() {
	RegisterPlanner("react", func(cfg PlannerConfig) (Planner, error) {
		if cfg.LLM == nil {
			return nil, fmt.Errorf("react planner requires an LLM")
		}
		return NewReActPlanner(cfg.LLM), nil
	})
}

// ReActPlanner implements the ReAct (Reasoning + Acting) strategy.
// It sends the conversation to the LLM and interprets the response:
// - If the LLM returns tool calls → ActionTool
// - If the LLM returns text with no tool calls → ActionFinish
type ReActPlanner struct {
	llm llm.ChatModel
}

// NewReActPlanner creates a new ReAct planner with the given LLM.
func NewReActPlanner(model llm.ChatModel) *ReActPlanner {
	return &ReActPlanner{llm: model}
}

// Plan sends the initial messages to the LLM and returns actions based on
// the response.
func (p *ReActPlanner) Plan(ctx context.Context, state PlannerState) ([]Action, error) {
	return p.generate(ctx, state)
}

// Replan adds observation results to the message history and re-queries
// the LLM.
func (p *ReActPlanner) Replan(ctx context.Context, state PlannerState) ([]Action, error) {
	return p.generate(ctx, state)
}

// generate calls the LLM with the current state and converts the response
// into actions.
func (p *ReActPlanner) generate(ctx context.Context, state PlannerState) ([]Action, error) {
	// Build messages with observations
	messages := buildMessagesFromState(state)

	// Bind tools to LLM
	model := p.llm
	if len(state.Tools) > 0 {
		defs := toolDefinitions(state.Tools)
		model = model.BindTools(defs)
	}

	// Generate response
	resp, err := model.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("react planner: generate failed: %w", err)
	}

	return parseAIResponse(resp), nil
}

// parseAIResponse converts an AI message into planner actions.
func parseAIResponse(resp *schema.AIMessage) []Action {
	var actions []Action

	// If there are tool calls, emit tool actions
	if len(resp.ToolCalls) > 0 {
		for i := range resp.ToolCalls {
			tc := resp.ToolCalls[i]
			actions = append(actions, Action{
				Type:     ActionTool,
				ToolCall: &tc,
			})
		}
		return actions
	}

	// Otherwise, treat as final answer
	text := resp.Text()
	actions = append(actions, Action{
		Type:    ActionFinish,
		Message: text,
	})
	return actions
}

// buildMessagesFromState constructs the full message list from planner state,
// including tool call/result pairs from observations.
func buildMessagesFromState(state PlannerState) []schema.Message {
	msgs := make([]schema.Message, 0, len(state.Messages)+len(state.Observations)*2)
	msgs = append(msgs, state.Messages...)

	for _, obs := range state.Observations {
		if obs.Action.Type == ActionTool && obs.Action.ToolCall != nil {
			// Add AI message with tool call
			msgs = append(msgs, &schema.AIMessage{
				ToolCalls: []schema.ToolCall{*obs.Action.ToolCall},
			})

			// Add tool result
			var text string
			if obs.Result != nil {
				text = resultText(obs.Result)
			}
			if obs.Error != nil && text == "" {
				text = obs.Error.Error()
			}
			msgs = append(msgs, schema.NewToolMessage(obs.Action.ToolCall.ID, text))
		}
	}

	return msgs
}

// toolDefinitions converts tools to schema.ToolDefinition.
func toolDefinitions(tools []tool.Tool) []schema.ToolDefinition {
	defs := make([]schema.ToolDefinition, len(tools))
	for i, t := range tools {
		defs[i] = tool.ToDefinition(t)
	}
	return defs
}

// resultText extracts text from a tool result.
func resultText(r *tool.Result) string {
	for _, part := range r.Content {
		if tp, ok := part.(schema.TextPart); ok {
			return tp.Text
		}
	}
	return ""
}
