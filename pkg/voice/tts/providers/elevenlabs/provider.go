package elevenlabs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
	ttsiface "github.com/lookatitude/beluga-ai/pkg/voice/tts/iface"
)

// ElevenLabsProvider implements the TTSProvider interface for ElevenLabs
type ElevenLabsProvider struct {
	config     *ElevenLabsConfig
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewElevenLabsProvider creates a new ElevenLabs provider
func NewElevenLabsProvider(config *tts.Config) (ttsiface.TTSProvider, error) {
	if config == nil {
		return nil, tts.NewTTSError("NewElevenLabsProvider", tts.ErrCodeInvalidConfig,
			fmt.Errorf("config cannot be nil"))
	}

	// Convert base config to ElevenLabs config
	elConfig := &ElevenLabsConfig{
		Config: config,
	}

	// Set defaults if not provided
	if elConfig.ModelID == "" {
		elConfig.ModelID = "eleven_monolingual_v1"
	}
	if elConfig.Stability == 0 {
		elConfig.Stability = 0.5
	}
	if elConfig.SimilarityBoost == 0 {
		elConfig.SimilarityBoost = 0.5
	}
	if elConfig.OutputFormat == "" {
		elConfig.OutputFormat = "mp3_44100_128"
	}
	// Use BaseURL from embedded Config if available
	if elConfig.BaseURL == "" {
		if config.BaseURL != "" {
			elConfig.BaseURL = config.BaseURL
		} else {
			elConfig.BaseURL = "https://api.elevenlabs.io"
		}
	}
	if elConfig.Timeout == 0 {
		elConfig.Timeout = 30 * time.Second
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: elConfig.Timeout,
	}

	return &ElevenLabsProvider{
		config:     elConfig,
		httpClient: httpClient,
	}, nil
}

// GenerateSpeech implements the TTSProvider interface using ElevenLabs API
func (p *ElevenLabsProvider) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	startTime := time.Now()

	// Build request URL
	url := fmt.Sprintf("%s/v1/text-to-speech/%s", p.config.BaseURL, p.config.VoiceID)

	// Build request body
	requestBody := map[string]interface{}{
		"text":     text,
		"model_id": p.config.ModelID,
		"voice_settings": map[string]interface{}{
			"stability":         p.config.Stability,
			"similarity_boost":  p.config.SimilarityBoost,
			"style":             p.config.Style,
			"use_speaker_boost": p.config.UseSpeakerBoost,
		},
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
	req.Header.Set("xi-api-key", p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")

	// Add output format if specified
	if p.config.OutputFormat != "" {
		req.Header.Set("output_format", p.config.OutputFormat)
	}

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
			resp.Body.Close()
		}
	}

	if err != nil {
		_ = time.Since(startTime) // Record duration for potential metrics
		if ctx.Err() == context.DeadlineExceeded {
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
			fmt.Errorf("no audio data in response"))
	}

	duration := time.Since(startTime)

	// Record metrics if enabled
	if p.config.EnableMetrics {
		metrics := tts.GetMetrics()
		if metrics != nil {
			metrics.RecordGeneration(ctx, "elevenlabs", p.config.ModelID, p.config.VoiceID, duration)
		}
	}

	return audio, nil
}

// StreamGenerate implements the TTSProvider interface
// ElevenLabs supports streaming, but for simplicity we return the full audio as a reader
func (p *ElevenLabsProvider) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	// Generate speech first
	audio, err := p.GenerateSpeech(ctx, text)
	if err != nil {
		return nil, err
	}

	// Return as reader
	return bytes.NewReader(audio), nil
}
