package runtime

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"sync"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/tool"
)

// Compile-time check that Team implements agent.Agent.
var _ agent.Agent = (*Team)(nil)

// Team is a group of agents coordinated by an OrchestrationPattern.
// Teams implement the agent.Agent interface, enabling recursive composition
// where a Team can contain other Teams as members.
type Team struct {
	id      string
	persona agent.Persona
	agents  []agent.Agent
	pattern OrchestrationPattern
	tools   []tool.Tool
}

// TeamOption is a functional option for configuring a Team.
type TeamOption func(*Team)

// NewTeam creates a new Team with the given options. If no ID is provided,
// a default ID of "team" is used. If no pattern is provided, PipelinePattern
// is used as the default.
func NewTeam(opts ...TeamOption) *Team {
	t := &Team{
		id: "team",
	}
	for _, opt := range opts {
		opt(t)
	}
	if t.pattern == nil {
		t.pattern = PipelinePattern()
	}
	return t
}

// WithAgents sets the member agents for the team.
func WithAgents(agents ...agent.Agent) TeamOption {
	return func(t *Team) {
		t.agents = agents
	}
}

// WithPattern sets the orchestration pattern that determines how agents
// are coordinated during execution.
func WithPattern(p OrchestrationPattern) TeamOption {
	return func(t *Team) {
		t.pattern = p
	}
}

// WithTeamID sets the unique identifier for the team.
func WithTeamID(id string) TeamOption {
	return func(t *Team) {
		t.id = id
	}
}

// WithTeamPersona sets the persona for the team.
func WithTeamPersona(p agent.Persona) TeamOption {
	return func(t *Team) {
		t.persona = p
	}
}

// WithTeamTools sets additional tools available to the team.
func WithTeamTools(tools ...tool.Tool) TeamOption {
	return func(t *Team) {
		t.tools = tools
	}
}

// ID returns the team's unique identifier.
func (t *Team) ID() string { return t.id }

// Persona returns the team's persona.
func (t *Team) Persona() agent.Persona { return t.persona }

// Tools returns tools available to the team.
func (t *Team) Tools() []tool.Tool { return t.tools }

// Children returns the member agents of the team.
func (t *Team) Children() []agent.Agent { return t.agents }

// Invoke executes the team synchronously by running the orchestration pattern
// and collecting all text events into a single result string.
func (t *Team) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	if len(t.agents) == 0 {
		return "", core.NewError("runtime.team.invoke", core.ErrInvalidInput,
			fmt.Sprintf("team %q: no agents configured", t.id), nil)
	}

	var lastErr error
	var lastStageText strings.Builder
	lastStage := -1

	for event, err := range t.Stream(ctx, input, opts...) {
		if err != nil {
			lastErr = err
			break
		}
		switch event.Type {
		case agent.EventText:
			// Track which pipeline stage we are in; reset the buffer
			// when a new stage begins so Invoke returns only the final
			// stage's output.
			stage, _ := event.Metadata["pipeline_stage"].(int)
			if stage != lastStage {
				lastStageText.Reset()
				lastStage = stage
			}
			lastStageText.WriteString(event.Text)
		case agent.EventError:
			lastErr = core.NewError("runtime.team.invoke", core.ErrToolFailed,
				fmt.Sprintf("team %q: agent error: %s", t.id, event.Text), nil)
		}
	}

	if lastErr != nil {
		return lastStageText.String(), lastErr
	}
	return lastStageText.String(), nil
}

// Stream executes the team and returns an iterator of events by delegating
// to the configured orchestration pattern.
func (t *Team) Stream(ctx context.Context, input string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	if len(t.agents) == 0 {
		return func(yield func(agent.Event, error) bool) {
			yield(agent.Event{}, core.NewError("runtime.team.stream", core.ErrInvalidInput,
				fmt.Sprintf("team %q: no agents configured", t.id), nil))
		}
	}
	return t.pattern.Execute(ctx, t.agents, input)
}

// pipelinePattern executes agents sequentially. The text output of each agent
// becomes the input for the next agent in the sequence.
type pipelinePattern struct{}

// PipelinePattern returns an OrchestrationPattern that executes agents
// sequentially, feeding the output of each agent as input to the next.
func PipelinePattern() OrchestrationPattern {
	return &pipelinePattern{}
}

// Execute runs agents in sequence, piping text output forward.
// For each stage, the agent is invoked to produce a result that feeds
// the next stage. Events are streamed to the caller for observability.
func (p *pipelinePattern) Execute(ctx context.Context, agents []agent.Agent, input string) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		currentInput := input

		for i, a := range agents {
			select {
			case <-ctx.Done():
				yield(agent.Event{}, ctx.Err())
				return
			default:
			}

			// Use Invoke to get the clean final output for piping to the
			// next stage. This correctly handles nested teams that may emit
			// events from multiple internal stages.
			result, err := a.Invoke(ctx, currentInput)
			if err != nil {
				if !yield(agent.Event{}, core.NewError("runtime.team.pipeline", core.ErrToolFailed,
					fmt.Sprintf("pipeline stage %d (%s) failed", i, a.ID()), err)) {
					return
				}
				return
			}

			// Emit the stage result as a text event
			evt := agent.Event{
				Type:    agent.EventText,
				Text:    result,
				AgentID: a.ID(),
				Metadata: map[string]any{
					"pipeline_stage": i,
				},
			}
			if !yield(evt, nil) {
				return
			}

			// Emit a done event for the stage
			if !yield(agent.Event{
				Type:    agent.EventDone,
				AgentID: a.ID(),
				Metadata: map[string]any{
					"pipeline_stage": i,
				},
			}, nil) {
				return
			}

			// Feed result to the next stage
			if result != "" {
				currentInput = result
			}
		}
	}
}

// supervisorPattern uses a coordinator agent to delegate work to other agents.
type supervisorPattern struct {
	coordinator agent.Agent
}

// SupervisorPattern returns an OrchestrationPattern where a coordinator agent
// delegates work to the other agents. The coordinator receives the original
// input along with a description of available agents.
func SupervisorPattern(coordinator agent.Agent) OrchestrationPattern {
	return &supervisorPattern{coordinator: coordinator}
}

// Execute runs the supervisor pattern: the coordinator processes the input
// and its output is yielded as events.
func (s *supervisorPattern) Execute(ctx context.Context, agents []agent.Agent, input string) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		select {
		case <-ctx.Done():
			yield(agent.Event{}, ctx.Err())
			return
		default:
		}

		// Build a description of available agents for the coordinator
		var desc strings.Builder
		desc.WriteString("Available agents:\n")
		for _, a := range agents {
			desc.WriteString(fmt.Sprintf("- %s: %s\n", a.ID(), a.Persona().Role))
		}
		desc.WriteString("\nTask: ")
		desc.WriteString(input)

		for event, err := range s.coordinator.Stream(ctx, desc.String()) {
			if err != nil {
				if !yield(agent.Event{}, core.NewError("runtime.team.supervisor", core.ErrToolFailed,
					"supervisor coordinator failed", err)) {
					return
				}
				return
			}
			event.AgentID = s.coordinator.ID()
			if !yield(event, nil) {
				return
			}
		}
	}
}

// scatterGatherPattern executes all agents in parallel and aggregates results.
type scatterGatherPattern struct {
	aggregator agent.Agent
}

// ScatterGatherPattern returns an OrchestrationPattern that executes all
// agents in parallel, collects their outputs, and passes them to an aggregator
// agent for final synthesis.
func ScatterGatherPattern(aggregator agent.Agent) OrchestrationPattern {
	return &scatterGatherPattern{aggregator: aggregator}
}

// Execute runs all agents in parallel, collects results, then aggregates.
func (sg *scatterGatherPattern) Execute(ctx context.Context, agents []agent.Agent, input string) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		select {
		case <-ctx.Done():
			yield(agent.Event{}, ctx.Err())
			return
		default:
		}

		type agentResult struct {
			agentID string
			output  string
			err     error
		}

		results := make([]agentResult, len(agents))
		var wg sync.WaitGroup

		for i, a := range agents {
			wg.Add(1)
			go func(idx int, ag agent.Agent) {
				defer wg.Done()
				out, err := ag.Invoke(ctx, input)
				results[idx] = agentResult{
					agentID: ag.ID(),
					output:  out,
					err:     err,
				}
			}(i, a)
		}

		wg.Wait()

		// Emit individual agent results and check for errors
		var combined strings.Builder
		combined.WriteString("Agent outputs:\n")
		for _, r := range results {
			if r.err != nil {
				if !yield(agent.Event{}, core.NewError("runtime.team.scatter", core.ErrToolFailed,
					fmt.Sprintf("scatter agent %s failed", r.agentID), r.err)) {
					return
				}
				return
			}
			combined.WriteString(fmt.Sprintf("\n--- %s ---\n%s\n", r.agentID, r.output))
		}

		// Pass combined results to the aggregator
		for event, err := range sg.aggregator.Stream(ctx, combined.String()) {
			if err != nil {
				if !yield(agent.Event{}, core.NewError("runtime.team.scatter", core.ErrToolFailed,
					"scatter aggregator failed", err)) {
					return
				}
				return
			}
			event.AgentID = sg.aggregator.ID()
			if !yield(event, nil) {
				return
			}
		}
	}
}
