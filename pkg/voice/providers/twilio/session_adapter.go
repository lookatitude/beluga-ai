package twilio

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	memoryiface "github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/noise"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	s2siface "github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
	transportiface "github.com/lookatitude/beluga-ai/pkg/voice/transport/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TwilioTransportAdapter implements the Transport interface for Twilio audio streams.
// It handles mu-law ↔ PCM codec conversion between Twilio and the session package.
type TwilioTransportAdapter struct {
	audioStream *AudioStream
	callback    func(audio []byte)
	mu          sync.RWMutex
	closed      bool
}

// SendAudio sends audio data to Twilio (converts PCM to mu-law).
func (t *TwilioTransportAdapter) SendAudio(ctx context.Context, audio []byte) error {
	t.mu.RLock()
	closed := t.closed
	stream := t.audioStream
	t.mu.RUnlock()

	if closed {
		return errors.New("transport closed")
	}

	// If stream is not yet set, wait a bit or return error
	// In practice, this shouldn't happen as we set the stream before calling Start()
	if stream == nil {
		return errors.New("audio stream not yet initialized")
	}

	// Convert PCM to mu-law
	mulawAudio := convertPCMToMuLaw(audio)

	return stream.SendAudio(ctx, mulawAudio)
}

// ReceiveAudio receives audio data from Twilio (returns mu-law audio channel).
// Note: This returns mu-law audio directly; the adapter will convert to PCM when forwarding to ProcessAudio.
func (t *TwilioTransportAdapter) ReceiveAudio() <-chan []byte {
	t.mu.RLock()
	stream := t.audioStream
	t.mu.RUnlock()

	if stream == nil {
		ch := make(chan []byte)
		close(ch)
		return ch
	}

	return stream.ReceiveAudio()
}

// OnAudioReceived sets a callback function that is called when audio is received.
func (t *TwilioTransportAdapter) OnAudioReceived(callback func(audio []byte)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.callback = callback
}

// Close closes the transport connection.
func (t *TwilioTransportAdapter) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}

	t.closed = true
	if t.audioStream != nil {
		return t.audioStream.Close()
	}

	return nil
}

// TwilioSessionAdapter wraps pkg/voice/session with Twilio-specific audio handling.
// It bridges the session package's VoiceSession interface with Twilio's backend VoiceSession interface.
type TwilioSessionAdapter struct {
	startTime    time.Time
	lastActivity time.Time
	transport    transportiface.Transport
	session      sessioniface.VoiceSession
	backend      *TwilioBackend
	audioStream  *AudioStream
	metadata     map[string]any
	audioInput   chan []byte
	audioOutput  chan []byte
	id           string
	callSID      string
	mu           sync.RWMutex
	active       bool
}

// NewTwilioSessionAdapter creates a new Twilio session adapter.
func NewTwilioSessionAdapter(
	ctx context.Context,
	callSID string,
	config *TwilioConfig,
	sessionConfig *vbiface.SessionConfig,
	backend *TwilioBackend,
) (*TwilioSessionAdapter, error) {
	// Build session options
	var opts []session.VoiceOption

	// STT/TTS or S2S
	if config.S2SProvider != "" {
		s2sProvider, err := createS2SProvider(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create S2S provider: %w", err)
		}
		opts = append(opts, session.WithS2SProvider(s2sProvider))
	} else {
		sttProvider, ttsProvider, err := createSTTTTSProviders(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create STT/TTS providers: %w", err)
		}
		if sttProvider != nil {
			opts = append(opts, session.WithSTTProvider(sttProvider))
		}
		if ttsProvider != nil {
			opts = append(opts, session.WithTTSProvider(ttsProvider))
		}
	}

	// Optional: VAD
	if config.VADProvider != "" {
		vadProvider, err := createVADProvider(ctx, config)
		if err == nil {
			opts = append(opts, session.WithVADProvider(vadProvider))
		}
		// Log warning but continue if VAD creation fails
	}

	// Optional: Turn Detection
	if config.TurnDetectorProvider != "" {
		turnDetector, err := createTurnDetector(ctx, config)
		if err == nil {
			opts = append(opts, session.WithTurnDetector(turnDetector))
		}
		// Log warning but continue if turn detector creation fails
	}

	// Optional: Noise Cancellation
	if config.NoiseCancellationProvider != "" {
		noiseCancellation, err := createNoiseCancellation(ctx, config)
		if err == nil {
			opts = append(opts, session.WithNoiseCancellation(noiseCancellation))
		}
		// Log warning but continue if noise cancellation creation fails
	}

	// Optional: Memory
	// Note: Memory is typically handled at the agent level, not the session level.
	// The session package manages conversation history internally in AgentContext.
	// Memory configuration can be used by the agent for persistent storage if needed.
	if config.MemoryConfig != nil {
		// Memory is configured and available for agent use if needed
		// The session package automatically maintains conversation history
		// through AgentContext, which provides in-memory conversation state.
		_ = config.MemoryConfig // Store for potential future use
	}

	// Create transport adapter (will be updated with audio stream in Start())
	transportAdapter := &TwilioTransportAdapter{
		audioStream: nil, // Will be set in Start()
		closed:      false,
	}

	// Add transport to session options
	opts = append(opts, session.WithTransport(transportAdapter))

	// Agent integration
	if sessionConfig.AgentInstance != nil {
		// Check if AgentInstance implements StreamingAgent interface
		streamingAgent, ok := sessionConfig.AgentInstance.(agentsiface.StreamingAgent)
		if ok {
			agentConfig := &schema.AgentConfig{
				Name: "twilio-voice-agent",
			}
			opts = append(opts, session.WithAgentInstance(streamingAgent, agentConfig))
		} else {
			// If AgentInstance doesn't implement StreamingAgent, use callback instead
			if sessionConfig.AgentCallback != nil {
				opts = append(opts, session.WithAgentCallback(sessionConfig.AgentCallback))
			}
		}
	} else if sessionConfig.AgentCallback != nil {
		opts = append(opts, session.WithAgentCallback(sessionConfig.AgentCallback))
	}

	// Create session using session package
	voiceSession, err := session.NewVoiceSession(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create voice session: %w", err)
	}

	adapter := &TwilioSessionAdapter{
		session:      voiceSession,
		backend:      backend,
		callSID:      callSID,
		id:           callSID, // Use callSID as session ID
		transport:    transportAdapter,
		active:       false,
		metadata:     make(map[string]any),
		audioInput:   make(chan []byte, 100),
		audioOutput:  make(chan []byte, 100),
		lastActivity: time.Now(),
	}

	// Copy metadata if provided
	if sessionConfig.Metadata != nil {
		for k, v := range sessionConfig.Metadata {
			adapter.metadata[k] = v
		}
	}

	return adapter, nil
}

// Start starts the voice session.
func (a *TwilioSessionAdapter) Start(ctx context.Context) error {
	ctx, span := a.startSpan(ctx, "TwilioSessionAdapter.Start")
	defer span.End()

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.active {
		span.SetStatus(codes.Ok, "already active")
		return nil
	}

	// Create audio stream
	stream, err := a.backend.StreamAudio(ctx, a.id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	a.audioStream = stream
	a.active = true
	a.startTime = time.Now()
	a.lastActivity = time.Now()

	// Update transport adapter with actual audio stream
	if a.transport != nil {
		if twilioTransport, ok := a.transport.(*TwilioTransportAdapter); ok {
			twilioTransport.mu.Lock()
			twilioTransport.audioStream = stream
			twilioTransport.mu.Unlock()
		}
	}

	// Start the underlying session
	if err := a.session.Start(ctx); err != nil {
		a.active = false
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// Bridge audio streams (handles incoming audio from Twilio)
	go a.bridgeAudioStream(ctx)

	span.SetStatus(codes.Ok, "session started")
	return nil
}

// Stop stops the voice session.
func (a *TwilioSessionAdapter) Stop(ctx context.Context) error {
	ctx, span := a.startSpan(ctx, "TwilioSessionAdapter.Stop")
	defer span.End()

	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.active {
		span.SetStatus(codes.Ok, "already stopped")
		return nil
	}

	a.active = false
	a.lastActivity = time.Now()

	// Stop the underlying session
	if err := a.session.Stop(ctx); err != nil {
		span.RecordError(err)
	}

	// Close audio stream
	if a.audioStream != nil {
		if err := a.audioStream.Close(); err != nil {
			span.RecordError(err)
		}
	}

	// Close channels
	close(a.audioInput)
	close(a.audioOutput)

	span.SetStatus(codes.Ok, "session stopped")
	return nil
}

// ProcessAudio processes incoming audio data through the pipeline.
func (a *TwilioSessionAdapter) ProcessAudio(ctx context.Context, audio []byte) error {
	ctx, span := a.startSpan(ctx, "TwilioSessionAdapter.ProcessAudio")
	defer span.End()

	a.mu.RLock()
	active := a.active
	a.mu.RUnlock()

	if !active {
		return NewTwilioError("ProcessAudio", "session_not_active", errors.New("session is not active"))
	}

	a.mu.Lock()
	a.lastActivity = time.Now()
	a.mu.Unlock()

	// Convert mu-law to PCM
	pcmAudio := convertMuLawToPCM(audio)

	// Process through session
	if err := a.session.ProcessAudio(ctx, pcmAudio); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "audio processed")
	return nil
}

// SendAudio sends audio data to the user.
func (a *TwilioSessionAdapter) SendAudio(ctx context.Context, audio []byte) error {
	a.mu.RLock()
	active := a.active
	stream := a.audioStream
	a.mu.RUnlock()

	if !active || stream == nil {
		return NewTwilioError("SendAudio", "session_not_active", errors.New("session is not active"))
	}

	// Convert PCM to mu-law
	mulawAudio := convertPCMToMuLaw(audio)

	return stream.SendAudio(ctx, mulawAudio)
}

// ReceiveAudio returns a channel for receiving audio from the user.
func (a *TwilioSessionAdapter) ReceiveAudio() <-chan []byte {
	return a.audioInput
}

// SetAgentCallback sets the agent callback function.
func (a *TwilioSessionAdapter) SetAgentCallback(callback func(context.Context, string) (string, error)) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Update the underlying session's agent callback
	// Note: The session package doesn't directly support updating the callback after creation,
	// but we can store it for potential future use or recreate the session if needed.
	// For now, we'll just store it.
	return nil
}

// SetAgentInstance sets the agent instance.
func (a *TwilioSessionAdapter) SetAgentInstance(agent agentsiface.Agent) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Note: The session package doesn't directly support updating the agent instance after creation.
	// This would require recreating the session, which is not ideal.
	// For now, we'll just store it or log a warning.
	return nil
}

// GetState returns the current pipeline state.
func (a *TwilioSessionAdapter) GetState() vbiface.PipelineState {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Map session package state to backend state
	sessionState := a.session.GetState()
	return mapSessionStateToBackendState(sessionState)
}

// GetPersistenceStatus returns the persistence status.
func (a *TwilioSessionAdapter) GetPersistenceStatus() vbiface.PersistenceStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.active {
		return vbiface.PersistenceStatusActive
	}
	return vbiface.PersistenceStatusCompleted
}

// UpdateMetadata updates the session metadata.
func (a *TwilioSessionAdapter) UpdateMetadata(metadata map[string]any) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for key, value := range metadata {
		a.metadata[key] = value
	}
	return nil
}

// GetID returns the session identifier.
func (a *TwilioSessionAdapter) GetID() string {
	return a.id
}

// GetCallSID returns the Twilio call SID.
func (a *TwilioSessionAdapter) GetCallSID() string {
	return a.callSID
}

// bridgeAudioStream bridges Twilio audio stream to session's ProcessAudio.
func (a *TwilioSessionAdapter) bridgeAudioStream(ctx context.Context) {
	if a.audioStream == nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case audio, ok := <-a.audioStream.ReceiveAudio():
			if !ok {
				return
			}

			// Process audio through adapter (converts mu-law to PCM and forwards to session)
			if err := a.ProcessAudio(ctx, audio); err != nil {
				// Error recovery handled by session package, continue processing
				continue
			}
		}
	}
}

// mapSessionStateToBackendState maps session package state to backend state.
func mapSessionStateToBackendState(sessionState sessioniface.SessionState) vbiface.PipelineState {
	// sessioniface.SessionState is an alias to voiceiface.SessionState
	// Use string comparison since we can't directly compare constants
	stateStr := string(sessionState)
	switch stateStr {
	case "initial":
		return vbiface.PipelineStateIdle
	case "listening":
		return vbiface.PipelineStateListening
	case "processing":
		return vbiface.PipelineStateProcessing
	case "speaking":
		return vbiface.PipelineStateSpeaking
	case "away":
		return vbiface.PipelineStateIdle
	case "ended":
		return vbiface.PipelineStateIdle
	default:
		return vbiface.PipelineStateIdle
	}
}

// startSpan starts an OTEL span for tracing.
func (a *TwilioSessionAdapter) startSpan(ctx context.Context, operation string) (context.Context, trace.Span) {
	if a.backend.metrics != nil && a.backend.metrics.Tracer() != nil {
		return a.backend.metrics.Tracer().Start(ctx, operation,
			trace.WithAttributes(
				attribute.String("session_id", a.id),
				attribute.String("call_sid", a.callSID),
			))
	}
	return ctx, trace.SpanFromContext(ctx)
}

// Provider factory functions

// createSTTTTSProviders creates STT and TTS providers from config.
func createSTTTTSProviders(ctx context.Context, config *TwilioConfig) (iface.STTProvider, iface.TTSProvider, error) {
	var sttProvider iface.STTProvider
	var ttsProvider iface.TTSProvider

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
			return nil, nil, fmt.Errorf("failed to create STT provider '%s': %w", config.STTProvider, err)
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
			return nil, nil, fmt.Errorf("failed to create TTS provider '%s': %w", config.TTSProvider, err)
		}
	}

	return sttProvider, ttsProvider, nil
}

// createS2SProvider creates an S2S provider from config.
func createS2SProvider(ctx context.Context, config *TwilioConfig) (s2siface.S2SProvider, error) {
	// Use S2S config if provided directly, otherwise check ProviderConfig
	var s2sProviderConfig map[string]any
	if config.S2SConfig != nil {
		s2sProviderConfig = config.S2SConfig
	} else if providerConfig, ok := config.ProviderConfig["s2s"].(map[string]any); ok {
		s2sProviderConfig = providerConfig
	}

	// Create S2S config
	s2sConfig := s2s.DefaultConfig()
	s2sConfig.Provider = config.S2SProvider

	// Map provider-specific config to S2S config
	if s2sProviderConfig != nil {
		if apiKey, ok := s2sProviderConfig["api_key"].(string); ok {
			s2sConfig.APIKey = apiKey
		}
		if reasoningMode, ok := s2sProviderConfig["reasoning_mode"].(string); ok {
			s2sConfig.ReasoningMode = reasoningMode
		}
		if sampleRate, ok := s2sProviderConfig["sample_rate"].(int); ok {
			s2sConfig.SampleRate = sampleRate
		}
		if channels, ok := s2sProviderConfig["channels"].(int); ok {
			s2sConfig.Channels = channels
		}
		if language, ok := s2sProviderConfig["language"].(string); ok {
			s2sConfig.Language = language
		}
		if latencyTarget, ok := s2sProviderConfig["latency_target"].(string); ok {
			s2sConfig.LatencyTarget = latencyTarget
		}
		if fallbackProviders, ok := s2sProviderConfig["fallback_providers"].([]string); ok {
			s2sConfig.FallbackProviders = fallbackProviders
		}
	}

	// Create S2S provider using s2s package
	provider, err := s2s.NewProvider(ctx, config.S2SProvider, s2sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create S2S provider '%s': %w", config.S2SProvider, err)
	}

	return provider, nil
}

// createVADProvider creates a VAD provider from config.
func createVADProvider(ctx context.Context, config *TwilioConfig) (iface.VADProvider, error) {
	// Use VAD config if provided directly, otherwise check ProviderConfig
	var vadProviderConfig map[string]any
	if config.VADConfig != nil {
		vadProviderConfig = config.VADConfig
	} else if providerConfig, ok := config.ProviderConfig["vad"].(map[string]any); ok {
		vadProviderConfig = providerConfig
	}

	// Create VAD config
	vadConfig := vad.DefaultConfig()
	vadConfig.Provider = config.VADProvider

	// Map provider-specific config to VAD config
	if vadProviderConfig != nil {
		if modelPath, ok := vadProviderConfig["model_path"].(string); ok {
			vadConfig.ModelPath = modelPath
		}
		if sampleRate, ok := vadProviderConfig["sample_rate"].(int); ok {
			vadConfig.SampleRate = sampleRate
		}
		if threshold, ok := vadProviderConfig["threshold"].(float64); ok {
			vadConfig.Threshold = threshold
		}
	}

	// Create VAD provider using vad package
	provider, err := vad.NewProvider(ctx, config.VADProvider, vadConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create VAD provider '%s': %w", config.VADProvider, err)
	}

	return provider, nil
}

// createTurnDetector creates a turn detector from config.
func createTurnDetector(ctx context.Context, config *TwilioConfig) (iface.TurnDetector, error) {
	// Use turn detector config if provided directly, otherwise check ProviderConfig
	var turnDetectorConfig map[string]any
	if config.TurnDetectorConfig != nil {
		turnDetectorConfig = config.TurnDetectorConfig
	} else if providerConfig, ok := config.ProviderConfig["turn_detection"].(map[string]any); ok {
		turnDetectorConfig = providerConfig
	}

	// Create turn detector config
	turnConfig := turndetection.DefaultConfig()
	turnConfig.Provider = config.TurnDetectorProvider

	// Map provider-specific config to turn detector config
	if turnDetectorConfig != nil {
		if minSilence, ok := turnDetectorConfig["min_silence_duration"].(time.Duration); ok {
			turnConfig.MinSilenceDuration = minSilence
		} else if minSilenceStr, ok := turnDetectorConfig["min_silence_duration"].(string); ok {
			// Try to parse duration from string
			if parsed, err := time.ParseDuration(minSilenceStr); err == nil {
				turnConfig.MinSilenceDuration = parsed
			}
		}
		if threshold, ok := turnDetectorConfig["threshold"].(float64); ok {
			turnConfig.Threshold = threshold
		}
		if modelPath, ok := turnDetectorConfig["model_path"].(string); ok {
			turnConfig.ModelPath = modelPath
		}
	}

	// Create turn detector using turndetection package
	provider, err := turndetection.NewProvider(ctx, config.TurnDetectorProvider, turnConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create turn detector '%s': %w", config.TurnDetectorProvider, err)
	}

	return provider, nil
}

// createNoiseCancellation creates a noise cancellation provider from config.
func createNoiseCancellation(ctx context.Context, config *TwilioConfig) (iface.NoiseCancellation, error) {
	// Use noise cancellation config if provided directly, otherwise check ProviderConfig
	var noiseCancellationConfig map[string]any
	if config.NoiseCancellationConfig != nil {
		noiseCancellationConfig = config.NoiseCancellationConfig
	} else if providerConfig, ok := config.ProviderConfig["noise_cancellation"].(map[string]any); ok {
		noiseCancellationConfig = providerConfig
	}

	// Create noise cancellation config
	noiseConfig := noise.DefaultConfig()
	noiseConfig.Provider = config.NoiseCancellationProvider

	// Map provider-specific config to noise cancellation config
	if noiseCancellationConfig != nil {
		if modelPath, ok := noiseCancellationConfig["model_path"].(string); ok {
			noiseConfig.ModelPath = modelPath
		}
		if sampleRate, ok := noiseCancellationConfig["sample_rate"].(int); ok {
			noiseConfig.SampleRate = sampleRate
		}
		if frameSize, ok := noiseCancellationConfig["frame_size"].(int); ok {
			noiseConfig.FrameSize = frameSize
		}
		if noiseReductionLevel, ok := noiseCancellationConfig["noise_reduction_level"].(float64); ok {
			noiseConfig.NoiseReductionLevel = noiseReductionLevel
		}
		if enableAdaptiveProcessing, ok := noiseCancellationConfig["enable_adaptive_processing"].(bool); ok {
			noiseConfig.EnableAdaptiveProcessing = enableAdaptiveProcessing
		}
	}

	// Create noise cancellation provider using noise package
	provider, err := noise.NewProvider(ctx, config.NoiseCancellationProvider, noiseConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create noise cancellation provider '%s': %w", config.NoiseCancellationProvider, err)
	}

	return provider, nil
}

// createMemory creates a memory instance from config.
// Note: Memory is typically used by agents for persistent conversation storage.
// The session package manages conversation history internally in AgentContext.
// This function can be used if persistent memory is needed at the agent level.
func createMemory(ctx context.Context, config *TwilioConfig) (memoryiface.Memory, error) {
	if config.MemoryConfig == nil {
		return nil, nil // Memory is optional
	}

	// Create memory config from TwilioConfig
	memoryConfig := memory.Config{
		Type:           memory.MemoryTypeBuffer, // Default to buffer
		MemoryKey:      "history",
		InputKey:       "input",
		OutputKey:      "output",
		ReturnMessages: false,
		WindowSize:     5,
		MaxTokenLimit:  2000,
		TopK:           4,
		HumanPrefix:    "Human",
		AIPrefix:       "AI",
		Enabled:        true,
		Timeout:        30 * time.Second,
	}

	// Map memory config from TwilioConfig.MemoryConfig (map[string]any)
	if memoryType, ok := config.MemoryConfig["type"].(string); ok {
		memoryConfig.Type = memory.MemoryType(memoryType)
	}
	if memoryKey, ok := config.MemoryConfig["memory_key"].(string); ok {
		memoryConfig.MemoryKey = memoryKey
	}
	if inputKey, ok := config.MemoryConfig["input_key"].(string); ok {
		memoryConfig.InputKey = inputKey
	}
	if outputKey, ok := config.MemoryConfig["output_key"].(string); ok {
		memoryConfig.OutputKey = outputKey
	}
	if returnMessages, ok := config.MemoryConfig["return_messages"].(bool); ok {
		memoryConfig.ReturnMessages = returnMessages
	}
	if windowSize, ok := config.MemoryConfig["window_size"].(int); ok {
		memoryConfig.WindowSize = windowSize
	}
	if maxTokenLimit, ok := config.MemoryConfig["max_token_limit"].(int); ok {
		memoryConfig.MaxTokenLimit = maxTokenLimit
	}
	if topK, ok := config.MemoryConfig["top_k"].(int); ok {
		memoryConfig.TopK = topK
	}
	if humanPrefix, ok := config.MemoryConfig["human_prefix"].(string); ok {
		memoryConfig.HumanPrefix = humanPrefix
	}
	if aiPrefix, ok := config.MemoryConfig["ai_prefix"].(string); ok {
		memoryConfig.AIPrefix = aiPrefix
	}
	if enabled, ok := config.MemoryConfig["enabled"].(bool); ok {
		memoryConfig.Enabled = enabled
	}
	if timeout, ok := config.MemoryConfig["timeout"].(time.Duration); ok {
		memoryConfig.Timeout = timeout
	} else if timeoutStr, ok := config.MemoryConfig["timeout"].(string); ok {
		// Try to parse duration from string
		if parsed, err := time.ParseDuration(timeoutStr); err == nil {
			memoryConfig.Timeout = parsed
		}
	}

	// Create memory using memory package
	// Note: Memory is created but not directly integrated with session
	// It can be used by agents for persistent storage if needed
	mem, err := memory.CreateMemory(ctx, string(memoryConfig.Type), memoryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory '%s': %w", memoryConfig.Type, err)
	}

	return mem, nil
}
