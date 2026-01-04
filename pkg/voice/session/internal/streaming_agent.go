package internal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// StreamingAgent manages streaming agent response integration for voice sessions.
// It integrates agent streaming with TTS conversion for real-time voice responses.
type StreamingAgent struct {
	ttsProvider      voiceiface.TTSProvider
	agentInstance    *AgentInstance
	currentStream    <-chan agentsiface.AgentStreamChunk
	cancelFunc       context.CancelFunc
	chunkBuffer      chan agentsiface.AgentStreamChunk
	sentenceBuffer   strings.Builder
	maxBufferSize    int
	mu               sync.RWMutex
	streaming        bool
	dropOldestOnFull bool
}

// StreamingAgentConfig configures a StreamingAgent instance.
type StreamingAgentConfig struct {
	MaxBufferSize    int  // Maximum buffer size for chunks (default: 10)
	DropOldestOnFull bool // Drop oldest chunks when buffer is full (default: true)
}

// DefaultStreamingAgentConfig returns default configuration.
func DefaultStreamingAgentConfig() StreamingAgentConfig {
	return StreamingAgentConfig{
		MaxBufferSize:    10,
		DropOldestOnFull: true,
	}
}

// NewStreamingAgent creates a new streaming agent manager with an agent instance.
func NewStreamingAgent(agentInstance *AgentInstance, ttsProvider voiceiface.TTSProvider, config StreamingAgentConfig) *StreamingAgent {
	return &StreamingAgent{
		agentInstance:    agentInstance,
		ttsProvider:      ttsProvider,
		streaming:        false,
		chunkBuffer:      make(chan agentsiface.AgentStreamChunk, config.MaxBufferSize),
		maxBufferSize:    config.MaxBufferSize,
		dropOldestOnFull: config.DropOldestOnFull,
	}
}

// NewStreamingAgentWithCallback creates a new streaming agent manager with a callback (deprecated).
// Deprecated: Use NewStreamingAgent with agent instance instead.
func NewStreamingAgentWithCallback(agentCallback func(ctx context.Context, transcript string) (string, error)) *StreamingAgent {
	return &StreamingAgent{
		streaming: false,
	}
}

// StartStreaming starts streaming agent responses for the given transcript.
// It returns a channel of text chunks that can be converted to TTS audio.
func (sa *StreamingAgent) StartStreaming(ctx context.Context, transcript string) (<-chan string, error) {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	if sa.streaming {
		return nil, errors.New("streaming already active")
	}

	if sa.agentInstance == nil || sa.agentInstance.Agent == nil {
		return nil, errors.New("agent instance not set")
	}

	// Create streaming context with cancellation
	streamCtx, cancel := context.WithCancel(ctx)
	sa.cancelFunc = cancel

	// Prepare inputs for agent
	inputs := map[string]any{
		"input": transcript,
	}

	// Start agent streaming execution
	agentChunkChan, err := sa.agentInstance.Agent.StreamExecute(streamCtx, inputs)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start agent streaming: %w", err)
	}

	sa.currentStream = agentChunkChan
	sa.streaming = true

	// Create output channel for text chunks
	textChunkChan := make(chan string, sa.maxBufferSize)

	// Update agent state
	if err := sa.agentInstance.SetState(AgentStateStreaming); err != nil {
		sa.mu.Unlock()
		cancel()
		sa.mu.Lock()
		sa.streaming = false
		return nil, fmt.Errorf("failed to set agent state: %w", err)
	}

	// Record start time for metrics
	streamStartTime := time.Now()
	firstChunkTime := time.Time{}

	// Start goroutine to process agent chunks
	go func() {
		defer close(textChunkChan)
		defer func() {
			// Record metrics for streaming operation
			if sa.agentInstance != nil && sa.agentInstance.Agent != nil {
				if metrics := sa.agentInstance.Agent.GetMetrics(); metrics != nil {
					streamDuration := time.Since(streamStartTime)
					var latency time.Duration
					if !firstChunkTime.IsZero() {
						latency = firstChunkTime.Sub(streamStartTime)
					}
					metrics.RecordStreamingOperation(ctx, sa.agentInstance.Agent.GetConfig().Name, latency, streamDuration)
				}
			}

			sa.mu.Lock()
			sa.streaming = false
			sa.currentStream = nil
			sa.cancelFunc = nil
			sa.sentenceBuffer.Reset()
			if sa.agentInstance != nil {
				_ = sa.agentInstance.SetState(AgentStateIdle)
			}
			sa.mu.Unlock()
			cancel()
		}()

		for {
			select {
			case <-streamCtx.Done():
				// Context canceled - streaming interrupted
				return

			case agentChunk, ok := <-agentChunkChan:
				if !ok {
					// Stream closed - send any remaining buffered content
					if sa.sentenceBuffer.Len() > 0 {
						remaining := sa.sentenceBuffer.String()
						select {
						case textChunkChan <- remaining:
						case <-streamCtx.Done():
							return
						}
					}
					return
				}

				// Handle error chunks
				if agentChunk.Err != nil {
					// Log error but continue processing other chunks
					_ = agentChunk.Err
					continue
				}

				// Process content chunks
				if agentChunk.Content != "" {
					// Record first chunk time for latency measurement
					if firstChunkTime.IsZero() {
						firstChunkTime = time.Now()
					}

					// Record streaming chunk metrics
					if sa.agentInstance != nil && sa.agentInstance.Agent != nil {
						if metrics := sa.agentInstance.Agent.GetMetrics(); metrics != nil {
							metrics.RecordStreamingChunk(ctx, sa.agentInstance.Agent.GetConfig().Name)
						}
					}

					sa.sentenceBuffer.WriteString(agentChunk.Content)

					// Detect sentence boundaries and send complete sentences
					content := sa.sentenceBuffer.String()
					if sa.hasCompleteSentence(content) {
						select {
						case textChunkChan <- content:
							sa.sentenceBuffer.Reset()
						case <-streamCtx.Done():
							return
						default:
							// Channel full - handle backpressure
							if sa.dropOldestOnFull {
								// Drop oldest (can't do with unbuffered channel, so just skip)
								// For buffered channels, we could implement a drop-oldest strategy
								select {
								case textChunkChan <- content:
									sa.sentenceBuffer.Reset()
								case <-streamCtx.Done():
									return
								}
							} else {
								// Block until channel has space (default behavior)
								select {
								case textChunkChan <- content:
									sa.sentenceBuffer.Reset()
								case <-streamCtx.Done():
									return
								}
							}
						}
					}
				}

				// Handle tool calls (could trigger tool execution)
				if len(agentChunk.ToolCalls) > 0 {
					// For now, just log tool calls
					// In a full implementation, this would trigger tool execution
					_ = agentChunk.ToolCalls
				}

				// Handle finish chunks
				if agentChunk.Finish != nil {
					// Send any remaining content
					if sa.sentenceBuffer.Len() > 0 {
						remaining := sa.sentenceBuffer.String()
						select {
						case textChunkChan <- remaining:
						case <-streamCtx.Done():
							return
						}
					}
					return
				}
			}
		}
	}()

	return textChunkChan, nil
}

// StopStreaming stops streaming agent responses.
func (sa *StreamingAgent) StopStreaming() {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	if !sa.streaming {
		return
	}

	// Cancel streaming context
	if sa.cancelFunc != nil {
		sa.cancelFunc()
		sa.cancelFunc = nil
	}

	// Update agent state
	if sa.agentInstance != nil {
		_ = sa.agentInstance.SetState(AgentStateInterrupted)
	}

	sa.streaming = false
}

// IsStreaming returns whether streaming is active.
func (sa *StreamingAgent) IsStreaming() bool {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.streaming
}

// hasCompleteSentence detects if the text contains a complete sentence.
// A complete sentence ends with punctuation (. ! ?) followed by space or end of string.
func (sa *StreamingAgent) hasCompleteSentence(text string) bool {
	if len(text) == 0 {
		return false
	}

	// Check for sentence-ending punctuation
	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return false
	}

	lastChar := text[len(text)-1]
	return lastChar == '.' || lastChar == '!' || lastChar == '?'
}

// ConvertToAudio converts a text chunk to audio using TTS provider.
func (sa *StreamingAgent) ConvertToAudio(ctx context.Context, text string) ([]byte, error) {
	sa.mu.RLock()
	ttsProvider := sa.ttsProvider
	sa.mu.RUnlock()

	if ttsProvider == nil {
		return nil, errors.New("TTS provider not set")
	}

	return ttsProvider.GenerateSpeech(ctx, text)
}

// StreamToAudio starts streaming audio generation from text chunks.
func (sa *StreamingAgent) StreamToAudio(ctx context.Context, textChunks <-chan string) (<-chan []byte, error) {
	sa.mu.RLock()
	ttsProvider := sa.ttsProvider
	sa.mu.RUnlock()

	if ttsProvider == nil {
		return nil, errors.New("TTS provider not set")
	}

	audioCh := make(chan []byte, sa.maxBufferSize)

	go func() {
		defer close(audioCh)

		for {
			select {
			case <-ctx.Done():
				return

			case text, ok := <-textChunks:
				if !ok {
					return
				}

				// Convert text to audio
				audio, err := ttsProvider.GenerateSpeech(ctx, text)
				if err != nil {
					// Log error but continue with next chunk
					_ = err
					continue
				}

				// Send audio chunk
				select {
				case audioCh <- audio:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return audioCh, nil
}
