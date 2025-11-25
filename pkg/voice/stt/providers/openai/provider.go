package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
	sttiface "github.com/lookatitude/beluga-ai/pkg/voice/stt/iface"
)

// OpenAIProvider implements the STTProvider interface for OpenAI Whisper
type OpenAIProvider struct {
	config     *OpenAIConfig
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewOpenAIProvider creates a new OpenAI Whisper provider
func NewOpenAIProvider(config *stt.Config) (sttiface.STTProvider, error) {
	if config == nil {
		return nil, stt.NewSTTError("NewOpenAIProvider", stt.ErrCodeInvalidConfig,
			fmt.Errorf("config cannot be nil"))
	}

	// Convert base config to OpenAI config
	oaConfig := &OpenAIConfig{
		Config: config,
	}

	// Set defaults if not provided
	if oaConfig.Model == "" {
		oaConfig.Model = "whisper-1"
	}
	// Use BaseURL from embedded Config if available, otherwise use default
	if oaConfig.BaseURL == "" {
		if config.BaseURL != "" {
			oaConfig.BaseURL = config.BaseURL
		} else {
			oaConfig.BaseURL = "https://api.openai.com"
		}
	}
	// Copy Language from embedded Config if not set
	if oaConfig.Language == "" && config.Language != "" {
		oaConfig.Language = config.Language
	}
	if oaConfig.Timeout == 0 {
		oaConfig.Timeout = 30 * time.Second
	}
	if oaConfig.ResponseFormat == "" {
		oaConfig.ResponseFormat = "json"
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

// Transcribe implements the STTProvider interface using OpenAI Whisper API
func (p *OpenAIProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	startTime := time.Now()

	// Build request URL
	url := fmt.Sprintf("%s/v1/audio/transcriptions", p.config.BaseURL)

	// Create multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add file field
	fileWriter, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", stt.NewSTTError("Transcribe", stt.ErrCodeInvalidRequest, err)
	}
	if _, err := fileWriter.Write(audio); err != nil {
		return "", stt.NewSTTError("Transcribe", stt.ErrCodeInvalidRequest, err)
	}

	// Add model field
	if err := writer.WriteField("model", p.config.Model); err != nil {
		return "", stt.NewSTTError("Transcribe", stt.ErrCodeInvalidRequest, err)
	}

	// Add optional fields
	if p.config.Language != "" {
		if err := writer.WriteField("language", p.config.Language); err != nil {
			return "", stt.NewSTTError("Transcribe", stt.ErrCodeInvalidRequest, err)
		}
	}
	if p.config.Prompt != "" {
		if err := writer.WriteField("prompt", p.config.Prompt); err != nil {
			return "", stt.NewSTTError("Transcribe", stt.ErrCodeInvalidRequest, err)
		}
	}
	if err := writer.WriteField("response_format", p.config.ResponseFormat); err != nil {
		return "", stt.NewSTTError("Transcribe", stt.ErrCodeInvalidRequest, err)
	}
	if p.config.Temperature > 0 {
		if err := writer.WriteField("temperature", fmt.Sprintf("%.2f", p.config.Temperature)); err != nil {
			return "", stt.NewSTTError("Transcribe", stt.ErrCodeInvalidRequest, err)
		}
	}

	if err := writer.Close(); err != nil {
		return "", stt.NewSTTError("Transcribe", stt.ErrCodeInvalidRequest, err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return "", stt.NewSTTError("Transcribe", stt.ErrCodeInvalidRequest, err)
	}

	// Set headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Execute request with retry logic
	var resp *http.Response
	maxRetries := p.config.MaxRetries
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check context before making request
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		if attempt > 0 {
			// Wait before retry
			delay := time.Duration(float64(p.config.RetryDelay) * float64(attempt) * p.config.RetryBackoff)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err = p.httpClient.Do(req)

		// Success case - break immediately
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		// Check if error is retryable
		if err != nil && !stt.IsRetryableError(err) {
			break
		}

		// For HTTP responses, check if we should retry
		// Retry on 429 (TooManyRequests) or 5xx errors
		if resp != nil {
			statusCode := resp.StatusCode
			// Close response body before retrying
			resp.Body.Close()

			// Don't retry on non-retryable status codes (except 429 and 5xx)
			if statusCode != http.StatusTooManyRequests && statusCode < 500 {
				break
			}
			// If we get here, we should retry (429 or 5xx)
		}

		// If we've exhausted retries, break
		if attempt >= maxRetries {
			break
		}
	}

	// Handle final error or non-200 response
	if err != nil {
		_ = time.Since(startTime) // Record duration for potential metrics
		if ctx.Err() == context.DeadlineExceeded {
			return "", stt.NewSTTError("Transcribe", stt.ErrCodeTimeout, err)
		}
		return "", stt.ErrorFromHTTPStatus("Transcribe", 0, err)
	}

	if resp == nil {
		return "", stt.NewSTTError("Transcribe", stt.ErrCodeNetworkError, fmt.Errorf("no response received"))
	}

	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		_ = time.Since(startTime) // Record duration for potential metrics
		return "", stt.ErrorFromHTTPStatus("Transcribe", resp.StatusCode, fmt.Errorf("HTTP %d", resp.StatusCode))
	}

	// Parse response
	var response struct {
		Text string `json:"text"`
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", stt.NewSTTError("Transcribe", stt.ErrCodeInvalidResponse, err)
	}

	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return "", stt.NewSTTError("Transcribe", stt.ErrCodeMalformedResponse, err)
	}

	if response.Text == "" {
		return "", stt.NewSTTError("Transcribe", stt.ErrCodeEmptyResponse,
			fmt.Errorf("no transcript in response"))
	}

	transcript := response.Text
	duration := time.Since(startTime)

	// Record metrics if enabled
	if p.config.EnableMetrics {
		metrics := stt.GetMetrics()
		if metrics != nil {
			metrics.RecordTranscription(ctx, "openai", p.config.Model, duration)
		}
	}

	return transcript, nil
}

// StartStreaming implements the STTProvider interface
// Note: OpenAI Whisper API doesn't support streaming transcription directly
// This would require using a different approach or API
func (p *OpenAIProvider) StartStreaming(ctx context.Context) (iface.StreamingSession, error) {
	// OpenAI Whisper API doesn't support streaming transcription
	// Return an error indicating this limitation
	return nil, stt.NewSTTError("StartStreaming", stt.ErrCodeStreamError,
		fmt.Errorf("OpenAI Whisper API does not support streaming transcription"))
}
