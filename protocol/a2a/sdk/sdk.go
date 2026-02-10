package sdk

import (
	"context"
	"fmt"
	"iter"
	"net/http"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2aclient"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/a2aproject/a2a-go/a2asrv/eventqueue"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// ServerConfig holds configuration for creating an A2A SDK server.
type ServerConfig struct {
	// Name is the agent's display name in the AgentCard.
	Name string
	// Version is the agent version.
	Version string
	// Description is a human-readable description.
	Description string
	// URL is the endpoint URL for the agent.
	URL string
}

// NewServer creates an A2A request handler and agent card from a Beluga agent.
// The returned handler should be mounted on an HTTP server, and the card
// served at /.well-known/agent-card.json.
func NewServer(a agent.Agent, cfg ServerConfig) (http.Handler, *a2a.AgentCard) {
	executor := &belugaExecutor{agent: a}

	card := &a2a.AgentCard{
		Name:               cfg.Name,
		Description:        cfg.Description,
		Version:            cfg.Version,
		URL:                cfg.URL,
		ProtocolVersion:    "0.2.2",
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
		Skills:             buildSkills(a),
	}

	handler := a2asrv.NewHandler(executor)
	jsonrpcHandler := a2asrv.NewJSONRPCHandler(handler)

	mux := http.NewServeMux()
	mux.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(card))
	mux.Handle("/", jsonrpcHandler)

	return mux, card
}

// buildSkills creates A2A skills from the agent's tools.
func buildSkills(a agent.Agent) []a2a.AgentSkill {
	tools := a.Tools()
	skills := make([]a2a.AgentSkill, 0, len(tools)+1)

	// Add a default skill for the agent itself.
	persona := a.Persona()
	skills = append(skills, a2a.AgentSkill{
		ID:          a.ID(),
		Name:        persona.Role,
		Description: persona.Goal,
		Tags:        []string{"agent"},
	})

	// Add skills for each tool.
	for _, t := range tools {
		skills = append(skills, a2a.AgentSkill{
			ID:          t.Name(),
			Name:        t.Name(),
			Description: t.Description(),
			Tags:        []string{"tool"},
		})
	}

	return skills
}

// belugaExecutor implements a2asrv.AgentExecutor by delegating to a Beluga agent.
type belugaExecutor struct {
	agent agent.Agent
}

func (e *belugaExecutor) Execute(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error {
	// Extract input text from the request message.
	input := extractInput(reqCtx.Message)

	// Transition to working state.
	workingEvent := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateWorking, nil)
	if err := queue.Write(ctx, workingEvent); err != nil {
		return fmt.Errorf("a2a/sdk: write working event: %w", err)
	}

	// Invoke the Beluga agent.
	result, err := e.agent.Invoke(ctx, input)
	if err != nil {
		// Report failure.
		failMsg := a2a.NewMessage(a2a.MessageRoleAgent, a2a.TextPart{Text: err.Error()})
		failEvent := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateFailed, failMsg)
		failEvent.Final = true
		if writeErr := queue.Write(ctx, failEvent); writeErr != nil {
			return fmt.Errorf("a2a/sdk: write failed event: %w", writeErr)
		}
		return nil
	}

	// Report completion.
	doneMsg := a2a.NewMessage(a2a.MessageRoleAgent, a2a.TextPart{Text: result})
	doneEvent := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateCompleted, doneMsg)
	doneEvent.Final = true
	if err := queue.Write(ctx, doneEvent); err != nil {
		return fmt.Errorf("a2a/sdk: write completed event: %w", err)
	}

	return nil
}

func (e *belugaExecutor) Cancel(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error {
	cancelEvent := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateCanceled, nil)
	cancelEvent.Final = true
	if err := queue.Write(ctx, cancelEvent); err != nil {
		return fmt.Errorf("a2a/sdk: write cancel event: %w", err)
	}
	return nil
}

// extractInput gets the text content from an A2A message.
func extractInput(msg *a2a.Message) string {
	if msg == nil {
		return ""
	}
	for _, part := range msg.Parts {
		if tp, ok := part.(a2a.TextPart); ok {
			return tp.Text
		}
		// Also check pointer type.
		if tp, ok := part.(*a2a.TextPart); ok {
			return tp.Text
		}
	}
	return ""
}

// NewRemoteAgent creates a Beluga agent.Agent that delegates to a remote
// A2A agent via the official SDK client. It fetches the AgentCard from the
// remote server to populate the agent's identity.
func NewRemoteAgent(ctx context.Context, url string) (agent.Agent, error) {
	endpoints := []a2a.AgentInterface{
		{URL: url},
	}

	client, err := a2aclient.NewFromEndpoints(ctx, endpoints)
	if err != nil {
		return nil, fmt.Errorf("a2a/sdk: create client: %w", err)
	}

	card, err := client.GetAgentCard(ctx)
	if err != nil {
		client.Destroy()
		return nil, fmt.Errorf("a2a/sdk: get agent card: %w", err)
	}

	return &remoteAgent{
		client: client,
		card:   card,
	}, nil
}

// remoteAgent implements agent.Agent by delegating to a remote A2A server
// via the official SDK client.
type remoteAgent struct {
	client *a2aclient.Client
	card   *a2a.AgentCard
}

func (a *remoteAgent) ID() string { return a.card.Name }

func (a *remoteAgent) Persona() agent.Persona {
	return agent.Persona{
		Role: a.card.Name,
		Goal: a.card.Description,
	}
}

func (a *remoteAgent) Tools() []tool.Tool    { return nil }
func (a *remoteAgent) Children() []agent.Agent { return nil }

func (a *remoteAgent) Invoke(ctx context.Context, input string, _ ...agent.Option) (string, error) {
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: input})
	params := &a2a.MessageSendParams{Message: msg}

	result, err := a.client.SendMessage(ctx, params)
	if err != nil {
		return "", fmt.Errorf("a2a/sdk/invoke: %w", err)
	}

	return extractResultText(result), nil
}

func (a *remoteAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		result, err := a.Invoke(ctx, input, opts...)
		if err != nil {
			yield(agent.Event{}, err)
			return
		}
		if !yield(agent.Event{
			Type:    agent.EventText,
			Text:    result,
			AgentID: a.ID(),
		}, nil) {
			return
		}
		yield(agent.Event{
			Type:    agent.EventDone,
			AgentID: a.ID(),
		}, nil)
	}
}

// extractResultText extracts text from an A2A SendMessageResult.
func extractResultText(result a2a.SendMessageResult) string {
	if result == nil {
		return ""
	}

	// The result can be a Task or a Message.
	switch r := result.(type) {
	case *a2a.Task:
		if r.Status.Message != nil {
			return extractInput(r.Status.Message)
		}
		// Check artifacts.
		for _, artifact := range r.Artifacts {
			for _, part := range artifact.Parts {
				if tp, ok := part.(a2a.TextPart); ok {
					return tp.Text
				}
				if tp, ok := part.(*a2a.TextPart); ok {
					return tp.Text
				}
			}
		}
	case *a2a.Message:
		return extractInput(r)
	}

	return ""
}

// Compile-time interface checks.
var (
	_ agent.Agent         = (*remoteAgent)(nil)
	_ a2asrv.AgentExecutor = (*belugaExecutor)(nil)
)
