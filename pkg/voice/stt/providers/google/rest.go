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
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
)

// TranscribeREST performs transcription using Google Cloud Speech-to-Text REST API.
func (p *GoogleProvider) TranscribeREST(ctx context.Context, audio []byte) (string, error) {
	startTime := time.Now()

	// Build request URL
	url := p.config.BaseURL + "/v1/speech:recognize"
	if p.config.ProjectID != "" {
		url = fmt.Sprintf("%s/v1/projects/%s/locations/global:recognize", p.config.BaseURL, p.config.ProjectID)
	}

	// Encode audio as base64
	audioContent := base64.StdEncoding.EncodeToString(audio)

	// Build recognition config
	recognitionConfig := map[string]any{
		"encoding":                   "LINEAR16",
		"sampleRateHertz":            p.config.SampleRate,
		"languageCode":               p.config.LanguageCode,
		"model":                      p.config.Model,
		"enableAutomaticPunctuation": p.config.EnableAutomaticPunctuation,
		"enableWordTimeOffsets":      p.config.EnableWordTimeOffsets,
		"enableWordConfidence":       p.config.EnableWordConfidence,
		"useEnhanced":                p.config.UseEnhanced,
	}

	if p.config.EnableSpeakerDiarization {
		recognitionConfig["enableSpeakerDiarization"] = true
		recognitionConfig["diarizationSpeakerCount"] = p.config.DiarizationSpeakerCount
	}

	if len(p.config.AlternativeLanguageCodes) > 0 {
		recognitionConfig["alternativeLanguageCodes"] = p.config.AlternativeLanguageCodes
	}

	// Build request body
	requestBody := map[string]any{
		"config": recognitionConfig,
		"audio": map[string]any{
			"content": audioContent,
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", stt.NewSTTError("TranscribeREST", stt.ErrCodeInvalidRequest, err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", stt.NewSTTError("TranscribeREST", stt.ErrCodeInvalidRequest, err)
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
				return "", ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err = p.httpClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		// Check if error is retryable
		if err != nil && !stt.IsRetryableError(err) {
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
			return "", stt.NewSTTError("TranscribeREST", stt.ErrCodeTimeout, err)
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return "", ctx.Err()
		}
		return "", stt.ErrorFromHTTPStatus("TranscribeREST", 0, err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		_ = time.Since(startTime) // Record duration for potential metrics
		return "", stt.ErrorFromHTTPStatus("TranscribeREST", resp.StatusCode, fmt.Errorf("HTTP %d", resp.StatusCode))
	}

	// Parse response
	var response struct {
		Results []struct {
			Alternatives []struct {
				Transcript string  `json:"transcript"`
				Confidence float64 `json:"confidence"`
			} `json:"alternatives"`
		} `json:"results"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", stt.NewSTTError("TranscribeREST", stt.ErrCodeInvalidResponse, err)
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", stt.NewSTTError("TranscribeREST", stt.ErrCodeMalformedResponse, err)
	}

	// Extract transcript
	if len(response.Results) == 0 || len(response.Results[0].Alternatives) == 0 {
		return "", stt.NewSTTError("TranscribeREST", stt.ErrCodeEmptyResponse,
			errors.New("no transcript in response"))
	}

	transcript := response.Results[0].Alternatives[0].Transcript
	if transcript == "" {
		return "", stt.NewSTTError("TranscribeREST", stt.ErrCodeEmptyResponse,
			errors.New("no transcript in response"))
	}
	duration := time.Since(startTime)

	// Record metrics if enabled
	if p.config.EnableMetrics {
		metrics := stt.GetMetrics()
		if metrics != nil {
			metrics.RecordTranscription(ctx, "google", p.config.Model, duration)
		}
	}

	return transcript, nil
}
