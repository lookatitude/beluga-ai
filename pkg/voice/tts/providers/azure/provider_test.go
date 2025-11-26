package azure

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAzureProvider(t *testing.T) {
	tests := []struct {
		config  *tts.Config
		name    string
		wantErr bool
	}{
		{
			name: "valid config",
			config: &tts.Config{
				Provider: "azure",
				APIKey:   "test-key",
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
			config: &tts.Config{
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
				require.Error(t, err)
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, provider)
			}
		})
	}
}

func TestAzureProvider_GenerateSpeech_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/cognitiveservices/v1")
		assert.Equal(t, "test-key", r.Header.Get("Ocp-Apim-Subscription-Key"))
		assert.Equal(t, "application/ssml+xml", r.Header.Get("Content-Type"))

		// Read SSML body
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Hello, world!")

		// Return mock audio data
		audioData := []byte{1, 2, 3, 4, 5, 6, 7, 8}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(audioData)
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
		BaseURL:  server.URL,
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	text := "Hello, world!"

	audio, err := provider.GenerateSpeech(ctx, text)
	require.NoError(t, err)
	assert.NotEmpty(t, audio)
}

func TestAzureProvider_GenerateSpeech_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
		BaseURL:  server.URL,
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	text := "Hello, world!"

	_, err = provider.GenerateSpeech(ctx, text)
	require.Error(t, err)
}

func TestAzureProvider_StreamGenerate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		audioData := []byte{1, 2, 3, 4, 5, 6, 7, 8}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(audioData)
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
		BaseURL:  server.URL,
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	text := "Hello, world!"

	reader, err := provider.StreamGenerate(ctx, text)
	require.NoError(t, err)
	assert.NotNil(t, reader)

	audio, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.NotEmpty(t, audio)
}

func TestDefaultAzureConfig(t *testing.T) {
	config := DefaultAzureConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "eastus", config.Region)
	assert.Equal(t, "en-US-AriaNeural", config.VoiceName)
	assert.Equal(t, "en-US", config.Language)
	assert.Equal(t, "medium", config.VoiceRate)
}

func TestAzureConfig_GetBaseURL(t *testing.T) {
	config := DefaultAzureConfig()
	config.Region = "westus2"
	url := config.GetBaseURL()
	assert.Contains(t, url, "westus2")
	assert.Contains(t, url, "tts.speech.microsoft.com")
}

func TestAzureProvider_GenerateSpeech_EmptyAudio(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		// Return empty body
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
		BaseURL:  server.URL,
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = provider.GenerateSpeech(ctx, "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no audio data")
}

func TestAzureConfig_buildSSML(t *testing.T) {
	config := DefaultAzureConfig()
	ssml := config.buildSSML("Hello world")
	assert.Contains(t, ssml, "Hello world")
	assert.Contains(t, ssml, config.VoiceName)
	assert.Contains(t, ssml, config.Language)
}

func TestAzureConfig_buildSSML_WithVoiceStyle(t *testing.T) {
	config := DefaultAzureConfig()
	config.VoiceStyle = "cheerful"
	ssml := config.buildSSML("Hello world")
	assert.Contains(t, ssml, "style=\"cheerful\"")
}

func TestAzureConfig_buildSSML_WithVoiceRate(t *testing.T) {
	config := DefaultAzureConfig()
	config.VoiceRate = "fast"
	ssml := config.buildSSML("Hello world")
	assert.Contains(t, ssml, "rate=\"fast\"")
}

func TestAzureConfig_buildSSML_WithVoicePitch(t *testing.T) {
	config := DefaultAzureConfig()
	config.VoiceRate = "slow"
	config.VoicePitch = "high"
	ssml := config.buildSSML("Hello world")
	assert.Contains(t, ssml, "pitch=\"high\"")
}

func TestAzureProvider_StreamGenerate_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "azure",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
		BaseURL:  server.URL,
	}

	provider, err := NewAzureProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = provider.StreamGenerate(ctx, "test")
	require.Error(t, err)
}
