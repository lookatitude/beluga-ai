package amazon_nova

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
)

// AmazonNovaStreamingSession implements StreamingSession for Amazon Nova 2 Sonic.
type AmazonNovaStreamingSession struct {
	ctx      context.Context //nolint:containedctx // Required for streaming
	config   *AmazonNovaConfig
	provider *AmazonNovaProvider
	audioCh  chan iface.AudioOutputChunk
	closed   bool
	mu       sync.RWMutex
}

// NewAmazonNovaStreamingSession creates a new streaming session.
func NewAmazonNovaStreamingSession(ctx context.Context, config *AmazonNovaConfig, provider *AmazonNovaProvider) (*AmazonNovaStreamingSession, error) {
	session := &AmazonNovaStreamingSession{
		ctx:      ctx,
		config:   config,
		provider: provider,
		audioCh:  make(chan iface.AudioOutputChunk, 10),
	}

	// TODO: Implement actual streaming connection
	// This will involve:
	// 1. Establishing a bidirectional streaming connection to Bedrock Runtime
	// 2. Starting goroutines for sending audio and receiving responses
	// 3. Handling connection lifecycle

	return session, nil
}

// SendAudio implements the StreamingSession interface.
func (s *AmazonNovaStreamingSession) SendAudio(ctx context.Context, audio []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return s2s.NewS2SError("SendAudio", s2s.ErrCodeStreamClosed,
			errors.New("streaming session is closed"))
	}

	// TODO: Implement actual audio sending
	// This will send audio chunks to the streaming API

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

	// TODO: Close the actual streaming connection

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
