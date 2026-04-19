package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// maxResponseBodySize is the maximum size of a JSON-RPC response body (10 MB).
const maxResponseBodySize = 10 << 20

// maxRPCErrorMessageLen is the maximum length of an RPC error message included
// in returned errors.
const maxRPCErrorMessageLen = 512

// defaultMCPHTTPClient is used when no custom HTTP client is provided.
var defaultMCPHTTPClient = &http.Client{Timeout: 30 * time.Second}

// Compile-time interface assertion.
var _ Tool = (*mcpTool)(nil)

// ---------------------------------------------------------------------------
// JSON-RPC 2.0 types (unexported)
// ---------------------------------------------------------------------------

type jsonrpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ---------------------------------------------------------------------------
// MCP-specific protocol types (unexported)
// ---------------------------------------------------------------------------

type initializeParams struct {
	ProtocolVersion string     `json:"protocolVersion"`
	Capabilities    struct{}   `json:"capabilities"`
	ClientInfo      serverInfo `json:"clientInfo"`
}

type initializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    serverCapabilities `json:"capabilities"`
	ServerInfo      serverInfo         `json:"serverInfo"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type serverCapabilities struct {
	Tools *struct{} `json:"tools,omitempty"`
}

type toolInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type toolsListResult struct {
	Tools []toolInfo `json:"tools"`
}

type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type toolCallResult struct {
	Content []contentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type contentItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ---------------------------------------------------------------------------
// MCPOption and option constructors
// ---------------------------------------------------------------------------

// MCPOption configures the MCP client connection.
type MCPOption func(*mcpOptions)

type mcpOptions struct {
	sessionID  string
	headers    map[string]string
	httpClient *http.Client
}

// WithSessionID sets the Mcp-Session-Id header for session management.
func WithSessionID(id string) MCPOption {
	return func(o *mcpOptions) {
		o.sessionID = id
	}
}

// WithMCPHeaders sets additional HTTP headers for the MCP connection.
func WithMCPHeaders(headers map[string]string) MCPOption {
	return func(o *mcpOptions) {
		o.headers = headers
	}
}

// WithHTTPClient sets a custom HTTP client for the MCP connection.
func WithHTTPClient(c *http.Client) MCPOption {
	return func(o *mcpOptions) {
		o.httpClient = c
	}
}

// ---------------------------------------------------------------------------
// MCPClient
// ---------------------------------------------------------------------------

// MCPClient connects to an MCP server using the Streamable HTTP transport
// (March 2025 spec). It wraps remote tools as native Tool instances.
//
// Transport protocol:
//   - POST for client→server requests
//   - GET for server→client notifications
//   - DELETE for session termination
//   - Mcp-Session-Id header for session management
//   - Last-Event-ID for stream resumability
type MCPClient struct {
	serverURL    string
	opts         mcpOptions
	httpClient   *http.Client
	nextID       atomic.Int64
	sessionID    string
	capabilities *serverCapabilities
	connected    bool
	connecting   bool
	mu           sync.Mutex
}

// NewMCPClient creates a new MCP client targeting the given server URL.
// The URL is validated during Connect; an invalid URL will cause Connect to
// return an error.
func NewMCPClient(serverURL string, opts ...MCPOption) *MCPClient {
	o := mcpOptions{
		headers: make(map[string]string),
	}
	for _, opt := range opts {
		opt(&o)
	}

	// Deep-copy headers so callers cannot mutate them after construction.
	headersCopy := make(map[string]string, len(o.headers))
	for k, v := range o.headers {
		headersCopy[k] = v
	}
	o.headers = headersCopy

	hc := o.httpClient
	if hc == nil {
		hc = defaultMCPHTTPClient
	}

	c := &MCPClient{
		serverURL:  serverURL,
		opts:       o,
		httpClient: hc,
		sessionID:  o.sessionID,
	}
	return c
}

// call performs a JSON-RPC 2.0 request to the MCP server and decodes the
// result into dst. If dst is nil the result is discarded (used for
// notifications).
func (c *MCPClient) call(ctx context.Context, method string, params any, dst any) error {
	const op = "mcp.call"

	id := c.nextID.Add(1)
	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return core.NewError(op, core.ErrInvalidInput, "marshal request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.serverURL, bytes.NewReader(body))
	if err != nil {
		return core.NewError(op, core.ErrInvalidInput, "create request", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return core.NewError(op, core.ErrProviderDown, "send request", err)
	}
	defer resp.Body.Close()

	if err := c.checkStatusCode(op, resp.StatusCode); err != nil {
		return err
	}
	c.captureSessionID(resp)

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		return core.NewError(op, core.ErrInvalidInput,
			fmt.Sprintf("unexpected Content-Type %q, expected application/json", ct), nil)
	}

	limited := io.LimitReader(resp.Body, maxResponseBodySize)
	var rpcResp jsonrpcResponse
	if err := json.NewDecoder(limited).Decode(&rpcResp); err != nil {
		return core.NewError(op, core.ErrInvalidInput, "decode response", err)
	}

	if rpcResp.Error != nil {
		msg := rpcResp.Error.Message
		if len(msg) > maxRPCErrorMessageLen {
			msg = msg[:maxRPCErrorMessageLen] + "...(truncated)"
		}
		return core.NewError(op, core.ErrToolFailed,
			fmt.Sprintf("rpc error %d: %s", rpcResp.Error.Code, msg), nil)
	}

	if dst != nil && rpcResp.Result != nil {
		if err := json.Unmarshal(rpcResp.Result, dst); err != nil {
			return core.NewError(op, core.ErrInvalidInput, "unmarshal result", err)
		}
	}

	return nil
}

// setHeaders applies standard MCP headers to an HTTP request.
func (c *MCPClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	c.mu.Lock()
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	c.mu.Unlock()

	for k, v := range c.opts.headers {
		req.Header.Set(k, v)
	}
}

// checkStatusCode maps HTTP status codes to typed core.Error.
func (c *MCPClient) checkStatusCode(op string, code int) error {
	switch {
	case code == http.StatusUnauthorized || code == http.StatusForbidden:
		return core.NewError(op, core.ErrAuth,
			fmt.Sprintf("HTTP %d", code), nil)
	case code == http.StatusTooManyRequests:
		return core.NewError(op, core.ErrRateLimit, "rate limited", nil)
	case code >= 500:
		return core.NewError(op, core.ErrProviderDown,
			fmt.Sprintf("HTTP %d", code), nil)
	case code < 200 || code >= 300:
		return core.NewError(op, core.ErrInvalidInput,
			fmt.Sprintf("unexpected HTTP %d", code), nil)
	}
	return nil
}

// captureSessionID reads the Mcp-Session-Id header from the response.
func (c *MCPClient) captureSessionID(resp *http.Response) {
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		c.mu.Lock()
		c.sessionID = sid
		c.mu.Unlock()
	}
}

// notify sends a JSON-RPC 2.0 notification (no ID, no response expected).
func (c *MCPClient) notify(ctx context.Context, method string, params any) error {
	const op = "mcp.notify"

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return core.NewError(op, core.ErrInvalidInput, "marshal notification", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.serverURL, bytes.NewReader(body))
	if err != nil {
		return core.NewError(op, core.ErrInvalidInput, "create request", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return core.NewError(op, core.ErrProviderDown, "send notification", err)
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBodySize))
	_ = resp.Body.Close()

	if err := c.checkStatusCode(op, resp.StatusCode); err != nil {
		return err
	}

	return nil
}

// Connect establishes a session with the MCP server by sending the
// "initialize" JSON-RPC method and then a "notifications/initialized"
// notification.
func (c *MCPClient) Connect(ctx context.Context) error {
	const op = "mcp.Connect"

	// Atomically check and set connecting flag to prevent TOCTOU race
	// where concurrent goroutines both pass the connected check.
	// We cannot hold the mutex during call()/notify() as they also acquire it.
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return core.NewError(op, core.ErrInvalidInput, "already connected", nil)
	}
	if c.connecting {
		c.mu.Unlock()
		return core.NewError(op, core.ErrInvalidInput, "connect already in progress", nil)
	}
	c.connecting = true
	c.mu.Unlock()

	// On failure, clear the connecting flag so Connect can be retried.
	defer func() {
		c.mu.Lock()
		if !c.connected {
			c.connecting = false
		}
		c.mu.Unlock()
	}()

	// H3: Validate server URL.
	if c.serverURL == "" {
		return core.NewError(op, core.ErrInvalidInput, "server URL is empty", nil)
	}
	u, err := url.Parse(c.serverURL)
	if err != nil {
		return core.NewError(op, core.ErrInvalidInput, "invalid server URL", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return core.NewError(op, core.ErrInvalidInput,
			fmt.Sprintf("unsupported URL scheme %q, must be http or https", u.Scheme), nil)
	}
	if u.Host == "" {
		return core.NewError(op, core.ErrInvalidInput, "server URL has no host", nil)
	}

	params := initializeParams{
		ProtocolVersion: "2025-03-26",
		ClientInfo: serverInfo{
			Name:    "beluga-ai",
			Version: "1.0.0",
		},
	}

	var result initializeResult
	if err := c.call(ctx, "initialize", params, &result); err != nil {
		return err
	}

	c.mu.Lock()
	c.capabilities = &result.Capabilities
	c.connected = true
	c.mu.Unlock()

	// Send initialized notification (fire-and-forget style, but still
	// propagate errors).
	if err := c.notify(ctx, "notifications/initialized", nil); err != nil {
		return err
	}

	return nil
}

// ListTools retrieves the list of available tools from the MCP server.
func (c *MCPClient) ListTools(ctx context.Context) ([]Tool, error) {
	const op = "mcp.ListTools"

	c.mu.Lock()
	connected := c.connected
	c.mu.Unlock()

	if !connected {
		return nil, core.NewError(op, core.ErrInvalidInput, "not connected", nil)
	}

	var result toolsListResult
	if err := c.call(ctx, "tools/list", nil, &result); err != nil {
		return nil, err
	}

	tools := make([]Tool, len(result.Tools))
	for i, ti := range result.Tools {
		tools[i] = &mcpTool{
			client:      c,
			name:        ti.Name,
			description: ti.Description,
			inputSchema: ti.InputSchema,
		}
	}
	return tools, nil
}

// ExecuteTool invokes a named tool on the MCP server with the given input.
func (c *MCPClient) ExecuteTool(ctx context.Context, name string, input map[string]any) (*Result, error) {
	const op = "mcp.ExecuteTool"

	c.mu.Lock()
	connected := c.connected
	c.mu.Unlock()

	if !connected {
		return nil, core.NewError(op, core.ErrInvalidInput, "not connected", nil)
	}

	params := toolCallParams{
		Name:      name,
		Arguments: input,
	}

	var result toolCallResult
	if err := c.call(ctx, "tools/call", params, &result); err != nil {
		return nil, err
	}

	// Convert MCP content items to schema.ContentPart.
	// All content types are rendered as text; future types (image, resource)
	// can be added as additional cases.
	parts := make([]schema.ContentPart, 0, len(result.Content))
	for _, item := range result.Content {
		parts = append(parts, schema.TextPart{Text: item.Text})
	}

	return &Result{
		Content: parts,
		IsError: result.IsError,
	}, nil
}

// Close terminates the MCP session by sending an HTTP DELETE with the session
// header. Close is idempotent: calling it on an unconnected client returns nil.
func (c *MCPClient) Close(ctx context.Context) error {
	const op = "mcp.Close"

	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return nil
	}
	sessionID := c.sessionID
	c.connected = false
	c.capabilities = nil
	c.mu.Unlock()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.serverURL, nil)
	if err != nil {
		return core.NewError(op, core.ErrInvalidInput, "create delete request", err)
	}

	if sessionID != "" {
		httpReq.Header.Set("Mcp-Session-Id", sessionID)
	}

	for k, v := range c.opts.headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return core.NewError(op, core.ErrProviderDown, "send delete", err)
	}
	// H1: Drain up to a bounded amount before closing to allow connection reuse
	// while preventing memory exhaustion.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBodySize))
	_ = resp.Body.Close()

	return nil
}

// FromMCP connects to an MCP server and returns all its tools as native Tool
// instances along with the MCPClient so callers can close the session.
//
// The returned tools use the Streamable HTTP transport and can be used
// interchangeably with local tools. Callers must call client.Close() when done.
func FromMCP(ctx context.Context, serverURL string, opts ...MCPOption) ([]Tool, *MCPClient, error) {
	client := NewMCPClient(serverURL, opts...)
	if err := client.Connect(ctx); err != nil {
		return nil, nil, err
	}
	tools, err := client.ListTools(ctx)
	if err != nil {
		_ = client.Close(ctx)
		return nil, nil, err
	}
	return tools, client, nil
}

// ---------------------------------------------------------------------------
// mcpTool — implements tool.Tool backed by an MCPClient
// ---------------------------------------------------------------------------

type mcpTool struct {
	client      *MCPClient
	name        string
	description string
	inputSchema map[string]any
}

// Name returns the tool's name.
func (t *mcpTool) Name() string { return t.name }

// Description returns the tool's description.
func (t *mcpTool) Description() string { return t.description }

// InputSchema returns a shallow copy of the tool's JSON Schema for its input
// parameters, preventing callers from mutating internal state.
func (t *mcpTool) InputSchema() map[string]any {
	cp := make(map[string]any, len(t.inputSchema))
	for k, v := range t.inputSchema {
		cp[k] = v
	}
	return cp
}

// Execute delegates to the MCPClient's ExecuteTool method.
func (t *mcpTool) Execute(ctx context.Context, input map[string]any) (*Result, error) {
	return t.client.ExecuteTool(ctx, t.name, input)
}
