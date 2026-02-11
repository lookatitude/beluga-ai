package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// mockTool is a simple tool implementation for testing.
type mockTool struct {
	name    string
	desc    string
	schema  map[string]any
	execFn  func(ctx context.Context, input map[string]any) (*tool.Result, error)
}

func (t *mockTool) Name() string               { return t.name }
func (t *mockTool) Description() string         { return t.desc }
func (t *mockTool) InputSchema() map[string]any { return t.schema }
func (t *mockTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
	if t.execFn != nil {
		return t.execFn(ctx, input)
	}
	return tool.TextResult("ok"), nil
}

func TestNewServer(t *testing.T) {
	mt := &mockTool{
		name:   "greet",
		desc:   "Say hello",
		schema: map[string]any{"type": "object", "properties": map[string]any{"name": map[string]any{"type": "string"}}},
	}

	srv := NewServer("test-server", "1.0.0", mt)
	require.NotNil(t, srv)
}

func TestNewServerMultipleTools(t *testing.T) {
	t1 := &mockTool{name: "tool1", desc: "first tool"}
	t2 := &mockTool{name: "tool2", desc: "second tool"}

	srv := NewServer("multi", "2.0.0", t1, t2)
	require.NotNil(t, srv)
}

func TestNewServerNoTools(t *testing.T) {
	srv := NewServer("empty", "0.0.1")
	require.NotNil(t, srv)
}

func TestRoundTrip(t *testing.T) {
	mt := &mockTool{
		name:   "echo",
		desc:   "Echo input",
		schema: map[string]any{"type": "object"},
		execFn: func(ctx context.Context, input map[string]any) (*tool.Result, error) {
			msg, _ := input["message"].(string)
			return tool.TextResult("echo: " + msg), nil
		},
	}

	srv := NewServer("test", "1.0.0", mt)

	// Create in-memory transports for client-server communication.
	srvTransport, clientTransport := sdkmcp.NewInMemoryTransports()

	ctx := context.Background()

	// Connect server in background.
	done := make(chan error, 1)
	go func() {
		done <- srv.Run(ctx, srvTransport)
	}()

	// Connect client.
	_, session, err := NewClient(ctx, clientTransport)
	require.NoError(t, err)
	defer session.Close()

	// List tools.
	tools, err := FromSession(ctx, session)
	require.NoError(t, err)
	require.Len(t, tools, 1)

	assert.Equal(t, "echo", tools[0].Name())
	assert.Equal(t, "Echo input", tools[0].Description())

	// Call tool.
	result, err := tools[0].Execute(ctx, map[string]any{"message": "hello"})
	require.NoError(t, err)
	require.Len(t, result.Content, 1)
	assert.False(t, result.IsError)

	tp, ok := result.Content[0].(schema.TextPart)
	require.True(t, ok)
	assert.Equal(t, "echo: hello", tp.Text)
}

func TestRoundTripToolError(t *testing.T) {
	mt := &mockTool{
		name:   "fail",
		desc:   "Always fails",
		schema: map[string]any{"type": "object"},
		execFn: func(ctx context.Context, input map[string]any) (*tool.Result, error) {
			return &tool.Result{
				Content: []schema.ContentPart{schema.TextPart{Text: "something went wrong"}},
				IsError: true,
			}, nil
		},
	}

	srv := NewServer("test", "1.0.0", mt)
	srvTransport, clientTransport := sdkmcp.NewInMemoryTransports()

	ctx := context.Background()

	go srv.Run(ctx, srvTransport)

	_, session, err := NewClient(ctx, clientTransport)
	require.NoError(t, err)
	defer session.Close()

	tools, err := FromSession(ctx, session)
	require.NoError(t, err)
	require.Len(t, tools, 1)

	result, err := tools[0].Execute(ctx, nil)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestToSDKResult(t *testing.T) {
	result := &tool.Result{
		Content: []schema.ContentPart{
			schema.TextPart{Text: "hello"},
			schema.TextPart{Text: "world"},
		},
		IsError: false,
	}

	sdkResult := toSDKResult(result)
	assert.Len(t, sdkResult.Content, 2)
	assert.False(t, sdkResult.IsError)
}

func TestFromSDKResult(t *testing.T) {
	sdkResult := &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{
			&sdkmcp.TextContent{Text: "test"},
		},
		IsError: true,
	}

	result := fromSDKResult(sdkResult)
	require.Len(t, result.Content, 1)
	assert.True(t, result.IsError)

	tp, ok := result.Content[0].(schema.TextPart)
	require.True(t, ok)
	assert.Equal(t, "test", tp.Text)
}

func TestToInputSchema(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  map[string]any
	}{
		{
			name:  "nil",
			input: nil,
			want:  nil,
		},
		{
			name:  "map",
			input: map[string]any{"type": "object"},
			want:  map[string]any{"type": "object"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toInputSchema(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSdkToolInterface(t *testing.T) {
	st := &sdkTool{
		name:   "test-tool",
		desc:   "A test tool",
		schema: map[string]any{"type": "object"},
	}

	assert.Equal(t, "test-tool", st.Name())
	assert.Equal(t, "A test tool", st.Description())
	assert.Equal(t, map[string]any{"type": "object"}, st.InputSchema())
}

func TestRegisterTool_NilSchema(t *testing.T) {
	mt := &mockTool{
		name:   "nil-schema",
		desc:   "Tool with nil schema",
		schema: nil,
	}

	srv := NewServer("test", "1.0.0", mt)
	require.NotNil(t, srv)

	// Verify the tool works via round-trip.
	srvTransport, clientTransport := sdkmcp.NewInMemoryTransports()
	ctx := context.Background()

	go srv.Run(ctx, srvTransport)

	_, session, err := NewClient(ctx, clientTransport)
	require.NoError(t, err)
	defer session.Close()

	tools, err := FromSession(ctx, session)
	require.NoError(t, err)
	require.Len(t, tools, 1)
	assert.Equal(t, "nil-schema", tools[0].Name())
}

func TestRegisterTool_SchemaWithoutType(t *testing.T) {
	mt := &mockTool{
		name:   "no-type",
		desc:   "Tool with schema missing type",
		schema: map[string]any{"properties": map[string]any{"name": map[string]any{"type": "string"}}},
	}

	srv := NewServer("test", "1.0.0", mt)
	require.NotNil(t, srv)

	srvTransport, clientTransport := sdkmcp.NewInMemoryTransports()
	ctx := context.Background()

	go srv.Run(ctx, srvTransport)

	_, session, err := NewClient(ctx, clientTransport)
	require.NoError(t, err)
	defer session.Close()

	tools, err := FromSession(ctx, session)
	require.NoError(t, err)
	require.Len(t, tools, 1)
}

func TestRegisterTool_ExecuteError(t *testing.T) {
	mt := &mockTool{
		name:   "fail",
		desc:   "Tool that returns error",
		schema: map[string]any{"type": "object"},
		execFn: func(ctx context.Context, input map[string]any) (*tool.Result, error) {
			return nil, fmt.Errorf("execution failed")
		},
	}

	srv := NewServer("test", "1.0.0", mt)
	srvTransport, clientTransport := sdkmcp.NewInMemoryTransports()
	ctx := context.Background()

	go srv.Run(ctx, srvTransport)

	_, session, err := NewClient(ctx, clientTransport)
	require.NoError(t, err)
	defer session.Close()

	tools, err := FromSession(ctx, session)
	require.NoError(t, err)
	require.Len(t, tools, 1)

	result, err := tools[0].Execute(ctx, nil)
	require.NoError(t, err) // MCP wraps tool errors in result, not Go error.
	assert.True(t, result.IsError)
}

func TestExtractArgs_NilRequest(t *testing.T) {
	args, err := extractArgs(nil)
	assert.NoError(t, err)
	assert.Nil(t, args)
}

func TestExtractArgs_NilParams(t *testing.T) {
	args, err := extractArgs(&sdkmcp.CallToolRequest{})
	assert.NoError(t, err)
	assert.Nil(t, args)
}

func TestExtractArgs_WithArguments(t *testing.T) {
	rawArgs, _ := json.Marshal(map[string]any{"key": "value"})
	req := &sdkmcp.CallToolRequest{
		Params: &sdkmcp.CallToolParamsRaw{
			Name:      "test",
			Arguments: rawArgs,
		},
	}
	args, err := extractArgs(req)
	assert.NoError(t, err)
	assert.Equal(t, "value", args["key"])
}

func TestToInputSchema_Struct(t *testing.T) {
	// Test with a struct that needs marshal/unmarshal to become map[string]any.
	type testSchema struct {
		Type       string `json:"type"`
		Properties any    `json:"properties"`
	}
	s := testSchema{Type: "object", Properties: map[string]any{"x": 1}}

	result := toInputSchema(s)
	assert.NotNil(t, result)
	assert.Equal(t, "object", result["type"])
}

func TestToInputSchema_UnmarshalableToMap(t *testing.T) {
	// A channel can't be marshaled by json.
	ch := make(chan int)
	result := toInputSchema(ch)
	assert.Nil(t, result)
}

func TestToSDKResult_NonTextContent(t *testing.T) {
	// Use a content part that is NOT TextPart.
	result := &tool.Result{
		Content: []schema.ContentPart{
			schema.ImagePart{URL: "http://example.com/img.png"},
			schema.TextPart{Text: "hello"},
		},
		IsError: false,
	}

	sdkResult := toSDKResult(result)
	// Only TextPart should be converted; ImagePart is skipped.
	assert.Len(t, sdkResult.Content, 1)
}

func TestSdkTool_Execute_SessionClosed(t *testing.T) {
	mt := &mockTool{
		name:   "test",
		desc:   "Test tool",
		schema: map[string]any{"type": "object"},
	}

	srv := NewServer("test", "1.0.0", mt)
	srvTransport, clientTransport := sdkmcp.NewInMemoryTransports()
	ctx := context.Background()

	go srv.Run(ctx, srvTransport)

	_, session, err := NewClient(ctx, clientTransport)
	require.NoError(t, err)

	tools, err := FromSession(ctx, session)
	require.NoError(t, err)
	require.Len(t, tools, 1)

	// Close the session, then try to execute — should return error.
	session.Close()

	_, err = tools[0].Execute(ctx, nil)
	assert.Error(t, err)
}

func TestFromSDKResult_NonTextContent(t *testing.T) {
	sdkResult := &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{
			&sdkmcp.ImageContent{Data: []byte("base64data")},
			&sdkmcp.TextContent{Text: "test"},
		},
		IsError: false,
	}

	result := fromSDKResult(sdkResult)
	// Only TextContent should be converted; ImageContent is skipped.
	assert.Len(t, result.Content, 1)
	tp, ok := result.Content[0].(schema.TextPart)
	require.True(t, ok)
	assert.Equal(t, "test", tp.Text)
}
