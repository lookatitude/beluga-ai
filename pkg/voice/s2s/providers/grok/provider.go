package grok

import (
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
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
)

// GrokVoiceProvider implements the S2SProvider interface for Grok Voice Agent.
type GrokVoiceProvider struct {
	config       *GrokVoiceConfig
	httpClient   HTTPClient
	mu           sync.RWMutex
	providerName string
}

// HTTPClient is an interface for HTTP client operations.
// This allows dependency injection for testing.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// ProviderOption is a function type for configuring the provider.
type ProviderOption func(*GrokVoiceProvider)

// WithHTTPClient sets a custom HTTP client for the provider.
// This is useful for testing with mock HTTP clients.
func WithHTTPClient(client HTTPClient) ProviderOption {
	return func(p *GrokVoiceProvider) {
		p.httpClient = client
	}
}

// NewGrokVoiceProvider creates a new Grok Voice Agent provider.
func NewGrokVoiceProvider(config *s2s.Config, opts ...ProviderOption) (iface.S2SProvider, error) {
	if config == nil {
		return nil, s2s.NewS2SError("NewGrokVoiceProvider", s2s.ErrCodeInvalidConfig,
			errors.New("config cannot be nil"))
	}

	// Convert base config to Grok Voice config
	grokConfig := &GrokVoiceConfig{
		Config: config,
	}

	// Set defaults if not provided
	if grokConfig.APIEndpoint == "" {
		grokConfig.APIEndpoint = "https://api.x.ai/v1"
	}
	if grokConfig.Model == "" {
		grokConfig.Model = "grok-voice-agent"
	}
	if grokConfig.VoiceID == "" {
		grokConfig.VoiceID = "alloy"
	}
	if grokConfig.LanguageCode == "" {
		grokConfig.LanguageCode = "en-US"
	}
	if grokConfig.Timeout == 0 {
		grokConfig.Timeout = 30 * time.Second
	}
	if grokConfig.SampleRate == 0 {
		grokConfig.SampleRate = 24000
	}
	if grokConfig.AudioFormat == "" {
		grokConfig.AudioFormat = "pcm"
	}
	if grokConfig.Temperature == 0 {
		grokConfig.Temperature = 0.8
	}

	// Validate API key
	if grokConfig.APIKey == "" {
		return nil, s2s.NewS2SError("NewGrokVoiceProvider", s2s.ErrCodeInvalidConfig,
			errors.New("API key is required"))
	}

	provider := &GrokVoiceProvider{
		config:       grokConfig,
		providerName: "grok",
	}

	// Apply options
	for _, opt := range opts {
		opt(provider)
	}

	// Set default HTTP client if not provided
	if provider.httpClient == nil {
		provider.httpClient = &http.Client{Timeout: grokConfig.Timeout}
	}

	return provider, nil
}

// Process implements the S2SProvider interface using Grok Voice Agent API.
func (p *GrokVoiceProvider) Process(ctx context.Context, input *internal.AudioInput, convCtx *internal.ConversationContext, opts ...internal.STSOption) (*internal.AudioOutput, error) {
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

	// Prepare request
	requestBody, err := p.prepareGrokRequest(input, convCtx, stsOpts)
	if err != nil {
		s2s.RecordSpanError(span, err)
		return nil, s2s.NewS2SError("Process", s2s.ErrCodeInvalidRequest,
			fmt.Errorf("failed to prepare request: %w", err))
	}

	// Build API URL
	url := fmt.Sprintf("%s/audio/speech", p.config.APIEndpoint)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		s2s.RecordSpanError(span, err)
		return nil, s2s.NewS2SError("Process", s2s.ErrCodeInvalidRequest,
			fmt.Errorf("failed to create request: %w", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))

	// Execute request using injected HTTP client
	resp, err := p.httpClient.Do(req)
	if err != nil {
		s2s.RecordSpanError(span, err)
		s2s.RecordSpanLatency(span, time.Since(startTime))
		return nil, p.handleGrokError("Process", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s2s.RecordSpanError(span, err)
		s2s.RecordSpanLatency(span, time.Since(startTime))
		return nil, s2s.NewS2SError("Process", s2s.ErrCodeInvalidResponse,
			fmt.Errorf("failed to read response: %w", err))
	}

	if resp.StatusCode != http.StatusOK {
		s2s.RecordSpanError(span, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody)))
		s2s.RecordSpanLatency(span, time.Since(startTime))
		return nil, p.handleGrokError("Process", fmt.Errorf("API error: %s", string(respBody)))
	}

	// Parse response
	output, err := p.parseGrokResponse(respBody, input)
	if err != nil {
		s2s.RecordSpanError(span, err)
		s2s.RecordSpanLatency(span, time.Since(startTime))
		return nil, s2s.NewS2SError("Process", s2s.ErrCodeInvalidResponse,
			fmt.Errorf("failed to parse response: %w", err))
	}

	output.Latency = time.Since(startTime)
	s2s.RecordSpanLatency(span, output.Latency)
	s2s.RecordSpanAttributes(span, map[string]string{
		"output_size": fmt.Sprintf("%d", len(output.Data)),
		"latency_ms":  fmt.Sprintf("%d", output.Latency.Milliseconds()),
	})

	return output, nil
}

// prepareGrokRequest prepares the request body for Grok Voice Agent API.
func (p *GrokVoiceProvider) prepareGrokRequest(input *internal.AudioInput, convCtx *internal.ConversationContext, opts *internal.STSOptions) ([]byte, error) {
	// Encode audio as base64
	audioBase64 := base64.StdEncoding.EncodeToString(input.Data)

	// Build request payload
	request := map[string]any{
		"model": p.config.Model,
		"input": map[string]any{
			"audio": audioBase64,
			"format": map[string]any{
				"sample_rate": input.Format.SampleRate,
				"channels":    input.Format.Channels,
				"encoding":    input.Format.Encoding,
			},
		},
		"voice":           p.config.VoiceID,
		"response_format": p.config.AudioFormat,
		"temperature":     p.config.Temperature,
	}

	// Add conversation context if provided
	if convCtx != nil {
		if convCtx.ConversationID != "" {
			request["conversation_id"] = convCtx.ConversationID
		}
		if len(convCtx.History) > 0 {
			history := make([]map[string]any, 0, len(convCtx.History))
			for _, turn := range convCtx.History {
				history = append(history, map[string]any{
					"role":      turn.Role,
					"content":   turn.Content,
					"timestamp": turn.Timestamp.Unix(),
				})
			}
			request["conversation_history"] = history
		}
	}

	// Add options
	if opts != nil {
		if opts.Language != "" {
			request["language"] = opts.Language
		}
		if opts.VoiceID != "" {
			request["voice"] = opts.VoiceID
		}
	}

	return json.Marshal(request)
}

// parseGrokResponse parses the response from Grok Voice Agent API.
func (p *GrokVoiceProvider) parseGrokResponse(responseBody []byte, input *internal.AudioInput) (*internal.AudioOutput, error) {
	var response struct {
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

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Audio == "" {
		return nil, fmt.Errorf("no audio data in response")
	}

	// Decode base64 audio
	audioData, err := base64.StdEncoding.DecodeString(response.Audio)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio data: %w", err)
	}

	output := &internal.AudioOutput{
		Data: audioData,
		Format: internal.AudioFormat{
			SampleRate: response.Format.SampleRate,
			Channels:   response.Format.Channels,
			BitDepth:   input.Format.BitDepth,
			Encoding:   response.Format.Encoding,
		},
		Timestamp: time.Now(),
		Provider:  p.providerName,
		VoiceCharacteristics: internal.VoiceCharacteristics{
			VoiceID:  response.Voice.VoiceID,
			Language: response.Voice.Language,
		},
	}

	return output, nil
}

// handleGrokError handles errors from Grok API.
func (p *GrokVoiceProvider) handleGrokError(operation string, err error) error {
	if err == nil {
		return nil
	}

	var errorCode string
	var message string

	errStr := err.Error()
	if strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit") {
		errorCode = s2s.ErrCodeRateLimit
		message = "Grok API rate limit exceeded"
	} else if strings.Contains(errStr, "401") || strings.Contains(errStr, "403") || strings.Contains(errStr, "authentication") {
		errorCode = s2s.ErrCodeAuthentication
		message = "Grok API authentication failed"
	} else if strings.Contains(errStr, "quota") {
		errorCode = s2s.ErrCodeQuotaExceeded
		message = "Grok API quota exceeded"
	} else {
		errorCode = s2s.ErrCodeInvalidRequest
		message = "Grok API request failed"
	}

	return s2s.NewS2SErrorWithMessage(operation, errorCode, message, err)
}

// Name implements the S2SProvider interface.
func (p *GrokVoiceProvider) Name() string {
	return p.providerName
}

// GrokVoiceProvider implements StreamingS2SProvider interface.
var _ iface.StreamingS2SProvider = (*GrokVoiceProvider)(nil)

// StartStreaming implements the StreamingS2SProvider interface.
func (p *GrokVoiceProvider) StartStreaming(ctx context.Context, convCtx *internal.ConversationContext, opts ...internal.STSOption) (iface.StreamingSession, error) {
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

	// Ensure HTTP client is set for streaming
	if p.httpClient == nil {
		p.httpClient = &http.Client{Timeout: p.config.Timeout}
	}

	session, err := NewGrokVoiceStreamingSession(ctx, p.config, p)
	if err != nil {
		return nil, s2s.NewS2SError("StartStreaming", s2s.ErrCodeStreamError,
			fmt.Errorf("failed to create streaming session: %w", err))
	}

	return session, nil
}
