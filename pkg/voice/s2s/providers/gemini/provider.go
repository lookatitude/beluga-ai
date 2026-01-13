package gemini

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

// GeminiNativeProvider implements the S2SProvider interface for Gemini 2.5 Flash Native Audio.
type GeminiNativeProvider struct {
	config       *GeminiNativeConfig
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
type ProviderOption func(*GeminiNativeProvider)

// WithHTTPClient sets a custom HTTP client for the provider.
// This is useful for testing with mock HTTP clients.
func WithHTTPClient(client HTTPClient) ProviderOption {
	return func(p *GeminiNativeProvider) {
		p.httpClient = client
	}
}

// NewGeminiNativeProvider creates a new Gemini 2.5 Flash Native Audio provider.
func NewGeminiNativeProvider(config *s2s.Config, opts ...ProviderOption) (iface.S2SProvider, error) {
	if config == nil {
		return nil, s2s.NewS2SError("NewGeminiNativeProvider", s2s.ErrCodeInvalidConfig,
			errors.New("config cannot be nil"))
	}

	// Convert base config to Gemini Native config
	geminiConfig := &GeminiNativeConfig{
		Config: config,
	}

	// Set defaults if not provided
	if geminiConfig.APIEndpoint == "" {
		geminiConfig.APIEndpoint = "https://generativelanguage.googleapis.com/v1beta"
	}
	if geminiConfig.Model == "" {
		geminiConfig.Model = "gemini-2.5-flash"
	}
	if geminiConfig.Location == "" {
		geminiConfig.Location = "us-central1"
	}
	if geminiConfig.VoiceID == "" {
		geminiConfig.VoiceID = "default"
	}
	if geminiConfig.LanguageCode == "" {
		geminiConfig.LanguageCode = "en-US"
	}
	if geminiConfig.Timeout == 0 {
		geminiConfig.Timeout = 30 * time.Second
	}
	if geminiConfig.SampleRate == 0 {
		geminiConfig.SampleRate = 24000
	}
	if geminiConfig.AudioFormat == "" {
		geminiConfig.AudioFormat = "pcm"
	}
	if geminiConfig.Temperature == 0 {
		geminiConfig.Temperature = 0.8
	}
	// Enable streaming by default (bool defaults to false, so we set it to true)
	geminiConfig.EnableStreaming = true

	// Validate API key
	if geminiConfig.APIKey == "" {
		return nil, s2s.NewS2SError("NewGeminiNativeProvider", s2s.ErrCodeInvalidConfig,
			errors.New("API key is required"))
	}

	provider := &GeminiNativeProvider{
		config:       geminiConfig,
		providerName: "gemini",
	}

	// Apply options
	for _, opt := range opts {
		opt(provider)
	}

	// Set default HTTP client if not provided
	if provider.httpClient == nil {
		provider.httpClient = &http.Client{Timeout: geminiConfig.Timeout}
	}

	return provider, nil
}

// Process implements the S2SProvider interface using Gemini 2.5 Flash Native Audio API.
func (p *GeminiNativeProvider) Process(ctx context.Context, input *internal.AudioInput, convCtx *internal.ConversationContext, opts ...internal.STSOption) (*internal.AudioOutput, error) {
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
	requestBody, err := p.prepareGeminiRequest(input, convCtx, stsOpts)
	if err != nil {
		s2s.RecordSpanError(span, err)
		return nil, s2s.NewS2SError("Process", s2s.ErrCodeInvalidRequest,
			fmt.Errorf("failed to prepare request: %w", err))
	}

	// Build API URL
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.config.APIEndpoint, p.config.Model, p.config.APIKey)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		s2s.RecordSpanError(span, err)
		return nil, s2s.NewS2SError("Process", s2s.ErrCodeInvalidRequest,
			fmt.Errorf("failed to create request: %w", err))
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request using injected HTTP client
	resp, err := p.httpClient.Do(req)
	if err != nil {
		s2s.RecordSpanError(span, err)
		s2s.RecordSpanLatency(span, time.Since(startTime))
		return nil, p.handleGeminiError("Process", err)
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
		return nil, p.handleGeminiError("Process", fmt.Errorf("API error: %s", string(respBody)))
	}

	// Parse response
	output, err := p.parseGeminiResponse(respBody, input)
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

// prepareGeminiRequest prepares the request body for Gemini Native Audio API.
func (p *GeminiNativeProvider) prepareGeminiRequest(input *internal.AudioInput, convCtx *internal.ConversationContext, opts *internal.STSOptions) ([]byte, error) {
	// Encode audio as base64
	audioBase64 := base64.StdEncoding.EncodeToString(input.Data)

	// Build request payload
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
			"temperature": p.config.Temperature,
		},
	}

	// Add conversation history if provided
	if convCtx != nil && len(convCtx.History) > 0 {
		history := make([]map[string]any, 0, len(convCtx.History))
		for _, turn := range convCtx.History {
			history = append(history, map[string]any{
				"role": turn.Role,
				"parts": []map[string]any{
					{"text": turn.Content},
				},
			})
		}
		request["history"] = history
	}

	return json.Marshal(request)
}

// parseGeminiResponse parses the response from Gemini Native Audio API.
func (p *GeminiNativeProvider) parseGeminiResponse(responseBody []byte, input *internal.AudioInput) (*internal.AudioOutput, error) {
	var response struct {
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

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := response.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("no parts in candidate")
	}

	part := candidate.Content.Parts[0]
	if part.InlineData.Data == "" {
		return nil, fmt.Errorf("no audio data in response")
	}

	// Decode base64 audio
	audioData, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio data: %w", err)
	}

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
	}

	return output, nil
}

// handleGeminiError handles errors from Gemini API.
func (p *GeminiNativeProvider) handleGeminiError(operation string, err error) error {
	if err == nil {
		return nil
	}

	var errorCode string
	var message string

	errStr := err.Error()
	if strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit") {
		errorCode = s2s.ErrCodeRateLimit
		message = "Gemini API rate limit exceeded"
	} else if strings.Contains(errStr, "401") || strings.Contains(errStr, "403") || strings.Contains(errStr, "authentication") {
		errorCode = s2s.ErrCodeAuthentication
		message = "Gemini API authentication failed"
	} else if strings.Contains(errStr, "quota") {
		errorCode = s2s.ErrCodeQuotaExceeded
		message = "Gemini API quota exceeded"
	} else {
		errorCode = s2s.ErrCodeInvalidRequest
		message = "Gemini API request failed"
	}

	return s2s.NewS2SErrorWithMessage(operation, errorCode, message, err)
}

// Name implements the S2SProvider interface.
func (p *GeminiNativeProvider) Name() string {
	return p.providerName
}

// GeminiNativeProvider implements StreamingS2SProvider interface.
var _ iface.StreamingS2SProvider = (*GeminiNativeProvider)(nil)

// StartStreaming implements the StreamingS2SProvider interface.
func (p *GeminiNativeProvider) StartStreaming(ctx context.Context, convCtx *internal.ConversationContext, opts ...internal.STSOption) (iface.StreamingSession, error) {
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

	session, err := NewGeminiNativeStreamingSession(ctx, p.config, p)
	if err != nil {
		return nil, s2s.NewS2SError("StartStreaming", s2s.ErrCodeStreamError,
			fmt.Errorf("failed to create streaming session: %w", err))
	}

	return session, nil
}
