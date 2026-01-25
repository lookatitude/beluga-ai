package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
)

// GeminiNativeStreamingSession implements StreamingSession for Gemini 2.5 Flash Native Audio.
type GeminiNativeStreamingSession struct {
	httpClient     HTTPClient
	ctx            context.Context
	cancelFunc     context.CancelFunc
	provider       *GeminiNativeProvider
	audioCh        chan iface.AudioOutputChunk
	restartCh      chan struct{}
	config         *GeminiNativeConfig
	restartTimer   *time.Timer
	audioBuffer    []byte
	maxRetries     int
	retryDelay     time.Duration
	mu             sync.RWMutex
	closed         bool
	restartPending bool
}

// GeminiStreamResponse represents a streaming response from Gemini API.
type GeminiStreamResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				InlineData struct {
					MimeType string `json:"mimeType"`
					Data     string `json:"data"`
				} `json:"inlineData"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// NewGeminiNativeStreamingSession creates a new streaming session.
func NewGeminiNativeStreamingSession(ctx context.Context, config *GeminiNativeConfig, provider *GeminiNativeProvider) (*GeminiNativeStreamingSession, error) {
	streamCtx, cancel := context.WithCancel(ctx)
	session := &GeminiNativeStreamingSession{
		ctx:        ctx,
		config:     config,
		provider:   provider,
		audioCh:    make(chan iface.AudioOutputChunk, 10),
		httpClient: provider.httpClient,
		restartCh:  make(chan struct{}, 1),
		cancelFunc: cancel,
		maxRetries: 3,                      // Default max retries
		retryDelay: 100 * time.Millisecond, // Initial retry delay
	}

	// Gemini uses Server-Sent Events (SSE) for streaming
	// Start goroutine to handle streaming
	go session.handleStreaming(streamCtx)

	return session, nil
}

// handleStreaming handles the streaming connection using SSE.
func (s *GeminiNativeStreamingSession) handleStreaming(ctx context.Context) {
	defer close(s.audioCh)

	// Build streaming URL
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s", s.config.APIEndpoint, s.config.Model, s.config.APIKey)

	// Prepare request body
	requestBody, err := s.prepareStreamingRequest()
	if err != nil {
		s.audioCh <- iface.AudioOutputChunk{
			Error: fmt.Errorf("failed to prepare streaming request: %w", err),
		}
		return
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(requestBody))
	if err != nil {
		s.audioCh <- iface.AudioOutputChunk{
			Error: fmt.Errorf("failed to create request: %w", err),
		}
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.audioCh <- iface.AudioOutputChunk{
			Error: fmt.Errorf("failed to execute request: %w", err),
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		s.audioCh <- iface.AudioOutputChunk{
			Error: fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body)),
		}
		return
	}

	// Read SSE stream
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		s.mu.RLock()
		closed := s.closed
		s.mu.RUnlock()

		if closed {
			return
		}

		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return
			}

			chunk, err := s.processStreamChunk([]byte(data))
			if err != nil {
				s.audioCh <- iface.AudioOutputChunk{Error: err}
				continue
			}
			if chunk != nil {
				s.audioCh <- *chunk
			}
		}
	}

	if err := scanner.Err(); err != nil {
		if !s.isClosed() {
			s.audioCh <- iface.AudioOutputChunk{
				Error: fmt.Errorf("failed to read stream: %w", err),
			}
		}
	}
}

// prepareStreamingRequest prepares the request body for streaming.
func (s *GeminiNativeStreamingSession) prepareStreamingRequest() ([]byte, error) {
	// Use buffered audio if available
	audioData := s.audioBuffer
	if len(audioData) == 0 {
		audioData = []byte{} // Empty for initial request
	}

	audioBase64 := base64.StdEncoding.EncodeToString(audioData)

	request := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]any{
					{
						"inline_data": map[string]any{
							"mimeType": "audio/pcm",
							"data":     audioBase64,
						},
					},
				},
			},
		},
		"generationConfig": map[string]any{
			"temperature": s.config.Temperature,
		},
	}

	return json.Marshal(request)
}

// processStreamChunk processes a chunk from the SSE stream.
func (s *GeminiNativeStreamingSession) processStreamChunk(data []byte) (*iface.AudioOutputChunk, error) {
	var response GeminiStreamResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chunk: %w", err)
	}

	if len(response.Candidates) == 0 {
		return nil, nil
	}

	candidate := response.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return nil, nil
	}

	part := candidate.Content.Parts[0]
	if part.InlineData.Data == "" {
		return nil, nil
	}

	// Decode base64 audio
	audioData, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio: %w", err)
	}

	return &iface.AudioOutputChunk{
		Audio: audioData,
	}, nil
}

// isClosed checks if the session is closed.
func (s *GeminiNativeStreamingSession) isClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

// SendAudio implements the StreamingSession interface.
func (s *GeminiNativeStreamingSession) SendAudio(ctx context.Context, audio []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return s2s.NewS2SError("SendAudio", s2s.ErrCodeStreamClosed,
			errors.New("streaming session is closed"))
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return s2s.NewS2SError("SendAudio", s2s.ErrCodeContextCanceled,
			fmt.Errorf("context canceled: %w", ctx.Err()))
	case <-s.ctx.Done():
		return s2s.NewS2SError("SendAudio", s2s.ErrCodeContextCanceled,
			fmt.Errorf("session context canceled: %w", s.ctx.Err()))
	default:
	}

	// Buffer audio for sending
	s.mu.Lock()
	s.audioBuffer = append(s.audioBuffer, audio...)
	bufferSize := len(s.audioBuffer)
	shouldRestart := !s.restartPending // Only restart if not already pending
	if shouldRestart {
		s.restartPending = true
	}
	s.mu.Unlock()

	// NOTE: Gemini Native Audio streaming API is a one-way streaming API
	// (server-to-client only). To send audio, we restart the streaming session
	// with the accumulated audio buffer.
	//
	// For true bidirectional streaming, use OpenAI Realtime provider.

	// Debounce restarts: wait a short time to accumulate more audio chunks
	// This reduces the number of stream restarts when multiple chunks arrive quickly
	if shouldRestart {
		// Cancel any existing timer
		s.mu.Lock()
		if s.restartTimer != nil {
			s.restartTimer.Stop()
		}
		// Set a new timer to restart after a short delay (debouncing)
		restartDelay := 50 * time.Millisecond
		if bufferSize > 4096 { // If buffer is large enough, restart immediately
			restartDelay = 0
		}
		s.restartTimer = time.AfterFunc(restartDelay, func() {
			s.restartStreamWithRetry(ctx)
		})
		s.mu.Unlock()
	}

	return nil
}

// restartStreamWithRetry restarts the streaming session with retry logic.
func (s *GeminiNativeStreamingSession) restartStreamWithRetry(ctx context.Context) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	audioBuffer := make([]byte, len(s.audioBuffer))
	copy(audioBuffer, s.audioBuffer)
	s.audioBuffer = nil // Clear buffer after copying
	restartPending := s.restartPending
	s.restartPending = false
	s.mu.Unlock()

	if !restartPending {
		return
	}

	// Cancel current streaming context to stop the current stream
	s.cancelFunc()

	// Retry logic with exponential backoff
	var lastErr error
	backoff := s.retryDelay

	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return
			case <-s.ctx.Done():
				return
			case <-time.After(backoff):
				backoff = time.Duration(float64(backoff) * 1.5) // Exponential backoff
			}
		}

		// Check if closed
		s.mu.RLock()
		closed := s.closed
		s.mu.RUnlock()
		if closed {
			return
		}

		// Create new streaming context
		streamCtx, cancel := context.WithCancel(ctx)
		s.mu.Lock()
		s.cancelFunc = cancel
		// Temporarily store audio buffer for the streaming handler
		s.audioBuffer = audioBuffer
		s.mu.Unlock()

		// Start new streaming session with accumulated audio
		go func() {
			s.handleStreaming(streamCtx)
			// Clear buffer after streaming completes
			s.mu.Lock()
			if len(s.audioBuffer) == len(audioBuffer) {
				s.audioBuffer = nil
			}
			s.mu.Unlock()
		}()

		// For one-way streaming APIs, we assume success if no immediate error
		// The actual error handling happens in handleStreaming
		return
	}

	// If all retries failed, send error to audio channel
	if lastErr != nil {
		select {
		case s.audioCh <- iface.AudioOutputChunk{Error: lastErr}:
		default:
		}
	}
}

// ReceiveAudio implements the StreamingSession interface.
func (s *GeminiNativeStreamingSession) ReceiveAudio() <-chan iface.AudioOutputChunk {
	return s.audioCh
}

// Close implements the StreamingSession interface.
func (s *GeminiNativeStreamingSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	if s.restartTimer != nil {
		s.restartTimer.Stop()
	}
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	close(s.audioCh)

	return nil
}
