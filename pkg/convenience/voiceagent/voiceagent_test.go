package voiceagent

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder()

	if builder.memorySize != 50 {
		t.Errorf("expected default memorySize 50, got %d", builder.memorySize)
	}
	if builder.timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", builder.timeout)
	}
}

func TestBuilder_WithMethods(t *testing.T) {
	mockSTT := NewMockSTT()
	mockTTS := NewMockTTS()
	mockVAD := NewMockVAD()
	mockLLM := NewMockChatModel()
	mockMemory := NewMockMemory()

	var transcriptCalled bool
	var responseCalled bool
	var errorCalled bool

	builder := NewBuilder().
		WithSTT("deepgram").
		WithSTTInstance(mockSTT).
		WithTTS("elevenlabs").
		WithTTSInstance(mockTTS).
		WithVAD("silero").
		WithVADInstance(mockVAD).
		WithLLM("openai").
		WithLLMInstance(mockLLM).
		WithMemory(true).
		WithMemoryInstance(mockMemory).
		WithMemorySize(100).
		WithSystemPrompt("You are a helpful assistant.").
		WithTimeout(60 * time.Second).
		WithOnTranscript(func(text string, isFinal bool) { transcriptCalled = true }).
		WithOnResponse(func(text string) { responseCalled = true }).
		WithOnError(func(err error) { errorCalled = true })

	if builder.sttProvider != "deepgram" {
		t.Errorf("expected sttProvider 'deepgram', got %s", builder.sttProvider)
	}
	if builder.stt != mockSTT {
		t.Error("expected stt instance to be set")
	}
	if builder.ttsProvider != "elevenlabs" {
		t.Errorf("expected ttsProvider 'elevenlabs', got %s", builder.ttsProvider)
	}
	if builder.tts != mockTTS {
		t.Error("expected tts instance to be set")
	}
	if builder.vadProvider != "silero" {
		t.Errorf("expected vadProvider 'silero', got %s", builder.vadProvider)
	}
	if builder.vad != mockVAD {
		t.Error("expected vad instance to be set")
	}
	if builder.llmProvider != "openai" {
		t.Errorf("expected llmProvider 'openai', got %s", builder.llmProvider)
	}
	if builder.llm != mockLLM {
		t.Error("expected llm instance to be set")
	}
	if !builder.enableMemory {
		t.Error("expected memory to be enabled")
	}
	if builder.memory != mockMemory {
		t.Error("expected memory instance to be set")
	}
	if builder.memorySize != 100 {
		t.Errorf("expected memorySize 100, got %d", builder.memorySize)
	}
	if builder.systemPrompt != "You are a helpful assistant." {
		t.Errorf("expected systemPrompt 'You are a helpful assistant.', got %s", builder.systemPrompt)
	}
	if builder.timeout != 60*time.Second {
		t.Errorf("expected timeout 60s, got %v", builder.timeout)
	}

	// Test callbacks are set
	if builder.onTranscript == nil {
		t.Error("expected onTranscript callback to be set")
	}
	if builder.onResponse == nil {
		t.Error("expected onResponse callback to be set")
	}
	if builder.onError == nil {
		t.Error("expected onError callback to be set")
	}

	// Verify callbacks can be called
	builder.onTranscript("test", true)
	builder.onResponse("test")
	builder.onError(nil)

	if !transcriptCalled {
		t.Error("expected transcriptCalled to be true")
	}
	if !responseCalled {
		t.Error("expected responseCalled to be true")
	}
	if !errorCalled {
		t.Error("expected errorCalled to be true")
	}
}

func TestBuilder_Build_MissingSTT(t *testing.T) {
	mockTTS := NewMockTTS()

	builder := NewBuilder().
		WithTTSInstance(mockTTS)

	_, err := builder.Build(context.Background())
	if err == nil {
		t.Error("expected error for missing STT")
	}

	var vaErr *Error
	if !errors.As(err, &vaErr) {
		t.Fatal("expected *Error type")
	}
	if vaErr.Code != ErrCodeMissingSTT {
		t.Errorf("expected error code %s, got %s", ErrCodeMissingSTT, vaErr.Code)
	}
}

func TestBuilder_Build_MissingTTS(t *testing.T) {
	mockSTT := NewMockSTT()

	builder := NewBuilder().
		WithSTTInstance(mockSTT)

	_, err := builder.Build(context.Background())
	if err == nil {
		t.Error("expected error for missing TTS")
	}

	var vaErr *Error
	if !errors.As(err, &vaErr) {
		t.Fatal("expected *Error type")
	}
	if vaErr.Code != ErrCodeMissingTTS {
		t.Errorf("expected error code %s, got %s", ErrCodeMissingTTS, vaErr.Code)
	}
}

func TestBuilder_Build_WithInstances(t *testing.T) {
	mockSTT := NewMockSTT()
	mockTTS := NewMockTTS()

	agent, err := NewBuilder().
		WithSTTInstance(mockSTT).
		WithTTSInstance(mockTTS).
		Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent == nil {
		t.Fatal("expected agent to be non-nil")
	}

	// Verify providers are accessible
	if agent.GetSTT() != mockSTT {
		t.Error("expected STT provider to match")
	}
	if agent.GetTTS() != mockTTS {
		t.Error("expected TTS provider to match")
	}
	if agent.GetVAD() != nil {
		t.Error("expected VAD provider to be nil when not set")
	}
}

func TestBuilder_Build_WithAllProviders(t *testing.T) {
	mockSTT := NewMockSTT()
	mockTTS := NewMockTTS()
	mockVAD := NewMockVAD()
	mockLLM := NewMockChatModel()

	agent, err := NewBuilder().
		WithSTTInstance(mockSTT).
		WithTTSInstance(mockTTS).
		WithVADInstance(mockVAD).
		WithLLMInstance(mockLLM).
		WithSystemPrompt("Test prompt").
		Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent == nil {
		t.Fatal("expected agent to be non-nil")
	}

	if agent.GetSTT() != mockSTT {
		t.Error("expected STT provider to match")
	}
	if agent.GetTTS() != mockTTS {
		t.Error("expected TTS provider to match")
	}
	if agent.GetVAD() != mockVAD {
		t.Error("expected VAD provider to match")
	}
}

func TestBuilder_Build_ProviderNameNotImplemented(t *testing.T) {
	tests := []struct {
		name         string
		buildFunc    func() *Builder
		expectedCode string
	}{
		{
			name: "STT provider name",
			buildFunc: func() *Builder {
				return NewBuilder().
					WithSTT("deepgram").
					WithTTSInstance(NewMockTTS())
			},
			expectedCode: ErrCodeSTTCreation,
		},
		{
			name: "TTS provider name",
			buildFunc: func() *Builder {
				return NewBuilder().
					WithSTTInstance(NewMockSTT()).
					WithTTS("elevenlabs")
			},
			expectedCode: ErrCodeTTSCreation,
		},
		{
			name: "VAD provider name",
			buildFunc: func() *Builder {
				return NewBuilder().
					WithSTTInstance(NewMockSTT()).
					WithTTSInstance(NewMockTTS()).
					WithVAD("silero")
			},
			expectedCode: ErrCodeVADCreation,
		},
		{
			name: "LLM provider name",
			buildFunc: func() *Builder {
				return NewBuilder().
					WithSTTInstance(NewMockSTT()).
					WithTTSInstance(NewMockTTS()).
					WithLLM("openai")
			},
			expectedCode: ErrCodeAgentCreation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := tt.buildFunc()
			_, err := builder.Build(context.Background())
			if err == nil {
				t.Error("expected error for provider name lookup")
			}

			code := GetErrorCode(err)
			if code != tt.expectedCode {
				t.Errorf("expected error code %s, got %s", tt.expectedCode, code)
			}
		})
	}
}

func TestBuilder_Getters(t *testing.T) {
	builder := NewBuilder().
		WithSTT("deepgram").
		WithTTS("elevenlabs").
		WithVAD("silero").
		WithLLM("openai").
		WithSystemPrompt("Test prompt").
		WithMemory(true).
		WithMemorySize(75).
		WithTimeout(45 * time.Second)

	if builder.GetSTTProvider() != "deepgram" {
		t.Errorf("expected sttProvider 'deepgram', got %s", builder.GetSTTProvider())
	}
	if builder.GetTTSProvider() != "elevenlabs" {
		t.Errorf("expected ttsProvider 'elevenlabs', got %s", builder.GetTTSProvider())
	}
	if builder.GetVADProvider() != "silero" {
		t.Errorf("expected vadProvider 'silero', got %s", builder.GetVADProvider())
	}
	if builder.GetLLMProvider() != "openai" {
		t.Errorf("expected llmProvider 'openai', got %s", builder.GetLLMProvider())
	}
	if builder.GetSystemPrompt() != "Test prompt" {
		t.Errorf("expected systemPrompt 'Test prompt', got %s", builder.GetSystemPrompt())
	}
	if !builder.IsMemoryEnabled() {
		t.Error("expected memory to be enabled")
	}
	if builder.GetMemorySize() != 75 {
		t.Errorf("expected memorySize 75, got %d", builder.GetMemorySize())
	}
	if builder.GetTimeout() != 45*time.Second {
		t.Errorf("expected timeout 45s, got %v", builder.GetTimeout())
	}
}
