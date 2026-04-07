package runtime

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// teamMockAgent is a simple agent mock for testing.
type teamMockAgent struct {
	id       string
	persona  agent.Persona
	tools    []tool.Tool
	children []agent.Agent
	// invokeFn allows tests to control Invoke behavior.
	invokeFn func(ctx context.Context, input string) (string, error)
	// streamFn allows tests to control Stream behavior.
	streamFn func(ctx context.Context, input string) iter.Seq2[agent.Event, error]
}

var _ agent.Agent = (*teamMockAgent)(nil)

func (m *teamMockAgent) ID() string              { return m.id }
func (m *teamMockAgent) Persona() agent.Persona  { return m.persona }
func (m *teamMockAgent) Tools() []tool.Tool      { return m.tools }
func (m *teamMockAgent) Children() []agent.Agent { return m.children }

func (m *teamMockAgent) Invoke(ctx context.Context, input string, _ ...agent.Option) (string, error) {
	if m.invokeFn != nil {
		return m.invokeFn(ctx, input)
	}
	// Default: echo input with agent ID prefix
	return fmt.Sprintf("[%s] %s", m.id, input), nil
}

func (m *teamMockAgent) Stream(ctx context.Context, input string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	if m.streamFn != nil {
		return m.streamFn(ctx, input)
	}
	// Default: emit a single text event with the invoke result
	return func(yield func(agent.Event, error) bool) {
		result, err := m.Invoke(ctx, input)
		if err != nil {
			yield(agent.Event{}, err)
			return
		}
		if !yield(agent.Event{
			Type:    agent.EventText,
			Text:    result,
			AgentID: m.id,
		}, nil) {
			return
		}
		yield(agent.Event{
			Type:    agent.EventDone,
			AgentID: m.id,
		}, nil)
	}
}

// newMockAgent creates a mock agent with sensible defaults.
func newMockAgent(id string) *teamMockAgent {
	return &teamMockAgent{
		id:      id,
		persona: agent.Persona{Role: id + " role"},
	}
}

// newTransformAgent creates a mock agent that transforms input with a function.
func newTransformAgent(id string, fn func(string) string) *teamMockAgent {
	m := newMockAgent(id)
	m.invokeFn = func(_ context.Context, input string) (string, error) {
		return fn(input), nil
	}
	m.streamFn = func(ctx context.Context, input string) iter.Seq2[agent.Event, error] {
		return func(yield func(agent.Event, error) bool) {
			result := fn(input)
			if !yield(agent.Event{
				Type:    agent.EventText,
				Text:    result,
				AgentID: id,
			}, nil) {
				return
			}
			yield(agent.Event{
				Type:    agent.EventDone,
				AgentID: id,
			}, nil)
		}
	}
	return m
}

func TestTeamImplementsAgent(t *testing.T) {
	// Compile-time check is in team.go, but verify at runtime too.
	var a agent.Agent = NewTeam(WithTeamID("test"))
	assert.Equal(t, "test", a.ID())
}

func TestNewTeamDefaults(t *testing.T) {
	team := NewTeam()
	assert.Equal(t, "team", team.ID())
	assert.True(t, team.Persona().IsEmpty())
	assert.Nil(t, team.Tools())
	assert.Nil(t, team.Children())
	// Default pattern is PipelinePattern
	assert.NotNil(t, team.pattern)
}

func TestNewTeamWithOptions(t *testing.T) {
	a1 := newMockAgent("agent-1")
	a2 := newMockAgent("agent-2")
	p := PipelinePattern()

	team := NewTeam(
		WithTeamID("my-team"),
		WithAgents(a1, a2),
		WithPattern(p),
		WithTeamPersona(agent.Persona{Role: "coordinator"}),
	)

	assert.Equal(t, "my-team", team.ID())
	assert.Equal(t, "coordinator", team.Persona().Role)
	assert.Len(t, team.Children(), 2)
	assert.Equal(t, "agent-1", team.Children()[0].ID())
	assert.Equal(t, "agent-2", team.Children()[1].ID())
}

func TestTeamInvokeNoAgents(t *testing.T) {
	team := NewTeam(WithTeamID("empty"))
	_, err := team.Invoke(context.Background(), "hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no agents configured")
}

func TestTeamStreamNoAgents(t *testing.T) {
	team := NewTeam(WithTeamID("empty"))
	for _, err := range team.Stream(context.Background(), "hello") {
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no agents configured")
		break
	}
}

func TestPipelinePatternSequentialExecution(t *testing.T) {
	// Create agents that transform input in sequence:
	// "hello" -> "HELLO" -> "HELLO!!!"
	upper := newTransformAgent("upper", strings.ToUpper)
	exclaim := newTransformAgent("exclaim", func(s string) string { return s + "!!!" })

	team := NewTeam(
		WithTeamID("pipeline-team"),
		WithAgents(upper, exclaim),
		WithPattern(PipelinePattern()),
	)

	result, err := team.Invoke(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, "HELLO!!!", result)
}

func TestPipelinePatternStreamEvents(t *testing.T) {
	a1 := newTransformAgent("first", func(s string) string { return "A:" + s })
	a2 := newTransformAgent("second", func(s string) string { return "B:" + s })

	pattern := PipelinePattern()
	agents := []agent.Agent{a1, a2}

	var events []agent.Event
	for event, err := range pattern.Execute(context.Background(), agents, "input") {
		require.NoError(t, err)
		events = append(events, event)
	}

	// We expect text+done from first agent, then text+done from second agent
	require.Len(t, events, 4)

	// First agent events
	assert.Equal(t, agent.EventText, events[0].Type)
	assert.Equal(t, "A:input", events[0].Text)
	assert.Equal(t, "first", events[0].AgentID)
	assert.Equal(t, 0, events[0].Metadata["pipeline_stage"])

	assert.Equal(t, agent.EventDone, events[1].Type)
	assert.Equal(t, "first", events[1].AgentID)

	// Second agent gets output of first agent as input
	assert.Equal(t, agent.EventText, events[2].Type)
	assert.Equal(t, "B:A:input", events[2].Text)
	assert.Equal(t, "second", events[2].AgentID)
	assert.Equal(t, 1, events[2].Metadata["pipeline_stage"])

	assert.Equal(t, agent.EventDone, events[3].Type)
	assert.Equal(t, "second", events[3].AgentID)
}

func TestPipelinePatternSingleAgent(t *testing.T) {
	echo := newTransformAgent("echo", func(s string) string { return "echo:" + s })

	team := NewTeam(
		WithTeamID("single"),
		WithAgents(echo),
	)

	result, err := team.Invoke(context.Background(), "test")
	require.NoError(t, err)
	assert.Equal(t, "echo:test", result)
}

func TestPipelinePatternContextCancellation(t *testing.T) {
	// Create a slow agent that checks context
	slow := &teamMockAgent{
		id: "slow",
		streamFn: func(ctx context.Context, input string) iter.Seq2[agent.Event, error] {
			return func(yield func(agent.Event, error) bool) {
				select {
				case <-ctx.Done():
					yield(agent.Event{}, ctx.Err())
					return
				default:
					yield(agent.Event{Type: agent.EventText, Text: "slow-output"}, nil)
				}
			}
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	team := NewTeam(
		WithTeamID("cancel-test"),
		WithAgents(slow),
	)

	_, err := team.Invoke(ctx, "test")
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestPipelinePatternAgentError(t *testing.T) {
	failing := &teamMockAgent{
		id: "failing",
		invokeFn: func(_ context.Context, _ string) (string, error) {
			return "", fmt.Errorf("agent failed")
		},
		streamFn: func(_ context.Context, _ string) iter.Seq2[agent.Event, error] {
			return func(yield func(agent.Event, error) bool) {
				yield(agent.Event{}, fmt.Errorf("agent failed"))
			}
		},
	}
	good := newMockAgent("good")

	team := NewTeam(
		WithTeamID("error-test"),
		WithAgents(failing, good),
	)

	_, err := team.Invoke(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline stage 0")
	assert.Contains(t, err.Error(), "agent failed")
}

func TestRecursiveTeamComposition(t *testing.T) {
	// Inner team: upper -> exclaim
	upper := newTransformAgent("upper", strings.ToUpper)
	exclaim := newTransformAgent("exclaim", func(s string) string { return s + "!" })

	innerTeam := NewTeam(
		WithTeamID("inner"),
		WithAgents(upper, exclaim),
	)

	// Outer team: inner-team -> wrap
	wrap := newTransformAgent("wrap", func(s string) string { return "[" + s + "]" })

	outerTeam := NewTeam(
		WithTeamID("outer"),
		WithAgents(innerTeam, wrap),
	)

	result, err := outerTeam.Invoke(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, "[HELLO!]", result)
}

func TestSupervisorPattern(t *testing.T) {
	coordinator := newTransformAgent("coordinator", func(s string) string {
		return "coordinated: " + s
	})

	a1 := newMockAgent("worker-1")
	a2 := newMockAgent("worker-2")

	team := NewTeam(
		WithTeamID("supervisor-team"),
		WithAgents(a1, a2),
		WithPattern(SupervisorPattern(coordinator)),
	)

	result, err := team.Invoke(context.Background(), "do work")
	require.NoError(t, err)
	assert.Contains(t, result, "coordinated:")
	assert.Contains(t, result, "worker-1")
	assert.Contains(t, result, "worker-2")
}

func TestScatterGatherPattern(t *testing.T) {
	a1 := newTransformAgent("analyzer-1", func(s string) string { return "analysis-1:" + s })
	a2 := newTransformAgent("analyzer-2", func(s string) string { return "analysis-2:" + s })

	aggregator := newTransformAgent("aggregator", func(s string) string {
		return "aggregated: " + s
	})

	team := NewTeam(
		WithTeamID("scatter-team"),
		WithAgents(a1, a2),
		WithPattern(ScatterGatherPattern(aggregator)),
	)

	result, err := team.Invoke(context.Background(), "data")
	require.NoError(t, err)
	assert.Contains(t, result, "aggregated:")
	assert.Contains(t, result, "analysis-1:data")
	assert.Contains(t, result, "analysis-2:data")
}

func TestScatterGatherPatternAgentError(t *testing.T) {
	good := newMockAgent("good")
	failing := &teamMockAgent{
		id: "failing",
		invokeFn: func(_ context.Context, _ string) (string, error) {
			return "", fmt.Errorf("worker failed")
		},
		streamFn: func(_ context.Context, _ string) iter.Seq2[agent.Event, error] {
			return func(yield func(agent.Event, error) bool) {
				yield(agent.Event{}, fmt.Errorf("worker failed"))
			}
		},
	}

	aggregator := newMockAgent("aggregator")

	team := NewTeam(
		WithTeamID("scatter-err"),
		WithAgents(good, failing),
		WithPattern(ScatterGatherPattern(aggregator)),
	)

	_, err := team.Invoke(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scatter agent failing")
	assert.Contains(t, err.Error(), "worker failed")
}

func TestTeamWithTools(t *testing.T) {
	team := NewTeam(
		WithTeamID("tooled"),
		WithTeamTools(&mockTool{name: "search"}),
	)
	require.Len(t, team.Tools(), 1)
	assert.Equal(t, "search", team.Tools()[0].Name())
}

func TestPipelinePatternConsumerStops(t *testing.T) {
	// Verify that if the consumer stops iteration, we respect it
	a1 := newTransformAgent("a1", func(s string) string { return "out1" })
	a2 := newTransformAgent("a2", func(s string) string { return "out2" })

	pattern := PipelinePattern()
	count := 0
	for _, err := range pattern.Execute(context.Background(), []agent.Agent{a1, a2}, "in") {
		require.NoError(t, err)
		count++
		if count == 1 {
			break // Stop after first event
		}
	}
	assert.Equal(t, 1, count)
}

func TestScatterGatherContextCancellation(t *testing.T) {
	slow := &teamMockAgent{
		id: "slow",
		invokeFn: func(ctx context.Context, _ string) (string, error) {
			<-ctx.Done()
			return "", ctx.Err()
		},
	}

	aggregator := newMockAgent("agg")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	team := NewTeam(
		WithTeamID("sg-cancel"),
		WithAgents(slow),
		WithPattern(ScatterGatherPattern(aggregator)),
	)

	_, err := team.Invoke(ctx, "test")
	require.Error(t, err)
}

// mockTool is a minimal tool mock for testing.
type mockTool struct {
	name string
}

func (m *mockTool) Name() string                { return m.name }
func (m *mockTool) Description() string         { return "mock tool" }
func (m *mockTool) InputSchema() map[string]any { return nil }
func (m *mockTool) Execute(_ context.Context, _ map[string]any) (*tool.Result, error) {
	return tool.TextResult("ok"), nil
}
