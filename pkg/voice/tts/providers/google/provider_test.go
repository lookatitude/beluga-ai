package google

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGoogleProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *tts.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &tts.Config{
				Provider: "google",
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

func TestGoogleProvider_GenerateSpeech_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/v1/text:synthesize")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Parse request body
		var requestBody map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&requestBody)
		require.NoError(t, err)
		assert.Contains(t, requestBody, "input")
		assert.Contains(t, requestBody, "voice")

		// Return success response with base64 audio
		audioData := []byte{1, 2, 3, 4, 5, 6, 7, 8}
		audioBase64 := base64.StdEncoding.EncodeToString(audioData)
		response := map[string]interface{}{
			"audioContent": audioBase64,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "google",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
		BaseURL:  server.URL,
	}

	provider, err := NewGoogleProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	text := "Hello, world!"

	audio, err := provider.GenerateSpeech(ctx, text)
	assert.NoError(t, err)
	assert.NotEmpty(t, audio)
}

func TestGoogleProvider_GenerateSpeech_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "google",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
		BaseURL:  server.URL,
	}

	provider, err := NewGoogleProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	text := "Hello, world!"

	_, err = provider.GenerateSpeech(ctx, text)
	assert.Error(t, err)
}

func TestGoogleProvider_GenerateSpeech_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "google",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
		BaseURL:  server.URL,
	}

	provider, err := NewGoogleProvider(config)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	_, err = provider.GenerateSpeech(ctx, "test")
	assert.Error(t, err)
}

func TestGoogleProvider_GenerateSpeech_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "google",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
		BaseURL:  server.URL,
	}

	provider, err := NewGoogleProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = provider.GenerateSpeech(ctx, "test")
	assert.Error(t, err)
}

func TestGoogleProvider_GenerateSpeech_EmptyAudioContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"audioContent": "",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "google",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
		BaseURL:  server.URL,
	}

	provider, err := NewGoogleProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = provider.GenerateSpeech(ctx, "test")
	assert.Error(t, err)
}

func TestGoogleProvider_StreamGenerate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		audioData := []byte{1, 2, 3, 4, 5}
		audioBase64 := base64.StdEncoding.EncodeToString(audioData)
		response := map[string]interface{}{
			"audioContent": audioBase64,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "google",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
		BaseURL:  server.URL,
	}

	provider, err := NewGoogleProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	reader, err := provider.StreamGenerate(ctx, "test")
	assert.NoError(t, err)
	assert.NotNil(t, reader)

	audio, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.NotEmpty(t, audio)
}

func TestDefaultGoogleConfig(t *testing.T) {
	config := DefaultGoogleConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "en-US-Standard-A", config.VoiceName)
	assert.Equal(t, "en-US", config.LanguageCode)
	assert.Equal(t, "NEUTRAL", config.SSMLGender)
	assert.Equal(t, 1.0, config.SpeakingRate)
	assert.Equal(t, "MP3", config.AudioEncoding)
}
