package deepgram

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

// TranscribeREST performs transcription using Deepgram REST API as a fallback.
func (p *DeepgramProvider) TranscribeREST(ctx context.Context, audio []byte) (string, error) {
	startTime := time.Now()

	// Build request URL with query parameters
	url := fmt.Sprintf("%s/v1/listen?model=%s&language=%s&punctuate=%t&smart_format=%t",
		p.config.BaseURL,
		p.config.Model,
		p.config.Language,
		p.config.Punctuate,
		p.config.SmartFormat,
	)

	// Add optional parameters
	if p.config.Diarize {
		url += "&diarize=true"
	}
	if p.config.Multichannel {
		url += "&multichannel=true"
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(audio))
	if err != nil {
		return "", stt.NewSTTError("TranscribeREST", stt.ErrCodeInvalidRequest, err)
	}

	// Set headers
	req.Header.Set("Authorization", "Token "+p.config.APIKey)
	req.Header.Set("Content-Type", "audio/wav")

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
		Metadata struct {
			ModelInfo struct {
				Name string `json:"name"`
			} `json:"model_info"`
		} `json:"metadata"`
		Results struct {
			Channels []struct {
				Alternatives []struct {
					Transcript string  `json:"transcript"`
					Confidence float64 `json:"confidence"`
				} `json:"alternatives"`
			} `json:"channels"`
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
	if len(response.Results.Channels) == 0 ||
		len(response.Results.Channels[0].Alternatives) == 0 {
		return "", stt.NewSTTError("TranscribeREST", stt.ErrCodeEmptyResponse,
			errors.New("no transcript in response"))
	}

	transcript := response.Results.Channels[0].Alternatives[0].Transcript
	duration := time.Since(startTime)

	// Record metrics if enabled
	if p.config.EnableMetrics {
		metrics := stt.GetMetrics()
		if metrics != nil {
			metrics.RecordTranscription(ctx, "deepgram", p.config.Model, duration)
		}
	}

	return transcript, nil
}
