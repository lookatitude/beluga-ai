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

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
)

// GeminiNativeStreamingSession implements StreamingSession for Gemini 2.5 Flash Native Audio.
type GeminiNativeStreamingSession struct {
	ctx         context.Context //nolint:containedctx // Required for streaming
	config      *GeminiNativeConfig
	provider    *GeminiNativeProvider
	httpClient  HTTPClient
	audioCh     chan iface.AudioOutputChunk
	closed      bool
	mu          sync.RWMutex
	audioBuffer []byte
	restartCh   chan struct{} // Channel to signal streaming restart
	cancelFunc  context.CancelFunc // Cancel function for current streaming context
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
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
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
			fmt.Errorf("context cancelled: %w", ctx.Err()))
	case <-s.ctx.Done():
		return s2s.NewS2SError("SendAudio", s2s.ErrCodeContextCanceled,
			fmt.Errorf("session context cancelled: %w", s.ctx.Err()))
	default:
	}

	// Buffer audio for sending
	s.mu.Lock()
	s.audioBuffer = append(s.audioBuffer, audio...)
	audioBuffer := make([]byte, len(s.audioBuffer))
	copy(audioBuffer, s.audioBuffer)
	s.mu.Unlock()

	// NOTE: Gemini Native Audio streaming API is a one-way streaming API
	// (server-to-client only). To send audio, we restart the streaming session
	// with the accumulated audio buffer.
	//
	// For true bidirectional streaming, use OpenAI Realtime provider.

	// Cancel current streaming context to stop the current stream
	s.cancelFunc()

	// Create new streaming context
	streamCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancelFunc = cancel
	s.mu.Unlock()

	// Start new streaming session with accumulated audio
	go s.handleStreaming(streamCtx)

	return nil
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
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	close(s.audioCh)

	return nil
}
