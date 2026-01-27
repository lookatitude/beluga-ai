package agent

import (
	"context"
	"errors"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	memoryiface "github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// MockLLM is a mock implementation of llmsiface.LLM for testing.
type MockLLM struct {
	InvokeFunc       func(ctx context.Context, input any, options ...core.Option) (any, error)
	GetModelNameFunc func() string
	GetProviderFunc  func() string

	mu           sync.Mutex
	invokeCalls  []any
	invokeErrors []error
}

// NewMockLLM creates a new MockLLM with default behavior.
func NewMockLLM() *MockLLM {
	return &MockLLM{
		InvokeFunc: func(ctx context.Context, input any, options ...core.Option) (any, error) {
			return "Mock response", nil
		},
		GetModelNameFunc: func() string {
			return "mock-model"
		},
		GetProviderFunc: func() string {
			return "mock-provider"
		},
	}
}

// Invoke calls the mock InvokeFunc.
func (m *MockLLM) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	m.mu.Lock()
	m.invokeCalls = append(m.invokeCalls, input)
	m.mu.Unlock()

	if m.InvokeFunc != nil {
		return m.InvokeFunc(ctx, input, options...)
	}
	return "Mock response", nil
}

// GetModelName returns the mock model name.
func (m *MockLLM) GetModelName() string {
	if m.GetModelNameFunc != nil {
		return m.GetModelNameFunc()
	}
	return "mock-model"
}

// GetProviderName returns the mock provider name.
func (m *MockLLM) GetProviderName() string {
	if m.GetProviderFunc != nil {
		return m.GetProviderFunc()
	}
	return "mock-provider"
}

// GetInvokeCalls returns all inputs passed to Invoke.
func (m *MockLLM) GetInvokeCalls() []any {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.invokeCalls
}

// SetNextError sets an error to be returned on the next Invoke call.
func (m *MockLLM) SetNextError(err error) {
	m.InvokeFunc = func(ctx context.Context, input any, options ...core.Option) (any, error) {
		return nil, err
	}
}

// SetResponse sets a specific response for all Invoke calls.
func (m *MockLLM) SetResponse(response string) {
	m.InvokeFunc = func(ctx context.Context, input any, options ...core.Option) (any, error) {
		return response, nil
	}
}

// Ensure MockLLM implements llmsiface.LLM.
var _ llmsiface.LLM = (*MockLLM)(nil)

// MockChatModel is a mock implementation of llmsiface.ChatModel for testing.
type MockChatModel struct {
	*MockLLM
	GenerateFunc    func(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error)
	StreamChatFunc  func(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llmsiface.AIMessageChunk, error)
	BindToolsFunc   func(tools []core.Tool) llmsiface.ChatModel
	CheckHealthFunc func() map[string]any
	BatchFunc       func(ctx context.Context, inputs []any, options ...core.Option) ([]any, error)
	StreamFunc      func(ctx context.Context, input any, options ...core.Option) (<-chan any, error)

	mu            sync.Mutex
	generateCalls [][]schema.Message
	boundTools    []core.Tool
}

// NewMockChatModel creates a new MockChatModel with default behavior.
func NewMockChatModel() *MockChatModel {
	m := &MockChatModel{
		MockLLM: NewMockLLM(),
		GenerateFunc: func(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
			return schema.NewAIMessage("Mock AI response"), nil
		},
		StreamChatFunc: func(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llmsiface.AIMessageChunk, error) {
			ch := make(chan llmsiface.AIMessageChunk, 1)
			go func() {
				defer close(ch)
				ch <- llmsiface.AIMessageChunk{Content: "Mock streamed response"}
			}()
			return ch, nil
		},
		CheckHealthFunc: func() map[string]any {
			return map[string]any{"status": "healthy"}
		},
	}
	m.BatchFunc = func(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
		results := make([]any, len(inputs))
		for i, input := range inputs {
			result, err := m.Invoke(ctx, input, options...)
			if err != nil {
				return nil, err
			}
			results[i] = result
		}
		return results, nil
	}
	m.StreamFunc = func(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
		ch := make(chan any, 1)
		go func() {
			defer close(ch)
			result, err := m.Invoke(ctx, input, options...)
			if err != nil {
				ch <- err
			} else {
				ch <- result
			}
		}()
		return ch, nil
	}
	return m
}

// Generate calls the mock GenerateFunc.
func (m *MockChatModel) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	m.mu.Lock()
	m.generateCalls = append(m.generateCalls, messages)
	m.mu.Unlock()

	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, messages, options...)
	}
	return schema.NewAIMessage("Mock AI response"), nil
}

// StreamChat calls the mock StreamChatFunc.
func (m *MockChatModel) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llmsiface.AIMessageChunk, error) {
	if m.StreamChatFunc != nil {
		return m.StreamChatFunc(ctx, messages, options...)
	}
	ch := make(chan llmsiface.AIMessageChunk, 1)
	go func() {
		defer close(ch)
		ch <- llmsiface.AIMessageChunk{Content: "Mock streamed response"}
	}()
	return ch, nil
}

// BindTools creates a new ChatModel with tools bound.
func (m *MockChatModel) BindTools(tools []core.Tool) llmsiface.ChatModel {
	if m.BindToolsFunc != nil {
		return m.BindToolsFunc(tools)
	}
	newModel := NewMockChatModel()
	newModel.boundTools = tools
	newModel.GenerateFunc = m.GenerateFunc
	newModel.StreamChatFunc = m.StreamChatFunc
	return newModel
}

// Batch implements core.Runnable.
func (m *MockChatModel) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	if m.BatchFunc != nil {
		return m.BatchFunc(ctx, inputs, options...)
	}
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := m.Invoke(ctx, input, options...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

// Stream implements core.Runnable.
func (m *MockChatModel) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	if m.StreamFunc != nil {
		return m.StreamFunc(ctx, input, options...)
	}
	ch := make(chan any, 1)
	go func() {
		defer close(ch)
		result, err := m.Invoke(ctx, input, options...)
		if err != nil {
			ch <- err
		} else {
			ch <- result
		}
	}()
	return ch, nil
}

// CheckHealth returns the mock health status.
func (m *MockChatModel) CheckHealth() map[string]any {
	if m.CheckHealthFunc != nil {
		return m.CheckHealthFunc()
	}
	return map[string]any{"status": "healthy"}
}

// GetGenerateCalls returns all message lists passed to Generate.
func (m *MockChatModel) GetGenerateCalls() [][]schema.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.generateCalls
}

// SetGenerateResponse sets a specific response for Generate calls.
func (m *MockChatModel) SetGenerateResponse(content string) {
	m.GenerateFunc = func(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
		return schema.NewAIMessage(content), nil
	}
}

// SetGenerateError sets an error for Generate calls.
func (m *MockChatModel) SetGenerateError(err error) {
	m.GenerateFunc = func(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
		return nil, err
	}
}

// Ensure MockChatModel implements llmsiface.ChatModel.
var _ llmsiface.ChatModel = (*MockChatModel)(nil)

// MockMemory is a mock implementation of memoryiface.Memory for testing.
type MockMemory struct {
	MemoryVariablesFunc func() []string
	LoadMemoryFunc      func(ctx context.Context, inputs map[string]any) (map[string]any, error)
	SaveContextFunc     func(ctx context.Context, inputs, outputs map[string]any) error
	ClearFunc           func(ctx context.Context) error

	mu           sync.Mutex
	savedInputs  []map[string]any
	savedOutputs []map[string]any
	history      []schema.Message
}

// NewMockMemory creates a new MockMemory with default behavior.
func NewMockMemory() *MockMemory {
	return &MockMemory{
		MemoryVariablesFunc: func() []string {
			return []string{"history"}
		},
		LoadMemoryFunc: func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
			return map[string]any{}, nil
		},
		SaveContextFunc: func(ctx context.Context, inputs, outputs map[string]any) error {
			return nil
		},
		ClearFunc: func(ctx context.Context) error {
			return nil
		},
		history: []schema.Message{},
	}
}

// MemoryVariables returns the memory variable names.
func (m *MockMemory) MemoryVariables() []string {
	if m.MemoryVariablesFunc != nil {
		return m.MemoryVariablesFunc()
	}
	return []string{"history"}
}

// LoadMemoryVariables loads memory variables.
func (m *MockMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	if m.LoadMemoryFunc != nil {
		return m.LoadMemoryFunc(ctx, inputs)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return map[string]any{"history": m.history}, nil
}

// SaveContext saves context to memory.
func (m *MockMemory) SaveContext(ctx context.Context, inputs, outputs map[string]any) error {
	m.mu.Lock()
	m.savedInputs = append(m.savedInputs, inputs)
	m.savedOutputs = append(m.savedOutputs, outputs)
	m.mu.Unlock()

	if m.SaveContextFunc != nil {
		return m.SaveContextFunc(ctx, inputs, outputs)
	}
	return nil
}

// Clear clears the memory.
func (m *MockMemory) Clear(ctx context.Context) error {
	m.mu.Lock()
	m.history = []schema.Message{}
	m.savedInputs = nil
	m.savedOutputs = nil
	m.mu.Unlock()

	if m.ClearFunc != nil {
		return m.ClearFunc(ctx)
	}
	return nil
}

// GetSavedContexts returns all saved inputs and outputs.
func (m *MockMemory) GetSavedContexts() ([]map[string]any, []map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.savedInputs, m.savedOutputs
}

// SetHistory sets the mock history.
func (m *MockMemory) SetHistory(history []schema.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.history = history
}

// SetLoadError sets an error for LoadMemoryVariables.
func (m *MockMemory) SetLoadError(err error) {
	m.LoadMemoryFunc = func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
		return nil, err
	}
}

// SetSaveError sets an error for SaveContext.
func (m *MockMemory) SetSaveError(err error) {
	m.SaveContextFunc = func(ctx context.Context, inputs, outputs map[string]any) error {
		return err
	}
}

// Ensure MockMemory implements memoryiface.Memory.
var _ memoryiface.Memory = (*MockMemory)(nil)

// MockTool is a mock implementation of core.Tool for testing.
type MockTool struct {
	name        string
	description string
	ExecuteFunc func(ctx context.Context, input any) (any, error)
	BatchFunc   func(ctx context.Context, inputs []any) ([]any, error)

	mu           sync.Mutex
	executeCalls []any
}

// NewMockTool creates a new MockTool.
func NewMockTool(name, description string) *MockTool {
	return &MockTool{
		name:        name,
		description: description,
		ExecuteFunc: func(ctx context.Context, input any) (any, error) {
			return "Tool executed", nil
		},
		BatchFunc: func(ctx context.Context, inputs []any) ([]any, error) {
			results := make([]any, len(inputs))
			for i := range inputs {
				results[i] = "Tool executed"
			}
			return results, nil
		},
	}
}

// Name returns the tool name.
func (t *MockTool) Name() string {
	return t.name
}

// Description returns the tool description.
func (t *MockTool) Description() string {
	return t.description
}

// Definition returns the tool definition.
func (t *MockTool) Definition() core.ToolDefinition {
	return core.ToolDefinition{
		Name:        t.name,
		Description: t.description,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"input": map[string]any{"type": "string"},
			},
		},
	}
}

// Execute runs the tool.
func (t *MockTool) Execute(ctx context.Context, input any) (any, error) {
	t.mu.Lock()
	t.executeCalls = append(t.executeCalls, input)
	t.mu.Unlock()

	if t.ExecuteFunc != nil {
		return t.ExecuteFunc(ctx, input)
	}
	return "Tool executed", nil
}

// Batch executes the tool on multiple inputs.
func (t *MockTool) Batch(ctx context.Context, inputs []any) ([]any, error) {
	if t.BatchFunc != nil {
		return t.BatchFunc(ctx, inputs)
	}
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := t.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

// GetExecuteCalls returns all inputs passed to Execute.
func (t *MockTool) GetExecuteCalls() []any {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.executeCalls
}

// Ensure MockTool implements core.Tool.
var _ core.Tool = (*MockTool)(nil)

// Common test errors.
var (
	ErrMockLLM    = errors.New("mock LLM error")
	ErrMockMemory = errors.New("mock memory error")
	ErrMockTool   = errors.New("mock tool error")
)
