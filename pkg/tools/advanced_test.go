package tools

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/tools/iface"
)

// =============================================================================
// Registry Tests
// =============================================================================

func TestToolRegistry_NewToolRegistry(t *testing.T) {
	registry := NewToolRegistry()
	require.NotNil(t, registry)
	require.NotNil(t, registry.creators)
	require.NotNil(t, registry.tools)
}

func TestToolRegistry_RegisterType(t *testing.T) {
	registry := NewToolRegistry()

	creator := func(ctx context.Context, config ToolConfig) (iface.Tool, error) {
		return NewMockTool(config.Name, config.Description), nil
	}

	registry.RegisterType("test", creator)

	types := registry.ListToolTypes()
	assert.Contains(t, types, "test")
}

func TestToolRegistry_Create(t *testing.T) {
	registry := NewToolRegistry()

	creator := func(ctx context.Context, config ToolConfig) (iface.Tool, error) {
		return NewMockTool(config.Name, config.Description), nil
	}

	registry.RegisterType("test", creator)

	config := ToolConfig{
		Name:        "my-tool",
		Description: "A test tool",
		Type:        "test",
	}

	tool, err := registry.Create(context.Background(), "test", config)
	require.NoError(t, err)
	require.NotNil(t, tool)
	assert.Equal(t, "my-tool", tool.Name())
	assert.Equal(t, "A test tool", tool.Description())
}

func TestToolRegistry_Create_NotRegistered(t *testing.T) {
	registry := NewToolRegistry()

	config := ToolConfig{
		Name:        "my-tool",
		Description: "A test tool",
		Type:        "unknown",
	}

	tool, err := registry.Create(context.Background(), "unknown", config)
	assert.Error(t, err)
	assert.Nil(t, tool)
	assert.True(t, IsErrorCode(err, ErrorCodeNotFound))
}

func TestToolRegistry_RegisterTool(t *testing.T) {
	registry := NewToolRegistry()

	tool := NewMockTool("my-tool", "A test tool")
	err := registry.RegisterTool(tool)
	require.NoError(t, err)

	retrieved, err := registry.GetTool("my-tool")
	require.NoError(t, err)
	assert.Equal(t, tool, retrieved)
}

func TestToolRegistry_RegisterTool_EmptyName(t *testing.T) {
	registry := NewToolRegistry()

	tool := NewMockTool("", "A test tool")
	err := registry.RegisterTool(tool)
	assert.Error(t, err)
	assert.True(t, IsErrorCode(err, ErrorCodeInvalidInput))
}

func TestToolRegistry_RegisterTool_AlreadyExists(t *testing.T) {
	registry := NewToolRegistry()

	tool1 := NewMockTool("my-tool", "First tool")
	tool2 := NewMockTool("my-tool", "Second tool")

	err := registry.RegisterTool(tool1)
	require.NoError(t, err)

	err = registry.RegisterTool(tool2)
	assert.Error(t, err)
	assert.True(t, IsErrorCode(err, ErrorCodeAlreadyExists))
}

func TestToolRegistry_GetTool_NotFound(t *testing.T) {
	registry := NewToolRegistry()

	tool, err := registry.GetTool("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, tool)
	assert.True(t, IsErrorCode(err, ErrorCodeNotFound))
}

func TestToolRegistry_ListTools(t *testing.T) {
	registry := NewToolRegistry()

	_ = registry.RegisterTool(NewMockTool("tool1", "First"))
	_ = registry.RegisterTool(NewMockTool("tool2", "Second"))

	tools := registry.ListTools()
	assert.Len(t, tools, 2)
	assert.Contains(t, tools, "tool1")
	assert.Contains(t, tools, "tool2")
}

func TestToolRegistry_GetToolDescriptions(t *testing.T) {
	registry := NewToolRegistry()

	_ = registry.RegisterTool(NewMockTool("tool1", "First tool"))
	_ = registry.RegisterTool(NewMockTool("tool2", "Second tool"))

	desc := registry.GetToolDescriptions()
	assert.Contains(t, desc, "tool1")
	assert.Contains(t, desc, "First tool")
	assert.Contains(t, desc, "tool2")
	assert.Contains(t, desc, "Second tool")
}

func TestToolRegistry_GetToolDescriptions_Empty(t *testing.T) {
	registry := NewToolRegistry()

	desc := registry.GetToolDescriptions()
	assert.Equal(t, "No tools registered.", desc)
}

func TestToolRegistry_Clear(t *testing.T) {
	registry := NewToolRegistry()

	registry.RegisterType("test", func(ctx context.Context, config ToolConfig) (iface.Tool, error) {
		return NewMockTool(config.Name, config.Description), nil
	})
	_ = registry.RegisterTool(NewMockTool("tool1", "First"))

	registry.Clear()

	assert.Empty(t, registry.ListToolTypes())
	assert.Empty(t, registry.ListTools())
}

func TestGlobalRegistry(t *testing.T) {
	registry := GetRegistry()
	require.NotNil(t, registry)

	// Global registry should have built-in types
	// Note: We don't test specific types here since they may vary
}

// =============================================================================
// Error Tests
// =============================================================================

func TestToolError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ToolError
		contains string
	}{
		{
			name: "with message",
			err: &ToolError{
				Op:      "test_op",
				Code:    ErrorCodeInvalidInput,
				Message: "test message",
			},
			contains: "test message",
		},
		{
			name: "with underlying error",
			err: &ToolError{
				Op:   "test_op",
				Code: ErrorCodeTimeout,
				Err:  ErrTimeout,
			},
			contains: "timed out",
		},
		{
			name: "with tool name",
			err: &ToolError{
				Op:       "test_op",
				Code:     ErrorCodeExecutionFailed,
				ToolName: "calculator",
				Message:  "failed",
			},
			contains: "calculator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			assert.Contains(t, errStr, tt.contains)
			assert.Contains(t, errStr, "tools")
		})
	}
}

func TestToolError_Unwrap(t *testing.T) {
	inner := ErrTimeout
	err := NewTimeoutError("test-tool", "test", "timeout occurred")

	unwrapped := err.Unwrap()
	assert.Equal(t, inner, unwrapped)
}

func TestToolError_AddContext(t *testing.T) {
	err := NewToolError("test", ErrorCodeInvalidInput, "test", nil)
	err.AddContext("key1", "value1")
	err.AddContext("key2", 42)

	assert.Equal(t, "value1", err.Context["key1"])
	assert.Equal(t, 42, err.Context["key2"])
}

func TestNewToolError_Variants(t *testing.T) {
	tests := []struct {
		name      string
		createErr func() *ToolError
		wantCode  ErrorCode
	}{
		{
			name:      "invalid input",
			createErr: func() *ToolError { return NewInvalidInputError("op", "msg", nil) },
			wantCode:  ErrorCodeInvalidInput,
		},
		{
			name:      "invalid schema",
			createErr: func() *ToolError { return NewInvalidSchemaError("op", "msg", nil) },
			wantCode:  ErrorCodeInvalidSchema,
		},
		{
			name:      "execution error",
			createErr: func() *ToolError { return NewExecutionError("tool", "op", "msg", nil) },
			wantCode:  ErrorCodeExecutionFailed,
		},
		{
			name:      "timeout",
			createErr: func() *ToolError { return NewTimeoutError("tool", "op", "msg") },
			wantCode:  ErrorCodeTimeout,
		},
		{
			name:      "not found",
			createErr: func() *ToolError { return NewNotFoundError("op", "tool") },
			wantCode:  ErrorCodeNotFound,
		},
		{
			name:      "already exists",
			createErr: func() *ToolError { return NewAlreadyExistsError("op", "tool") },
			wantCode:  ErrorCodeAlreadyExists,
		},
		{
			name:      "unsupported",
			createErr: func() *ToolError { return NewUnsupportedError("op", "msg") },
			wantCode:  ErrorCodeUnsupported,
		},
		{
			name:      "rate limit",
			createErr: func() *ToolError { return NewRateLimitError("tool", "op", "5s") },
			wantCode:  ErrorCodeRateLimited,
		},
		{
			name:      "permission denied",
			createErr: func() *ToolError { return NewPermissionDeniedError("tool", "op", "msg") },
			wantCode:  ErrorCodePermissionDenied,
		},
		{
			name:      "internal error",
			createErr: func() *ToolError { return NewInternalError("op", "msg", nil) },
			wantCode:  ErrorCodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createErr()
			assert.Equal(t, tt.wantCode, err.Code)
		})
	}
}

func TestIsToolError(t *testing.T) {
	toolErr := NewInvalidInputError("op", "msg", nil)
	assert.True(t, IsToolError(toolErr))

	regularErr := errors.New("regular error")
	assert.False(t, IsToolError(regularErr))
}

func TestAsToolError(t *testing.T) {
	toolErr := NewInvalidInputError("op", "msg", nil)
	result, ok := AsToolError(toolErr)
	assert.True(t, ok)
	assert.Equal(t, toolErr, result)

	regularErr := errors.New("regular error")
	result, ok = AsToolError(regularErr)
	assert.False(t, ok)
	assert.Nil(t, result)
}

func TestGetErrorCode(t *testing.T) {
	toolErr := NewInvalidInputError("op", "msg", nil)
	code, ok := GetErrorCode(toolErr)
	assert.True(t, ok)
	assert.Equal(t, ErrorCodeInvalidInput, code)

	regularErr := errors.New("regular error")
	code, ok = GetErrorCode(regularErr)
	assert.False(t, ok)
	assert.Equal(t, ErrorCode(""), code)
}

func TestIsErrorCode(t *testing.T) {
	toolErr := NewInvalidInputError("op", "msg", nil)
	assert.True(t, IsErrorCode(toolErr, ErrorCodeInvalidInput))
	assert.False(t, IsErrorCode(toolErr, ErrorCodeTimeout))

	regularErr := errors.New("regular error")
	assert.False(t, IsErrorCode(regularErr, ErrorCodeTimeout))
}

// =============================================================================
// Config Tests
// =============================================================================

func TestConfig_SetDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.SetDefaults()

	require.NotNil(t, cfg.Global)
	assert.True(t, cfg.Global.Enabled)
	assert.Equal(t, 30*time.Second, cfg.Global.Timeout)
	assert.Equal(t, 3, cfg.Global.MaxRetries)
	assert.Equal(t, 10, cfg.Global.MaxConcurrency)

	require.NotNil(t, cfg.API)
	assert.True(t, cfg.API.Enabled)

	require.NotNil(t, cfg.Shell)
	assert.True(t, cfg.Shell.Enabled)

	require.NotNil(t, cfg.MCP)
	assert.True(t, cfg.MCP.Enabled)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "valid default config",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid max retries",
			cfg: func() *Config {
				c := DefaultConfig()
				c.Global.MaxRetries = 100 // Too high
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid max concurrency",
			cfg: func() *Config {
				c := DefaultConfig()
				c.Global.MaxConcurrency = 0 // Too low
				return c
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestToolConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ToolConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: ToolConfig{
				Name:        "test",
				Description: "A test tool",
				Type:        "calculator",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			cfg: ToolConfig{
				Description: "A test tool",
				Type:        "calculator",
			},
			wantErr: true,
		},
		{
			name: "missing description",
			cfg: ToolConfig{
				Name: "test",
				Type: "calculator",
			},
			wantErr: true,
		},
		{
			name: "missing type",
			cfg: ToolConfig{
				Name:        "test",
				Description: "A test tool",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Metrics Tests
// =============================================================================

func TestMetrics_NewMetrics(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")

	m, err := NewMetrics(meter, tracer)
	require.NoError(t, err)
	require.NotNil(t, m)
}

func TestMetrics_RecordExecution(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")

	m, err := NewMetrics(meter, tracer)
	require.NoError(t, err)

	// Should not panic
	m.RecordExecution(context.Background(), "calculator", "calculator", 100*time.Millisecond, true)
	m.RecordExecution(context.Background(), "calculator", "calculator", 100*time.Millisecond, false)
}

func TestMetrics_RecordBatch(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")

	m, err := NewMetrics(meter, tracer)
	require.NoError(t, err)

	// Should not panic
	m.RecordBatch(context.Background(), "calculator", "calculator", 5, 500*time.Millisecond, true)
}

func TestMetrics_RecordError(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")

	m, err := NewMetrics(meter, tracer)
	require.NoError(t, err)

	// Should not panic
	m.RecordError(context.Background(), "calculator", "calculator", "invalid_input")
}

func TestMetrics_NilSafe(t *testing.T) {
	var m *Metrics

	ctx := context.Background()

	// All methods should be nil-safe
	m.RecordExecution(ctx, "tool", "type", time.Second, true)
	m.RecordBatch(ctx, "tool", "type", 5, time.Second, true)
	m.RecordError(ctx, "tool", "type", "err")
	m.StartExecution(ctx, "tool", "type")
	m.EndExecution(ctx, "tool", "type")
	m.RecordToolRegistered(ctx, "tool", "type")
	m.RecordToolUnregistered(ctx, "tool", "type")
}

func TestNoOpMetrics(t *testing.T) {
	m := NoOpMetrics()
	require.NotNil(t, m)
	require.NotNil(t, m.tracer)
}

// =============================================================================
// MockTool Tests
// =============================================================================

func TestMockTool_Execute(t *testing.T) {
	tool := NewMockTool("test", "A test tool")

	result, err := tool.Execute(context.Background(), map[string]any{"input": "hello"})
	require.NoError(t, err)
	assert.Equal(t, "mock result", result)
	assert.Equal(t, 1, tool.ExecuteCalls)
}

func TestMockTool_WithExecute(t *testing.T) {
	tool := NewMockTool("test", "A test tool").
		WithExecute(func(ctx context.Context, input any) (any, error) {
			return "custom result", nil
		})

	result, err := tool.Execute(context.Background(), map[string]any{"input": "hello"})
	require.NoError(t, err)
	assert.Equal(t, "custom result", result)
}

func TestMockTool_Batch(t *testing.T) {
	tool := NewMockTool("test", "A test tool")

	inputs := []any{
		map[string]any{"input": "a"},
		map[string]any{"input": "b"},
	}

	results, err := tool.Batch(context.Background(), inputs)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, 1, tool.BatchCalls)
}

func TestMockTool_Definition(t *testing.T) {
	tool := NewMockTool("test", "A test tool")

	def := tool.Definition()
	assert.Equal(t, "test", def.Name)
	assert.Equal(t, "A test tool", def.Description)
	assert.NotNil(t, def.InputSchema)
}

// =============================================================================
// Options Tests
// =============================================================================

func TestOptions(t *testing.T) {
	cfg := defaultOptionConfig()

	WithTimeout(5 * time.Second)(cfg)
	assert.Equal(t, 5*time.Second, cfg.timeout)

	WithMaxRetries(5)(cfg)
	assert.Equal(t, 5, cfg.maxRetries)

	WithMaxConcurrency(20)(cfg)
	assert.Equal(t, 20, cfg.maxConcurrency)

	WithMetrics(false)(cfg)
	assert.False(t, cfg.enableMetrics)
}
