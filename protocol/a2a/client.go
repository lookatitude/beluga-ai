package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"iter"
	"net/http"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/tool"
)

const (
	opGetCard    = "a2a/get_card: "
	opCreateTask = "a2a/create_task: "
	opGetTask    = "a2a/get_task: "
	opCancelTask = "a2a/cancel_task: "
	opInvoke     = "a2a/invoke: "

	unexpectedStatusFmt = "unexpected status %d"
)

// A2AClient connects to a remote A2A agent over HTTP.
type A2AClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new A2A client pointing at the given base URL.
func NewClient(baseURL string) *A2AClient {
	return &A2AClient{
		baseURL:    baseURL,
		httpClient: http.DefaultClient,
	}
}

// GetCard retrieves the Agent Card from the remote agent.
func (c *A2AClient) GetCard(ctx context.Context) (*AgentCard, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/.well-known/agent.json", nil)
	if err != nil {
		return nil, core.Errorf(core.ErrInvalidInput, opGetCard+"%w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, opGetCard+"%w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, core.Errorf(core.ErrProviderDown, opGetCard+unexpectedStatusFmt, resp.StatusCode)
	}

	var card AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, core.Errorf(core.ErrProviderDown, opGetCard+"%w", err)
	}
	return &card, nil
}

// CreateTask submits a new task to the remote agent.
func (c *A2AClient) CreateTask(ctx context.Context, req TaskRequest) (*Task, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, core.Errorf(core.ErrInvalidInput, opCreateTask+"%w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/tasks", bytes.NewReader(body))
	if err != nil {
		return nil, core.Errorf(core.ErrInvalidInput, opCreateTask+"%w", err)
	}
	httpReq.Header.Set(contentTypeHeader, contentTypeJSON)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, opCreateTask+"%w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, core.Errorf(core.ErrProviderDown, opCreateTask+"%s", errResp.Error)
	}

	var taskResp TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&taskResp); err != nil {
		return nil, core.Errorf(core.ErrProviderDown, opCreateTask+"%w", err)
	}
	return &taskResp.Task, nil
}

// GetTask retrieves the current state of a task.
func (c *A2AClient) GetTask(ctx context.Context, taskID string) (*Task, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/tasks/"+taskID, nil)
	if err != nil {
		return nil, core.Errorf(core.ErrInvalidInput, opGetTask+"%w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, opGetTask+"%w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, core.Errorf(core.ErrNotFound, opGetTask+"task not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, core.Errorf(core.ErrProviderDown, opGetTask+unexpectedStatusFmt, resp.StatusCode)
	}

	var taskResp TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&taskResp); err != nil {
		return nil, core.Errorf(core.ErrProviderDown, opGetTask+"%w", err)
	}
	return &taskResp.Task, nil
}

// CancelTask requests cancellation of a running task.
func (c *A2AClient) CancelTask(ctx context.Context, taskID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/tasks/"+taskID+"/cancel", nil)
	if err != nil {
		return core.Errorf(core.ErrInvalidInput, opCancelTask+"%w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return core.Errorf(core.ErrProviderDown, opCancelTask+"%w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return core.Errorf(core.ErrNotFound, opCancelTask+"task not found")
	}
	if resp.StatusCode != http.StatusOK {
		return core.Errorf(core.ErrProviderDown, opCancelTask+unexpectedStatusFmt, resp.StatusCode)
	}

	return nil
}

// NewRemoteAgent wraps an A2A endpoint as a local agent.Agent.
// It fetches the Agent Card to populate the agent's identity.
func NewRemoteAgent(baseURL string) (agent.Agent, error) {
	client := NewClient(baseURL)
	ctx := context.Background()

	card, err := client.GetCard(ctx)
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "a2a/remote_agent: %w", err)
	}

	return &remoteAgent{
		client: client,
		card:   *card,
	}, nil
}

// remoteAgent implements agent.Agent by delegating to a remote A2A server.
type remoteAgent struct {
	client *A2AClient
	card   AgentCard
}

func (a *remoteAgent) ID() string { return a.card.Name }

func (a *remoteAgent) Persona() agent.Persona {
	return agent.Persona{
		Role: a.card.Name,
		Goal: a.card.Description,
	}
}

func (a *remoteAgent) Tools() []tool.Tool      { return nil }
func (a *remoteAgent) Children() []agent.Agent { return nil }

func (a *remoteAgent) Invoke(ctx context.Context, input string, _ ...agent.Option) (string, error) {
	task, err := a.client.CreateTask(ctx, TaskRequest{Input: input})
	if err != nil {
		return "", core.Errorf(core.ErrProviderDown, opInvoke+"%w", err)
	}

	// Poll until terminal state with exponential backoff.
	delay := 100 * time.Millisecond
	const maxDelay = 5 * time.Second

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(delay):
		}

		task, err = a.client.GetTask(ctx, task.ID)
		if err != nil {
			return "", core.Errorf(core.ErrProviderDown, opInvoke+"%w", err)
		}

		switch task.Status {
		case StatusCompleted:
			return task.Output, nil
		case StatusFailed:
			return "", core.Errorf(core.ErrProviderDown, opInvoke+"task failed: %s", task.Error)
		case StatusCanceled:
			return "", core.Errorf(core.ErrTimeout, opInvoke+"task canceled")
		}

		// Exponential backoff, capped at maxDelay.
		delay = delay * 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
}

func (a *remoteAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		result, err := a.Invoke(ctx, input, opts...)
		if err != nil {
			yield(agent.Event{}, err)
			return
		}
		yield(agent.Event{
			Type:    agent.EventText,
			Text:    result,
			AgentID: a.ID(),
		}, nil)
		yield(agent.Event{
			Type:    agent.EventDone,
			AgentID: a.ID(),
		}, nil)
	}
}
