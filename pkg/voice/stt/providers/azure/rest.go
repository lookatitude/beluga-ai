package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
)

// TranscribeREST performs transcription using Azure Speech Services REST API.
func (p *AzureProvider) TranscribeREST(ctx context.Context, audio []byte) (string, error) {
	startTime := time.Now()

	// Build request URL
	url := fmt.Sprintf("%s/speech/recognition/conversation/cognitiveservices/v1?language=%s&format=detailed",
		p.config.GetBaseURL(),
		p.config.Language,
	)

	// Add optional parameters
	if p.config.EnablePunctuation {
		url += "&punctuation=true"
	}
	if p.config.EnableWordLevelTimestamps {
		url += "&wordLevelTimestamps=true"
	}
	if p.config.EnableSpeakerDiarization {
		url += "&diarization=true"
	}
	if p.config.EndpointID != "" {
		url += "&endpointId=" + p.config.EndpointID
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(audio))
	if err != nil {
		return "", stt.NewSTTError("TranscribeREST", stt.ErrCodeInvalidRequest, err)
	}

	// Set headers
	req.Header.Set("Ocp-Apim-Subscription-Key", p.config.APIKey)
	req.Header.Set("Content-Type", "audio/wav")
	req.Header.Set("Accept", "application/json")

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
		RecognitionStatus string `json:"RecognitionStatus"`
		DisplayText       string `json:"DisplayText"`
		Offset            int64  `json:"Offset"`
		Duration          int64  `json:"Duration"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", stt.NewSTTError("TranscribeREST", stt.ErrCodeInvalidResponse, err)
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", stt.NewSTTError("TranscribeREST", stt.ErrCodeMalformedResponse, err)
	}

	// Check recognition status
	if response.RecognitionStatus != "Success" {
		return "", stt.NewSTTError("TranscribeREST", stt.ErrCodeEmptyResponse,
			fmt.Errorf("recognition failed with status: %s", response.RecognitionStatus))
	}

	if response.DisplayText == "" {
		return "", stt.NewSTTError("TranscribeREST", stt.ErrCodeEmptyResponse,
			errors.New("no transcript in response"))
	}

	transcript := response.DisplayText
	duration := time.Since(startTime)

	// Record metrics if enabled
	if p.config.EnableMetrics {
		metrics := stt.GetMetrics()
		if metrics != nil {
			metrics.RecordTranscription(ctx, "azure", p.config.Language, duration)
		}
	}

	return transcript, nil
}
