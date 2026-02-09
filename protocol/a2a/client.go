package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
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
		return nil, fmt.Errorf("a2a/get_card: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("a2a/get_card: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("a2a/get_card: unexpected status %d", resp.StatusCode)
	}

	var card AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, fmt.Errorf("a2a/get_card: %w", err)
	}
	return &card, nil
}

// CreateTask submits a new task to the remote agent.
func (c *A2AClient) CreateTask(ctx context.Context, req TaskRequest) (*Task, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("a2a/create_task: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/tasks", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("a2a/create_task: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("a2a/create_task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("a2a/create_task: %s", errResp.Error)
	}

	var taskResp TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&taskResp); err != nil {
		return nil, fmt.Errorf("a2a/create_task: %w", err)
	}
	return &taskResp.Task, nil
}

// GetTask retrieves the current state of a task.
func (c *A2AClient) GetTask(ctx context.Context, taskID string) (*Task, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/tasks/"+taskID, nil)
	if err != nil {
		return nil, fmt.Errorf("a2a/get_task: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("a2a/get_task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("a2a/get_task: task not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("a2a/get_task: unexpected status %d", resp.StatusCode)
	}

	var taskResp TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&taskResp); err != nil {
		return nil, fmt.Errorf("a2a/get_task: %w", err)
	}
	return &taskResp.Task, nil
}

// CancelTask requests cancellation of a running task.
func (c *A2AClient) CancelTask(ctx context.Context, taskID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/tasks/"+taskID+"/cancel", nil)
	if err != nil {
		return fmt.Errorf("a2a/cancel_task: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("a2a/cancel_task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("a2a/cancel_task: task not found")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("a2a/cancel_task: unexpected status %d", resp.StatusCode)
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
		return nil, fmt.Errorf("a2a/remote_agent: %w", err)
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

func (a *remoteAgent) Tools() []tool.Tool    { return nil }
func (a *remoteAgent) Children() []agent.Agent { return nil }

func (a *remoteAgent) Invoke(ctx context.Context, input string, _ ...agent.Option) (string, error) {
	task, err := a.client.CreateTask(ctx, TaskRequest{Input: input})
	if err != nil {
		return "", fmt.Errorf("a2a/invoke: %w", err)
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
			return "", fmt.Errorf("a2a/invoke: %w", err)
		}

		switch task.Status {
		case StatusCompleted:
			return task.Output, nil
		case StatusFailed:
			return "", fmt.Errorf("a2a/invoke: task failed: %s", task.Error)
		case StatusCanceled:
			return "", fmt.Errorf("a2a/invoke: task canceled")
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
