// Package mock provides centralized mock implementations for testing.
// It offers easy-to-use mock factories for all major interfaces in the Beluga AI Framework,
// reducing boilerplate in test code.
//
// Example usage:
//
//	llm := mock.NewLLM(mock.WithResponse("Hello!"))
//	embedder := mock.NewEmbedder(mock.WithDimension(1536))
//	stt := mock.NewSTT(mock.WithTranscription("transcribed text"))
package mock

import (
	"context"
	"sync/atomic"

	"github.com/lookatitude/beluga-ai/pkg/core"
)

// LLM provides a mock LLM implementation for testing.
type LLM struct {
	response      string
	errorResponse error
	callCount     int64
	modelName     string
	providerName  string
}

// LLMOption configures the mock LLM.
type LLMOption func(*LLM)

// NewLLM creates a new mock LLM with the given options.
func NewLLM(opts ...LLMOption) *LLM {
	m := &LLM{
		response:     "mock response",
		modelName:    "mock-model",
		providerName: "mock",
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// WithResponse sets the mock response.
func WithResponse(response string) LLMOption {
	return func(m *LLM) {
		m.response = response
	}
}

// WithError sets an error to return.
func WithError(err error) LLMOption {
	return func(m *LLM) {
		m.errorResponse = err
	}
}

// WithModelName sets the model name.
func WithModelName(name string) LLMOption {
	return func(m *LLM) {
		m.modelName = name
	}
}

// Invoke implements the LLM interface.
func (m *LLM) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	atomic.AddInt64(&m.callCount, 1)
	if m.errorResponse != nil {
		return nil, m.errorResponse
	}
	return m.response, nil
}

// GetModelName returns the model name.
func (m *LLM) GetModelName() string {
	return m.modelName
}

// GetProviderName returns the provider name.
func (m *LLM) GetProviderName() string {
	return m.providerName
}

// CallCount returns the number of calls made.
func (m *LLM) CallCount() int {
	return int(atomic.LoadInt64(&m.callCount))
}

// Reset resets the call count.
func (m *LLM) Reset() {
	atomic.StoreInt64(&m.callCount, 0)
}

// Embedder provides a mock embedder implementation for testing.
type Embedder struct {
	dimension     int
	errorResponse error
	callCount     int64
}

// EmbedderOption configures the mock embedder.
type EmbedderOption func(*Embedder)

// NewEmbedder creates a new mock embedder with the given options.
func NewEmbedder(opts ...EmbedderOption) *Embedder {
	e := &Embedder{
		dimension: 1536,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// WithDimension sets the embedding dimension.
func WithDimension(dim int) EmbedderOption {
	return func(e *Embedder) {
		e.dimension = dim
	}
}

// WithEmbedderError sets an error to return.
func WithEmbedderError(err error) EmbedderOption {
	return func(e *Embedder) {
		e.errorResponse = err
	}
}

// EmbedDocuments embeds multiple documents.
func (e *Embedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	atomic.AddInt64(&e.callCount, 1)
	if e.errorResponse != nil {
		return nil, e.errorResponse
	}
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = make([]float32, e.dimension)
		for j := range result[i] {
			result[i][j] = 0.1
		}
	}
	return result, nil
}

// EmbedQuery embeds a single query.
func (e *Embedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	atomic.AddInt64(&e.callCount, 1)
	if e.errorResponse != nil {
		return nil, e.errorResponse
	}
	result := make([]float32, e.dimension)
	for i := range result {
		result[i] = 0.1
	}
	return result, nil
}

// CallCount returns the number of calls made.
func (e *Embedder) CallCount() int {
	return int(atomic.LoadInt64(&e.callCount))
}

// Reset resets the call count.
func (e *Embedder) Reset() {
	atomic.StoreInt64(&e.callCount, 0)
}

// STT provides a mock speech-to-text implementation for testing.
type STT struct {
	transcription string
	errorResponse error
	callCount     int64
}

// STTOption configures the mock STT.
type STTOption func(*STT)

// NewSTT creates a new mock STT with the given options.
func NewSTT(opts ...STTOption) *STT {
	s := &STT{
		transcription: "mock transcription",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// WithTranscription sets the mock transcription.
func WithTranscription(text string) STTOption {
	return func(s *STT) {
		s.transcription = text
	}
}

// WithSTTError sets an error to return.
func WithSTTError(err error) STTOption {
	return func(s *STT) {
		s.errorResponse = err
	}
}

// Transcribe transcribes audio data.
func (s *STT) Transcribe(ctx context.Context, audio []byte) (string, error) {
	atomic.AddInt64(&s.callCount, 1)
	if s.errorResponse != nil {
		return "", s.errorResponse
	}
	return s.transcription, nil
}

// CallCount returns the number of calls made.
func (s *STT) CallCount() int {
	return int(atomic.LoadInt64(&s.callCount))
}

// Reset resets the call count.
func (s *STT) Reset() {
	atomic.StoreInt64(&s.callCount, 0)
}

// TTS provides a mock text-to-speech implementation for testing.
type TTS struct {
	audioData     []byte
	errorResponse error
	callCount     int64
}

// TTSOption configures the mock TTS.
type TTSOption func(*TTS)

// NewTTS creates a new mock TTS with the given options.
func NewTTS(opts ...TTSOption) *TTS {
	t := &TTS{
		audioData: []byte("mock audio data"),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// WithAudioData sets the mock audio data.
func WithAudioData(data []byte) TTSOption {
	return func(t *TTS) {
		t.audioData = data
	}
}

// WithTTSError sets an error to return.
func WithTTSError(err error) TTSOption {
	return func(t *TTS) {
		t.errorResponse = err
	}
}

// Synthesize synthesizes speech from text.
func (t *TTS) Synthesize(ctx context.Context, text string) ([]byte, error) {
	atomic.AddInt64(&t.callCount, 1)
	if t.errorResponse != nil {
		return nil, t.errorResponse
	}
	return t.audioData, nil
}

// CallCount returns the number of calls made.
func (t *TTS) CallCount() int {
	return int(atomic.LoadInt64(&t.callCount))
}

// Reset resets the call count.
func (t *TTS) Reset() {
	atomic.StoreInt64(&t.callCount, 0)
}

// Tool provides a mock tool implementation for testing.
type Tool struct {
	name          string
	description   string
	result        any
	errorResponse error
	callCount     int64
}

// ToolOption configures the mock tool.
type ToolOption func(*Tool)

// NewTool creates a new mock tool with the given options.
func NewTool(name, description string, opts ...ToolOption) *Tool {
	t := &Tool{
		name:        name,
		description: description,
		result:      "mock result",
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// WithResult sets the mock result.
func WithResult(result any) ToolOption {
	return func(t *Tool) {
		t.result = result
	}
}

// WithToolError sets an error to return.
func WithToolError(err error) ToolOption {
	return func(t *Tool) {
		t.errorResponse = err
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return t.name
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return t.description
}

// Execute executes the tool.
func (t *Tool) Execute(ctx context.Context, input any) (any, error) {
	atomic.AddInt64(&t.callCount, 1)
	if t.errorResponse != nil {
		return nil, t.errorResponse
	}
	return t.result, nil
}

// CallCount returns the number of calls made.
func (t *Tool) CallCount() int {
	return int(atomic.LoadInt64(&t.callCount))
}

// Reset resets the call count.
func (t *Tool) Reset() {
	atomic.StoreInt64(&t.callCount, 0)
}
