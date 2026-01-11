package amazon_nova

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
)

// AmazonNovaStreamingSession implements StreamingSession for Amazon Nova 2 Sonic.
type AmazonNovaStreamingSession struct {
	ctx            context.Context //nolint:containedctx // Required for streaming
	config         *AmazonNovaConfig
	provider       *AmazonNovaProvider
	audioCh        chan iface.AudioOutputChunk
	closed         bool
	mu             sync.RWMutex
	stream         *bedrockruntime.InvokeModelWithResponseStreamOutput
	audioBuffer    []byte
	conversationID string
}

// NewAmazonNovaStreamingSession creates a new streaming session.
func NewAmazonNovaStreamingSession(ctx context.Context, config *AmazonNovaConfig, provider *AmazonNovaProvider) (*AmazonNovaStreamingSession, error) {
	session := &AmazonNovaStreamingSession{
		ctx:      ctx,
		config:   config,
		provider: provider,
		audioCh:  make(chan iface.AudioOutputChunk, 10),
	}

	// Initialize streaming connection to Bedrock Runtime
	// For Nova 2 Sonic, we use InvokeModelWithResponseStream for streaming
	modelID := fmt.Sprintf("amazon.%s-v1:0", config.Model)
	
	// Prepare initial request
	requestBody, err := session.prepareStreamingRequest()
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
func (s *AmazonNovaStreamingSession) prepareStreamingRequest() ([]byte, error) {
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
		"streaming": true,
		"enable_automatic_punctuation": s.config.EnableAutomaticPunctuation,
	}

	return json.Marshal(request)
}

// receiveStreamingResponses receives streaming responses from Bedrock.
func (s *AmazonNovaStreamingSession) receiveStreamingResponses() {
	defer close(s.audioCh)

	if s.stream == nil {
		return
	}

	for event := range s.stream.GetStream().Events() {
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
			SampleRate int    `json:"sample_rate,omitempty"`
			Channels   int    `json:"channels,omitempty"`
			Encoding   string `json:"encoding,omitempty"`
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return s2s.NewS2SError("SendAudio", s2s.ErrCodeStreamClosed,
			errors.New("streaming session is closed"))
	}

	// Buffer audio for sending
	// NOTE: AWS Bedrock InvokeModelWithResponseStream is a one-way streaming API
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
