package simulation

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/llm"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

const userPromptTemplate = `You are simulating a user with the following persona and goals.

Persona: %s
Goal: %s

Current conversation history:
%s

The agent's last response was:
%s

Respond as the user would. If the goal has been achieved, respond with exactly: [GOAL_COMPLETE]
If the goal cannot be achieved, respond with exactly: [GOAL_FAILED]
Otherwise, provide the next user message to continue toward the goal.`

// userOptions holds configuration for a SimulatedUser.
type userOptions struct {
	model   llm.ChatModel
	persona string
	goal    string
}

// UserOption configures a SimulatedUser.
type UserOption func(*userOptions)

// WithUserModel sets the LLM that drives the simulated user.
func WithUserModel(m llm.ChatModel) UserOption {
	return func(o *userOptions) {
		o.model = m
	}
}

// WithPersona sets the persona description for the simulated user.
func WithPersona(persona string) UserOption {
	return func(o *userOptions) {
		o.persona = persona
	}
}

// WithGoal sets the goal the simulated user is trying to achieve.
func WithGoal(goal string) UserOption {
	return func(o *userOptions) {
		o.goal = goal
	}
}

// UserResponse holds the simulated user's response and status.
type UserResponse struct {
	// Message is the user's next message.
	Message string

	// GoalComplete indicates the user considers the goal achieved.
	GoalComplete bool

	// GoalFailed indicates the user considers the goal unachievable.
	GoalFailed bool
}

// SimulatedUser is an LLM-driven user persona that generates realistic
// user messages based on a persona description and goal.
type SimulatedUser struct {
	opts    userOptions
	history []string
}

// NewSimulatedUser creates a new SimulatedUser with the given options.
func NewSimulatedUser(opts ...UserOption) (*SimulatedUser, error) {
	o := userOptions{
		persona: "A helpful and cooperative user",
	}
	for _, opt := range opts {
		opt(&o)
	}
	if o.model == nil {
		return nil, core.NewError("simulation.user.new", core.ErrInvalidInput, "model is required", nil)
	}
	if o.goal == "" {
		return nil, core.NewError("simulation.user.new", core.ErrInvalidInput, "goal is required", nil)
	}
	return &SimulatedUser{opts: o}, nil
}

// Respond generates the next user message given the agent's last response.
func (u *SimulatedUser) Respond(ctx context.Context, agentResponse string) (*UserResponse, error) {
	historyText := "(no prior messages)"
	if len(u.history) > 0 {
		historyText = ""
		for _, h := range u.history {
			historyText += h + "\n"
		}
	}

	prompt := fmt.Sprintf(userPromptTemplate,
		u.opts.persona,
		u.opts.goal,
		historyText,
		agentResponse,
	)

	resp, err := u.opts.model.Generate(ctx, []schema.Message{
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "simulated user: llm generate: %w", err)
	}

	text := resp.Text()
	u.history = append(u.history, "Agent: "+agentResponse, "User: "+text)

	switch text {
	case "[GOAL_COMPLETE]":
		return &UserResponse{Message: text, GoalComplete: true}, nil
	case "[GOAL_FAILED]":
		return &UserResponse{Message: text, GoalFailed: true}, nil
	default:
		return &UserResponse{Message: text}, nil
	}
}

// Reset clears the conversation history.
func (u *SimulatedUser) Reset() {
	u.history = nil
}
