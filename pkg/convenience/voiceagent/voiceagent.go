// Package voiceagent provides a simplified API for creating voice-enabled agents.
// It reduces the boilerplate typically required to set up STT, TTS, VAD,
// and LLM integration for conversational voice applications.
//
// Note: This package is a work in progress. For production use, please use the
// individual packages directly (stt, tts, vad, llms, voicesession).
//
// Example intended usage (future):
//
//	agent, err := voiceagent.NewBuilder().
//	    WithSTT("deepgram").
//	    WithTTS("elevenlabs").
//	    WithVAD("silero").
//	    WithLLM("openai").
//	    WithMemory(true).
//	    Build(ctx)
//
//	session, err := agent.StartSession(ctx)
package voiceagent

// Builder provides a fluent interface for constructing voice agents.
// This is a placeholder for future implementation.
type Builder struct {
	sttProvider  string
	ttsProvider  string
	vadProvider  string
	llmProvider  string
	systemPrompt string
	enableMemory bool
	memorySize   int
}

// NewBuilder creates a new voice agent builder.
func NewBuilder() *Builder {
	return &Builder{
		memorySize: 50,
	}
}

// WithSTT sets the STT provider by name.
func (b *Builder) WithSTT(providerName string) *Builder {
	b.sttProvider = providerName
	return b
}

// WithTTS sets the TTS provider by name.
func (b *Builder) WithTTS(providerName string) *Builder {
	b.ttsProvider = providerName
	return b
}

// WithVAD sets the VAD provider by name.
func (b *Builder) WithVAD(providerName string) *Builder {
	b.vadProvider = providerName
	return b
}

// WithLLM sets the LLM provider by name.
func (b *Builder) WithLLM(providerName string) *Builder {
	b.llmProvider = providerName
	return b
}

// WithMemory enables conversation memory.
func (b *Builder) WithMemory(enable bool) *Builder {
	b.enableMemory = enable
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

// GetSTTProvider returns the configured STT provider.
func (b *Builder) GetSTTProvider() string {
	return b.sttProvider
}

// GetTTSProvider returns the configured TTS provider.
func (b *Builder) GetTTSProvider() string {
	return b.ttsProvider
}

// GetVADProvider returns the configured VAD provider.
func (b *Builder) GetVADProvider() string {
	return b.vadProvider
}

// GetLLMProvider returns the configured LLM provider.
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
