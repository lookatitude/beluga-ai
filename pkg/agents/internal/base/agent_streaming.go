// Package base provides streaming implementation for BaseAgent.
// This file implements the StreamingAgent interface methods.
package base

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// streamingState tracks the current streaming operation state.
type streamingState struct {
	currentChan <-chan iface.AgentStreamChunk
	cancelFunc  context.CancelFunc
	mu          sync.RWMutex
	active      bool
}

// StreamExecute implements the StreamingAgent interface.
// It executes the agent with streaming LLM responses.
func (a *BaseAgent) StreamExecute(ctx context.Context, inputs map[string]any) (<-chan iface.AgentStreamChunk, error) {
	// Validate inputs
	if err := a.validateInputs(inputs); err != nil {
		return nil, fmt.Errorf("streaming error for agent %s StreamExecute: invalid input: %w", a.name, err)
	}

	// Check if streaming is enabled
	streamingConfig := a.getStreamingConfig()
	if !streamingConfig.EnableStreaming {
		return nil, fmt.Errorf("streaming error for agent %s StreamExecute: streaming is not enabled (code: streaming_not_supported)", a.name)
	}

	// Check if LLM supports streaming
	chatModel, ok := a.llm.(llmsiface.ChatModel)
	if !ok {
		return nil, fmt.Errorf("streaming error for agent %s StreamExecute: LLM does not implement ChatModel interface for streaming (code: streaming_not_supported)", a.name)
	}

	// Start metrics tracking
	startTime := time.Now()
	firstChunkTime := time.Time{}

	// Create output channel with buffer
	bufferSize := streamingConfig.ChunkBufferSize
	if bufferSize <= 0 {
		bufferSize = 20 // Default buffer size
	}
	outputChan := make(chan iface.AgentStreamChunk, bufferSize)

	// Create context with timeout if configured
	streamCtx := ctx
	var streamCancel context.CancelFunc
	if streamingConfig.MaxStreamDuration > 0 {
		streamCtx, streamCancel = context.WithTimeout(ctx, streamingConfig.MaxStreamDuration)
	} else {
		streamCtx, streamCancel = context.WithCancel(ctx)
	}

	// Track streaming state
	a.setStreamingActive(true, streamCancel)

	// Build messages from inputs
	messages, err := a.buildMessagesFromInputs(inputs)
	if err != nil {
		streamCancel()
		a.setStreamingActive(false, nil)
		return nil, fmt.Errorf("streaming error for agent %s StreamExecute: failed to build messages: %w", a.name, err)
	}

	// Start streaming LLM response
	llmChunkChan, err := chatModel.StreamChat(streamCtx, messages)
	if err != nil {
		streamCancel()
		a.setStreamingActive(false, nil)
		return nil, fmt.Errorf("streaming error for agent %s StreamExecute: failed to start LLM stream (code: stream_error): %w", a.name, err)
	}

	// Start goroutine to process LLM chunks
	go func() {
		defer func() {
			streamCancel()
			a.setStreamingActive(false, nil)
			close(outputChan)
		}()

		var accumulatedContent strings.Builder
		var accumulatedToolCalls []schema.ToolCall
		var toolCallBuilder map[string]*schema.ToolCallChunk
		var chunkCount int

		// Process LLM chunks
		for {
			select {
			case <-streamCtx.Done():
				// Context canceled - record metrics and send error chunk
				totalDuration := time.Since(startTime)
				latency := time.Duration(0)
				if !firstChunkTime.IsZero() {
					latency = firstChunkTime.Sub(startTime)
				}

				// Record streaming operation metrics even on cancellation
				if a.metrics != nil {
					a.metrics.RecordStreamingOperation(streamCtx, a.name, latency, totalDuration)
				}

				// Context canceled - send error chunk and exit
				outputChan <- iface.AgentStreamChunk{
					Err: fmt.Errorf("streaming error for agent %s StreamExecute: stream interrupted (code: stream_interrupted): %w", a.name, streamCtx.Err()),
					Metadata: map[string]any{
						"chunks_processed": chunkCount,
						"total_duration":   totalDuration.Seconds(),
					},
				}
				return

			case llmChunk, ok := <-llmChunkChan:
				if !ok {
					// LLM stream closed - send final chunk
					totalDuration := time.Since(startTime)
					latency := time.Duration(0)
					if !firstChunkTime.IsZero() {
						latency = firstChunkTime.Sub(startTime)
					}

					// Record streaming operation metrics
					if a.metrics != nil {
						a.metrics.RecordStreamingOperation(streamCtx, a.name, latency, totalDuration)
					}

					finalContent := accumulatedContent.String()
					if finalContent != "" || len(accumulatedToolCalls) > 0 {
						finish := &iface.AgentFinish{
							ReturnValues: map[string]any{
								"output": finalContent,
							},
							Log: fmt.Sprintf("Streaming completed with %d chunks", chunkCount),
						}

						outputChan <- iface.AgentStreamChunk{
							Content:   finalContent,
							ToolCalls: accumulatedToolCalls,
							Finish:    finish,
							Metadata: map[string]any{
								"chunks_processed": chunkCount,
								"total_duration":   totalDuration.Seconds(),
								"final":            true,
							},
						}
					}
					return
				}

				// Record first chunk time for latency metrics
				if firstChunkTime.IsZero() {
					firstChunkTime = time.Now()
				}

				// Handle LLM chunk error
				if llmChunk.Err != nil {
					outputChan <- iface.AgentStreamChunk{
						Err: fmt.Errorf("streaming error for agent %s StreamExecute: LLM chunk error (code: stream_error): %w", a.name, llmChunk.Err),
						Metadata: map[string]any{
							"chunks_processed": chunkCount,
							"error_chunk":      true,
						},
					}
					return
				}

				// Accumulate content
				if llmChunk.Content != "" {
					accumulatedContent.WriteString(llmChunk.Content)
				}

				// Process tool calls
				if len(llmChunk.ToolCallChunks) > 0 {
					if toolCallBuilder == nil {
						toolCallBuilder = make(map[string]*schema.ToolCallChunk)
					}
					// Accumulate tool call chunks (they may come in parts)
					for _, tcChunk := range llmChunk.ToolCallChunks {
						if existing, exists := toolCallBuilder[tcChunk.ID]; exists {
							// Merge chunk into existing
							if tcChunk.Name != "" {
								existing.Name = tcChunk.Name
							}
							if tcChunk.Arguments != "" {
								existing.Arguments += tcChunk.Arguments
							}
						} else {
							tc := tcChunk // Copy
							toolCallBuilder[tcChunk.ID] = &tc
						}
					}
				}

				// Create agent chunk from LLM chunk
				agentChunk := iface.AgentStreamChunk{
					Content: llmChunk.Content,
					Metadata: map[string]any{
						"chunk_index": chunkCount,
						"timestamp":   time.Now(),
					},
				}

				// Convert tool call chunks to tool calls if complete
				if len(llmChunk.ToolCallChunks) > 0 {
					// For now, create tool calls from chunks (simplified)
					// In a full implementation, we'd need to accumulate chunks until complete
					toolCalls := make([]schema.ToolCall, 0, len(llmChunk.ToolCallChunks))
					for _, tcChunk := range llmChunk.ToolCallChunks {
						if tcChunk.Name != "" {
							toolCalls = append(toolCalls, schema.ToolCall{
								ID:        tcChunk.ID,
								Name:      tcChunk.Name,
								Arguments: tcChunk.Arguments,
							})
						}
					}
					if len(toolCalls) > 0 {
						agentChunk.ToolCalls = toolCalls
						accumulatedToolCalls = append(accumulatedToolCalls, toolCalls...)
					}
				}

				// Apply sentence boundary detection if enabled
				if streamingConfig.SentenceBoundary {
					// Only send chunk if we have a complete sentence
					content := accumulatedContent.String()
					if a.hasCompleteSentence(content) {
						agentChunk.Content = content
						accumulatedContent.Reset()
					} else {
						// Skip sending this chunk, wait for more content
						continue
					}
				}

				// Send chunk
				chunkCount++
				select {
				case outputChan <- agentChunk:
					// Record chunk metrics
					if a.metrics != nil {
						a.metrics.RecordStreamingChunk(streamCtx, a.name)
					}
				case <-streamCtx.Done():
					return
				}
			}
		}
	}()

	return outputChan, nil
}

// StreamPlan implements the StreamingAgent interface.
// It plans the next action with streaming model responses.
func (a *BaseAgent) StreamPlan(ctx context.Context, intermediateSteps []iface.IntermediateStep, inputs map[string]any) (<-chan iface.AgentStreamChunk, error) {
	// Similar implementation to StreamExecute but focused on planning
	// For now, delegate to StreamExecute with planning context
	return a.StreamExecute(ctx, inputs)
}

// Helper methods

// validateInputs validates that all required input variables are present.
func (a *BaseAgent) validateInputs(inputs map[string]any) error {
	required := a.InputVariables()
	for _, key := range required {
		if _, exists := inputs[key]; !exists {
			return fmt.Errorf("missing required input variable: %s", key)
		}
	}
	return nil
}

// getStreamingConfig returns the streaming configuration from options.
func (a *BaseAgent) getStreamingConfig() iface.StreamingConfig {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.streamingConfig
}

// buildMessagesFromInputs converts inputs to messages for LLM.
func (a *BaseAgent) buildMessagesFromInputs(inputs map[string]any) ([]schema.Message, error) {
	// Simple implementation - convert inputs to a single user message
	// In a full implementation, this would handle conversation history, etc.
	messages := make([]schema.Message, 0)

	// Extract input text
	if inputText, ok := inputs["input"].(string); ok {
		messages = append(messages, schema.NewHumanMessage(inputText))
	} else {
		// Convert all inputs to a text representation
		var parts []string
		for key, value := range inputs {
			parts = append(parts, fmt.Sprintf("%s: %v", key, value))
		}
		messages = append(messages, schema.NewHumanMessage(strings.Join(parts, "\n")))
	}

	return messages, nil
}

// hasCompleteSentence checks if the content ends with a sentence boundary.
func (a *BaseAgent) hasCompleteSentence(content string) bool {
	if content == "" {
		return false
	}

	// Check for sentence-ending punctuation followed by space or end of string
	trimmed := strings.TrimSpace(content)
	if len(trimmed) == 0 {
		return false
	}

	lastChar := trimmed[len(trimmed)-1]
	sentenceEnders := []rune{'.', '!', '?'}
	for _, ender := range sentenceEnders {
		if rune(lastChar) == ender {
			return true
		}
	}

	return false
}

// setStreamingActive sets the streaming active state and cancel function.
func (a *BaseAgent) setStreamingActive(active bool, cancel context.CancelFunc) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// TODO: Store streaming state if needed
	// For now, just track in agent state
	if active {
		// Mark as streaming state
		// This is a placeholder - full implementation would track active streams
	} else {
		// Clear streaming state
		if cancel != nil {
			cancel()
		}
	}
}
