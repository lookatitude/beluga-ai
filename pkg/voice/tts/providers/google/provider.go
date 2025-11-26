package google

import (
	"bytes"
	"context"
	"encoding/base64"
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

// GoogleProvider implements the TTSProvider interface for Google Cloud Text-to-Speech.
type GoogleProvider struct {
	config     *GoogleConfig
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewGoogleProvider creates a new Google Cloud Text-to-Speech provider.
func NewGoogleProvider(config *tts.Config) (ttsiface.TTSProvider, error) {
	if config == nil {
		return nil, tts.NewTTSError("NewGoogleProvider", tts.ErrCodeInvalidConfig,
			errors.New("config cannot be nil"))
	}

	// Convert base config to Google config
	gConfig := &GoogleConfig{
		Config: config,
	}

	// Set defaults if not provided
	if gConfig.VoiceName == "" {
		gConfig.VoiceName = "en-US-Standard-A"
	}
	if gConfig.LanguageCode == "" {
		gConfig.LanguageCode = "en-US"
	}
	if gConfig.SSMLGender == "" {
		gConfig.SSMLGender = "NEUTRAL"
	}
	if gConfig.SpeakingRate == 0 {
		gConfig.SpeakingRate = 1.0
	}
	if gConfig.AudioEncoding == "" {
		gConfig.AudioEncoding = "MP3"
	}
	if gConfig.SampleRateHertz == 0 {
		gConfig.SampleRateHertz = 24000
	}
	// Use BaseURL from embedded Config if available
	if gConfig.BaseURL == "" {
		if config.BaseURL != "" {
			gConfig.BaseURL = config.BaseURL
		} else {
			gConfig.BaseURL = "https://texttospeech.googleapis.com"
		}
	}
	if gConfig.Timeout == 0 {
		gConfig.Timeout = 30 * time.Second
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: gConfig.Timeout,
	}

	return &GoogleProvider{
		config:     gConfig,
		httpClient: httpClient,
	}, nil
}

// GenerateSpeech implements the TTSProvider interface using Google Cloud Text-to-Speech API.
func (p *GoogleProvider) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	startTime := time.Now()

	// Build request URL
	url := p.config.BaseURL + "/v1/text:synthesize"
	if p.config.ProjectID != "" {
		url = fmt.Sprintf("%s/v1/projects/%s/locations/global:text:synthesize", p.config.BaseURL, p.config.ProjectID)
	}

	// Build request body
	requestBody := map[string]any{
		"input": map[string]any{
			"text": text,
		},
		"voice": map[string]any{
			"languageCode": p.config.LanguageCode,
			"name":         p.config.VoiceName,
			"ssmlGender":   p.config.SSMLGender,
		},
		"audioConfig": map[string]any{
			"audioEncoding":   p.config.AudioEncoding,
			"sampleRateHertz": p.config.SampleRateHertz,
		},
	}

	if p.config.SpeakingRate > 0 {
		requestBody["audioConfig"].(map[string]any)["speakingRate"] = p.config.SpeakingRate
	}
	if p.config.Pitch != 0 {
		requestBody["audioConfig"].(map[string]any)["pitch"] = p.config.Pitch
	}
	if p.config.VolumeGainDb != 0 {
		requestBody["audioConfig"].(map[string]any)["volumeGainDb"] = p.config.VolumeGainDb
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
	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
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

	// Parse response
	var response struct {
		AudioContent string `json:"audioContent"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, tts.NewTTSError("GenerateSpeech", tts.ErrCodeInvalidResponse, err)
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, tts.NewTTSError("GenerateSpeech", tts.ErrCodeMalformedResponse, err)
	}

	if response.AudioContent == "" {
		return nil, tts.NewTTSError("GenerateSpeech", tts.ErrCodeEmptyResponse,
			errors.New("no audio content in response"))
	}

	// Decode base64 audio
	audio, err := base64.StdEncoding.DecodeString(response.AudioContent)
	if err != nil {
		return nil, tts.NewTTSError("GenerateSpeech", tts.ErrCodeMalformedResponse, err)
	}

	if len(audio) == 0 {
		return nil, tts.NewTTSError("GenerateSpeech", tts.ErrCodeEmptyResponse,
			errors.New("decoded audio is empty"))
	}

	duration := time.Since(startTime)

	// Record metrics if enabled
	if p.config.EnableMetrics {
		metrics := tts.GetMetrics()
		if metrics != nil {
			metrics.RecordGeneration(ctx, "google", p.config.VoiceName, p.config.LanguageCode, duration)
		}
	}

	return audio, nil
}

// StreamGenerate implements the TTSProvider interface
// Google Cloud Text-to-Speech API doesn't support streaming, so we return the full audio as a reader.
func (p *GoogleProvider) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	// Generate speech first
	audio, err := p.GenerateSpeech(ctx, text)
	if err != nil {
		return nil, err
	}

	// Return as reader
	return bytes.NewReader(audio), nil
}
