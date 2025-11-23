package openai

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

func TestNewOpenAIProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *tts.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &tts.Config{
				Provider: "openai",
				APIKey:   "test-key",
				Model:    "tts-1",
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
				Provider: "openai",
				APIKey:   "test-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewOpenAIProvider(tt.config)
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

func TestOpenAIProvider_GenerateSpeech_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/v1/audio/speech")
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Return mock audio data
		audioData := []byte{1, 2, 3, 4, 5, 6, 7, 8}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(audioData)
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "openai",
		APIKey:   "test-key",
		Model:    "tts-1",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewOpenAIProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	text := "Hello, world!"

	audio, err := provider.GenerateSpeech(ctx, text)
	assert.NoError(t, err)
	assert.NotEmpty(t, audio)
	assert.Equal(t, []byte{1, 2, 3, 4, 5, 6, 7, 8}, audio)
}

func TestOpenAIProvider_GenerateSpeech_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "openai",
		APIKey:   "test-key",
		Model:    "tts-1",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewOpenAIProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	text := "Hello, world!"

	_, err = provider.GenerateSpeech(ctx, text)
	assert.Error(t, err)
}

func TestOpenAIProvider_StreamGenerate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/v1/audio/speech")
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		// Return streaming audio data
		audioData := []byte{1, 2, 3, 4, 5, 6, 7, 8}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(audioData)
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "openai",
		APIKey:   "test-key",
		Model:    "tts-1",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewOpenAIProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	text := "Hello, world!"

	reader, err := provider.StreamGenerate(ctx, text)
	assert.NoError(t, err)
	assert.NotNil(t, reader)

	// Read from stream
	audio, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.NotEmpty(t, audio)
}

func TestOpenAIProvider_StreamGenerate_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "openai",
		APIKey:   "test-key",
		Model:    "tts-1",
		Timeout:  30 * time.Second,
	}
	config.BaseURL = server.URL

	provider, err := NewOpenAIProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	text := "Hello, world!"

	_, err = provider.StreamGenerate(ctx, text)
	assert.Error(t, err)
}

func TestOpenAIProvider_GenerateSpeech_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "openai",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
		BaseURL:  server.URL,
	}

	provider, err := NewOpenAIProvider(config)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	_, err = provider.GenerateSpeech(ctx, "test")
	assert.Error(t, err)
}

func TestOpenAIProvider_StreamGenerate_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "openai",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
		BaseURL:  server.URL,
	}

	provider, err := NewOpenAIProvider(config)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	_, err = provider.StreamGenerate(ctx, "test")
	assert.Error(t, err)
}

func TestDefaultOpenAIConfig(t *testing.T) {
	config := DefaultOpenAIConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "tts-1", config.Model)
	assert.Equal(t, "alloy", config.Voice)
	assert.Equal(t, "mp3", config.ResponseFormat)
	assert.Equal(t, 1.0, config.Speed)
	assert.Equal(t, "https://api.openai.com", config.BaseURL)
}
