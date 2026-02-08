// Package mocktool provides a mock Tool implementation for testing.
// It allows test authors to configure a tool's name, description, input schema,
// and execution behavior, and tracks call history for assertions.
package mocktool

import (
	"context"
	"sync"

	"github.com/lookatitude/beluga-ai/schema"
)

// MockTool is a configurable mock for the Tool interface.
// It records Execute calls and can return preset results or errors.
type MockTool struct {
	mu sync.Mutex

	name        string
	description string
	inputSchema map[string]any
	result      *schema.ToolResult
	err         error
	executeFn   func(ctx context.Context, input map[string]any) (*schema.ToolResult, error)

	executeCalls int
	lastInput    map[string]any
}

// Option configures a MockTool.
type Option func(*MockTool)

// New creates a MockTool with the given name and description, and applies
// any additional options.
func New(name, description string, opts ...Option) *MockTool {
	m := &MockTool{
		name:        name,
		description: description,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// WithResult configures the mock to return the given ToolResult from Execute.
func WithResult(result *schema.ToolResult) Option {
	return func(m *MockTool) {
		m.result = result
	}
}

// WithError configures the mock to return the given error from Execute.
func WithError(err error) Option {
	return func(m *MockTool) {
		m.err = err
	}
}

// WithInputSchema sets the JSON Schema returned by InputSchema.
func WithInputSchema(s map[string]any) Option {
	return func(m *MockTool) {
		m.inputSchema = s
	}
}

// WithExecuteFunc sets a custom function to call on Execute, overriding
// the canned result/error.
func WithExecuteFunc(fn func(ctx context.Context, input map[string]any) (*schema.ToolResult, error)) Option {
	return func(m *MockTool) {
		m.executeFn = fn
	}
}

// Name returns the tool's configured name.
func (m *MockTool) Name() string {
	return m.name
}

// Description returns the tool's configured description.
func (m *MockTool) Description() string {
	return m.description
}

// InputSchema returns the tool's input JSON Schema. If none was configured,
// it returns a minimal object schema.
func (m *MockTool) InputSchema() map[string]any {
	if m.inputSchema != nil {
		return m.inputSchema
	}
	return map[string]any{"type": "object"}
}

// Execute runs the tool, returning the configured result or error.
// It records the call and input for later inspection.
func (m *MockTool) Execute(ctx context.Context, input map[string]any) (*schema.ToolResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.executeCalls++
	m.lastInput = input

	if m.executeFn != nil {
		return m.executeFn(ctx, input)
	}

	if m.err != nil {
		return nil, m.err
	}

	if m.result != nil {
		return m.result, nil
	}

	// Default: return a text result.
	return &schema.ToolResult{
		Content: []schema.ContentPart{schema.TextPart{Text: "mock result"}},
	}, nil
}

// ExecuteCalls returns the number of times Execute has been called.
func (m *MockTool) ExecuteCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.executeCalls
}

// LastInput returns the input map passed to the most recent Execute call.
func (m *MockTool) LastInput() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastInput
}

// SetResult updates the canned result for subsequent calls.
func (m *MockTool) SetResult(result *schema.ToolResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.result = result
	m.err = nil
}

// SetError updates the error for subsequent calls.
func (m *MockTool) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
	m.result = nil
}

// Reset clears all recorded calls and configuration.
func (m *MockTool) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executeCalls = 0
	m.lastInput = nil
	m.result = nil
	m.err = nil
	m.executeFn = nil
}
