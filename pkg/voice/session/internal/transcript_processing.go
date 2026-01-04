package internal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
)

// ProcessTranscript processes a transcript through the agent and generates a response.
// It supports both streaming and non-streaming agent execution.
func (s *VoiceSessionImpl) ProcessTranscript(ctx context.Context, transcript string) error {
	s.mu.RLock()
	active := s.active
	streamingAgent := s.streamingAgent
	agentIntegration := s.agentIntegration
	s.mu.RUnlock()

	if !active {
		return newSessionError("ProcessTranscript", "session_not_active",
			errors.New("session is not active"))
	}

	// Update conversation context if agent instance exists
	if agentIntegration != nil {
		instance := agentIntegration.GetAgentInstance()
		if instance != nil {
			// Add user message to conversation history
			instance.UpdateContext(func(ctx *AgentContext) {
				ctx.ConversationHistory = append(ctx.ConversationHistory, schema.NewHumanMessage(transcript))
				ctx.StreamingActive = false
			})
		}
	}

	// Transition to processing state
	s.mu.Lock()
	if !s.stateMachine.SetState(sessioniface.SessionState("processing")) {
		s.mu.Unlock()
		return newSessionError("ProcessTranscript", "invalid_state", nil)
	}
	s.state = s.stateMachine.GetState()
	if s.stateChangeCallback != nil {
		s.stateChangeCallback(s.state)
	}
	s.mu.Unlock()

	// Use streaming agent if available, otherwise fall back to non-streaming
	if streamingAgent != nil && !streamingAgent.IsStreaming() {
		return s.processTranscriptWithStreaming(ctx, transcript)
	}

	// Fall back to non-streaming processing
	return s.processTranscriptNonStreaming(ctx, transcript)
}

// processTranscriptWithStreaming processes a transcript using streaming agent execution.
func (s *VoiceSessionImpl) processTranscriptWithStreaming(ctx context.Context, transcript string) error {
	s.mu.RLock()
	streamingAgent := s.streamingAgent
	s.mu.RUnlock()

	if streamingAgent == nil {
		return newSessionError("ProcessTranscript", "agent_not_configured",
			errors.New("streaming agent not available"))
	}

	// Stop any existing streaming before starting new one
	if streamingAgent.IsStreaming() {
		streamingAgent.StopStreaming()
	}

	// Start streaming agent response
	textChunkChan, err := streamingAgent.StartStreaming(ctx, transcript)
	if err != nil {
		return newSessionError("ProcessTranscript", "stream_error",
			fmt.Errorf("failed to start streaming: %w", err))
	}

	// Process text chunks and convert to audio
	go func() {
		defer func() {
			// Transition back to listening state
			s.mu.Lock()
			s.stateMachine.SetState(sessioniface.SessionState("listening"))
			s.state = s.stateMachine.GetState()
			if s.stateChangeCallback != nil {
				s.stateChangeCallback(s.state)
			}
			s.mu.Unlock()

			// Update agent context
			s.mu.RLock()
			agentIntegration := s.agentIntegration
			s.mu.RUnlock()
			if agentIntegration != nil {
				instance := agentIntegration.GetAgentInstance()
				if instance != nil {
					instance.UpdateContext(func(ctx *AgentContext) {
						ctx.StreamingActive = false
					})
				}
			}
		}()

		// Transition to speaking state when we start receiving chunks
		firstChunk := true

		for {
			select {
			case <-ctx.Done():
				streamingAgent.StopStreaming()
				return

			case textChunk, ok := <-textChunkChan:
				if !ok {
					// Stream completed
					return
				}

				if textChunk == "" {
					continue
				}

				// Transition to speaking state on first chunk
				if firstChunk {
					s.mu.Lock()
					s.stateMachine.SetState(sessioniface.SessionState("speaking"))
					s.state = s.stateMachine.GetState()
					if s.stateChangeCallback != nil {
						s.stateChangeCallback(s.state)
					}
					s.mu.Unlock()
					firstChunk = false
				}

				// Convert text chunk to audio and play
				if err := s.playTextChunk(ctx, textChunk); err != nil {
					// Log error but continue with next chunk
					_ = err
				}

				// Update conversation context with agent response
				s.mu.RLock()
				agentIntegration := s.agentIntegration
				s.mu.RUnlock()
				if agentIntegration != nil {
					instance := agentIntegration.GetAgentInstance()
					if instance != nil {
						instance.UpdateContext(func(ctx *AgentContext) {
							ctx.ConversationHistory = append(ctx.ConversationHistory, schema.NewAIMessage(textChunk))
						})
					}
				}
			}
		}
	}()

	return nil
}

// processTranscriptNonStreaming processes a transcript using non-streaming agent execution.
func (s *VoiceSessionImpl) processTranscriptNonStreaming(ctx context.Context, transcript string) error {
	s.mu.RLock()
	agentIntegration := s.agentIntegration
	s.mu.RUnlock()

	if agentIntegration == nil {
		return newSessionError("ProcessTranscript", "agent_not_configured",
			errors.New("agent integration not available"))
	}

	// Generate response (non-streaming)
	response, err := agentIntegration.GenerateResponse(ctx, transcript)
	if err != nil {
		return newSessionError("ProcessTranscript", "agent_error",
			fmt.Errorf("failed to generate response: %w", err))
	}

	// Update conversation context
	s.mu.RLock()
	agentInstance := agentIntegration.GetAgentInstance()
	s.mu.RUnlock()
	if agentInstance != nil {
		agentInstance.UpdateContext(func(ctx *AgentContext) {
			ctx.ConversationHistory = append(ctx.ConversationHistory, schema.NewAIMessage(response))
		})
	}

	// Convert response to speech and play
	return s.playTextChunk(ctx, response)
}

// playTextChunk converts a text chunk to audio and plays it.
func (s *VoiceSessionImpl) playTextChunk(ctx context.Context, text string) error {
	if text == "" {
		return nil
	}

	s.mu.RLock()
	ttsProvider := s.ttsProvider
	transport := s.transport
	s.mu.RUnlock()

	if ttsProvider == nil {
		return errors.New("TTS provider not set")
	}

	// Generate speech from text
	audio, err := ttsProvider.GenerateSpeech(ctx, text)
	if err != nil {
		return fmt.Errorf("failed to generate speech: %w", err)
	}

	// Send audio through transport if available
	if transport != nil {
		if err := transport.SendAudio(ctx, audio); err != nil {
			return fmt.Errorf("failed to send audio: %w", err)
		}
	}

	return nil
}

// HandleInterruption interrupts any active streaming or playback.
func (s *VoiceSessionImpl) HandleInterruption(ctx context.Context) error {
	s.mu.Lock()
	streamingAgent := s.streamingAgent
	s.mu.Unlock()

	// Stop streaming agent if active
	if streamingAgent != nil && streamingAgent.IsStreaming() {
		streamingAgent.StopStreaming()

		// Update agent state
		if s.agentIntegration != nil {
			instance := s.agentIntegration.GetAgentInstance()
			if instance != nil {
				instance.UpdateContext(func(ctx *AgentContext) {
					ctx.StreamingActive = false
					ctx.LastInterruption = time.Now()
				})
				_ = instance.SetState(AgentStateInterrupted)
			}
		}
	}

	// Transition to listening state
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stateMachine.SetState(sessioniface.SessionState("listening"))
	s.state = s.stateMachine.GetState()
	if s.stateChangeCallback != nil {
		s.stateChangeCallback(s.state)
	}

	return nil
}
