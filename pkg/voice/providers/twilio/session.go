package twilio

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TwilioVoiceSession implements the VoiceSession interface for Twilio.
type TwilioVoiceSession struct {
	startTime         time.Time
	lastActivity      time.Time
	sttProvider       iface.STTProvider
	agentInstance     agentsiface.Agent
	ttsProvider       iface.TTSProvider
	sessionConfig     *vbiface.SessionConfig
	audioStream       *AudioStream
	backend           *TwilioBackend
	audioInput        chan []byte
	agentCallback     func(context.Context, string) (string, error)
	config            *TwilioConfig
	partialState      map[string]any
	metadata          map[string]any
	audioOutput       chan []byte
	state             vbiface.PipelineState
	persistenceStatus vbiface.PersistenceStatus
	id                string
	callSID           string
	mu                sync.RWMutex
	active            bool
}

// NewTwilioVoiceSession creates a new Twilio voice session.
func NewTwilioVoiceSession(
	callSID string,
	config *TwilioConfig,
	sessionConfig *vbiface.SessionConfig,
	backend *TwilioBackend,
) (*TwilioVoiceSession, error) {
	sessionID := uuid.New().String()

	// Get STT and TTS providers from registry
	var sttProvider iface.STTProvider
	var ttsProvider iface.TTSProvider

	ctx := context.Background()

	if config.STTProvider != "" {
		// Create STT config from provider-specific config
		sttConfig := stt.DefaultConfig()
		if providerConfig, ok := config.ProviderConfig["stt"].(map[string]any); ok {
			// Map provider-specific config to STT config
			if apiKey, ok := providerConfig["api_key"].(string); ok {
				sttConfig.APIKey = apiKey
			}
			if language, ok := providerConfig["language"].(string); ok {
				sttConfig.Language = language
			}
			if model, ok := providerConfig["model"].(string); ok {
				sttConfig.Model = model
			}
		}

		// Get STT provider from registry
		var err error
		sttProvider, err = stt.NewProvider(ctx, config.STTProvider, sttConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create STT provider '%s': %w", config.STTProvider, err)
		}
	}

	if config.TTSProvider != "" {
		// Create TTS config from provider-specific config
		ttsConfig := tts.DefaultConfig()
		if providerConfig, ok := config.ProviderConfig["tts"].(map[string]any); ok {
			// Map provider-specific config to TTS config
			if apiKey, ok := providerConfig["api_key"].(string); ok {
				ttsConfig.APIKey = apiKey
			}
			if voice, ok := providerConfig["voice"].(string); ok {
				ttsConfig.Voice = voice
			}
			if model, ok := providerConfig["model"].(string); ok {
				ttsConfig.Model = model
			}
		}

		// Get TTS provider from registry
		var err error
		ttsProvider, err = tts.NewProvider(ctx, config.TTSProvider, ttsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create TTS provider '%s': %w", config.TTSProvider, err)
		}
	}

	return &TwilioVoiceSession{
		id:                sessionID,
		callSID:           callSID,
		config:            config,
		sessionConfig:     sessionConfig,
		backend:           backend,
		sttProvider:       sttProvider,
		ttsProvider:       ttsProvider,
		state:             vbiface.PipelineStateIdle,
		persistenceStatus: vbiface.PersistenceStatusActive,
		metadata:          make(map[string]any),
		audioOutput:       make(chan []byte, 100),
		audioInput:        make(chan []byte, 100),
		active:            false,
		partialState:      make(map[string]any),
	}, nil
}

// Start starts the voice session.
func (s *TwilioVoiceSession) Start(ctx context.Context) error {
	ctx, span := s.startSpan(ctx, "TwilioVoiceSession.Start")
	defer span.End()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		span.SetStatus(codes.Ok, "already active")
		return nil
	}

	s.active = true
	s.state = vbiface.PipelineStateListening
	s.startTime = time.Now()
	s.lastActivity = time.Now()

	// Create audio stream
	stream, err := s.backend.StreamAudio(ctx, s.id)
	if err != nil {
		s.active = false
		s.state = vbiface.PipelineStateError
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	s.audioStream = stream

	// Start audio processing goroutines
	go s.processIncomingAudio(ctx)
	go s.processOutgoingAudio(ctx)

	span.SetStatus(codes.Ok, "session started")
	return nil
}

// Stop stops the voice session.
func (s *TwilioVoiceSession) Stop(ctx context.Context) error {
	ctx, span := s.startSpan(ctx, "TwilioVoiceSession.Stop")
	defer span.End()

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		span.SetStatus(codes.Ok, "already stopped")
		return nil
	}

	s.active = false
	s.state = vbiface.PipelineStateIdle
	s.lastActivity = time.Now()

	// Close audio stream
	if s.audioStream != nil {
		if err := s.audioStream.Close(); err != nil {
			span.RecordError(err)
		}
	}

	// Save partial state for potential resumption (edge case: call drops)
	s.partialState["last_activity"] = s.lastActivity
	s.partialState["state"] = string(s.state)

	span.SetStatus(codes.Ok, "session stopped")
	return nil
}

// ProcessAudio processes incoming audio data through the pipeline.
func (s *TwilioVoiceSession) ProcessAudio(ctx context.Context, audio []byte) error {
	ctx, span := s.startSpan(ctx, "TwilioVoiceSession.ProcessAudio")
	defer span.End()

	s.mu.RLock()
	active := s.active
	sttProvider := s.sttProvider
	agentCallback := s.agentCallback
	agentInstance := s.agentInstance
	s.mu.RUnlock()

	if !active {
		return NewTwilioError("ProcessAudio", "session_not_active", errors.New("session is not active"))
	}

	s.mu.Lock()
	s.lastActivity = time.Now()
	s.mu.Unlock()

	// Convert mu-law to linear PCM if needed
	linearAudio := convertMuLawToPCM(audio)

	// Process through STT → Agent → TTS pipeline
	if sttProvider != nil {
		// Transcribe audio
		transcript, err := sttProvider.Transcribe(ctx, linearAudio)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		span.SetAttributes(attribute.String("transcript", transcript))

		// Retrieve relevant transcriptions for RAG context (T091, FR-032)
		var ragContext string
		if s.backend.config.Retriever != nil {
			transcriptionMgr := NewTranscriptionManager(s.backend)
			relevantTranscriptions, err := transcriptionMgr.RetrieveRelevantTranscriptions(ctx, transcript, 5)
			if err == nil && len(relevantTranscriptions) > 0 {
				// Build context from relevant transcriptions
				ragContext = "Previous conversations:\n"
				var ragContextSb240 strings.Builder
				for _, t := range relevantTranscriptions {
					ragContextSb240.WriteString(fmt.Sprintf("- %s\n", t.Text))
				}
				ragContext += ragContextSb240.String()
			}
		}

		// Combine transcript with RAG context
		enhancedTranscript := transcript
		if ragContext != "" {
			enhancedTranscript = fmt.Sprintf("%s\n\n%s", ragContext, transcript)
		}

		// Process through agent
		var response string
		if agentInstance != nil {
			// Use agent instance (full implementation would use executor)
			if agentCallback != nil {
				var err error
				response, err = agentCallback(ctx, enhancedTranscript)
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
					return err
				}
			}
		} else if agentCallback != nil {
			var err error
			response, err = agentCallback(ctx, enhancedTranscript)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return err
			}
		}

		// Convert response to speech
		if response != "" && s.ttsProvider != nil {
			s.mu.RLock()
			ttsProvider := s.ttsProvider
			s.mu.RUnlock()

			audioResponse, err := ttsProvider.GenerateSpeech(ctx, response)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return err
			}

			// Convert linear PCM to mu-law
			mulawResponse := convertPCMToMuLaw(audioResponse)

			// Send audio response
			if s.audioStream != nil {
				if err := s.audioStream.SendAudio(ctx, mulawResponse); err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
					return err
				}
			}
		}
	}

	span.SetStatus(codes.Ok, "audio processed")
	return nil
}

// SendAudio sends audio data to the user.
func (s *TwilioVoiceSession) SendAudio(ctx context.Context, audio []byte) error {
	s.mu.RLock()
	active := s.active
	stream := s.audioStream
	s.mu.RUnlock()

	if !active || stream == nil {
		return NewTwilioError("SendAudio", "session_not_active", errors.New("session is not active"))
	}

	return stream.SendAudio(ctx, audio)
}

// ReceiveAudio returns a channel for receiving audio from the user.
func (s *TwilioVoiceSession) ReceiveAudio() <-chan []byte {
	return s.audioInput
}

// SetAgentCallback sets the agent callback function.
func (s *TwilioVoiceSession) SetAgentCallback(callback func(context.Context, string) (string, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.agentCallback = callback
	return nil
}

// SetAgentInstance sets the agent instance.
func (s *TwilioVoiceSession) SetAgentInstance(agent agentsiface.Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.agentInstance = agent
	return nil
}

// GetState returns the current pipeline state.
func (s *TwilioVoiceSession) GetState() vbiface.PipelineState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// GetPersistenceStatus returns the persistence status.
func (s *TwilioVoiceSession) GetPersistenceStatus() vbiface.PersistenceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.persistenceStatus
}

// UpdateMetadata updates the session metadata.
func (s *TwilioVoiceSession) UpdateMetadata(metadata map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, value := range metadata {
		s.metadata[key] = value
	}
	return nil
}

// GetID returns the session identifier.
func (s *TwilioVoiceSession) GetID() string {
	return s.id
}

// GetCallSID returns the Twilio call SID.
func (s *TwilioVoiceSession) GetCallSID() string {
	return s.callSID
}

// processIncomingAudio processes incoming audio from Twilio.
func (s *TwilioVoiceSession) processIncomingAudio(ctx context.Context) {
	if s.audioStream == nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case audio, ok := <-s.audioStream.ReceiveAudio():
			if !ok {
				return
			}

			// Process audio through pipeline
			if err := s.ProcessAudio(ctx, audio); err != nil {
				// Log error but continue processing
				continue
			}
		}
	}
}

// processOutgoingAudio processes outgoing audio to Twilio.
func (s *TwilioVoiceSession) processOutgoingAudio(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case audio, ok := <-s.audioOutput:
			if !ok {
				return
			}

			// Send audio to Twilio
			if s.audioStream != nil {
				if err := s.audioStream.SendAudio(ctx, audio); err != nil {
					// Log error but continue processing
					continue
				}
			}
		}
	}
}

// startSpan starts an OTEL span for tracing.
func (s *TwilioVoiceSession) startSpan(ctx context.Context, operation string) (context.Context, trace.Span) {
	if s.backend.metrics != nil && s.backend.metrics.Tracer() != nil {
		return s.backend.metrics.Tracer().Start(ctx, operation,
			trace.WithAttributes(
				attribute.String("session_id", s.id),
				attribute.String("call_sid", s.callSID),
			))
	}
	return ctx, trace.SpanFromContext(ctx)
}

// ResumeSession resumes a session from partial state (edge case: call drops).
func (s *TwilioVoiceSession) ResumeSession(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Restore state from partial state
	if lastActivity, ok := s.partialState["last_activity"].(time.Time); ok {
		s.lastActivity = lastActivity
	}

	// Restore conversation context if available
	// This allows resumption if customer calls back

	return nil
}
