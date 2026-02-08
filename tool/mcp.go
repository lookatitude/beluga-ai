package tool

import (
	"context"
	"errors"
)

// MCPOption configures the MCP client connection.
type MCPOption func(*mcpOptions)

type mcpOptions struct {
	sessionID   string
	lastEventID string
	headers     map[string]string
}

// WithSessionID sets the Mcp-Session-Id header for session management.
func WithSessionID(id string) MCPOption {
	return func(o *mcpOptions) {
		o.sessionID = id
	}
}

// WithLastEventID sets the Last-Event-ID header for stream resumability.
func WithLastEventID(id string) MCPOption {
	return func(o *mcpOptions) {
		o.lastEventID = id
	}
}

// WithMCPHeaders sets additional HTTP headers for the MCP connection.
func WithMCPHeaders(headers map[string]string) MCPOption {
	return func(o *mcpOptions) {
		o.headers = headers
	}
}

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
	serverURL string
	opts      mcpOptions
}

// NewMCPClient creates a new MCP client targeting the given server URL.
func NewMCPClient(serverURL string, opts ...MCPOption) *MCPClient {
	o := mcpOptions{
		headers: make(map[string]string),
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &MCPClient{
		serverURL: serverURL,
		opts:      o,
	}
}

// Connect establishes a session with the MCP server.
func (c *MCPClient) Connect(ctx context.Context) error {
	return errors.New("mcp: Connect not implemented")
}

// ListTools retrieves the list of available tools from the MCP server.
func (c *MCPClient) ListTools(ctx context.Context) ([]Tool, error) {
	return nil, errors.New("mcp: ListTools not implemented")
}

// ExecuteTool invokes a named tool on the MCP server with the given input.
func (c *MCPClient) ExecuteTool(ctx context.Context, name string, input map[string]any) (*Result, error) {
	return nil, errors.New("mcp: ExecuteTool not implemented")
}

// Close terminates the MCP session using DELETE.
func (c *MCPClient) Close(ctx context.Context) error {
	return errors.New("mcp: Close not implemented")
}

// FromMCP connects to an MCP server and returns all its tools as native Tool
// instances. This is a convenience function that creates an MCPClient,
// connects, and retrieves tools.
//
// The returned tools use the Streamable HTTP transport and can be used
// interchangeably with local tools.
func FromMCP(ctx context.Context, serverURL string, opts ...MCPOption) ([]Tool, error) {
	return nil, errors.New("mcp: FromMCP not implemented")
}
