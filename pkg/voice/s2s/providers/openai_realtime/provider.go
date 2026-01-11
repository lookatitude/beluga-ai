package openai_realtime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
)

// OpenAIRealtimeProvider implements the S2SProvider interface for OpenAI Realtime API.
type OpenAIRealtimeProvider struct {
	config       *OpenAIRealtimeConfig
	wsDialer     WebSocketDialer
	mu           sync.RWMutex
	providerName string
}

// WebSocketDialer is an interface for WebSocket dialing operations.
// This allows dependency injection for testing.
type WebSocketDialer interface {
	Dial(url string, headers map[string][]string) (WebSocketConn, error)
}

// WebSocketConn is an interface for WebSocket connection operations.
type WebSocketConn interface {
	ReadJSON(v interface{}) error
	WriteJSON(v interface{}) error
	Close() error
}

// ProviderOption is a function type for configuring the provider.
type ProviderOption func(*OpenAIRealtimeProvider)

// WithWebSocketDialer sets a custom WebSocket dialer for the provider.
// This is useful for testing with mock WebSocket dialers.
func WithWebSocketDialer(dialer WebSocketDialer) ProviderOption {
	return func(p *OpenAIRealtimeProvider) {
		p.wsDialer = dialer
	}
}

// NewOpenAIRealtimeProvider creates a new OpenAI Realtime API provider.
func NewOpenAIRealtimeProvider(config *s2s.Config, opts ...ProviderOption) (iface.S2SProvider, error) {
	if config == nil {
		return nil, s2s.NewS2SError("NewOpenAIRealtimeProvider", s2s.ErrCodeInvalidConfig,
			errors.New("config cannot be nil"))
	}

	// Convert base config to OpenAI Realtime config
	openaiConfig := &OpenAIRealtimeConfig{
		Config: config,
	}

	// Set defaults if not provided
	if openaiConfig.APIEndpoint == "" {
		openaiConfig.APIEndpoint = "https://api.openai.com/v1"
	}
	if openaiConfig.Model == "" {
		openaiConfig.Model = "gpt-4o-realtime-preview"
	}
	if openaiConfig.VoiceID == "" {
		openaiConfig.VoiceID = "alloy"
	}
	if openaiConfig.LanguageCode == "" {
		openaiConfig.LanguageCode = "en-US"
	}
	if openaiConfig.Timeout == 0 {
		openaiConfig.Timeout = 30 * time.Second
	}
	if openaiConfig.SampleRate == 0 {
		openaiConfig.SampleRate = 24000
	}
	if openaiConfig.AudioFormat == "" {
		openaiConfig.AudioFormat = "pcm"
	}
	if openaiConfig.Temperature == 0 {
		openaiConfig.Temperature = 0.8
	}

	// Validate API key
	if openaiConfig.APIKey == "" {
		return nil, s2s.NewS2SError("NewOpenAIRealtimeProvider", s2s.ErrCodeInvalidConfig,
			errors.New("API key is required"))
	}

	provider := &OpenAIRealtimeProvider{
		config:       openaiConfig,
		providerName: "openai_realtime",
	}

	// Apply options
	for _, opt := range opts {
		opt(provider)
	}

	return provider, nil
}

// Process implements the S2SProvider interface using OpenAI Realtime API.
// Note: OpenAI Realtime API is primarily designed for streaming. For non-streaming
// use cases, we create a temporary streaming session and collect the output.
func (p *OpenAIRealtimeProvider) Process(ctx context.Context, input *internal.AudioInput, convCtx *internal.ConversationContext, opts ...internal.STSOption) (*internal.AudioOutput, error) {
	startTime := time.Now()

	// Start tracing
	ctx, span := s2s.StartProcessSpan(ctx, p.providerName, p.config.Model, input.Language)
	defer span.End()

	// Check context cancellation
	select {
	case <-ctx.Done():
		s2s.RecordSpanError(span, ctx.Err())
		return nil, s2s.NewS2SError("Process", s2s.ErrCodeContextCanceled,
			fmt.Errorf("context cancelled: %w", ctx.Err()))
	default:
	}

	// Validate input
	if err := internal.ValidateAudioInput(input); err != nil {
		s2s.RecordSpanError(span, err)
		return nil, s2s.NewS2SError("Process", s2s.ErrCodeInvalidInput, err)
	}

	// Apply options
	stsOpts := &internal.STSOptions{}
	for _, opt := range opts {
		opt(stsOpts)
	}

	// For non-streaming, use a temporary streaming session
	// OpenAI Realtime API is WebSocket-based and designed for streaming
	session, err := p.StartStreaming(ctx, convCtx, opts...)
	if err != nil {
		s2s.RecordSpanError(span, err)
		return nil, err
	}
	defer session.Close()

	// Send audio input
	if err := session.SendAudio(ctx, input.Data); err != nil {
		s2s.RecordSpanError(span, err)
		return nil, s2s.NewS2SError("Process", s2s.ErrCodeStreamError,
			fmt.Errorf("failed to send audio: %w", err))
	}

	// Collect audio output chunks
	var audioData []byte
	audioCh := session.ReceiveAudio()
	timeout := time.NewTimer(p.config.Timeout)
	defer timeout.Stop()

	for {
		select {
		case chunk, ok := <-audioCh:
			if !ok {
				// Channel closed, done receiving
				goto done
			}
			if chunk.Error != nil {
				s2s.RecordSpanError(span, chunk.Error)
				return nil, s2s.NewS2SError("Process", s2s.ErrCodeStreamError, chunk.Error)
			}
			audioData = append(audioData, chunk.Audio...)
		case <-timeout.C:
			s2s.RecordSpanError(span, fmt.Errorf("timeout waiting for audio"))
			return nil, s2s.NewS2SError("Process", s2s.ErrCodeTimeout,
				fmt.Errorf("timeout waiting for audio output"))
		case <-ctx.Done():
			s2s.RecordSpanError(span, ctx.Err())
			return nil, s2s.NewS2SError("Process", s2s.ErrCodeContextCanceled,
				fmt.Errorf("context cancelled: %w", ctx.Err()))
		}
	}

done:
	output := &internal.AudioOutput{
		Data: audioData,
		Format: internal.AudioFormat{
			SampleRate: p.config.SampleRate,
			Channels:   input.Format.Channels,
			BitDepth:   input.Format.BitDepth,
			Encoding:   p.config.AudioFormat,
		},
		Timestamp: time.Now(),
		Provider:  p.providerName,
		VoiceCharacteristics: internal.VoiceCharacteristics{
			VoiceID:  p.config.VoiceID,
			Language: p.config.LanguageCode,
		},
		Latency: time.Since(startTime),
	}

	s2s.RecordSpanLatency(span, output.Latency)
	s2s.RecordSpanAttributes(span, map[string]string{
		"output_size": fmt.Sprintf("%d", len(output.Data)),
		"latency_ms":  fmt.Sprintf("%d", output.Latency.Milliseconds()),
	})

	return output, nil
}

// Name implements the S2SProvider interface.
func (p *OpenAIRealtimeProvider) Name() string {
	return p.providerName
}

// OpenAIRealtimeProvider implements StreamingS2SProvider interface.
var _ iface.StreamingS2SProvider = (*OpenAIRealtimeProvider)(nil)

// StartStreaming implements the StreamingS2SProvider interface.
func (p *OpenAIRealtimeProvider) StartStreaming(ctx context.Context, convCtx *internal.ConversationContext, opts ...internal.STSOption) (iface.StreamingSession, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, s2s.NewS2SError("StartStreaming", s2s.ErrCodeContextCanceled,
			fmt.Errorf("context cancelled: %w", ctx.Err()))
	default:
	}

	if !p.config.EnableStreaming {
		return nil, s2s.NewS2SError("StartStreaming", s2s.ErrCodeInvalidConfig,
			errors.New("streaming is disabled in configuration"))
	}

	session, err := NewOpenAIRealtimeStreamingSession(ctx, p.config, p)
	if err != nil {
		return nil, s2s.NewS2SError("StartStreaming", s2s.ErrCodeStreamError,
			fmt.Errorf("failed to create streaming session: %w", err))
	}

	return session, nil
}
