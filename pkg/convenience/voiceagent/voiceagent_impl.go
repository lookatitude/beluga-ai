package voiceagent

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	memoryiface "github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	sttiface "github.com/lookatitude/beluga-ai/pkg/stt/iface"
	ttsiface "github.com/lookatitude/beluga-ai/pkg/tts/iface"
	vadiface "github.com/lookatitude/beluga-ai/pkg/vad/iface"
)

// convenienceVoiceAgent implements the VoiceAgent interface.
type convenienceVoiceAgent struct {
	stt          sttiface.STTProvider
	tts          ttsiface.TTSProvider
	vad          vadiface.VADProvider
	llm          llmsiface.ChatModel
	memory       memoryiface.Memory
	systemPrompt string
	timeout      time.Duration
	onTranscript TranscriptCallback
	onResponse   ResponseCallback
	onError      ErrorCallback
	metrics      *Metrics

	mu       sync.RWMutex
	sessions map[string]*voiceSession
}

// StartSession creates and starts a new voice session.
func (a *convenienceVoiceAgent) StartSession(ctx context.Context) (Session, error) {
	const op = "voiceagent.StartSession"

	sessionID := uuid.New().String()

	// Start session span
	ctx, span := a.metrics.StartSessionSpan(ctx, sessionID, "start_session")
	if span != nil {
		defer span.End()
	}

	session := &voiceSession{
		id:           sessionID,
		agent:        a,
		audioCh:      make(chan []byte, 100),
		responsesCh:  make(chan []byte, 100),
		transcript:   "",
		active:       false,
		onTranscript: a.onTranscript,
		onResponse:   a.onResponse,
		onError:      a.onError,
	}

	// Register session
	a.mu.Lock()
	if a.sessions == nil {
		a.sessions = make(map[string]*voiceSession)
	}
	a.sessions[sessionID] = session
	a.mu.Unlock()

	// Start the session
	if err := session.Start(ctx); err != nil {
		a.mu.Lock()
		delete(a.sessions, sessionID)
		a.mu.Unlock()
		return nil, NewError(op, ErrCodeSessionCreation, err)
	}

	a.metrics.RecordSessionStart(ctx, sessionID)
	return session, nil
}

// ProcessAudio transcribes audio, processes it with the LLM, and generates audio response.
func (a *convenienceVoiceAgent) ProcessAudio(ctx context.Context, audio []byte) ([]byte, error) {
	const op = "voiceagent.ProcessAudio"

	// Apply timeout if configured
	if a.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.timeout)
		defer cancel()
	}

	// Transcribe audio
	startSTT := time.Now()
	text, err := a.stt.Transcribe(ctx, audio)
	sttLatency := time.Since(startSTT)
	if err != nil {
		a.metrics.RecordTranscription(ctx, sttLatency, false)
		if a.onError != nil {
			a.onError(err)
		}
		return nil, NewError(op, ErrCodeTranscription, err)
	}
	a.metrics.RecordTranscription(ctx, sttLatency, true)

	if a.onTranscript != nil {
		a.onTranscript(text, true)
	}

	// Process with LLM if available
	var response string
	if a.llm != nil {
		response, err = a.processWithLLM(ctx, text)
		if err != nil {
			if a.onError != nil {
				a.onError(err)
			}
			return nil, NewError(op, ErrCodeExecution, err)
		}
	} else {
		// No LLM, just echo back the text
		response = text
	}

	if a.onResponse != nil {
		a.onResponse(response)
	}

	// Synthesize response to audio
	startTTS := time.Now()
	audioResponse, err := a.tts.GenerateSpeech(ctx, response)
	ttsLatency := time.Since(startTTS)
	if err != nil {
		a.metrics.RecordSynthesis(ctx, ttsLatency, false)
		if a.onError != nil {
			a.onError(err)
		}
		return nil, NewError(op, ErrCodeSynthesis, err)
	}
	a.metrics.RecordSynthesis(ctx, ttsLatency, true)

	return audioResponse, nil
}

// ProcessText processes text input and generates a text response.
func (a *convenienceVoiceAgent) ProcessText(ctx context.Context, text string) (string, error) {
	const op = "voiceagent.ProcessText"

	// Apply timeout if configured
	if a.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.timeout)
		defer cancel()
	}

	if a.onTranscript != nil {
		a.onTranscript(text, true)
	}

	// Process with LLM if available
	var response string
	var err error
	if a.llm != nil {
		response, err = a.processWithLLM(ctx, text)
		if err != nil {
			if a.onError != nil {
				a.onError(err)
			}
			return "", NewError(op, ErrCodeExecution, err)
		}
	} else {
		// No LLM, just echo back the text
		response = text
	}

	if a.onResponse != nil {
		a.onResponse(response)
	}

	return response, nil
}

// processWithLLM processes text with the LLM, including memory if configured.
func (a *convenienceVoiceAgent) processWithLLM(ctx context.Context, text string) (string, error) {
	messages := []schema.Message{}

	// Add system prompt if configured
	if a.systemPrompt != "" {
		messages = append(messages, schema.NewSystemMessage(a.systemPrompt))
	}

	// Load memory context if available
	if a.memory != nil {
		memVars, err := a.memory.LoadMemoryVariables(ctx, nil)
		if err == nil {
			if history, ok := memVars["history"].([]schema.Message); ok {
				messages = append(messages, history...)
			}
		}
	}

	// Add user message
	messages = append(messages, schema.NewHumanMessage(text))

	// Generate response
	aiMsg, err := a.llm.Generate(ctx, messages)
	if err != nil {
		return "", err
	}

	response := aiMsg.GetContent()

	// Save to memory if available
	if a.memory != nil {
		_ = a.memory.SaveContext(ctx, map[string]any{
			"input": text,
		}, map[string]any{
			"output": response,
		})
	}

	return response, nil
}

// GetSTT returns the STT provider.
func (a *convenienceVoiceAgent) GetSTT() sttiface.STTProvider {
	return a.stt
}

// GetTTS returns the TTS provider.
func (a *convenienceVoiceAgent) GetTTS() ttsiface.TTSProvider {
	return a.tts
}

// GetVAD returns the VAD provider.
func (a *convenienceVoiceAgent) GetVAD() vadiface.VADProvider {
	return a.vad
}

// Shutdown gracefully stops the voice agent and releases resources.
func (a *convenienceVoiceAgent) Shutdown() error {
	const op = "voiceagent.Shutdown"

	a.mu.Lock()
	defer a.mu.Unlock()

	var lastErr error
	for id, session := range a.sessions {
		if err := session.Stop(); err != nil {
			lastErr = err
		}
		delete(a.sessions, id)
	}

	if lastErr != nil {
		return NewError(op, ErrCodeShutdown, lastErr)
	}

	return nil
}
