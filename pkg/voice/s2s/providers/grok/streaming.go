package grok

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

// GrokVoiceStreamingSession implements StreamingSession for Grok Voice Agent.
type GrokVoiceStreamingSession struct {
	ctx         context.Context //nolint:containedctx // Required for streaming
	config      *GrokVoiceConfig
	provider    *GrokVoiceProvider
	httpClient  HTTPClient
	audioCh     chan iface.AudioOutputChunk
	closed      bool
	mu          sync.RWMutex
	audioBuffer []byte
}

// GrokStreamResponse represents a streaming response from Grok API.
type GrokStreamResponse struct {
	Audio  string `json:"audio"`
	Format struct {
		SampleRate int    `json:"sample_rate"`
		Channels   int    `json:"channels"`
		Encoding   string `json:"encoding"`
	} `json:"format"`
	Voice struct {
		VoiceID  string `json:"voice_id"`
		Language string `json:"language"`
	} `json:"voice"`
}

// NewGrokVoiceStreamingSession creates a new streaming session.
func NewGrokVoiceStreamingSession(ctx context.Context, config *GrokVoiceConfig, provider *GrokVoiceProvider) (*GrokVoiceStreamingSession, error) {
	session := &GrokVoiceStreamingSession{
		ctx:        ctx,
		config:     config,
		provider:   provider,
		audioCh:    make(chan iface.AudioOutputChunk, 10),
		httpClient: provider.httpClient,
	}

	// Grok uses Server-Sent Events (SSE) for streaming
	// Start goroutine to handle streaming
	go session.handleStreaming(ctx)

	return session, nil
}

// handleStreaming handles the streaming connection using SSE.
func (s *GrokVoiceStreamingSession) handleStreaming(ctx context.Context) {
	defer close(s.audioCh)

	// Build streaming URL
	url := fmt.Sprintf("%s/audio/speech/stream", s.config.APIEndpoint)

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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.APIKey))
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
func (s *GrokVoiceStreamingSession) prepareStreamingRequest() ([]byte, error) {
	// Use buffered audio if available
	audioData := s.audioBuffer
	if len(audioData) == 0 {
		audioData = []byte{} // Empty for initial request
	}

	audioBase64 := base64.StdEncoding.EncodeToString(audioData)

	request := map[string]any{
		"model": s.config.Model,
		"input": map[string]any{
			"audio": audioBase64,
			"format": map[string]any{
				"sample_rate": 24000,
				"channels":     1,
				"encoding":    "pcm",
			},
		},
		"voice":           s.config.VoiceID,
		"response_format": s.config.AudioFormat,
		"temperature":     s.config.Temperature,
		"stream":          true,
	}

	return json.Marshal(request)
}

// processStreamChunk processes a chunk from the SSE stream.
func (s *GrokVoiceStreamingSession) processStreamChunk(data []byte) (*iface.AudioOutputChunk, error) {
	var response GrokStreamResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chunk: %w", err)
	}

	if response.Audio == "" {
		return nil, nil
	}

	// Decode base64 audio
	audioData, err := base64.StdEncoding.DecodeString(response.Audio)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio: %w", err)
	}

	return &iface.AudioOutputChunk{
		Audio: audioData,
	}, nil
}

// isClosed checks if the session is closed.
func (s *GrokVoiceStreamingSession) isClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

// SendAudio implements the StreamingSession interface.
func (s *GrokVoiceStreamingSession) SendAudio(ctx context.Context, audio []byte) error {
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
	// NOTE: Grok Voice Agent streaming API is a one-way streaming API
	// (server-to-client only). It does not support sending additional audio input
	// after the initial streaming request. Audio input must be included in the
	// initial request body.
	//
	// For bidirectional streaming, use:
	// 1. The non-streaming Process() method for each audio chunk
	// 2. OpenAI Realtime provider which supports true bidirectional streaming
	//
	// This method buffers audio for potential future use or for documentation purposes.
	s.audioBuffer = append(s.audioBuffer, audio...)

	return nil
}

// ReceiveAudio implements the StreamingSession interface.
func (s *GrokVoiceStreamingSession) ReceiveAudio() <-chan iface.AudioOutputChunk {
	return s.audioCh
}

// Close implements the StreamingSession interface.
func (s *GrokVoiceStreamingSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	close(s.audioCh)

	return nil
}
