package voiceagent

import (
	"context"
	"io"
	"strings"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	memoryiface "github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	voiceiface "github.com/lookatitude/beluga-ai/pkg/voiceutils/iface"
)

// MockSTTProvider is a mock implementation of STTProvider for testing.
type MockSTTProvider struct {
	TranscribeFunc     func(ctx context.Context, audio []byte) (string, error)
	StartStreamingFunc func(ctx context.Context) (voiceiface.StreamingSession, error)

	mu              sync.Mutex
	TranscribeCalls int
}

// NewMockSTT creates a new mock STT provider with default behavior.
func NewMockSTT() *MockSTTProvider {
	return &MockSTTProvider{
		TranscribeFunc: func(_ context.Context, _ []byte) (string, error) {
			return "Hello, this is a transcription.", nil
		},
	}
}

// Transcribe implements STTProvider.
func (m *MockSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	m.mu.Lock()
	m.TranscribeCalls++
	m.mu.Unlock()

	if m.TranscribeFunc != nil {
		return m.TranscribeFunc(ctx, audio)
	}
	return "", nil
}

// StartStreaming implements STTProvider.
func (m *MockSTTProvider) StartStreaming(ctx context.Context) (voiceiface.StreamingSession, error) {
	if m.StartStreamingFunc != nil {
		return m.StartStreamingFunc(ctx)
	}
	return &MockStreamingSession{}, nil
}

// SetTranscription sets a fixed transcription response.
func (m *MockSTTProvider) SetTranscription(text string) {
	m.TranscribeFunc = func(_ context.Context, _ []byte) (string, error) {
		return text, nil
	}
}

// MockStreamingSession is a mock streaming session for testing.
type MockStreamingSession struct {
	SendAudioFunc         func(ctx context.Context, audio []byte) error
	ReceiveTranscriptFunc func() <-chan voiceiface.TranscriptResult
	CloseFunc             func() error

	transcriptCh chan voiceiface.TranscriptResult
}

// SendAudio implements StreamingSession.
func (m *MockStreamingSession) SendAudio(ctx context.Context, audio []byte) error {
	if m.SendAudioFunc != nil {
		return m.SendAudioFunc(ctx, audio)
	}
	return nil
}

// ReceiveTranscript implements StreamingSession.
func (m *MockStreamingSession) ReceiveTranscript() <-chan voiceiface.TranscriptResult {
	if m.ReceiveTranscriptFunc != nil {
		return m.ReceiveTranscriptFunc()
	}
	if m.transcriptCh == nil {
		m.transcriptCh = make(chan voiceiface.TranscriptResult)
	}
	return m.transcriptCh
}

// Close implements StreamingSession.
func (m *MockStreamingSession) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// MockTTSProvider is a mock implementation of TTSProvider for testing.
type MockTTSProvider struct {
	GenerateSpeechFunc func(ctx context.Context, text string) ([]byte, error)
	StreamGenerateFunc func(ctx context.Context, text string) (io.Reader, error)

	mu                  sync.Mutex
	GenerateSpeechCalls int
}

// NewMockTTS creates a new mock TTS provider with default behavior.
func NewMockTTS() *MockTTSProvider {
	return &MockTTSProvider{
		GenerateSpeechFunc: func(_ context.Context, text string) ([]byte, error) {
			return []byte("audio:" + text), nil
		},
	}
}

// GenerateSpeech implements TTSProvider.
func (m *MockTTSProvider) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	m.mu.Lock()
	m.GenerateSpeechCalls++
	m.mu.Unlock()

	if m.GenerateSpeechFunc != nil {
		return m.GenerateSpeechFunc(ctx, text)
	}
	return nil, nil
}

// StreamGenerate implements TTSProvider.
func (m *MockTTSProvider) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	if m.StreamGenerateFunc != nil {
		return m.StreamGenerateFunc(ctx, text)
	}
	return strings.NewReader("audio:" + text), nil
}

// SetGenerateResponse sets a fixed audio response.
func (m *MockTTSProvider) SetGenerateResponse(audio []byte) {
	m.GenerateSpeechFunc = func(_ context.Context, _ string) ([]byte, error) {
		return audio, nil
	}
}

// MockVADProvider is a mock implementation of VADProvider for testing.
type MockVADProvider struct {
	ProcessFunc       func(ctx context.Context, audio []byte) (bool, error)
	ProcessStreamFunc func(ctx context.Context, audioCh <-chan []byte) (<-chan voiceiface.VADResult, error)

	mu           sync.Mutex
	ProcessCalls int
}

// NewMockVAD creates a new mock VAD provider with default behavior.
func NewMockVAD() *MockVADProvider {
	return &MockVADProvider{
		ProcessFunc: func(_ context.Context, _ []byte) (bool, error) {
			return true, nil // Always detect voice
		},
	}
}

// Process implements VADProvider.
func (m *MockVADProvider) Process(ctx context.Context, audio []byte) (bool, error) {
	m.mu.Lock()
	m.ProcessCalls++
	m.mu.Unlock()

	if m.ProcessFunc != nil {
		return m.ProcessFunc(ctx, audio)
	}
	return true, nil
}

// ProcessStream implements VADProvider.
func (m *MockVADProvider) ProcessStream(ctx context.Context, audioCh <-chan []byte) (<-chan voiceiface.VADResult, error) {
	if m.ProcessStreamFunc != nil {
		return m.ProcessStreamFunc(ctx, audioCh)
	}

	resultCh := make(chan voiceiface.VADResult)
	go func() {
		defer close(resultCh)
		for range audioCh {
			select {
			case resultCh <- voiceiface.VADResult{HasVoice: true, Confidence: 0.9}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return resultCh, nil
}

// SetVoiceDetection sets a fixed voice detection result.
func (m *MockVADProvider) SetVoiceDetection(hasVoice bool) {
	m.ProcessFunc = func(_ context.Context, _ []byte) (bool, error) {
		return hasVoice, nil
	}
}

// MockChatModel is a mock implementation of ChatModel for testing.
type MockChatModel struct {
	InvokeFunc     func(ctx context.Context, input any, options ...core.Option) (any, error)
	GenerateFunc   func(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error)
	StreamChatFunc func(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llmsiface.AIMessageChunk, error)
	BindToolsFunc  func(toolsToBind []core.Tool) llmsiface.ChatModel

	mu            sync.Mutex
	GenerateCalls int
	LastMessages  []schema.Message
}

// NewMockChatModel creates a new mock chat model with default behavior.
func NewMockChatModel() *MockChatModel {
	return &MockChatModel{
		GenerateFunc: func(_ context.Context, _ []schema.Message, _ ...core.Option) (schema.Message, error) {
			return schema.NewAIMessage("This is a mock response."), nil
		},
	}
}

// Invoke implements ChatModel.
func (m *MockChatModel) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	if m.InvokeFunc != nil {
		return m.InvokeFunc(ctx, input, options...)
	}
	// Default: treat input as messages and generate
	if messages, ok := input.([]schema.Message); ok {
		msg, err := m.Generate(ctx, messages, options...)
		if err != nil {
			return nil, err
		}
		return msg.GetContent(), nil
	}
	return "mock response", nil
}

// Batch implements core.Runnable.
func (m *MockChatModel) Batch(_ context.Context, _ []any, _ ...core.Option) ([]any, error) {
	return []any{"batch response"}, nil
}

// Stream implements core.Runnable.
func (m *MockChatModel) Stream(_ context.Context, _ any, _ ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	ch <- "stream response"
	close(ch)
	return ch, nil
}

// Generate implements ChatModel.
func (m *MockChatModel) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	m.mu.Lock()
	m.GenerateCalls++
	m.LastMessages = messages
	m.mu.Unlock()

	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, messages)
	}
	return schema.NewAIMessage("mock response"), nil
}

// StreamChat implements ChatModel.
func (m *MockChatModel) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llmsiface.AIMessageChunk, error) {
	if m.StreamChatFunc != nil {
		return m.StreamChatFunc(ctx, messages, options...)
	}
	ch := make(chan llmsiface.AIMessageChunk, 1)
	ch <- llmsiface.AIMessageChunk{Content: "mock stream response"}
	close(ch)
	return ch, nil
}

// BindTools implements ChatModel.
func (m *MockChatModel) BindTools(toolsToBind []core.Tool) llmsiface.ChatModel {
	if m.BindToolsFunc != nil {
		return m.BindToolsFunc(toolsToBind)
	}
	return m
}

// GetModelName implements ChatModel.
func (m *MockChatModel) GetModelName() string {
	return "mock-model"
}

// GetProviderName implements LLM.
func (m *MockChatModel) GetProviderName() string {
	return "mock"
}

// CheckHealth implements ChatModel.
func (m *MockChatModel) CheckHealth() map[string]any {
	return map[string]any{"status": "healthy"}
}

// SetGenerateResponse sets a fixed response for Generate calls.
func (m *MockChatModel) SetGenerateResponse(response string) {
	m.GenerateFunc = func(_ context.Context, _ []schema.Message, _ ...core.Option) (schema.Message, error) {
		return schema.NewAIMessage(response), nil
	}
}

// MockMemory is a mock implementation of Memory for testing.
type MockMemory struct {
	MemoryVariablesFunc     func() []string
	LoadMemoryVariablesFunc func(ctx context.Context, inputs map[string]any) (map[string]any, error)
	SaveContextFunc         func(ctx context.Context, inputs, outputs map[string]any) error
	ClearFunc               func(ctx context.Context) error

	mu        sync.Mutex
	SaveCalls int
	History   []schema.Message
}

// NewMockMemory creates a new mock memory with default behavior.
func NewMockMemory() *MockMemory {
	return &MockMemory{
		History: []schema.Message{},
	}
}

// MemoryVariables implements Memory.
func (m *MockMemory) MemoryVariables() []string {
	if m.MemoryVariablesFunc != nil {
		return m.MemoryVariablesFunc()
	}
	return []string{"history"}
}

// LoadMemoryVariables implements Memory.
func (m *MockMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	if m.LoadMemoryVariablesFunc != nil {
		return m.LoadMemoryVariablesFunc(ctx, inputs)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return map[string]any{"history": m.History}, nil
}

// SaveContext implements Memory.
func (m *MockMemory) SaveContext(ctx context.Context, inputs, outputs map[string]any) error {
	m.mu.Lock()
	m.SaveCalls++
	if input, ok := inputs["input"].(string); ok {
		m.History = append(m.History, schema.NewHumanMessage(input))
	}
	if output, ok := outputs["output"].(string); ok {
		m.History = append(m.History, schema.NewAIMessage(output))
	}
	m.mu.Unlock()

	if m.SaveContextFunc != nil {
		return m.SaveContextFunc(ctx, inputs, outputs)
	}
	return nil
}

// Clear implements Memory.
func (m *MockMemory) Clear(ctx context.Context) error {
	if m.ClearFunc != nil {
		return m.ClearFunc(ctx)
	}
	m.mu.Lock()
	m.History = []schema.Message{}
	m.mu.Unlock()
	return nil
}

// SetHistory sets the conversation history.
func (m *MockMemory) SetHistory(messages []schema.Message) {
	m.mu.Lock()
	m.History = messages
	m.mu.Unlock()
}

// Compile-time interface checks.
var (
	_ voiceiface.STTProvider      = (*MockSTTProvider)(nil)
	_ voiceiface.StreamingSession = (*MockStreamingSession)(nil)
	_ voiceiface.TTSProvider      = (*MockTTSProvider)(nil)
	_ voiceiface.VADProvider      = (*MockVADProvider)(nil)
	_ llmsiface.ChatModel         = (*MockChatModel)(nil)
	_ memoryiface.Memory          = (*MockMemory)(nil)
)
