package sdk

import (
	"context"
	"encoding/json"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// NewServer creates an MCP server using the official SDK and registers the
// given Beluga tools. Each tool.Tool is exposed as an MCP tool with its
// name, description, and input schema.
func NewServer(name, version string, tools ...tool.Tool) *sdkmcp.Server {
	srv := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    name,
		Version: version,
	}, nil)

	for _, t := range tools {
		registerTool(srv, t)
	}

	return srv
}

// registerTool registers a single Beluga tool with the MCP SDK server.
func registerTool(srv *sdkmcp.Server, t tool.Tool) {
	inputSchema := t.InputSchema()
	// The MCP SDK requires InputSchema to have type "object".
	if inputSchema == nil {
		inputSchema = map[string]any{"type": "object"}
	} else if _, ok := inputSchema["type"]; !ok {
		inputSchema["type"] = "object"
	}

	mcpTool := &sdkmcp.Tool{
		Name:        t.Name(),
		Description: t.Description(),
		InputSchema: inputSchema,
	}

	handler := func(ctx context.Context, req *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
		// Extract arguments from the raw request params.
		args, err := extractArgs(req)
		if err != nil {
			return &sdkmcp.CallToolResult{
				Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: "invalid arguments: " + err.Error()}},
				IsError: true,
			}, nil
		}

		result, err := t.Execute(ctx, args)
		if err != nil {
			return &sdkmcp.CallToolResult{
				Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil
		}

		return toSDKResult(result), nil
	}

	srv.AddTool(mcpTool, handler)
}

// extractArgs extracts the arguments map from an MCP CallToolRequest.
func extractArgs(req *sdkmcp.CallToolRequest) (map[string]any, error) {
	if req == nil || req.Params == nil {
		return nil, nil
	}

	// The raw params contain the full CallToolParams; marshal then unmarshal
	// to extract arguments.
	raw, err := json.Marshal(req.Params)
	if err != nil {
		return nil, fmt.Errorf("marshal params: %w", err)
	}

	var params struct {
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("unmarshal params: %w", err)
	}

	return params.Arguments, nil
}

// toSDKResult converts a Beluga tool.Result to an MCP SDK CallToolResult.
func toSDKResult(result *tool.Result) *sdkmcp.CallToolResult {
	content := make([]sdkmcp.Content, 0, len(result.Content))
	for _, part := range result.Content {
		if tp, ok := part.(schema.TextPart); ok {
			content = append(content, &sdkmcp.TextContent{Text: tp.Text})
		}
	}

	return &sdkmcp.CallToolResult{
		Content: content,
		IsError: result.IsError,
	}
}

// NewClient creates an MCP client using the official SDK and connects it
// to a server via the given transport. It returns the client and session.
// The caller should close the session when done.
func NewClient(ctx context.Context, transport sdkmcp.Transport) (*sdkmcp.Client, *sdkmcp.ClientSession, error) {
	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "beluga-client",
		Version: "1.0.0",
	}, nil)

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("mcp/sdk: connect: %w", err)
	}

	return client, session, nil
}

// FromSession lists tools from an MCP client session and returns them as
// native Beluga tool.Tool instances. Each remote tool is wrapped so that
// Execute calls the remote MCP server via the session.
func FromSession(ctx context.Context, session *sdkmcp.ClientSession) ([]tool.Tool, error) {
	result, err := session.ListTools(ctx, &sdkmcp.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("mcp/sdk: list tools: %w", err)
	}

	tools := make([]tool.Tool, 0, len(result.Tools))
	for _, t := range result.Tools {
		tools = append(tools, &sdkTool{
			session: session,
			name:    t.Name,
			desc:    t.Description,
			schema:  toInputSchema(t.InputSchema),
		})
	}

	return tools, nil
}

// toInputSchema converts the SDK's InputSchema (any) to a map[string]any.
func toInputSchema(s any) map[string]any {
	if s == nil {
		return nil
	}

	// If it's already a map, return directly.
	if m, ok := s.(map[string]any); ok {
		return m
	}

	// Otherwise marshal/unmarshal.
	data, err := json.Marshal(s)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return m
}

// sdkTool wraps a remote MCP tool discovered via the official SDK as a
// native Beluga tool.Tool.
type sdkTool struct {
	session *sdkmcp.ClientSession
	name    string
	desc    string
	schema  map[string]any
}

func (t *sdkTool) Name() string               { return t.name }
func (t *sdkTool) Description() string         { return t.desc }
func (t *sdkTool) InputSchema() map[string]any { return t.schema }

func (t *sdkTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
	result, err := t.session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      t.name,
		Arguments: input,
	})
	if err != nil {
		return nil, fmt.Errorf("mcp/sdk/execute: %w", err)
	}

	return fromSDKResult(result), nil
}

// fromSDKResult converts an MCP SDK CallToolResult to a Beluga tool.Result.
func fromSDKResult(result *sdkmcp.CallToolResult) *tool.Result {
	parts := make([]schema.ContentPart, 0, len(result.Content))
	for _, c := range result.Content {
		if tc, ok := c.(*sdkmcp.TextContent); ok {
			parts = append(parts, schema.TextPart{Text: tc.Text})
		}
	}

	return &tool.Result{
		Content: parts,
		IsError: result.IsError,
	}
}

// Compile-time interface check.
var _ tool.Tool = (*sdkTool)(nil)
