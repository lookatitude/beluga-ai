package learning

import (
	"context"
	"encoding/json"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/tool"
)

// DynamicTool implements tool.Tool with a runtime-defined schema and a pluggable
// CodeExecutor for execution. Unlike FuncTool which requires compile-time generics,
// DynamicTool supports schemas and execution logic defined at runtime, making it
// suitable for agent-generated tools.
type DynamicTool struct {
	name        string
	description string
	schema      map[string]any
	code        string
	executor    CodeExecutor
	version     int
}

// Compile-time check that DynamicTool implements tool.Tool.
var _ tool.Tool = (*DynamicTool)(nil)

// DynamicToolOption configures a DynamicTool.
type DynamicToolOption func(*DynamicTool)

// WithVersion sets the version number for the dynamic tool.
func WithVersion(v int) DynamicToolOption {
	return func(d *DynamicTool) {
		d.version = v
	}
}

// NewDynamicTool creates a new DynamicTool with the given name, description,
// input schema, source code, and code executor. The schema should be a valid
// JSON Schema object describing the tool's expected input parameters.
func NewDynamicTool(
	name, description string,
	inputSchema map[string]any,
	code string,
	executor CodeExecutor,
	opts ...DynamicToolOption,
) *DynamicTool {
	dt := &DynamicTool{
		name:        name,
		description: description,
		schema:      inputSchema,
		code:        code,
		executor:    executor,
		version:     1,
	}
	for _, opt := range opts {
		opt(dt)
	}
	return dt
}

// Name returns the tool's unique identifier.
func (d *DynamicTool) Name() string { return d.name }

// Description returns the tool's human-readable description.
func (d *DynamicTool) Description() string { return d.description }

// InputSchema returns the JSON Schema describing the tool's expected input.
func (d *DynamicTool) InputSchema() map[string]any { return d.schema }

// Code returns the source code of the dynamic tool.
func (d *DynamicTool) Code() string { return d.code }

// Version returns the version number of the dynamic tool.
func (d *DynamicTool) Version() int { return d.version }

// Execute runs the tool by serializing the input map to JSON and passing it
// along with the tool's code to the configured CodeExecutor.
func (d *DynamicTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, core.Errorf(core.ErrInvalidInput, "dynamic tool %s: failed to marshal input: %w", d.name, err)
	}

	output, err := d.executor.Execute(ctx, d.code, string(inputJSON))
	if err != nil {
		return nil, core.Errorf(core.ErrToolFailed, "dynamic tool %s: execution failed: %w", d.name, err)
	}

	return tool.TextResult(output), nil
}
