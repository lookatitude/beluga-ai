package amazon_nova

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
)

// AmazonNovaStreamingSession implements StreamingSession for Amazon Nova 2 Sonic.
type AmazonNovaStreamingSession struct {
	ctx            context.Context
	stream         *bedrockruntime.InvokeModelWithResponseStreamOutput
	provider       *AmazonNovaProvider
	audioCh        chan iface.AudioOutputChunk
	config         *AmazonNovaConfig
	restartTimer   *time.Timer
	conversationID string
	audioBuffer    []byte
	maxRetries     int
	retryDelay     time.Duration
	mu             sync.RWMutex
	closed         bool
	restartPending bool
}

// NewAmazonNovaStreamingSession creates a new streaming session.
func NewAmazonNovaStreamingSession(ctx context.Context, config *AmazonNovaConfig, provider *AmazonNovaProvider) (*AmazonNovaStreamingSession, error) {
	session := &AmazonNovaStreamingSession{
		ctx:        ctx,
		config:     config,
		provider:   provider,
		audioCh:    make(chan iface.AudioOutputChunk, 10),
		maxRetries: 3,                      // Default max retries
		retryDelay: 100 * time.Millisecond, // Initial retry delay
	}

	// Initialize streaming connection to Bedrock Runtime
	// For Nova 2 Sonic, we use InvokeModelWithResponseStream for streaming
	modelID := fmt.Sprintf("amazon.%s-v1:0", config.Model)

	// Prepare initial request (no audio yet - will be sent via SendAudio)
	requestBody, err := session.prepareStreamingRequest(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare streaming request: %w", err)
	}

	invokeInput := &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     aws.String(modelID),
		ContentType: aws.String("application/json"),
		Body:        requestBody,
	}

	// Start streaming
	stream, err := provider.client.InvokeModelWithResponseStream(ctx, invokeInput)
	if err != nil {
		return nil, fmt.Errorf("failed to start streaming: %w", err)
	}

	session.stream = stream

	// Start goroutine to receive streaming responses
	go session.receiveStreamingResponses()

	return session, nil
}

// prepareStreamingRequest prepares the initial request for streaming.
// If audioData is provided, it will be included in the request.
func (s *AmazonNovaStreamingSession) prepareStreamingRequest(audioData []byte) ([]byte, error) {
	request := map[string]any{
		"voice": map[string]any{
			"voice_id": s.config.VoiceID,
			"language": s.config.LanguageCode,
		},
		"output": map[string]any{
			"format": map[string]any{
				"sample_rate": s.config.SampleRate,
				"channels":    1,
				"encoding":    s.config.AudioFormat,
			},
		},
		"streaming":                    true,
		"enable_automatic_punctuation": s.config.EnableAutomaticPunctuation,
	}

	// Include audio input if provided
	if len(audioData) > 0 {
		audioBase64 := base64.StdEncoding.EncodeToString(audioData)
		request["input"] = map[string]any{
			"audio": audioBase64,
			"format": map[string]any{
				"sample_rate": s.config.SampleRate,
				"channels":    1,
				"encoding":    "PCM", // Default encoding
			},
		}
	}

	return json.Marshal(request)
}

// receiveStreamingResponses receives streaming responses from Bedrock.
func (s *AmazonNovaStreamingSession) receiveStreamingResponses() {
	defer func() {
		// Only close channel if not already closed by Close() method
		s.mu.Lock()
		wasClosed := s.closed
		if !wasClosed {
			s.closed = true
			close(s.audioCh)
		}
		s.mu.Unlock()
	}()

	// Safely get the stream under lock
	s.mu.RLock()
	streamOutput := s.stream
	s.mu.RUnlock()

	if streamOutput == nil {
		return
	}

	// Check if stream is valid (mock clients may return nil stream)
	stream := streamOutput.GetStream()
	if stream == nil {
		return
	}

	for event := range stream.Events() {
		s.mu.RLock()
		closed := s.closed
		s.mu.RUnlock()

		if closed {
			return
		}

		switch v := event.(type) {
		case *types.ResponseStreamMemberChunk:
			// Process chunk
			chunk, err := s.processStreamingChunk(v.Value)
			if err != nil {
				s.audioCh <- iface.AudioOutputChunk{
					Error: err,
				}
				return
			}
			if chunk != nil {
				s.audioCh <- *chunk
			}
		case *types.UnknownUnionMember:
			// Unknown event type, log and continue
			continue
		}
	}
}

// processStreamingChunk processes a streaming chunk from Bedrock.
func (s *AmazonNovaStreamingSession) processStreamingChunk(chunk types.PayloadPart) (*iface.AudioOutputChunk, error) {
	// Parse chunk data
	var chunkData struct {
		Audio  string `json:"audio,omitempty"`
		Format struct {
			Encoding   string `json:"encoding,omitempty"`
			SampleRate int    `json:"sample_rate,omitempty"`
			Channels   int    `json:"channels,omitempty"`
		} `json:"format,omitempty"`
	}

	// Extract bytes from PayloadPart
	chunkBytes := chunk.Bytes
	if len(chunkBytes) == 0 {
		return nil, nil
	}

	if err := json.Unmarshal(chunkBytes, &chunkData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chunk: %w", err)
	}

	if chunkData.Audio == "" {
		return nil, nil
	}

	// Decode base64 audio
	audioData, err := base64.StdEncoding.DecodeString(chunkData.Audio)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio: %w", err)
	}

	return &iface.AudioOutputChunk{
		Audio: audioData,
	}, nil
}

// SendAudio implements the StreamingSession interface.
func (s *AmazonNovaStreamingSession) SendAudio(ctx context.Context, audio []byte) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return s2s.NewS2SError("SendAudio", s2s.ErrCodeStreamClosed,
			errors.New("streaming session is closed"))
	}

	// Buffer audio for sending
	s.audioBuffer = append(s.audioBuffer, audio...)
	audioBuffer := make([]byte, len(s.audioBuffer))
	copy(audioBuffer, s.audioBuffer)

	// Close current stream if it exists
	if s.stream != nil && s.stream.GetStream() != nil {
		_ = s.stream.GetStream().Close()
		s.stream = nil
	}
	s.mu.Unlock()

	// NOTE: AWS Bedrock InvokeModelWithResponseStream is a one-way streaming API
	// (server-to-client only). To send audio, we need to create a new streaming request.
	// This implementation creates a new streaming session with accumulated audio.
	//
	// For true bidirectional streaming, use OpenAI Realtime provider.

	// Create new streaming request with accumulated audio
	modelID := fmt.Sprintf("amazon.%s-v1:0", s.config.Model)
	requestBody, err := s.prepareStreamingRequest(audioBuffer)
	if err != nil {
		return s2s.NewS2SError("SendAudio", s2s.ErrCodeInvalidRequest,
			fmt.Errorf("failed to prepare streaming request: %w", err))
	}

	invokeInput := &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     aws.String(modelID),
		ContentType: aws.String("application/json"),
		Body:        requestBody,
	}

	// Start new streaming session
	stream, err := s.provider.client.InvokeModelWithResponseStream(ctx, invokeInput)
	if err != nil {
		return s2s.NewS2SError("SendAudio", s2s.ErrCodeStreamError,
			fmt.Errorf("failed to start new streaming session: %w", err))
	}

	s.mu.Lock()
	s.stream = stream
	s.audioBuffer = nil // Clear buffer after sending
	s.mu.Unlock()

	// Restart receiver goroutine for new stream
	go s.receiveStreamingResponses()

	return nil
}

// ReceiveAudio implements the StreamingSession interface.
func (s *AmazonNovaStreamingSession) ReceiveAudio() <-chan iface.AudioOutputChunk {
	return s.audioCh
}

// Close implements the StreamingSession interface.
func (s *AmazonNovaStreamingSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	if s.restartTimer != nil {
		s.restartTimer.Stop()
	}
	close(s.audioCh)

	// Close the streaming connection
	if s.stream != nil && s.stream.GetStream() != nil {
		_ = s.stream.GetStream().Close()
	}

	return nil
}

// AmazonNovaProvider implements StreamingS2SProvider interface.
var _ iface.StreamingS2SProvider = (*AmazonNovaProvider)(nil)

// StartStreaming implements the StreamingS2SProvider interface.
func (p *AmazonNovaProvider) StartStreaming(ctx context.Context, convCtx *internal.ConversationContext, opts ...internal.STSOption) (iface.StreamingSession, error) {
	if !p.config.EnableStreaming {
		return nil, s2s.NewS2SError("StartStreaming", s2s.ErrCodeInvalidConfig,
			errors.New("streaming is disabled in configuration"))
	}

	session, err := NewAmazonNovaStreamingSession(ctx, p.config, p)
	if err != nil {
		return nil, s2s.NewS2SError("StartStreaming", s2s.ErrCodeStreamError,
			fmt.Errorf("failed to create streaming session: %w", err))
	}

	return session, nil
}
