package google

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

func TestNewGoogleProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *stt.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &stt.Config{
				Provider: "google",
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
				Provider: "google",
				APIKey:   "test-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGoogleProvider(tt.config)
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

func TestGoogleProvider_Transcribe_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/v1/speech:recognize")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Parse request body
		var requestBody map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&requestBody)
		require.NoError(t, err)

		// Verify request structure
		assert.Contains(t, requestBody, "config")
		assert.Contains(t, requestBody, "audio")

		// Return success response
		response := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"alternatives": []map[string]interface{}{
						{
							"transcript": "Hello, this is a test transcription",
							"confidence": 0.95,
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "google",
		APIKey:   "test-key",
		Model:    "default",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewGoogleProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	transcript, err := provider.Transcribe(ctx, audio)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, this is a test transcription", transcript)
}

func TestGoogleProvider_Transcribe_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "google",
		APIKey:   "test-key",
		Model:    "default",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewGoogleProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	_, err = provider.Transcribe(ctx, audio)
	assert.Error(t, err)
}

func TestGoogleProvider_Transcribe_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "google",
		APIKey:   "test-key",
		Model:    "default",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewGoogleProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	_, err = provider.Transcribe(ctx, audio)
	assert.Error(t, err)
}

func TestGoogleProvider_Transcribe_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"results": []map[string]interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "google",
		APIKey:   "test-key",
		Model:    "default",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewGoogleProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	_, err = provider.Transcribe(ctx, audio)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no transcript")
}

func TestGoogleProvider_StartStreaming(t *testing.T) {
	config := &stt.Config{
		Provider: "google",
		APIKey:   "test-key",
		Model:    "default",
		Timeout:  30 * time.Second,
	}

	provider, err := NewGoogleProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	session, err := provider.StartStreaming(ctx)
	// We expect an error as streaming is not yet fully implemented
	assert.Error(t, err)
	assert.Nil(t, session)
}

func TestGoogleProvider_Transcribe_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "google",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewGoogleProvider(config)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	audio := []byte{1, 2, 3, 4, 5}
	_, err = provider.Transcribe(ctx, audio)
	assert.Error(t, err)
}

func TestGoogleProvider_Transcribe_NoResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"alternatives": []map[string]interface{}{
						{
							"transcript": "",
							"confidence": 0.0,
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "google",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewGoogleProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	_, err = provider.Transcribe(ctx, audio)
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "no transcript")
	}
}

func TestGoogleProvider_Transcribe_MultipleAlternatives(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"alternatives": []map[string]interface{}{
						{
							"transcript": "First alternative",
							"confidence": 0.9,
						},
						{
							"transcript": "Second alternative",
							"confidence": 0.8,
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "google",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewGoogleProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	transcript, err := provider.Transcribe(ctx, audio)
	assert.NoError(t, err)
	assert.Equal(t, "First alternative", transcript)
}

func TestGoogleProvider_Transcribe_WithProjectID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify URL contains project ID
		assert.Contains(t, r.URL.Path, "/v1/projects/test-project/locations/global:recognize")

		response := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"alternatives": []map[string]interface{}{
						{
							"transcript": "Test transcription",
							"confidence": 0.95,
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "google",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewGoogleProvider(config)
	require.NoError(t, err)

	// Set ProjectID
	gConfig := provider.(*GoogleProvider).config
	gConfig.ProjectID = "test-project"

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	transcript, err := provider.Transcribe(ctx, audio)
	assert.NoError(t, err)
	assert.Equal(t, "Test transcription", transcript)
}

func TestGoogleProvider_Transcribe_WithOptionalParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request to verify optional params
		var requestBody map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&requestBody)
		require.NoError(t, err)

		config := requestBody["config"].(map[string]interface{})
		assert.True(t, config["enableSpeakerDiarization"].(bool))
		assert.Equal(t, 2.0, config["diarizationSpeakerCount"])

		response := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"alternatives": []map[string]interface{}{
						{
							"transcript": "Test",
							"confidence": 0.95,
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &stt.Config{
		Provider: "google",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewGoogleProvider(config)
	require.NoError(t, err)

	// Set optional parameters
	gConfig := provider.(*GoogleProvider).config
	gConfig.EnableSpeakerDiarization = true
	gConfig.DiarizationSpeakerCount = 2

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	transcript, err := provider.Transcribe(ctx, audio)
	assert.NoError(t, err)
	assert.Equal(t, "Test", transcript)
}

func TestDefaultGoogleConfig(t *testing.T) {
	config := DefaultGoogleConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "default", config.Model)
	assert.Equal(t, "en-US", config.LanguageCode)
	assert.True(t, config.EnableAutomaticPunctuation)
	assert.Equal(t, "https://speech.googleapis.com", config.BaseURL)
}
