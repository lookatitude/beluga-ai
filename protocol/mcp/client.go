package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// MCPClient connects to a remote MCP server over Streamable HTTP transport.
type MCPClient struct {
	serverURL  string
	httpClient *http.Client
	nextID     atomic.Int64
}

// NewClient creates a new MCP client pointing at the given server URL.
func NewClient(serverURL string) *MCPClient {
	return &MCPClient{
		serverURL:  serverURL,
		httpClient: http.DefaultClient,
	}
}

// Initialize performs the MCP handshake and returns the server's capabilities.
func (c *MCPClient) Initialize(ctx context.Context) (*ServerCapabilities, error) {
	var result InitializeResult
	if err := c.call(ctx, "initialize", nil, &result); err != nil {
		return nil, fmt.Errorf("mcp/initialize: %w", err)
	}
	return &result.Capabilities, nil
}

// ListTools returns the list of tools available on the remote MCP server.
func (c *MCPClient) ListTools(ctx context.Context) ([]ToolInfo, error) {
	var result struct {
		Tools []ToolInfo `json:"tools"`
	}
	if err := c.call(ctx, "tools/list", nil, &result); err != nil {
		return nil, fmt.Errorf("mcp/list_tools: %w", err)
	}
	return result.Tools, nil
}

// CallTool invokes a named tool on the remote MCP server with the given arguments.
func (c *MCPClient) CallTool(ctx context.Context, name string, args map[string]any) (*ToolCallResult, error) {
	params := ToolCallParams{
		Name:      name,
		Arguments: args,
	}
	var result ToolCallResult
	if err := c.call(ctx, "tools/call", params, &result); err != nil {
		return nil, fmt.Errorf("mcp/call_tool: %w", err)
	}
	return &result, nil
}

func (c *MCPClient) call(ctx context.Context, method string, params any, result any) error {
	id := c.nextID.Add(1)
	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.serverURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	var resp Response
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	// Re-marshal the result to decode into the target type.
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}
	if err := json.Unmarshal(resultBytes, result); err != nil {
		return fmt.Errorf("decode result: %w", err)
	}

	return nil
}

// FromMCP connects to an MCP server and returns its tools as native tool.Tool instances.
func FromMCP(ctx context.Context, serverURL string) ([]tool.Tool, error) {
	client := NewClient(serverURL)

	if _, err := client.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("mcp/from_mcp: %w", err)
	}

	infos, err := client.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("mcp/from_mcp: %w", err)
	}

	tools := make([]tool.Tool, len(infos))
	for i, info := range infos {
		tools[i] = &mcpTool{
			client: client,
			info:   info,
		}
	}
	return tools, nil
}

// mcpTool wraps an MCP remote tool as a native tool.Tool.
type mcpTool struct {
	client *MCPClient
	info   ToolInfo
}

func (t *mcpTool) Name() string              { return t.info.Name }
func (t *mcpTool) Description() string        { return t.info.Description }
func (t *mcpTool) InputSchema() map[string]any { return t.info.InputSchema }

func (t *mcpTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
	result, err := t.client.CallTool(ctx, t.info.Name, input)
	if err != nil {
		return nil, fmt.Errorf("mcp/execute: %w", err)
	}

	parts := make([]schema.ContentPart, 0, len(result.Content))
	for _, item := range result.Content {
		if item.Type == "text" {
			parts = append(parts, schema.TextPart{Text: item.Text})
		}
	}

	return &tool.Result{
		Content: parts,
		IsError: result.IsError,
	}, nil
}
