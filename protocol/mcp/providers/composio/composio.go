// Package composio provides a Composio MCP integration for the Beluga AI
// protocol layer. It connects to the Composio API for tool discovery and
// execution, wrapping Composio tools as native tool.Tool instances.
//
// Composio provides access to hundreds of integrations and actions through
// its unified API, which can be consumed as MCP-compatible tools.
//
// Usage:
//
//	client, err := composio.New(
//	    composio.WithAPIKey("cmp-..."),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	tools, err := client.ListTools(ctx)
package composio

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/internal/httpclient"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// Client connects to the Composio API for tool discovery and execution.
type Client struct {
	client *httpclient.Client
}

// Option configures a Client.
type Option func(*config)

type config struct {
	baseURL string
	apiKey  string
	timeout time.Duration
}

// WithBaseURL sets the Composio API base URL.
func WithBaseURL(url string) Option {
	return func(c *config) { c.baseURL = url }
}

// WithAPIKey sets the Composio API key.
func WithAPIKey(key string) Option {
	return func(c *config) { c.apiKey = key }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// New creates a new Composio client.
func New(opts ...Option) (*Client, error) {
	cfg := &config{
		baseURL: "https://backend.composio.dev",
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.apiKey == "" {
		return nil, fmt.Errorf("composio: API key is required")
	}

	client := httpclient.New(
		httpclient.WithBaseURL(cfg.baseURL),
		httpclient.WithHeader("x-api-key", cfg.apiKey),
		httpclient.WithTimeout(cfg.timeout),
	)

	return &Client{client: client}, nil
}

// actionInfo describes a Composio action returned by the API.
type actionInfo struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"display_name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	AppName     string         `json:"appName"`
}

// actionsResponse is the Composio list actions API response.
type actionsResponse struct {
	Items []actionInfo `json:"items"`
}

// executeRequest is the Composio execute action API request.
type executeRequest struct {
	Input map[string]any `json:"input"`
}

// executeResponse is the Composio execute action API response.
type executeResponse struct {
	Data        any    `json:"data"`
	Error       string `json:"error,omitempty"`
	Successful  bool   `json:"successfull"`
}

// ListTools queries the Composio API for available actions and returns them
// as native tool.Tool instances.
func (c *Client) ListTools(ctx context.Context) ([]tool.Tool, error) {
	resp, err := httpclient.DoJSON[actionsResponse](ctx, c.client, "GET", "/api/v1/actions", nil)
	if err != nil {
		return nil, fmt.Errorf("composio: list tools: %w", err)
	}

	tools := make([]tool.Tool, len(resp.Items))
	for i, info := range resp.Items {
		tools[i] = &composioTool{
			client: c,
			info:   info,
		}
	}
	return tools, nil
}

// composioTool wraps a Composio action as a native tool.Tool.
type composioTool struct {
	client *Client
	info   actionInfo
}

func (t *composioTool) Name() string {
	return t.info.Name
}

func (t *composioTool) Description() string {
	desc := t.info.Description
	if desc == "" {
		desc = t.info.DisplayName
	}
	return desc
}

func (t *composioTool) InputSchema() map[string]any {
	if t.info.Parameters != nil {
		return t.info.Parameters
	}
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

func (t *composioTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
	req := executeRequest{Input: input}

	path := fmt.Sprintf("/api/v1/actions/%s/execute", t.info.Name)
	resp, err := httpclient.DoJSON[executeResponse](ctx, t.client.client, "POST", path, req)
	if err != nil {
		return nil, fmt.Errorf("composio: execute %s: %w", t.info.Name, err)
	}

	if !resp.Successful {
		errMsg := resp.Error
		if errMsg == "" {
			errMsg = "action execution failed"
		}
		return &tool.Result{
			Content: []schema.ContentPart{schema.TextPart{Text: errMsg}},
			IsError: true,
		}, nil
	}

	output := fmt.Sprintf("%v", resp.Data)
	return &tool.Result{
		Content: []schema.ContentPart{schema.TextPart{Text: output}},
	}, nil
}
