package internal

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"time"

	"go.opentelemetry.io/otel/codes"
	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
	chatmodelsiface "github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	memoryiface "github.com/lookatitude/beluga-ai/pkg/memory/iface"
	multimodaliface "github.com/lookatitude/beluga-ai/pkg/multimodal/iface"
	orchestrationiface "github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	s2siface "github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

// PipelineOrchestrator orchestrates audio processing pipelines (STT/TTS and S2S).
// Optimized for concurrent processing with thread-safe operations (T174, T176).
type PipelineOrchestrator struct {
	config *vbiface.Config

	// Voice providers
	sttProvider              voiceiface.STTProvider
	ttsProvider              voiceiface.TTSProvider
	s2sProvider              s2siface.S2SProvider
	vadProvider              voiceiface.VADProvider
	turnDetector             voiceiface.TurnDetector
	noiseCancellation        voiceiface.NoiseCancellation

	// Package integrations (optional)
	memory         memoryiface.Memory
	orchestrator   orchestrationiface.Orchestrator
	retriever      interface{} // retrieversiface.Retriever
	vectorStore    vectorstoresiface.VectorStore
	embedder       embeddingsiface.Embedder
	multimodalModel multimodaliface.MultimodalModel
	promptTemplate interface{} // promptsiface.PromptTemplate
	chatModel      chatmodelsiface.ChatModel

	// Custom processors
	customProcessors []vbiface.CustomProcessor

	// State
	mu sync.RWMutex
}

// NewPipelineOrchestrator creates a new pipeline orchestrator.
func NewPipelineOrchestrator(config *vbiface.Config) *PipelineOrchestrator {
	return &PipelineOrchestrator{
		config:          config,
		customProcessors: config.CustomProcessors,
	}
}

// SetSTTProvider sets the STT provider.
func (po *PipelineOrchestrator) SetSTTProvider(provider voiceiface.STTProvider) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.sttProvider = provider
}

// SetTTSProvider sets the TTS provider.
func (po *PipelineOrchestrator) SetTTSProvider(provider voiceiface.TTSProvider) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.ttsProvider = provider
}

// SetS2SProvider sets the S2S provider.
func (po *PipelineOrchestrator) SetS2SProvider(provider s2siface.S2SProvider) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.s2sProvider = provider
}

// SetVADProvider sets the VAD provider.
func (po *PipelineOrchestrator) SetVADProvider(provider voiceiface.VADProvider) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.vadProvider = provider
}

// SetTurnDetector sets the turn detector.
func (po *PipelineOrchestrator) SetTurnDetector(detector voiceiface.TurnDetector) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.turnDetector = detector
}

// SetNoiseCancellation sets the noise cancellation provider.
func (po *PipelineOrchestrator) SetNoiseCancellation(provider voiceiface.NoiseCancellation) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.noiseCancellation = provider
}

// SetMemory sets the memory integration.
func (po *PipelineOrchestrator) SetMemory(memory memoryiface.Memory) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.memory = memory
}

// SetOrchestrator sets the orchestration integration.
func (po *PipelineOrchestrator) SetOrchestrator(orchestrator orchestrationiface.Orchestrator) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.orchestrator = orchestrator
}

// SetRetriever sets the retriever integration.
func (po *PipelineOrchestrator) SetRetriever(retriever interface{}) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.retriever = retriever
}

// SetVectorStore sets the vector store integration.
func (po *PipelineOrchestrator) SetVectorStore(vectorStore vectorstoresiface.VectorStore) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.vectorStore = vectorStore
}

// SetEmbedder sets the embedder integration.
func (po *PipelineOrchestrator) SetEmbedder(embedder embeddingsiface.Embedder) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.embedder = embedder
}

// SetMultimodalModel sets the multimodal model integration.
func (po *PipelineOrchestrator) SetMultimodalModel(model multimodaliface.MultimodalModel) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.multimodalModel = model
}

// SetPromptTemplate sets the prompt template integration.
func (po *PipelineOrchestrator) SetPromptTemplate(template interface{}) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.promptTemplate = template
}

// SetChatModel sets the chat model integration.
func (po *PipelineOrchestrator) SetChatModel(chatModel chatmodelsiface.ChatModel) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.chatModel = chatModel
}

// ProcessAudio processes audio through the configured pipeline (STT/TTS or S2S).
func (po *PipelineOrchestrator) ProcessAudio(ctx context.Context, audio []byte, agentCallback func(context.Context, string) (string, error), agentInstance agentsiface.Agent) ([]byte, error) {
	ctx, span := backend.StartSpan(ctx, "PipelineOrchestrator.ProcessAudio", po.config.Provider)
	defer span.End()

	backend.AddSpanAttributes(span, map[string]any{
		"audio_size":    len(audio),
		"pipeline_type": string(po.config.PipelineType),
	})

	po.mu.RLock()
	pipelineType := po.config.PipelineType
	po.mu.RUnlock()

	switch pipelineType {
	case vbiface.PipelineTypeSTTTTS:
		return po.processSTTTTSPipeline(ctx, audio, agentCallback, agentInstance)
	case vbiface.PipelineTypeS2S:
		return po.processS2SPipeline(ctx, audio, agentCallback, agentInstance)
	default:
		err := backend.NewBackendError("ProcessAudio", backend.ErrCodePipelineError,
			fmt.Errorf("unknown pipeline type: %s", pipelineType))
		backend.RecordSpanError(span, err)
		return nil, err
	}
}

// processSTTTTSPipeline processes audio through STT → Agent → TTS pipeline.
func (po *PipelineOrchestrator) processSTTTTSPipeline(ctx context.Context, audio []byte, agentCallback func(context.Context, string) (string, error), agentInstance agentsiface.Agent) ([]byte, error) {
	ctx, span := backend.StartSpan(ctx, "PipelineOrchestrator.processSTTTTSPipeline", po.config.Provider)
	defer span.End()
	startTime := time.Now()

	// Apply custom processors (before STT)
	audio, err := po.applyCustomProcessors(ctx, audio, "pre_stt")
	if err != nil {
		return nil, backend.WrapError("processSTTTTSPipeline", err)
	}

	// Apply noise cancellation if available
	po.mu.RLock()
	noiseCancellation := po.noiseCancellation
	po.mu.RUnlock()

	if noiseCancellation != nil {
		cleaned, err := noiseCancellation.Process(ctx, audio)
		if err != nil {
			// Log error but continue
		} else {
			audio = cleaned
		}
	}

	// Apply VAD if available
	po.mu.RLock()
	vadProvider := po.vadProvider
	po.mu.RUnlock()

	if vadProvider != nil {
		hasVoice, err := vadProvider.Process(ctx, audio)
		if err != nil {
			// Log error but continue
		} else if !hasVoice {
			// No voice activity, return empty audio
			return []byte{}, nil
		}
	}

	// STT: Convert audio to text
	po.mu.RLock()
	sttProvider := po.sttProvider
	po.mu.RUnlock()

	if sttProvider == nil {
		return nil, backend.NewBackendError("processSTTTTSPipeline", backend.ErrCodePipelineError,
			fmt.Errorf("STT provider not configured"))
	}

	transcript, err := sttProvider.Transcribe(ctx, audio)
	if err != nil {
		return nil, backend.WrapError("processSTTTTSPipeline", err)
	}

	// Turn detection: Check if turn is complete with fallback mechanisms (T293)
	po.mu.RLock()
	turnDetector := po.turnDetector
	po.mu.RUnlock()

	if turnDetector != nil {
		isComplete, err := turnDetector.DetectTurn(ctx, audio)
		if err != nil {
			// Turn detection failure - use fallback mechanism (T293)
			backend.LogWithOTELContext(ctx, slog.LevelWarn, "Turn detection failed, using fallback",
				"error", err)
			// Fallback: Assume turn is complete if audio length exceeds threshold
			// This prevents pipeline from getting stuck
			audioLengthThreshold := 1024 // 1KB threshold
			if len(audio) >= audioLengthThreshold {
				isComplete = true
			} else {
				// Turn not complete, wait for more audio
				return []byte{}, nil
			}
		} else if !isComplete {
			// Turn not complete, wait for more audio
			return []byte{}, nil
		}
	}

	// Store transcript in memory if available
	po.mu.RLock()
	memory := po.memory
	po.mu.RUnlock()

	if memory != nil {
		// Store user message in conversation history
		_ = memory // TODO: Implement memory storage
	}

	// Agent: Process transcript and generate response with timeout handling (T291)
	var response string
	agentTimeout := po.config.Timeout
	if agentTimeout == 0 {
		agentTimeout = 30 * time.Second // Default timeout
	}

	agentCtx, agentCancel := context.WithTimeout(ctx, agentTimeout)
	defer agentCancel()

	agentResponseChan := make(chan string, 1)
	agentErrorChan := make(chan error, 1)

	// Execute agent callback in goroutine to handle timeout
	go func() {
		var resp string
		var err error

		if agentInstance != nil {
			// Use agent instance - agents use Plan method, but for simple use case we'll use agentCallback
			// TODO: Implement proper agent execution using executor pattern
			// For now, fall through to agentCallback
			if agentCallback != nil {
				resp, err = agentCallback(agentCtx, transcript)
			} else {
				err = fmt.Errorf("agent instance does not implement Invoke and no callback provided")
			}
		} else if agentCallback != nil {
			// Use agent callback
			resp, err = agentCallback(agentCtx, transcript)
		} else {
			err = fmt.Errorf("neither agent instance nor callback provided")
		}

		if err != nil {
			agentErrorChan <- err
		} else {
			agentResponseChan <- resp
		}
	}()

	// Wait for agent response with timeout
	select {
	case response = <-agentResponseChan:
		// Success
	case err = <-agentErrorChan:
		// Agent error - prevent response generation failures (T298)
		backend.LogWithOTELContext(ctx, slog.LevelError, "Agent error during response generation",
			"error", err, "transcript", transcript)
		return nil, backend.NewBackendError("processSTTTTSPipeline", backend.ErrCodeAgentError, err)
	case <-agentCtx.Done():
		// Timeout - return timeout error code (T291)
		err := agentCtx.Err()
		backend.LogWithOTELContext(ctx, slog.LevelWarn, "Agent response timeout",
			"timeout", agentTimeout, "transcript", transcript)
		return nil, backend.NewBackendError("processSTTTTSPipeline", backend.ErrCodeTimeout, err)
	case <-ctx.Done():
		// Context canceled
		return nil, backend.NewBackendError("processSTTTTSPipeline", backend.ErrCodeContextCanceled, ctx.Err())
	}

	// Store response in memory if available
	if memory != nil {
		// Store agent response in conversation history
		_ = memory // TODO: Implement memory storage
	}

	// TTS: Convert response text to audio
	po.mu.RLock()
	ttsProvider := po.ttsProvider
	po.mu.RUnlock()

	if ttsProvider == nil {
		return nil, backend.NewBackendError("processSTTTTSPipeline", backend.ErrCodePipelineError,
			fmt.Errorf("TTS provider not configured"))
	}

	outputAudio, err := ttsProvider.GenerateSpeech(ctx, response)
	if err != nil {
		return nil, backend.WrapError("processSTTTTSPipeline", err)
	}

	// Apply custom processors (after TTS)
	outputAudio, err = po.applyCustomProcessors(ctx, outputAudio, "post_tts")
	if err != nil {
		return nil, backend.WrapError("processSTTTTSPipeline", err)
	}

	// Check latency target
	latency := time.Since(startTime)
	backend.AddSpanAttributes(span, map[string]any{
		"latency_ms":        latency.Milliseconds(),
		"latency_target_ms": po.config.LatencyTarget.Milliseconds(),
	})

	if latency > po.config.LatencyTarget {
		backend.LogWithOTELContext(ctx, slog.LevelWarn, "Latency target exceeded",
			"latency_ms", latency.Milliseconds(),
			"target_ms", po.config.LatencyTarget.Milliseconds())
	}

	span.SetStatus(codes.Ok, "audio processed successfully")
	return outputAudio, nil
}

// processS2SPipeline processes audio through S2S pipeline (bypasses text transcription).
func (po *PipelineOrchestrator) processS2SPipeline(ctx context.Context, audio []byte, agentCallback func(context.Context, string) (string, error), agentInstance agentsiface.Agent) ([]byte, error) {
	ctx, span := backend.StartSpan(ctx, "PipelineOrchestrator.processS2SPipeline", po.config.Provider)
	defer span.End()

	startTime := time.Now()
	s2sTargetLatency := 300 * time.Millisecond

	backend.AddSpanAttributes(span, map[string]any{
		"pipeline_type":      "s2s",
		"target_latency_ms":  s2sTargetLatency.Milliseconds(),
		"audio_size":         len(audio),
	})

	// Apply custom processors (before S2S)
	audio, err := po.applyCustomProcessors(ctx, audio, "pre_s2s")
	if err != nil {
		return nil, backend.WrapError("processS2SPipeline", err)
	}

	// Optional: Route through agent for reasoning before S2S (User Story 2 acceptance scenario 2)
	// If agent is provided, transcribe audio, get agent response, then use response text for S2S
	var agentResponseText string
	if agentInstance != nil || agentCallback != nil {
		po.mu.RLock()
		sttProvider := po.sttProvider
		po.mu.RUnlock()

		if sttProvider != nil {
			// Transcribe audio for agent processing
			transcript, err := sttProvider.Transcribe(ctx, audio)
			if err != nil {
				backend.LogWithOTELContext(ctx, slog.LevelWarn, "Failed to transcribe for agent, proceeding with direct S2S", "error", err)
			} else {
				// Get agent response
				if agentInstance != nil {
					// TODO: Implement proper agent execution using executor pattern
					// For now, use agentCallback if available
					if agentCallback != nil {
						response, err := agentCallback(ctx, transcript)
						if err != nil {
							backend.LogWithOTELContext(ctx, slog.LevelWarn, "Agent callback failed, proceeding with direct S2S", "error", err)
						} else {
							agentResponseText = response
						}
					}
				} else if agentCallback != nil {
					response, err := agentCallback(ctx, transcript)
					if err != nil {
						backend.LogWithOTELContext(ctx, slog.LevelWarn, "Agent callback failed, proceeding with direct S2S", "error", err)
					} else {
						agentResponseText = response
					}
				}

				if agentResponseText != "" {
					backend.AddSpanAttributes(span, map[string]any{
						"agent_integration": true,
						"transcript_length": len(transcript),
						"response_length":   len(agentResponseText),
					})
					// Note: S2S providers that support text input can use agentResponseText
					// For now, we'll pass it in the conversation context
				}
			}
		}
	}

	// S2S: Process audio directly
	po.mu.RLock()
	s2sProvider := po.s2sProvider
	memory := po.memory
	po.mu.RUnlock()

	if s2sProvider == nil {
		err := backend.NewBackendError("processS2SPipeline", backend.ErrCodePipelineError,
			fmt.Errorf("S2S provider not configured"))
		backend.RecordSpanError(span, err)
		return nil, err
	}

	// Create audio input using S2S helper (avoids importing s2s/internal)
	audioInput := s2s.NewAudioInput(audio, "")
	if audioInput == nil {
		err := backend.NewBackendError("processS2SPipeline", backend.ErrCodePipelineError,
			fmt.Errorf("failed to create S2S audio input"))
		backend.RecordSpanError(span, err)
		return nil, err
	}

	// Create conversation context using S2S helper
	conversationContext := s2s.NewConversationContext("")
	if conversationContext == nil {
		err := backend.NewBackendError("processS2SPipeline", backend.ErrCodePipelineError,
			fmt.Errorf("failed to create S2S conversation context"))
		backend.RecordSpanError(span, err)
		return nil, err
	}

	// Populate conversation context from memory if available
	if memory != nil {
		// TODO: Retrieve conversation history from memory
		// For now, add agent response text if available
		if agentResponseText != "" {
			if conversationContext.AgentState == nil {
				conversationContext.AgentState = make(map[string]any)
			}
			conversationContext.AgentState["response_text"] = agentResponseText
		}
	}

	// Process through S2S provider with network latency spike handling (T292)
	s2sStartTime := time.Now()
	
	// Create context with timeout for S2S processing to handle network latency spikes
	s2sTimeout := po.config.Timeout
	if s2sTimeout == 0 {
		s2sTimeout = 10 * time.Second // Default S2S timeout
	}
	s2sCtx, s2sCancel := context.WithTimeout(ctx, s2sTimeout)
	defer s2sCancel()

	// Use buffering and retry for network latency spikes (T292)
	maxRetries := 2
	retryDelay := 100 * time.Millisecond
	var audioOutput interface{} // Use interface{} since we can't import internal package

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-s2sCtx.Done():
				return nil, backend.NewBackendError("processS2SPipeline", backend.ErrCodeTimeout, s2sCtx.Err())
			case <-time.After(retryDelay):
			}
			backend.LogWithOTELContext(ctx, slog.LevelWarn, "Retrying S2S processing due to network latency",
				"attempt", attempt+1, "max_retries", maxRetries)
		}

		var s2sErr error
		audioOutput, s2sErr = s2sProvider.Process(s2sCtx, audioInput, conversationContext)
		if s2sErr == nil {
			break // Success
		}

		// Check if error is retryable (network-related)
		if !backend.IsRetryableError(s2sErr) || attempt >= maxRetries {
			backend.RecordSpanError(span, s2sErr)
			return nil, backend.WrapError("processS2SPipeline", s2sErr)
		}
	}

	s2sLatency := time.Since(s2sStartTime)
	backend.AddSpanAttributes(span, map[string]any{
		"s2s_provider_latency_ms": s2sLatency.Milliseconds(),
	})

	// Extract audio data from output using S2S helper with validation (T297)
	// Use reflection to extract Data field since we can't import internal package
	var outputAudioData []byte
	if audioOutput == nil {
		err := backend.NewBackendError("processS2SPipeline", backend.ErrCodePipelineError,
			fmt.Errorf("S2S provider returned nil audio output"))
		backend.RecordSpanError(span, err)
		return nil, err
	}

	// Use reflection to extract Data field from AudioOutput
	rv := reflect.ValueOf(audioOutput)
	if rv.Kind() == reflect.Ptr && !rv.IsNil() {
		rv = rv.Elem()
	}
	if dataField := rv.FieldByName("Data"); dataField.IsValid() && dataField.Kind() == reflect.Slice {
		if dataField.Type().Elem().Kind() == reflect.Uint8 {
			outputAudioData = dataField.Bytes()
		}
	}

	if outputAudioData == nil {
		err := backend.NewBackendError("processS2SPipeline", backend.ErrCodePipelineError,
			fmt.Errorf("S2S provider returned invalid audio output structure"))
		backend.RecordSpanError(span, err)
		return nil, err
	}

	// Validate S2S output (T297) - check for malformed output
	if len(outputAudioData) == 0 {
		err := backend.NewBackendError("processS2SPipeline", backend.ErrCodePipelineError,
			fmt.Errorf("S2S provider returned empty audio output"))
		backend.RecordSpanError(span, err)
		return nil, err
	}

	// Additional validation: check if output is reasonable size
	// Output should be similar size to input (within reasonable bounds)
	maxOutputSize := len(audio) * 10 // Allow up to 10x input size
	minOutputSize := len(audio) / 10 // Allow down to 1/10 input size
	if len(outputAudioData) > maxOutputSize || len(outputAudioData) < minOutputSize {
		backend.LogWithOTELContext(ctx, slog.LevelWarn, "S2S output size suspicious, may be malformed",
			"input_size", len(audio), "output_size", len(outputAudioData),
			"min_expected", minOutputSize, "max_expected", maxOutputSize)
		// Continue processing but log warning
	}

	// Apply custom processors (after S2S)
	outputAudio, err := po.applyCustomProcessors(ctx, outputAudioData, "post_s2s")
	if err != nil {
		return nil, backend.WrapError("processS2SPipeline", err)
	}

	// Check latency target (S2S should be <300ms per SC-005)
	totalLatency := time.Since(startTime)
	backend.AddSpanAttributes(span, map[string]any{
		"total_latency_ms":    totalLatency.Milliseconds(),
		"output_audio_size":   len(outputAudio),
		"latency_target_met":  totalLatency <= s2sTargetLatency,
	})

	if totalLatency > s2sTargetLatency {
		backend.LogWithOTELContext(ctx, slog.LevelWarn, "S2S latency target exceeded",
			"latency_ms", totalLatency.Milliseconds(),
			"target_ms", s2sTargetLatency.Milliseconds(),
			"s2s_provider_latency_ms", s2sLatency.Milliseconds())
	} else {
		backend.LogWithOTELContext(ctx, slog.LevelInfo, "S2S pipeline completed within latency target",
			"latency_ms", totalLatency.Milliseconds())
	}

	span.SetStatus(codes.Ok, "S2S audio processed successfully")
	return outputAudio, nil
}

// applyCustomProcessors applies custom audio processors in order.
func (po *PipelineOrchestrator) applyCustomProcessors(ctx context.Context, audio []byte, stage string) ([]byte, error) {
	po.mu.RLock()
	processors := po.customProcessors
	po.mu.RUnlock()

	if len(processors) == 0 {
		return audio, nil
	}

	// Sort processors by order
	sortedProcessors := make([]vbiface.CustomProcessor, len(processors))
	copy(sortedProcessors, processors)
	// Sort by GetOrder() - lower order = earlier processing
	for i := 0; i < len(sortedProcessors)-1; i++ {
		for j := i + 1; j < len(sortedProcessors); j++ {
			if sortedProcessors[i].GetOrder() > sortedProcessors[j].GetOrder() {
				sortedProcessors[i], sortedProcessors[j] = sortedProcessors[j], sortedProcessors[i]
			}
		}
	}

	result := audio
	for _, processor := range sortedProcessors {
		metadata := map[string]any{
			"stage": stage,
		}
		processed, err := processor.Process(ctx, result, metadata)
		if err != nil {
			return nil, backend.WrapError("applyCustomProcessors", err)
		}
		result = processed
	}

	return result, nil
}

// HandleInterruption handles interruption when user speaks during agent response.
func (po *PipelineOrchestrator) HandleInterruption(ctx context.Context) error {
	// TODO: Implement interruption handling
	// This should:
	// 1. Stop current TTS playback
	// 2. Clear audio buffers
	// 3. Transition to listening state
	return nil
}

// ProcessTurn processes a complete turn (speech boundary detected).
func (po *PipelineOrchestrator) ProcessTurn(ctx context.Context, audio []byte, agentCallback func(context.Context, string) (string, error), agentInstance agentsiface.Agent) ([]byte, error) {
	// Process complete turn through pipeline
	return po.ProcessAudio(ctx, audio, agentCallback, agentInstance)
}
