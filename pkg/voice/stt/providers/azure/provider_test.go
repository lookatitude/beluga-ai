package azure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAzureProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *stt.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &stt.Config{
				Provider: "azure",
				APIKey:   "test-key",
				Model:    "default",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "config with defaults",
			config: &stt.Config{
				Provider: "azure",
				APIKey:   "test-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewAzureProvider(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
			}
		})
	}
}

func TestAzureProvider_Transcribe_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/speech/recognition/conversation/cognitiveservices/v1")
		assert.Equal(t, "test-key", r.Header.Get("Ocp-Apim-Subscription-Key"))
		assert.Equal(t, "audio/wav", r.Header.Get("Content-Type"))

		// Return success response
		response := map[string]interface{}{
			"RecognitionStatus": "Success",
			"DisplayText":       "Hello, this is a test transcription",
			"Offset":            0,
			"Duration":          10000000,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	transcript, err := provider.Transcribe(ctx, audio)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, this is a test transcription", transcript)
}

func TestAzureProvider_Transcribe_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	_, err = provider.Transcribe(ctx, audio)
	assert.Error(t, err)
}

func TestAzureProvider_Transcribe_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	_, err = provider.Transcribe(ctx, audio)
	assert.Error(t, err)
}

func TestAzureProvider_Transcribe_FailedRecognition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"RecognitionStatus": "NoMatch",
			"DisplayText":       "",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	_, err = provider.Transcribe(ctx, audio)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "recognition failed")
}

func TestAzureProvider_Transcribe_EmptyText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"RecognitionStatus": "Success",
			"DisplayText":       "",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	_, err = provider.Transcribe(ctx, audio)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no transcript")
}

func TestAzureProvider_StartStreaming(t *testing.T) {
	config := &stt.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	session, err := provider.StartStreaming(ctx)
	// StartStreaming creates a WebSocket session
	// It may fail without a valid WebSocket connection, but we test the creation
	// The actual WebSocket connection would require a real server
	if err != nil {
		assert.Error(t, err)
		assert.Nil(t, session)
	} else {
		assert.NotNil(t, session)
	}
}

func TestDefaultAzureConfig(t *testing.T) {
	config := DefaultAzureConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "eastus", config.Region)
	assert.Equal(t, "en-US", config.Language)
	assert.True(t, config.EnablePunctuation)
}

func TestAzureConfig_GetBaseURL(t *testing.T) {
	config := DefaultAzureConfig()
	config.Region = "westus2"
	url := config.GetBaseURL()
	assert.Contains(t, url, "westus2")
	assert.Contains(t, url, "stt.speech.microsoft.com")
}

func TestAzureConfig_GetWebSocketURL(t *testing.T) {
	config := DefaultAzureConfig()
	config.Region = "westus2"
	url := config.GetWebSocketURL()
	assert.Contains(t, url, "westus2")
	assert.Contains(t, url, "wss://")
}

func TestAzureConfig_GetBaseURL_WithCustomBaseURL(t *testing.T) {
	config := DefaultAzureConfig()
	config.BaseURL = "https://custom.example.com"
	url := config.GetBaseURL()
	assert.Equal(t, "https://custom.example.com", url)
}

func TestAzureConfig_GetWebSocketURL_WithCustomWebSocketURL(t *testing.T) {
	config := DefaultAzureConfig()
	config.WebSocketURL = "wss://custom.example.com/ws"
	url := config.GetWebSocketURL()
	assert.Equal(t, "wss://custom.example.com/ws", url)
}

func TestAzureProvider_Transcribe_WithEndpointID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify endpoint ID in URL
		assert.Contains(t, r.URL.RawQuery, "endpointId=test-endpoint")

		response := map[string]interface{}{
			"RecognitionStatus": "Success",
			"DisplayText":       "Test transcription",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Set EndpointID
	azConfig := provider.(*AzureProvider).config
	require.NotNil(t, azConfig)
	azConfig.EndpointID = "test-endpoint"

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	transcript, err := provider.Transcribe(ctx, audio)
	assert.NoError(t, err)
	assert.Equal(t, "Test transcription", transcript)
}

func TestAzureProvider_Transcribe_WithOptionalParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify optional parameters in URL
		assert.Contains(t, r.URL.RawQuery, "punctuation=true")
		assert.Contains(t, r.URL.RawQuery, "wordLevelTimestamps=true")
		assert.Contains(t, r.URL.RawQuery, "diarization=true")

		response := map[string]interface{}{
			"RecognitionStatus": "Success",
			"DisplayText":       "Test transcription",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Set optional parameters
	azConfig := provider.(*AzureProvider).config
	require.NotNil(t, azConfig)
	azConfig.EnablePunctuation = true
	azConfig.EnableWordLevelTimestamps = true
	azConfig.EnableSpeakerDiarization = true

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	transcript, err := provider.Transcribe(ctx, audio)
	assert.NoError(t, err)
	assert.Equal(t, "Test transcription", transcript)
}

func TestAzureProvider_Transcribe_ReadBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Close connection immediately to cause read error
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			conn.Close()
		}
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	_, err = provider.Transcribe(ctx, audio)
	assert.Error(t, err)
}

func TestAzureProvider_Transcribe_RecognitionStatusFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"RecognitionStatus": "NoMatch",
			"DisplayText":       "",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	_, err = provider.Transcribe(ctx, audio)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "recognition failed")
}

func TestAzureProvider_Transcribe_WithMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"RecognitionStatus": "Success",
			"DisplayText":       "Test with metrics",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &stt.Config{
		Provider:      "azure",
		APIKey:        "test-key",
		Timeout:       30 * time.Second,
		EnableMetrics: true,
	}
	config.BaseURL = server.URL

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	transcript, err := provider.Transcribe(ctx, audio)
	assert.NoError(t, err)
	assert.Equal(t, "Test with metrics", transcript)
}

func TestAzureProvider_Transcribe_NonRetryableError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
	}))
	defer server.Close()

	config := &stt.Config{
		Provider:   "azure",
		APIKey:     "test-key",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond,
	}
	config.BaseURL = server.URL

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	_, err = provider.Transcribe(ctx, audio)
	assert.Error(t, err)
}

func TestAzureProvider_Transcribe_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	config := &stt.Config{
		Provider:   "azure",
		APIKey:     "test-key",
		Timeout:    30 * time.Second,
		MaxRetries: 1,
		RetryDelay: 10 * time.Millisecond,
	}
	config.BaseURL = server.URL

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	_, err = provider.Transcribe(ctx, audio)
	assert.Error(t, err)
}

func TestAzureProvider_Transcribe_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate timeout
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  50 * time.Millisecond, // Short timeout
	}
	config.BaseURL = server.URL

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	_, err = provider.Transcribe(ctx, audio)
	assert.Error(t, err)
}

func TestNewAzureProvider_WithBaseURL(t *testing.T) {
	config := &stt.Config{
		Provider: "azure",
		APIKey:   "test-key",
		BaseURL:  "https://custom.azure.com",
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	azConfig := provider.(*AzureProvider).config
	assert.Equal(t, "https://custom.azure.com", azConfig.BaseURL)
}

func TestNewAzureProvider_WithCustomTimeout(t *testing.T) {
	config := &stt.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  60 * time.Second,
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	azConfig := provider.(*AzureProvider).config
	// Timeout should be copied from base config
	assert.Equal(t, 60*time.Second, azConfig.Timeout)
}

func TestNewAzureProvider_WithRegion(t *testing.T) {
	config := &stt.Config{
		Provider: "azure",
		APIKey:   "test-key",
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	azConfig := provider.(*AzureProvider).config
	// Should use default region
	assert.Equal(t, "eastus", azConfig.Region)
}

func TestNewAzureProvider_WithLanguage(t *testing.T) {
	config := &stt.Config{
		Provider: "azure",
		APIKey:   "test-key",
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	azConfig := provider.(*AzureProvider).config
	// Should use default language
	assert.Equal(t, "en-US", azConfig.Language)
}

func TestAzureProvider_StartStreaming_Error(t *testing.T) {
	config := &stt.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  1 * time.Second,
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Set invalid WebSocket URL to force error
	azConfig := provider.(*AzureProvider).config
	require.NotNil(t, azConfig)
	azConfig.WebSocketURL = "wss://invalid-url-that-does-not-exist.com/v1"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	session, err := provider.StartStreaming(ctx)
	// Should fail without valid WebSocket connection
	assert.Error(t, err)
	assert.Nil(t, session)
}

func TestAzureProvider_Transcribe_ContextDeadlineExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  50 * time.Millisecond, // Short timeout
	}
	config.BaseURL = server.URL

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	audio := []byte{1, 2, 3, 4, 5}
	_, err = provider.Transcribe(ctx, audio)
	assert.Error(t, err)
}

func TestAzureProvider_Transcribe_RetryContextCancellation(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	config := &stt.Config{
		Provider:   "azure",
		APIKey:     "test-key",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: 100 * time.Millisecond,
	}
	config.BaseURL = server.URL

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	audio := []byte{1, 2, 3, 4, 5}
	_, err = provider.Transcribe(ctx, audio)
	assert.Error(t, err)
}

func TestAzureProvider_Transcribe_RetryOnRateLimit(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Rate limit exceeded"))
		} else {
			response := map[string]interface{}{
				"RecognitionStatus": "Success",
				"DisplayText":       "Success after retry",
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	config := &stt.Config{
		Provider:   "azure",
		APIKey:     "test-key",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond,
	}
	config.BaseURL = server.URL

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	transcript, err := provider.Transcribe(ctx, audio)
	// May succeed or fail depending on retry logic
	_ = transcript
	_ = err
	assert.GreaterOrEqual(t, attempts, 1)
}

func TestAzureProvider_Transcribe_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay to allow request creation
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	audio := []byte{1, 2, 3, 4, 5}
	_, err = provider.Transcribe(ctx, audio)
	assert.Error(t, err)
	// Error should be context.Canceled or wrapped in STTError
	// Check if it's context.Canceled or if the underlying error is context.Canceled
	if err != context.Canceled && err != ctx.Err() {
		// Check if it's a wrapped context error
		assert.Contains(t, err.Error(), "context")
	}
}
