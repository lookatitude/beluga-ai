package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/lookatitude/beluga-ai/v2/tool"
)

// Client defaults.
const (
	// DefaultClientTimeout is the HTTP timeout used by NewClient unless
	// WithClientHTTPClient supplies a client.
	DefaultClientTimeout = 30 * time.Second
	// maxClientResponseBytes bounds how much of a response body is read.
	maxClientResponseBytes = 10 << 20
	// headerSessionID carries the Streamable HTTP session identifier.
	headerSessionID = "Mcp-Session-Id"
	// headerProtocolVersion carries the negotiated protocol revision.
	headerProtocolVersion = "MCP-Protocol-Version"
	// defaultClientName / defaultClientVersion populate clientInfo.
	defaultClientName    = "beluga-ai"
	defaultClientVersion = "2"
)

// MCPClient connects to a remote MCP server over Streamable HTTP transport.
//
// Initialize performs the 2025-06-18 handshake: it sends protocolVersion,
// capabilities and clientInfo, stores the Mcp-Session-Id returned by the
// server (resending it on every later request), and sends the
// notifications/initialized notification. Responses may be either
// application/json or text/event-stream; for SSE the first JSON-RPC response
// whose id matches the request is used. Response bodies are bounded to
// 10 MiB.
type MCPClient struct { //nolint:revive // exported name is public API; renaming to remove the stutter would break callers
	serverURL  string
	httpClient *http.Client
	nextID     atomic.Int64

	clientName    string
	clientVersion string

	mu              sync.RWMutex
	sessionID       string
	protocolVersion string
}

// ClientOption configures an MCPClient.
type ClientOption func(*MCPClient)

// WithClientHTTPClient sets the HTTP client used for requests. A nil client
// is ignored. The default is an http.Client with DefaultClientTimeout.
func WithClientHTTPClient(hc *http.Client) ClientOption {
	return func(c *MCPClient) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithClientInfo sets the clientInfo name and version sent in initialize.
func WithClientInfo(name, version string) ClientOption {
	return func(c *MCPClient) {
		if name != "" {
			c.clientName = name
		}
		if version != "" {
			c.clientVersion = version
		}
	}
}

// NewClient creates a new MCP client pointing at the given server URL.
// serverURL is validated at request time by validateServerURL, which
// rejects anything that does not parse as an http(s) URL with a host.
func NewClient(serverURL string, opts ...ClientOption) *MCPClient {
	c := &MCPClient{
		serverURL:     serverURL,
		httpClient:    &http.Client{Timeout: DefaultClientTimeout},
		clientName:    defaultClientName,
		clientVersion: defaultClientVersion,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// SessionID returns the Mcp-Session-Id assigned by the server during
// Initialize, or "" if the server did not assign one.
func (c *MCPClient) SessionID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sessionID
}

// initializeParams are the parameters of the "initialize" request.
type initializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ClientInfo      ServerInfo     `json:"clientInfo"`
}

// validateServerURL parses raw and returns its canonical form if it is a
// well-formed http or https URL with a non-empty host. Anything else is
// rejected with ErrInvalidInput. This sanitises the URL before it reaches
// the outbound HTTP request, so gosec/GHAS taint analysis recognises the
// origin as validated.
func validateServerURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", core.Errorf(core.ErrInvalidInput, "mcp: parse server URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", core.Errorf(core.ErrInvalidInput, "mcp: server URL scheme must be http or https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return "", core.Errorf(core.ErrInvalidInput, "mcp: server URL must include a host")
	}
	return u.String(), nil
}

// Initialize performs the MCP handshake and returns the server's capabilities.
func (c *MCPClient) Initialize(ctx context.Context) (*ServerCapabilities, error) {
	params := initializeParams{
		ProtocolVersion: ProtocolVersionLatest,
		Capabilities:    map[string]any{},
		ClientInfo:      ServerInfo{Name: c.clientName, Version: c.clientVersion},
	}
	var result InitializeResult
	if err := c.call(ctx, "initialize", params, &result); err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "mcp/initialize: %w", err)
	}
	c.mu.Lock()
	c.protocolVersion = result.ProtocolVersion
	c.mu.Unlock()

	if err := c.notify(ctx, "notifications/initialized", nil); err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "mcp/initialize: %w", err)
	}
	return &result.Capabilities, nil
}

// ListTools returns the list of tools available on the remote MCP server.
func (c *MCPClient) ListTools(ctx context.Context) ([]ToolInfo, error) {
	var result struct {
		Tools []ToolInfo `json:"tools"`
	}
	if err := c.call(ctx, "tools/list", nil, &result); err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "mcp/list_tools: %w", err)
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
		return nil, core.Errorf(core.ErrProviderDown, "mcp/call_tool: %w", err)
	}
	return &result, nil
}

// post sends one JSON-RPC message and returns the HTTP response. The caller
// must close the body.
func (c *MCPClient) post(ctx context.Context, msg Request) (*http.Response, error) {
	body, err := json.Marshal(msg)
	if err != nil {
		return nil, core.Errorf(core.ErrInvalidInput, "marshal request: %w", err)
	}

	validatedURL, err := validateServerURL(c.serverURL)
	if err != nil {
		return nil, err
	}
	// #nosec G704 -- validatedURL has been parsed and scheme-checked by validateServerURL
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, validatedURL, bytes.NewReader(body))
	if err != nil {
		return nil, core.Errorf(core.ErrInvalidInput, "create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")
	c.mu.RLock()
	sessionID, version := c.sessionID, c.protocolVersion
	c.mu.RUnlock()
	if sessionID != "" {
		httpReq.Header.Set(headerSessionID, sessionID)
	}
	if version != "" {
		httpReq.Header.Set(headerProtocolVersion, version)
	}

	// #nosec G704 -- httpReq uses validatedURL, sanitised above
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "send request: %w", err)
	}
	if sid := httpResp.Header.Get(headerSessionID); sid != "" {
		c.mu.Lock()
		c.sessionID = sid
		c.mu.Unlock()
	}
	return httpResp, nil
}

// notify sends a JSON-RPC notification (no id). Any 2xx status is success.
func (c *MCPClient) notify(ctx context.Context, method string, params any) error {
	httpResp, err := c.post(ctx, Request{JSONRPC: "2.0", Method: method, Params: params})
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(httpResp.Body, maxClientResponseBytes))
	_ = httpResp.Body.Close()
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return core.Errorf(core.ErrProviderDown, "notification %s: HTTP %d", method, httpResp.StatusCode)
	}
	return nil
}

func (c *MCPClient) call(ctx context.Context, method string, params any, result any) error {
	id := c.nextID.Add(1)
	httpResp, err := c.post(ctx, Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return err
	}
	defer func() { _ = httpResp.Body.Close() }()

	body := io.LimitReader(httpResp.Body, maxClientResponseBytes)

	var resp *Response
	mediaType, _, _ := mime.ParseMediaType(httpResp.Header.Get("Content-Type"))
	if mediaType == "text/event-stream" {
		resp, err = readSSEResponse(body, id)
	} else {
		resp = &Response{}
		err = json.NewDecoder(body).Decode(resp)
		if err != nil {
			err = core.Errorf(core.ErrProviderDown, "decode response: %w", err)
		}
	}
	if err != nil {
		if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
			return core.Errorf(core.ErrProviderDown, "HTTP %d", httpResp.StatusCode)
		}
		return err
	}

	if resp.Error != nil {
		return core.Errorf(core.ErrProviderDown, "rpc error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return core.Errorf(core.ErrProviderDown, "HTTP %d", httpResp.StatusCode)
	}

	// Re-marshal the result to decode into the target type.
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return core.Errorf(core.ErrInvalidInput, "marshal result: %w", err)
	}
	if err := json.Unmarshal(resultBytes, result); err != nil {
		return core.Errorf(core.ErrInvalidInput, "decode result: %w", err)
	}

	return nil
}

// readSSEResponse scans a text/event-stream body and returns the first
// JSON-RPC response whose id equals id. Server-initiated requests and
// notifications on the stream are skipped.
func readSSEResponse(body io.Reader, id int64) (*Response, error) {
	br := bufio.NewReader(body)
	var data strings.Builder
	dispatch := func() (*Response, bool) {
		if data.Len() == 0 {
			return nil, false
		}
		payload := data.String()
		data.Reset()
		var resp Response
		if json.Unmarshal([]byte(payload), &resp) != nil {
			return nil, false
		}
		if !idEquals(resp.ID, id) || (resp.Result == nil && resp.Error == nil) {
			return nil, false
		}
		return &resp, true
	}
	for {
		line, err := br.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		switch {
		case line == "":
			if resp, ok := dispatch(); ok {
				return resp, nil
			}
		case strings.HasPrefix(line, "data:"):
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
		if err != nil {
			if resp, ok := dispatch(); ok {
				return resp, nil
			}
			if errors.Is(err, io.EOF) {
				return nil, core.Errorf(core.ErrProviderDown, "event stream ended without a response for request %d", id)
			}
			return nil, core.Errorf(core.ErrProviderDown, "read event stream: %w", err)
		}
	}
}

// idEquals reports whether a decoded JSON-RPC id equals the numeric id.
func idEquals(got any, want int64) bool {
	switch v := got.(type) {
	case float64:
		return v == float64(want)
	case string:
		return v == strconv.FormatInt(want, 10)
	default:
		return false
	}
}

// FromMCP connects to an MCP server and returns its tools as native tool.Tool instances.
func FromMCP(ctx context.Context, serverURL string) ([]tool.Tool, error) {
	client := NewClient(serverURL)

	if _, err := client.Initialize(ctx); err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "mcp/from_mcp: %w", err)
	}

	infos, err := client.ListTools(ctx)
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "mcp/from_mcp: %w", err)
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

func (t *mcpTool) Name() string                { return t.info.Name }
func (t *mcpTool) Description() string         { return t.info.Description }
func (t *mcpTool) InputSchema() map[string]any { return t.info.InputSchema }

func (t *mcpTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
	result, err := t.client.CallTool(ctx, t.info.Name, input)
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "mcp/execute: %w", err)
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
