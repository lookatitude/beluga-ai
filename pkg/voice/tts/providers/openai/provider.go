package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
	ttsiface "github.com/lookatitude/beluga-ai/pkg/voice/tts/iface"
)

// OpenAIProvider implements the TTSProvider interface for OpenAI TTS.
type OpenAIProvider struct {
	config     *OpenAIConfig
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewOpenAIProvider creates a new OpenAI TTS provider.
func NewOpenAIProvider(config *tts.Config) (ttsiface.TTSProvider, error) {
	if config == nil {
		return nil, tts.NewTTSError("NewOpenAIProvider", tts.ErrCodeInvalidConfig,
			errors.New("config cannot be nil"))
	}

	// Convert base config to OpenAI config
	oaConfig := &OpenAIConfig{
		Config: config,
	}

	// Set defaults if not provided
	if oaConfig.Model == "" {
		oaConfig.Model = "tts-1"
	}
	if oaConfig.Voice == "" {
		oaConfig.Voice = "alloy"
	}
	if oaConfig.ResponseFormat == "" {
		oaConfig.ResponseFormat = "mp3"
	}
	if oaConfig.Speed == 0 {
		oaConfig.Speed = 1.0
	}
	// Use BaseURL from embedded Config if available
	if oaConfig.BaseURL == "" {
		if config.BaseURL != "" {
			oaConfig.BaseURL = config.BaseURL
		} else {
			oaConfig.BaseURL = "https://api.openai.com"
		}
	}
	if oaConfig.Timeout == 0 {
		oaConfig.Timeout = 30 * time.Second
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: oaConfig.Timeout,
	}

	return &OpenAIProvider{
		config:     oaConfig,
		httpClient: httpClient,
	}, nil
}

// GenerateSpeech implements the TTSProvider interface using OpenAI TTS API.
func (p *OpenAIProvider) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	startTime := time.Now()

	// Build request URL
	url := p.config.BaseURL + "/v1/audio/speech"

	// Build request body
	requestBody := map[string]any{
		"model": p.config.Model,
		"input": text,
		"voice": p.config.Voice,
	}

	if p.config.ResponseFormat != "" {
		requestBody["response_format"] = p.config.ResponseFormat
	}
	if p.config.Speed > 0 {
		requestBody["speed"] = p.config.Speed
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, tts.NewTTSError("GenerateSpeech", tts.ErrCodeInvalidRequest, err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, tts.NewTTSError("GenerateSpeech", tts.ErrCodeInvalidRequest, err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	// Execute request with retry logic
	var resp *http.Response
	maxRetries := p.config.MaxRetries
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			delay := time.Duration(float64(p.config.RetryDelay) * float64(attempt) * p.config.RetryBackoff)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err = p.httpClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		// Check if error is retryable
		if err != nil && !tts.IsRetryableError(err) {
			break
		}
		if resp != nil && resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			break
		}

		if resp != nil {
			_ = resp.Body.Close()
		}
	}

	if err != nil {
		_ = time.Since(startTime) // Record duration for potential metrics
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, tts.NewTTSError("GenerateSpeech", tts.ErrCodeTimeout, err)
		}
		return nil, tts.ErrorFromHTTPStatus("GenerateSpeech", 0, err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		_ = time.Since(startTime) // Record duration for potential metrics
		return nil, tts.ErrorFromHTTPStatus("GenerateSpeech", resp.StatusCode, fmt.Errorf("HTTP %d", resp.StatusCode))
	}

	// Read audio data
	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, tts.NewTTSError("GenerateSpeech", tts.ErrCodeInvalidResponse, err)
	}

	if len(audio) == 0 {
		return nil, tts.NewTTSError("GenerateSpeech", tts.ErrCodeEmptyResponse,
			errors.New("no audio data in response"))
	}

	duration := time.Since(startTime)

	// Record metrics if enabled
	if p.config.EnableMetrics {
		metrics := tts.GetMetrics()
		if metrics != nil {
			metrics.RecordGeneration(ctx, "openai", p.config.Model, p.config.Voice, duration)
		}
	}

	return audio, nil
}

// StreamGenerate implements the TTSProvider interface
// OpenAI TTS API doesn't support streaming, so we return the full audio as a reader.
func (p *OpenAIProvider) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	// Generate speech first
	audio, err := p.GenerateSpeech(ctx, text)
	if err != nil {
		return nil, err
	}

	// Return as reader
	return bytes.NewReader(audio), nil
}
