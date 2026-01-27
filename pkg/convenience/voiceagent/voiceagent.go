// Package voiceagent provides a simplified API for creating voice-enabled agents.
// It reduces the boilerplate typically required to set up STT, TTS, VAD,
// and LLM integration for conversational voice applications.
//
// Example usage:
//
//	agent, err := voiceagent.NewBuilder().
//	    WithSTTInstance(sttProvider).
//	    WithTTSInstance(ttsProvider).
//	    WithLLMInstance(llmModel).
//	    WithMemory(true).
//	    Build(ctx)
//
//	session, err := agent.StartSession(ctx)
package voiceagent

import (
	"context"
	"time"

	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	memoryiface "github.com/lookatitude/beluga-ai/pkg/memory/iface"
	sttiface "github.com/lookatitude/beluga-ai/pkg/stt/iface"
	ttsiface "github.com/lookatitude/beluga-ai/pkg/tts/iface"
	vadiface "github.com/lookatitude/beluga-ai/pkg/vad/iface"
)

// Builder provides a fluent interface for constructing voice agents.
type Builder struct {
	// Provider instances
	stt sttiface.STTProvider
	tts ttsiface.TTSProvider
	vad vadiface.VADProvider
	llm llmsiface.ChatModel

	// Provider names for registry lookup
	sttProvider string
	ttsProvider string
	vadProvider string
	llmProvider string

	// Memory configuration
	memory       memoryiface.Memory
	enableMemory bool
	memorySize   int

	// Agent configuration
	systemPrompt string
	timeout      time.Duration

	// Callbacks
	onTranscript TranscriptCallback
	onResponse   ResponseCallback
	onError      ErrorCallback

	// Metrics
	metrics *Metrics
}

// NewBuilder creates a new voice agent builder with sensible defaults.
func NewBuilder() *Builder {
	return &Builder{
		memorySize: 50,
		timeout:    30 * time.Second,
	}
}

// WithSTT sets the STT provider by name for registry lookup.
func (b *Builder) WithSTT(providerName string) *Builder {
	b.sttProvider = providerName
	return b
}

// WithSTTInstance sets the STT provider instance directly.
func (b *Builder) WithSTTInstance(stt sttiface.STTProvider) *Builder {
	b.stt = stt
	return b
}

// WithTTS sets the TTS provider by name for registry lookup.
func (b *Builder) WithTTS(providerName string) *Builder {
	b.ttsProvider = providerName
	return b
}

// WithTTSInstance sets the TTS provider instance directly.
func (b *Builder) WithTTSInstance(tts ttsiface.TTSProvider) *Builder {
	b.tts = tts
	return b
}

// WithVAD sets the VAD provider by name for registry lookup.
func (b *Builder) WithVAD(providerName string) *Builder {
	b.vadProvider = providerName
	return b
}

// WithVADInstance sets the VAD provider instance directly.
func (b *Builder) WithVADInstance(vad vadiface.VADProvider) *Builder {
	b.vad = vad
	return b
}

// WithLLM sets the LLM provider by name for registry lookup.
func (b *Builder) WithLLM(providerName string) *Builder {
	b.llmProvider = providerName
	return b
}

// WithLLMInstance sets the LLM instance directly.
func (b *Builder) WithLLMInstance(llm llmsiface.ChatModel) *Builder {
	b.llm = llm
	return b
}

// WithMemory enables conversation memory.
func (b *Builder) WithMemory(enable bool) *Builder {
	b.enableMemory = enable
	return b
}

// WithMemoryInstance sets a custom memory implementation.
func (b *Builder) WithMemoryInstance(mem memoryiface.Memory) *Builder {
	b.memory = mem
	b.enableMemory = true
	return b
}

// WithMemorySize sets the memory size (number of messages to retain).
func (b *Builder) WithMemorySize(size int) *Builder {
	b.memorySize = size
	return b
}

// WithSystemPrompt sets the system prompt for the voice agent.
func (b *Builder) WithSystemPrompt(prompt string) *Builder {
	b.systemPrompt = prompt
	return b
}

// WithTimeout sets the timeout for operations.
func (b *Builder) WithTimeout(timeout time.Duration) *Builder {
	b.timeout = timeout
	return b
}

// WithOnTranscript sets the callback for transcript events.
func (b *Builder) WithOnTranscript(fn TranscriptCallback) *Builder {
	b.onTranscript = fn
	return b
}

// WithOnResponse sets the callback for response events.
func (b *Builder) WithOnResponse(fn ResponseCallback) *Builder {
	b.onResponse = fn
	return b
}

// WithOnError sets the callback for error events.
func (b *Builder) WithOnError(fn ErrorCallback) *Builder {
	b.onError = fn
	return b
}

// WithMetrics sets a custom metrics instance.
func (b *Builder) WithMetrics(m *Metrics) *Builder {
	b.metrics = m
	return b
}

// Build creates a new VoiceAgent based on the builder configuration.
func (b *Builder) Build(ctx context.Context) (VoiceAgent, error) {
	const op = "voiceagent.Builder.Build"

	// Get or create metrics
	metrics := b.metrics
	if metrics == nil {
		metrics = GetMetrics()
		if metrics == nil {
			metrics = NoOpMetrics()
		}
	}

	// Start build span
	ctx, span := metrics.StartBuildSpan(ctx)
	if span != nil {
		defer span.End()
	}

	// Validate required fields
	if b.stt == nil && b.sttProvider == "" {
		metrics.RecordBuild(ctx, false)
		return nil, NewError(op, ErrCodeMissingSTT, ErrMissingSTT)
	}
	if b.tts == nil && b.ttsProvider == "" {
		metrics.RecordBuild(ctx, false)
		return nil, NewError(op, ErrCodeMissingTTS, ErrMissingTTS)
	}

	// Resolve STT if provider name given (registry lookup would go here)
	stt := b.stt
	if stt == nil && b.sttProvider != "" {
		// For now, return error - registry integration would resolve the provider
		metrics.RecordBuild(ctx, false)
		return nil, NewErrorWithMessage(op, ErrCodeSTTCreation, "STT provider registry lookup not yet implemented", nil).
			WithField("provider", b.sttProvider)
	}

	// Resolve TTS if provider name given
	tts := b.tts
	if tts == nil && b.ttsProvider != "" {
		// For now, return error - registry integration would resolve the provider
		metrics.RecordBuild(ctx, false)
		return nil, NewErrorWithMessage(op, ErrCodeTTSCreation, "TTS provider registry lookup not yet implemented", nil).
			WithField("provider", b.ttsProvider)
	}

	// Resolve VAD if provider name given (optional)
	vad := b.vad
	if vad == nil && b.vadProvider != "" {
		// For now, return error - registry integration would resolve the provider
		metrics.RecordBuild(ctx, false)
		return nil, NewErrorWithMessage(op, ErrCodeVADCreation, "VAD provider registry lookup not yet implemented", nil).
			WithField("provider", b.vadProvider)
	}

	// Resolve LLM if provider name given (optional)
	llm := b.llm
	if llm == nil && b.llmProvider != "" {
		// For now, return error - registry integration would resolve the provider
		metrics.RecordBuild(ctx, false)
		return nil, NewErrorWithMessage(op, ErrCodeAgentCreation, "LLM provider registry lookup not yet implemented", nil).
			WithField("provider", b.llmProvider)
	}

	// Create memory if enabled but no instance provided
	memory := b.memory
	if b.enableMemory && memory == nil {
		// Memory would be created here with memorySize
		// For now, we just note that memory is enabled
	}

	// Create the voice agent
	agent := &convenienceVoiceAgent{
		stt:          stt,
		tts:          tts,
		vad:          vad,
		llm:          llm,
		memory:       memory,
		systemPrompt: b.systemPrompt,
		timeout:      b.timeout,
		onTranscript: b.onTranscript,
		onResponse:   b.onResponse,
		onError:      b.onError,
		metrics:      metrics,
	}

	metrics.RecordBuild(ctx, true)
	return agent, nil
}

// Getters for builder inspection

// GetSTTProvider returns the configured STT provider name.
func (b *Builder) GetSTTProvider() string {
	return b.sttProvider
}

// GetTTSProvider returns the configured TTS provider name.
func (b *Builder) GetTTSProvider() string {
	return b.ttsProvider
}

// GetVADProvider returns the configured VAD provider name.
func (b *Builder) GetVADProvider() string {
	return b.vadProvider
}

// GetLLMProvider returns the configured LLM provider name.
func (b *Builder) GetLLMProvider() string {
	return b.llmProvider
}

// GetSystemPrompt returns the configured system prompt.
func (b *Builder) GetSystemPrompt() string {
	return b.systemPrompt
}

// IsMemoryEnabled returns whether memory is enabled.
func (b *Builder) IsMemoryEnabled() bool {
	return b.enableMemory
}

// GetMemorySize returns the configured memory size.
func (b *Builder) GetMemorySize() int {
	return b.memorySize
}

// GetTimeout returns the configured timeout.
func (b *Builder) GetTimeout() time.Duration {
	return b.timeout
}
