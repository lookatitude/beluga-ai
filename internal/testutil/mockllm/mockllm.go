package mockllm

import (
	"context"
	"iter"
	"sync"

	"github.com/lookatitude/beluga-ai/schema"
)

// GenerateOption mirrors the llm.GenerateOption type so this package does not
// import the llm package (avoiding circular dependencies).
type GenerateOption func(any)

// MockChatModel is a configurable mock for the ChatModel interface.
// It records all Generate calls and can return preset responses, errors,
// or streaming chunks.
type MockChatModel struct {
	mu sync.Mutex

	response     *schema.AIMessage
	err          error
	streamChunks []schema.StreamChunk
	modelID      string
	boundTools   []schema.ToolDefinition

	generateCalls int
	lastMessages  []schema.Message
}

// Option configures a MockChatModel.
type Option func(*MockChatModel)

// New creates a MockChatModel with the given options.
func New(opts ...Option) *MockChatModel {
	m := &MockChatModel{
		modelID: "mock-model",
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// WithResponse configures the mock to return the given AIMessage from Generate.
func WithResponse(msg *schema.AIMessage) Option {
	return func(m *MockChatModel) {
		m.response = msg
	}
}

// WithError configures the mock to return the given error from Generate and Stream.
func WithError(err error) Option {
	return func(m *MockChatModel) {
		m.err = err
	}
}

// WithStreamChunks configures the mock to yield the given chunks from Stream.
func WithStreamChunks(chunks []schema.StreamChunk) Option {
	return func(m *MockChatModel) {
		m.streamChunks = chunks
	}
}

// WithModelID sets the model identifier returned by ModelID.
func WithModelID(id string) Option {
	return func(m *MockChatModel) {
		m.modelID = id
	}
}

// Generate returns the configured response or error. It records the call
// and the messages for later inspection.
func (m *MockChatModel) Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.generateCalls++
	m.lastMessages = make([]schema.Message, len(msgs))
	copy(m.lastMessages, msgs)

	if m.err != nil {
		return nil, m.err
	}
	if m.response != nil {
		return m.response, nil
	}

	// Default: return an empty AIMessage.
	return &schema.AIMessage{}, nil
}

// Stream returns an iter.Seq2 that yields the configured stream chunks.
// If an error is configured, the first yield returns that error.
func (m *MockChatModel) Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	m.mu.Lock()
	m.generateCalls++
	m.lastMessages = make([]schema.Message, len(msgs))
	copy(m.lastMessages, msgs)
	chunks := m.streamChunks
	streamErr := m.err
	m.mu.Unlock()

	return func(yield func(schema.StreamChunk, error) bool) {
		if streamErr != nil {
			yield(schema.StreamChunk{}, streamErr)
			return
		}
		for _, chunk := range chunks {
			if ctx.Err() != nil {
				yield(schema.StreamChunk{}, ctx.Err())
				return
			}
			if !yield(chunk, nil) {
				return
			}
		}
	}
}

// BindTools returns a new MockChatModel with the given tools recorded.
// The returned mock shares the same response/error configuration.
func (m *MockChatModel) BindTools(tools []schema.ToolDefinition) *MockChatModel {
	m.mu.Lock()
	defer m.mu.Unlock()

	clone := &MockChatModel{
		response:     m.response,
		err:          m.err,
		streamChunks: m.streamChunks,
		modelID:      m.modelID,
		boundTools:   tools,
	}
	return clone
}

// ModelID returns the configured model identifier.
func (m *MockChatModel) ModelID() string {
	return m.modelID
}

// GenerateCalls returns the number of times Generate or Stream has been called.
func (m *MockChatModel) GenerateCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.generateCalls
}

// LastMessages returns the messages passed to the most recent Generate or
// Stream call.
func (m *MockChatModel) LastMessages() []schema.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]schema.Message, len(m.lastMessages))
	copy(result, m.lastMessages)
	return result
}

// BoundTools returns the tool definitions set via BindTools.
func (m *MockChatModel) BoundTools() []schema.ToolDefinition {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.boundTools
}

// SetResponse updates the canned response for subsequent calls.
func (m *MockChatModel) SetResponse(msg *schema.AIMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.response = msg
	m.err = nil
}

// SetError updates the error for subsequent calls.
func (m *MockChatModel) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
	m.response = nil
}

// Reset clears all recorded calls and configuration.
func (m *MockChatModel) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.generateCalls = 0
	m.lastMessages = nil
	m.response = nil
	m.err = nil
	m.streamChunks = nil
	m.boundTools = nil
}
